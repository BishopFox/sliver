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
	"errors"
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/log"

	"github.com/dgraph-io/badger"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	dbDirName      = "db"
	rootDBDirName  = "root"
	bucketsDirName = "buckets"
)

var (
	rootDB       = getRootDB()
	dbLog        = log.NamedLogger("db", "")
	dbCache      = &map[string]*Bucket{}
	dbCacheMutex = &sync.Mutex{}
)

// Bucket - Badger database and namespaced logger
type Bucket struct {
	db  *badger.DB
	Log *logrus.Entry
}

// These are just pass-thru functions to prevent external callers from
// trying to manually manage db transcations

// Set - Set a key/value (simplified API)
func (b *Bucket) Set(key string, value []byte) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), value)
	})
}

// Get - Get a value for a given key (simplified API)
func (b *Bucket) Get(key string) ([]byte, error) {
	var value []byte
	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		value, err = item.ValueCopy(nil)
		return err
	})
	return value, err
}

// Delete - Delete a key/value
func (b *Bucket) Delete(key string) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

// List - Returns a list of keys filtered by prefix. Note that modify this list will not affect the database.
func (b *Bucket) List(prefix string) ([]string, error) {
	keyPrefix := []byte(prefix)
	keys := []string{}
	err := b.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(keyPrefix); it.ValidForPrefix(keyPrefix); it.Next() {
			item := it.Item()
			keys = append(keys, string(item.Key()))
		}
		return nil
	})
	return keys, err
}

// Map - Returns a map of key/values filtered by prefix. Note that modify this map will not affect the database.
func (b *Bucket) Map(prefix string) (map[string][]byte, error) {
	keyPrefix := []byte(prefix)
	bucketMap := map[string][]byte{}
	err := b.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(keyPrefix); it.ValidForPrefix(keyPrefix); it.Next() {
			item := it.Item()
			value, err := item.ValueCopy(nil)
			if err != nil {
				continue
			}
			bucketMap[string(item.Key())] = value
		}
		return nil
	})
	return bucketMap, err
}

// Ptr to the root databasea that maps bucket Names <-> UUIDs
func getRootDB() *badger.DB {
	rootDir := assets.GetRootAppDir()
	dbDir := path.Join(rootDir, dbDirName, rootDBDirName)
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		os.MkdirAll(dbDir, os.ModePerm)
	}
	opts := badger.DefaultOptions
	opts.Dir = dbDir
	opts.ValueDir = dbDir
	opts.Logger = log.NamedLogger("db", "root")
	db, err := badger.Open(opts)
	if err != nil {
		dbLog.Fatal(err)
	}
	return db
}

// GetBucket returns a namespaced database, names are mapped to directoires
// thru the rootDB which stores Name<->UUID pairs, this allows us to support
// bucket names with arbitrary string values
func GetBucket(name string) (*Bucket, error) {
	if len(name) == 0 {
		return nil, errors.New("Invalid bucket name")
	}
	rootDir := assets.GetRootAppDir()

	txn := rootDB.NewTransaction(true)
	defer txn.Discard()

	var bucketUUID string
	item, err := txn.Get([]byte(name))
	if err == badger.ErrKeyNotFound {
		dbLog.Infof("No bucket for name %#v", name)
		id := uuid.New()
		txn.Set([]byte(name), []byte(id.String()))
		if err := txn.Commit(); err != nil {
			dbLog.Debugf("Failed to create bucket %#v, %v", name, err)
			return nil, err
		}
		dbLog.Infof("Created new bucket with name %#v (%s)", name, id.String())
		bucketUUID = id.String()
	} else if err != nil {
		dbLog.Debugf("rootDB error %v", err)
		return nil, err
	} else {
		val, _ := item.ValueCopy(nil)
		bucketUUID = string(val)
		// dbLog.Debugf("Using bucket %#v (%s)", name, bucketUUID)
	}

	// We can only call open() once on each directory so we save references
	// to buckets when we open them.
	dbCacheMutex.Lock()
	defer dbCacheMutex.Unlock()
	if bucket, ok := (*dbCache)[bucketUUID]; ok {
		return bucket, nil
	}

	// No open handle to database, open/create the bucket
	bucketDir := path.Join(rootDir, dbDirName, bucketsDirName, bucketUUID)
	if _, err := os.Stat(bucketDir); os.IsNotExist(err) {
		os.MkdirAll(bucketDir, os.ModePerm)
	}
	dbLog.Debugf("Loading db from %s", bucketDir)
	opts := badger.DefaultOptions
	opts.Dir = bucketDir
	opts.ValueDir = bucketDir
	logger := log.NamedLogger("db", name)
	opts.Logger = logger
	db, err := badger.Open(opts)
	if err != nil {
		dbLog.Errorf("Failed to open db %s", err)
		return nil, err
	}
	bucket := &Bucket{
		db:  db,
		Log: logger,
	}
	(*dbCache)[bucketUUID] = bucket
	return bucket, nil
}

// DeleteBucket - Deletes a bucket from the filesystem and rootDB
func DeleteBucket(name string) error {
	if len(name) == 0 {
		return errors.New("Invalid bucket name")
	}
	rootDir := assets.GetRootAppDir()

	txn := rootDB.NewTransaction(true)
	defer txn.Discard()

	var bucketUUID string
	item, err := txn.Get([]byte(name))
	if err == badger.ErrKeyNotFound {
		return nil
	} else if err != nil {
		dbLog.Debugf("rootDB error %v", err)
		return err
	} else {
		val, _ := item.ValueCopy(nil)
		bucketUUID = string(val)
		if len(bucketUUID) == 0 {
			err = fmt.Errorf("Invalid bucket uuid %#v", bucketUUID)
			dbLog.Error(err)
			return err
		}
	}
	dbLog.Debugf("Delete bucket %#v (%s)", name, bucketUUID)
	txn.Delete([]byte(name))
	if err := txn.Commit(); err != nil {
		dbLog.Debugf("Failed to delete bucket %#v %v", name, err)
		return err
	}

	bucketDir := path.Join(rootDir, dbDirName, bucketsDirName, bucketUUID)
	if _, err := os.Stat(bucketDir); os.IsNotExist(err) {
		return nil
	}
	dbLog.Debugf("Removing bucket dir %s", bucketDir)
	err = os.RemoveAll(bucketDir)
	if err != nil {
		return err
	}
	return nil
}
