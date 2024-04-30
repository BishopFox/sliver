package traverse

import "os"

// UserHomeDir returns the current user's home directory.
func UserHomeDir(tc Context) (string, error) {
	return os.UserHomeDir()
}

// UserCacheDir returns the default root directory to use for user-specific cached data.
func UserCacheDir(tc Context) (string, error) {
	return os.UserCacheDir()
}

// UserConfigDir returns the default root directory to use for user-specific configuration data.
func UserConfigDir(tc Context) (string, error) {
	return os.UserConfigDir()
}

// TempDir returns the default directory to use for temporary files.
func TempDir(tc Context) (string, error) {
	return os.TempDir(), nil
}
