//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type uiStrings struct {
	Open                 string
	Preview              string
	CopyPath             string
	Desktop              string
	Explorer             string
	Refresh              string
	Info                 string
	SearchLabel          string
	StatusPreparing      string
	ColumnName           string
	ColumnPath           string
	ColumnSize           string
	ColumnModified       string
	MenuFile             string
	MenuActions          string
	MenuSettings         string
	MenuHelp             string
	MenuExit             string
	MenuCopyDesktop      string
	MenuShowExplorer     string
	MenuLanguage         string
	MenuEnglish          string
	MenuTurkish          string
	MenuAbout            string
	AboutTitle           string
	AboutMessage         string
	WarnSelect           string
	WarnDesktopMissing   string
	StatusCopiedDesktop  string
	StatusCopiedPath     string
	FileKind             string
	FolderKind           string
	InfoTitle            string
	StatusMinChars       string
	StatusResults        string
	StatusIndex          string
	ScanRunning          string
	ScanReady            string
	PreviewTitle         string
	PreviewBinary        string
	PreviewOpenFailed    string
	CopyFailed           string
	CopyCanceled         string
	ClipboardOpenFailed  string
	ClipboardWriteFailed string
	SettingsSaved        string
	Cut                  string
	Delete               string
	Rename               string
	StatusCut            string
	StatusDeleted        string
	StatusRenamed        string
	DeleteFailed         string
	DeleteTitle          string
	ConfirmDeleteText    string
	RenameInvalid        string
}

var englishUI = uiStrings{
	Open:                 "Open",
	Preview:              "Preview",
	CopyPath:             "Copy Path",
	Desktop:              "Desktop",
	Explorer:             "Explorer",
	Refresh:              "Refresh",
	Info:                 "Info",
	SearchLabel:          "Search:",
	StatusPreparing:      "Indexing...",
	ColumnName:           "Name",
	ColumnPath:           "Path",
	ColumnSize:           "Size",
	ColumnModified:       "Modified",
	MenuFile:             "File",
	MenuActions:          "Actions",
	MenuSettings:         "Settings",
	MenuHelp:             "Help",
	MenuExit:             "Exit",
	MenuCopyDesktop:      "Copy to Desktop",
	MenuShowExplorer:     "Show in Explorer",
	MenuLanguage:         "Language",
	MenuEnglish:          "English",
	MenuTurkish:          "Türkçe",
	MenuAbout:            "About",
	AboutTitle:           "About QBFind",
	AboutMessage:         "About QBFind\n\nSingle-file Go/Win32 file finder.\nCommon user folders are indexed first and extension searches such as .pdf or ext:pdf are supported.",
	WarnSelect:           "Select a result first.",
	WarnDesktopMissing:   "Desktop folder could not be found.",
	StatusCopiedDesktop:  "Selected item was copied to the Desktop.",
	StatusCopiedPath:     "Path copied to clipboard.",
	FileKind:             "File",
	FolderKind:           "Folder",
	InfoTitle:            "Info",
	StatusMinChars:       "Type at least 2 characters. Index: %s items%s",
	StatusResults:        "Showing %s results. Index: %s items%s",
	StatusIndex:          "Index: %s items%s",
	ScanRunning:          " (scanning)",
	ScanReady:            " (ready)",
	PreviewTitle:         "Preview",
	PreviewBinary:        "This file type does not have an in-app text preview. QBFind asked Windows to preview it.",
	PreviewOpenFailed:    "Windows could not preview this file type.",
	CopyFailed:           "Copy failed. Code: {code}",
	CopyCanceled:         "Copy was canceled.",
	ClipboardOpenFailed:  "Clipboard could not be opened.",
	ClipboardWriteFailed: "Clipboard could not be written.",
	SettingsSaved:        "Saved. Re-scan to apply.",
	Cut:                  "Cut",
	Delete:               "Delete",
	Rename:               "Rename",
	StatusCut:            "Selected items were cut to the clipboard.",
	StatusDeleted:        "Selected items were moved to the Recycle Bin.",
	StatusRenamed:        "Item renamed.",
	DeleteFailed:         "Delete failed. Code: {code}",
	DeleteTitle:          "Delete",
	ConfirmDeleteText:    "Move %d selected item(s) to the Recycle Bin?",
	RenameInvalid:        "Invalid file name.",
}

