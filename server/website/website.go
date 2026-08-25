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
	"crypto/sha256"
	"encoding/hex"
	"errors"
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

// IsUploadAllowed - Check if the site allows public upload content
func IsUploadAllowed(name string) (bool, error) {
	webContentDir, err := getWebContentDir()
	if err != nil {
		return false, err
	}

	website, err := db.WebsiteByName(name, webContentDir)
	if err != nil {
		return false, err
	}

	return website.AllowsUpload, nil
}

// SetUploadAllowed - Enable or disable public PUT/POST uploads for a website.
// Enabling creates the website when it does not exist yet, which is how an
// upload-only site (one that hosts no static content) gets created.
func SetUploadAllowed(name string, allowed bool) (*clientpb.Website, error) {
	webContentDir, err := getWebContentDir()
	if err != nil {
		return nil, err
	}

	if allowed {
		if _, err := db.AddWebSite(name, webContentDir); err != nil {
			return nil, err
		}
	}

	return db.SetWebsiteUploadAllowed(name, allowed, webContentDir)
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

	if pbWebContent.Size == 0 && len(pbWebContent.Content) > 0 {
		pbWebContent.Size = uint64(len(pbWebContent.Content))
	}
	if pbWebContent.OriginalFile == "" {
		pbWebContent.OriginalFile = filepath.Base(pbWebContent.Path)
	}

	if len(pbWebContent.Content) > 0 {
		sha := sha256.Sum256(pbWebContent.Content)
		pbWebContent.Sha256 = hex.EncodeToString(sha[:])
	}

	webContent, err := db.AddContent(pbWebContent, webContentDir)
	if err != nil {
		return err
	}

	// Write content to disk when provided (metadata-only updates skip disk writes)
	if len(pbWebContent.Content) > 0 {
		webContentPath := filepath.Join(webContentDir, webContent.ID)
		return os.WriteFile(webContentPath, pbWebContent.Content, 0600)
	}

	return nil
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

// ErrWebsiteNotEmpty - The website still holds content, so its record was kept
var ErrWebsiteNotEmpty = errors.New("website still has content")

// RemoveWebsiteIfEmpty - Delete the website record, but only once it holds no
// content of either kind. This is the only place the record is deleted, so the
// "no website without its content, no content without its website" invariant
// cannot be broken by a caller that forgot to check.
func RemoveWebsiteIfEmpty(name string) error {
	webContentDir, err := getWebContentDir()
	if err != nil {
		return err
	}

	website, err := db.WebsiteByName(name, webContentDir)
	if err != nil {
		return err
	}

	contents, uploaded, err := db.WebsiteContentCounts(website.ID)
	if err != nil {
		return err
	}
	if contents > 0 || uploaded > 0 {
		return ErrWebsiteNotEmpty
	}

	return db.RemoveWebSite(website.ID)
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

// UploadContent - Add uploaded website content for a path
func UploadContent(name string, pbWebContent *clientpb.WebUploadedContent) error {
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

	if pbWebContent.Size == 0 && len(pbWebContent.Content) > 0 {
		pbWebContent.Size = uint64(len(pbWebContent.Content))
	}

	if len(pbWebContent.Content) > 0 {
		sha := sha256.Sum256(pbWebContent.Content)
		pbWebContent.Sha256 = hex.EncodeToString(sha[:])
	}

	webContent, err := db.AddUploadedContent(pbWebContent, webContentDir)
	if err != nil {
		return err
	}

	// Write content to disk when provided (metadata-only updates skip disk writes)
	if len(pbWebContent.Content) > 0 {
		webContentPath := filepath.Join(webContentDir, webContent.ID)
		return os.WriteFile(webContentPath, pbWebContent.Content, 0600)
	}

	return nil
}

// MapUploadedContent - List the content uploaded to a specific site
func MapUploadedContent(name string, eager bool) (*clientpb.Website, error) {
	webContentDir, err := getWebContentDir()
	if err != nil {
		return nil, err
	}

	website, err := db.WebsiteByName(name, webContentDir)
	if err != nil {
		return nil, err
	}

	uploaded, err := db.UploadedWebContents(website.ID, webContentDir, eager)
	if err != nil {
		return nil, err
	}
	website.Uploaded = uploaded

	return website, nil
}

// RemoveUploadedContent - Remove one piece of uploaded content and the file backing it
func RemoveUploadedContent(id string) error {
	webContentDir, err := getWebContentDir()
	if err != nil {
		return err
	}

	content, err := db.UploadedWebContentByContentID(id, webContentDir, false)
	if err != nil {
		return err
	}

	// An upload with an empty body never got a file on disk, so a missing file is not an error
	err = os.Remove(filepath.Join(webContentDir, content.ID))
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return db.RemoveUploadedContent(content.ID)
}

// GetUploadedContent - Get a single piece of uploaded content, including its raw bytes
func GetUploadedContent(id string) (*clientpb.WebUploadedContent, error) {
	webContentDir, err := getWebContentDir()
	if err != nil {
		return nil, err
	}

	return db.UploadedWebContentByContentID(id, webContentDir, true)
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
