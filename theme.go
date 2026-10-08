//go:build windows

package main

import (
	"sync"
	"syscall"
	"unsafe"
)

const (
	LVM_GETHEADER      = LVM_FIRST + 31
	LVM_SETIMAGELIST   = LVM_FIRST + 3
	LVM_SETBKCOLOR     = LVM_FIRST + 1
	LVM_SETTEXTCOLOR   = LVM_FIRST + 36
	LVM_SETTEXTBKCOLOR = LVM_FIRST + 38

	HDM_FIRST    = 0x1200
	HDM_SETITEMW = HDM_FIRST + 12
	HDI_FORMAT   = 0x0004
	HDF_STRING   = 0x4000
	HDF_SORTUP   = 0x0400
	HDF_SORTDOWN = 0x0200

	LVSIL_SMALL = 1

	SHGFI_ICON               = 0x00000100
	SHGFI_SMALLICON          = 0x00000001
	SHGFI_SYSICONINDEX       = 0x00004000
	SHGFI_USEFILEATTRIBUTES  = 0x00000010
	FILE_ATTRIBUTE_DIRECTORY = 0x00000010
	FILE_ATTRIBUTE_NORMAL    = 0x00000080

	WM_CTLCOLORSTATIC  = 0x0138
	WM_CTLCOLOREDIT    = 0x0133
	WM_CTLCOLORLISTBOX = 0x0134
	WM_ERASEBKGND      = 0x0014
	WM_DRAWITEM        = 0x002B

	BS_OWNERDRAW  = 0x0000000B
	GWL_STYLE     = ^uintptr(15) // -16
	ODT_BUTTON    = 4
	ODS_SELECTED  = 0x0001
	DT_CENTER     = 0x00000001
	DT_VCENTER    = 0x00000004
	DT_SINGLELINE = 0x00000020
	BKMODE_TRANSPARENT = 1

	DWMWA_USE_IMMERSIVE_DARK_MODE = 20

	colorDarkBg      = 0x001E1E1E
	colorDarkPanel   = 0x00262626
	colorDarkPressed = 0x003A3A3A
	colorDarkText    = 0x00E6E6E6
	colorWhite       = 0x00FFFFFF
	colorBlack       = 0x00000000
)

var (
	procCreateSolidBrush      = gdi32.NewProc("CreateSolidBrush")
	procSetTextColor          = gdi32.NewProc("SetTextColor")
	procSetBkColor            = gdi32.NewProc("SetBkColor")
	procSetBkMode             = gdi32.NewProc("SetBkMode")
	procFillRect              = gdi32.NewProc("FillRect")
	procDrawTextW             = gdi32.NewProc("DrawTextW")
	procSetWindowTheme        = syscall.NewLazyDLL("uxtheme.dll").NewProc("SetWindowTheme")
	procDwmSetWindowAttribute = syscall.NewLazyDLL("dwmapi.dll").NewProc("DwmSetWindowAttribute")
	procSetWindowLongPtr      = user32.NewProc("SetWindowLongPtrW")
	procSHGetFileInfoW        = shell32.NewProc("SHGetFileInfoW")
)

var (
	darkMode          bool
	darkPanelBrush    uintptr
	darkPressedBrush  uintptr
	iconMu            sync.Mutex
	iconCache         = map[string]int32{}
	shellImageList    uintptr
	headerSortedCol   int32 = -1
)

type HDITEM struct {
	Mask       uint32
	Cxy        int32
	PszText    *uint16
	Hbm        uintptr
	CchTextMax int32
	Fmt        int32
	LParam     uintptr
	IImage     int32
	IOrder     int32
	Type       int32
	PvFilter   uintptr
	State      uint32
}

type SHFILEINFO struct {
	HIcon         uintptr
	IIcon         int32
	DwAttributes  uint32
	SzDisplayName [260]uint16
	SzTypeName    [80]uint16
}

type DRAWITEMSTRUCT struct {
	CtlType    uint32
	CtlID      uint32
	ItemID     uint32
	ItemAction uint32
	ItemState  uint32
	HwndItem   uintptr
	HDC        uintptr
	RcItem     RECT
	ItemData   uintptr
}

func initTheme() {
	darkPanelBrush, _, _ = procCreateSolidBrush.Call(colorDarkPanel)
	darkPressedBrush, _, _ = procCreateSolidBrush.Call(colorDarkPressed)
}

func initShellIcons() {
	var info SHFILEINFO
	ret, _, _ := procSHGetFileInfoW.Call(
		uintptr(unsafe.Pointer(utf16Ptr("folder"))),
		FILE_ATTRIBUTE_DIRECTORY,
		uintptr(unsafe.Pointer(&info)),
		unsafe.Sizeof(info),
		SHGFI_SYSICONINDEX|SHGFI_SMALLICON|SHGFI_USEFILEATTRIBUTES,
	)
	if ret != 0 {
		shellImageList = ret
		procSendMessage.Call(app.hList, LVM_SETIMAGELIST, LVSIL_SMALL, shellImageList)
	}
}

