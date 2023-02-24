package wasm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bishopfox/sliver/client/console"
	"github.com/desertbit/grumble"
)

func parseMemFS(ctx *grumble.Context, con *console.SliverConsoleClient) (map[string][]byte, error) {

	memfs := make(map[string][]byte)

	fileArg := ctx.Flags.String("file")
	if fileArg != "" {
		files, err := filepath.Glob(fileArg)
		if err != nil {
			return nil, err
		}
		count := 0
		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				return nil, fmt.Errorf("(file) %s: %s", file, err)
			}
			con.PrintInfof("Adding '%s' to memfs (%d bytes)", file, len(data))
			memfs[filepath.Base(file)] = data
			count++
		}
		con.PrintInfof("Added %d files to memfs\n", count)
	}

	dirArg := ctx.Flags.String("dir")
	if dirArg != "" {
		dirs, err := filepath.Glob(dirArg)
		if err != nil {
			return nil, err
		}
		dirCount := 0
		fileCount := 0
		for _, dir := range dirs {
			err = filepath.WalkDir(dir, func(walkingPath string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				data, err := os.ReadFile(walkingPath)
				if err != nil {
					return fmt.Errorf("%s: %s", walkingPath, err)
				}
				con.PrintInfof("Adding '%s' to memfs (%d bytes)", walkingPath, len(data))
				memfs[filepath.Base(walkingPath)] = data
				fileCount++
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("(dir) %s: %s", dir, err)
			}
			dirCount++
		}
		con.PrintInfof("Added %d files from %d directories to memfs\n", fileCount, dirCount)
	}

	return nil, nil
}
