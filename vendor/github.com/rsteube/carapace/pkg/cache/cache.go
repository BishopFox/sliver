// Package cache provides cache keys
package cache

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Key provides a cache key.
type Key func() (string, error)

// String creates a CacheKey for given strings.
func String(s ...string) Key {
	return func() (string, error) {
		return strings.Join(s, "\n"), nil
	}
}

// FileChecksum creates a CacheKey for given file.
func FileChecksum(file string) Key {
	return func() (checksum string, err error) {
		var content []byte
		if content, err = os.ReadFile(file); err == nil {
			checksum = fmt.Sprintf("%x", sha1.Sum(content))
		}
		return
	}
}

// FileStats creates a CacheKey for given file.
func FileStats(file string) Key {
	return func() (checksum string, err error) {
		var path string
		if path, err = filepath.Abs(file); err == nil {
			var info os.FileInfo
			if info, err = os.Stat(file); err == nil {
				return String(path, strconv.FormatInt(info.Size(), 10), info.ModTime().String())()
			}
		}
		return
	}
}
