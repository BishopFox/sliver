// Copyright © 2019 The vt-go authors. All Rights Reserved.
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
	"io"
	"mime/multipart"
	"net/url"
)

// MonitorUploader represents a  VT Monitor file upload.
type MonitorUploader struct {
	cli *Client
}

// upload sends a file to VirusTotal Monitor. The file content is read from
// the r io.Reader and sent to Monitor with the provided params.
// The function also sends a float32 through the progress channel indicating the
// percentage of the file that has been already uploaded. The progress channel
// can be nil if the caller is not interested in receiving upload progress
// updates. The received object is returned as soon as the file is uploaded.
func (s *MonitorUploader) upload(r io.Reader, params map[string]string, progress chan<- float32) (*Object, error) {
	var uploadURL *url.URL
	var payloadSize int64

	b := bytes.Buffer{}

	// Create multipart writer for the file
	w := multipart.NewWriter(&b)

	// Assign a filename, in monitor is not used but AppEng requieres the form to have it
	f, err := w.CreateFormFile("file", "monitor_upload")
	if err != nil {
		return nil, err
	}

	// Copy data from input stream to the multiparted file
	if payloadSize, err = io.Copy(f, r); err != nil {
		return nil, err
	}

	for key, value := range params {
		w.WriteField(key, value)
	}

	w.Close()

	if payloadSize > maxPayloadSize {
		// Payload is bigger than supported by AppEngine in a POST request,
		// let's ask for an upload URL.
		var u string
		if _, err := s.cli.GetData(URL("monitor/items/upload_url"), &u); err != nil {
			return nil, err
		}
		if uploadURL, err = url.Parse(u); err != nil {
			return nil, err
		}
	} else {
		uploadURL = URL("monitor/items")
	}

	pr := &progressReader{
		reader:     &b,
		total:      int64(b.Len()),
		progressCh: progress}

	headers := map[string]string{"Content-Type": w.FormDataContentType()}

	httpResp, err := s.cli.sendRequest("POST", uploadURL, pr, headers)
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

// Upload uploads a file to your VT Monitor account using a monitorPath
// destination path. The file content is read from the r io.Reader and sent to
// Monitor. If the remote path already exists the contents will be replaced.
// The function also sends a float32 through the progress channel indicating the
// percentage of the file that has been already uploaded. The progress channel
// can be nil if the caller is not interested in receiving upload progress
// updates. The received object is returned as soon as the file is uploaded.
func (s *MonitorUploader) Upload(r io.Reader, monitorPath string, progress chan<- float32) (*Object, error) {
	params := map[string]string{"path": monitorPath}
	return s.upload(r, params, progress)
}

// Replace modifies the contents of Monitor file identified by its
// monitorItemID. If the monitorItemID was previously deleted or does not exist
// a new file with the uploaded contents will be created. The file content is
// read from the r io.Reader and sent to Monitor. The function also sends a
// float32 through the progress channel indicating the percentage of the file
// that has been already uploaded. The progress channel can be nil if the caller
// is not interested in receiving upload progress updates.
// The received object is returned as soon as the file is uploaded.
func (s *MonitorUploader) Replace(r io.Reader, monitorItemID string, progress chan<- float32) (*Object, error) {
	params := map[string]string{"item": monitorItemID}
	return s.upload(r, params, progress)
}
