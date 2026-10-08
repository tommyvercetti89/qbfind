//go:build windows

package main

import (
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf16"
	"unsafe"
)

const indexCacheVersion = 1

type indexCacheFile struct {
	Version int
	Saved   time.Time
	Entries []fileEntry
}

func indexCachePath() string {
	base := os.Getenv("APPDATA")
	if base == "" {
		if dir, err := os.UserConfigDir(); err == nil {
			base = dir
		}
	}
	if base == "" {
		return ""
	}
	return filepath.Join(base, "QBFind", "index.cache.gz")
}

func saveIndexCache(entries []fileEntry) {
	path := indexCachePath()
	if path == "" || len(entries) == 0 {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0700)
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return
	}
	gz := gzip.NewWriter(f)
	err = gob.NewEncoder(gz).Encode(indexCacheFile{Version: indexCacheVersion, Saved: time.Now(), Entries: entries})
	if cerr := gz.Close(); err == nil {
		err = cerr
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(tmp)
		return
	}
	_ = os.Remove(path)
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
	}
}

func loadIndexCache() ([]fileEntry, bool) {
	path := indexCachePath()
	if path == "" {
		return nil, false
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, false
	}
	defer gz.Close()
	var cache indexCacheFile
	if err := gob.NewDecoder(gz).Decode(&cache); err != nil {
		return nil, false
	}
	if cache.Version != indexCacheVersion || len(cache.Entries) == 0 {
		return nil, false
	}
	if time.Since(cache.Saved) > 30*24*time.Hour {
		return nil, false
	}
	return cache.Entries, true
}

func initIndexing() {
	if entries, ok := loadIndexCache(); ok {
		app.mu.Lock()
		app.entries = entries
		app.mu.Unlock()
		atomic.StoreInt64(&indexedCount, int64(len(entries)))
		atomic.StoreInt32(&scanning, 0)
		logf("index cache loaded: %d entries", len(entries))
		var candidates []string
		for _, root := range commonScanRoots() {
			if driveAllowed(root) {
				candidates = append(candidates, root)
			}
		}
		watchRoots = uniquePhysicalDirs(candidates)
		startWatchers()
		return
	}
	startIndexing(true)
}

func startIndexing(reset bool) {
	stopWatchers()
	id := atomic.AddInt64(&scanID, 1)
	atomic.StoreInt32(&scanning, 1)
	if reset {
		app.mu.Lock()
		app.entries = nil
		app.mu.Unlock()
		atomic.StoreInt64(&indexedCount, 0)
	}
	go scanComputer(id)
}

func scanComputer(id int64) {
	defer func() {
		if r := recover(); r != nil {
			logPanic("scanComputer", r)
			atomic.StoreInt32(&scanning, 0)
		}
	}()
	roots := logicalDrives()
	if len(roots) == 0 {
		atomic.StoreInt32(&scanning, 0)
		return
	}
	var candidates []string
	for _, root := range commonScanRoots() {
		if driveAllowed(root) {
			candidates = append(candidates, root)
		}
	}
	quickRoots := uniquePhysicalDirs(candidates)
	watchRoots = quickRoots
	quickRootSet := make(map[string]struct{}, len(candidates))
	for _, root := range candidates {
		quickRootSet[pathKey(root)] = struct{}{}
	}

	tasks := make(chan string, 256)
	var taskWG sync.WaitGroup
	var workerWG sync.WaitGroup
	workers := runtime.NumCPU() * 2
	if workers < 2 {
		workers = 2
	}
	if workers > 16 {
		workers = 16
	}

	for i := 0; i < workers; i++ {
		workerWG.Add(1)
		go func() {
			defer workerWG.Done()
			batch := make([]fileEntry, 0, 512)
			for root := range tasks {
				walkTree(id, root, &batch, quickRootSet)
				flushBatch(id, &batch)
				taskWG.Done()
			}
		}()
	}

	rootBatch := make([]fileEntry, 0, 256)
	for _, root := range quickRoots {
		if atomic.LoadInt64(&scanID) != id {
			break
		}
		taskWG.Add(1)
		tasks <- root
	}
	for _, root := range roots {
		if atomic.LoadInt64(&scanID) != id {
			break
		}
		enqueueRoot(id, root, tasks, &taskWG, &rootBatch)
		flushBatch(id, &rootBatch)
	}

	taskWG.Wait()
	close(tasks)
	workerWG.Wait()

	if atomic.LoadInt64(&scanID) == id {
		atomic.StoreInt32(&scanning, 0)
		app.mu.RLock()
		entries := app.entries
		app.mu.RUnlock()
		saveIndexCache(entries)
		logf("scan %d complete: %d entries", id, len(entries))
		startWatchers()
	}
}

