package d3d

import (
	"fmt"
	//"image"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modDXGI                = windows.NewLazySystemDLL("dxgi.dll")
	procCreateDXGIFactory1 = modDXGI.NewProc("CreateDXGIFactory1")

	// iid_IDXGIDevice, _   = windows.GUIDFromString("{54ec77fa-1377-44e6-8c32-88fd5f44c84c}")
	iid_IDXGIDevice1, _ = windows.GUIDFromString("{77db970f-6276-48ba-ba28-070143b4392c}")
	// iid_IDXGIAdapter, _  = windows.GUIDFromString("{2411E7E1-12AC-4CCF-BD14-9798E8534DC0}")
	iid_IDXGIAdapter1, _ = windows.GUIDFromString("{29038f61-3839-4626-91fd-086879011a05}")
	// iid_IDXGIOutput, _   = windows.GUIDFromString("{ae02eedb-c735-4690-8d52-5a8dc20213aa}")
	iid_IDXGIOutput1, _ = windows.GUIDFromString("{00cddea8-939b-4b83-a340-a685226666cc}")
	//iid_IDXGIOutput5, _  = windows.GUIDFromString("{80A07424-AB52-42EB-833C-0C42FD282D98}")

	iid_IDXGIFactory1, _ = windows.GUIDFromString("{770aae78-f26f-4dba-a829-253c83d1b387}")
	// iid_IDXGIResource, _ = windows.GUIDFromString("{035f3ab4-482e-4e50-b41f-8a7f8bd8960b}")
	iid_IDXGISurface, _ = windows.GUIDFromString("{cafcb56c-6ac3-4889-bf47-9e23bbd260ec}")
)

const (
	DXGI_MAP_READ    = 1 << 0
	DXGI_MAP_WRITE   = 1 << 1
	DXGI_MAP_DISCARD = 1 << 2
)

type IDXGIFactory1 struct {
	vtbl *iDXGIFactory1Vtbl
}

func (obj *IDXGIFactory1) Release() int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.Release,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0,
	)
	return int32(ret)
}

func (obj *IDXGIFactory1) EnumAdapters1(adapter uint, pp **IDXGIAdapter1) int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.EnumAdapters1,
		3,
		uintptr(unsafe.Pointer(obj)),
		uintptr(adapter),
		uintptr(unsafe.Pointer(pp)),
	)
	return int32(ret)
}

func _CreateDXGIFactory1(ppFactory **IDXGIFactory1) error {
	ret, _, _ := syscall.Syscall(
		procCreateDXGIFactory1.Addr(),
		2,
		uintptr(unsafe.Pointer(&iid_IDXGIFactory1)),
		uintptr(unsafe.Pointer(ppFactory)),
		0,
	)
	if ret != 0 {
		return HRESULT(ret)
	}

	return nil
}

