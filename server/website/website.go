package website

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
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	clientpb "github.com/bishopfox/sliver/protobuf/client"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/log"
)

const (
	websiteBucketName = "websites" // keys are <website name>.<path> -> clientpb.WebContent{} (json)
)

var (
	websiteLog = log.NamedLogger("website", "content")
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
func GetContent(websiteName string, path string) (string, []byte, error) {
	bucket, err := db.GetBucket(websiteBucketName)
	if err != nil {
		return "", []byte{}, err
	}

	path = normalizePath(path)
	webContentRaw, err := bucket.Get(fmt.Sprintf("%s.%s", websiteName, path))
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
func AddContent(websiteName string, path string, contentType string, content []byte) error {
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
	key := fmt.Sprintf("%s.%s", websiteName, path)
	websiteLog.Infof("[add] %s", key)
	bucket.Set(key, webContent)
	return nil
}

// RemoveContent - Remove website content for a path
func RemoveContent(website string, path string) error {
	bucket, err := db.GetBucket(websiteBucketName)
	if err != nil {
		return err
	}
	path = normalizePath(path)
	key := fmt.Sprintf("%s.%s", website, path)
	websiteLog.Infof("[delete] %s", key)
	return bucket.Delete(key)
}

// ListWebsites - List all websites
func ListWebsites() ([]string, error) {
	bucket, err := db.GetBucket(websiteBucketName)
	if err != nil {
		return nil, err
	}

	keys, err := bucket.Map("")
	if err != nil {
		return nil, err
	}

	// Because Go doesn't have a generic Keys()
	websites := make(map[string]bool)
	for key := range keys {
		name := strings.Split(key, ".")[0] // Split on '.' and take the zero'th
		websites[name] = true
	}
	websiteNames := make([]string, 0, len(websites))
	for k := range websites {
		websiteNames = append(websiteNames, k)
	}
	return websiteNames, nil
}

// ListContent - List the content of a specific site, returns map of path->json(content-type/size)
func ListContent(websiteName string) (*clientpb.Website, error) {
	bucket, err := db.GetBucket(websiteBucketName)
	if err != nil {
		return nil, err
	}
	websiteContent, err := bucket.Map(fmt.Sprintf("%s.", websiteName))
	if err != nil {
		return nil, err
	}
	pbWebsite := &clientpb.Website{
		Name:    websiteName,
		Content: map[string]*clientpb.WebContent{},
	}
	for key, contentRaw := range websiteContent {
		webContent := &clientpb.WebContent{}
		err := json.Unmarshal(contentRaw, webContent)
		if err != nil {
			continue
		}
		webContent.Content = []byte{} // Remove actual file contents
		webContent.Path = key[len(fmt.Sprintf("%s.", websiteName)):]
		pbWebsite.Content[webContent.Path] = webContent
	}
	return pbWebsite, nil
}
