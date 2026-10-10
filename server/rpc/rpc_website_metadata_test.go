package rpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/website"
	"google.golang.org/protobuf/proto"
)

func roundTripWebsiteAddContent(t *testing.T, req *clientpb.WebsiteAddContent) *clientpb.WebsiteAddContent {
	t.Helper()
	data, err := proto.Marshal(req)
	if err != nil {
		t.Fatalf("marshal website content request: %v", err)
	}
	decoded := &clientpb.WebsiteAddContent{}
	if err := proto.Unmarshal(data, decoded); err != nil {
		t.Fatalf("unmarshal website content request: %v", err)
	}
	return decoded
}

func assertStoredWebsiteContent(t *testing.T, name string, want *clientpb.WebContent) {
	t.Helper()
	got, err := website.GetContent(name, want.Path)
	if err != nil {
		t.Fatalf("get stored website content: %v", err)
	}
	if !proto.Equal(got, want) {
		t.Errorf("stored website content = %v, want %v", got, want)
	}
}

func TestWebsiteAddLegacyMetadataPreservesContent(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content []byte
	}{
		{name: "nil body"},
		{name: "empty body", content: []byte{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name, original := setupWebsiteUpdateTest(t)
			const path = "/index.html"
			const newMIME = "text/plain; charset=utf-8"
			req := roundTripWebsiteAddContent(t, &clientpb.WebsiteAddContent{
				Name: name,
				Contents: map[string]*clientpb.WebContent{
					path: {
						Path:        path,
						ContentType: newMIME,
						Content:     tc.content,
						Size:        99999,
						Sha256:      "untrusted metadata hash",
					},
				},
			})
			if req.Contents[path].Content != nil {
				t.Fatal("expected legacy nil and empty bodies to decode identically")
			}
			if _, err := (&Server{}).WebsiteAddContent(context.Background(), req); err != nil {
				t.Fatalf("update legacy website metadata: %v", err)
			}
			want := proto.Clone(original.Contents[path]).(*clientpb.WebContent)
			want.ContentType = newMIME
			assertStoredWebsiteContent(t, name, want)
		})
	}
}

func TestWebsiteAddLegacyNonemptyBodyReplacesContent(t *testing.T) {
	name, original := setupWebsiteUpdateTest(t)
	const path = "/index.html"
	const newMIME = "text/plain; charset=utf-8"
	const originalFile = "replacement-index.txt"
	body := []byte("replacement from a legacy client")
	req := roundTripWebsiteAddContent(t, &clientpb.WebsiteAddContent{
		Name: name,
		Contents: map[string]*clientpb.WebContent{
			path: {
				Path:         path,
				ContentType:  newMIME,
				OriginalFile: originalFile,
				Content:      body,
			},
		},
	})
	if _, err := (&Server{}).WebsiteAddContent(context.Background(), req); err != nil {
		t.Fatalf("replace legacy website content: %v", err)
	}
	want := proto.Clone(original.Contents[path]).(*clientpb.WebContent)
	want.Content = body
	want.Size = uint64(len(body))
	hash := sha256.Sum256(body)
	want.Sha256 = hex.EncodeToString(hash[:])
	want.ContentType = newMIME
	want.OriginalFile = originalFile
	assertStoredWebsiteContent(t, name, want)
}

func TestWebsiteAddLegacyMetadataDoesNotRecreateMissingContent(t *testing.T) {
	name, original := setupWebsiteUpdateTest(t)
	const path = "/index.html"
	const newMIME = "text/plain; charset=utf-8"
	before := original.Contents[path]
	webContentDir := filepath.Join(assets.GetRootAppDir(), "web")
	backingFile := filepath.Join(webContentDir, before.ID)
	t.Cleanup(func() {
		_ = os.Remove(backingFile)
		_ = db.RemoveContent(before.ID)
	})
	if err := os.Remove(backingFile); err != nil {
		t.Fatalf("remove content backing file: %v", err)
	}
	req := roundTripWebsiteAddContent(t, &clientpb.WebsiteAddContent{
		Name: name,
		Contents: map[string]*clientpb.WebContent{
			path: {Path: path, ContentType: newMIME},
		},
	})
	if _, err := (&Server{}).WebsiteAddContent(context.Background(), req); err != nil {
		t.Fatalf("update metadata with missing backing file: %v", err)
	}
	if _, err := os.Stat(backingFile); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("metadata update recreated missing backing file: stat error = %v", err)
	}
	if _, err := website.GetContent(name, path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("get missing website content error = %v, want file not found", err)
	}
	stored, err := db.WebContentByIDAndPath(original.ID, path, webContentDir, false)
	if err != nil {
		t.Fatalf("get metadata without backing file: %v", err)
	}
	want := proto.Clone(before).(*clientpb.WebContent)
	want.Content = nil
	want.ContentType = newMIME
	if !proto.Equal(stored, want) {
		t.Errorf("stored metadata = %v, want %v", stored, want)
	}
}
