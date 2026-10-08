//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	settingsHwnd       uintptr
	settingsEdit       uintptr
	settingsStatus     uintptr
	settingsLabel      uintptr
	settingsHint       uintptr
	settingsDrivesLbl  uintptr
	settingsDrivesEdit uintptr
	settingsDrivesHint uintptr
	settingsSaveBtn    uintptr
	settingsCloseBtn   uintptr
	settingsValue      string
	settingsDrivesVal  string
)

func registerSettingsClass(hInst uintptr, cursor uintptr) {
	wc := WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		LpfnWndProc:   syscall.NewCallback(settingsWndProc),
		HInstance:     hInst,
		HCursor:       cursor,
		HbrBackground: COLOR_BTNFACE + 1,
		LpszClassName: utf16Ptr(settingsClassName),
	}
	procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
}

func openSettingsWindow() {
	if settingsHwnd != 0 {
		procShowWindow.Call(settingsHwnd, SW_SHOW)
		procSetForegroundWindow.Call(settingsHwnd)
		return
	}
	settingsValue = loadSettings()["exclude"]
	settingsDrivesVal = loadSettings()["drives"]
	hInst, _, _ := procGetModuleHandle.Call(0)
	var mainRect RECT
	procGetWindowRect.Call(app.hwnd, uintptr(unsafe.Pointer(&mainRect)))
	width := int32(540)
	height := int32(460)
	x := mainRect.Left + ((mainRect.Right-mainRect.Left)-width)/2
	y := mainRect.Top + ((mainRect.Bottom-mainRect.Top)-height)/2
	hwnd, _, _ := procCreateWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(utf16Ptr(settingsClassName))),
		uintptr(unsafe.Pointer(utf16Ptr(ui().SettingsTitle))),
		WS_POPUP|WS_CAPTION|WS_SYSMENU|WS_VISIBLE,
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		app.hwnd,
		0,
		hInst,
		0,
	)
	if hwnd != 0 {
		settingsHwnd = hwnd
	}
}

func settingsWndProc(hwnd uintptr, msg uint32, wParam uintptr, lParam uintptr) (result uintptr) {
	defer func() {
		if r := recover(); r != nil {
			logPanic("settingsWndProc", r)
			result = 0
		}
	}()
	switch msg {
	case WM_CREATE:
		createSettingsControls(hwnd)
		return 0
	case WM_SIZE:
		layoutSettingsControls(hwnd)
		return 0
	case WM_COMMAND:
		switch loword(wParam) {
		case idExcludeSave:
			setExclusions(getWindowText(settingsEdit))
			setDrives(getWindowText(settingsDrivesEdit))
			procSetWindowText.Call(settingsStatus, uintptr(unsafe.Pointer(utf16Ptr(ui().SettingsSaved))))
		case idExcludeClose:
			procSendMessage.Call(hwnd, WM_CLOSE, 0, 0)
		}
		return 0
	case WM_CLOSE:
		// DefWindowProc destroys the window
	case WM_DESTROY:
		settingsHwnd = 0
		settingsEdit = 0
		settingsStatus = 0
		settingsLabel = 0
		settingsHint = 0
		settingsDrivesLbl = 0
		settingsDrivesEdit = 0
		settingsDrivesHint = 0
		settingsSaveBtn = 0
		settingsCloseBtn = 0
		return 0
	}
	ret, _, _ := procDefWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func createSettingsControls(hwnd uintptr) {
	s := ui()
	settingsLabel = createStatic(hwnd, 0, s.SettingsExcludeLabel)
	settingsHint = createStatic(hwnd, 0, s.SettingsExcludeHint)
	settingsEdit = createWindow("EDIT", settingsValue, WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|WS_VSCROLL|ES_MULTILINE|ES_AUTOVSCROLL, 0, 0, 0, 0, hwnd, idExcludeEdit)
	settingsDrivesLbl = createStatic(hwnd, 0, s.SettingsDrivesLabel)
	settingsDrivesHint = createStatic(hwnd, 0, s.SettingsDrivesHint)
	settingsDrivesEdit = createWindow("EDIT", settingsDrivesVal, WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL, 0, 0, 0, 0, hwnd, 0)
	settingsStatus = createStatic(hwnd, 0, "")
	settingsSaveBtn = createButton(hwnd, idExcludeSave, s.Save)
	settingsCloseBtn = createButton(hwnd, idExcludeClose, s.Close)
	setFont(settingsLabel)
	setFont(settingsHint)
	setFont(settingsEdit)
	setFont(settingsDrivesLbl)
	setFont(settingsDrivesHint)
	setFont(settingsDrivesEdit)
	setFont(settingsStatus)
}

func layoutSettingsControls(hwnd uintptr) {
	if settingsEdit == 0 {
		return
	}
	var rc RECT
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	w := rc.Right - rc.Left
	h := rc.Bottom - rc.Top
	margin := int32(12)
	buttonW := int32(90)
	buttonH := int32(28)
	labelH := int32(18)
	hintH := int32(34)
	statusH := int32(18)
	driveEditH := int32(24)

	buttonsY := h - buttonH - margin
	statusY := buttonsY - 6 - statusH
	drivesHintY := statusY - 6 - hintH
	drivesEditY := drivesHintY - 6 - driveEditH
	drivesLabelY := drivesEditY - 4 - labelH
	excludeHintY := drivesLabelY - 8 - hintH
	editY := margin + labelH + 6
	editH := excludeHintY - 6 - editY
	if editH < 50 {
		editH = 50
	}

	procMoveWindow.Call(settingsLabel, uintptr(margin), uintptr(margin), uintptr(w-margin*2), uintptr(labelH), 1)
	procMoveWindow.Call(settingsEdit, uintptr(margin), uintptr(editY), uintptr(w-margin*2), uintptr(editH), 1)
	procMoveWindow.Call(settingsHint, uintptr(margin), uintptr(excludeHintY), uintptr(w-margin*2), uintptr(hintH), 1)
	procMoveWindow.Call(settingsDrivesLbl, uintptr(margin), uintptr(drivesLabelY), uintptr(w-margin*2), uintptr(labelH), 1)
	procMoveWindow.Call(settingsDrivesEdit, uintptr(margin), uintptr(drivesEditY), uintptr(w-margin*2), uintptr(driveEditH), 1)
	procMoveWindow.Call(settingsDrivesHint, uintptr(margin), uintptr(drivesHintY), uintptr(w-margin*2), uintptr(hintH), 1)
	procMoveWindow.Call(settingsStatus, uintptr(margin), uintptr(statusY), uintptr(w-margin*2), uintptr(statusH), 1)
	procMoveWindow.Call(settingsSaveBtn, uintptr(w-margin-buttonW*2-8), uintptr(buttonsY), uintptr(buttonW), uintptr(buttonH), 1)
	procMoveWindow.Call(settingsCloseBtn, uintptr(w-margin-buttonW), uintptr(buttonsY), uintptr(buttonW), uintptr(buttonH), 1)
}
