package rpc

import (
	"context"
	"errors"
	"fmt"
	insecureRand "math/rand"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/website"
)

func TestWebsiteRemoveDeletesWebsiteWithContent(t *testing.T) {
	websiteName := fmt.Sprintf("rpc-website-remove-%d-%d", time.Now().UnixNano(), insecureRand.Int63())
	rpcServer := &Server{}
	t.Cleanup(func() {
		cleanupWebsiteTestData(websiteName)
	})

	_, err := rpcServer.WebsiteAddContent(context.Background(), &clientpb.WebsiteAddContent{
		Name: websiteName,
		Contents: map[string]*clientpb.WebContent{
			"/index.html": {
				Path:        "/index.html",
				ContentType: "text/html; charset=utf-8",
				Content:     []byte("<html><body>ok</body></html>"),
			},
			"/assets/app.js": {
				Path:        "/assets/app.js",
				ContentType: "application/javascript",
				Content:     []byte("console.log('ok')"),
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to add website content: %v", err)
	}

	_, err = rpcServer.WebsiteRemove(context.Background(), &clientpb.Website{Name: websiteName})
	if err != nil {
		t.Fatalf("failed to remove website: %v", err)
	}

	_, err = website.WebsiteByName(websiteName)
	if !errors.Is(err, db.ErrRecordNotFound) {
		t.Fatalf("expected website to be removed, got error: %v", err)
	}

	_, err = website.GetContent(websiteName, "/index.html")
	if !errors.Is(err, db.ErrRecordNotFound) {
		t.Fatalf("expected website content to be removed, got error: %v", err)
	}
}

// Uploaded content is loot, removing a website must never destroy it as a side
// effect. The static content still goes, only the record is kept behind.
func TestWebsiteRemoveKeepsWebsiteWithUploads(t *testing.T) {
	websiteName := newWebsiteTestName(t, "rpc-website-remove-uploads")
	rpcServer := &Server{}

	addStaticTestContent(t, rpcServer, websiteName, "/index.html")
	uploadID := addUploadedTestContent(t, websiteName, "/drop/report.bin")

	_, err := rpcServer.WebsiteRemove(context.Background(), &clientpb.Website{Name: websiteName})
	if err != nil {
		t.Fatalf("failed to remove website content: %v", err)
	}

	if _, err := website.GetContent(websiteName, "/index.html"); !errors.Is(err, db.ErrRecordNotFound) {
		t.Fatalf("expected static content to be removed, got error: %v", err)
	}
	if _, err := website.WebsiteByName(websiteName); err != nil {
		t.Fatalf("website was removed while it still had uploads: %v", err)
	}
	if _, err := website.GetUploadedContent(uploadID); err != nil {
		t.Fatalf("uploaded content was removed: %v", err)
	}

	// And once the uploads are gone the record can be removed
	if _, err := rpcServer.WebsiteRemoveUploaded(context.Background(), &clientpb.WebsiteRemoveUploaded{
		Name: websiteName,
		IDs:  []string{uploadID},
	}); err != nil {
		t.Fatalf("failed to remove uploaded content: %v", err)
	}
	if _, err := rpcServer.WebsiteRemove(context.Background(), &clientpb.Website{Name: websiteName}); err != nil {
		t.Fatalf("failed to remove website: %v", err)
	}
	if _, err := website.WebsiteByName(websiteName); !errors.Is(err, db.ErrRecordNotFound) {
		t.Fatalf("expected website to be removed, got error: %v", err)
	}
}

// Once the uploads are gone the website removes normally
func TestWebsiteRemoveSucceedsAfterUploadsCleared(t *testing.T) {
	websiteName := newWebsiteTestName(t, "rpc-website-remove-cleared")
	rpcServer := &Server{}

	addStaticTestContent(t, rpcServer, websiteName, "/index.html")
	uploadID := addUploadedTestContent(t, websiteName, "/drop/report.bin")

	_, err := rpcServer.WebsiteRemoveUploaded(context.Background(), &clientpb.WebsiteRemoveUploaded{
		Name: websiteName,
		IDs:  []string{uploadID},
	})
	if err != nil {
		t.Fatalf("failed to remove uploaded content: %v", err)
	}

	_, err = rpcServer.WebsiteRemove(context.Background(), &clientpb.Website{Name: websiteName})
	if err != nil {
		t.Fatalf("failed to remove website: %v", err)
	}
	if _, err := website.WebsiteByName(websiteName); !errors.Is(err, db.ErrRecordNotFound) {
		t.Fatalf("expected website to be removed, got error: %v", err)
	}
}

// Static and uploaded content can share a path, rm-content must only take the static one
func TestWebsiteRemoveContentKeepsUploadAtSamePath(t *testing.T) {
	websiteName := newWebsiteTestName(t, "rpc-website-rm-content-uploads")
	rpcServer := &Server{}

	addStaticTestContent(t, rpcServer, websiteName, "/shared")
	uploadID := addUploadedTestContent(t, websiteName, "/shared")

	_, err := rpcServer.WebsiteRemoveContent(context.Background(), &clientpb.WebsiteRemoveContent{
		Name:  websiteName,
		Paths: []string{"/shared"},
	})
	if err != nil {
		t.Fatalf("failed to remove content: %v", err)
	}

	if _, err := website.GetContent(websiteName, "/shared"); !errors.Is(err, db.ErrRecordNotFound) {
		t.Fatalf("expected static content to be removed, got error: %v", err)
	}
	if _, err := website.GetUploadedContent(uploadID); err != nil {
		t.Fatalf("uploaded content at the same path was removed: %v", err)
	}
	// The website still holds an upload, so its record has to stay
	if _, err := website.WebsiteByName(websiteName); err != nil {
		t.Fatalf("website was removed while it still had uploads: %v", err)
	}
}

// The mirror case, removing uploads must not take the static content or the website
func TestWebsiteRemoveUploadedKeepsStaticContent(t *testing.T) {
	websiteName := newWebsiteTestName(t, "rpc-website-rm-uploaded")
	rpcServer := &Server{}

	addStaticTestContent(t, rpcServer, websiteName, "/index.html")
	uploadID := addUploadedTestContent(t, websiteName, "/drop/report.bin")

	site, err := rpcServer.WebsiteRemoveUploaded(context.Background(), &clientpb.WebsiteRemoveUploaded{
		Name: websiteName,
		IDs:  []string{uploadID},
	})
	if err != nil {
		t.Fatalf("failed to remove uploaded content: %v", err)
	}
	if len(site.Uploaded) != 0 {
		t.Fatalf("expected no uploads left, got %d", len(site.Uploaded))
	}
	if _, err := website.GetContent(websiteName, "/index.html"); err != nil {
		t.Fatalf("static content was removed: %v", err)
	}
	if _, err := website.WebsiteByName(websiteName); err != nil {
		t.Fatalf("website was removed while it still had static content: %v", err)
	}
}

// The guard itself, whichever kind of content remains
func TestRemoveWebsiteIfEmpty(t *testing.T) {
	rpcServer := &Server{}

	t.Run("static content remains", func(t *testing.T) {
		websiteName := newWebsiteTestName(t, "rpc-website-empty-static")
		addStaticTestContent(t, rpcServer, websiteName, "/index.html")
		if err := website.RemoveWebsiteIfEmpty(websiteName); !errors.Is(err, website.ErrWebsiteNotEmpty) {
			t.Fatalf("expected ErrWebsiteNotEmpty, got %v", err)
		}
		if _, err := website.WebsiteByName(websiteName); err != nil {
			t.Fatalf("website was removed: %v", err)
		}
	})

	t.Run("uploaded content remains", func(t *testing.T) {
		websiteName := newWebsiteTestName(t, "rpc-website-empty-uploaded")
		if _, err := website.AddWebsite(websiteName); err != nil {
			t.Fatalf("failed to create website: %v", err)
		}
		addUploadedTestContent(t, websiteName, "/drop/report.bin")
		if err := website.RemoveWebsiteIfEmpty(websiteName); !errors.Is(err, website.ErrWebsiteNotEmpty) {
			t.Fatalf("expected ErrWebsiteNotEmpty, got %v", err)
		}
		if _, err := website.WebsiteByName(websiteName); err != nil {
			t.Fatalf("website was removed: %v", err)
		}
	})

	t.Run("no content at all", func(t *testing.T) {
		websiteName := newWebsiteTestName(t, "rpc-website-empty-none")
		if _, err := website.AddWebsite(websiteName); err != nil {
			t.Fatalf("failed to create website: %v", err)
		}
		if err := website.RemoveWebsiteIfEmpty(websiteName); err != nil {
			t.Fatalf("failed to remove empty website: %v", err)
		}
		if _, err := website.WebsiteByName(websiteName); !errors.Is(err, db.ErrRecordNotFound) {
			t.Fatalf("expected website to be removed, got error: %v", err)
		}
	})
}

// The websites listing reports a count per kind, so it has to carry both
func TestWebsitesListCarriesUploadedContent(t *testing.T) {
	websiteName := newWebsiteTestName(t, "rpc-websites-list")
	rpcServer := &Server{}

	addStaticTestContent(t, rpcServer, websiteName, "/index.html")
	addUploadedTestContent(t, websiteName, "/drop/one.bin")
	addUploadedTestContent(t, websiteName, "/drop/two.bin")

	listing, err := rpcServer.Websites(context.Background(), &commonpb.Empty{})
	if err != nil {
		t.Fatalf("failed to list websites: %v", err)
	}

	var site *clientpb.Website
	for _, candidate := range listing.Websites {
		if candidate.Name == websiteName {
			site = candidate
			break
		}
	}
	if site == nil {
		t.Fatalf("website %q missing from the listing", websiteName)
	}
	if len(site.Contents) != 1 {
		t.Errorf("static contents: got %d want 1", len(site.Contents))
	}
	if len(site.Uploaded) != 2 {
		t.Errorf("uploaded objects: got %d want 2", len(site.Uploaded))
	}
	// The listing is lazy, raw bodies only come back on download
	for _, content := range site.Uploaded {
		if len(content.Content) != 0 {
			t.Errorf("listing carried %d bytes of upload body", len(content.Content))
		}
	}
}

func newWebsiteTestName(t *testing.T, prefix string) string {
	t.Helper()
	name := fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), insecureRand.Int63())
	t.Cleanup(func() { cleanupWebsiteTestData(name) })
	return name
}

func addStaticTestContent(t *testing.T, rpcServer *Server, websiteName string, path string) {
	t.Helper()
	_, err := rpcServer.WebsiteAddContent(context.Background(), &clientpb.WebsiteAddContent{
		Name: websiteName,
		Contents: map[string]*clientpb.WebContent{
			path: {Path: path, ContentType: "text/plain", Content: []byte("static " + path)},
		},
	})
	if err != nil {
		t.Fatalf("failed to add website content: %v", err)
	}
}

func addUploadedTestContent(t *testing.T, websiteName string, path string) string {
	t.Helper()
	err := website.UploadContent(websiteName, &clientpb.WebUploadedContent{
		Method:      "POST",
		Path:        path,
		ContentType: "application/octet-stream",
		Content:     []byte("loot " + path),
		ReceivedAt:  time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("failed to store uploaded content: %v", err)
	}
	site, err := website.MapUploadedContent(websiteName, false)
	if err != nil {
		t.Fatalf("failed to list uploaded content: %v", err)
	}
	for _, content := range site.Uploaded {
		if content.Path == path {
			return content.ID
		}
	}
	t.Fatalf("uploaded content %q was not stored", path)
	return ""
}

func cleanupWebsiteTestData(name string) {
	web, err := website.MapUploadedContent(name, false)
	if err != nil {
		return
	}

	for _, content := range web.Contents {
		_ = website.RemoveContent(name, content.GetPath())
	}
	for _, content := range web.Uploaded {
		_ = website.RemoveUploadedContent(content.ID)
	}

	dbWebsite, err := website.WebsiteByName(name)
	if err != nil {
		return
	}
	_ = db.RemoveWebSite(dbWebsite.ID)
}
