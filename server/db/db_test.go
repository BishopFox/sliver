package db

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
	"bytes"
	"crypto/rand"
	"strings"
	"testing"
)

func randomData() []byte {
	buf := make([]byte, 128)
	rand.Read(buf)
	return buf
}

func TestGetBucket(t *testing.T) {

	data := randomData()

	bucket, err := GetBucket("test")
	if err != nil {
		t.Errorf("Failed to create bucket %v", err)
		return
	}

	err = bucket.Set("foo", data)
	if err != nil {
		t.Errorf("Failed write to bucket %v", err)
		return
	}

	value, err := bucket.Get("foo")
	if err != nil {
		t.Errorf("Failed to fetch value %s", err)
		return
	}
	if !bytes.Equal(value, data) {
		t.Errorf("Fetched value does not match sample %v != %v", value, data)
		return
	}

	err = DeleteBucket("test")
	if err != nil {
		t.Errorf("Failed to delete bucket %v", err)
	}
}

func TestGetBucketInvalidName(t *testing.T) {
	_, err := GetBucket("")
	if err == nil {
		t.Errorf("Failed to create bucket %v", err)
	}
}

func TestBucketList(t *testing.T) {
	data := randomData()
	bucket, err := GetBucket("test")
	if err != nil {
		t.Errorf("Failed to create bucket %v", err)
		return
	}
	err = bucket.Set("foo", data)
	if err != nil {
		t.Errorf("Failed write to bucket %v", err)
		return
	}

	keys, err := bucket.List("f")
	if err != nil {
		t.Errorf("Failed to list bucket %v", err)
		return
	}
	for _, k := range keys {
		if !strings.HasPrefix(k, "f") {
			t.Errorf("Key '%s' does not have 'f' prefix", k)
		}
	}
}

func TestBucketMap(t *testing.T) {
	data := randomData()
	bucket, err := GetBucket("test")
	if err != nil {
		t.Errorf("Failed to create bucket %v", err)
		return
	}
	err = bucket.Set("foo", data)
	if err != nil {
		t.Errorf("Failed write to bucket %v", err)
		return
	}
	bucketMap, err := bucket.Map("f")
	if !bytes.Equal(bucketMap["foo"], data) {
		t.Errorf("Fetched value does not match sample %v != %v", bucketMap["foo"], data)
		return
	}
}
