package sliverpb

import (
	"google.golang.org/protobuf/proto"
)

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

------------------------------------------------------------------------------------


    ██     ██  █████  ██████  ███    ██ ██ ███    ██  ██████
    ██     ██ ██   ██ ██   ██ ████   ██ ██ ████   ██ ██
    ██  █  ██ ███████ ██████  ██ ██  ██ ██ ██ ██  ██ ██   ███
    ██ ███ ██ ██   ██ ██   ██ ██  ██ ██ ██ ██  ██ ██ ██    ██
     ███ ███  ██   ██ ██   ██ ██   ████ ██ ██   ████  ██████

	!!! The order of constants is APPEND ONLY !!!

	If you insert values into this file it is very important that you only append the
	order of the constants. If you do not you will break backwards compatibility with
	implants.

*/

const (

	// MsgRegister - Initial message from sliver with metadata
	MsgRegister = uint32(1 + iota)

	// MsgTaskReq - A local shellcode injection task
	MsgTaskReq

	// MsgPing - Confirm connection is open used as req/resp
	MsgPing

	// MsgKillSessionReq - Kill request to the sliver process
	MsgKillSessionReq

	// MsgLsReq - Request a directory listing from the remote system
	MsgLsReq
	// MsgLs - Directory listing (resp to MsgDirListReq)
	MsgLs

	// MsgDownloadReq - Request to download a file from the remote system
	MsgDownloadReq
	// MsgDownload - File contents for download (resp to DownloadReq)
	MsgDownload

	// MsgUploadReq - Upload a file to the remote file system
	MsgUploadReq
	// MsgUpload - Confirms the success/failure of the file upload (resp to MsgUploadReq)
	MsgUpload

	// MsgCdReq - Request a change directory on the remote system
	MsgCdReq

	// MsgPwdReq - A request to get the CWD from the remote process
	MsgPwdReq
	// MsgPwd - The CWD of the remote process (resp to MsgPwdReq)
	MsgPwd

	// MsgRmReq - Request to delete remote file
	MsgRmReq
	// MsgRm - Confirms the success/failure of delete request (resp to MsgRmReq)
	MsgRm

	// MsgMkdirReq - Request to create a directory on the remote system
	MsgMkdirReq
	// MsgMkdir - Confirms the success/failure of the mkdir request (resp to MsgMkdirReq)
	MsgMkdir

	// MsgPsReq - List processes req
	MsgPsReq
	// MsgPs - List processes resp
	MsgPs

	// MsgShellReq - Request to open a shell tunnel
	MsgShellReq
	// MsgShell - Response on starting shell
	MsgShell

	// MsgTunnelData - Data for duplex tunnels
	MsgTunnelData
	// MsgTunnelClose - Close a duplex tunnel
	MsgTunnelClose

	// MsgProcessDumpReq - Request to create a process dump
	MsgProcessDumpReq
	// MsgProcessDump - Dump of process)
	MsgProcessDump
	// MsgImpersonateReq - Request for process impersonation
	MsgImpersonateReq
	// MsgImpersonate - Output of the impersonation command
	MsgImpersonate
	// MsgRunAsReq - Request to run process as user
	MsgRunAsReq
	// MsgRunAs - Run process as user
	MsgRunAs
	// MsgRevToSelf - Revert to self
	MsgRevToSelf
	// MsgRevToSelfReq - Request to revert to self
	MsgRevToSelfReq
	// MsgInvokeGetSystemReq - Elevate as SYSTEM user
	MsgInvokeGetSystemReq
	// MsgGetSystem - Response to getsystem request
	MsgGetSystem
	// MsgInvokeExecuteAssemblyReq - Request to load and execute a .NET assembly
	MsgInvokeExecuteAssemblyReq
	// MsgExecuteAssemblyReq - Request to load and execute a .NET assembly
	MsgExecuteAssemblyReq
	// MsgExecuteAssembly - Output of the assembly execution
	MsgExecuteAssembly
	// MsgInvokeMigrateReq - Spawn a new sliver in a designated process
	MsgInvokeMigrateReq

	// MsgSideloadReq - request to sideload a binary
	MsgSideloadReq
	// MsgSideload - output of the binary
	MsgSideload

	// MsgSpawnDllReq - Reflective DLL injection request
	MsgSpawnDllReq
	// MsgSpawnDll - Reflective DLL injection output
	MsgSpawnDll

	// MsgIfconfigReq - Ifconfig (network interface config) request
	MsgIfconfigReq
	// MsgIfconfig - Ifconfig response
	MsgIfconfig

	// MsgExecuteReq - Execute a command on the remote system
	MsgExecuteReq

	// MsgTerminateReq - Request to kill a remote process
	MsgTerminateReq

	// MsgTerminate - Kill a remote process
	MsgTerminate

	// MsgScreenshotReq - Request to take a screenshot
	MsgScreenshotReq

	// MsgScreenshot - Response with the screenshots
	MsgScreenshot

	// MsgNetstatReq - Netstat request
	MsgNetstatReq

	// *** Pivots ***

	// MsgPivotStartListenerReq - Start a listener
	MsgPivotStartListenerReq
	// MsgPivotStopListenerReq - Stop a listener
	MsgPivotStopListenerReq
	// MsgPivotListenersReq - List listeners request
	MsgPivotListenersReq
	// MsgPivotListeners - List listeners response
	MsgPivotListeners
	// MsgPivotPeerPing - Pivot peer ping message
	MsgPivotPeerPing
	// MsgPivotServerPing - Pivot peer ping message
	MsgPivotServerPing
	// PivotServerKeyExchange - Pivot to server key exchange
	MsgPivotServerKeyExchange
	// MsgPivotPeerEnvelope - An envelope from a pivot peer
	MsgPivotPeerEnvelope
	// MsgPivotPeerFailure - Failure to send an envelope to a pivot peer
	MsgPivotPeerFailure
	// MsgPivotSessionEnvelope
	MsgPivotSessionEnvelope

	// MsgStartServiceReq - Request to start a service
	MsgStartServiceReq
	// MsgStartService - Response to start service request
	MsgStartService
	// MsgStopServiceReq - Request to stop a remote service
	MsgStopServiceReq
	// MsgRemoveServiceReq - Request to remove a remote service
	MsgRemoveServiceReq
	// MsgMakeTokenReq - Request for MakeToken
	MsgMakeTokenReq
	// MsgMakeToken - Response for MakeToken
	MsgMakeToken
	// MsgEnvReq - Request to get environment variables
	MsgEnvReq
	// MsgEnvInfo - Response to environment variable request
	MsgEnvInfo
	// MsgSetEnvReq
	MsgSetEnvReq
	// MsgSetEnv
	MsgSetEnv
	// MsgExecuteWindowsReq - Execute request executed with the current (Windows) token
	MsgExecuteWindowsReq
	// MsgRegistryReadReq
	MsgRegistryReadReq
	// MsgRegistryWriteReq
	MsgRegistryWriteReq
	// MsgRegistryCreateKeyReq
	MsgRegistryCreateKeyReq

	// MsgWGStartPortFwdReq - Request to start a port forwarding in a WG transport
	MsgWGStartPortFwdReq
	// MsgWGStopPortFwdReq - Request to stop a port forwarding in a WG transport
	MsgWGStopPortFwdReq
	// MsgWGStartSocks - Request to start a socks server in a WG transport
	MsgWGStartSocksReq
	// MsgWGStopSocks - Request to stop a socks server in a WG transport
	MsgWGStopSocksReq
	// MsgWGListForwarders
	MsgWGListForwardersReq
	// MsgWGListSocks
	MsgWGListSocksReq

	// MsgPortfwdReq - Establish a port forward
	MsgPortfwdReq
	// MsgPortfwd - Response of port forward
	MsgPortfwd

	// MsgSocksData - Response of SocksData
	MsgSocksData

	// MsgReconfigureReq
	MsgReconfigureReq

	// MsgReconfigure - Set Reconfigure
	MsgReconfigure

	// MsgUnsetEnvReq
	MsgUnsetEnvReq

	// MsgSSHCommandReq - Run a SSH command
	MsgSSHCommandReq

	// MsgGetPrivsReq - Get privileges (Windows)
	MsgGetPrivsReq

	// MsgRegistryListReq - List registry sub keys
	MsgRegistrySubKeysListReq
	// MsgRegistryListValuesReq - List registry values
	MsgRegistryListValuesReq
	// MsgRegisterExtensionReq - Register a new extension
	MsgRegisterExtensionReq

	// MsgCallExtensionReq - Run an extension command
	MsgCallExtensionReq
	// MsgListExtensionsReq - List loaded extensions
	MsgListExtensionsReq

	// MsgBeaconRegister - Register a new beacon
	MsgBeaconRegister
	// MsgBeaconTasks - Send/recv batches of beacon tasks
	MsgBeaconTasks

	// MsgOpenSession - Open a new session
	MsgOpenSession
	// MsgCloseSession - Close the active session
	MsgCloseSession

	// MsgRegistryDeleteKeyReq
	MsgRegistryDeleteKeyReq

	// MsgMvReq - Request to move or rename a file
	MsgMvReq
	// MsgMv - Confirms the success/failure of the mv request (resp to MsgMvReq)
	MsgMv

	// MsgCurrentTokenOwnerReq - Request to query the thread token owner
	MsgCurrentTokenOwnerReq
	// MsgCurrentTokenOwner - Replies with the current thread owner (resp to MsfCurrentToken)
	MsgCurrentTokenOwner
	// MsgInvokeInProcExecuteAssemblyReq - Request to load and execute a .NET assembly in-process
	MsgInvokeInProcExecuteAssemblyReq

	MsgRportFwdStopListenerReq

	MsgRportFwdStartListenerReq

	MsgRportFwdListener

	MsgRportFwdListeners

	MsgRportFwdListenersReq

	MsgRPortfwdReq

	// MsgChmodReq - Request to chmod a file
	MsgChmodReq
	// MsgChmod - Replies with file path
	MsgChmod
	// MsgChownReq - Request to chown a file
	MsgChownReq
	// MsgChown - Replies with file path
	MsgChown
	// MsgChtimesReq - Request to chtimes a file
	MsgChtimesReq
	// MsgChown - Replies with file path
	MsgChtimes

	// MsgChmodReq - Request to chmod a file
	MsgMemfilesListReq

	// MsgChownReq - Request to chown a file
	MsgMemfilesAddReq
	// MsgChown - Replies with file path
	MsgMemfilesAdd

	// MsgChtimesReq - Request to chtimes a file
	MsgMemfilesRmReq
	// MsgChown - Replies with file path
	MsgMemfilesRm

	// Wasm Extension messages
	MsgRegisterWasmExtensionReq
	MsgDeregisterWasmExtensionReq
	MsgRegisterWasmExtension
	MsgListWasmExtensionsReq
	MsgListWasmExtensions
	MsgExecWasmExtensionReq
	MsgExecWasmExtension

	// MsgCpReq - Request to copy a file from one place to another
	MsgCpReq
	// MsgCp - Confirms the success/failure, as well as the total number of bytes
	// written of the cp request (resp to MsgCpReq)
	MsgCp

	// MsgGrepReq - Request to grep for data
	MsgGrepReq

	// Services messages
	MsgServicesReq
	MsgServiceDetailReq
	MsgStartServiceByNameReq

	MsgRegistryReadHiveReq

	// MsgMountReq - Request filesystem mounts
	MsgMountReq
)

