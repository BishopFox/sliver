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

package vt

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
)

// URLScanner represents a URL scanner.
type URLScanner struct {
	cli *Client
}

// Scan sends a URL to VirusTotal for scanning. An analysis object is returned
// as soon as the URL is submitted.
func (s *URLScanner) Scan(url string) (*Object, error) {

	b := bytes.Buffer{}
	w := multipart.NewWriter(&b)

	f, err := w.CreateFormField("url")
	if err != nil {
		return nil, err
	}

	if _, err = f.Write([]byte(url)); err != nil {
		return nil, err
	}

	w.Close()

	headers := map[string]string{"Content-Type": w.FormDataContentType()}

	httpResp, err := s.cli.sendRequest("POST", URL("urls"), &b, headers)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	apiResp, err := s.cli.parseResponse(httpResp)
	if err != nil {
		return nil, err
	}

	analysis := &Object{}
	if err := json.Unmarshal(apiResp.Data, analysis); err != nil {
		return nil, err
	}

	return analysis, nil
}
