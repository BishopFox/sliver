package db

import (
	"errors"
	"fmt"
	"os"
	"path"

	"sliver/server/assets"
	"sliver/server/log"

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
	rootDB = getRootDB()
	dbLog  = log.NamedLogger("db", "")
)

// Bucket - Badger database and namespaced logger
type Bucket struct {
	DB  *badger.DB
	Log *logrus.Entry
}

func getRootDB() *badger.DB {
	rootDir := assets.GetRootAppDir()
	dbDir := path.Join(rootDir, dbDirName, rootDBDirName)
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		os.MkdirAll(dbDir, os.ModePerm)
	}
	opts := badger.DefaultOptions
	opts.Dir = dbDir
	opts.ValueDir = dbDir
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
		if err := txn.Commit(nil); err != nil {
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
		dbLog.Debugf("Using bucket %#v (%s)", name, bucketUUID)
	}

	bucketDir := path.Join(rootDir, dbDirName, bucketsDirName, bucketUUID)
	if _, err := os.Stat(bucketDir); os.IsNotExist(err) {
		os.MkdirAll(bucketDir, os.ModePerm)
	}
	dbLog.Debugf("Loading db from %s", bucketDir)
	opts := badger.DefaultOptions
	opts.Dir = bucketDir
	opts.ValueDir = bucketDir
	db, err := badger.Open(opts)
	if err != nil {
		dbLog.Errorf("Failed to open db %s", err)
		return nil, err
	}
	return &Bucket{
		DB:  db,
		Log: log.NamedLogger("db", name),
	}, nil
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
	if err := txn.Commit(nil); err != nil {
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
