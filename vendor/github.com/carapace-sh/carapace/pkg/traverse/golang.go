package traverse

import (
	"os"
	"path/filepath"
	"strings"
)

// GoBinDir TODO experimental
func GoBinDir(tc Context) (dir string, err error) {
	if path, ok := tc.LookupEnv("GOBIN"); ok {
		return filepath.ToSlash(path), nil
	}

	if path, ok := tc.LookupEnv("GOPATH"); ok {
		dir = strings.Split(path, string(os.PathListSeparator))[0]
	}

	if dir == "" {
		if dir, err = UserHomeDir(tc); err != nil {
			return "", err
		}
		dir = filepath.Join(dir, "go")
	}

	return filepath.ToSlash(filepath.Join(dir, "bin")), nil
}
