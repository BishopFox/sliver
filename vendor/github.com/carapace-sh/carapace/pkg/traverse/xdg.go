package traverse

import (
	"path/filepath"
)

// XdgCacheHome returns the cache directory (fallback to UserCacheDir).
func XdgCacheHome(tc Context) (dir string, err error) {
	if dir = tc.Getenv("XDG_CACHE_HOME"); dir == "" {
		dir, err = UserCacheDir(tc)
	}
	dir = filepath.ToSlash(dir)
	return
}

// XdgConfigHome returns the home directory (fallback to UserConfigDir).
func XdgConfigHome(tc Context) (dir string, err error) {
	if dir = tc.Getenv("XDG_CONFIG_HOME"); dir == "" {
		dir, err = UserConfigDir(tc)
	}
	dir = filepath.ToSlash(dir)
	return
}
