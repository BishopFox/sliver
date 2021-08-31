package website

/*
	Sliver Implant Framework
	Copyright (C) 2020  Bishop Fox

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
	"bytes"
	"crypto/rand"
	"errors"
	insecureRand "math/rand"
	"testing"

	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/log"
)

const (
	website1 = "testWebsite1"
	website2 = "testWebsite2"

	contentType1 = "foo/bar"
	contentType2 = "foo/bar2"
)

var (
	data1          = randomData()
	data2          = randomData()
	data3          = randomData()
	websiteTestLog = log.NamedLogger("website", "test")
)

func randomData() []byte {
	buf := make([]byte, insecureRand.Intn(256))
	rand.Read(buf)
	return buf
}

func TestAddContent(t *testing.T) {
	err := AddContent(website1, "/data1", contentType1, data1)
	if err != nil {
		t.Error(err)
	}
	err = AddContent(website2, "/data2", contentType2, data2)
	if err != nil {
		t.Error(err)
	}
}

func TestGetContent(t *testing.T) {

	err := AddContent(website1, "/data1", contentType1, data1)
	if err != nil {
		t.Error(err)
	}
	err = AddContent(website2, "/data2", contentType2, data2)
	if err != nil {
		t.Error(err)
	}

	// Website 1
	contentType, content, err := GetContent(website1, "/data1")
	if err != nil {
		t.Error(err)
	}

	if contentType != contentType1 {
		t.Errorf("ContentType mismatch: %s != %s", contentType, contentType1)
	}

	if !bytes.Equal(content, data1) {
		t.Errorf("Content does not match sample")
	}

	// Website 2
	contentType, content, err = GetContent(website2, "/data2")
	if err != nil {
		t.Error(err)
	}

	if contentType != contentType2 {
		t.Errorf("ContentType mismatch: %s != %s", contentType, contentType2)
	}

	if !bytes.Equal(content, data2) {
		t.Errorf("Content does not match sample")
	}
}

func TestContentMap(t *testing.T) {
	err := AddContent(website1, "/data1", contentType1, data1)
	if err != nil {
		t.Error(err)
	}
	err = AddContent(website1, "/data2", contentType2, data2)
	if err != nil {
		t.Error(err)
	}

	contentMap, err := MapContent(website1, true)
	if err != nil {
		t.Error(err)
	}

	content := contentMap.Contents["/data1"].GetContent()
	if !bytes.Equal(content, data1) {
		t.Errorf("Content map %v does not match sample %v != %v", contentMap, content, data1)
	}

}

func contains(haystack []string, needle string) bool {
	for _, elem := range haystack {
		if elem == needle {
			return true
		}
	}
	return false
}

func TestNames(t *testing.T) {
	err := AddContent(website1, "/data1", contentType1, data1)
	if err != nil {
		t.Error(err)
	}
	err = AddContent(website1, "/data2", contentType2, data2)
	if err != nil {
		t.Error(err)
	}

	names, err := Names()
	if err != nil {
		t.Error(err)
	}
	if !contains(names, website1) {
		t.Errorf("Names returned an incomplete list of websites")
	}
}

func TestRemoveContent(t *testing.T) {
	err := AddContent(website1, "/data1", contentType1, data1)
	if err != nil {
		t.Error(err)
	}
	err = AddContent(website1, "/data2", contentType2, data2)
	if err != nil {
		t.Error(err)
	}

	_, _, err = GetContent(website1, "/data1")
	if err != nil {
		t.Error(err)
	}

	err = RemoveContent(website1, "/data1")
	if err != nil {
		t.Error(err)
	}

	_, _, err = GetContent(website1, "/foobar")
	if !errors.Is(err, db.ErrRecordNotFound) {
		t.Errorf("Expected ErrRecordNotFound, but got %v", err)
	}

	_, _, err = GetContent(website1, "/data1")
	if !errors.Is(err, db.ErrRecordNotFound) {
		t.Errorf("Expected ErrRecordNotFound, but got %v", err)
	}

}
