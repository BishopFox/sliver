package website

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	clientpb "github.com/bishopfox/sliver/protobuf/client"
	"github.com/bishopfox/sliver/server/db"
)

const (
	websiteBucketName = "websites" // keys are <website name>.<path> -> clientpb.WebContent{} (json)
)

func normalizePath(path string) string {
	if !strings.HasSuffix(path, "/") {
		path = "/" + path
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return "/"
	}
	return path
}

// GetContent - Get static content for a given path
func GetContent(website string, path string) (string, []byte, error) {
	bucket, err := db.GetBucket(websiteBucketName)
	if err != nil {
		return "", []byte{}, err
	}

	path = normalizePath(path)
	webContentRaw, err := bucket.Get(fmt.Sprintf("%s.%s", website, path))
	if err != nil {
		return "", []byte{}, err
	}

	webContent := &clientpb.WebContent{}
	err = json.Unmarshal(webContentRaw, webContent)
	if err != nil {
		return "", []byte{}, err
	}
	return webContent.ContentType, webContent.Content, nil
}

// AddContent - Add website content for a path
func AddContent(website string, path string, contentType string, content []byte) error {
	bucket, err := db.GetBucket(websiteBucketName)
	if err != nil {
		return err
	}
	webContent, err := json.Marshal(&clientpb.WebContent{
		ContentType: contentType,
		Content:     content,
		Size:        uint64(len(content)),
	})
	if err != nil {
		return err
	}
	path = normalizePath(path)
	bucket.Set(fmt.Sprintf("%s.%s", website, path), webContent)
	return nil
}

// RemoveContent - Remove website content for a path
func RemoveContent(website string, path string) error {
	bucket, err := db.GetBucket(websiteBucketName)
	if err != nil {
		return err
	}
	path = normalizePath(path)
	return bucket.Delete(fmt.Sprintf("%s.%s", website, path))
}

// ListWebsites - List all websites
func ListWebsites() ([]string, error) {
	bucket, err := db.GetBucket(websiteBucketName)
	if err != nil {
		return nil, err
	}

	keys, err := bucket.List("")
	if err != nil {
		return nil, err
	}

	// Because Go doesn't have a generic Keys()
	websites := make(map[string]bool)
	for _, k := range keys {
		name := strings.Split(k, ".")[0] // Split on '.' and take the zero'th
		websites[name] = true
	}
	websiteNames := make([]string, 0, len(websites))
	for k := range websites {
		websiteNames = append(websiteNames, k)
	}
	return websiteNames, nil
}

// ListContent - List the content of a specific site, returns map of path->json(content-type/size)
func ListContent(website string) (*map[string][]byte, error) {
	bucket, err := db.GetBucket(websiteBucketName)
	if err != nil {
		return nil, err
	}
	websiteContent, err := bucket.Map(fmt.Sprintf("%s.", website))
	if err != nil {
		return nil, err
	}
	siteContent := &map[string][]byte{}
	for key, contentRaw := range websiteContent {
		webContent := &clientpb.WebContent{}
		err := json.Unmarshal(contentRaw, webContent)
		if err != nil {
			continue
		}
		webContent.Content = []byte{} // Remove actual file contents
		path := key[len(fmt.Sprintf("%s.", website)):]
		content, _ := json.Marshal(webContent)
		(*siteContent)[path] = content
	}
	return siteContent, nil
}
