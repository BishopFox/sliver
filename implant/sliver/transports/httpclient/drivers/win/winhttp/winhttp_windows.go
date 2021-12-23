package winhttp

import (
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var winhttp *windows.LazyDLL = windows.NewLazySystemDLL("Winhttp")

// AddRequestHeaders is WinHttpAddRequestHeaders from winhttp.h
func AddRequestHeaders(
	reqHndl uintptr,
	header string,
	addMethod uintptr,
) error {
	var e error
	var ok uintptr
	var proc string = "WinHttpAddRequestHeaders"
	var pswzHeader uintptr
	var tmp *uint16

	if header == "" {
		// Weird, just do nothing
		return nil
	}

	header = strings.TrimSpace(header) + "\r\n"

	// Convert to Windows types
	if tmp, e = windows.UTF16PtrFromString(header); e != nil {
		return convertFail(header, e)
	}

	pswzHeader = uintptr(unsafe.Pointer(tmp))

	ok, _, e = winhttp.NewProc(proc).Call(
		reqHndl,
		pswzHeader,
		uintptr(len(header)),
		addMethod,
	)
	if ok == 0 {
		return fmt.Errorf("%s: %w", proc, e)
	}

	return nil
}

// Connect is WinHttpConnect from winhttp.h
func Connect(
	sessionHndl uintptr,
	serverName string,
	serverPort int,
) (uintptr, error) {
	var connHndl uintptr
	var e error
	var proc string = "WinHttpConnect"
	var pswzServerName uintptr
	var tmp *uint16

	// Convert to Windows types
	if serverName != "" {
		if tmp, e = windows.UTF16PtrFromString(serverName); e != nil {
			return 0, convertFail(serverName, e)
		}

		pswzServerName = uintptr(unsafe.Pointer(tmp))
	}

	connHndl, _, e = winhttp.NewProc(proc).Call(
		sessionHndl,
		pswzServerName,
		uintptr(serverPort),
		0,
	)
	if connHndl == 0 {
		return 0, fmt.Errorf("%s: %w", proc, e)
	}

	return connHndl, nil
}

// Open is WinHttpOpen from winhttp.h
func Open(
	userAgent string,
	accessType uintptr,
	proxy string,
	proxyBypass string,
	flags uintptr,
) (uintptr, error) {
	var e error
	var proc string = "WinHttpOpen"
	var pszAgent uintptr
	var pszProxy uintptr
	var pszProxyBypass uintptr
	var sessionHndl uintptr
	var tmp *uint16

	// Convert to Windows types
	if userAgent != "" {
		if tmp, e = windows.UTF16PtrFromString(userAgent); e != nil {
			return 0, convertFail(userAgent, e)
		}

		pszAgent = uintptr(unsafe.Pointer(tmp))
	}

	if proxy != "" {
		if tmp, e = windows.UTF16PtrFromString(proxy); e != nil {
			return 0, convertFail(proxy, e)
		}

		pszProxy = uintptr(unsafe.Pointer(tmp))
	}

	if proxyBypass != "" {
		tmp, e = windows.UTF16PtrFromString(proxyBypass)
		if e != nil {
			return 0, convertFail(proxyBypass, e)
		}

		pszProxyBypass = uintptr(unsafe.Pointer(tmp))
	}

	sessionHndl, _, e = winhttp.NewProc(proc).Call(
		pszAgent,
		accessType,
		pszProxy,
		pszProxyBypass,
		flags,
	)
	if sessionHndl == 0 {
		return 0, fmt.Errorf("%s: %w", proc, e)
	}

	return sessionHndl, nil
}

// OpenRequest is WinHttpOpenRequest from winhttp.h
func OpenRequest(
	connHndl uintptr,
	verb string,
	objectName string,
	version string,
	referrer string,
	acceptTypes []string,
	flags uintptr,
) (uintptr, error) {
	var e error
	var ppwszAcceptTypes []*uint16
	var proc string = "WinHttpOpenRequest"
	var pwszObjectName uintptr
	var pwszReferrer uintptr
	var pwszVerb uintptr
	var pwszVersion uintptr
	var reqHndl uintptr
	var tmp *uint16

	// Convert to Windows types
	ppwszAcceptTypes = make([]*uint16, 1)
	for _, theType := range acceptTypes {
		if theType == "" {
			continue
		}

		tmp, e = windows.UTF16PtrFromString(theType)
		if e != nil {
			return 0, convertFail(theType, e)
		}

		ppwszAcceptTypes = append(ppwszAcceptTypes, tmp)
	}

	if objectName != "" {
		if tmp, e = windows.UTF16PtrFromString(objectName); e != nil {
			return 0, convertFail(objectName, e)
		}

		pwszObjectName = uintptr(unsafe.Pointer(tmp))
	}

	if referrer != "" {
		if tmp, e = windows.UTF16PtrFromString(referrer); e != nil {
			return 0, convertFail(referrer, e)
		}

		pwszReferrer = uintptr(unsafe.Pointer(tmp))
	}

	if verb != "" {
		if tmp, e = windows.UTF16PtrFromString(verb); e != nil {
			return 0, convertFail(verb, e)
		}

		pwszVerb = uintptr(unsafe.Pointer(tmp))
	}

	if version != "" {
		if tmp, e = windows.UTF16PtrFromString(version); e != nil {
			return 0, convertFail(version, e)
		}

		pwszVersion = uintptr(unsafe.Pointer(tmp))
	}

	reqHndl, _, e = winhttp.NewProc(proc).Call(
		connHndl,
		pwszVerb,
		pwszObjectName,
		pwszVersion,
		pwszReferrer,
		uintptr(unsafe.Pointer(&ppwszAcceptTypes[0])),
		flags,
	)
	if reqHndl == 0 {
		return 0, fmt.Errorf("%s: %w", proc, e)
	}

	return reqHndl, nil
}

// QueryDataAvailable is WinHttpQueryDataAvailable from winhttp.h
func QueryDataAvailable(reqHndl uintptr, bytesToRead *int64) error {
	var e error
	var proc string = "WinHttpQueryDataAvailable"
	var success uintptr

	success, _, e = winhttp.NewProc(proc).Call(
		reqHndl,
		uintptr(unsafe.Pointer(bytesToRead)),
	)
	if success == 0 {
		return fmt.Errorf("%s: %w", proc, e)
	}

	return nil
}

// QueryHeaders is WinHttpQueryHeaders from winhttp.h
func QueryHeaders(
	reqHndl uintptr,
	info uintptr,
	name string,
	buffer *[]byte,
	bufferLen *int,
	index *int,
) error {
	var b []uint16
	var e error
	var proc string = "WinHttpQueryHeaders"
	var pwszName uintptr
	var success uintptr
	var tmp *uint16

	// Convert to Windows types
	if *bufferLen > 0 {
		b = make([]uint16, *bufferLen)
	} else {
		b = make([]uint16, 1)
	}

	if (name != "") && (info == WinhttpQueryCustom) {
		if tmp, e = windows.UTF16PtrFromString(name); e != nil {
			return convertFail(name, e)
		}

		pwszName = uintptr(unsafe.Pointer(tmp))
	} else {
		pwszName = WinhttpHeaderNameByIndex
	}

	success, _, e = winhttp.NewProc(proc).Call(
		reqHndl,
		info,
		pwszName,
		uintptr(unsafe.Pointer(&b[0])),
		uintptr(unsafe.Pointer(bufferLen)),
		uintptr(unsafe.Pointer(index)),
	)
	if success == 0 {
		return fmt.Errorf("%s: %w", proc, e)
	}

	*buffer = []byte(windows.UTF16ToString(b))

	return nil
}

// ReadData is WinHttpReadData from winhttp.h
func ReadData(
	reqHndl uintptr,
	buffer *[]byte,
	bytesToRead int64,
	bytesRead *int64,
) error {
	var b []byte
	var e error
	var proc string = "WinHttpReadData"
	var success uintptr

	if bytesToRead > 0 {
		b = make([]byte, bytesToRead)
	} else {
		b = make([]byte, 1)
	}

	success, _, e = winhttp.NewProc(proc).Call(
		reqHndl,
		uintptr(unsafe.Pointer(&b[0])),
		uintptr(bytesToRead),
		uintptr(unsafe.Pointer(bytesRead)),
	)
	if success == 0 {
		return fmt.Errorf("%s: %w", proc, e)
	}

	*buffer = b

	return nil
}

// ReceiveResponse is WinHttpReceiveResponse from winhttp.h
func ReceiveResponse(reqHndl uintptr) error {
	var e error
	var proc string = "WinHttpReceiveResponse"
	var success uintptr

	success, _, e = winhttp.NewProc(proc).Call(reqHndl, 0)
	if success == 0 {
		return fmt.Errorf("%s: %w", proc, e)
	}

	return nil
}

// SendRequest is WinHttpSendRequest from winhttp.h
func SendRequest(
	reqHndl uintptr,
	headers string,
	headersLen int,
	data []byte,
	dataLen int,
) error {
	var body uintptr
	var e error
	var proc string = "WinHttpSendRequest"
	var pwszHeaders uintptr
	var success uintptr
	var tmp *uint16

	// Pointer to data if provided
	if (data != nil) && (len(data) > 0) {
		body = uintptr(unsafe.Pointer(&data[0]))
	}

	// Convert to Windows types
	if headers != "" {
		if tmp, e = windows.UTF16PtrFromString(headers); e != nil {
			return convertFail(headers, e)
		}

		pwszHeaders = uintptr(unsafe.Pointer(tmp))
	}

	success, _, e = winhttp.NewProc(proc).Call(
		reqHndl,
		pwszHeaders,
		uintptr(headersLen),
		body,
		uintptr(dataLen),
		uintptr(dataLen),
	)
	if success == 0 {
		return fmt.Errorf("%s: %w", proc, e)
	}

	return nil
}

// SetOption is WinHttpSetOption from winhttp.h
func SetOption(hndl, opt uintptr, val []byte, valLen int) error {
	var e error
	var proc string = "WinHttpSetOption"
	var success uintptr

	// Pointer to data if provided
	if valLen == 0 {
		val = make([]byte, 1)
	}

	success, _, e = winhttp.NewProc(proc).Call(
		hndl,
		opt,
		uintptr(unsafe.Pointer(&val[0])),
		uintptr(valLen),
	)
	if success == 0 {
		return fmt.Errorf("%s: %w", proc, e)
	}

	return nil
}
