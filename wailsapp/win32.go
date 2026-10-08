//go:build windows

package main

import (
	"syscall"
)

const (
	SW_SHOWNORMAL = 1

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

	CF_UNICODETEXT = 13
	GMEM_MOVEABLE  = 0x0002
	GMEM_ZEROINIT  = 0x0040

	SEM_FAILCRITICALERRORS = 0x0001

	CF_HDROP       = 15
	DATADIR_GET    = 1
	DVASPECT_CONTENT = 1
	TYMED_HGLOBAL    = 1

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

	FO_DELETE = 0x0003

	FILE_LIST_DIRECTORY           = 0x0001
	FILE_SHARE_READ               = 0x00000001
	FILE_SHARE_WRITE              = 0x00000002
	FILE_SHARE_DELETE             = 0x00000004
	OPEN_EXISTING                 = 3
	FILE_FLAG_BACKUP_SEMANTICS    = 0x02000000
	INVALID_HANDLE_VALUE          = ^uintptr(0)
	FILE_NOTIFY_CHANGE_FILE_NAME  = 0x00000001
	FILE_NOTIFY_CHANGE_DIR_NAME   = 0x00000002
	FILE_NOTIFY_CHANGE_ATTRIBUTES = 0x00000004
	FILE_NOTIFY_CHANGE_SIZE       = 0x00000008
	FILE_NOTIFY_CHANGE_LAST_WRITE = 0x00000010
	FILE_ACTION_ADDED             = 1
	FILE_ACTION_REMOVED           = 2
	FILE_ACTION_MODIFIED          = 3
	FILE_ACTION_RENAMED_OLD_NAME  = 4
	FILE_ACTION_RENAMED_NEW_NAME  = 5
	FILE_NOTIFY_FILTER            = FILE_NOTIFY_CHANGE_FILE_NAME | FILE_NOTIFY_CHANGE_DIR_NAME | FILE_NOTIFY_CHANGE_ATTRIBUTES | FILE_NOTIFY_CHANGE_SIZE | FILE_NOTIFY_CHANGE_LAST_WRITE
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")

	procOpenClipboard           = user32.NewProc("OpenClipboard")
	procEmptyClipboard          = user32.NewProc("EmptyClipboard")
	procSetClipboardData        = user32.NewProc("SetClipboardData")
	procCloseClipboard          = user32.NewProc("CloseClipboard")
	procRegisterClipboardFormat = user32.NewProc("RegisterClipboardFormatW")

	procGetLogicalDrives   = kernel32.NewProc("GetLogicalDrives")
	procGetDriveType       = kernel32.NewProc("GetDriveTypeW")
	procGetDiskFreeSpaceEx    = kernel32.NewProc("GetDiskFreeSpaceExW")
	procSetErrorMode          = kernel32.NewProc("SetErrorMode")
	procCreateFileW           = kernel32.NewProc("CreateFileW")
	procReadDirectoryChangesW = kernel32.NewProc("ReadDirectoryChangesW")
	procCloseHandle           = kernel32.NewProc("CloseHandle")
	procCancelIoEx            = kernel32.NewProc("CancelIoEx")
	procGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	procGlobalLock       = kernel32.NewProc("GlobalLock")
	procGlobalUnlock     = kernel32.NewProc("GlobalUnlock")
	procGlobalFree       = kernel32.NewProc("GlobalFree")

	procShellExecute    = shell32.NewProc("ShellExecuteW")
	procSHFileOperation = shell32.NewProc("SHFileOperationW")
	procSHGetFolderPath = shell32.NewProc("SHGetFolderPathW")

	procOleInitialize   = ole32.NewProc("OleInitialize")
	procOleUninitialize = ole32.NewProc("OleUninitialize")
	procDoDragDrop      = ole32.NewProc("DoDragDrop")
)

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
