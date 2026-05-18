//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"
)

func openSelected() {
	e, ok := selectedEntry()
	if !ok {
		warn(ui().WarnSelect)
		return
	}
	shellExecute(app.hwnd, "open", e.Path, "", "")
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

func copySelectedPath() {
	e, ok := selectedEntry()
	if !ok {
		warn(ui().WarnSelect)
		return
	}
	if err := setClipboardText(e.Path); err != nil {
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
	e, ok := selectedEntry()
	if !ok {
		warn(ui().WarnSelect)
		return
	}
	desktop := desktopPath()
	if desktop == "" {
		warn(ui().WarnDesktopMissing)
		return
	}
	if err := shellCopyTo(e.Path, desktop); err != nil {
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

func shellCopyTo(from string, to string) error {
	fromMulti := utf16Multi([]string{from})
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
		return fmt.Errorf(ui().CopyFailed, ret)
	}
	if op.FAnyOperationsAborted != 0 {
		return fmt.Errorf(ui().CopyCanceled)
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
		var parsed interface{}
		if err := json.Unmarshal([]byte(text), &parsed); err == nil {
			if formatted, err := json.MarshalIndent(parsed, "", "  "); err == nil {
				text = string(formatted)
			}
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
