package wininet

import (
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var wininet *windows.LazyDLL = windows.NewLazySystemDLL("Wininet")

// HTTPAddRequestHeadersW is from wininet.h
func HTTPAddRequestHeadersW(
	reqHndl uintptr,
	header string,
	addMethod uintptr,
) error {
	var err error
	var ok uintptr
	var proc string = "HttpAddRequestHeadersW"
	var pswzHeader uintptr
	var tmp *uint16

	if header == "" {
		// Weird, just do nothing
		return nil
	}
	header = strings.TrimSpace(header) + "\r\n"

	// Convert to Windows types
	if tmp, err = windows.UTF16PtrFromString(header); err != nil {
		return convertFail(header, err)
	}

	pswzHeader = uintptr(unsafe.Pointer(tmp))

	ok, _, err = wininet.NewProc(proc).Call(
		reqHndl,
		pswzHeader,
		uintptr(len(header)),
		addMethod,
	)
	if ok == 0 {
		return fmt.Errorf("%s: %w", proc, err)
	}

	return nil
}

// HTTPOpenRequestW is from wininet.h
func HTTPOpenRequestW(
	connHndl uintptr,
	verb string,
	objectName string,
	version string,
	referrer string,
	acceptTypes []string,
	flags uintptr,
	context uintptr,
) (uintptr, error) {
	var err error
	var lpcwstrObjectName uintptr
	var lpcwstrReferrer uintptr
	var lpcwstrVerb uintptr
	var lpcwstrVersion uintptr
	var lplpcwstrAcceptTypes []*uint16
	var proc string = "HttpOpenRequestW"
	var reqHndl uintptr
	var tmp *uint16

	// Convert to Windows types
	lplpcwstrAcceptTypes = make([]*uint16, 1)
	for _, theType := range acceptTypes {
		if theType == "" {
			continue
		}

		tmp, err = windows.UTF16PtrFromString(theType)
		if err != nil {
			return 0, convertFail(theType, err)
		}

		lplpcwstrAcceptTypes = append(lplpcwstrAcceptTypes, tmp)
	}

	if objectName != "" {
		if tmp, err = windows.UTF16PtrFromString(objectName); err != nil {
			return 0, convertFail(objectName, err)
		}

		lpcwstrObjectName = uintptr(unsafe.Pointer(tmp))
	}

	if referrer != "" {
		if tmp, err = windows.UTF16PtrFromString(referrer); err != nil {
			return 0, convertFail(referrer, err)
		}

		lpcwstrReferrer = uintptr(unsafe.Pointer(tmp))
	}

	if verb != "" {
		if tmp, err = windows.UTF16PtrFromString(verb); err != nil {
			return 0, convertFail(verb, err)
		}

		lpcwstrVerb = uintptr(unsafe.Pointer(tmp))
	}

	if version != "" {
		if tmp, err = windows.UTF16PtrFromString(version); err != nil {
			return 0, convertFail(version, err)
		}

		lpcwstrVersion = uintptr(unsafe.Pointer(tmp))
	}

	reqHndl, _, err = wininet.NewProc(proc).Call(
		connHndl,
		lpcwstrVerb,
		lpcwstrObjectName,
		lpcwstrVersion,
		lpcwstrReferrer,
		uintptr(unsafe.Pointer(&lplpcwstrAcceptTypes[0])),
		flags,
		context,
	)
	if reqHndl == 0 {
		return 0, fmt.Errorf("%s: %w", proc, err)
	}

	return reqHndl, nil
}

// HTTPQueryInfoW is from wininet.h
func HTTPQueryInfoW(
	reqHndl uintptr,
	info uintptr,
	buffer *[]byte,
	bufferLen *int,
	index *int,
) error {
	var b []uint16
	var err error
	var proc string = "HttpQueryInfoW"
	var success uintptr
	var tmp string

	if *bufferLen > 0 {
		b = make([]uint16, *bufferLen)
	} else {
		b = make([]uint16, 1)
	}

	success, _, err = wininet.NewProc(proc).Call(
		reqHndl,
		info,
		uintptr(unsafe.Pointer(&b[0])),
		uintptr(unsafe.Pointer(bufferLen)),
		uintptr(unsafe.Pointer(index)),
	)
	if success == 0 {
		return fmt.Errorf("%s: %w", proc, err)
	}

	tmp = windows.UTF16ToString(b)
	*buffer = []byte(tmp)

	return nil
}

// HTTPSendRequestW is from wininet.h
func HTTPSendRequestW(
	reqHndl uintptr,
	headers string,
	headersLen int,
	data []byte,
	dataLen int,
) error {
	var body uintptr
	var err error
	var lpcwstrHeaders uintptr
	var proc string = "HttpSendRequestW"
	var success uintptr
	var tmp *uint16

	// Pointer to data if provided
	if (data != nil) && (len(data) > 0) {
		body = uintptr(unsafe.Pointer(&data[0]))
	}

	// Convert to Windows types
	if headersLen > 0 {
		if tmp, err = windows.UTF16PtrFromString(headers); err != nil {
			return convertFail(headers, err)
		}

		lpcwstrHeaders = uintptr(unsafe.Pointer(tmp))
	}

	success, _, err = wininet.NewProc(proc).Call(
		reqHndl,
		lpcwstrHeaders,
		uintptr(headersLen),
		body,
		uintptr(dataLen),
	)
	if success == 0 {
		return fmt.Errorf("%s: %w", proc, err)
	}

	return nil
}

// InternetConnectW is from wininet.h
func InternetConnectW(
	sessionHndl uintptr,
	serverName string,
	serverPort int,
	username string,
	password string,
	service uintptr,
	flags uintptr,
	context uintptr,
) (uintptr, error) {
	var connHndl uintptr
	var err error
	var lpcwstrServerName uintptr
	var lpcwstrUserName uintptr
	var lpcwstrPassword uintptr
	var proc string = "InternetConnectW"
	var tmp *uint16

	// Convert to Windows types
	if password != "" {
		if tmp, err = windows.UTF16PtrFromString(password); err != nil {
			return 0, convertFail(password, err)
		}

		lpcwstrPassword = uintptr(unsafe.Pointer(tmp))
	}

	if serverName != "" {
		if tmp, err = windows.UTF16PtrFromString(serverName); err != nil {
			return 0, convertFail(serverName, err)
		}

		lpcwstrServerName = uintptr(unsafe.Pointer(tmp))
	}

	if username != "" {
		if tmp, err = windows.UTF16PtrFromString(username); err != nil {
			return 0, convertFail(username, err)
		}

		lpcwstrUserName = uintptr(unsafe.Pointer(tmp))
	}

	connHndl, _, err = wininet.NewProc(proc).Call(
		sessionHndl,
		lpcwstrServerName,
		uintptr(serverPort),
		lpcwstrUserName,
		lpcwstrPassword,
		service,
		flags,
		context,
	)
	if connHndl == 0 {
		return 0, fmt.Errorf("%s: %w", proc, err)
	}

	return connHndl, nil
}

// InternetOpenW is from wininet.h
func InternetOpenW(
	userAgent string,
	accessType uintptr,
	proxy string,
	proxyBypass string,
	flags uintptr,
) (uintptr, error) {
	var err error
	var lpszAgent uintptr
	var lpszProxy uintptr
	var lpszProxyBypass uintptr
	var proc string = "InternetOpenW"
	var sessionHndl uintptr
	var tmp *uint16

	// Convert to Windows types
	if userAgent != "" {
		if tmp, err = windows.UTF16PtrFromString(userAgent); err != nil {
			return 0, convertFail(userAgent, err)
		}

		lpszAgent = uintptr(unsafe.Pointer(tmp))
	}

	if proxy != "" {
		if tmp, err = windows.UTF16PtrFromString(proxy); err != nil {
			return 0, convertFail(proxy, err)
		}

		lpszProxy = uintptr(unsafe.Pointer(tmp))
	}

	if proxyBypass != "" {
		tmp, err = windows.UTF16PtrFromString(proxyBypass)
		if err != nil {
			return 0, convertFail(proxyBypass, err)
		}

		lpszProxyBypass = uintptr(unsafe.Pointer(tmp))
	}

	sessionHndl, _, err = wininet.NewProc(proc).Call(
		lpszAgent,
		accessType,
		lpszProxy,
		lpszProxyBypass,
		flags,
	)
	if sessionHndl == 0 {
		return 0, fmt.Errorf("%s: %w", proc, err)
	}

	return sessionHndl, nil
}

// InternetQueryDataAvailable is from wininet.h
func InternetQueryDataAvailable(
	reqHndl uintptr,
	bytesAvailable *int64,
) error {
	var err error
	var proc string = "InternetQueryDataAvailable"
	var success uintptr

	success, _, err = wininet.NewProc(proc).Call(
		reqHndl,
		uintptr(unsafe.Pointer(bytesAvailable)),
		0,
		0,
	)
	if success == 0 {
		return fmt.Errorf("%s: %w", proc, err)
	}

	return nil
}

// InternetReadFile is from wininet.h
func InternetReadFile(
	reqHndl uintptr,
	buffer *[]byte,
	bytesToRead int64,
	bytesRead *int64,
) error {
	var b []byte
	var err error
	var proc string = "InternetReadFile"
	var success uintptr

	if bytesToRead > 0 {
		b = make([]byte, bytesToRead)
	} else {
		b = make([]byte, 1)
	}

	success, _, err = wininet.NewProc(proc).Call(
		reqHndl,
		uintptr(unsafe.Pointer(&b[0])),
		uintptr(bytesToRead),
		uintptr(unsafe.Pointer(bytesRead)),
	)
	if success == 0 {
		return fmt.Errorf("%s: %w", proc, err)
	}

	*buffer = b

	return nil
}

// InternetSetOptionW is from wininet.h
func InternetSetOptionW(
	hndl uintptr,
	opt uintptr,
	val []byte,
	valLen int,
) error {
	var err error
	var proc string = "InternetSetOptionW"
	var success uintptr

	// Pointer to data if provided
	if valLen == 0 {
		val = make([]byte, 1)
	}

	success, _, err = wininet.NewProc(proc).Call(
		hndl,
		opt,
		uintptr(unsafe.Pointer(&val[0])),
		uintptr(valLen),
	)
	if success == 0 {
		return fmt.Errorf("%s: %w", proc, err)
	}

	return nil
}

// InternetErrorDlg is from wininet.h
func InternetErrorDlg(
	hWnd uintptr,
	hRequest uintptr,
	dwError uint32,
	dwFlags uint32,
	lppvData *[]byte,
) (uintptr, error) {
	var buf []byte
	var proc string = "InternetErrorDlg"
	var success uintptr

	buf = make([]byte, 1024) //arbitrary size (safe?)

	success, _, _ = wininet.NewProc(proc).Call(
		uintptr(hWnd),
		hRequest,
		uintptr(dwError),
		uintptr(dwFlags),
		uintptr(unsafe.Pointer(&buf[0])),
	)
	// we expect 12032 if user hits okay, otherwise we return the response code as an error
	if uint32(success) != ERROR_INTERNET_FORCE_RETRY {
		return success, fmt.Errorf("%s: %s", proc, success)
	}

	*lppvData = buf

	return success, nil
}
