package d3d

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modD3D11              = windows.NewLazySystemDLL("d3d11.dll")
	procD3D11CreateDevice = modD3D11.NewProc("D3D11CreateDevice")

	// iid_ID3D11Device1, _   = windows.GUIDFromString("{a04bfb29-08ef-43d6-a49c-a9bdbdcbe686}")
	iid_ID3D11Texture2D, _ = windows.GUIDFromString("{6f15aaf2-d208-4e89-9ab4-489535d34f9c}")
	iid_ID3D11Debug, _     = windows.GUIDFromString("{79cf2233-7536-4948-9d36-1e4692dc5760}")
	iid_ID3D11InfoQueue, _ = windows.GUIDFromString("{6543dbb6-1b48-42f5-ab82-e97ec74326f6}")
)

const (
	D3D11_USAGE_DEFAULT = 0
	D3D11_USAGE_STAGING = 3

	D3D11_CPU_ACCESS_READ = 0x20000

	D3D11_RLDO_SUMMARY         = 0x1
	D3D11_RLDO_DETAIL          = 0x2
	D3D11_RLDO_IGNORE_INTERNAL = 0x4

	D3D11_CREATE_DEVICE_DEBUG        = 0x2
	D3D11_CREATE_DEVICE_BGRA_SUPPORT = 0x20

	D3D11_SDK_VERSION = 7
)

func _D3D11CreateDevice(ppDevice **ID3D11Device, ppDeviceContext **ID3D11DeviceContext) error {
	var factory1 *IDXGIFactory1
	if err := _CreateDXGIFactory1(&factory1); err != nil {
		return fmt.Errorf("CreateDXGIFactory1: %w", err)
	}
	defer factory1.Release()

	var adapter1 *IDXGIAdapter1
	if hr := factory1.EnumAdapters1(0, &adapter1); failed(hr) {
		return fmt.Errorf("failed to enumerate desktop adapter. %w", HRESULT(hr))
	}
	defer adapter1.Release()

	fflags := [...]uint32{
		// 0xc100, // D3D_FEATURE_LEVEL_12_1
		// 0xc000, // D3D_FEATURE_LEVEL_12_0
		0xb100, // D3D_FEATURE_LEVEL_11_1
		0xb000, // D3D_FEATURE_LEVEL_11_0
		// 0xa100, // D3D_FEATURE_LEVEL_10_1
		// 0xa000, // D3D_FEATURE_LEVEL_10_0
		// 0x9300, // D3D_FEATURE_LEVEL_9_3
		// 0x9200, // D3D_FEATURE_LEVEL_9_2
		// 0x9100, // D3D_FEATURE_LEVEL_9_1
		// 0x1000, // D3D_FEATURE_LEVEL_1_0_CORE <-- unsupported!
	}
	featureLevel := 0x9100
	flags :=
		//  D3D11_CREATE_DEVICE_DEBUG |
		0

	ret, _, _ := syscall.Syscall12(
		procD3D11CreateDevice.Addr(),
		10,
		uintptr(unsafe.Pointer(adapter1)),   // pAdapter
		uintptr(0),                          // driverType: 1 = Hardware
		0,                                   // software
		uintptr(flags),                      // flags
		uintptr(unsafe.Pointer(&fflags[0])), // supported feature levels
		uintptr(len(fflags)),                // number of levels
		uintptr(D3D11_SDK_VERSION),
		uintptr(unsafe.Pointer(ppDevice)),        // *D3D11Device
		uintptr(unsafe.Pointer(&featureLevel)),   // feature level
		uintptr(unsafe.Pointer(ppDeviceContext)), // *D3D11DeviceContext
		0,
		0,
	)

	if ret != 0 {
		return HRESULT(ret)
	}
	return nil
}

func NewD3D11Device() (*ID3D11Device, *ID3D11DeviceContext, error) {
	var device *ID3D11Device
	var deviceCtx *ID3D11DeviceContext

	err := _D3D11CreateDevice(&device, &deviceCtx)

	if err != nil || device == nil || deviceCtx == nil {
		return nil, nil, err
	}

	return device, deviceCtx, nil
}

type ID3D11Texture2D struct {
	vtbl *iD3D11Texture2DVtbl
}