func enqueueRoot(id int64, root string, tasks chan<- string, taskWG *sync.WaitGroup, batch *[]fileEntry) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, de := range entries {
		if atomic.LoadInt64(&scanID) != id {
			return
		}
		if isExcludedName(de.Name()) {
			continue
		}
		info, err := de.Info()
		if err != nil {
			continue
		}
		full := filepath.Join(root, de.Name())
		isDir := info.IsDir()
		*batch = append(*batch, makeEntry(full, de.Name(), info, isDir))
		if len(*batch) >= 512 {
			flushBatch(id, batch)
		}
		if isDir && !isReparse(info) {
			taskWG.Add(1)
			tasks <- full
		}
	}
}

func walkTree(id int64, root string, batch *[]fileEntry, quickRootSet map[string]struct{}) {
	if isExcludedName(filepath.Base(root)) {
		return
	}
	stack := []string{root}
	rootKey := pathKey(root)
	for len(stack) > 0 {
		if atomic.LoadInt64(&scanID) != id {
			return
		}
		dir := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if key := pathKey(dir); key != rootKey {
			if _, ok := quickRootSet[key]; ok {
				continue
			}
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, de := range entries {
			if atomic.LoadInt64(&scanID) != id {
				return
			}
			if isExcludedName(de.Name()) {
				continue
			}
			info, err := de.Info()
			if err != nil {
				continue
			}
			full := filepath.Join(dir, de.Name())
			isDir := info.IsDir()
			*batch = append(*batch, makeEntry(full, de.Name(), info, isDir))
			if len(*batch) >= 512 {
				flushBatch(id, batch)
			}
			if isDir && !isReparse(info) {
				stack = append(stack, full)
			}
		}
	}
}

func makeEntry(path string, name string, info fs.FileInfo, isDir bool) fileEntry {
	lowerPath := searchFold(path)
	lowerExt := searchFold(filepath.Ext(name))
	return fileEntry{
		Path:      path,
		Name:      name,
		LowerPath: lowerPath,
		LowerName: searchFold(name),
		LowerExt:  lowerExt,
		Size:      info.Size(),
		ModTime:   info.ModTime(),
		IsDir:     isDir,
		Priority:  entryPriority(path),
	}
}

func flushBatch(id int64, batch *[]fileEntry) {
	if len(*batch) == 0 || atomic.LoadInt64(&scanID) != id {
		*batch = (*batch)[:0]
		return
	}
	app.mu.Lock()
	app.entries = append(app.entries, (*batch)...)
	app.mu.Unlock()
	atomic.AddInt64(&indexedCount, int64(len(*batch)))
	*batch = (*batch)[:0]
}

func logicalDrives() []string {
	mask, _, _ := procGetLogicalDrives.Call()
	var roots []string
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) == 0 {
			continue
		}
		root := fmt.Sprintf("%c:\\", 'A'+rune(i))
		if !driveAllowed(root) {
			continue
		}
		typ, _, _ := procGetDriveType.Call(uintptr(unsafePointer(utf16Ptr(root))))
		switch typ {
		case DRIVE_FIXED, DRIVE_REMOVABLE, DRIVE_REMOTE, DRIVE_RAMDISK:
			if driveReady(root) {
				roots = append(roots, root)
			}
		}
	}
	return roots
}

func driveAllowed(root string) bool {
	if len(selectedDrives) == 0 || len(root) == 0 {
		return true
	}
	letter := strings.ToUpper(string(root[0]))
	for _, drive := range selectedDrives {
		if drive == letter {
			return true
		}
	}
	return false
}

