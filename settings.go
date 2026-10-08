//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"unsafe"
)

func ui() uiStrings {
	if app.language == langTurkish {
		return turkishUI
	}
	return englishUI
}

func buttonText(id uintptr) string {
	s := ui()
	switch id {
	case idBtnOpen:
		return s.Open
	case idBtnPreview:
		return s.Preview
	case idBtnCopyPath:
		return s.CopyPath
	case idBtnDesktop:
		return s.Desktop
	case idBtnExplorer:
		return s.Explorer
	case idBtnRefresh:
		return s.Refresh
	case idBtnInfo:
		return s.Info
	}
	return ""
}

func setLanguage(language string) {
	if language != langTurkish {
		language = langEnglish
	}
	if app.language == language {
		return
	}
	app.language = language
	saveLanguage(language)
	updateLanguageUI()
}

func updateLanguageUI() {
	if settingsHwnd != 0 {
		procSendMessage.Call(settingsHwnd, WM_CLOSE, 0, 0)
	}
	s := ui()
	if app.hwnd != 0 {
		procSetWindowText.Call(app.hwnd, uintptr(unsafe.Pointer(utf16Ptr(appTitle))))
		createMenu(app.hwnd)
	}
	for i, btn := range app.buttons {
		if i < len(app.buttonIDs) {
			procSetWindowText.Call(btn, uintptr(unsafe.Pointer(utf16Ptr(buttonText(app.buttonIDs[i])))))
		}
	}
	if app.hLabel != 0 {
		procSetWindowText.Call(app.hLabel, uintptr(unsafe.Pointer(utf16Ptr(s.SearchLabel))))
	}
	if app.hList != 0 {
		setListColumn(0, s.ColumnName, 165, LVCFMT_LEFT)
		setListColumn(1, s.ColumnPath, 360, LVCFMT_LEFT)
		setListColumn(2, s.ColumnSize, 80, LVCFMT_RIGHT)
		setListColumn(3, s.ColumnModified, 130, LVCFMT_LEFT)
	}
	layoutControls(app.hwnd)
	if strings.TrimSpace(app.lastQuery) != "" {
		runSearch()
	} else if app.hStatus != 0 {
		setStatus(fmt.Sprintf(s.StatusIndex, formatInt(atomic.LoadInt64(&indexedCount)), scanSuffix()))
	}
}

func loadSettings() map[string]string {
	out := make(map[string]string)
	path := configPath()
	if path == "" {
		return out
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if i := strings.Index(line, "="); i > 0 {
			out[strings.TrimSpace(line[:i])] = strings.TrimSpace(line[i+1:])
			continue
		}
		if out["language"] == "" {
			out["language"] = line
		}
	}
	return out
}

func saveSettings(settings map[string]string) {
	path := configPath()
	if path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return
	}
	var b strings.Builder
	for _, key := range []string{"language", "exclude", "drives", "history", "saved"} {
		if value, ok := settings[key]; ok && value != "" {
			fmt.Fprintf(&b, "%s=%s\n", key, value)
		}
	}
	_ = os.WriteFile(path, []byte(b.String()), 0600)
}

func loadLanguage() string {
	if loadSettings()["language"] == langTurkish {
		return langTurkish
	}
	return langEnglish
}

func saveLanguage(language string) {
	settings := loadSettings()
	settings["language"] = language
	saveSettings(settings)
}

func parseExclusions(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, searchFold(part))
	}
	return out
}

func loadExclusions() []string {
	return parseExclusions(loadSettings()["exclude"])
}

func setExclusions(raw string) {
	settings := loadSettings()
	settings["exclude"] = raw
	saveSettings(settings)
	excludeSegments = parseExclusions(raw)
}

func parseDrives(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		part = strings.ToUpper(part)
		part = strings.TrimSuffix(part, `\`)
		part = strings.TrimSuffix(part, ":")
		part = strings.TrimSuffix(part, `\`)
		if len(part) == 1 && part[0] >= 'A' && part[0] <= 'Z' {
			out = append(out, part)
		}
	}
	return out
}

func loadDrives() []string {
	return parseDrives(loadSettings()["drives"])
}

func setDrives(raw string) {
	settings := loadSettings()
	settings["drives"] = raw
	saveSettings(settings)
	selectedDrives = parseDrives(raw)
}

const listSeparator = "|||"

func loadList(key string) []string {
	raw := loadSettings()[key]
	if raw == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(raw, listSeparator) {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func saveList(key string, values []string) {
	settings := loadSettings()
	settings[key] = strings.Join(values, listSeparator)
	saveSettings(settings)
}

func loadHistory() []string {
	return loadList("history")
}

func addHistory(query string) []string {
	query = strings.TrimSpace(query)
	if query == "" {
		return loadHistory()
	}
	out := []string{query}
	for _, item := range loadHistory() {
		if strings.EqualFold(item, query) {
			continue
		}
		if len(out) >= 15 {
			break
		}
		out = append(out, item)
	}
	saveList("history", out)
	return out
}

func loadSavedSearches() []string {
	return loadList("saved")
}

func toggleSavedSearch(query string) []string {
	query = strings.TrimSpace(query)
	if query == "" {
		return loadSavedSearches()
	}
	var out []string
	found := false
	for _, item := range loadSavedSearches() {
		if strings.EqualFold(item, query) {
			found = true
			continue
		}
		out = append(out, item)
	}
	if !found {
		out = append(out, query)
	}
	saveList("saved", out)
	return out
}

func configPath() string {
	base := os.Getenv("APPDATA")
	if base == "" {
		if dir, err := os.UserConfigDir(); err == nil {
			base = dir
		}
	}
	if base == "" {
		return ""
	}
	return filepath.Join(base, "QBFind", "settings.txt")
}
