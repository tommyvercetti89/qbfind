//go:build windows

package main

import (
	"sync/atomic"
	"syscall"
	"unsafe"
)

type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

var (
	iidIUnknown       = GUID{0x00000000, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	iidIDataObject    = GUID{0x0000010e, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	iidIDropSource    = GUID{0x00000121, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	iidIEnumFORMATETC = GUID{0x00000103, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
)

type FORMATETC struct {
	CfFormat uint16
	Ptd      uintptr
	DwAspect uint32
	Lindex   int32
	Tymed    uint32
}

type STGMEDIUM struct {
	Tymed          uint32
	HGlobal        uintptr
	PUnkForRelease uintptr
}

type DROPFILES struct {
	PFiles uint32
	Pt     POINT
	FNC    int32
	FWide  int32
}

type dataObjectVtbl struct {
	QueryInterface     uintptr
	AddRef             uintptr
	Release            uintptr
	GetData            uintptr
	GetDataHere        uintptr
	QueryGetData       uintptr
	GetCanonicalFormat uintptr
	SetData            uintptr
	EnumFormatEtc      uintptr
	DAdvise            uintptr
	DUnadvise          uintptr
	EnumDAdvise        uintptr
}

type dataObject struct {
	lpVtbl *dataObjectVtbl
	ref    int32
	files  []string
}

var dataObjectVtblInst = dataObjectVtbl{
	QueryInterface:     syscall.NewCallback(dataObjectQueryInterface),
	AddRef:             syscall.NewCallback(dataObjectAddRef),
	Release:            syscall.NewCallback(dataObjectRelease),
	GetData:            syscall.NewCallback(dataObjectGetData),
	GetDataHere:        syscall.NewCallback(dataObjectGetDataHere),
	QueryGetData:       syscall.NewCallback(dataObjectQueryGetData),
	GetCanonicalFormat: syscall.NewCallback(dataObjectGetCanonicalFormat),
	SetData:            syscall.NewCallback(dataObjectSetData),
	EnumFormatEtc:      syscall.NewCallback(dataObjectEnumFormatEtc),
	DAdvise:            syscall.NewCallback(dataObjectDAdvise),
	DUnadvise:          syscall.NewCallback(dataObjectDUnadvise),
	EnumDAdvise:        syscall.NewCallback(dataObjectEnumDAdvise),
}

func doDragFiles(files []string) {
	if len(files) == 0 {
		return
	}
	data := &dataObject{lpVtbl: &dataObjectVtblInst, ref: 1, files: append([]string(nil), files...)}
	source := &dropSource{lpVtbl: &dropSourceVtblInst, ref: 1}
	var effect uint32
	procDoDragDrop.Call(
		uintptr(unsafe.Pointer(data)),
		uintptr(unsafe.Pointer(source)),
		DROPEFFECT_COPY|DROPEFFECT_MOVE,
		uintptr(unsafe.Pointer(&effect)),
	)
	runtimeKeepAlive(data)
	runtimeKeepAlive(source)
}

func dataObjectQueryInterface(this uintptr, riid uintptr, ppv uintptr) uintptr {
	defer callbackGuard("dataObjectQueryInterface")
	if ppv == 0 {
		return E_POINTER
	}
	*(*uintptr)(unsafe.Pointer(ppv)) = 0
	if isGUID(riid, &iidIUnknown) || isGUID(riid, &iidIDataObject) {
		*(*uintptr)(unsafe.Pointer(ppv)) = this
		dataObjectAddRef(this)
		return S_OK
	}
	return E_NOINTERFACE
}

func dataObjectAddRef(this uintptr) uintptr {
	defer callbackGuard("dataObjectAddRef")
	obj := (*dataObject)(unsafe.Pointer(this))
	return uintptr(atomic.AddInt32(&obj.ref, 1))
}

func dataObjectRelease(this uintptr) uintptr {
	defer callbackGuard("dataObjectRelease")
	obj := (*dataObject)(unsafe.Pointer(this))
	n := atomic.AddInt32(&obj.ref, -1)
	return uintptr(n)
}

func dataObjectGetData(this uintptr, pFormat uintptr, pMedium uintptr) uintptr {
	defer callbackGuard("dataObjectGetData")
	if pFormat == 0 || pMedium == 0 {
		return E_INVALIDARG
	}
	obj := (*dataObject)(unsafe.Pointer(this))
	format := (*FORMATETC)(unsafe.Pointer(pFormat))
	medium := (*STGMEDIUM)(unsafe.Pointer(pMedium))
	*medium = STGMEDIUM{}

	if format.CfFormat == CF_HDROP && format.Tymed&TYMED_HGLOBAL != 0 {
		h := createHDrop(obj.files)
		if h == 0 {
			return E_INVALIDARG
		}
		medium.Tymed = TYMED_HGLOBAL
		medium.HGlobal = h
		return S_OK
	}

	if cfPreferredDropEffect != 0 && format.CfFormat == cfPreferredDropEffect && format.Tymed&TYMED_HGLOBAL != 0 {
		h := createDropEffect(DROPEFFECT_COPY)
		if h == 0 {
			return E_INVALIDARG
		}
		medium.Tymed = TYMED_HGLOBAL
		medium.HGlobal = h
		return S_OK
	}

	return DV_E_FORMATETC
}

func dataObjectGetDataHere(this uintptr, pFormat uintptr, pMedium uintptr) uintptr {
	defer callbackGuard("dataObjectGetDataHere")
	return E_NOTIMPL
}

func dataObjectQueryGetData(this uintptr, pFormat uintptr) uintptr {
	defer callbackGuard("dataObjectQueryGetData")
	if pFormat == 0 {
		return E_INVALIDARG
	}
	format := (*FORMATETC)(unsafe.Pointer(pFormat))
	if format.CfFormat == CF_HDROP && format.Tymed&TYMED_HGLOBAL != 0 {
		return S_OK
	}
	if cfPreferredDropEffect != 0 && format.CfFormat == cfPreferredDropEffect && format.Tymed&TYMED_HGLOBAL != 0 {
		return S_OK
	}
	return DV_E_FORMATETC
}

func dataObjectGetCanonicalFormat(this uintptr, pFormatIn uintptr, pFormatOut uintptr) uintptr {
	defer callbackGuard("dataObjectGetCanonicalFormat")
	if pFormatOut != 0 {
		*(*FORMATETC)(unsafe.Pointer(pFormatOut)) = FORMATETC{}
	}
	return E_NOTIMPL
}

func dataObjectSetData(this uintptr, pFormat uintptr, pMedium uintptr, release uintptr) uintptr {
	defer callbackGuard("dataObjectSetData")
	return E_NOTIMPL
}

func dataObjectEnumFormatEtc(this uintptr, direction uintptr, ppEnum uintptr) uintptr {
	defer callbackGuard("dataObjectEnumFormatEtc")
	if ppEnum == 0 {
		return E_POINTER
	}
	*(*uintptr)(unsafe.Pointer(ppEnum)) = 0
	if direction != DATADIR_GET {
		return E_NOTIMPL
	}
	enum := newFormatEnum(0)
	*(*uintptr)(unsafe.Pointer(ppEnum)) = uintptr(unsafe.Pointer(enum))
	return S_OK
}

func dataObjectDAdvise(this uintptr, pFormat uintptr, advf uintptr, sink uintptr, connection uintptr) uintptr {
	defer callbackGuard("dataObjectDAdvise")
	return OLE_E_ADVISENOTSUPPORTED
}

func dataObjectDUnadvise(this uintptr, connection uintptr) uintptr {
	defer callbackGuard("dataObjectDUnadvise")
	return OLE_E_ADVISENOTSUPPORTED
}

func dataObjectEnumDAdvise(this uintptr, ppEnum uintptr) uintptr {
	defer callbackGuard("dataObjectEnumDAdvise")
	return OLE_E_ADVISENOTSUPPORTED
}

type dropSourceVtbl struct {
	QueryInterface    uintptr
	AddRef            uintptr
	Release           uintptr
	QueryContinueDrag uintptr
	GiveFeedback      uintptr
}

type dropSource struct {
	lpVtbl *dropSourceVtbl
	ref    int32
}

var dropSourceVtblInst = dropSourceVtbl{
	QueryInterface:    syscall.NewCallback(dropSourceQueryInterface),
	AddRef:            syscall.NewCallback(dropSourceAddRef),
	Release:           syscall.NewCallback(dropSourceRelease),
	QueryContinueDrag: syscall.NewCallback(dropSourceQueryContinueDrag),
	GiveFeedback:      syscall.NewCallback(dropSourceGiveFeedback),
}

func dropSourceQueryInterface(this uintptr, riid uintptr, ppv uintptr) uintptr {
	defer callbackGuard("dropSourceQueryInterface")
	if ppv == 0 {
		return E_POINTER
	}
	*(*uintptr)(unsafe.Pointer(ppv)) = 0
	if isGUID(riid, &iidIUnknown) || isGUID(riid, &iidIDropSource) {
		*(*uintptr)(unsafe.Pointer(ppv)) = this
		dropSourceAddRef(this)
		return S_OK
	}
	return E_NOINTERFACE
}

func dropSourceAddRef(this uintptr) uintptr {
	defer callbackGuard("dropSourceAddRef")
	src := (*dropSource)(unsafe.Pointer(this))
	return uintptr(atomic.AddInt32(&src.ref, 1))
}

func dropSourceRelease(this uintptr) uintptr {
	defer callbackGuard("dropSourceRelease")
	src := (*dropSource)(unsafe.Pointer(this))
	return uintptr(atomic.AddInt32(&src.ref, -1))
}

func dropSourceQueryContinueDrag(this uintptr, escapePressed uintptr, keyState uintptr) uintptr {
	defer callbackGuard("dropSourceQueryContinueDrag")
	if escapePressed != 0 {
		return DRAGDROP_S_CANCEL
	}
	if keyState&MK_LBUTTON == 0 {
		return DRAGDROP_S_DROP
	}
	return S_OK
}

func dropSourceGiveFeedback(this uintptr, effect uintptr) uintptr {
	defer callbackGuard("dropSourceGiveFeedback")
	return DRAGDROP_S_USEDEFAULTCURSORS
}

type enumFormatVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	Next           uintptr
	Skip           uintptr
	Reset          uintptr
	Clone          uintptr
}

type enumFormat struct {
	lpVtbl  *enumFormatVtbl
	ref     int32
	index   int32
	formats []FORMATETC
}

var enumFormatVtblInst enumFormatVtbl

func initOleVTables() {
	enumFormatVtblInst = enumFormatVtbl{
		QueryInterface: syscall.NewCallback(enumFormatQueryInterface),
		AddRef:         syscall.NewCallback(enumFormatAddRef),
		Release:        syscall.NewCallback(enumFormatRelease),
		Next:           syscall.NewCallback(enumFormatNext),
		Skip:           syscall.NewCallback(enumFormatSkip),
		Reset:          syscall.NewCallback(enumFormatReset),
		Clone:          syscall.NewCallback(enumFormatClone),
	}
}

func newFormatEnum(index int32) *enumFormat {
	formats := []FORMATETC{
		{CfFormat: CF_HDROP, DwAspect: DVASPECT_CONTENT, Lindex: -1, Tymed: TYMED_HGLOBAL},
	}
	if cfPreferredDropEffect != 0 {
		formats = append(formats, FORMATETC{CfFormat: cfPreferredDropEffect, DwAspect: DVASPECT_CONTENT, Lindex: -1, Tymed: TYMED_HGLOBAL})
	}
	enum := &enumFormat{lpVtbl: &enumFormatVtblInst, ref: 1, index: index, formats: formats}
	oleKeepAlive.Store(uintptr(unsafe.Pointer(enum)), enum)
	return enum
}

func enumFormatQueryInterface(this uintptr, riid uintptr, ppv uintptr) uintptr {
	defer callbackGuard("enumFormatQueryInterface")
	if ppv == 0 {
		return E_POINTER
	}
	*(*uintptr)(unsafe.Pointer(ppv)) = 0
	if isGUID(riid, &iidIUnknown) || isGUID(riid, &iidIEnumFORMATETC) {
		*(*uintptr)(unsafe.Pointer(ppv)) = this
		enumFormatAddRef(this)
		return S_OK
	}
	return E_NOINTERFACE
}

func enumFormatAddRef(this uintptr) uintptr {
	defer callbackGuard("enumFormatAddRef")
	enum := (*enumFormat)(unsafe.Pointer(this))
	return uintptr(atomic.AddInt32(&enum.ref, 1))
}

func enumFormatRelease(this uintptr) uintptr {
	defer callbackGuard("enumFormatRelease")
	enum := (*enumFormat)(unsafe.Pointer(this))
	n := atomic.AddInt32(&enum.ref, -1)
	if n <= 0 {
		oleKeepAlive.Delete(this)
	}
	return uintptr(n)
}

func enumFormatNext(this uintptr, celt uintptr, rgelt uintptr, pceltFetched uintptr) uintptr {
	defer callbackGuard("enumFormatNext")
	if rgelt == 0 {
		return E_POINTER
	}
	enum := (*enumFormat)(unsafe.Pointer(this))
	fetched := uintptr(0)
	itemSize := unsafe.Sizeof(FORMATETC{})
	for fetched < celt && int(enum.index) < len(enum.formats) {
		dst := (*FORMATETC)(unsafe.Pointer(rgelt + fetched*itemSize))
		*dst = enum.formats[enum.index]
		enum.index++
		fetched++
	}
	if pceltFetched != 0 {
		*(*uint32)(unsafe.Pointer(pceltFetched)) = uint32(fetched)
	}
	if fetched == celt {
		return S_OK
	}
	return S_FALSE
}

func enumFormatSkip(this uintptr, celt uintptr) uintptr {
	defer callbackGuard("enumFormatSkip")
	enum := (*enumFormat)(unsafe.Pointer(this))
	enum.index += int32(celt)
	if int(enum.index) > len(enum.formats) {
		enum.index = int32(len(enum.formats))
		return S_FALSE
	}
	return S_OK
}

func enumFormatReset(this uintptr) uintptr {
	defer callbackGuard("enumFormatReset")
	enum := (*enumFormat)(unsafe.Pointer(this))
	enum.index = 0
	return S_OK
}

func enumFormatClone(this uintptr, ppEnum uintptr) uintptr {
	defer callbackGuard("enumFormatClone")
	if ppEnum == 0 {
		return E_POINTER
	}
	enum := (*enumFormat)(unsafe.Pointer(this))
	clone := newFormatEnum(enum.index)
	*(*uintptr)(unsafe.Pointer(ppEnum)) = uintptr(unsafe.Pointer(clone))
	return S_OK
}

func createHDrop(files []string) uintptr {
	encoded := utf16Multi(files)
	headerSize := unsafe.Sizeof(DROPFILES{})
	total := headerSize + uintptr(len(encoded))*2
	h, _, _ := procGlobalAlloc.Call(GMEM_MOVEABLE|GMEM_ZEROINIT, total)
	if h == 0 {
		return 0
	}
	p, _, _ := procGlobalLock.Call(h)
	if p == 0 {
		procGlobalFree.Call(h)
		return 0
	}
	df := (*DROPFILES)(unsafe.Pointer(p))
	df.PFiles = uint32(headerSize)
	df.FWide = 1
	dst := (*[1 << 28]uint16)(unsafe.Pointer(p + headerSize))[:len(encoded):len(encoded)]
	copy(dst, encoded)
	procGlobalUnlock.Call(h)
	return h
}

func createDropEffect(effect uint32) uintptr {
	h, _, _ := procGlobalAlloc.Call(GMEM_MOVEABLE|GMEM_ZEROINIT, 4)
	if h == 0 {
		return 0
	}
	p, _, _ := procGlobalLock.Call(h)
	if p == 0 {
		procGlobalFree.Call(h)
		return 0
	}
	*(*uint32)(unsafe.Pointer(p)) = effect
	procGlobalUnlock.Call(h)
	return h
}

func isGUID(p uintptr, want *GUID) bool {
	if p == 0 {
		return false
	}
	got := (*GUID)(unsafe.Pointer(p))
	return got.Data1 == want.Data1 && got.Data2 == want.Data2 && got.Data3 == want.Data3 && got.Data4 == want.Data4
}

func initCommonControls() {
	icc := INITCOMMONCONTROLSEX{
		DwSize: uint32(unsafe.Sizeof(INITCOMMONCONTROLSEX{})),
		DwICC:  ICC_LISTVIEW_CLASSES,
	}
	procInitCommonControlsEx.Call(uintptr(unsafe.Pointer(&icc)))
}
