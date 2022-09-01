// +build windows

package clr

import (
	"fmt"
	"syscall"
	"unsafe"
)

// SafeArray represents a safe array
// defined in OAIdl.h
// typedef struct tagSAFEARRAY {
//   USHORT         cDims;
//   USHORT         fFeatures;
//   ULONG          cbElements;
//   ULONG          cLocks;
//   PVOID          pvData;
//   SAFEARRAYBOUND rgsabound[1];
// } SAFEARRAY;
// https://docs.microsoft.com/en-us/windows/win32/api/oaidl/ns-oaidl-safearray
// https://docs.microsoft.com/en-us/archive/msdn-magazine/2017/march/introducing-the-safearray-data-structure
type SafeArray struct {
	// cDims is the number of dimensions
	cDims uint16
	// fFeatures is the feature flags
	fFeatures uint16
	// cbElements is the size of an array element
	cbElements uint32
	// cLocks is the number of times the array has been locked without a corresponding unlock
	cLocks uint32
	// pvData is the data
	pvData uintptr
	// rgsabout is one bound for each dimension
	rgsabound [1]SafeArrayBound
}

// SafeArrayBound represents the bounds of one dimension of the array
// typedef struct tagSAFEARRAYBOUND {
//   ULONG cElements;
//   LONG  lLbound;
// } SAFEARRAYBOUND, *LPSAFEARRAYBOUND;
// https://docs.microsoft.com/en-us/windows/win32/api/oaidl/ns-oaidl-safearraybound
type SafeArrayBound struct {
	// cElements is the number of elements in the dimension
	cElements uint32
	// lLbound is the lowerbound of the dimension
	lLbound int32
}

// CreateSafeArray is a wrapper function that takes in a Go byte array and creates a SafeArray containing unsigned bytes
// by making two syscalls and copying raw memory into the correct spot.
func CreateSafeArray(rawBytes []byte) (*SafeArray, error) {
	debugPrint("Entering into safearray.CreateSafeArray()...")

	safeArrayBounds := SafeArrayBound{
		cElements: uint32(len(rawBytes)),
		lLbound:   int32(0),
	}

	safeArray, err := SafeArrayCreate(VT_UI1, 1, &safeArrayBounds)
	if err != nil {
		return nil, err
	}
	// now we need to use RtlCopyMemory to copy our bytes to the SafeArray
	modNtDll := syscall.MustLoadDLL("ntdll.dll")
	procRtlCopyMemory := modNtDll.MustFindProc("RtlCopyMemory")

	// TODO Replace RtlCopyMemory with SafeArrayPutElement or SafeArrayAccessData

	// void RtlCopyMemory(
	//   void*       Destination,
	//   const void* Source,
	//   size_t      Length
	// );
	// https://docs.microsoft.com/en-us/windows-hardware/drivers/ddi/wdm/nf-wdm-rtlcopymemory
	_, _, err = procRtlCopyMemory.Call(
		safeArray.pvData,
		uintptr(unsafe.Pointer(&rawBytes[0])),
		uintptr(len(rawBytes)),
	)

	if err != syscall.Errno(0) {
		return nil, err
	}

	return safeArray, nil
}

// SafeArrayCreate creates a new array descriptor, allocates and initializes the data for the array, and returns a pointer to the new array descriptor.
// SAFEARRAY * SafeArrayCreate(
//   VARTYPE        vt,
//   UINT           cDims,
//   SAFEARRAYBOUND *rgsabound
// );
// Varient types: https://docs.microsoft.com/en-us/windows/win32/api/wtypes/ne-wtypes-varenum
// https://docs.microsoft.com/en-us/windows/win32/api/oleauto/nf-oleauto-safearraycreate
func SafeArrayCreate(vt uint16, cDims uint32, rgsabound *SafeArrayBound) (safeArray *SafeArray, err error) {
	debugPrint("Entering into safearray.SafeArrayCreate()...")

	modOleAuto := syscall.MustLoadDLL("OleAut32.dll")
	procSafeArrayCreate := modOleAuto.MustFindProc("SafeArrayCreate")

	ret, _, err := procSafeArrayCreate.Call(
		uintptr(vt),
		uintptr(cDims),
		uintptr(unsafe.Pointer(rgsabound)),
	)

	if err != syscall.Errno(0) {
		return
	}
	err = nil

	if ret == 0 {
		err = fmt.Errorf("the OleAut32!SafeArrayCreate function return 0x%x and the SafeArray was not created", ret)
		return
	}

	// Unable to avoid misuse of unsafe.Pointer because the Windows API call returns the safeArray pointer in the "ret" value. This is a go vet false positive
	safeArray = (*SafeArray)(unsafe.Pointer(ret))
	return
}

