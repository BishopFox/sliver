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
	"errors"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/log"
)

const (
	websiteBucketName = "websites" // keys are <website name>.<path> -> clientpb.WebContent{} (json)
)

var (
	websiteLog = log.NamedLogger("website", "content")
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
	website, err := db.WebsiteByName(websiteName)
	if err != nil {
		return "", []byte{}, err
	}

	dbSession := db.Session()
	content := models.WebContent{}
	// Use path without any query parameters
	u, err := url.Parse(path)
	if err != nil {
		return "", []byte{}, err
	}
	result := dbSession.Where(&models.WebContent{
		WebsiteID: website.ID,
		Path:      u.Path,
	}).First(&content)
	if result.Error != nil {
		return "", []byte{}, result.Error
	}

	webContentDir, err := getWebContentDir()
	if err != nil {
		return "", []byte{}, err
	}
	data, err := ioutil.ReadFile(filepath.Join(webContentDir, content.ID.String()))
	return content.ContentType, data, err
}

// AddWebsite - Add website with no content
func AddWebsite(websiteName string) (*models.Website, error) {
	dbSession := db.Session()
	website, err := db.WebsiteByName(websiteName)
	if errors.Is(err, db.ErrRecordNotFound) {
		website = &models.Website{Name: websiteName}
		err = dbSession.Create(&website).Error
	}
	dbSession.Commit()
	return website, err
}

// AddContent - Add website content for a path
func AddContent(websiteName string, path string, contentType string, content []byte) error {
	dbSession := db.Session()

	website, err := db.WebsiteByName(websiteName)
	if errors.Is(err, db.ErrRecordNotFound) {
		website = &models.Website{Name: websiteName}
		err = dbSession.Create(&website).Error
	}
	if err != nil {
		return err
	}

	webContent, err := webContentByPath(website, path)
	if errors.Is(err, db.ErrRecordNotFound) {
		webContent = &models.WebContent{
			WebsiteID:   website.ID,
			Path:        path,
			ContentType: contentType,
			Size:        len(content),
		}
		err = dbSession.Create(webContent).Error
	} else {
		webContent.ContentType = contentType
		webContent.Size = len(content)
		err = dbSession.Save(webContent).Error
	}
	if err != nil {
		return err
	}
	dbSession.Commit()

	// Write content to disk
	webContentDir, err := getWebContentDir()
	if err != nil {
		return err
	}
	webContentPath := filepath.Join(webContentDir, webContent.ID.String())
	return ioutil.WriteFile(webContentPath, content, 0600)
}

func webContentByPath(website *models.Website, path string) (*models.WebContent, error) {
	dbSession := db.Session()
	webContent := models.WebContent{}
	err := dbSession.Where(&models.WebContent{
		WebsiteID: website.ID,
		Path:      path,
	}).First(&webContent).Error
	return &webContent, err
}

// RemoveContent - Remove website content for a path
func RemoveContent(websiteName string, path string) error {
	website, err := db.WebsiteByName(websiteName)
	if err != nil {
		return err
	}
	dbSession := db.Session()
	content := models.WebContent{}
	result := dbSession.Where(&models.WebContent{
		WebsiteID: website.ID,
		Path:      path,
	}).First(&content)
	if result.Error != nil {
		return result.Error
	}

	// Delete file
	webContentsDir, err := getWebContentDir()
	if err != nil {
		return err
	}
	err = os.Remove(filepath.Join(webContentsDir, content.ID.String()))
	if err != nil {
		return err
	}

	// Delete row
	result = dbSession.Delete(&models.WebContent{}, content.ID)
	dbSession.Commit()
	return result.Error
}

// Names - List all websites
func Names() ([]string, error) {
	websites := []*models.Website{}
	dbSession := db.Session()
	result := dbSession.Where(&models.Website{}).Find(&websites)
	if result.Error != nil {
		return nil, result.Error
	}
	names := []string{}
	for _, website := range websites {
		names = append(names, website.Name)
	}
	return names, nil
}

// MapContent - List the content of a specific site, returns map of path->json(content-type/size)
func MapContent(websiteName string, eagerLoadContents bool) (*clientpb.Website, error) {
	website := models.Website{}
	dbSession := db.Session()
	result := dbSession.Where(&models.Website{
		Name: websiteName,
	}).Preload("WebContents").Find(&website)
	if result.Error != nil {
		return nil, result.Error
	}

	pbWebsite := &clientpb.Website{
		Name:     website.Name,
		Contents: map[string]*clientpb.WebContent{},
	}

	webContentDir, err := getWebContentDir()
	if err != nil {
		return nil, err
	}
	websiteLog.Debugf("%d WebContent(s)", len(website.WebContents))
	for _, content := range website.WebContents {
		if eagerLoadContents {
			data, err := ioutil.ReadFile(filepath.Join(webContentDir, content.ID.String()))
			websiteLog.Debugf("Read %d bytes of content", len(data))
			if err != nil {
				websiteLog.Error(err)
				continue
			}
			pbWebsite.Contents[content.Path] = content.ToProtobuf(data)
		} else {
			pbWebsite.Contents[content.Path] = content.ToProtobuf([]byte{})
		}
	}

	return pbWebsite, nil
}
