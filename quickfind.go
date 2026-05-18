//go:build windows

package main

import (
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	appClassName = "GoQBFindWin32"
	appTitle     = "QBFind"

	winW = 760
	winH = 460

	idBtnOpen     = 1001
	idBtnDesktop  = 1002
	idBtnExplorer = 1003
	idBtnRefresh  = 1004
	idBtnInfo     = 1005
	idBtnCopyPath = 1006
	idBtnPreview  = 1007
	idMenuExit    = 1101
	idMenuAbout   = 1102
	idLangEnglish = 1201
	idLangTurkish = 1202
	idSearch      = 2001
	idList        = 3001
	idStatus      = 4001

	timerSearch  = 1
	timerRefresh = 2
)

const (
	WS_OVERLAPPED       = 0x00000000
	WS_CAPTION          = 0x00C00000
	WS_SYSMENU          = 0x00080000
	WS_THICKFRAME       = 0x00040000
	WS_MINIMIZEBOX      = 0x00020000
	WS_MAXIMIZEBOX      = 0x00010000
	WS_OVERLAPPEDWINDOW = WS_OVERLAPPED | WS_CAPTION | WS_SYSMENU | WS_THICKFRAME | WS_MINIMIZEBOX | WS_MAXIMIZEBOX
	WS_CHILD            = 0x40000000
	WS_VISIBLE          = 0x10000000
	WS_BORDER           = 0x00800000
	WS_TABSTOP          = 0x00010000
	WS_CLIPSIBLINGS     = 0x04000000
	WS_VSCROLL          = 0x00200000
	WS_HSCROLL          = 0x00100000

	ES_AUTOHSCROLL = 0x0080

	BS_PUSHBUTTON = 0x00000000

	MF_STRING    = 0x00000000
	MF_POPUP     = 0x00000010
	MF_SEPARATOR = 0x00000800
	MF_CHECKED   = 0x00000008

	LVS_REPORT        = 0x0001
	LVS_SINGLESEL     = 0x0004
	LVS_SHOWSELALWAYS = 0x0008

	LVS_EX_GRIDLINES     = 0x00000001
	LVS_EX_FULLROWSELECT = 0x00000020
	LVS_EX_DOUBLEBUFFER  = 0x00010000

	LVM_FIRST                    = 0x1000
	LVM_GETITEMCOUNT             = LVM_FIRST + 4
	LVM_DELETEALLITEMS           = LVM_FIRST + 9
	LVM_GETNEXTITEM              = LVM_FIRST + 12
	LVM_ENSUREVISIBLE            = LVM_FIRST + 19
	LVM_SETITEMSTATE             = LVM_FIRST + 43
	LVM_HITTEST                  = LVM_FIRST + 18
	LVM_INSERTITEMW              = LVM_FIRST + 77
	LVM_SETCOLUMNW               = LVM_FIRST + 96
	LVM_INSERTCOLUMNW            = LVM_FIRST + 97
	LVM_SETITEMTEXTW             = LVM_FIRST + 116
	LVM_SETEXTENDEDLISTVIEWSTYLE = LVM_FIRST + 54

	LVIF_TEXT     = 0x0001
	LVIF_STATE    = 0x0008
	LVIF_PARAM    = 0x0004
	LVCF_FMT      = 0x0001
	LVCF_WIDTH    = 0x0002
	LVCF_TEXT     = 0x0004
	LVCF_SUBITEM  = 0x0008
	LVCFMT_LEFT   = 0
	LVCFMT_RIGHT  = 1
	LVNI_SELECTED = 0x0002
	LVIS_FOCUSED  = 0x0001
	LVIS_SELECTED = 0x0002

	WM_CREATE      = 0x0001
	WM_DESTROY     = 0x0002
	WM_SIZE        = 0x0005
	WM_COMMAND     = 0x0111
	WM_TIMER       = 0x0113
	WM_NOTIFY      = 0x004E
	WM_CONTEXTMENU = 0x007B
	WM_SETFONT     = 0x0030
	WM_SETREDRAW   = 0x000B
	WM_CLOSE       = 0x0010
	WM_KEYDOWN     = 0x0100

	NM_DBLCLK     = 0xFFFFFFFD
	NM_RCLICK     = 0xFFFFFFFB
	LVN_BEGINDRAG = 0xFFFFFF93
	EN_CHANGE     = 0x0300

	SW_SHOWNORMAL = 1
	SW_SHOW       = 5

	COLOR_BTNFACE    = 15
	IDC_ARROW        = 32512
	IDI_APPLICATION  = 32512
	DEFAULT_GUI_FONT = 17

	MB_OK              = 0x00000000
	MB_ICONINFORMATION = 0x00000040
	MB_ICONWARNING     = 0x00000030

	ICC_LISTVIEW_CLASSES = 0x00000001
	TPM_RIGHTBUTTON      = 0x00000002

	DRIVE_REMOVABLE = 2
	DRIVE_FIXED     = 3
	DRIVE_REMOTE    = 4
	DRIVE_RAMDISK   = 6

	FILE_ATTRIBUTE_REPARSE_POINT = 0x00000400

	FO_COPY               = 0x0002
	FOF_RENAMEONCOLLISION = 0x0008
	FOF_ALLOWUNDO         = 0x0040
	FOF_NOCONFIRMMKDIR    = 0x0200

	CSIDL_DESKTOPDIRECTORY = 0x0010
	SHGFP_TYPE_CURRENT     = 0

	maxResults = 500
)

