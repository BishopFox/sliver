package sliverpb

// Message Name Constants

const (

	// MsgRegister - Initial message from sliver with metadata
	MsgRegister = uint32(1 + iota)

	// MsgTask - A local shellcode injection task
	MsgTask

	// MsgRemoteTask - Remote thread injection task
	MsgRemoteTask

	// MsgPsListReq - Request to get process information
	MsgPsListReq
	// MsgPsList - Process information (resp to MsgPsReq)
	MsgPsList

	// MsgPing - Confirm connection is open used as req/resp
	MsgPing

	// MsgKill - Kill request to the sliver process
	MsgKill

	// MsgDirListReq - Request a directory listing from the remote system
	MsgDirListReq
	// MsgDirList - Directory listing (resp to MsgDirListReq)
	MsgDirList

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
	// MsgCd - Confirms the success/failure of the `cd` request (resp to MsgCdReq)
	MsgCd

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
	// MsgProcessDumpReq - Request to create a process dump
	MsgProcessDumpReq
	// MsgProcessDump - Dump of process)
	MsgProcessDump
	// MsgExecuteAssemblyReq - Request to load and execute a .NET assembly
	MsgExecuteAssemblyReq
	// MsgExecuteAssembly - Output of the assembly execution
	MsgExecuteAssembly
)
