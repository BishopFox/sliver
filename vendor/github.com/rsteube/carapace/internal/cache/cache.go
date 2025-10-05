// Package cache provides disk cache for Actions
package cache

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rsteube/carapace/internal/env"
	"github.com/rsteube/carapace/internal/export"
	"github.com/rsteube/carapace/internal/uid"
	"github.com/rsteube/carapace/pkg/cache/key"
	"github.com/rsteube/carapace/pkg/xdg"
)

func WriteE(file string, e export.Export) (err error) {
	var m []byte
	if m, err = json.Marshal(e); err == nil {
		err = Write(file, m)
	}
	return
}

func Write(file string, content []byte) (err error) {
	return os.WriteFile(file, content, 0600)
}

func LoadE(file string, timeout time.Duration) (*export.Export, error) { // TODO reference
	content, err := Load(file, timeout)
	if err != nil {
		return nil, err
	}

	var e export.Export
	if err := json.Unmarshal(content, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

func Load(file string, timeout time.Duration) (b []byte, err error) {
	var stat os.FileInfo
	if stat, err = os.Stat(file); os.IsNotExist(err) || (timeout >= 0 && stat.ModTime().Add(timeout).Before(time.Now())) {
		return nil, errors.New("not exists or timeout exceeded")
	}
	return os.ReadFile(file)
}

// CacheDir creates a cache folder for current user and returns the path.
func CacheDir(name string) (dir string, err error) {
	var userCacheDir string
	userCacheDir, err = xdg.UserCacheDir()
	if err != nil {
		return
	}

	if m, sandboxErr := env.Sandbox(); sandboxErr == nil {
		userCacheDir = m.CacheDir()
	}

	dir = fmt.Sprintf("%v/carapace/%v/%v", userCacheDir, uid.Executable(), name)
	err = os.MkdirAll(dir, 0700)
	return
}

// File returns the cache filename for given values
// TODO cleanup
func File(callerFile string, callerLine int, keys ...key.Key) (file string, err error) {
	uid := uidKeys(callerFile, strconv.Itoa(callerLine))
	ids := make([]string, 0)
	for _, key := range keys {
		id, err := key()
		if err != nil {
			return "", err
		}
		ids = append(ids, id)
	}
	if dir, err := CacheDir(uid); err == nil {
		file = dir + "/" + uidKeys(ids...)
	}
	return
}

func uidKeys(keys ...string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(strings.Join(keys, "\001"))))
}
