// +build windows

package clr

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

var Debug = false

// checkOK evaluates a HRESULT code for a caller and determines if there was an error
func checkOK(hr uintptr, caller string) error {
	if hr != S_OK {
		return fmt.Errorf("%s returned 0x%08x", caller, hr)
	} else {
		return nil
	}
}

func utf16Le(s string) []byte {
	enc := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder()
	var buf bytes.Buffer
	t := transform.NewWriter(&buf, enc)
	t.Write([]byte(s))
	return buf.Bytes()
}

func expectsParams(input string) bool {
	return !strings.Contains(input, "Void Main()")
}

// ReadUnicodeStr takes a pointer to a unicode string in memory and returns a string value
func ReadUnicodeStr(ptr unsafe.Pointer) string {
	debugPrint("Entering into utils.ReadUnicodeStr()...")
	var byteVal uint16
	out := make([]uint16, 0)
	for i := 0; ; i++ {
		byteVal = *(*uint16)(unsafe.Pointer(ptr))
		if byteVal == 0x0000 {
			break
		}
		out = append(out, byteVal)
		ptr = unsafe.Pointer(uintptr(ptr) + 2)
	}
	return string(utf16.Decode(out))
}

// debugPrint is used to print messages only when debug has been enabled
func debugPrint(message string) {
	if Debug {
		fmt.Println("[DEBUG] " + message)
	}
}

// PrepareParameters creates a safe array of strings (arguments) nested inside a Variant object, which is itself
// appended to the final safe array
func PrepareParameters(params []string) (*SafeArray, error) {
	sab := SafeArrayBound{
		cElements: uint32(len(params)),
		lLbound:   0,
	}
	listStrSafeArrayPtr, err := SafeArrayCreate(VT_BSTR, 1, &sab) // VT_BSTR
	if err != nil {
		return nil, err
	}
	for i, p := range params {
		bstr, err := SysAllocString(p)
		if err != nil {
			return nil, err
		}
		SafeArrayPutElement(listStrSafeArrayPtr, int32(i), bstr)
	}

	paramVariant := Variant{
		VT:  VT_BSTR | VT_ARRAY, // VT_BSTR | VT_ARRAY
		Val: uintptr(unsafe.Pointer(listStrSafeArrayPtr)),
	}

	sab2 := SafeArrayBound{
		cElements: uint32(1),
		lLbound:   0,
	}
	paramsSafeArrayPtr, err := SafeArrayCreate(VT_VARIANT, 1, &sab2) // VT_VARIANT
	if err != nil {
		return nil, err
	}
	err = SafeArrayPutElement(paramsSafeArrayPtr, int32(0), unsafe.Pointer(&paramVariant))
	if err != nil {
		return nil, err
	}
	return paramsSafeArrayPtr, nil
}
