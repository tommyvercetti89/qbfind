//go:build windows

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"
)

func openSelected() {
	entries := selectedEntries()
	if len(entries) == 0 {
		warn(ui().WarnSelect)
		return
	}
	for _, e := range entries {
		shellExecute(app.hwnd, "open", e.Path, "", "")
	}
}

func showSelectedInExplorer() {
	e, ok := selectedEntry()
	if !ok {
		warn(ui().WarnSelect)
		return
	}
	args := `/select,"` + e.Path + `"`
	shellExecute(app.hwnd, "open", "explorer.exe", args, "")
}

func cutSelectedFiles() {
	entries := selectedEntries()
	if len(entries) == 0 {
		warn(ui().WarnSelect)
		return
	}
	paths := make([]string, 0, len(entries))
	for _, e := range entries {
		paths = append(paths, e.Path)
	}
	if err := setClipboardFiles(paths, true); err != nil {
		warn(err.Error())
		return
	}
	setStatus(ui().StatusCut)
}

func deleteSelected() {
	entries := selectedEntries()
	if len(entries) == 0 {
		warn(ui().WarnSelect)
		return
	}
	paths := make([]string, 0, len(entries))
	for _, e := range entries {
		paths = append(paths, e.Path)
	}
	fromMulti := utf16Multi(paths)
	op := SHFILEOPSTRUCT{
		Hwnd:   app.hwnd,
		WFunc:  FO_DELETE,
		PFrom:  &fromMulti[0],
		FFlags: FOF_ALLOWUNDO,
	}
	ret, _, _ := procSHFileOperation.Call(uintptr(unsafe.Pointer(&op)))
	runtimeKeepAlive(fromMulti)
	if ret != 0 {
		warn(strings.Replace(ui().DeleteFailed, "{code}", strconv.FormatUint(uint64(ret), 10), 1))
		return
	}
	if op.FAnyOperationsAborted != 0 {
		warn(ui().CopyCanceled)
		return
	}
	removeEntriesByPath(paths)
	setStatus(ui().StatusDeleted)
}

func renameSelected() {
	ret, _, _ := procSendMessage.Call(app.hList, LVM_GETNEXTITEM, ^uintptr(0), LVNI_SELECTED)
	idx := int32(ret)
	if idx < 0 || int(idx) >= len(app.results) {
		warn(ui().WarnSelect)
		return
	}
	procSendMessage.Call(app.hList, LVM_EDITLABELW, uintptr(idx), 0)
}

func applyRename(info *NMLVDISPINFO) {
	if info.Item.PszText == nil {
		return
	}
	idx := int(info.Item.IItem)
	if idx < 0 || idx >= len(app.results) {
		return
	}
	newName := strings.TrimSpace(utf16PtrString(info.Item.PszText))
	old := app.results[idx]
	if newName == "" || newName == old.Name {
		return
	}
	if strings.ContainsAny(newName, `\/:*?"<>|`) {
		warn(ui().RenameInvalid)
		return
	}
	newPath := filepath.Join(filepath.Dir(old.Path), newName)
	if err := os.Rename(old.Path, newPath); err != nil {
		warn(err.Error())
		return
	}
	info2, err := os.Stat(newPath)
	if err != nil {
		warn(err.Error())
		return
	}
	newEntry := makeEntry(newPath, newName, info2, info2.IsDir())
	app.results[idx] = newEntry
	updateMasterEntryByLowerPath(old.LowerPath, newEntry)
	buildQuickEntryIndex(watchRoots)
	procInvalidateRect.Call(app.hList, 0, 1)
}

