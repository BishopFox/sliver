package db

import (
	"os"
	"path"

	"sliver/server/assets"
	"sliver/server/log"

	"github.com/dgraph-io/badger"
	"github.com/google/uuid"
)

const (
	dbDirName      = "db"
	rootDBDirName  = "root"
	bucketsDirName = "buckets"
)

var (
	rootDB = getRootDB()
	dbLog  = log.NamedLogger("db", "all")
)

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

// Bucket returns a namespaced database, names are mapped to directoires
// thru the rootDB which stores Name<->UUID pairs, this allows us to support
// bucket names with arbitrary string values
func Bucket(name string) (*badger.DB, error) {
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
		var val []byte
		item.ValueCopy(val)
		bucketUUID = string(val)
		dbLog.Debugf("Using bucket %#v (%s)", name, bucketUUID)
	}

	bucketDir := path.Join(rootDir, dbDirName, bucketsDirName, bucketUUID)
	if _, err := os.Stat(bucketDir); os.IsNotExist(err) {
		os.MkdirAll(bucketDir, os.ModePerm)
	}
	dbLog.Debugf("Loading db from bucket dir: %s", bucketDir)
	opts := badger.DefaultOptions
	opts.Dir = bucketDir
	opts.ValueDir = bucketDir
	return badger.Open(opts)
}