// Constants to replace enums
const (
	// Port forward protocols
	PortFwdProtoTCP = 1
	PortFwdProtoUDP = 2

	// Registry types
	RegistryTypeBinary = 1
	RegistryTypeString = 2
	RegistryTypeDWORD  = 3
	RegistryTypeQWORD  = 4
)

// MsgNumber - Get a message number of type
func MsgNumber(request proto.Message) uint32 {
	switch request.(type) {

	case *Register:
		return MsgRegister
	case *TaskReq:
		return MsgTaskReq
	case *Ping:
		return MsgPing
	case *KillReq:
		return MsgKillSessionReq
	case *LsReq:
		return MsgLsReq
	case *Ls:
		return MsgLs
	case *DownloadReq:
		return MsgDownloadReq
	case *Download:
		return MsgDownload
	case *UploadReq:
		return MsgUploadReq
	case *Upload:
		return MsgUpload
	case *CdReq:
		return MsgCdReq
	case *PwdReq:
		return MsgPwdReq
	case *Pwd:
		return MsgPwd
	case *RmReq:
		return MsgRmReq
	case *Rm:
		return MsgRm
	case *MkdirReq:
		return MsgMkdirReq
	case *Mkdir:
		return MsgMkdir
	case *PsReq:
		return MsgPsReq
	case *Ps:
		return MsgPs
	case *ShellReq:
		return MsgShellReq
	case *Shell:
		return MsgShell
	case *ProcessDumpReq:
		return MsgProcessDumpReq
	case *ProcessDump:
		return MsgProcessDump
	case *ImpersonateReq:
		return MsgImpersonateReq
	case *Impersonate:
		return MsgImpersonate
	case *RunAsReq:
		return MsgRunAsReq
	case *RunAs:
		return MsgRunAs
	case *RevToSelfReq:
		return MsgRevToSelfReq
	case *InvokeGetSystemReq:
		return MsgInvokeGetSystemReq
	case *GetSystem:
		return MsgGetSystem
	case *ExecuteAssemblyReq:
		return MsgExecuteAssemblyReq
	case *InvokeExecuteAssemblyReq:
		return MsgInvokeExecuteAssemblyReq
	case *ExecuteAssembly:
		return MsgExecuteAssembly
	case *ExecuteWindowsReq:
		return MsgExecuteWindowsReq
	case *InvokeMigrateReq:
		return MsgInvokeMigrateReq
	case *SideloadReq:
		return MsgSideloadReq
	case *Sideload:
		return MsgSideload
	case *SpawnDllReq:
		return MsgSpawnDllReq
	case *SpawnDll:
		return MsgSpawnDll
	case *IfconfigReq:
		return MsgIfconfigReq
	case *Ifconfig:
		return MsgIfconfig
	case *ExecuteReq:
		return MsgExecuteReq
	case *TerminateReq:
		return MsgTerminateReq
	case *Terminate:
		return MsgTerminate
	case *ScreenshotReq:
		return MsgScreenshotReq
	case *Screenshot:
		return MsgScreenshot
	case *NetstatReq:
		return MsgNetstatReq

	case *PivotStartListenerReq:
		return MsgPivotStartListenerReq
	case *PivotStopListenerReq:
		return MsgPivotStopListenerReq
	case *PivotListenersReq:
		return MsgPivotListenersReq
	case *PivotListeners:
		return MsgPivotListeners
	case *PivotPing:
		return MsgPivotPeerPing

	case *StartServiceReq:
		return MsgStartServiceReq
	case *StopServiceReq:
		return MsgStopServiceReq
	case *RemoveServiceReq:
		return MsgRemoveServiceReq
	case *MakeTokenReq:
		return MsgMakeTokenReq
	case *MakeToken:
		return MsgMakeToken
	case *EnvReq:
		return MsgEnvReq
	case *EnvInfo:
		return MsgEnvInfo
	case *SetEnvReq:
		return MsgSetEnvReq
	case *SetEnv:
		return MsgSetEnv
	case *UnsetEnvReq:
		return MsgUnsetEnvReq
	case *RegistryReadReq:
		return MsgRegistryReadReq
	case *RegistryWriteReq:
		return MsgRegistryWriteReq
	case *RegistryCreateKeyReq:
		return MsgRegistryCreateKeyReq
	case *RegistryDeleteKeyReq:
		return MsgRegistryDeleteKeyReq

	case *WGPortForwardStartReq:
		return MsgWGStartPortFwdReq
	case *WGPortForwardStopReq:
		return MsgWGStopPortFwdReq
	case *WGSocksStartReq:
		return MsgWGStartSocksReq
	case *WGSocksStopReq:
		return MsgWGStopSocksReq
	case *WGTCPForwardersReq:
		return MsgWGListForwardersReq
	case *WGSocksServersReq:
		return MsgWGListSocksReq

	case *PortfwdReq:
		return MsgPortfwdReq
	case *Portfwd:
		return MsgPortfwd

	case *ReconfigureReq:
		return MsgReconfigureReq
	case *Reconfigure:
		return MsgReconfigure

	case *SSHCommandReq:
		return MsgSSHCommandReq

	case *GetPrivsReq:
		return MsgGetPrivsReq
	case *RegistrySubKeyListReq:
		return MsgRegistrySubKeysListReq
	case *RegistryListValuesReq:
		return MsgRegistryListValuesReq

	case *RegisterExtensionReq:
		return MsgRegisterExtensionReq
	case *CallExtensionReq:
		return MsgCallExtensionReq
	case *ListExtensionsReq:
		return MsgListExtensionsReq

	case *BeaconTasks:
		return MsgBeaconTasks

	case *OpenSession:
		return MsgOpenSession
	case *CloseSession:
		return MsgCloseSession

	case *MvReq:
		return MsgMvReq
	case *Mv:
		return MsgMv

	case *CpReq:
		return MsgCpReq
	case *Cp:
		return MsgCp

	case *CurrentTokenOwnerReq:
		return MsgCurrentTokenOwnerReq
	case *CurrentTokenOwner:
		return MsgCurrentTokenOwner
	case *InvokeInProcExecuteAssemblyReq:
		return MsgInvokeInProcExecuteAssemblyReq

	case *RportFwdStartListenerReq:
		return MsgRportFwdStartListenerReq
	case *RportFwdStopListenerReq:
		return MsgRportFwdStopListenerReq
	case *RportFwdListenersReq:
		return MsgRportFwdListenersReq
	case *RportFwdListeners:
		return MsgRportFwdListeners
	case *RPortfwdReq:
		return MsgRPortfwdReq

	case *ChmodReq:
		return MsgChmodReq
	case *Chmod:
		return MsgChmod
	case *ChownReq:
		return MsgChownReq
	case *Chown:
		return MsgChown
	case *ChtimesReq:
		return MsgChtimesReq
	case *Chtimes:
		return MsgChtimes

	case *GrepReq:
		return MsgGrepReq

	case *MountReq:
		return MsgMountReq

	case *MemfilesListReq:
		return MsgMemfilesListReq

	case *MemfilesAddReq:
		return MsgMemfilesAddReq
	case *MemfilesAdd:
		return MsgMemfilesAdd
	case *MemfilesRmReq:
		return MsgMemfilesRmReq
	case *MemfilesRm:
		return MsgMemfilesRm

	case *RegisterWasmExtensionReq:
		return MsgRegisterWasmExtensionReq
	case *DeregisterWasmExtensionReq:
		return MsgDeregisterWasmExtensionReq
	case *ListWasmExtensionsReq:
		return MsgListWasmExtensionsReq
	case *ExecWasmExtensionReq:
		return MsgExecWasmExtensionReq

	case *ServicesReq:
		return MsgServicesReq

	case *ServiceDetailReq:
		return MsgServiceDetailReq

	case *StartServiceByNameReq:
		return MsgStartServiceByNameReq

	case *RegistryReadHiveReq:
		return MsgRegistryReadHiveReq
	}
	return uint32(0)
}