// NewXASession casts your ppv from above to a *XASession
func NewIDXGIOutputDuplication(device *ID3D11Device, deviceCtx *ID3D11DeviceContext, output uint) (*OutputDuplicator, error) {
	var hr int32

	// DEBUG

	var d3dDebug *ID3D11Debug
	hr = device.QueryInterface(iid_ID3D11Debug, &d3dDebug)
	if !failed(hr) {
		defer d3dDebug.Release()

		var d3dInfoQueue *ID3D11InfoQueue
		hr = d3dDebug.QueryInterface(iid_ID3D11InfoQueue, &d3dInfoQueue)
		if failed(hr) {
			return nil, fmt.Errorf("failed at device.QueryInterface. %w", HRESULT(hr))
		}
		defer d3dInfoQueue.Release()
		// defer d3dDebug.ReportLiveDeviceObjects(D3D11_RLDO_SUMMARY | D3D11_RLDO_DETAIL)
	}

	var dxgiDevice1 *IDXGIDevice1
	hr = device.QueryInterface(iid_IDXGIDevice1, &dxgiDevice1)
	if failed(hr) {
		return nil, fmt.Errorf("failed at device.QueryInterface. %w", HRESULT(hr))
	}
	defer dxgiDevice1.Release()

	var pdxgiAdapter unsafe.Pointer
	hr = dxgiDevice1.GetParent(iid_IDXGIAdapter1, &pdxgiAdapter)
	if failed(hr) {
		return nil, fmt.Errorf("failed at dxgiDevice1.GetAdapter. %w", HRESULT(hr))
	}
	dxgiAdapter := (*IDXGIAdapter1)(pdxgiAdapter)
	defer dxgiAdapter.Release()

	var dxgiOutput *IDXGIOutput
	// const DXGI_ERROR_NOT_FOUND = 0x887A0002
	hr = int32(dxgiAdapter.EnumOutputs(output, &dxgiOutput))
	if failed(hr) {
		return nil, fmt.Errorf("failed at dxgiAdapter.EnumOutputs. %w", HRESULT(hr))
	}
	defer dxgiOutput.Release()

	var dxgiOutput1 *IDXGIOutput1
	hr = dxgiOutput.QueryInterface(iid_IDXGIOutput1, &dxgiOutput1)
	if failed(hr) {
		return nil, fmt.Errorf("failed at dxgiOutput.QueryInterface. %w", HRESULT(hr))
	}
	defer dxgiOutput1.Release()
	var dup *IDXGIOutputDuplication
	//image.NewRGBA
	hr = dxgiOutput1.DuplicateOutput(dxgiDevice1, &dup)
	if failed(hr) {
		return nil, fmt.Errorf("failed at dxgiOutput1.DuplicateOutput. %v", uint32(hr))
	}

	return &OutputDuplicator{device: device, deviceCtx: deviceCtx, outputDuplication: dup, needsSwizzle: true}, nil
}

type IDXGIAdapter1 struct {
	vtbl *IDXGIAdapter1Vtbl
}

func (obj *IDXGIAdapter1) Release() int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.Release,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0,
	)
	return int32(ret)
}

func (obj *IDXGIAdapter1) EnumOutputs(output uint, pp **IDXGIOutput) uint32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.EnumOutputs,
		3,
		uintptr(unsafe.Pointer(obj)),
		uintptr(output),
		uintptr(unsafe.Pointer(pp)),
	)
	return uint32(ret)
}

type IDXGIAdapter struct {
	vtbl *IDXGIAdapterVtbl
}

func (obj *IDXGIAdapter) EnumOutputs(output uint, pp **IDXGIOutput) uint32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.EnumOutputs,
		3,
		uintptr(unsafe.Pointer(obj)),
		uintptr(output),
		uintptr(unsafe.Pointer(pp)),
	)
	return uint32(ret)
}

func (obj *IDXGIAdapter) Release() int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.Release,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0,
	)
	return int32(ret)
}

type IDXGIDevice struct {
	vtbl *iDXGIDeviceVtbl
}

func (obj *IDXGIDevice) GetGPUThreadPriority(priority *int) int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.GetGPUThreadPriority,
		2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(priority)),
		0,
	)
	return int32(ret)
}
func (obj *IDXGIDevice) QueryInterface(iid windows.GUID, pp interface{}) int32 {
	return reflectQueryInterface(obj, obj.vtbl.QueryInterface, &iid, pp)
}
func (obj *IDXGIDevice) GetParent(iid windows.GUID, pp *unsafe.Pointer) int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.GetParent,
		3,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(&iid)),
		uintptr(unsafe.Pointer(pp)),
	)
	return int32(ret)
}
func (obj *IDXGIDevice) GetAdapter(pAdapter **IDXGIAdapter) int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.GetAdapter,
		2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(pAdapter)),
		0,
	)
	return int32(ret)
}
func (obj *IDXGIDevice) Release() int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.Release,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0,
	)
	return int32(ret)
}

