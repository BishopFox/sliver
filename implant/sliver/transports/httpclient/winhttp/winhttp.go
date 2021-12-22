//go:build windows

package winhttp

// Package winhttp provides a Golang-ish interface to the WinHTTP
// API. Because WinHTTP has a somewhat consistent pattern of use that
// contrasts with what ideomatic Golang expects, there will still be
// some rough edges (see GetOption and SetOption for examples), but
// this package should enable someone with basic understanding of
// WinHTTP and knowledge of Go to be able to use it to produce
// software making use of winhttp.dll.
//
// Some notes:
// - Error handling may look a little weird. As per
//   https://godoc.org/golang.org/x/sys/windows#LazyProc.Call , we
//   *must* check the primary return value to see if the error is
//   valid. To be more Golang-y, we do that here and return nil for
//   errors when we don't have an error condition.
// - We currently do not support WinHTTP in async operation.
// - To avoid constantly looking up procs, we cache them here in
//   private globals.
// - To help mesh one's understanding of what's going on here against
//   MSDN documentation, we're using field and argument names from MSDN.
// - Arguments and struct fields marked as reserved in MSDN are either
//   not present in function arguments here or not exported in struct
//   types. We have to include them in structs to preserve memory
//   layouts that Windows will expect.

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// HInternet is an internet handle given to us by WinHttp.
type HInternet uintptr

var wh = windows.NewLazySystemDLL("winhttp.dll")
var k32 = windows.NewLazySystemDLL("kernel32.dll")

var gF = k32.NewProc("GlobalFree")

func globalFree(ptr *uint16) {
	if ptr != nil {
		gF.Call(uintptr(unsafe.Pointer(ptr)))
	}
}

type WinHttpCurrentUserIeProxyConfig struct {
	FAutoDetect       bool
	LpszAutoConfigUrl *uint16
	LpszProxy         *uint16
	LpszProxyBypass   *uint16
}

func FreeWinHttpCurrentUserIeProxyConfig(cfg *WinHttpCurrentUserIeProxyConfig) {
	globalFree(cfg.LpszAutoConfigUrl)
	globalFree(cfg.LpszProxy)
	globalFree(cfg.LpszProxyBypass)
}

type WinHttpAutoproxyOptions struct {
	DwFlags                uint32
	DwAutoDetectFlags      uint32
	LpszAutoConfigUrl      *uint16
	lpvReserved            uintptr
	dwReserved             uint32
	FAutoLogonIfChallenged bool
}

type WinHttpProxyInfo struct {
	DwAccessType    uint32
	LpszProxy       *uint16
	LpszProxyBypass *uint16
}

func FreeWinHttpProxyInfo(cfg *WinHttpProxyInfo) {
	globalFree(cfg.LpszProxy)
	globalFree(cfg.LpszProxyBypass)
}

var closeHandle = wh.NewProc("WinHttpCloseHandle")

// CloseHandle will close an hInternet handle. It is OK if the handle
// is already NULL.
func CloseHandle(hInternet HInternet) {
	if hInternet != 0 {
		closeHandle.Call(uintptr(hInternet))
	}
}

var winHttpOpen = wh.NewProc("WinHttpOpen")

// Open creates a new WinHTTP Session for us.
func Open(userAgent string, accessType uint32, proxyName string, proxyBypass string, flags uint32) (HInternet, error) {
	var pname *uint16
	var pbypass *uint16
	if proxyName != "" {
		pname = windows.StringToUTF16Ptr(proxyName)
	}
	if proxyBypass != "" {
		pbypass = windows.StringToUTF16Ptr(proxyBypass)
	}
	h, _, err := winHttpOpen.Call(
		uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(userAgent))),
		uintptr(accessType),
		uintptr(unsafe.Pointer(pname)),
		uintptr(unsafe.Pointer(pbypass)),
		uintptr(flags),
	)

	if h != 0 {
		err = nil
	}

	return HInternet(h), err
}

var queryOption = wh.NewProc("WinHttpQueryOption")

//go:uintptrescapes

// QueryOption queries an option on the specified handle.
//
// Note that because we don't know what type of data we'll be getting
// via buffer, this function is tagged with a pragma to behave
// properly when using the form uintptr(unsafe.Pointer(&obj)) in its
// function call, similar to the wired-in compiler tricks for
// syscall.Syscall(). See the documentation to unsafe.Pointer
// (https://golang.org/pkg/unsafe/#Pointer) for more information.
func QueryOption(hInternet HInternet, option uint32, buffer uintptr, bufferLength *uint32) error {
	r, _, err := queryOption.Call(
		uintptr(hInternet),
		uintptr(option),
		buffer,
		uintptr(unsafe.Pointer(bufferLength)),
	)

	if r != 1 {
		return err
	}
	return nil
}