func driveReady(root string) bool {
	var available uint64
	ret, _, _ := procGetDiskFreeSpaceEx.Call(
		uintptr(unsafePointer(utf16Ptr(root))),
		uintptr(unsafe.Pointer(&available)),
		0,
		0,
	)
	return ret != 0
}

func initPathPriority() {
	priorityRootKeys = existingPathKeys(commonUserFolders())
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		userRootKeys = existingPathKeys([]string{home})
	}
	noisy := []string{
		os.Getenv("SystemRoot"),
		os.Getenv("ProgramFiles"),
		os.Getenv("ProgramFiles(x86)"),
		os.Getenv("ProgramW6432"),
		os.Getenv("ProgramData"),
		os.Getenv("TEMP"),
		os.Getenv("TMP"),
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		noisy = append(noisy, filepath.Join(home, "AppData"))
	}
	noisyRootKeys = existingPathKeys(noisy)
}

func commonScanRoots() []string {
	return existingDirs(commonUserFolders())
}

func uniquePhysicalDirs(paths []string) []string {
	var out []string
	var seen []fs.FileInfo
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			continue
		}
		duplicate := false
		for _, known := range seen {
			if os.SameFile(known, info) {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}
		seen = append(seen, info)
		out = append(out, path)
	}
	return out
}

func commonUserFolders() []string {
	var paths []string
	addPath := func(path string) {
		if path != "" {
			paths = append(paths, path)
		}
	}
	addPath(desktopPath())
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		appendKnownFolderNames(&paths, home)
	}
	if oneDrive := os.Getenv("OneDrive"); oneDrive != "" {
		appendKnownFolderNames(&paths, oneDrive)
	}
	if appData := os.Getenv("APPDATA"); appData != "" {
		addPath(filepath.Join(appData, "Microsoft", "Windows", "Start Menu"))
	}
	if programData := os.Getenv("ProgramData"); programData != "" {
		addPath(filepath.Join(programData, "Microsoft", "Windows", "Start Menu"))
	}
	return paths
}

func appendKnownFolderNames(paths *[]string, base string) {
	names := []string{
		"Desktop",
		"MasaÃ¼stÃ¼",
		"Downloads",
		"Ä°ndirilenler",
		"Documents",
		"Belgeler",
		"Pictures",
		"Resimler",
		"Music",
		"MÃ¼zik",
		"Videos",
		"Videolar",
	}
	for _, name := range names {
		*paths = append(*paths, filepath.Join(base, name))
	}
}

func existingDirs(paths []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, path := range paths {
		key := pathKey(path)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, path)
	}
	return out
}

func existingPathKeys(paths []string) []string {
	dirs := existingDirs(paths)
	keys := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		keys = append(keys, pathKey(dir))
	}
	return keys
}

func entryPriority(path string) int {
	key := pathKey(path)
	if key == "" {
		return 3
	}
	if isUnderAnyKey(key, priorityRootKeys) {
		return 0
	}
	if isUnderAnyKey(key, noisyRootKeys) || hasNoisySegment(key) {
		return 5
	}
	if isUnderAnyKey(key, userRootKeys) {
		return 1
	}
	return 3
}

func removeEntriesByPath(paths []string) {
	if len(paths) == 0 {
		return
	}
	keys := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		keys[searchFold(p)] = struct{}{}
	}
	app.mu.Lock()
	entries := make([]fileEntry, 0, len(app.entries))
	for _, e := range app.entries {
		if _, found := keys[e.LowerPath]; found {
			continue
		}
		entries = append(entries, e)
	}
	app.entries = entries
	app.mu.Unlock()
	atomic.StoreInt64(&indexedCount, int64(len(entries)))
	buildQuickEntryIndex(watchRoots)
}

func updateMasterEntryByLowerPath(lowerPath string, newEntry fileEntry) {
	app.mu.Lock()
	entries := make([]fileEntry, len(app.entries))
	copy(entries, app.entries)
	for i := range entries {
		if entries[i].LowerPath == lowerPath {
			entries[i] = newEntry
			break
		}
	}
	app.entries = entries
	app.mu.Unlock()
}

