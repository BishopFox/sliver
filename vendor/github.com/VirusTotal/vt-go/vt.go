// Copyright © 2017 The vt-go authors. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//
// Package vt is a client library for the VirusTotal API v3. It makes the use
// of the VirusTotal's REST API easier for Go developers.
//
package vt

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

const (
	version = "0.3"
)

const (
	// Maximum size of payloads posted to VirusTotal's API endpoints.
	maxPayloadSize = 30 * 1024 * 1024 // 30 MB
	// Maximum file size that can scanned by VirusTotal.
	maxFileSize = 650 * 1024 * 1024 // 650 MB
)

var baseURL = url.URL{
	Scheme: "https",
	Host:   "www.virustotal.com",
	Path:   "api/v3/"}

// Request is the top level structure of an API request.
type Request struct {
	Data interface{} `json:"data"`
}

// Response is the top level structure of an API response.
type Response struct {
	Data  json.RawMessage        `json:"data"`
	Meta  map[string]interface{} `json:"meta"`
	Links Links                  `json:"links"`
	Error Error                  `json:"error"`
}

// Error contains information about an API error.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface.
func (e Error) Error() string {
	return e.Message
}

// URL returns a full VirusTotal API URL from a relative path (i.e: a path
// without the domain name and the "/api/v3/" prefix). The path can contain
// format 'verbs' as defined in the "fmt". This function is useful for creating
// URLs to be passed to any function expecting a *url.URL in this library.
func URL(pathFmt string, a ...interface{}) *url.URL {
	path := fmt.Sprintf(pathFmt, a...)
	url, err := url.Parse(path)
	if err != nil {
		msg := fmt.Sprintf(
			"error formatting URL \"%s\": %s",
			pathFmt, err)
		panic(msg)
	}
	return baseURL.ResolveReference(url)
}

// SetHost allows to change the host used while sending requests to the
// VirusTotal API. The default host is "www.virustotal.com" you rarely need to
// change it.
func SetHost(host string) {
	if strings.HasPrefix(host, "https://") {
		baseURL.Scheme = "https"
		baseURL.Host = strings.TrimPrefix(host, "https://")
	} else if strings.HasPrefix(host, "http://") {
		baseURL.Scheme = "http"
		baseURL.Host = strings.TrimPrefix(host, "http://")
	} else {
		baseURL.Host = host
	}
}