const (
	CF_UNICODETEXT = 13
	CF_HDROP       = 15

	DATADIR_GET = 1

	DVASPECT_CONTENT = 1
	TYMED_HGLOBAL    = 1

	GMEM_MOVEABLE = 0x0002
	GMEM_ZEROINIT = 0x0040

	DROPEFFECT_NONE = 0
	DROPEFFECT_COPY = 1
	DROPEFFECT_MOVE = 2

	MK_LBUTTON = 0x0001

	S_OK                         uintptr = 0x00000000
	S_FALSE                      uintptr = 0x00000001
	E_NOTIMPL                    uintptr = 0x80004001
	E_NOINTERFACE                uintptr = 0x80004002
	E_POINTER                    uintptr = 0x80004003
	E_INVALIDARG                 uintptr = 0x80070057
	DV_E_FORMATETC               uintptr = 0x80040064
	OLE_E_ADVISENOTSUPPORTED     uintptr = 0x80040003
	DRAGDROP_S_DROP              uintptr = 0x00040100
	DRAGDROP_S_CANCEL            uintptr = 0x00040101
	DRAGDROP_S_USEDEFAULTCURSORS uintptr = 0x00040102
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	comctl32 = syscall.NewLazyDLL("comctl32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")

	procRegisterClassEx         = user32.NewProc("RegisterClassExW")
	procCreateWindowEx          = user32.NewProc("CreateWindowExW")
	procDefWindowProc           = user32.NewProc("DefWindowProcW")
	procShowWindow              = user32.NewProc("ShowWindow")
	procUpdateWindow            = user32.NewProc("UpdateWindow")
	procGetMessage              = user32.NewProc("GetMessageW")
	procTranslateMessage        = user32.NewProc("TranslateMessage")
	procDispatchMessage         = user32.NewProc("DispatchMessageW")
	procPostQuitMessage         = user32.NewProc("PostQuitMessage")
	procLoadCursor              = user32.NewProc("LoadCursorW")
	procLoadIcon                = user32.NewProc("LoadIconW")
	procSendMessage             = user32.NewProc("SendMessageW")
	procMoveWindow              = user32.NewProc("MoveWindow")
	procGetClientRect           = user32.NewProc("GetClientRect")
	procGetCursorPos            = user32.NewProc("GetCursorPos")
	procScreenToClient          = user32.NewProc("ScreenToClient")
	procSetWindowText           = user32.NewProc("SetWindowTextW")
	procGetWindowTextLength     = user32.NewProc("GetWindowTextLengthW")
	procGetWindowText           = user32.NewProc("GetWindowTextW")
	procSetTimer                = user32.NewProc("SetTimer")
	procKillTimer               = user32.NewProc("KillTimer")
	procMessageBox              = user32.NewProc("MessageBoxW")
	procInvalidateRect          = user32.NewProc("InvalidateRect")
	procRegisterClipboardFormat = user32.NewProc("RegisterClipboardFormatW")
	procCreateMenu              = user32.NewProc("CreateMenu")
	procCreatePopupMenu         = user32.NewProc("CreatePopupMenu")
	procAppendMenu              = user32.NewProc("AppendMenuW")
	procSetMenu                 = user32.NewProc("SetMenu")
	procDrawMenuBar             = user32.NewProc("DrawMenuBar")
	procTrackPopupMenu          = user32.NewProc("TrackPopupMenu")
	procDestroyMenu             = user32.NewProc("DestroyMenu")
	procOpenClipboard           = user32.NewProc("OpenClipboard")
	procEmptyClipboard          = user32.NewProc("EmptyClipboard")
	procSetClipboardData        = user32.NewProc("SetClipboardData")
	procCloseClipboard          = user32.NewProc("CloseClipboard")

	procGetModuleHandle  = kernel32.NewProc("GetModuleHandleW")
	procGetLogicalDrives = kernel32.NewProc("GetLogicalDrives")
	procGetDriveType     = kernel32.NewProc("GetDriveTypeW")
	procGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	procGlobalLock       = kernel32.NewProc("GlobalLock")
	procGlobalUnlock     = kernel32.NewProc("GlobalUnlock")
	procGlobalFree       = kernel32.NewProc("GlobalFree")

	procGetStockObject = gdi32.NewProc("GetStockObject")

	procInitCommonControlsEx = comctl32.NewProc("InitCommonControlsEx")

	procShellExecute    = shell32.NewProc("ShellExecuteW")
	procSHFileOperation = shell32.NewProc("SHFileOperationW")
	procSHGetFolderPath = shell32.NewProc("SHGetFolderPathW")

	procOleInitialize   = ole32.NewProc("OleInitialize")
	procOleUninitialize = ole32.NewProc("OleUninitialize")
	procDoDragDrop      = ole32.NewProc("DoDragDrop")
)

