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

	// MsgNamedPipesReq - Request to take create a new named pipe listener
	MsgNamedPipesReq
	// MsgNamedPipes - Response with the result
	MsgNamedPipes

	// MsgTCPPivotReq - Request to take create a new MTLS listener
	MsgTCPPivotReq
	// MsgTCPPivot - Response with the result
	MsgTCPPivot
	// MsgPivotListReq
	MsgPivotListReq

	// MsgPivotOpen - Request to create a new pivot tunnel
	MsgPivotOpen
	// MsgPivotClose - Request to notify the closing of an existing pivot tunnel
	MsgPivotClose
	// MsgPivotData - Request that encapsulates and envelope form a sliver to the server though the pivot and viceversa
	MsgPivotData
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
	// MsgExecuteTokenReq - Execute request executed with the current (Windows) token
	MsgExecuteTokenReq
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

	// MsgReconnectIntervalReq
	MsgReconnectIntervalReq

	MsgReconnectInterval

	// MsgPollIntervalReq
	MsgPollIntervalReq

	MsgPollInterval

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

	case *KillSessionReq:
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
	case *ExecuteTokenReq:
		return MsgExecuteTokenReq

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

	case *NamedPipesReq:
		return MsgNamedPipesReq
	case *NamedPipes:
		return MsgNamedPipes

	case *TCPPivotReq:
		return MsgTCPPivotReq
	case *TCPPivot:
		return MsgTCPPivot

	case *PivotOpen:
		return MsgPivotOpen
	case *PivotClose:
		return MsgPivotClose
	case *PivotData:
		return MsgPivotData
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

	case *PivotListReq:
		return MsgPivotListReq

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

	case *ReconnectIntervalReq:
		return MsgReconnectIntervalReq

	case *ReconnectInterval:
		return MsgReconnectInterval

	case *PollIntervalReq:
		return MsgPollIntervalReq

	case *PollInterval:
		return MsgPollInterval
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
	}

	return uint32(0)
}
