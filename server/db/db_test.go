package db

import "testing"

func TestBucket(t *testing.T) {
	_, err := Bucket("test")
	if err != nil {
		t.Errorf("Failed to create bucket %v", err)
	}
}
