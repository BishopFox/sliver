//go:build windows

// Package rpc implements the Sliver server RPC surface.
package rpc

import "golang.org/x/sys/windows"

func publishCrackFileChunk(temporaryPath, finalPath string) error {
	temporary, err := windows.UTF16PtrFromString(temporaryPath)
	if err != nil {
		return err
	}
	final, err := windows.UTF16PtrFromString(finalPath)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(temporary, final, windows.MOVEFILE_WRITE_THROUGH)
}

func syncCrackFileDirectory(string) error {
	// MOVEFILE_WRITE_THROUGH waits for the rename to reach disk. Windows does
	// not expose the directory-fsync primitive used on Unix.
	return nil
}
