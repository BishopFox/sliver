package filecache

import (
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path"
	"sync"
)

// New returns a new Cache implemented by fileCache.
func New(dir string) Cache {
	return newFileCache(dir)
}

func newFileCache(dir string) *fileCache {
	return &fileCache{dirPath: dir}
}

// fileCache persists compiled functions into dirPath.
//
// Note: this can be expanded to do binary signing/verification, set TTL on each entry, etc.
type fileCache struct {
	dirPath string
	mux     sync.RWMutex
}

type fileReadCloser struct {
	*os.File
	fc *fileCache
}

func (fc *fileCache) path(key Key) string {
	return path.Join(fc.dirPath, hex.EncodeToString(key[:]))
}

func (fc *fileCache) Get(key Key) (content io.ReadCloser, ok bool, err error) {
	// TODO: take lock per key for more efficiency vs the complexity of impl.
	fc.mux.RLock()
	unlock := fc.mux.RUnlock
	defer func() {
		if unlock != nil {
			unlock()
		}
	}()

	f, err := os.Open(fc.path(key))
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	} else {
		// Unlock is done inside the content.Close() at the call site.
		unlock = nil
		return &fileReadCloser{File: f, fc: fc}, true, nil
	}
}

// Close wraps the os.File Close to release the read lock on fileCache.
func (f *fileReadCloser) Close() (err error) {
	defer f.fc.mux.RUnlock()
	err = f.File.Close()
	return
}

func (fc *fileCache) Add(key Key, content io.Reader) (err error) {
	// TODO: take lock per key for more efficiency vs the complexity of impl.
	fc.mux.Lock()
	defer fc.mux.Unlock()

	// Use rename for an atomic write
	path := fc.path(key)
	file, err := os.Create(path + ".tmp")
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			_ = os.Remove(file.Name())
		}
	}()
	defer file.Close()
	if _, err = io.Copy(file, content); err != nil {
		return
	}
	if err = file.Sync(); err != nil {
		return
	}
	if err = file.Close(); err != nil {
		return
	}
	err = os.Rename(file.Name(), path)
	return
}

func (fc *fileCache) Delete(key Key) (err error) {
	// TODO: take lock per key for more efficiency vs the complexity of impl.
	fc.mux.Lock()
	defer fc.mux.Unlock()

	err = os.Remove(fc.path(key))
	if errors.Is(err, os.ErrNotExist) {
		err = nil
	}
	return
}