var setOption = wh.NewProc("WinHttpSetOption")

//go:uintptrescapes

// SetOption sets an option on the specified handle.
//
// Note that because we don't know what type of data we'll be getting
// via buffer, this function is tagged with a pragma to behave
// properly when using the form uintptr(unsafe.Pointer(&obj)) in its
// function call, similar to the wired-in compiler tricks for
// syscall.Syscall(). See the documentation to unsafe.Pointer
// (https://golang.org/pkg/unsafe/#Pointer) for more information.
func SetOption(
	hInternet HInternet,
	option uint32,
	buffer uintptr,
	bufferLength uintptr,
) error {
	r, _, err := setOption.Call(
		uintptr(hInternet),
		uintptr(option),
		buffer,
		bufferLength,
	)

	if r != 1 {
		return err
	}
	return nil
}

var setStatusCallback = wh.NewProc("WinHttpSetStatusCallback")

// SetStatusCallback sets a callback for status events from WinHTTP.
//
// See documentation on the callback function signature at
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa383917(v=vs.85).aspx
func SetStatusCallback(
	hInternet HInternet,
	internetCallback interface{},
	notificationFlags uint32,
) error {
	r, _, err := setStatusCallback.Call(
		uintptr(hInternet),
		syscall.NewCallback(internetCallback),
		uintptr(notificationFlags),
		0,
	)

	// WinHttpSetStatusCallback returns a uintptr representing a
	// pointer to the previous callback or a value of -1 in the
	// case of error. For those playing along at home, we can't do
	// much with the uintptr to a function, nor can we represent
	// -1 as valid uintptr. So, we do the below to return true or
	// false for success instead.
	if r == ^uintptr(0) {
		return err
	}
	return nil
}

var sendRequest = wh.NewProc("WinHttpSendRequest")

// SendRequest sends the given hRequest across the network, with
// optionally setting additional headers or body data.
func SendRequest(
	hRequest HInternet,
	headers string,
	optional *byte,
	optionalLength int,
	totalLength int,
	context *uint32,
) error {
	var r uintptr
	var err error
	if headers != "" {
		r, _, err = sendRequest.Call(
			uintptr(hRequest),
			uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(headers))),
			^uintptr(0),
			uintptr(unsafe.Pointer(optional)),
			uintptr(optionalLength),
			uintptr(totalLength),
			uintptr(unsafe.Pointer(context)),
		)
	} else {
		r, _, err = sendRequest.Call(
			uintptr(hRequest),
			uintptr(WINHTTP_NO_ADDITIONAL_HEADERS),
			uintptr(0),
			uintptr(unsafe.Pointer(optional)),
			uintptr(optionalLength),
			uintptr(totalLength),
			uintptr(unsafe.Pointer(context)),
		)
	}

	if r != 1 {
		return err
	}

	return nil
}

var receiveResponse = wh.NewProc("WinHttpReceiveResponse")

// ReceiveResponse waits to receive the response to an HTTP request
// sent by SendRequest.
func ReceiveResponse(hRequest HInternet) error {
	if r, _, err := receiveResponse.Call(uintptr(hRequest), 0); r != 1 {
		return err
	}

	return nil
}

var connect = wh.NewProc("WinHttpConnect")

// Connect creates an hConnect handle pointing to a specified server.
func Connect(
	hSession HInternet,
	serverName string,
	port int,
) (HInternet, error) {
	r, _, err := connect.Call(
		uintptr(hSession),
		uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(serverName))),
		uintptr(port),
		0,
	)

	if r != 0 {
		err = nil
	}
	return HInternet(r), err
}

var openRequest = wh.NewProc("WinHttpOpenRequest")

// OpenRequest creates a new request for a given hConnect.
func OpenRequest(
	hConnect HInternet,
	verb string,
	objectName string,
	version string,
	referrer string,
	acceptTypes []string,
	flags uint32,
) (HInternet, error) {
	var verbptr *uint16
	if verb != "" {
		verbptr = windows.StringToUTF16Ptr(verb)
	}

	var objptr *uint16
	if objectName != "" {
		objptr = windows.StringToUTF16Ptr(objectName)
	}

	var verptr *uint16
	if version != "" {
		verptr = windows.StringToUTF16Ptr(version)
	}

	var refptr *uint16
	if referrer != "" {
		refptr = windows.StringToUTF16Ptr(referrer)
	}

	r, _, err := openRequest.Call(
		uintptr(hConnect),
		uintptr(unsafe.Pointer(verbptr)),
		uintptr(unsafe.Pointer(objptr)),
		uintptr(unsafe.Pointer(verptr)),
		uintptr(unsafe.Pointer(refptr)),
		0, // Not implementing ppwszAcceptTypes at this time.
		uintptr(flags),
	)

	if r != 0 {
		err = nil
	}

	return HInternet(r), err
}

