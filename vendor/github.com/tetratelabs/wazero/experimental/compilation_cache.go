package experimental

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/tetratelabs/wazero/internal/compilationcache"
	"github.com/tetratelabs/wazero/internal/version"
)

// WithCompilationCacheDirName configures the destination directory of the compilation cache.
// Regardless of the usage of this, the compiled functions are cached in memory, but its lifetime is
// bound to the lifetime of wazero.Runtime or wazero.CompiledModule.
//
// If the dirname doesn't exist, this creates the directory.
//
// With the given non-empty directory, wazero persists the cache into the directory and that cache
// will be used as long as the running wazero version match the version of compilation wazero.
//
// A cache is only valid for use in one wazero.Runtime at a time. Concurrent use
// of a wazero.Runtime is supported, but multiple runtimes must not share the
// same directory.
//
// Note: The embedder must safeguard this directory from external changes.
//
// Usage:
//
//	ctx, _ := experimental.WithCompilationCacheDirName(context.Background(), "/home/me/.cache/wazero")
//	r := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigCompiler())
func WithCompilationCacheDirName(ctx context.Context, baseDirname string) (context.Context, error) {
	// Allow overriding for testing
	var wazeroVersion string
	if v := ctx.Value(version.WazeroVersionKey{}); v != nil {
		wazeroVersion = v.(string)
	} else {
		wazeroVersion = version.GetWazeroVersion()
	}

	// Resolve a potentially relative directory into an absolute one.
	var err error
	baseDirname, err = filepath.Abs(baseDirname)
	if err != nil {
		return nil, err
	}

	// Ensure the user-supplied directory.
	if err = mkdir(baseDirname); err != nil {
		return nil, err
	}

	// Create a version-specific directory to avoid conflicts.
	dirname := path.Join(baseDirname, "wazero-"+wazeroVersion+"-"+runtime.GOARCH+"-"+runtime.GOOS)
	if err = mkdir(dirname); err != nil {
		return nil, err
	}

	ctx = context.WithValue(ctx, compilationcache.FileCachePathKey{}, dirname)
	return ctx, nil
}

func mkdir(dirname string) error {
	if st, err := os.Stat(dirname); errors.Is(err, os.ErrNotExist) {
		// If the directory not found, create the cache dir.
		if err = os.MkdirAll(dirname, 0o700); err != nil {
			return fmt.Errorf("create directory %s: %v", dirname, err)
		}
	} else if err != nil {
		return err
	} else if !st.IsDir() {
		return fmt.Errorf("%s is not dir", dirname)
	}
	return nil
}
