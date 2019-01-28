package sliverpb

// Message Name Constants

const (

	// MsgRegister - Initial message from sliver with metadata
	MsgRegister = "register"

	// MsgTask - A local shellcode injection task
	MsgTask = "task"

	// MsgRemoteTask - Remote thread injection task
	MsgRemoteTask = "remoteTask"

	// MsgPsListReq - Request to get process information
	MsgPsListReq = "psListReq"
	// MsgPsList - Process information (resp to MsgPsReq)
	MsgPsList = "psList"

	// MsgPing - Confirm connection is open used as req/resp
	MsgPing = "ping"

	// MsgKill - Kill request to the sliver process
	MsgKill = "kill"

	// MsgDirListReq - Request a directory listing from the remote system
	MsgDirListReq = "dirListReq"
	// MsgDirList - Directory listing (resp to MsgDirListReq)
	MsgDirList = "dirList"

	// MsgDownloadReq - Request to download a file from the remote system
	MsgDownloadReq = "downloadReq"
	// MsgDownload - File contents for download (resp to DownloadReq)
	MsgDownload = "download"

	// MsgUploadReq - Upload a file to the remote file system
	MsgUploadReq = "uploadReq"
	// MsgUpload - Confirms the success/failure of the file upload (resp to MsgUploadReq)
	MsgUpload = "upload"

	// MsgCdReq - Request a change directory on the remote system
	MsgCdReq = "cdReq"
	// MsgCd - Confirms the success/failure of the `cd` request (resp to MsgCdReq)
	MsgCd = "cd"

	// MsgPwdReq - A request to get the CWD from the remote process
	MsgPwdReq = "pwdReq"
	// MsgPwd - The CWD of the remote process (resp to MsgPwdReq)
	MsgPwd = "pwd"

	// MsgRmReq - Request to delete remote file
	MsgRmReq = "rmReq"
	// MsgRm - Confirms the success/failure of delete request (resp to MsgRmReq)
	MsgRm = "rm"

	// MsgMkdirReq - Request to create a directory on the remote system
	MsgMkdirReq = "mkdirReq"
	// MsgMkdir - Confirms the success/failure of the mkdir request (resp to MsgMkdirReq)
	MsgMkdir = "mkdir"
	// MsgProcessDumpReq - Request to create a process dump
	MsgProcessDumpReq = "dumpReq"
	// MsgProcessDump - Dump of process)
	MsgProcessDump = "dump"
)
