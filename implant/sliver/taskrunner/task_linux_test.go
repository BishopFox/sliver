//go:build linux

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
)

const localTaskHelperEnv = "SLIVER_LOCALTASK_HELPER"
const localTaskSentinel = "PARENT_ALIVE_AFTER_LOCALTASK"

// linux/amd64: xor eax,eax ; xor edi,edi ; mov al,0x3c ; syscall  — exit(0)
var exitShellcodeAmd64 = []byte{0x31, 0xc0, 0x31, 0xff, 0xb0, 0x3c, 0x0f, 0x05}

// linux/amd64: xor eax,eax ; mov al,0x3c ; mov edi,42 ; syscall  — exit(42)
var exit42ShellcodeAmd64 = []byte{0x31, 0xc0, 0xb0, 0x3c, 0xbf, 0x2a, 0x00, 0x00, 0x00, 0x0f, 0x05}

func TestMain(m *testing.M) {
	switch os.Getenv(localTaskHelperEnv) {
	case "exit0":
		_ = LocalTask(exitShellcodeAmd64, false)
		time.Sleep(500 * time.Millisecond)
		fmt.Println(localTaskSentinel)
		os.Exit(0)
	case "exit42":
		err := LocalTask(exit42ShellcodeAmd64, false)
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
	if runtime.GOARCH != "amd64" {
		t.Skipf("exit shellcode is amd64-specific; GOARCH=%s", runtime.GOARCH)
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

func TestLocalTask_ReportsChildFailure(t *testing.T) {
	out := runHelper(t, "exit42")
	if !bytes.Contains(out, []byte("LOCALTASK_ERR=shellcode child exited 42")) {
		t.Fatalf("expected exit-42 error in helper output:\n%s", out)
	}
}
