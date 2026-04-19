//go:build darwin

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
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"
	"unsafe"
)

const localTaskHelperEnv = "SLIVER_LOCALTASK_HELPER"
const localTaskSentinel = "PARENT_ALIVE_AFTER_LOCALTASK"

// darwin/amd64 mov eax,0x2000001 ; xor edi,edi ; syscall  — exit(0)
var exitShellcodeDarwinAmd64 = []byte{0xb8, 0x01, 0x00, 0x00, 0x02, 0x31, 0xff, 0x0f, 0x05}

// darwin/amd64 mov eax,0x2000001 ; mov edi,42 ; syscall  — exit(42)
var exit42ShellcodeDarwinAmd64 = []byte{0xb8, 0x01, 0x00, 0x00, 0x02, 0xbf, 0x2a, 0x00, 0x00, 0x00, 0x0f, 0x05}

// darwin/arm64 movz x0,#0 ; movz x16,#1 ; svc #0x80  — exit(0)
// Stored as [N]uint32 so the backing storage is 4-byte aligned (ARM64 SIGBUS
// otherwise).
var exitShellcodeDarwinArm64Words = [3]uint32{0xd2800000, 0xd2800030, 0xd4001001}

// darwin/arm64 movz x0,#42 ; movz x16,#1 ; svc #0x80  — exit(42)
var exit42ShellcodeDarwinArm64Words = [3]uint32{0xd2800540, 0xd2800030, 0xd4001001}

func asBytes[T any](words *T) []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(words)), unsafe.Sizeof(*words))
}

func exitShellcode() []byte {
	switch runtime.GOARCH {
	case "amd64":
		return exitShellcodeDarwinAmd64
	case "arm64":
		return asBytes(&exitShellcodeDarwinArm64Words)
	}
	return nil
}

func exit42Shellcode() []byte {
	switch runtime.GOARCH {
	case "amd64":
		return exit42ShellcodeDarwinAmd64
	case "arm64":
		return asBytes(&exit42ShellcodeDarwinArm64Words)
	}
	return nil
}

func TestMain(m *testing.M) {
	switch os.Getenv(localTaskHelperEnv) {
	case "exit0":
		_ = LocalTask(exitShellcode(), false)
		time.Sleep(500 * time.Millisecond)
		fmt.Println(localTaskSentinel)
		os.Exit(0)
	case "exit42":
		err := LocalTask(exit42Shellcode(), false)
		if err != nil {
			fmt.Printf("LOCALTASK_ERR=%s\n", err.Error())
		} else {
			fmt.Println("LOCALTASK_ERR=<nil>")
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func runHelper(t *testing.T, mode string) []byte {
	t.Helper()
	if exitShellcode() == nil {
		t.Skipf("no exit shellcode for GOARCH=%s", runtime.GOARCH)
	}
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), localTaskHelperEnv+"="+mode)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("helper subprocess (%s) exited abnormally: %v\noutput: %s", mode, err, out)
	}
	return out
}

func TestLocalTask_ParentSurvives(t *testing.T) {
	out := runHelper(t, "exit0")
	if !bytes.Contains(out, []byte(localTaskSentinel)) {
		t.Fatalf("expected sentinel %q in helper output:\n%s", localTaskSentinel, out)
	}
}

// Accepts signal-kill as well as clean non-zero exit: macOS Apple Silicon
// W^X faults mprotected-page execution with SIGBUS regardless of the
// shellcode bytes, which still exercises LocalTask's error path.
func TestLocalTask_ReportsChildFailure(t *testing.T) {
	out := runHelper(t, "exit42")
	if bytes.Contains(out, []byte("LOCALTASK_ERR=shellcode child exited 42")) {
		return
	}
	if bytes.Contains(out, []byte("LOCALTASK_ERR=shellcode child killed by signal")) {
		return
	}
	t.Fatalf("expected LOCALTASK_ERR for exit-42 or signal kill:\n%s", out)
}
