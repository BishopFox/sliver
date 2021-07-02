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
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"os"
)

type progressReader struct {
	reader     io.Reader
	total      int64
	read       int64
	progressCh chan<- float32
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.read += int64(n)
	if pr.progressCh != nil {
		pr.progressCh <- float32(pr.read) / float32(pr.total) * 100
	}
	return n, err
}

// FileScanner represents a file scanner.
type FileScanner struct {
	cli *Client
}

// Scan sends a file to VirusTotal for scanning. The file content is read from
// the r io.Reader and sent to VirusTotal with the provided file name which can
// be left blank. The function also sends a float32 through the progress channel
// indicating the percentage of the file that has been already uploaded. The
// progress channel can be nil if the caller is not interested in receiving
// upload progress updates. An analysis object is returned as soon as the file
// is uploaded.
func (s *FileScanner) Scan(r io.Reader, filename string, progress chan<- float32) (*Object, error) {

	var uploadURL *url.URL
	var payloadSize int64

	b := bytes.Buffer{}

	// Create multipart writer for the file
	w := multipart.NewWriter(&b)
	f, err := w.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}

	// Copy data from input stream to the multiparted file
	if payloadSize, err = io.Copy(f, r); err != nil {
		return nil, err
	}

	w.Close()


	if payloadSize > maxFileSize {
		return nil, fmt.Errorf("file size can't be larger than %d bytes", maxFileSize)
	} else if payloadSize > maxPayloadSize {
		// Payload is bigger than supported by AppEngine in a POST request,
		// let's ask for an upload URL.
		var u string
		if _, err := s.cli.GetData(URL("files/upload_url"), &u); err != nil {
			return nil, err
		}
		if uploadURL, err = url.Parse(u); err != nil {
			return nil, err
		}
	} else {
		uploadURL = URL("files")
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

// ScanFile sends a file to VirusTotal for scanning. This function is similar to
// Scan but it receive an *os.File instead of a io.Reader and a file name.
func (s *FileScanner) ScanFile(f *os.File, progress chan<- float32) (*Object, error) {
	return s.Scan(f, f.Name(), progress)
}