type IDXGIDevice1 struct {
	vtbl *iDXGIDevice1Vtbl
}

func (obj *IDXGIDevice1) QueryInterface(iid windows.GUID, pp interface{}) int32 {
	return reflectQueryInterface(obj, obj.vtbl.QueryInterface, &iid, pp)
}

func (obj *IDXGIDevice1) GetParent(iid windows.GUID, pp *unsafe.Pointer) int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.GetParent,
		3,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(&iid)),
		uintptr(unsafe.Pointer(pp)),
	)

	return int32(ret)
}
func (obj *IDXGIDevice1) GetAdapter(pAdapter *IDXGIAdapter) int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.GetAdapter,
		2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(&pAdapter)),
		0,
	)

	return int32(ret)
}
func (obj *IDXGIDevice1) Release() int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.Release,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0,
	)
	return int32(ret)
}

type IDXGIOutput struct {
	vtbl *iDXGIOutputVtbl
}

func (obj *IDXGIOutput) QueryInterface(iid windows.GUID, pp interface{}) int32 {
	return reflectQueryInterface(obj, obj.vtbl.QueryInterface, &iid, pp)
}

func (obj *IDXGIOutput) GetParent(iid windows.GUID, pp *unsafe.Pointer) int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.GetParent,
		3,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(&iid)),
		uintptr(unsafe.Pointer(pp)),
	)
	return int32(ret)
}

func (obj *IDXGIOutput) Release() int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.Release,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0,
	)
	return int32(ret)
}

type IDXGIOutput1 struct {
	vtbl *iDXGIOutput1Vtbl
}

func (obj *IDXGIOutput1) DuplicateOutput(device1 *IDXGIDevice1, ppOutputDuplication **IDXGIOutputDuplication) int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.DuplicateOutput,
		3,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(device1)),
		uintptr(unsafe.Pointer(ppOutputDuplication)),
	)
	return int32(ret)
}

func (obj *IDXGIOutput1) GetParent(iid windows.GUID, pp *unsafe.Pointer) int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.GetParent,
		3,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(&iid)),
		uintptr(unsafe.Pointer(pp)),
	)
	return int32(ret)
}

func (obj *IDXGIOutput1) Release() int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.Release,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0,
	)
	return int32(ret)
}

type IDXGIOutput5 struct {
	vtbl *iDXGIOutput5Vtbl
}

type DXGI_FORMAT uint32

func (obj *IDXGIOutput5) DuplicateOutput1(device1 *IDXGIDevice1, flags uint, pSupportedFormats []DXGI_FORMAT, ppOutputDuplication **IDXGIOutputDuplication) int32 {
	pFormats := &pSupportedFormats[0]
	ret, _, _ := syscall.Syscall6(
		obj.vtbl.DuplicateOutput1,
		6,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(device1)),
		uintptr(flags),
		uintptr(len(pSupportedFormats)),
		uintptr(unsafe.Pointer(pFormats)),
		uintptr(unsafe.Pointer(ppOutputDuplication)),
	)
	return int32(ret)
}

func (obj *IDXGIOutput5) GetParent(iid windows.GUID, pp *unsafe.Pointer) int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.GetParent,
		3,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(&iid)),
		uintptr(unsafe.Pointer(pp)),
	)
	return int32(ret)
}

func (obj *IDXGIOutput5) Release() int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.Release,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0,
	)
	return int32(ret)
}

type IDXGIResource struct {
	vtbl *iDXGIResourceVtbl
}

func (obj *IDXGIResource) QueryInterface(iid windows.GUID, pp interface{}) int32 {
	return reflectQueryInterface(obj, obj.vtbl.QueryInterface, &iid, pp)
}
func (obj *IDXGIResource) Release() int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.Release,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0,
	)
	return int32(ret)
}

type IDXGISurface struct {
	vtbl *iDXGISurfaceVtbl
}

