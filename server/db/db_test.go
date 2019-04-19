package db

import (
	"testing"

	"github.com/dgraph-io/badger"
)

func TestBucket(t *testing.T) {
	bucket, err := GetBucket("test")
	if err != nil {
		t.Errorf("Failed to create bucket %v", err)
	}

	err = bucket.DB.Update(func(txn *badger.Txn) error {
		txn.Set([]byte("foo"), []byte("bar"))
		return nil
	})
	if err != nil {
		t.Errorf("Failed write to bucket %v", err)
	}

	err = DeleteBucket("test")
	if err != nil {
		t.Errorf("Failed to delete bucket %v", err)
	}
}