func isExcludedName(name string) bool {
	if len(excludeSegments) == 0 {
		return false
	}
	folded := searchFold(name)
	for _, pattern := range excludeSegments {
		if folded == pattern {
			return true
		}
		if strings.ContainsAny(pattern, "*?") {
			if ok, err := filepath.Match(pattern, folded); err == nil && ok {
				return true
			}
		}
	}
	return false
}

func hasNoisySegment(key string) bool {
	noisySegments := []string{
		`\windows\`,
		`\program files\`,
		`\program files (x86)\`,
		`\programdata\`,
		`\appdata\`,
		`\node_modules\`,
		`\.git\`,
		`\system volume information\`,
		`\$recycle.bin\`,
	}
	for _, segment := range noisySegments {
		if strings.Contains(key, segment) {
			return true
		}
	}
	return false
}

func isUnderAnyKey(key string, roots []string) bool {
	for _, root := range roots {
		if sameOrUnderKey(key, root) {
			return true
		}
	}
	return false
}

func sameOrUnderKey(key, root string) bool {
	if key == "" || root == "" {
		return false
	}
	if key == root {
		return true
	}
	if strings.HasSuffix(root, `\`) || strings.HasSuffix(root, `/`) {
		return strings.HasPrefix(key, root)
	}
	return strings.HasPrefix(key, root+string(filepath.Separator))
}

func pathKey(path string) string {
	if path == "" {
		return ""
	}
	return strings.ToLower(filepath.Clean(path))
}

func unsafePointer(p *uint16) unsafe.Pointer {
	return unsafe.Pointer(p)
}

type watchChangeOp struct {
	action uint32
	path   string
}

type watchChange struct {
	removeKey string
	add       *fileEntry
}

type dirWatcher struct {
	h    uintptr
	once sync.Once
}

func (w *dirWatcher) close() {
	w.once.Do(func() {
		procCloseHandle.Call(w.h)
	})
}

var (
	watchGen        int64
	watchMu         sync.Mutex
	watchHandles    []*dirWatcher
	watchRoots      []string
	quickEntryIndex map[string]int
)

func startWatchers() {
	stopWatchers()
	gen := atomic.AddInt64(&watchGen, 1)
	roots := watchRoots
	if len(roots) == 0 {
		return
	}
	buildQuickEntryIndex(roots)

	type activeWatcher struct {
		root string
		w    *dirWatcher
	}
	var active []activeWatcher
	for _, root := range roots {
		h, _, _ := procCreateFileW.Call(
			uintptr(unsafe.Pointer(utf16Ptr(root))),
			FILE_LIST_DIRECTORY,
			FILE_SHARE_READ|FILE_SHARE_WRITE|FILE_SHARE_DELETE,
			0,
			OPEN_EXISTING,
			FILE_FLAG_BACKUP_SEMANTICS,
			0,
		)
		if h == INVALID_HANDLE_VALUE || h == 0 {
			continue
		}
		active = append(active, activeWatcher{root: root, w: &dirWatcher{h: h}})
	}
	handles := make([]*dirWatcher, 0, len(active))
	for _, item := range active {
		handles = append(handles, item.w)
	}
	watchMu.Lock()
	watchHandles = handles
	watchMu.Unlock()
	for _, item := range active {
		go watchLoop(gen, item.root, item.w)
	}
	if len(active) > 0 {
		logf("live index watching %d folders", len(active))
	}
}

func stopWatchers() {
	atomic.AddInt64(&watchGen, 1)
	watchMu.Lock()
	watchers := watchHandles
	watchHandles = nil
	watchMu.Unlock()
	for _, w := range watchers {
		procCancelIoEx.Call(w.h, 0)
		w.close()
	}
}

func watchLoop(gen int64, root string, w *dirWatcher) {
	defer w.close()
	buffer := make([]byte, 64*1024)
	for {
		if atomic.LoadInt64(&watchGen) != gen {
			return
		}
		var returned uint32
		ret, _, _ := procReadDirectoryChangesW.Call(
			w.h,
			uintptr(unsafe.Pointer(&buffer[0])),
			uintptr(len(buffer)),
			1,
			FILE_NOTIFY_FILTER,
			uintptr(unsafe.Pointer(&returned)),
			0,
			0,
		)
		if ret == 0 {
			return
		}
		if returned == 0 {
			continue
		}
		if atomic.LoadInt64(&watchGen) != gen {
			return
		}
		changes := parseWatchChanges(root, buffer[:returned])
		if len(changes) > 0 {
			applyWatchChanges(changes, gen)
		}
	}
}

func parseWatchChanges(root string, buffer []byte) []watchChange {
	var ops []watchChangeOp
	offset := 0
	for {
		if offset+12 > len(buffer) {
			break
		}
		next := *(*uint32)(unsafe.Pointer(&buffer[offset]))
		action := *(*uint32)(unsafe.Pointer(&buffer[offset+4]))
		nameLen := *(*uint32)(unsafe.Pointer(&buffer[offset+8]))
		if nameLen == 0 || offset+12+int(nameLen) > len(buffer) {
			break
		}
		name := utf16BytesToString(buffer[offset+12 : offset+12+int(nameLen)])
		ops = append(ops, watchChangeOp{action: action, path: filepath.Join(root, name)})
		if next == 0 {
			break
		}
		offset += int(next)
	}
	var changes []watchChange
	for _, op := range ops {
		switch op.action {
		case FILE_ACTION_ADDED, FILE_ACTION_MODIFIED, FILE_ACTION_RENAMED_NEW_NAME:
			info, err := os.Stat(op.path)
			if err != nil {
				continue
			}
			entry := makeEntry(op.path, info.Name(), info, info.IsDir())
			changes = append(changes, watchChange{add: &entry})
		case FILE_ACTION_REMOVED, FILE_ACTION_RENAMED_OLD_NAME:
			changes = append(changes, watchChange{removeKey: searchFold(op.path)})
		}
	}
	return changes
}

func utf16BytesToString(b []byte) string {
	u16 := make([]uint16, len(b)/2)
	for i := range u16 {
		u16[i] = uint16(b[i*2]) | uint16(b[i*2+1])<<8
	}
	return string(utf16.Decode(u16))
}

func buildQuickEntryIndex(roots []string) {
	rootKeys := make([]string, 0, len(roots))
	for _, root := range roots {
		rootKeys = append(rootKeys, pathKey(root))
	}
	app.mu.Lock()
	defer app.mu.Unlock()
	index := make(map[string]int)
	for i, e := range app.entries {
		if isUnderAnyKey(pathKey(e.Path), rootKeys) {
			index[e.LowerPath] = i
		}
	}
	quickEntryIndex = index
}

func applyWatchChanges(changes []watchChange, gen int64) {
	app.mu.Lock()
	defer app.mu.Unlock()
	if atomic.LoadInt64(&watchGen) != gen {
		return
	}

	var entries []fileEntry
	ensureCopy := func() {
		if entries == nil {
			entries = make([]fileEntry, len(app.entries))
			copy(entries, app.entries)
		}
	}
	var delta int64

	removeKey := func(key string) {
		idx, ok := quickEntryIndex[key]
		if !ok {
			return
		}
		ensureCopy()
		last := len(entries) - 1
		moved := entries[last]
		entries[idx] = moved
		entries = entries[:last]
		delete(quickEntryIndex, key)
		if movedKey := moved.LowerPath; movedKey != key {
			if _, exists := quickEntryIndex[movedKey]; exists {
				quickEntryIndex[movedKey] = idx
			}
		}
		delta--
	}

	for _, change := range changes {
		if change.removeKey != "" {
			prefix := change.removeKey + `\`
			for key := range quickEntryIndex {
				if strings.HasPrefix(key, prefix) {
					removeKey(key)
				}
			}
			removeKey(change.removeKey)
			continue
		}
		if change.add == nil {
			continue
		}
		entry := *change.add
		if idx, ok := quickEntryIndex[entry.LowerPath]; ok {
			ensureCopy()
			entries[idx] = entry
			continue
		}
		if entries == nil {
			app.entries = append(app.entries, entry)
			quickEntryIndex[entry.LowerPath] = len(app.entries) - 1
		} else {
			entries = append(entries, entry)
			quickEntryIndex[entry.LowerPath] = len(entries) - 1
		}
		delta++
	}
	if entries != nil {
		app.entries = entries
	}
	if delta != 0 {
		atomic.AddInt64(&indexedCount, delta)
	}
}