var addRequestHeaders = wh.NewProc("WinHttpAddRequestHeaders")

// AddRequestHeaders adds HTTP headers to the specified Request.
func AddRequestHeaders(
	hRequest HInternet,
	header string,
	modifiers uint32,
) error {
	r, _, err := addRequestHeaders.Call(
		uintptr(hRequest),
		uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(header))),
		^uintptr(0),
		uintptr(modifiers),
	)

	if r != 1 {
		return err
	}
	return nil
}

var queryHeaders = wh.NewProc("WinHttpQueryHeaders")

// QueryHeaders retreives information from the headers of the
// specified Request.
func QueryHeaders(
	hRequest HInternet,
	infoLevel uint32,
	name string,
	buffer *uint16,
	bufferLength *uint32,
	index *uint32,
) error {
	var nameptr *uint16
	if name != "" {
		nameptr = windows.StringToUTF16Ptr(name)
	}
	r, _, err := queryHeaders.Call(
		uintptr(hRequest),
		uintptr(infoLevel),
		uintptr(unsafe.Pointer(nameptr)),
		uintptr(unsafe.Pointer(buffer)),
		uintptr(unsafe.Pointer(bufferLength)),
		uintptr(unsafe.Pointer(index)),
	)

	if r != 1 {
		return err
	}
	return nil
}

var queryDataAvailable = wh.NewProc("WinHttpQueryDataAvailable")

// QueryDataAvailable returns if we have data available to read.
func QueryDataAvailable(
	hRequest HInternet,
) (uint, error) {
	var d uint32
	r, _, err := queryDataAvailable.Call(
		uintptr(hRequest),
		uintptr(unsafe.Pointer(&d)),
	)

	if r != 1 {
		return 0, err
	}

	return uint(d), nil
}

var readData = wh.NewProc("WinHttpReadData")

// ReadData reads data from the Request into buffer.
func ReadData(
	hRequest HInternet,
	buffer []byte,
) (int, error) {
	if len(buffer) > 0 {
		var b uint32
		r, _, err := readData.Call(
			uintptr(hRequest),
			uintptr(unsafe.Pointer(&buffer[0])),
			uintptr(len(buffer)),
			uintptr(unsafe.Pointer(&b)),
		)
		if r != 1 {
			return 0, err
		}
		return int(b), nil
	} else {
		return 0, nil
	}
}

var getProxyForUrl = wh.NewProc("WinHttpGetProxyForUrl")

func GetProxyForUrl(
	hSession HInternet,
	url string,
	autoProxyOptions *WinHttpAutoproxyOptions,
) (*WinHttpProxyInfo, error) {

	p := new(WinHttpProxyInfo)

	r, _, err := getProxyForUrl.Call(
		uintptr(hSession),
		uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(url))),
		uintptr(unsafe.Pointer(autoProxyOptions)),
		uintptr(unsafe.Pointer(p)),
	)

	if r != 1 {
		return nil, err
	}
	return p, nil
}

var getIEProxyConfigForCurrentUser = wh.NewProc("WinHttpGetIEProxyConfigForCurrentUser")

func GetIEProxyConfigForCurrentUser() (*WinHttpCurrentUserIeProxyConfig, error) {
	i := new(WinHttpCurrentUserIeProxyConfig)
	r, _, err := getIEProxyConfigForCurrentUser.Call(
		uintptr(unsafe.Pointer(i)),
	)

	if r != 1 {
		return nil, err
	}
	return i, nil
}

var setTimeouts = wh.NewProc("WinHttpSetTimeouts")

func SetTimeouts(
	hInternet HInternet,
	resolveTimeout int,
	connectTimeout int,
	sendTimeout int,
	receiveTimeout int,
) error {
	r, _, err := setTimeouts.Call(
		uintptr(hInternet),
		uintptr(resolveTimeout),
		uintptr(connectTimeout),
		uintptr(sendTimeout),
		uintptr(receiveTimeout),
	)

	if r != 1 {
		return err
	}

	return nil
}