// SysAllocString converts a Go string to a BTSR string, that is a unicode string prefixed with its length.
// Allocates a new string and copies the passed string into it.
// It returns a pointer to the string's content.
//  BSTR SysAllocString(
//    const OLECHAR *psz
//  );
// https://docs.microsoft.com/en-us/windows/win32/api/oleauto/nf-oleauto-sysallocstring
func SysAllocString(str string) (unsafe.Pointer, error) {
	debugPrint("Entering into safearray.SysAllocString()...")

	modOleAuto := syscall.MustLoadDLL("OleAut32.dll")
	sysAllocString := modOleAuto.MustFindProc("SysAllocString")

	input := utf16Le(str)
	ret, _, err := sysAllocString.Call(
		uintptr(unsafe.Pointer(&input[0])),
	)

	if err != syscall.Errno(0) {
		return nil, err
	}
	// TODO Return a pointer to a BSTR instead of an unsafe.Pointer
	// Unable to avoid misuse of unsafe.Pointer because the Windows API call returns the safeArray pointer in the "ret" value. This is a go vet false positive
	return unsafe.Pointer(ret), nil
}

// SafeArrayPutElement pushes an element to the safe array at a given index
//  HRESULT SafeArrayPutElement(
//	  SAFEARRAY *psa,
//	  LONG      *rgIndices,
//	  void      *pv
//  );
// https://docs.microsoft.com/en-us/windows/win32/api/oleauto/nf-oleauto-safearrayputelement
func SafeArrayPutElement(psa *SafeArray, rgIndices int32, pv unsafe.Pointer) error {
	debugPrint("Entering into safearray.SafeArrayPutElement()...")

	modOleAuto := syscall.MustLoadDLL("OleAut32.dll")
	safeArrayPutElement := modOleAuto.MustFindProc("SafeArrayPutElement")

	hr, _, err := safeArrayPutElement.Call(
		uintptr(unsafe.Pointer(psa)),
		uintptr(unsafe.Pointer(&rgIndices)),
		uintptr(pv),
	)
	if err != syscall.Errno(0) {
		return err
	}
	if hr != S_OK {
		return fmt.Errorf("the OleAut32!SafeArrayPutElement call returned a non-zero HRESULT: 0x%x", hr)
	}
	return nil
}

// SafeArrayLock increments the lock count of an array, and places a pointer to the array data in pvData of the array descriptor
// HRESULT SafeArrayLock(
//   SAFEARRAY *psa
// );
// https://docs.microsoft.com/en-us/windows/win32/api/oleauto/nf-oleauto-safearraylock
func SafeArrayLock(psa *SafeArray) error {
	debugPrint("Entering into safearray.SafeArrayLock()...")

	modOleAuto := syscall.MustLoadDLL("OleAut32.dll")
	safeArrayCreate := modOleAuto.MustFindProc("SafeArrayCreate")

	hr, _, err := safeArrayCreate.Call(uintptr(unsafe.Pointer(psa)))

	if err != syscall.Errno(0) {
		return err
	}

	if hr != S_OK {
		return fmt.Errorf("the OleAut32!SafeArrayCreate function returned a non-zero HRESULT: 0x%x", hr)
	}

	return nil
}

// SafeArrayGetVartype gets the VARTYPE stored in the specified safe array
// HRESULT SafeArrayGetVartype(
//   SAFEARRAY *psa,
//   VARTYPE   *pvt
// );
// https://docs.microsoft.com/en-us/windows/win32/api/oleauto/nf-oleauto-safearraygetvartype
func SafeArrayGetVartype(psa *SafeArray) (uint16, error) {
	debugPrint("Entering into safearray.SafeArrayGetVartype()...")

	var vt uint16

	modOleAuto := syscall.MustLoadDLL("OleAut32.dll")
	safeArrayGetVartype := modOleAuto.MustFindProc("SafeArrayGetVartype")

	hr, _, err := safeArrayGetVartype.Call(
		uintptr(unsafe.Pointer(psa)),
		uintptr(unsafe.Pointer(&vt)),
	)

	if err != syscall.Errno(0) {
		return 0, err
	}
	if hr != S_OK {
		return 0, fmt.Errorf("the OleAut32!SafeArrayGetVartype function returned a non-zero HRESULT: 0x%x", hr)
	}
	return vt, nil
}

