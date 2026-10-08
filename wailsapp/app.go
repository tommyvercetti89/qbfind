package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	releasesAPI  = "https://api.github.com/repos/tommyvercetti89/qbfind/releases/latest"
	releasesPage = "https://github.com/tommyvercetti89/qbfind/releases/latest"
)

const (
	langEnglish = "en"
	langTurkish = "tr"
)

// AppVersion is overridden at build time via -ldflags "-X main.AppVersion=..."
var AppVersion = "dev"

var (
	app *App

	indexedCount int64
	scanning     int32
	scanID       int64

	priorityRootKeys []string
	userRootKeys     []string
	noisyRootKeys    []string
	excludeSegments  []string
	selectedDrives   []string

	cfPreferredDropEffect uint16
	oleKeepAlive          sync.Map
)

type App struct {
	ctx      context.Context
	language string

	quit     chan struct{}
	quitOnce sync.Once

	mu      sync.RWMutex
	entries []fileEntry
}

type fileEntry struct {
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	LowerPath string    `json:"lowerPath"`
	LowerName string    `json:"lowerName"`
	LowerExt  string    `json:"lowerExt"`
	Size      int64     `json:"size"`
	ModTime   time.Time `json:"modTime"`
	IsDir     bool      `json:"isDir"`
	Priority  int       `json:"priority"`
}

type PreviewResult struct {
	Success bool   `json:"success"`
	Text    string `json:"text"`
	IsText  bool   `json:"isText"`
	Size    int64  `json:"size"`
	Name    string `json:"name"`
}

func NewApp() *App {
	a := &App{
		language: langEnglish,
		quit:     make(chan struct{}),
	}
	app = a
	return a
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.language = loadLanguage()
	procSetErrorMode.Call(SEM_FAILCRITICALERRORS)
	initPathPriority()
	initOleVTables()
	cfPreferredDropEffect = uint16(registerClipboardFormat("Preferred DropEffect"))
	excludeSegments = loadExclusions()
	selectedDrives = loadDrives()
	logf("QBFind Premium starting (version=%s, language=%s)", AppVersion, a.language)

	// Status emitter: only notifies the frontend when the index count or scanning state changes
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logPanic("statusEmitter", r)
			}
		}()
		lastCount := int64(-1)
		lastScanning := false
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-a.quit:
				return
			case <-ticker.C:
				count := atomic.LoadInt64(&indexedCount)
				isScanning := atomic.LoadInt32(&scanning) != 0
				if count == lastCount && isScanning == lastScanning {
					continue
				}
				lastCount = count
				lastScanning = isScanning
				wailsruntime.EventsEmit(a.ctx, "status_update", map[string]interface{}{
					"count":    count,
					"scanning": isScanning,
					"suffix":   scanSuffix(),
				})
			}
		}
	}()

	// Load the cached index for instant results, falling back to a full scan
	initIndexing()
}

func (a *App) shutdown(ctx context.Context) {
	logf("QBFind Premium shutting down")
	stopWatchers()
	a.quitOnce.Do(func() {
		close(a.quit)
	})
}

func (a *App) Search(query string) (results []fileEntry) {
	defer func() {
		if r := recover(); r != nil {
			logPanic("Search", r)
			results = nil
		}
	}()
	return searchIndex(query, maxResults)
}

func (a *App) OpenFile(path string) {
	shellExecute(0, "open", path, "", "")
}

func (a *App) ShowInExplorer(path string) {
	args := `/select,"` + path + `"`
	shellExecute(0, "open", "explorer.exe", args, "")
}

func (a *App) CopyPath(path string) string {
	if err := setClipboardText(path); err != nil {
		return err.Error()
	}
	return ui().StatusCopiedPath
}

func (a *App) CopyToDesktop(path string) string {
	desktop := desktopPath()
	if desktop == "" {
		return ui().WarnDesktopMissing
	}
	if err := shellCopyTo(path, desktop); err != nil {
		return err.Error()
	}
	return ui().StatusCopiedDesktop
}

func (a *App) GetPreview(path string, size int64, name string) (result PreviewResult) {
	defer func() {
		if r := recover(); r != nil {
			logPanic("GetPreview", r)
			result = PreviewResult{Success: false, Text: ui().PreviewOpenFailed, Size: size, Name: name}
		}
	}()
	text, ok := getTextPreview(path, size, name)
	if ok {
		return PreviewResult{
			Success: true,
			Text:    text,
			IsText:  true,
			Size:    size,
			Name:    name,
		}
	}
	ret := shellExecute(0, "preview", path, "", "")
	if ret <= 32 {
		return PreviewResult{
			Success: false,
			Text:    ui().PreviewOpenFailed,
			IsText:  false,
			Size:    size,
			Name:    name,
		}
	}
	return PreviewResult{
		Success: true,
		Text:    ui().PreviewBinary,
		IsText:  false,
		Size:    size,
		Name:    name,
	}
}