var turkishUI = uiStrings{
	Open:                 "Aç",
	Preview:              "Önizle",
	CopyPath:             "Yolu Kopyala",
	Desktop:              "Masaüstü",
	Explorer:             "Explorer",
	Refresh:              "Yenile",
	Info:                 "Bilgi",
	SearchLabel:          "Ara:",
	StatusPreparing:      "İndeks hazırlanıyor...",
	ColumnName:           "Ad",
	ColumnPath:           "Yol",
	ColumnSize:           "Boyut",
	ColumnModified:       "Değişim",
	MenuFile:             "Dosya",
	MenuActions:          "İşlem",
	MenuSettings:         "Ayarlar",
	MenuHelp:             "Yardım",
	MenuExit:             "Çıkış",
	MenuCopyDesktop:      "Masaüstüne Kopyala",
	MenuShowExplorer:     "Explorer'da Göster",
	MenuLanguage:         "Dil",
	MenuEnglish:          "English",
	MenuTurkish:          "Türkçe",
	MenuAbout:            "Hakkında",
	AboutTitle:           "QBFind Hakkında",
	AboutMessage:         "QBFind\n\nTek dosyalı Go/Win32 dosya bulucu.\nSık kullanılan kullanıcı klasörleri önce indekslenir; .pdf veya ext:pdf gibi uzantı aramaları desteklenir.",
	WarnSelect:           "Önce bir sonuç seçin.",
	WarnDesktopMissing:   "Masaüstü klasörü bulunamadı.",
	StatusCopiedDesktop:  "Seçili öğe masaüstüne kopyalandı.",
	StatusCopiedPath:     "Yol panoya kopyalandı.",
	FileKind:             "Dosya",
	FolderKind:           "Klasör",
	InfoTitle:            "Bilgi",
	StatusMinChars:       "En az 2 karakter yazın. İndeks: %s öğe%s",
	StatusResults:        "%s sonuç gösteriliyor. İndeks: %s öğe%s",
	StatusIndex:          "İndeks: %s öğe%s",
	ScanRunning:          " (taranıyor)",
	ScanReady:            " (hazır)",
	PreviewTitle:         "Önizleme",
	PreviewBinary:        "Bu dosya türü için uygulama içi metin önizlemesi yok. QBFind Windows'tan önizleme istedi.",
	PreviewOpenFailed:    "Windows bu dosya türünü önizleyemedi.",
	CopyFailed:           "Kopyalama başarısız oldu. Kod: {code}",
	CopyCanceled:         "Kopyalama iptal edildi.",
	ClipboardOpenFailed:  "Pano açılamadı.",
	ClipboardWriteFailed: "Panoya yazılamadı.",
	SettingsSaved:        "Kaydedildi. Uygulamak için yeniden tarayın.",
	Cut:                  "Kes",
	Delete:               "Sil",
	Rename:               "Yeniden Adlandır",
	StatusCut:            "Seçili öğeler panoya kesildi.",
	StatusDeleted:        "Seçili öğeler geri dönüşüm kutusuna taşındı.",
	StatusRenamed:        "Öğe yeniden adlandırıldı.",
	DeleteFailed:         "Silme başarısız oldu. Kod: {code}",
	DeleteTitle:          "Sil",
	ConfirmDeleteText:    "Seçili %d öğe geri dönüşüm kutusuna taşınsın mı?",
	RenameInvalid:        "Geçersiz dosya adı.",
}

func ui() uiStrings {
	if app != nil && app.language == langTurkish {
		return turkishUI
	}
	return englishUI
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
	if app != nil && app.ctx != nil {
		wailsruntime.EventsEmit(app.ctx, "language_changed", app.language)
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
		part = strings.TrimSuffix(strings.ToUpper(part), ":")
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
