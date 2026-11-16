//go:build !windows

package spoof

import "os/exec"

// SpoofParent - Stub for non-windows platforms
func SpoofParent(ppid uint32, cmd *exec.Cmd) error {
	return nil
}
