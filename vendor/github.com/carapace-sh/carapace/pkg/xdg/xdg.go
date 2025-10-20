package xdg

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// UserCacheDir returns the user cache base directory.
func UserCacheDir() (dir string, err error) {
	if dir = os.Getenv("XDG_CACHE_HOME"); dir == "" || !filepath.IsAbs(dir) {
		dir, err = os.UserCacheDir()
	}
	dir = filepath.ToSlash(dir)
	return
}

// UserConfigDir returns the user config base directory.
func UserConfigDir() (dir string, err error) {
	if dir = os.Getenv("XDG_CONFIG_HOME"); dir == "" || !filepath.IsAbs(dir) {
		dir, err = os.UserConfigDir()
	}
	dir = filepath.ToSlash(dir)
	return
}

// ConfigDirs returns the global config base directories.
func ConfigDirs() (dirs []string, err error) {
	switch runtime.GOOS {
	case "windows":
		dir, ok := os.LookupEnv("PROGRAMDATA")
		if !ok {
			return nil, errors.New("missing PROGRAMDATA environment variable")
		}
		dirs = append(dirs, dir)

	case "darwin":
		dirs = append(dirs, "/Library/Application Support")
	default:
		dirs = append(dirs, "/etc/xdg")

	}

	if v, ok := os.LookupEnv("XDG_CONFIG_DIRS"); ok {
		xdgDirs := make([]string, 0)
		for _, dir := range strings.Split(v, string(os.PathSeparator)) {
			if filepath.IsAbs(dir) {
				xdgDirs = append(xdgDirs, dir)
			}
		}
		dirs = append(xdgDirs, dirs...)
	}

	for index, dir := range dirs {
		dirs[index] = filepath.ToSlash(dir)
	}
	return
}