func removeEntriesByPath(paths []string) {
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

	results := make([]fileEntry, 0, len(app.results))
	for _, e := range app.results {
		if _, found := keys[e.LowerPath]; found {
			continue
		}
		results = append(results, e)
	}
	app.results = results
	procSendMessage.Call(app.hList, LVM_SETITEMCOUNT, uintptr(len(app.results)), 0)
	procInvalidateRect.Call(app.hList, 0, 1)
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

func copySelectedPath() {
	entries := selectedEntries()
	if len(entries) == 0 {
		warn(ui().WarnSelect)
		return
	}
	paths := make([]string, 0, len(entries))
	for _, e := range entries {
		paths = append(paths, e.Path)
	}
	if err := setClipboardText(strings.Join(paths, "\r\n")); err != nil {
		warn(err.Error())
		return
	}
	setStatus(ui().StatusCopiedPath)
}

func previewSelected() {
	e, ok := selectedEntry()
	if !ok {
		warn(ui().WarnSelect)
		return
	}
	if showTextPreview(e) {
		return
	}
	ret := shellExecute(app.hwnd, "preview", e.Path, "", "")
	if ret <= 32 {
		warn(ui().PreviewOpenFailed)
		return
	}
	setStatus(ui().PreviewBinary)
}

func copySelectedToDesktop() {
	entries := selectedEntries()
	if len(entries) == 0 {
		warn(ui().WarnSelect)
		return
	}
	desktop := desktopPath()
	if desktop == "" {
		warn(ui().WarnDesktopMissing)
		return
	}
	paths := make([]string, 0, len(entries))
	for _, e := range entries {
		paths = append(paths, e.Path)
	}
	if err := shellCopyTo(paths, desktop); err != nil {
		warn(err.Error())
		return
	}
	setStatus(ui().StatusCopiedDesktop)
}

func showSelectedInfo() {
	e, ok := selectedEntry()
	if !ok {
		warn(ui().WarnSelect)
		return
	}
	s := ui()
	kind := s.FileKind
	if e.IsDir {
		kind = s.FolderKind
	}
	msg := fmt.Sprintf("%s\n\n%s: %s\n%s: %s\n%s: %s\n%s: %s", kind, s.ColumnName, e.Name, s.ColumnPath, e.Path, s.ColumnSize, formatEntrySize(e), s.ColumnModified, e.ModTime.Format("2006-01-02 15:04:05"))
	info(s.InfoTitle, msg)
}

const (
	releasesAPI  = "https://api.github.com/repos/tommyvercetti89/qbfind/releases/latest"
	releasesPage = "https://github.com/tommyvercetti89/qbfind/releases/latest"
)

type uiNotice struct {
	title string
	text  string
	url   string
}

func postNotice(n *uiNotice) {
	key := uintptr(unsafe.Pointer(n))
	pendingUI.Store(key, n)
	if ret, _, _ := procPostMessage.Call(app.hwnd, WM_UIMSG, 0, key); ret == 0 {
		pendingUI.Delete(key)
	}
}

func checkForUpdates() {
	go func() {
		s := ui()
		tag, err := fetchLatestTag()
		if err != nil {
			postNotice(&uiNotice{title: s.InfoTitle, text: s.UpdateFailed})
			return
		}
		if tag == "" || compareVersions(strings.TrimPrefix(tag, "v"), appVersion) <= 0 {
			postNotice(&uiNotice{title: s.InfoTitle, text: s.UpdateNone})
			return
		}
		postNotice(&uiNotice{
			title: s.InfoTitle,
			text:  strings.Replace(s.UpdateAvailable, "{version}", tag, 1),
			url:   releasesPage,
		})
	}()
}

func fetchLatestTag() (string, error) {
	client := &http.Client{Timeout: 6 * time.Second}
	req, err := http.NewRequest(http.MethodGet, releasesAPI, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "QBFind")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	return payload.TagName, nil
}

func compareVersions(a string, b string) int {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	for i := 0; i < len(as) || i < len(bs); i++ {
		av, bv := 0, 0
		if i < len(as) {
			av, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			bv, _ = strconv.Atoi(bs[i])
		}
		if av != bv {
			if av > bv {
				return 1
			}
			return -1
		}
	}
	return 0
}

func shellExecute(hwnd uintptr, verb, file, params, dir string) uintptr {
	ret, _, _ := procShellExecute.Call(
		hwnd,
		uintptr(unsafe.Pointer(utf16Ptr(verb))),
		uintptr(unsafe.Pointer(utf16Ptr(file))),
		uintptr(unsafe.Pointer(utf16Ptr(params))),
		uintptr(unsafe.Pointer(utf16Ptr(dir))),
		SW_SHOWNORMAL,
	)
	return ret
}

func shellCopyTo(from []string, to string) error {
	fromMulti := utf16Multi(from)
	toMulti := utf16Multi([]string{to})
	op := SHFILEOPSTRUCT{
		Hwnd:   app.hwnd,
		WFunc:  FO_COPY,
		PFrom:  &fromMulti[0],
		PTo:    &toMulti[0],
		FFlags: FOF_ALLOWUNDO | FOF_NOCONFIRMMKDIR | FOF_RENAMEONCOLLISION,
	}
	ret, _, _ := procSHFileOperation.Call(uintptr(unsafe.Pointer(&op)))
	runtimeKeepAlive(fromMulti)
	runtimeKeepAlive(toMulti)
	if ret != 0 {
		return copyFailedError(ret)
	}
	if op.FAnyOperationsAborted != 0 {
		return errors.New(ui().CopyCanceled)
	}
	return nil
}

func showTextPreview(e fileEntry) bool {
	if e.IsDir || e.Size > 256*1024 {
		return false
	}
	ext := strings.TrimPrefix(e.LowerExt, ".")
	if !isTextPreviewExtension(ext) && e.Size > 64*1024 {
		return false
	}
	data, err := os.ReadFile(e.Path)
	if err != nil {
		return false
	}
	text, ok := decodePreviewText(data)
	if !ok {
		return false
	}
	if ext == "json" {
		var buf bytes.Buffer
		if err := json.Indent(&buf, []byte(text), "", "  "); err == nil {
			text = buf.String()
		}
	}
	const maxRunes = 4000
	runes := []rune(text)
	if len(runes) > maxRunes {
		text = string(runes[:maxRunes]) + "\n..."
	}
	info(fmt.Sprintf("%s - %s", ui().PreviewTitle, e.Name), text)
	return true
}

func isTextPreviewExtension(ext string) bool {
	switch ext {
	case "txt", "md", "csv", "log", "json", "xml", "html", "htm", "css", "js", "ts", "go", "py", "ps1", "bat", "cmd", "ini", "yaml", "yml", "toml", "sql", "rtf":
		return true
	}
	return false
}

func decodePreviewText(data []byte) (string, bool) {
	if len(data) == 0 {
		return "(empty)", true
	}
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return string(data[3:]), true
	}
	if len(data) >= 2 {
		if data[0] == 0xFF && data[1] == 0xFE {
			return decodeUTF16Preview(data[2:], false), true
		}
		if data[0] == 0xFE && data[1] == 0xFF {
			return decodeUTF16Preview(data[2:], true), true
		}
	}
	checkLen := len(data)
	if checkLen > 4096 {
		checkLen = 4096
	}
	if strings.Count(string(data[:checkLen]), "\x00") > 0 {
		return "", false
	}
	if utf8.Valid(data) {
		return string(data), true
	}
	return strings.ToValidUTF8(string(data), ""), true
}

func decodeUTF16Preview(data []byte, bigEndian bool) string {
	if len(data)%2 == 1 {
		data = data[:len(data)-1]
	}
	u16 := make([]uint16, len(data)/2)
	for i := range u16 {
		if bigEndian {
			u16[i] = uint16(data[i*2])<<8 | uint16(data[i*2+1])
		} else {
			u16[i] = uint16(data[i*2+1])<<8 | uint16(data[i*2])
		}
	}
	return string(utf16.Decode(u16))
}
