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
    ----------------------------------------------------------------------

    Not sure I ever really appreciated Go's `iota` until just now ...

*/

export class SliverPB {
    // MsgRegister - Initial message from sliver with metadata
    static readonly MsgRegister = 1;

    // MsgTask - A local shellcode injection task
    static readonly MsgTask = 2;

    // MsgRemoteTask - Remote thread injection task
    static readonly MsgRemoteTask = 3;

    // MsgPing - Confirm connection is open used as req/resp
    static readonly MsgPing = 4;

    // MsgKill - Kill request to the sliver process
    static readonly MsgKill = 5;

    // MsgLsReq - Request a directory listing from the remote system
    static readonly MsgLsReq = 6;
    // MsgLs - Directory listing (resp to MsgDirListReq)
    static readonly MsgLs = 7;

    // MsgDownloadReq - Request to download a file from the remote system
    static readonly MsgDownloadReq = 8;
    // MsgDownload - File contents for download (resp to DownloadReq)
    static readonly MsgDownload = 9;

    // MsgUploadReq - Upload a file to the remote file system
    static readonly MsgUploadReq = 10;
    // MsgUpload - Confirms the success/failure of the file upload (resp to MsgUploadReq)
    static readonly MsgUpload = 11;

    // MsgCdReq - Request a change directory on the remote system
    static readonly MsgCdReq = 12;
    // MsgCd - Confirms the success/failure of the `cd` request (resp to MsgCdReq)
    static readonly MsgCd = 13;

    // MsgPwdReq - A request to get the CWD from the remote process
    static readonly MsgPwdReq = 14;
    // MsgPwd - The CWD of the remote process (resp to MsgPwdReq)
    static readonly MsgPwd = 15;

    // MsgRmReq - Request to delete remote file
    static readonly MsgRmReq = 16;
    // MsgRm - Confirms the success/failure of delete request (resp to MsgRmReq)
    static readonly MsgRm = 17;

    // MsgMkdirReq - Request to create a directory on the remote system
    static readonly MsgMkdirReq = 18;
    // MsgMkdir - Confirms the success/failure of the mkdir request (resp to MsgMkdirReq)
    static readonly MsgMkdir = 19;

    // MsgPsReq - List processes req
    static readonly MsgPsReq = 20;
    // MsgPs - List processes resp
    static readonly MsgPs = 21;

    // MsgShellReq - Starts an interactive shell
    static readonly MsgShellReq = 22;
    // MsgShell - Response on starting shell
    static readonly MsgShell = 23;

    // MsgTunnelData - Data for duplex tunnels
    static readonly MsgTunnelData = 24;
    // MsgTunnelClose - Close a duplex tunnel
    static readonly MsgTunnelClose = 25;

    // MsgProcessDumpReq - Request to create a process dump
    static readonly MsgProcessDumpReq = 26;
    // MsgProcessDump - Dump of process)
    static readonly MsgProcessDump = 27;
    // MsgImpersonateReq - Request for process impersonatio
    static readonly MsgImpersonateReq = 28;
    // MsgImpersonate - Output of the impersonation command
    static readonly MsgImpersonate = 29;
    // MsgGetSystemReq - Elevate as SYSTEM user
    static readonly MsgGetSystemReq = 30;
    // MsgGetSystem - Response to getsystem request
    static readonly MsgGetSystem = 31;
    // MsgElevateReq - Request to run a new sliver session in an elevated context
    static readonly MsgElevateReq = 32;
    // MsgElevate - Response to the elevation request
    static readonly MsgElevate = 33;
    // MsgExecuteAssemblyReq - Request to load and execute a .NET assembly
    static readonly MsgExecuteAssemblyReq = 34;
    // MsgExecuteAssembly - Output of the assembly execution
    static readonly MsgExecuteAssembly = 35;
    // MsgMigrateReq - Spawn a new sliver in a designated process
    static readonly MsgMigrateReq = 36;

    // MsgIfconfigReq - Ifconfig (network interface config) request
    static readonly MsgIfconfigReq = 37;
}

export class ClientPB {

    static readonly clientOffset = 10000; // To avoid duplications with sliverpb constants

    // MsgEvent - Initial message from sliver with metadata
    static readonly MsgEvent = ClientPB.clientOffset + 1;

    // MsgSessions - Sessions message
    static readonly MsgSessions = ClientPB.clientOffset + 2;

    // MsgPlayers - List active players
    static readonly MsgPlayers = ClientPB.clientOffset + 3;

    // MsgJobs - Jobs message
    static readonly MsgJobs = ClientPB.clientOffset + 4;
    static readonly MsgJobKill = ClientPB.clientOffset + 5;
    // MsgTcp - TCP message
    static readonly MsgTcp = ClientPB.clientOffset + 6;
    // MsgMtls - MTLS message
    static readonly MsgMtls = ClientPB.clientOffset + 7;

    // MsgDns - DNS message
    static readonly MsgDns = ClientPB.clientOffset + 8;

    // MsgHttp - HTTP(S) listener  message
    static readonly MsgHttp = ClientPB.clientOffset + 9;
    // MsgHttps - HTTP(S) listener  message
    static readonly MsgHttps = ClientPB.clientOffset + 10;

    // MsgMsf - MSF message
    static readonly MsgMsf = ClientPB.clientOffset + 11;

    // MsgMsfInject - MSF injection message
    static readonly MsgMsfInject = ClientPB.clientOffset + 12;

    // MsgTunnelCreate - Create tunnel message
    static readonly MsgTunnelCreate = ClientPB.clientOffset + 13;
    static readonly MsgTunnelClose = ClientPB.clientOffset + 14;

    // MsgGenerate - Generate message
    static readonly MsgGenerate = ClientPB.clientOffset + 15;
    static readonly MsgNewProfile = ClientPB.clientOffset + 16;
    static readonly MsgProfiles = ClientPB.clientOffset + 17;

    static readonly MsgTask = ClientPB.clientOffset + 18;
    static readonly MsgMigrate = ClientPB.clientOffset + 19;
    static readonly MsgGetSystemReq = ClientPB.clientOffset + 20;
    static readonly MsgEggReq = ClientPB.clientOffset + 21;
    static readonly MsgExecuteAssemblyReq = ClientPB.clientOffset + 22;

    static readonly MsgRegenerate = ClientPB.clientOffset + 23;

    static readonly MsgListSliverBuilds = ClientPB.clientOffset + 24;
    static readonly MsgListCanaries = ClientPB.clientOffset + 25;

    // Website related messages
    static readonly MsgWebsiteList = ClientPB.clientOffset + 26;
    static readonly MsgWebsiteAddContent = ClientPB.clientOffset + 27;
    static readonly MsgWebsiteRemoveContent = ClientPB.clientOffset + 28;
}
