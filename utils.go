//go:build windows

package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

func isReparse(info fs.FileInfo) bool {
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok || data == nil {
		return info.Mode()&os.ModeSymlink != 0
	}
	return data.FileAttributes&FILE_ATTRIBUTE_REPARSE_POINT != 0
}

func desktopPath() string {
	var buf [260]uint16
	ret, _, _ := procSHGetFolderPath.Call(0, CSIDL_DESKTOPDIRECTORY, 0, SHGFP_TYPE_CURRENT, uintptr(unsafe.Pointer(&buf[0])))
	if int32(ret) == 0 {
		if s := syscall.UTF16ToString(buf[:]); s != "" {
			return s
		}
	}
	if home := os.Getenv("USERPROFILE"); home != "" {
		return filepath.Join(home, "Desktop")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, "Desktop")
	}
	return ""
}

func formatEntrySize(e fileEntry) string {
	if e.IsDir {
		if app.language == langTurkish {
			return "<KLASÖR>"
		}
		return "<DIR>"
	}
	size := float64(e.Size)
	units := []string{"B", "KB", "MB", "GB", "TB"}
	i := 0
	for size >= 1024 && i < len(units)-1 {
		size /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%d %s", e.Size, units[i])
	}
	return fmt.Sprintf("%.1f %s", size, units[i])
}

func formatInt(v int64) string {
	s := fmt.Sprintf("%d", v)
	n := len(s)
	if n <= 3 {
		return s
	}
	var b strings.Builder
	first := n % 3
	if first == 0 {
		first = 3
	}
	b.WriteString(s[:first])
	for i := first; i < n; i += 3 {
		if app.language == langTurkish {
			b.WriteByte('.')
		} else {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func scanSuffix() string {
	s := ui()
	if atomic.LoadInt32(&scanning) != 0 {
		return s.ScanRunning
	}
	return s.ScanReady
}

func copyFailedError(code uintptr) error {
	return errors.New(strings.Replace(ui().CopyFailed, "{code}", strconv.FormatUint(uint64(code), 10), 1))
}

func setClipboardFiles(paths []string, move bool) error {
	if len(paths) == 0 {
		return errors.New(ui().WarnSelect)
	}
	if ret, _, _ := procOpenClipboard.Call(app.hwnd); ret == 0 {
		return errors.New(ui().ClipboardOpenFailed)
	}
	defer procCloseClipboard.Call()
	procEmptyClipboard.Call()

	effect := uint32(DROPEFFECT_COPY)
	if move {
		effect = DROPEFFECT_MOVE
	}
	if hEffect := createDropEffect(effect); hEffect != 0 {
		if ret, _, _ := procSetClipboardData.Call(uintptr(cfPreferredDropEffect), hEffect); ret == 0 {
			procGlobalFree.Call(hEffect)
		}
	}
	hDrop := createHDrop(paths)
	if hDrop == 0 {
		return errors.New(ui().ClipboardWriteFailed)
	}
	if ret, _, _ := procSetClipboardData.Call(CF_HDROP, hDrop); ret == 0 {
		procGlobalFree.Call(hDrop)
		return errors.New(ui().ClipboardWriteFailed)
	}
	return nil
}

func utf16PtrString(p *uint16) string {
	if p == nil {
		return ""
	}
	var buf []uint16
	for _, v := range unsafe.Slice(p, 1024) {
		if v == 0 {
			break
		}
		buf = append(buf, v)
	}
	return syscall.UTF16ToString(buf)
}

func setClipboardText(text string) error {
	if ret, _, _ := procOpenClipboard.Call(app.hwnd); ret == 0 {
		return errors.New(ui().ClipboardOpenFailed)
	}
	defer procCloseClipboard.Call()
	procEmptyClipboard.Call()

	chars := append(utf16.Encode([]rune(text)), 0)
	handle, _, _ := procGlobalAlloc.Call(GMEM_MOVEABLE|GMEM_ZEROINIT, uintptr(len(chars)*2))
	if handle == 0 {
		return errors.New(ui().ClipboardWriteFailed)
	}
	ptr, _, _ := procGlobalLock.Call(handle)
	if ptr == 0 {
		procGlobalFree.Call(handle)
		return errors.New(ui().ClipboardWriteFailed)
	}
	for i, ch := range chars {
		*(*uint16)(unsafe.Pointer(ptr + uintptr(i*2))) = ch
	}
	procGlobalUnlock.Call(handle)
	if ret, _, _ := procSetClipboardData.Call(CF_UNICODETEXT, handle); ret == 0 {
		procGlobalFree.Call(handle)
		return errors.New(ui().ClipboardWriteFailed)
	}
	return nil
}

func getWindowText(hwnd uintptr) string {
	length, _, _ := procGetWindowTextLength.Call(hwnd)
	if length == 0 {
		return ""
	}
	buf := make([]uint16, length+1)
	procGetWindowText.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), length+1)
	return syscall.UTF16ToString(buf)
}

func registerClipboardFormat(name string) uint32 {
	ret, _, _ := procRegisterClipboardFormat.Call(uintptr(unsafe.Pointer(utf16Ptr(name))))
	return uint32(ret)
}

func utf16Ptr(s string) *uint16 {
	return syscall.StringToUTF16Ptr(s)
}

func utf16Multi(values []string) []uint16 {
	var out []uint16
	for _, value := range values {
		out = append(out, utf16.Encode([]rune(value))...)
		out = append(out, 0)
	}
	out = append(out, 0)
	return out
}

func loword(v uintptr) uintptr {
	return v & 0xffff
}

func hiword(v uintptr) uintptr {
	return (v >> 16) & 0xffff
}

func xFromLParam(v uintptr) int32 {
	return int32(int16(v & 0xffff))
}

func yFromLParam(v uintptr) int32 {
	return int32(int16((v >> 16) & 0xffff))
}

func runtimeKeepAlive(x interface{}) {
	runtime.KeepAlive(x)
}

func atomicAddInt64(addr *int64, delta int64) int64 {
	return atomic.AddInt64(addr, delta)
}
