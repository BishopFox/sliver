package sliverpb

// Message Name Constants

const (

	// MsgRegister - Initial message from sliver with metadata
	MsgRegister = uint32(1)

	// MsgTask - A local shellcode injection task
	MsgTask = uint32(2)

	// MsgRemoteTask - Remote thread injection task
	MsgRemoteTask = uint32(3)

	// MsgPsListReq - Request to get process information
	MsgPsListReq = uint32(4)
	// MsgPsList - Process information (resp to MsgPsReq)
	MsgPsList = uint32(5)

	// MsgPing - Confirm connection is open used as req/resp
	MsgPing = uint32(6)

	// MsgKill - Kill request to the sliver process
	MsgKill = uint32(7)

	// MsgDirListReq - Request a directory listing from the remote system
	MsgDirListReq = uint32(8)
	// MsgDirList - Directory listing (resp to MsgDirListReq)
	MsgDirList = uint32(9)

	// MsgDownloadReq - Request to download a file from the remote system
	MsgDownloadReq = uint32(10)
	// MsgDownload - File contents for download (resp to DownloadReq)
	MsgDownload = uint32(11)

	// MsgUploadReq - Upload a file to the remote file system
	MsgUploadReq = uint32(12)
	// MsgUpload - Confirms the success/failure of the file upload (resp to MsgUploadReq)
	MsgUpload = uint32(13)

	// MsgCdReq - Request a change directory on the remote system
	MsgCdReq = uint32(14)
	// MsgCd - Confirms the success/failure of the `cd` request (resp to MsgCdReq)
	MsgCd = uint32(15)

	// MsgPwdReq - A request to get the CWD from the remote process
	MsgPwdReq = uint32(16)
	// MsgPwd - The CWD of the remote process (resp to MsgPwdReq)
	MsgPwd = uint32(17)

	// MsgRmReq - Request to delete remote file
	MsgRmReq = uint32(18)
	// MsgRm - Confirms the success/failure of delete request (resp to MsgRmReq)
	MsgRm = uint32(19)

	// MsgMkdirReq - Request to create a directory on the remote system
	MsgMkdirReq = uint32(20)
	// MsgMkdir - Confirms the success/failure of the mkdir request (resp to MsgMkdirReq)
	MsgMkdir = uint32(21)
	// MsgProcessDumpReq - Request to create a process dump
	MsgProcessDumpReq = uint32(22)
	// MsgProcessDump - Dump of process)
	MsgProcessDump = uint32(23)

	// MsgShellReq - Starts an interactive shell
	MsgShellReq = uint32(24)
	// MsgShellData - Data for stdin/out/err
	MsgShellData = uint32(25)
)
