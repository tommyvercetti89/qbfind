//go:build windows

package main

import (
	"sort"
	"strings"
	"unsafe"
)

var (
	sortColumn    int32 = -1
	sortAscending bool  = true
)

func clearList() {
	procSendMessage.Call(app.hList, WM_SETREDRAW, 0, 0)
	procSendMessage.Call(app.hList, LVM_SETITEMCOUNT, 0, 0)
	procSendMessage.Call(app.hList, WM_SETREDRAW, 1, 0)
	procInvalidateRect.Call(app.hList, 0, 1)
}

func fillList(results []fileEntry, keepPath string) {
	procSendMessage.Call(app.hList, WM_SETREDRAW, 0, 0)
	procSendMessage.Call(app.hList, LVM_SETITEMCOUNT, uintptr(len(results)), 0)
	procSendMessage.Call(app.hList, WM_SETREDRAW, 1, 0)
	procInvalidateRect.Call(app.hList, 0, 1)
	if idx, ok := findResultIndex(keepPath); ok {
		selectListRow(idx)
	}
}

func findResultIndex(path string) (int, bool) {
	if path == "" {
		return 0, false
	}
	for i, e := range app.results {
		if strings.EqualFold(e.Path, path) {
			return i, true
		}
	}
	return 0, false
}

func fillListDisplayInfo(info *NMLVDISPINFO) {
	idx := int(info.Item.IItem)
	if idx < 0 || idx >= len(app.results) {
		return
	}
	e := app.results[idx]
	switch info.Item.ISubItem {
	case 0:
		info.Item.PszText = utf16Ptr(e.Name)
		info.Item.IImage = iconIndexFor(e)
	case 1:
		info.Item.PszText = utf16Ptr(e.Path)
	case 2:
		info.Item.PszText = utf16Ptr(formatEntrySize(e))
	case 3:
		info.Item.PszText = utf16Ptr(e.ModTime.Format("2006-01-02 15:04"))
	}
}

func selectedEntry() (fileEntry, bool) {
	entries := selectedEntries()
	if len(entries) == 0 {
		return fileEntry{}, false
	}
	return entries[0], true
}

func selectedEntries() []fileEntry {
	var out []fileEntry
	start := ^uintptr(0)
	for {
		ret, _, _ := procSendMessage.Call(app.hList, LVM_GETNEXTITEM, start, LVNI_SELECTED)
		idx := int32(ret)
		if idx < 0 || int(idx) >= len(app.results) {
			return out
		}
		out = append(out, app.results[idx])
		start = uintptr(idx)
	}
}

func sortResults(col int) {
	if col < 0 || col > 3 || len(app.results) < 2 {
		return
	}
	if int(sortColumn) == col {
		sortAscending = !sortAscending
	} else {
		sortColumn = int32(col)
		sortAscending = true
	}
	sort.SliceStable(app.results, func(i, j int) bool {
		a, b := app.results[i], app.results[j]
		cmp := 0
		switch col {
		case 0:
			cmp = strings.Compare(a.LowerName, b.LowerName)
		case 1:
			cmp = strings.Compare(a.LowerPath, b.LowerPath)
		case 2:
			if a.Size < b.Size {
				cmp = -1
			} else if a.Size > b.Size {
				cmp = 1
			}
		default:
			if a.ModTime.Before(b.ModTime) {
				cmp = -1
			} else if a.ModTime.After(b.ModTime) {
				cmp = 1
			}
		}
		if !sortAscending {
			cmp = -cmp
		}
		return cmp < 0
	})
	updateSortIndicator()
	procInvalidateRect.Call(app.hList, 0, 1)
}

func selectedPath() string {
	e, ok := selectedEntry()
	if !ok {
		return ""
	}
	return e.Path
}

func selectListRow(row int) {
	if row < 0 {
		return
	}
	clear := LVITEM{
		Mask:      LVIF_STATE,
		StateMask: LVIS_SELECTED | LVIS_FOCUSED,
	}
	procSendMessage.Call(app.hList, LVM_SETITEMSTATE, ^uintptr(0), uintptr(unsafe.Pointer(&clear)))
	item := LVITEM{
		Mask:      LVIF_STATE,
		State:     LVIS_SELECTED | LVIS_FOCUSED,
		StateMask: LVIS_SELECTED | LVIS_FOCUSED,
	}
	procSendMessage.Call(app.hList, LVM_SETITEMSTATE, uintptr(row), uintptr(unsafe.Pointer(&item)))
	procSendMessage.Call(app.hList, LVM_ENSUREVISIBLE, uintptr(row), 0)
}

func selectListItemAtScreen(x, y int32) bool {
	if app.hList == 0 {
		return false
	}
	pt := POINT{X: x, Y: y}
	procScreenToClient.Call(app.hList, uintptr(unsafe.Pointer(&pt)))
	hit := LVHITTESTINFO{Pt: pt}
	ret, _, _ := procSendMessage.Call(app.hList, LVM_HITTEST, 0, uintptr(unsafe.Pointer(&hit)))
	if int32(ret) < 0 || hit.IItem < 0 || int(hit.IItem) >= len(app.results) {
		return false
	}
	selectListRow(int(hit.IItem))
	return true
}

func showContextMenu(x, y int32) {
	if _, ok := selectedEntry(); !ok {
		return
	}
	s := ui()
	menu, _, _ := procCreatePopupMenu.Call()
	appendMenu(menu, MF_STRING, idBtnOpen, s.Open)
	appendMenu(menu, MF_STRING, idBtnPreview, s.Preview)
	appendMenu(menu, MF_STRING, idBtnCopyPath, s.CopyPath)
	appendMenu(menu, MF_STRING, idBtnCut, s.Cut)
	appendMenu(menu, MF_SEPARATOR, 0, "")
	appendMenu(menu, MF_STRING, idBtnRename, s.Rename)
	appendMenu(menu, MF_STRING, idBtnDelete, s.Delete)
	appendMenu(menu, MF_SEPARATOR, 0, "")
	appendMenu(menu, MF_STRING, idBtnExplorer, s.MenuShowExplorer)
	appendMenu(menu, MF_STRING, idBtnDesktop, s.MenuCopyDesktop)
	appendMenu(menu, MF_SEPARATOR, 0, "")
	appendMenu(menu, MF_STRING, idBtnInfo, s.Info)
	procTrackPopupMenu.Call(menu, TPM_RIGHTBUTTON, uintptr(x), uintptr(y), 0, app.hwnd, 0)
	procDestroyMenu.Call(menu)
}