func (a *App) StartScan(reset bool) {
	startIndexing(reset)
}

func (a *App) GetLanguage() string {
	return a.language
}

func (a *App) SetLanguage(lang string) {
	setLanguage(lang)
}

func (a *App) GetExclude() string {
	return loadSettings()["exclude"]
}

func (a *App) GetDrives() string {
	return loadSettings()["drives"]
}

func (a *App) SetDrives(raw string) string {
	defer func() {
		if r := recover(); r != nil {
			logPanic("SetDrives", r)
		}
	}()
	setDrives(raw)
	return ui().SettingsSaved
}

func (a *App) SetExclude(raw string) string {
	defer func() {
		if r := recover(); r != nil {
			logPanic("SetExclude", r)
		}
	}()
	setExclusions(raw)
	return ui().SettingsSaved
}

func (a *App) StartDrag(paths []string) {
	if len(paths) == 0 {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			logPanic("StartDrag", r)
		}
	}()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	procOleInitialize.Call(0)
	defer procOleUninitialize.Call()
	doDragFiles(paths)
}

func (a *App) CutFiles(paths []string) string {
	if len(paths) == 0 {
		return ui().WarnSelect
	}
	if err := setClipboardFiles(paths, true); err != nil {
		return err.Error()
	}
	return ui().StatusCut
}

func (a *App) DeleteToRecycleBin(paths []string) string {
	if len(paths) == 0 {
		return ui().WarnSelect
	}
	if a.ctx != nil {
		confirm, err := wailsruntime.MessageDialog(a.ctx, wailsruntime.MessageDialogOptions{
			Type:    wailsruntime.QuestionDialog,
			Title:   ui().DeleteTitle,
			Message: fmt.Sprintf(ui().ConfirmDeleteText, len(paths)),
		})
		if err != nil || confirm != "Yes" {
			return ui().CopyCanceled
		}
	}
	fromMulti := utf16Multi(paths)
	op := SHFILEOPSTRUCT{
		Hwnd:   0,
		WFunc:  FO_DELETE,
		PFrom:  &fromMulti[0],
		FFlags: FOF_ALLOWUNDO,
	}
	ret, _, _ := procSHFileOperation.Call(uintptr(unsafe.Pointer(&op)))
	runtimeKeepAlive(fromMulti)
	if ret != 0 {
		return strings.Replace(ui().DeleteFailed, "{code}", strconv.FormatUint(uint64(ret), 10), 1)
	}
	if op.FAnyOperationsAborted != 0 {
		return ui().CopyCanceled
	}
	removeEntriesByPath(paths)
	return ui().StatusDeleted
}

func (a *App) RenameFile(path string, newName string) string {
	newName = strings.TrimSpace(newName)
	if newName == "" || strings.ContainsAny(newName, `\/:*?"<>|`) {
		return ui().RenameInvalid
	}
	newPath := filepath.Join(filepath.Dir(path), newName)
	if err := os.Rename(path, newPath); err != nil {
		return err.Error()
	}
	if info, err := os.Stat(newPath); err == nil {
		updateMasterEntryByLowerPath(searchFold(path), makeEntry(newPath, newName, info, info.IsDir()))
		buildQuickEntryIndex(watchRoots)
	}
	return ui().StatusRenamed
}

func (a *App) GetHistory() []string {
	return loadHistory()
}

func (a *App) AddHistory(query string) []string {
	return addHistory(query)
}

func (a *App) GetSavedSearches() []string {
	return loadSavedSearches()
}

func (a *App) ToggleSavedSearch(query string) []string {
	return toggleSavedSearch(query)
}

func (a *App) CheckForUpdate() string {
	current := strings.TrimPrefix(AppVersion, "v")
	if current == "" || current == "dev" {
		return ""
	}
	client := &http.Client{Timeout: 6 * time.Second}
	req, err := http.NewRequest(http.MethodGet, releasesAPI, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "QBFind")
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return ""
	}
	tag := strings.TrimPrefix(payload.TagName, "v")
	if tag == "" || compareVersions(tag, current) <= 0 {
		return ""
	}
	return payload.TagName
}

func (a *App) OpenURL(url string) {
	if a.ctx == nil {
		return
	}
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return
	}
	wailsruntime.BrowserOpenURL(a.ctx, url)
}

func (a *App) GetReleasesURL() string {
	return releasesPage
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

func (a *App) GetAboutMessage() map[string]string {
	s := ui()
	return map[string]string{
		"title":   s.AboutTitle,
		"message": s.AboutMessage,
		"version": AppVersion,
	}
}
