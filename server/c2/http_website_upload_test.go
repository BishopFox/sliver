package c2

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	insecureRand "math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/website"
	"github.com/gofrs/uuid"
)

func TestWebsiteUploadHandler(t *testing.T) {
	for _, method := range []string{http.MethodPut, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			websiteName := fmt.Sprintf("c2-upload-%d-%d", time.Now().UnixNano(), insecureRand.Int63())
			t.Cleanup(func() { cleanupUploadedTestData(websiteName) })

			site, err := website.AddWebsite(websiteName)
			if err != nil {
				t.Fatalf("failed to create website: %v", err)
			}
			if _, err := website.SetUploadAllowed(websiteName, true); err != nil {
				t.Fatalf("failed to allow uploads: %v", err)
			}

			server := &SliverHTTPC2{ServerConf: &clientpb.HTTPListenerReq{Website: websiteName}}
			body := []byte("exfiltrated bytes")
			req := httptest.NewRequest(method, "/drop/report.bin?run=1&id=7", bytes.NewReader(body))
			req.Header.Set("User-Agent", "test-agent/1.0")
			req.Header.Set("Content-Type", "application/octet-stream")
			req.RemoteAddr = "10.0.0.5:4444"

			resp := httptest.NewRecorder()
			server.router().ServeHTTP(resp, req)

			if resp.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", resp.Code)
			}

			stored, err := website.MapUploadedContent(websiteName, false)
			if err != nil {
				t.Fatalf("failed to list uploaded content: %v", err)
			}
			if len(stored.Uploaded) != 1 {
				t.Fatalf("expected 1 uploaded object, got %d", len(stored.Uploaded))
			}

			uploaded := stored.Uploaded[0]
			if uploaded.Method != method {
				t.Errorf("method: got %q want %q", uploaded.Method, method)
			}
			if uploaded.Path != "/drop/report.bin" {
				t.Errorf("path: got %q want %q", uploaded.Path, "/drop/report.bin")
			}
			if uploaded.URLParameters != "run=1&id=7" {
				t.Errorf("url parameters: got %q want %q", uploaded.URLParameters, "run=1&id=7")
			}
			if uploaded.UserAgent != "test-agent/1.0" {
				t.Errorf("user agent: got %q", uploaded.UserAgent)
			}
			if uploaded.ContentType != "application/octet-stream" {
				t.Errorf("content type: got %q", uploaded.ContentType)
			}
			if uploaded.Size != uint64(len(body)) {
				t.Errorf("size: got %d want %d", uploaded.Size, len(body))
			}
			if uploaded.RemoteAddress == "" {
				t.Error("remote address was not recorded")
			}
			if uploaded.WebsiteID != site.ID {
				t.Errorf("website id: got %q want %q", uploaded.WebsiteID, site.ID)
			}
			digest := sha256.Sum256(body)
			if want := hex.EncodeToString(digest[:]); uploaded.Sha256 != want {
				t.Errorf("sha256: got %q want %q", uploaded.Sha256, want)
			}
			// The listing is lazy, the raw body only comes back on download
			if len(uploaded.Content) != 0 {
				t.Errorf("expected no content in the listing, got %d bytes", len(uploaded.Content))
			}

			headers := map[string][]string{}
			if err := json.Unmarshal([]byte(uploaded.Headers), &headers); err != nil {
				t.Fatalf("headers are not valid json (%q): %v", uploaded.Headers, err)
			}
			if got := headers["User-Agent"]; len(got) != 1 || got[0] != "test-agent/1.0" {
				t.Errorf("headers did not keep the user agent: %v", headers)
			}

			full, err := website.GetUploadedContent(uploaded.ID)
			if err != nil {
				t.Fatalf("failed to fetch uploaded content: %v", err)
			}
			if !bytes.Equal(full.Content, body) {
				t.Errorf("content: got %q want %q", full.Content, body)
			}
		})
	}
}

// A GET for a path with no static content must still 404 rather than be stored
func TestWebsiteUploadHandlerIgnoresGet(t *testing.T) {
	websiteName := fmt.Sprintf("c2-upload-get-%d-%d", time.Now().UnixNano(), insecureRand.Int63())
	t.Cleanup(func() { cleanupUploadedTestData(websiteName) })

	if _, err := website.AddWebsite(websiteName); err != nil {
		t.Fatalf("failed to create website: %v", err)
	}
	if _, err := website.SetUploadAllowed(websiteName, true); err != nil {
		t.Fatalf("failed to allow uploads: %v", err)
	}

	server := &SliverHTTPC2{ServerConf: &clientpb.HTTPListenerReq{Website: websiteName}}
	req := httptest.NewRequest(http.MethodGet, "/drop/report.bin", nil)
	resp := httptest.NewRecorder()
	server.router().ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.Code)
	}

	stored, err := website.MapUploadedContent(websiteName, false)
	if err != nil {
		t.Fatalf("failed to list uploaded content: %v", err)
	}
	if len(stored.Uploaded) != 0 {
		t.Fatalf("a GET was stored as an upload: %d objects", len(stored.Uploaded))
	}
}

// A website that has not opted in must not store anything
func TestWebsiteUploadHandlerRejectsWhenUploadsDisabled(t *testing.T) {
	websiteName := fmt.Sprintf("c2-upload-disabled-%d-%d", time.Now().UnixNano(), insecureRand.Int63())
	t.Cleanup(func() { cleanupUploadedTestData(websiteName) })

	if _, err := website.AddWebsite(websiteName); err != nil {
		t.Fatalf("failed to create website: %v", err)
	}

	server := &SliverHTTPC2{ServerConf: &clientpb.HTTPListenerReq{Website: websiteName}}
	req := httptest.NewRequest(http.MethodPost, "/drop/report.bin", bytes.NewReader([]byte("loot")))
	resp := httptest.NewRecorder()
	server.router().ServeHTTP(resp, req)

	if resp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.Code)
	}

	stored, err := website.MapUploadedContent(websiteName, false)
	if err != nil {
		t.Fatalf("failed to list uploaded content: %v", err)
	}
	if len(stored.Uploaded) != 0 {
		t.Fatalf("content was stored for a website with uploads disabled: %d objects", len(stored.Uploaded))
	}
}

func cleanupUploadedTestData(name string) {
	site, err := website.WebsiteByName(name)
	if err != nil {
		return
	}
	siteUUID, err := uuid.FromString(site.ID)
	if err != nil {
		return
	}
	db.Session().Where(&models.WebUploadedContent{WebsiteID: siteUUID}).Delete(&models.WebUploadedContent{})
	db.RemoveWebSite(site.ID)
}
