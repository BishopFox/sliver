package db

import (
	"bytes"
	"crypto/rand"
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
