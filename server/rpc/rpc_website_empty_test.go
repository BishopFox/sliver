package rpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/website"
)

func setupEmptyWebsiteTest(t *testing.T) string {
	t.Helper()
	name := fmt.Sprintf("rpc-website-empty-%d-%d", time.Now().UnixNano(), rand.Int63())
	t.Cleanup(func() { cleanupWebsiteTestData(name) })
	return name
}

func assertEmptyWebsiteContent(t *testing.T, name, path, contentType string) {
	t.Helper()
	wantHash := sha256.Sum256(nil)
	wantSHA256 := hex.EncodeToString(wantHash[:])

	for _, eager := range []bool{false, true} {
		site, err := website.MapContent(name, eager)
		if err != nil {
			t.Fatalf("map website content (eager=%t): %v", eager, err)
		}
		if len(site.Contents) != 1 {
			t.Fatalf("map website content (eager=%t) returned %d paths, want 1", eager, len(site.Contents))
		}
		content := site.Contents[path]
		if content == nil {
			t.Fatalf("map website content (eager=%t) omitted %q", eager, path)
		}
		if content.Size != 0 || len(content.Content) != 0 || content.Sha256 != wantSHA256 || content.ContentType != contentType {
			t.Errorf("mapped content (eager=%t) = size %d, bytes %d, SHA256 %q, MIME %q; want 0, 0, %q, %q", eager, content.Size, len(content.Content), content.Sha256, content.ContentType, wantSHA256, contentType)
		}
	}

	content, err := website.GetContent(name, path)
	if err != nil {
		t.Fatalf("get empty website content: %v", err)
	}
	if content.Size != 0 || len(content.Content) != 0 || content.Sha256 != wantSHA256 || content.ContentType != contentType {
		t.Errorf("fetched content = size %d, bytes %d, SHA256 %q, MIME %q; want 0, 0, %q, %q", content.Size, len(content.Content), content.Sha256, content.ContentType, wantSHA256, contentType)
	}

	var row models.WebContent
	if err := db.Session().Where("id = ?", content.ID).First(&row).Error; err != nil {
		t.Fatalf("find empty content row: %v", err)
	}
	if row.Size != 0 || row.Sha256 != wantSHA256 || row.ContentType != contentType || row.Path != path {
		t.Errorf("stored row = size %d, SHA256 %q, MIME %q, path %q; want 0, %q, %q, %q", row.Size, row.Sha256, row.ContentType, row.Path, wantSHA256, contentType, path)
	}

	backingFile := filepath.Join(assets.GetRootAppDir(), "web", content.ID)
	fileInfo, err := os.Stat(backingFile)
	if err != nil {
		t.Fatalf("stat empty content backing file: %v", err)
	}
	if fileInfo.Size() != 0 {
		t.Errorf("empty content backing file has %d bytes, want 0", fileInfo.Size())
	}
}

func TestWebsiteAddZeroByteContent(t *testing.T) {
	name := setupEmptyWebsiteTest(t)
	const path = "/empty.txt"
	const contentType = "text/plain; charset=utf-8"

	site, err := (&Server{}).WebsiteAddContent(context.Background(), &clientpb.WebsiteAddContent{
		Name: name,
		Contents: map[string]*clientpb.WebContent{
			path: {Path: path, ContentType: contentType, Content: []byte{}},
		},
	})
	if err != nil {
		t.Fatalf("add zero-byte website content: %v", err)
	}
	if site.Contents[path] == nil {
		t.Fatalf("add response omitted zero-byte content at %q", path)
	}
	assertEmptyWebsiteContent(t, name, path, contentType)
}

func TestWebsiteReplaceWithZeroByteContent(t *testing.T) {
	name := setupEmptyWebsiteTest(t)
	const path = "/existing.txt"
	const originalMIME = "text/plain"
	const updatedMIME = "application/octet-stream"

	rpcServer := &Server{}
	_, err := rpcServer.WebsiteAddContent(context.Background(), &clientpb.WebsiteAddContent{
		Name: name,
		Contents: map[string]*clientpb.WebContent{
			path: {Path: path, ContentType: originalMIME, Content: []byte("old content")},
		},
	})
	if err != nil {
		t.Fatalf("add initial website content: %v", err)
	}

	_, err = rpcServer.WebsiteAddContent(context.Background(), &clientpb.WebsiteAddContent{
		Name: name,
		Contents: map[string]*clientpb.WebContent{
			path: {Path: path, ContentType: originalMIME, Content: []byte{}},
		},
	})
	if err != nil {
		t.Fatalf("replace website content with zero bytes: %v", err)
	}
	assertEmptyWebsiteContent(t, name, path, originalMIME)

	_, err = rpcServer.WebsiteUpdateContent(context.Background(), &clientpb.WebsiteAddContent{
		Name: name,
		Contents: map[string]*clientpb.WebContent{
			path: {ContentType: updatedMIME},
		},
	})
	if err != nil {
		t.Fatalf("update MIME of zero-byte content: %v", err)
	}
	assertEmptyWebsiteContent(t, name, path, updatedMIME)
}
