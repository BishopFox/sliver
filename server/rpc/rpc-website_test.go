package rpc

import (
	"context"
	"errors"
	"fmt"
	insecureRand "math/rand"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
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

func cleanupWebsiteTestData(name string) {
	web, err := website.MapContent(name, false)
	if err != nil {
		return
	}

	for _, content := range web.Contents {
		_ = website.RemoveContent(name, content.GetPath())
	}

	dbWebsite, err := website.WebsiteByName(name)
	if err != nil {
		return
	}
	_ = db.RemoveWebSite(dbWebsite.ID)
}
