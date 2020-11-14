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
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
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
	// TODO
	return "", []byte{}, nil
}

// AddContent - Add website content for a path
func AddContent(websiteName string, path string, contentType string, content []byte) error {

	// dbSession := db.Session()
	// dbSession.Create(&webContent)

	return nil
}

// RemoveContent - Remove website content for a path
func RemoveContent(website string, path string) error {
	// TODO
	return nil
}

// ListWebsites - List all websites
func ListWebsites() ([]string, error) {
	// TODO
	return []string{}, nil
}

// ListContent - List the content of a specific site, returns map of path->json(content-type/size)
func ListContent(websiteName string) (*clientpb.Website, error) {

	pbWebsite := &clientpb.Website{
		Name:     websiteName,
		Contents: map[string]*clientpb.WebContent{},
	}
	// TODO
	return pbWebsite, nil
}
