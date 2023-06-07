package xdg

import "os"

// UserCacheDir returns the cache base directory.
func UserCacheDir() (dir string, err error) {
	if dir = os.Getenv("XDG_CACHE_HOME"); dir == "" {
		dir, err = os.UserCacheDir()
	}
	return
}

// UserConfigDir returns the config base directory.
func UserConfigDir() (dir string, err error) {
	if dir = os.Getenv("XDG_CONFIG_HOME"); dir == "" {
		dir, err = os.UserConfigDir()
	}
	return
}