func (obj *IDXGISurface) QueryInterface(iid windows.GUID, pp interface{}) int32 {
	return reflectQueryInterface(obj, obj.vtbl.QueryInterface, &iid, pp)
}
func (obj *IDXGISurface) Map(pLockedRect *DXGI_MAPPED_RECT, mapFlags uint32) int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.Map,
		3,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(pLockedRect)),
		uintptr(mapFlags),
	)
	return int32(ret)
}
func (obj *IDXGISurface) Unmap() int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.Unmap,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0,
	)
	return int32(ret)
}
func (obj *IDXGISurface) Release() int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.Release,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0,
	)
	return int32(ret)
}

type IDXGIOutputDuplication struct {
	vtbl *iDXGIOutputDuplicationVtbl
}

func (obj *IDXGIOutputDuplication) GetFrameMoveRects(buffer []_DXGI_OUTDUPL_MOVE_RECT, rectsRequired *uint32) int32 {
	var buf *_DXGI_OUTDUPL_MOVE_RECT
	if len(buffer) > 0 {
		buf = &buffer[0]
	}
	size := uint32(len(buffer) * 24)
	ret, _, _ := syscall.Syscall6(
		obj.vtbl.GetFrameMoveRects,
		4,
		uintptr(unsafe.Pointer(obj)),
		uintptr(size),
		uintptr(unsafe.Pointer(buf)),
		uintptr(unsafe.Pointer(rectsRequired)),
		0,
		0,
	)
	*rectsRequired = *rectsRequired / 24
	return int32(ret)
}
func (obj *IDXGIOutputDuplication) GetFrameDirtyRects(buffer []RECT, rectsRequired *uint32) int32 {
	var buf *RECT
	if len(buffer) > 0 {
		buf = &buffer[0]
	}
	size := uint32(len(buffer) * 16)
	ret, _, _ := syscall.Syscall6(
		obj.vtbl.GetFrameDirtyRects,
		4,
		uintptr(unsafe.Pointer(obj)),
		uintptr(size),
		uintptr(unsafe.Pointer(buf)),
		uintptr(unsafe.Pointer(rectsRequired)),
		0,
		0,
	)
	*rectsRequired = *rectsRequired / 16
	return int32(ret)
}
func (obj *IDXGIOutputDuplication) GetDesc(desc *_DXGI_OUTDUPL_DESC) int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.GetDesc,
		2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(desc)),
		0,
	)
	return int32(ret)
}

func (obj *IDXGIOutputDuplication) MapDesktopSurface(pLockedRect *DXGI_MAPPED_RECT) int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.MapDesktopSurface,
		2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(pLockedRect)),
		0,
	)
	return int32(ret)
}
func (obj *IDXGIOutputDuplication) UnMapDesktopSurface() int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.UnMapDesktopSurface,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0,
	)
	return int32(ret)
}
func (obj *IDXGIOutputDuplication) AddRef() uint32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.AddRef,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0,
	)
	return uint32(ret)
}

func (obj *IDXGIOutputDuplication) Release() uint32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.Release,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0,
	)
	return uint32(ret)
}

func (obj *IDXGIOutputDuplication) AcquireNextFrame(timeoutMs uint, pFrameInfo *_DXGI_OUTDUPL_FRAME_INFO, ppDesktopResource **IDXGIResource) uint32 {
	ret, _, _ := syscall.Syscall6(
		obj.vtbl.AcquireNextFrame,    // function address
		4,                            // number of parameters to this function
		uintptr(unsafe.Pointer(obj)), // always pass the COM object address first
		uintptr(timeoutMs),           // then all function parameters follow
		uintptr(unsafe.Pointer(pFrameInfo)),
		uintptr(unsafe.Pointer(ppDesktopResource)),
		0,
		0,
	)
	return uint32(ret)
}

func (obj *IDXGIOutputDuplication) ReleaseFrame() {
	syscall.Syscall(
		obj.vtbl.ReleaseFrame,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0,
	)
}
