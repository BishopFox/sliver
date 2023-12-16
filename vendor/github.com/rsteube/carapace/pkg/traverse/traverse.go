package traverse

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
)

type Context interface {
	Abs(s string) (string, error)
	Getenv(key string) string
	LookupEnv(key string) (string, bool)
}

// Parent returns the first parent directory containing any of the given names/directories.
func Parent(names ...string) func(tc Context) (string, error) {
	return func(tc Context) (string, error) {
		wd, err := tc.Abs("")
		if err != nil {
			return "", err
		}

		for _, name := range names {
			if dir, err := traverse(wd, name); err == nil {
				return filepath.Dir(dir), nil
			}
		}
		formattedNames := fmt.Sprintf("%#v", names)
		formattedNames = strings.TrimPrefix(formattedNames, "[]string{")
		formattedNames = strings.TrimSuffix(formattedNames, "}")
		return "", errors.New("could not find parent directory containing any of: " + formattedNames)
	}
}

// TODO also stop at `~`
func traverse(path string, name string) (target string, err error) {
	var absPath string
	if absPath, err = filepath.Abs(path); err == nil {
		target = filepath.ToSlash(absPath + "/" + strings.TrimSuffix(name, "/"))
		if _, err = os.Stat(target); err != nil {
			parent := filepath.Dir(absPath)
			if parent != path {
				return traverse(parent, name)
			} else {
				err = errors.New("could not find: " + name)
			}
		}
	}
	return
}

// Flag returns the value of given flag.
func Flag(f *pflag.Flag) func(tc Context) (string, error) {
	return func(tc Context) (string, error) {
		if f == nil {
			return "", errors.New("invalid argument [traverse.Flag]")
		}
		return f.Value.String(), nil
	}
}