type appState struct {
	hwnd      uintptr
	hLabel    uintptr
	hSearch   uintptr
	hList     uintptr
	hStatus   uintptr
	hFont     uintptr
	buttons   []uintptr
	buttonIDs []uintptr
	language  string

	mu      sync.RWMutex
	entries []fileEntry
	results []fileEntry

	lastQuery        string
	lastIndexedShown int64
}

type fileEntry struct {
	Path      string
	Name      string
	LowerPath string
	LowerName string
	LowerExt  string
	Size      int64
	ModTime   time.Time
	IsDir     bool
	Priority  int
}

type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

type POINT struct {
	X int32
	Y int32
}

type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type INITCOMMONCONTROLSEX struct {
	DwSize uint32
	DwICC  uint32
}

type NMHDR struct {
	HwndFrom uintptr
	IdFrom   uintptr
	Code     uint32
}

type NMLISTVIEW struct {
	Hdr      NMHDR
	IItem    int32
	ISubItem int32
	NewState uint32
	OldState uint32
	Changed  uint32
	PtAction POINT
	LParam   uintptr
}

type LVHITTESTINFO struct {
	Pt       POINT
	Flags    uint32
	IItem    int32
	ISubItem int32
	IGroup   int32
}

type LVITEM struct {
	Mask       uint32
	IItem      int32
	ISubItem   int32
	State      uint32
	StateMask  uint32
	PszText    *uint16
	CchTextMax int32
	IImage     int32
	LParam     uintptr
	IIndent    int32
	IGroupId   int32
	CColumns   uint32
	PuColumns  uintptr
	PiColFmt   uintptr
	IGroup     int32
}

type LVCOLUMN struct {
	Mask       uint32
	Fmt        int32
	Cx         int32
	PszText    *uint16
	CchTextMax int32
	ISubItem   int32
	IImage     int32
	IOrder     int32
	CxMin      int32
	CxDefault  int32
	CxIdeal    int32
}

type SHFILEOPSTRUCT struct {
	Hwnd                  uintptr
	WFunc                 uint32
	PFrom                 *uint16
	PTo                   *uint16
	FFlags                uint16
	FAnyOperationsAborted int32
	HNameMappings         uintptr
	LpszProgressTitle     *uint16
}

const (
	langEnglish = "en"
	langTurkish = "tr"
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
	CopyFailed:           "Copy failed. Code: %d",
	CopyCanceled:         "Copy was canceled.",
	ClipboardOpenFailed:  "Clipboard could not be opened.",
	ClipboardWriteFailed: "Clipboard could not be written.",
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
	CopyFailed:           "Kopyalama başarısız oldu. Kod: %d",
	CopyCanceled:         "Kopyalama iptal edildi.",
	ClipboardOpenFailed:  "Pano açılamadı.",
	ClipboardWriteFailed: "Panoya yazılamadı.",
}

var (
	app appState

	indexedCount int64
	scanning     int32
	scanID       int64

	priorityRootKeys []string
	userRootKeys     []string
	noisyRootKeys    []string

	cfPreferredDropEffect uint16
	oleKeepAlive          sync.Map
)

func main() {
	runtime.LockOSThread()

	app.language = loadLanguage()
	initPathPriority()

	initCommonControls()
	initOleVTables()
	procOleInitialize.Call(0)
	defer procOleUninitialize.Call()

	cfPreferredDropEffect = uint16(registerClipboardFormat("Preferred DropEffect"))

	hInst, _, _ := procGetModuleHandle.Call(0)
	cursor, _, _ := procLoadCursor.Call(0, IDC_ARROW)
	icon, _, _ := procLoadIcon.Call(0, IDI_APPLICATION)

	className := utf16Ptr(appClassName)
	wc := WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		LpfnWndProc:   syscall.NewCallback(wndProc),
		HInstance:     hInst,
		HIcon:         icon,
		HCursor:       cursor,
		HbrBackground: COLOR_BTNFACE + 1,
		LpszClassName: className,
		HIconSm:       icon,
	}
	procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))

	hwnd, _, _ := procCreateWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(utf16Ptr(appTitle))),
		WS_OVERLAPPEDWINDOW,
		100,
		100,
		winW,
		winH,
		0,
		0,
		hInst,
		0,
	)
	if hwnd == 0 {
		return
	}
	app.hwnd = hwnd

	procShowWindow.Call(hwnd, SW_SHOW)
	procUpdateWindow.Call(hwnd)

	var msg MSG
	for {
		ret, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		if msg.Message == WM_KEYDOWN && msg.WParam == 27 { // 27 = VK_ESCAPE (Escape key)
			procSetWindowText.Call(app.hSearch, uintptr(unsafe.Pointer(utf16Ptr(""))))
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}
}