func iconIndexFor(e fileEntry) int32 {
	key := e.LowerExt
	attrs := uintptr(FILE_ATTRIBUTE_NORMAL)
	if e.IsDir {
		key = "<dir>"
		attrs = FILE_ATTRIBUTE_DIRECTORY
	}
	if key == "" {
		key = "<file>"
	}
	iconMu.Lock()
	idx, ok := iconCache[key]
	iconMu.Unlock()
	if ok {
		return idx
	}
	var info SHFILEINFO
	ret, _, _ := procSHGetFileInfoW.Call(
		uintptr(unsafe.Pointer(utf16Ptr(key))),
		attrs,
		uintptr(unsafe.Pointer(&info)),
		unsafe.Sizeof(info),
		SHGFI_SYSICONINDEX|SHGFI_SMALLICON|SHGFI_USEFILEATTRIBUTES,
	)
	var result int32
	if ret != 0 {
		result = info.IIcon
	}
	iconMu.Lock()
	iconCache[key] = result
	iconMu.Unlock()
	return result
}

func updateSortIndicator() {
	header, _, _ := procSendMessage.Call(app.hList, LVM_GETHEADER, 0, 0)
	if header == 0 {
		return
	}
	if headerSortedCol == sortColumn && sortColumn < 0 {
		return
	}
	headerSortedCol = sortColumn
	for i := int32(0); i < 4; i++ {
		fmtFlags := int32(HDF_STRING)
		if sortColumn == i {
			if sortAscending {
				fmtFlags |= HDF_SORTUP
			} else {
				fmtFlags |= HDF_SORTDOWN
			}
		}
		item := HDITEM{Mask: HDI_FORMAT, Fmt: fmtFlags}
		procSendMessage.Call(header, HDM_SETITEMW, uintptr(i), uintptr(unsafe.Pointer(&item)))
	}
}

func darkCtlColor(hdc uintptr) uintptr {
	procSetTextColor.Call(hdc, colorDarkText)
	procSetBkColor.Call(hdc, colorDarkPanel)
	return darkPanelBrush
}

func darkErase(hwnd, hdc uintptr) uintptr {
	var rc RECT
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), darkPanelBrush)
	return 1
}

func drawOwnerButton(dis *DRAWITEMSTRUCT) {
	brush := darkPanelBrush
	if dis.ItemState&ODS_SELECTED != 0 && darkPressedBrush != 0 {
		brush = darkPressedBrush
	}
	procFillRect.Call(dis.HDC, uintptr(unsafe.Pointer(&dis.RcItem)), brush)
	procSetBkMode.Call(dis.HDC, BKMODE_TRANSPARENT)
	procSetTextColor.Call(dis.HDC, colorDarkText)
	text := getWindowText(dis.HwndItem)
	if text != "" {
		procDrawTextW.Call(
			dis.HDC,
			uintptr(unsafe.Pointer(utf16Ptr(text))),
			^uintptr(0),
			uintptr(unsafe.Pointer(&dis.RcItem)),
			DT_CENTER|DT_VCENTER|DT_SINGLELINE,
		)
	}
}

func applyTheme() {
	if app.hwnd == 0 {
		return
	}
	buttonStyle := uintptr(BS_PUSHBUTTON)
	value := int32(0)
	if darkMode {
		buttonStyle = BS_OWNERDRAW
		value = 1
	}
	for _, btn := range app.buttons {
		procSetWindowLongPtr.Call(btn, GWL_STYLE, buttonStyle)
		procInvalidateRect.Call(btn, 0, 1)
	}
	if err := procSetWindowTheme.Find(); err == nil {
		mode := utf16Ptr("")
		if darkMode {
			mode = utf16Ptr("DarkMode_Explorer")
		}
		procSetWindowTheme.Call(app.hList, uintptr(unsafe.Pointer(mode)), 0)
		procSetWindowTheme.Call(app.hSearch, uintptr(unsafe.Pointer(mode)), 0)
	}
	if err := procDwmSetWindowAttribute.Find(); err == nil {
		procDwmSetWindowAttribute.Call(app.hwnd, DWMWA_USE_IMMERSIVE_DARK_MODE, uintptr(unsafe.Pointer(&value)), 4)
	}
	if darkMode {
		procSendMessage.Call(app.hList, LVM_SETBKCOLOR, 0, colorDarkBg)
		procSendMessage.Call(app.hList, LVM_SETTEXTCOLOR, 0, colorDarkText)
		procSendMessage.Call(app.hList, LVM_SETTEXTBKCOLOR, 0, colorDarkBg)
	} else {
		procSendMessage.Call(app.hList, LVM_SETBKCOLOR, 0, colorWhite)
		procSendMessage.Call(app.hList, LVM_SETTEXTCOLOR, 0, colorBlack)
		procSendMessage.Call(app.hList, LVM_SETTEXTBKCOLOR, 0, colorWhite)
	}
	procInvalidateRect.Call(app.hwnd, 0, 1)
}

func loadTheme() {
	darkMode = loadSettings()["theme"] == "dark"
}

func saveTheme() {
	settings := loadSettings()
	if darkMode {
		settings["theme"] = "dark"
	} else {
		delete(settings, "theme")
	}
	saveSettings(settings)
}
