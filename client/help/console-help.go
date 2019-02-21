package help

import (
	"bytes"
	consts "sliver/client/constants"
	"text/template"
)

const (
	// ANSI Colors
	normal    = "\033[0m"
	black     = "\033[30m"
	red       = "\033[31m"
	green     = "\033[32m"
	orange    = "\033[33m"
	blue      = "\033[34m"
	purple    = "\033[35m"
	cyan      = "\033[36m"
	gray      = "\033[37m"
	bold      = "\033[1m"
	clearln   = "\r\x1b[2K"
	upN       = "\033[%dA"
	downN     = "\033[%dB"
	underline = "\033[4m"
)

var (
	cmdHelp = map[string]string{
		consts.JobsStr:       jobsHelp,
		consts.SessionsStr:   sessionsHelp,
		consts.BackgroundStr: backgroundHelp,
		consts.InfoStr:       infoHelp,
		consts.UseStr:        useHelp,
		consts.GenerateStr:   generateHelp,

		consts.MsfStr:             msfHelp,
		consts.InjectStr:          injectHelp,
		consts.PsStr:              psHelp,
		consts.PingStr:            pingHelp,
		consts.KillStr:            killHelp,
		consts.LsStr:              lsHelp,
		consts.CdStr:              cdHelp,
		consts.CatStr:             catHelp,
		consts.DownloadStr:        downloadHelp,
		consts.UploadStr:          uploadHelp,
		consts.MkdirStr:           mkdirHelp,
		consts.RmStr:              rmHelp,
		consts.ProcdumpStr:        procdumpHelp,
		consts.ExecuteAssemblyStr: executeAssemblyHelp,
	}

	jobsHelp = `[[.Bold]]Command:[[.Normal]] jobs <options>
	[[.Bold]]About:[[.Normal]] Manange jobs/listeners.`

	sessionsHelp = `[[.Bold]]Command:[[.Normal]] sessions <options>
[[.Bold]]About:[[.Normal]] List files on remote system.`

	backgroundHelp = `[[.Bold]]Command:[[.Normal]] background
[[.Bold]]About:[[.Normal]] Background the active sliver.`

	infoHelp = `[[.Bold]]Command:[[.Normal]] info <sliver name>
[[.Bold]]About:[[.Normal]] Get information about a sliver by name, or for the active sliver.`

	useHelp = `[[.Bold]]Command:[[.Normal]] use [sliver name]
[[.Bold]]About:[[.Normal]] Switch the active sliver, a valid name must be provided (see sessions).`

	generateHelp = `[[.Bold]]Command:[[.Normal]] generate <options>
[[.Bold]]About:[[.Normal]] Generate a new sliver binary.`

	msfHelp = `[[.Bold]]Command:[[.Normal]] msf [--lhost] <options>
[[.Bold]]About:[[.Normal]] Execute a metasploit payload in the current process.`

	injectHelp = `[[.Bold]]Command:[[.Normal]] inject [--pid] [--lhost] <options>
[[.Bold]]About:[[.Normal]] Execute a metasploit payload in a remote process.`

	psHelp = `[[.Bold]]Command:[[.Normal]] ps <options>
[[.Bold]]About:[[.Normal]] List processes on remote system.`

	pingHelp = `[[.Bold]]Command:[[.Normal]] ping <sliver name>
[[.Bold]]About:[[.Normal]] Ping sliver by name or the active sliver.`

	killHelp = `[[.Bold]]Command:[[.Normal]] kill <sliver name>
[[.Bold]]About:[[.Normal]] Kill a remote sliver process (does not delete file).`

	lsHelp = `[[.Bold]]Command:[[.Normal]] ls
[[.Bold]]About:[[.Normal]] List remote files in current directory.`

	cdHelp = `[[.Bold]]Command:[[.Normal]] cd [dir]
[[.Bold]]About:[[.Normal]] Change working directory.`

	pwdHelp = `[[.Bold]]Command:[[.Normal]] pwd
[[.Bold]]About:[[.Normal]] Print working directory.`

	mkdirHelp = `[[.Bold]]Command:[[.Normal]] mkdir <remote path> 
[[.Bold]]About:[[.Normal]] Create a remote directory.`

	rmHelp = `[[.Bold]]Command:[[.Normal]] rm <remote file> 
[[.Bold]]About:[[.Normal]] Delete a remote file or directory.`

	catHelp = `[[.Bold]]Command:[[.Normal]] cat <remote file> 
[[.Bold]]About:[[.Normal]] Cat a remote file to stdout.`

	downloadHelp = `[[.Bold]]Command:[[.Normal]] download <remote src> <local dst>
[[.Bold]]About:[[.Normal]] Download a file from the remote system.`

	uploadHelp = `[[.Bold]]Command:[[.Normal]] upload <local src> <remote dst>
[[.Bold]]About:[[.Normal]] Upload a file to the remote system.`

	procdumpHelp = `[[.Bold]]Command:[[.Normal]] procdump <pid>
[[.Bold]]About:[[.Normal]] Dumps the process memory given a process identifier (pid)`

	impersonateHelp = `[[.Bold]]Command:[[.Normal]] impersonate [--username] [--process] [--args]
[[.Bold]]About:[[.Normal]] Run a new process in the context of the designated user`

	elevateHelp = `[[.Bold]]Command:[[.Normal]] elevate
[[.Bold]]About:[[.Normal]] Spawn a new sliver session as an elevated process (UAC bypass)`
	executeAssemblyHelp = `[[.Bold]]Command:[[.Normal]] execute-assembly <path to assembly> [arguments]
[[.Bold]]About:[[.Normal]] Executes the .NET assembly in a child process.`
)

// GetHelpFor - Get help string for a command
func GetHelpFor(cmdName string) string {
	if 0 < len(cmdName) {
		if helpTmpl, ok := cmdHelp[cmdName]; ok {
			outputBuf := bytes.NewBufferString("")
			tmpl, _ := template.New("help").Delims("[[", "]]").Parse(helpTmpl)
			tmpl.Execute(outputBuf, struct {
				Normal    string
				Bold      string
				Underline string
			}{
				Normal:    normal,
				Bold:      bold,
				Underline: underline,
			})

			return outputBuf.String()
		}
	}
	return ""
}
