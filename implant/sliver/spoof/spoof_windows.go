//go:build windows
// +build windows

package spoof

import (
	"golang.org/x/sys/windows"
	"os/exec"
	"syscall"
	// {{if .Config.Debug}}
	"log"
	// {{end}}
)

func SpoofParent(ppid uint32, cmd *exec.Cmd) error {
	parentHandle, err := windows.OpenProcess(windows.PROCESS_CREATE_PROCESS|windows.PROCESS_DUP_HANDLE|windows.PROCESS_QUERY_INFORMATION, false, ppid)
	if err != nil {
		//{{if .Config.Debug}}
		log.Printf("OpenProcess failed: %v\n", err)
		//{{end}}
		return err
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &windows.SysProcAttr{}
	}
	cmd.SysProcAttr.ParentProcess = syscall.Handle(parentHandle)
	return nil
}
