package wasm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/util"
	"github.com/spf13/cobra"
)

func parseMemFS(cmd *cobra.Command, con *console.SliverClient, args []string) (map[string][]byte, error) {
	memfs := make(map[string][]byte)

	totalSize := 0
	fileArg, _ := cmd.Flags().GetString("file")
	dirFlag, _ := cmd.Flags().GetString("dir")
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
			totalSize += len(data)
		}
		if dirFlag == "" {
			con.PrintInfof("Added %d file(s) to memfs\n", count)
			return memfs, nil
		}
	}

	dirArg, _ := cmd.Flags().GetString("dir")
	dirCount := 0
	fileCount := 0
	if dirArg != "" {
		dirs, err := filepath.Glob(dirArg)
		if err != nil {
			return nil, err
		}
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
				totalSize += len(data)
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("(dir) %s: %s", dir, err)
			}
			dirCount++
		}
	}
	con.PrintInfof("Added %d files from %d directories to memfs (%s)\n",
		fileCount, dirCount, util.ByteCountBinary(int64(totalSize)))
	return memfs, nil
}
