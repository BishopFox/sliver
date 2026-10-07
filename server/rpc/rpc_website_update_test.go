package rpc

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/website"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func setupWebsiteUpdateTest(t *testing.T) (string, *clientpb.Website) {
	t.Helper()
	name := fmt.Sprintf("rpc-website-update-%d-%d", time.Now().UnixNano(), rand.Int63())
	t.Cleanup(func() { cleanupWebsiteTestData(name) })

	_, err := (&Server{}).WebsiteAddContent(context.Background(), &clientpb.WebsiteAddContent{
		Name: name,
		Contents: map[string]*clientpb.WebContent{
			"/index.html": {
				Path:         "/index.html",
				ContentType:  "text/html; charset=utf-8",
				OriginalFile: "original-index.html",
				Content:      []byte("<html>first</html>"),
			},
			"/assets/app.js": {
				Path:         "/assets/app.js",
				ContentType:  "application/javascript",
				OriginalFile: "original-app.js",
				Content:      []byte("console.log('second')"),
			},
		},
	})
	if err != nil {
		t.Fatalf("add website content: %v", err)
	}

	site, err := website.MapContent(name, true)
	if err != nil {
		t.Fatalf("read website content: %v", err)
	}
	if len(site.Contents) != 2 {
		t.Fatalf("expected two initial content paths, got %d", len(site.Contents))
	}
	return name, site
}

func assertWebsiteUpdateState(t *testing.T, name string, original *clientpb.Website, updatedMIME string) {
	t.Helper()
	site, err := website.MapContent(name, true)
	if err != nil {
		t.Fatalf("read website content after update: %v", err)
	}
	if len(site.Contents) != 2 {
		t.Fatalf("expected exactly two content paths after update, got %d", len(site.Contents))
	}
	for path, before := range original.Contents {
		after := site.Contents[path]
		if after == nil {
			t.Fatalf("content path %q disappeared", path)
		}
		wantMIME := before.ContentType
		if path == "/index.html" && updatedMIME != "" {
			wantMIME = updatedMIME
		}
		if after.ContentType != wantMIME {
			t.Errorf("%q MIME = %q, want %q", path, after.ContentType, wantMIME)
		}
		if after.OriginalFile != before.OriginalFile {
			t.Errorf("%q original filename = %q, want %q", path, after.OriginalFile, before.OriginalFile)
		}
		if after.ID != before.ID {
			t.Errorf("%q row ID = %q, want %q", path, after.ID, before.ID)
		}
		if !bytes.Equal(after.Content, before.Content) {
			t.Errorf("%q content bytes changed", path)
		}
	}

	var count int64
	if err := db.Session().Model(&models.WebContent{}).Where("website_id = ?", site.ID).Count(&count).Error; err != nil {
		t.Fatalf("count website content rows: %v", err)
	}
	if count != 2 {
		t.Errorf("website has %d content rows, want 2", count)
	}
}

func TestWebsiteUpdateContentTargetsMapPath(t *testing.T) {
	name, original := setupWebsiteUpdateTest(t)
	const newMIME = "text/plain; charset=utf-8"

	_, err := (&Server{}).WebsiteUpdateContent(context.Background(), &clientpb.WebsiteAddContent{
		Name: name,
		Contents: map[string]*clientpb.WebContent{
			"/index.html": {ContentType: newMIME},
		},
	})
	if err != nil {
		t.Fatalf("update website MIME: %v", err)
	}
	assertWebsiteUpdateState(t, name, original, newMIME)
}

func TestWebsiteUpdateContentMissingPath(t *testing.T) {
	name, original := setupWebsiteUpdateTest(t)

	_, err := (&Server{}).WebsiteUpdateContent(context.Background(), &clientpb.WebsiteAddContent{
		Name: name,
		Contents: map[string]*clientpb.WebContent{
			"/missing.html": {ContentType: "text/plain"},
		},
	})
	if got := status.Code(err); got != codes.NotFound {
		t.Fatalf("missing path returned %v, want NotFound: %v", got, err)
	}
	assertWebsiteUpdateState(t, name, original, "")
}

func TestWebsiteUpdateContentRejectsMismatchedPath(t *testing.T) {
	name, original := setupWebsiteUpdateTest(t)

	_, err := (&Server{}).WebsiteUpdateContent(context.Background(), &clientpb.WebsiteAddContent{
		Name: name,
		Contents: map[string]*clientpb.WebContent{
			"/index.html": {Path: "/assets/app.js", ContentType: "text/plain"},
		},
	})
	if got := status.Code(err); got != codes.InvalidArgument {
		t.Fatalf("mismatched path returned %v, want InvalidArgument: %v", got, err)
	}
	assertWebsiteUpdateState(t, name, original, "")
}

func TestWebsiteUpdateContentRejectsInvalidContents(t *testing.T) {
	tests := []struct {
		name     string
		contents map[string]*clientpb.WebContent
	}{
		{name: "empty map", contents: map[string]*clientpb.WebContent{}},
		{name: "empty path", contents: map[string]*clientpb.WebContent{"": {ContentType: "text/plain"}}},
		{name: "nil content", contents: map[string]*clientpb.WebContent{"/index.html": nil}},
		{name: "empty MIME", contents: map[string]*clientpb.WebContent{"/index.html": {}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			name, original := setupWebsiteUpdateTest(t)
			_, err := (&Server{}).WebsiteUpdateContent(context.Background(), &clientpb.WebsiteAddContent{
				Name:     name,
				Contents: test.contents,
			})
			if got := status.Code(err); got != codes.InvalidArgument {
				t.Fatalf("invalid content returned %v, want InvalidArgument: %v", got, err)
			}
			assertWebsiteUpdateState(t, name, original, "")
		})
	}
}
