package sliverpb

import (
	proto "github.com/golang/protobuf/proto"
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
*/

// Message Name Constants

const (

	// MsgRegister - Initial message from sliver with metadata
	MsgRegister = uint32(1 + iota)

	// MsgTaskReq - A local shellcode injection task
	MsgTaskReq

	// MsgRemoteTaskReq - Remote thread injection task
	MsgRemoteTaskReq

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

	// MsgShellReq - Starts an interactive shell
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
	// MsgRunAs - Run process as user
	MsgRunAs
	// MsgRevToSelf - Revert to self
	MsgRevToSelf
	// MsgInvokeGetSystemReq - Elevate as SYSTEM user
	MsgInvokeGetSystemReq
	// MsgGetSystem - Response to getsystem request
	MsgGetSystem
	// MsgElevateReq - Request to run a new sliver session in an elevated context
	MsgElevateReq
	//MsgElevate - Response to the elevation request
	MsgElevate
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
	// MsgTerminate - Kill a remote process
	MsgTerminate

	// MsgScreenshotReq - Request to take a screenshot
	MsgScreenshotReq

	// MsgScreenshot - Response with the screenshots
	MsgScreenshot

	// MsgNetstatReq - Netstat request
	MsgNetstatReq
)

// MsgNumber - Get a message number of type
func MsgNumber(request proto.Message) uint32 {
	switch request.(type) {

	case *Register:
		return MsgRegister

	case *TaskReq:
		return MsgTaskReq

	case *RemoteTaskReq:
		return MsgRemoteTaskReq

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

	case *TunnelData:
		return MsgTunnelData
	case *TunnelClose:
		return MsgTunnelClose

	case *ProcessDumpReq:
		return MsgProcessDumpReq
	case *ProcessDump:
		return MsgProcessDump

	case *ImpersonateReq:
		return MsgImpersonateReq
	case *Impersonate:
		return MsgImpersonate

	case *RunAs:
		return MsgRunAs

	case *RevToSelf:
		return MsgRevToSelf

	case *InvokeGetSystemReq:
		return MsgInvokeGetSystemReq
	case *GetSystem:
		return MsgGetSystem

	case *ElevateReq:
		return MsgElevateReq
	case *Elevate:
		return MsgElevate

	case *ExecuteAssemblyReq:
		return MsgExecuteAssemblyReq
	case *ExecuteAssembly:
		return MsgExecuteAssembly

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

	case *Terminate:
		return MsgTerminate

	case *ScreenshotReq:
		return MsgScreenshotReq
	case *Screenshot:
		return MsgScreenshot

	case *NetstatReq:
		return MsgNetstatReq

	}
	return uint32(0)
}