// SafeArrayAccessData increments the lock count of an array, and retrieves a pointer to the array data
// HRESULT SafeArrayAccessData(
//   SAFEARRAY  *psa,
//   void HUGEP **ppvData
// );
// https://docs.microsoft.com/en-us/windows/win32/api/oleauto/nf-oleauto-safearrayaccessdata
func SafeArrayAccessData(psa *SafeArray) (*uintptr, error) {
	debugPrint("Entering into safearray.SafeArrayAccessData()...")

	var ppvData *uintptr

	modOleAuto := syscall.MustLoadDLL("OleAut32.dll")
	safeArrayAccessData := modOleAuto.MustFindProc("SafeArrayAccessData")

	hr, _, err := safeArrayAccessData.Call(
		uintptr(unsafe.Pointer(psa)),
		uintptr(unsafe.Pointer(&ppvData)),
	)

	if err != syscall.Errno(0) {
		return nil, err
	}
	if hr != S_OK {
		return nil, fmt.Errorf("the oleaut32!SafeArrayAccessData function returned a non-zero HRESULT: 0x%x", hr)
	}
	return ppvData, nil
}

// SafeArrayGetLBound gets the lower bound for any dimension of the specified safe array
// HRESULT SafeArrayGetLBound(
//   SAFEARRAY *psa,
//   UINT      nDim,
//   LONG      *plLbound
// );
// https://docs.microsoft.com/en-us/windows/win32/api/oleauto/nf-oleauto-safearraygetlbound
func SafeArrayGetLBound(psa *SafeArray, nDim uint32) (uint32, error) {
	debugPrint("Entering into safearray.SafeArrayGetLBound()...")
	var plLbound uint32
	modOleAuto := syscall.MustLoadDLL("OleAut32.dll")
	safeArrayGetLBound := modOleAuto.MustFindProc("SafeArrayGetLBound")

	hr, _, err := safeArrayGetLBound.Call(
		uintptr(unsafe.Pointer(psa)),
		uintptr(nDim),
		uintptr(unsafe.Pointer(&plLbound)),
	)

	if err != syscall.Errno(0) {
		return 0, err
	}
	if hr != S_OK {
		return 0, fmt.Errorf("the oleaut32!SafeArrayGetLBound function returned a non-zero HRESULT: 0x%x", hr)
	}
	return plLbound, nil
}

// SafeArrayGetUBound gets the upper bound for any dimension of the specified safe array
// HRESULT SafeArrayGetUBound(
//   SAFEARRAY *psa,
//   UINT      nDim,
//   LONG      *plUbound
// );
// https://docs.microsoft.com/en-us/windows/win32/api/oleauto/nf-oleauto-safearraygetubound
func SafeArrayGetUBound(psa *SafeArray, nDim uint32) (uint32, error) {
	debugPrint("Entering into safearray.SafeArrayGetUBound()...")

	var plUbound uint32

	modOleAuto := syscall.MustLoadDLL("OleAut32.dll")
	safeArrayGetUBound := modOleAuto.MustFindProc("SafeArrayGetUBound")

	hr, _, err := safeArrayGetUBound.Call(
		uintptr(unsafe.Pointer(psa)),
		uintptr(nDim),
		uintptr(unsafe.Pointer(&plUbound)),
	)

	if err != syscall.Errno(0) {
		return 0, err
	}
	if hr != S_OK {
		return 0, fmt.Errorf("the oleaut32!SafeArrayGetUBound function returned a non-zero HRESULT: 0x%x", hr)
	}
	return plUbound, nil
}

// SafeArrayDestroy Destroys an existing array descriptor and all of the data in the array.
// If objects are stored in the array, Release is called on each object in the array.
// HRESULT SafeArrayDestroy(
//   SAFEARRAY *psa
// );
func SafeArrayDestroy(psa *SafeArray) error {
	debugPrint("Entering into safearray.SafeArrayDestroy()...")

	modOleAuto := syscall.MustLoadDLL("OleAut32.dll")
	safeArrayDestroy := modOleAuto.MustFindProc("SafeArrayDestroy")

	hr, _, err := safeArrayDestroy.Call(
		uintptr(unsafe.Pointer(psa)),
		0,
		0,
	)

	if err != syscall.Errno(0) {
		return fmt.Errorf("the oleaut32!SafeArrayDestroy function call returned an error:\n%s", err)
	}
	if hr != S_OK {
		return fmt.Errorf("the oleaut32!SafeArrayDestroy function returned a non-zero HRESULT: 0x%x", hr)
	}
	return nil
}
