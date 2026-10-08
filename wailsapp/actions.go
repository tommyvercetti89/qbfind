//go:build windows

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"
)

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
		Hwnd:   0,
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

func getTextPreview(path string, size int64, name string) (string, bool) {
	if size > 256*1024 {
		return "", false
	}
	ext := strings.TrimPrefix(searchFold(filepathExt(name)), ".")
	if !isTextPreviewExtension(ext) && size > 64*1024 {
		return "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	text, ok := decodePreviewText(data)
	if !ok {
		return "", false
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
	return text, true
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

func filepathExt(path string) string {
	for i := len(path) - 1; i >= 0 && !os.IsPathSeparator(path[i]); i-- {
		if path[i] == '.' {
			return path[i:]
		}
	}
	return ""
}