func (obj *ID3D11Texture2D) GetDesc(desc *_D3D11_TEXTURE2D_DESC) int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.GetDesc,
		2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(desc)),
		0,
	)
	return int32(ret)
}
func (obj *ID3D11Texture2D) Release() int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.Release,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0,
	)
	return int32(ret)
}
func (obj *ID3D11Texture2D) QueryInterface(iid windows.GUID, pp interface{}) int32 {
	return reflectQueryInterface(obj, obj.vtbl.QueryInterface, &iid, pp)
}

type ID3D11Device struct {
	vtbl *iD3D11DeviceVtbl
}

func (obj *ID3D11Device) QueryInterface(iid windows.GUID, pp interface{}) int32 {
	return reflectQueryInterface(obj, obj.vtbl.QueryInterface, &iid, pp)
}

func (obj *ID3D11Device) CreateTexture2D(desc *_D3D11_TEXTURE2D_DESC, ppTexture2D **ID3D11Texture2D) int32 {
	ret, _, _ := syscall.Syscall6(
		obj.vtbl.CreateTexture2D,
		4,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(desc)),
		0,
		uintptr(unsafe.Pointer(ppTexture2D)),
		0,
		0,
	)
	return int32(ret)
}

func (obj *ID3D11Device) Release() int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.Release,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0,
	)
	return int32(ret)
}

type ID3D11Device1 struct {
	vtbl *iD3D11DeviceVtbl
}

func (obj *ID3D11Device1) Release() int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.Release,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0,
	)
	return int32(ret)
}

func (obj *ID3D11Device1) CreateTexture2D(desc *_D3D11_TEXTURE2D_DESC, ppTexture2D **ID3D11Texture2D) int32 {
	ret, _, _ := syscall.Syscall6(
		obj.vtbl.CreateTexture2D,
		4,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(desc)),
		0,
		uintptr(unsafe.Pointer(ppTexture2D)),
		0,
		0,
	)
	return int32(ret)
}

type ID3D11DeviceContext struct {
	vtbl *iD3D11DeviceContextVtbl
}

func (obj *ID3D11DeviceContext) CopyResourceDXGI(dst, src *IDXGIResource) int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.CopyResource,
		3,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(dst)),
		uintptr(unsafe.Pointer(src)),
	)
	return int32(ret)
}
func (obj *ID3D11DeviceContext) CopyResource2D(dst, src *ID3D11Texture2D) int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.CopyResource,
		3,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(dst)),
		uintptr(unsafe.Pointer(src)),
	)
	return int32(ret)
}
func (obj *ID3D11DeviceContext) CopySubresourceRegion2D(dst *ID3D11Texture2D, dstSubResource, dstX, dstY, dstZ uint32, src *ID3D11Texture2D, srcSubResource uint32, pSrcBox *_D3D11_BOX) int32 {
	ret, _, _ := syscall.Syscall9(
		obj.vtbl.CopySubresourceRegion,
		9,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(dst)),
		uintptr(dstSubResource),
		uintptr(dstX),
		uintptr(dstY),
		uintptr(dstZ),
		uintptr(unsafe.Pointer(src)),
		uintptr(srcSubResource),
		uintptr(unsafe.Pointer(pSrcBox)),
	)
	return int32(ret)
}

func (obj *ID3D11DeviceContext) CopySubresourceRegion(dst *ID3D11Resource, dstSubResource, dstX, dstY, dstZ uint32, src *ID3D11Resource, srcSubResource uint32, pSrcBox *_D3D11_BOX) int32 {
	ret, _, _ := syscall.Syscall9(
		obj.vtbl.CopySubresourceRegion,
		9,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(dst)),
		uintptr(dstSubResource),
		uintptr(dstX),
		uintptr(dstY),
		uintptr(dstZ),
		uintptr(unsafe.Pointer(src)),
		uintptr(srcSubResource),
		uintptr(unsafe.Pointer(pSrcBox)),
	)
	return int32(ret)
}
func (obj *ID3D11DeviceContext) Release() int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.Release,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0,
	)
	return int32(ret)
}

type ID3D11Resource struct {
	vtbl *iD3D11ResourceVtbl
}

func (obj *ID3D11Resource) Release() int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.Release,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0,
	)
	return int32(ret)
}
