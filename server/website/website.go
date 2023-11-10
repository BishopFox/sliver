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
	"net/url"
	"os"
	"path/filepath"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/db"
)

func getWebContentDir() (string, error) {
	webContentDir := filepath.Join(assets.GetRootAppDir(), "web")
	// websiteLog.Debugf("Web content dir: %s", webContentDir)
	if _, err := os.Stat(webContentDir); os.IsNotExist(err) {
		err = os.MkdirAll(webContentDir, 0700)
		if err != nil {
			return "", err
		}
	}
	return webContentDir, nil
}

// GetContent - Get static content for a given path
func GetContent(name string, path string) (*clientpb.WebContent, error) {
	webContentDir, err := getWebContentDir()
	if err != nil {
		return nil, err
	}

	website, err := db.WebsiteByName(name, webContentDir)
	if err != nil {
		return nil, err
	}

	// Use path without any query parameters
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	webContent, err := db.WebContentByIDAndPath(website.ID, u.Path, webContentDir, true)
	if err != nil {
		return nil, err
	}

	return webContent, err
}

// AddContent - Add website content for a path
func AddContent(name string, pbWebContent *clientpb.WebContent) error {
	// websiteName string, path string, contentType string, content []byte
	var (
		err     error
		website *clientpb.Website
	)

	webContentDir, err := getWebContentDir()
	if err != nil {
		return err
	}

	if pbWebContent.WebsiteID == "" {
		website, err = db.AddWebSite(name, webContentDir)
		if err != nil {
			return err
		}
		pbWebContent.WebsiteID = website.ID
	}

	webContent, err := db.AddContent(pbWebContent, webContentDir)
	if err != nil {
		return err
	}

	// Write content to disk
	webContentPath := filepath.Join(webContentDir, webContent.ID)
	return os.WriteFile(webContentPath, pbWebContent.Content, 0600)
}

// RemoveContent - Remove website content for a path
func RemoveContent(name string, path string) error {
	webContentDir, err := getWebContentDir()
	if err != nil {
		return err
	}

	website, err := db.WebsiteByName(name, webContentDir)
	if err != nil {
		return err
	}

	content, err := db.WebContentByIDAndPath(website.ID, path, webContentDir, true)
	if err != nil {
		return err
	}

	// Delete file
	webContentsDir, err := getWebContentDir()
	if err != nil {
		return err
	}
	err = os.Remove(filepath.Join(webContentsDir, content.ID))
	if err != nil {
		return err
	}

	// Delete row
	err = db.RemoveContent(content.ID)
	return err
}

// Names - List all websites
func Names() ([]string, error) {
	webContentsDir, err := getWebContentDir()
	if err != nil {
		return nil, err
	}

	websites, err := db.Websites(webContentsDir)
	if err != nil {
		return nil, err
	}

	names := []string{}
	for _, website := range websites {
		names = append(names, website.Name)
	}
	return names, nil
}

// MapContent - List the content of a specific site, returns map of path->json(content-type/size)
func MapContent(name string, eager bool) (*clientpb.Website, error) {
	webContentDir, err := getWebContentDir()
	if err != nil {
		return nil, err
	}

	website, err := db.WebsiteByName(name, webContentDir)
	if err != nil {
		return nil, err
	}

	if eager {
		eagerContents := map[string]*clientpb.WebContent{}
		for _, content := range website.Contents {
			eagerContent, err := db.WebContentByIDAndPath(website.ID, content.Path, webContentDir, true)
			if err != nil {
				continue
			}
			eagerContents[content.Path] = eagerContent
		}
		website.Contents = eagerContents
	}

	return website, nil
}

func AddWebsite(name string) (*clientpb.Website, error) {
	webContentDir, err := getWebContentDir()
	if err != nil {
		return nil, err
	}
	website, err := db.AddWebSite(name, webContentDir)
	if err != nil {
		return nil, err
	}
	return website, nil
}

func WebsiteByName(name string) (*clientpb.Website, error) {
	webContentDir, err := getWebContentDir()
	if err != nil {
		return nil, err
	}
	dbWebsite, err := db.WebsiteByName(name, webContentDir)
	if err != nil {
		return nil, err
	}
	return dbWebsite, nil
}
