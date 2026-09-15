//go:build !windows

// Package rpc implements the Sliver server RPC surface.
package rpc

import "os"

func publishCrackFileChunk(temporaryPath, finalPath string) error {
	return os.Rename(temporaryPath, finalPath)
}

func syncCrackFileDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	if err := directory.Sync(); err != nil {
		_ = directory.Close()
		return err
	}
	return directory.Close()
}
