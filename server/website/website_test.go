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

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/db"
)

const (
	website1 = "testWebsite1"
	website2 = "testWebsite2"

	contentType1 = "foo/bar"
	contentType2 = "foo/bar2"
)

var (
	data1 = randomData()
	data2 = randomData()
)

func randomData() []byte {
	buf := make([]byte, insecureRand.Intn(256))
	rand.Read(buf)
	return buf
}

func TestAddContent(t *testing.T) {
	webContent := clientpb.WebContent{
		Path:        "/data1",
		ContentType: contentType1,
		Size:        uint64(len(data1)),
		Content:     data1,
	}
	err := AddContent(website1, &webContent)
	if err != nil {
		t.Error(err)
	}
	webContent2 := clientpb.WebContent{
		Path:        "/data2",
		ContentType: contentType2,
		Size:        uint64(len(data2)),
		Content:     data1,
	}
	err = AddContent(website2, &webContent2)
	if err != nil {
		t.Error(err)
	}
}

func TestGetContent(t *testing.T) {

	webContent := clientpb.WebContent{
		Path:        "/data1",
		ContentType: contentType1,
		Size:        uint64(len(data1)),
		Content:     data1,
	}
	err := AddContent(website1, &webContent)
	if err != nil {
		t.Error(err)
	}
	webContent2 := clientpb.WebContent{
		Path:        "/data2",
		ContentType: contentType2,
		Size:        uint64(len(data2)),
		Content:     data2,
	}
	err = AddContent(website2, &webContent2)
	if err != nil {
		t.Error(err)
	}

	// Website 1
	content, err := GetContent(website1, "/data1")
	if err != nil {
		t.Error(err)
	}

	if content.ContentType != contentType1 {
		t.Errorf("ContentType mismatch: %s != %s", content.ContentType, contentType1)
	}

	if !bytes.Equal(content.Content, data1) {
		t.Errorf("Content does not match sample")
	}

	// Website 2
	content2, err := GetContent(website2, "/data2")
	if err != nil {
		t.Error(err)
	}

	if content2.ContentType != contentType2 {
		t.Errorf("ContentType mismatch: %s != %s", content2.ContentType, contentType2)
	}

	if !bytes.Equal(content2.Content, data2) {
		t.Errorf("Content does not match sample: %v != %v", content2.Content, data2)
	}
}

func TestContentMap(t *testing.T) {
	webContent := clientpb.WebContent{
		Path:        "/a/b/c/data1",
		ContentType: contentType1,
		Size:        uint64(len(data1)),
		Content:     data1,
	}
	err := AddContent(website1, &webContent)
	if err != nil {
		t.Error(err)
	}
	webContent2 := clientpb.WebContent{
		Path:        "/a/b/data2",
		ContentType: contentType2,
		Size:        uint64(len(data2)),
		Content:     data2,
	}
	err = AddContent(website2, &webContent2)
	if err != nil {
		t.Error(err)
	}

	contentMap, err := MapContent(website1, true)
	if err != nil {
		t.Error(err)
	}

	content := contentMap.Contents["/a/b/c/data1"].GetContent()
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
	webContent := clientpb.WebContent{
		Path:        "/a/b/c/data1",
		ContentType: contentType1,
		Size:        uint64(len(data1)),
		Content:     data1,
	}
	err := AddContent(website1, &webContent)
	if err != nil {
		t.Error(err)
	}
	webContent2 := clientpb.WebContent{
		Path:        "/a/b/data2",
		ContentType: contentType2,
		Size:        uint64(len(data2)),
		Content:     data1,
	}
	err = AddContent(website2, &webContent2)
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
	webContent := clientpb.WebContent{
		Path:        "/data1",
		ContentType: contentType1,
		Size:        uint64(len(data1)),
		Content:     data1,
	}
	err := AddContent(website1, &webContent)
	if err != nil {
		t.Error(err)
	}
	webContent2 := clientpb.WebContent{
		Path:        "/data2",
		ContentType: contentType2,
		Size:        uint64(len(data2)),
		Content:     data2,
	}
	err = AddContent(website2, &webContent2)
	if err != nil {
		t.Error(err)
	}

	_, err = GetContent(website1, "/data1")
	if err != nil {
		t.Error(err)
	}

	err = RemoveContent(website1, "/data1")
	if err != nil {
		t.Error(err)
	}

	_, err = GetContent(website1, "/foobar")
	if !errors.Is(err, db.ErrRecordNotFound) {
		t.Errorf("Expected ErrRecordNotFound, but got %v", err)
	}

	_, err = GetContent(website1, "/data1")
	if !errors.Is(err, db.ErrRecordNotFound) {
		t.Errorf("Expected ErrRecordNotFound, but got %v", err)
	}

}
