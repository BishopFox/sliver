package console

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox
	Copyright (C) 2019 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
*/

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bishopfox/sliver/client/assets"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/client/theme"
	"github.com/bishopfox/sliver/client/version"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/util"
	"github.com/gofrs/uuid"
	"github.com/kballard/go-shellquote"
	"github.com/reeflective/console"
	"github.com/reeflective/readline"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

const (
	defaultTimeout = 60
)

const (
	// Terminal control sequences (not "styling").
	// Terminal 控制序列（不是__PH0__）。
	Clearln = "\r\x1b[2K"
	UpN     = "\033[%dA"
	DownN   = "\033[%dB"
)

// Observer - A function to call when the sessions changes.
// Observer - A 在会话 changes. 时调用的函数
type (
	Observer           func(*clientpb.Session, *clientpb.Beacon)
	BeaconTaskCallback func(*clientpb.BeaconTask)
)

type SliverClient struct {
	App                      *console.Console      //控制台对象
	Rpc                      rpcpb.SliverRPCClient //rpc对象
	ActiveTarget             *ActiveTarget         //当前活跃目标
	EventListeners           *sync.Map
	BeaconTaskCallbacks      map[string]BeaconTaskCallback
	BeaconTaskCallbacksMutex *sync.Mutex
	Settings                 *assets.ClientSettings
	IsServer                 bool
	IsCLI                    bool
	serverCmds               console.Commands
	sliverCmds               console.Commands

	jsonHandler      slog.Handler
	printf           func(format string, args ...any) (int, error)
	stdoutPipeWriter *os.File
	stdoutPipeDone   chan struct{}
	stdoutPipeOnce   sync.Once

	connMu                 sync.Mutex
	grpcConn               *grpc.ClientConn
	connDetails            *ConnectionDetails
	connCancel             context.CancelFunc
	connWg                 *sync.WaitGroup
	connectionHooksApplied bool
	logCommandHookApplied  bool

	// These writers are always safe to use (never error); the underlying stream
	// These writers 始终可以安全使用（不会出错）；底层流
	// can be swapped on server switch to avoid breaking io.MultiWriter/io.Copy.
	// 可以在服务器交换机上交换以避免破坏 io.MultiWriter/io.Copy.
	jsonRemoteWriter      *optionalRemoteWriter
	asciicastRemoteWriter *optionalRemoteWriter
	jsonRemoteStream      rpcpb.SliverRPC_ClientLogClient
	asciicastRemoteStream rpcpb.SliverRPC_ClientLogClient
}

// NewConsole creates the sliver client (and console), creating menus and prompts.
// NewConsole 创建 sliver 客户端（和控制台），创建菜单和 prompts.
// The returned console does neither have commands nor a working RPC connection yet,
// The 返回的控制台既没有命令，也没有有效的 RPC 连接，
// thus has not started monitoring any server events, or started the application.
// 因此尚未开始监视任何服务器事件，或启动 application.
func NewConsole(isServer bool) *SliverClient {
	assets.Setup(false, false)
	_ = theme.LoadAndSetCurrentTheme()
	ApplyTheme(theme.Current())
	settings, _ := assets.LoadSettings()

	con := &SliverClient{
		App: console.New("sliver"),
		ActiveTarget: &ActiveTarget{
			observers:  map[int]Observer{},
			observerID: 0,
		},
		EventListeners:           &sync.Map{},
		BeaconTaskCallbacks:      map[string]BeaconTaskCallback{},
		BeaconTaskCallbacksMutex: &sync.Mutex{},
		IsServer:                 isServer,
		Settings:                 settings,
		connWg:                   &sync.WaitGroup{},
	}
	// Ensure logging never panics even if console logs are disabled.
	// 即使控制台日志是 disabled.，Ensure 日志记录也不会出现恐慌
	con.jsonHandler = slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})

	// The active target needs access to the console
	// The 活动目标需要访问控制台
	// to automatically switch between command menus.
	// 在命令 menus. 之间自动切换
	con.ActiveTarget.con = con

	// Readline-shell (edition) settings
	// Readline__PH0__（版本）设置
	if settings.VimMode {
		con.App.Shell().Config.Set("editing-mode", "vi")
	}

	// Global console settings
	// Global 控制台设置
	con.App.NewlineBefore = true
	con.App.NewlineAfter = true

	// Server menu.
	server := con.App.Menu(consts.ServerMenu)
	server.Short = "Server commands"
	server.Prompt().Primary = con.GetPrompt
	server.AddInterrupt(readline.ErrInterrupt, con.exitConsole) // Ctrl-C
	server.AddInterrupt(readline.ErrInterrupt, con.exitConsole) // Ctrl__PH0__

	server.AddHistorySourceFile("server history", filepath.Join(assets.GetRootAppDir(), "history"))

	// Implant menu.
	sliver := con.App.NewMenu(consts.ImplantMenu)
	sliver.Short = "Implant commands"
	sliver.Prompt().Primary = con.GetPrompt
	sliver.AddInterrupt(io.EOF, con.exitImplantMenu) // Ctrl-D
	sliver.AddInterrupt(io.EOF, con.exitImplantMenu) // Ctrl__PH0__

	con.App.SetPrintLogo(func(_ *console.Console) {
		con.PrintLogo()
	})

	return con
}

// Init requires a working RPC connection to the sliver server, and 2 different sets of commands.
// Init 需要与 sliver 服务器的工作 RPC 连接，以及 2 组不同的 commands.
// If run is true, the console application is started, making this call blocking. Otherwise, commands and
// If run 为 true，控制台应用程序启动，调用 blocking. Otherwise，命令和
// RPC connection are bound to the console (making the console ready to run), but the console does not start.
// RPC 连接绑定到控制台（使控制台准备好运行），但控制台不 start.
func StartClient(con *SliverClient, rpc rpcpb.SliverRPCClient, grpcConn *grpc.ClientConn, details *ConnectionDetails, serverCmds, sliverCmds console.Commands, run bool, rcScript string) error {
	con.IsCLI = !run
	con.serverCmds = serverCmds
	con.sliverCmds = sliverCmds

	// The console application needs to query the terminal for cursor positions
	// The 控制台应用程序需要查询终端的光标位置
	// when asynchronously printing logs (that is, when no command is running).
	// 异步打印日志时（即没有命令运行时）。
	// If ran from a system shell, however, those queries will block because
	// If 从系统 shell 运行，但是，这些查询将被阻止，因为
	// the system shell is in control of stdin. So just use the classic Printf.
	// 系统 shell 控制 stdin. So 只需使用经典的 Printf.
	if con.IsCLI {
		con.printf = fmt.Printf
	} else {
		con.printf = con.App.TransientPrintf
	}

	// Bind commands to the app
	// Bind 对应用程序的命令
	server := con.App.Menu(consts.ServerMenu)
	server.SetCommands(serverCmds)

	sliver := con.App.Menu(consts.ImplantMenu)
	sliver.SetCommands(sliverCmds)

	con.applyConnectionHooksOnce()

	// console logger
	// 控制台记录器
	if con.Settings.ConsoleLogs {
		// Classic logs
		// Classic 日志
		consoleLog := getConsoleLogFile()
		con.ensureJSONRemoteWriter()
		con.setupLogger(consoleLog, con.jsonRemoteWriter)
		defer consoleLog.Close()

		// Ascii cast sessions (complete terminal interface) are only useful
		// Ascii 投射会话（完整的终端界面）仅有用
		// for the interactive console. In CLI mode they would clobber stdout.
		// 对于交互式 console. In CLI 模式，他们会破坏 stdout.
		if !con.IsCLI {
			asciicastLog := getConsoleAsciicastFile()
			defer asciicastLog.Close()

			con.ensureAsciicastRemoteWriter()
			con.setupAsciicastRecord(asciicastLog, con.asciicastRemoteWriter)
		}
	}

	if err := con.SetConnection(rpc, grpcConn, details); err != nil {
		return err
	}

	if rcScript != "" {
		originalPrintf := con.printf
		con.printf = fmt.Printf
		con.runRCScript(serverCmds, sliverCmds, rcScript)
		con.printf = originalPrintf
	}

	if !con.IsCLI {
		return con.App.Start()
	}

	return nil
}

func (con *SliverClient) runRCScript(serverCmds, sliverCmds console.Commands, rcScript string) {
	scanner := bufio.NewScanner(strings.NewReader(rcScript))
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if err := con.runRCLine(serverCmds, sliverCmds, line); err != nil {
			con.PrintErrorf("rc line %d error: %s", lineNumber, err)
		}
	}

	if err := scanner.Err(); err != nil {
		con.PrintErrorf("rc script error: %s", err)
	}
}

func (con *SliverClient) runRCLine(serverCmds, sliverCmds console.Commands, line string) error {
	args, err := shellquote.Split(line)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	if con.serverCmds == nil {
		con.serverCmds = serverCmds
	}
	if con.sliverCmds == nil {
		con.sliverCmds = sliverCmds
	}

	menu := con.App.ActiveMenu()
	if menu != nil {
		// Reset per line to avoid stale roots when rc scripts switch menus.
		// 每行 Reset 以避免 rc 脚本切换 menus. 时出现陈旧根
		menu.Command = nil
	}

	for _, hook := range con.App.PreCmdRunLineHooks {
		args, err = hook(args)
		if err != nil {
			return fmt.Errorf("pre-run line error: %w", err)
		}
	}

	if len(args) == 0 {
		return nil
	}

	menu = con.App.ActiveMenu()
	if menu == nil {
		return fmt.Errorf("no active menu")
	}
	if menu.Command == nil {
		con.setMenuCommand(menu, args)
	}
	if menu.Command == nil {
		return fmt.Errorf("no commands available")
	}

	target, _, _ := menu.Command.Find(args)
	if err := menu.CheckIsAvailable(target); err != nil {
		return err
	}

	for _, hook := range con.App.PreCmdRunHooks {
		if err := hook(); err != nil {
			return fmt.Errorf("pre-run error: %w", err)
		}
	}

	menu.SetArgs(args)
	menu.SetContext(context.Background())

	if err := menu.Execute(); err != nil {
		return err
	}

	return nil
}

func (con *SliverClient) applyConnectionHooksOnce() {
	con.connMu.Lock()
	defer con.connMu.Unlock()

	if con.connectionHooksApplied {
		return
	}
	con.connectionHooksApplied = true

	con.App.PreCmdRunLineHooks = append(con.App.PreCmdRunLineHooks, con.allowServerRootCommands)
	if shell := con.App.Shell(); shell != nil && shell.Completer != nil {
		baseCompleter := shell.Completer
		shell.Completer = func(line []rune, cursor int) readline.Completions {
			con.prepareCompletion(line, cursor)
			return baseCompleter(line, cursor)
		}
	}
}

func (con *SliverClient) startEventLoop(ctx context.Context) {
	eventStream, err := con.Rpc.Events(ctx, &commonpb.Empty{})
	if err != nil {
		fmt.Printf("%s%s\n", Warn, err)
		return
	}
	for {
		event, err := eventStream.Recv()
		if err == io.EOF || event == nil {
			return
		}

		go con.triggerEventListeners(event)

		// Trigger event based on type
		// Trigger 基于类型的事件
		switch event.EventType {

		case consts.CanaryEvent:
			con.PrintEventErrorf("%s %s has been burned (DNS Canary)", StyleBold.Render("WARNING:"), event.Session.Name)
			sessions := con.GetSessionsByName(event.Session.Name)
			for _, session := range sessions {
				shortID := strings.Split(session.ID, "-")[0]
				con.PrintErrorf("\t🔥 Session %s is affected", shortID)
			}

		case consts.WatchtowerEvent:
			msg := string(event.Data)
			con.PrintEventErrorf("%s %s has been burned (seen on %s)", StyleBold.Render("WARNING:"), event.Session.Name, msg)
			sessions := con.GetSessionsByName(event.Session.Name)
			for _, session := range sessions {
				shortID := strings.Split(session.ID, "-")[0]
				con.PrintErrorf("\t🔥 Session %s is affected", shortID)
			}

		case consts.JoinedEvent:
			if con.Settings.UserConnect {
				con.PrintInfof("%s has joined the game", event.Client.Operator.Name)
			}
		case consts.LeftEvent:
			if con.Settings.UserConnect {
				con.PrintInfof("%s left the game", event.Client.Operator.Name)
			}

		case consts.JobStoppedEvent:
			job := event.Job
			con.PrintErrorf("Job #%d stopped (%s/%s)", job.ID, job.Protocol, job.Name)

		case consts.SessionOpenedEvent:
			session := event.Session
			currentTime := time.Now().Format(time.RFC1123)
			shortID := strings.Split(session.ID, "-")[0]
			con.PrintEventInfof("Session %s %s - %s (%s) - %s/%s - %v",
				shortID, session.Name, session.RemoteAddress, session.Hostname, session.OS, session.Arch, currentTime)

		case consts.SessionUpdateEvent:
			session := event.Session
			currentTime := time.Now().Format(time.RFC1123)
			shortID := strings.Split(session.ID, "-")[0]
			con.PrintInfof("Session %s has been updated - %v", shortID, currentTime)

		case consts.SessionClosedEvent:
			session := event.Session
			currentTime := time.Now().Format(time.RFC1123)
			shortID := strings.Split(session.ID, "-")[0]
			con.PrintEventErrorf("Lost session %s %s - %s (%s) - %s/%s - %v",
				shortID, session.Name, session.RemoteAddress, session.Hostname, session.OS, session.Arch, currentTime)
			activeSession := con.ActiveTarget.GetSession()
			core.GetTunnels().CloseForSession(session.ID)
			core.CloseCursedProcesses(session.ID)
			if activeSession != nil && activeSession.ID == session.ID {
				con.ActiveTarget.Set(nil, nil)
				con.PrintErrorf("Active session disconnected")
			}

		case consts.BeaconRegisteredEvent:
			beacon := &clientpb.Beacon{}
			proto.Unmarshal(event.Data, beacon)
			currentTime := time.Now().Format(time.RFC1123)
			shortID := strings.Split(beacon.ID, "-")[0]
			con.PrintEventInfof("Beacon %s %s - %s (%s) - %s/%s - %v",
				shortID, beacon.Name, beacon.RemoteAddress, beacon.Hostname, beacon.OS, beacon.Arch, currentTime)

		case consts.BeaconTaskResultEvent:
			con.triggerBeaconTaskCallback(event.Data)

		}

		con.triggerReactions(event)
	}
}

// CreateEventListener - creates a new event listener and returns its ID.
// CreateEventListener - 创建一个新事件 listener 并返回其 ID.
func (con *SliverClient) CreateEventListener() (string, <-chan *clientpb.Event) {
	listener := make(chan *clientpb.Event, 100)
	listenerID, _ := uuid.NewV4()
	con.EventListeners.Store(listenerID.String(), listener)
	return listenerID.String(), listener
}

// RemoveEventListener - removes an event listener given its id.
// RemoveEventListener - 删除事件 listener（给定其 id.）
func (con *SliverClient) RemoveEventListener(listenerID string) {
	value, ok := con.EventListeners.LoadAndDelete(listenerID)
	if ok {
		close(value.(chan *clientpb.Event))
	}
}

func (con *SliverClient) triggerEventListeners(event *clientpb.Event) {
	con.EventListeners.Range(func(key, value interface{}) bool {
		listener := value.(chan *clientpb.Event)
		listener <- event // Do not block while sending the event to the listener
		listener <- event // Do 在将事件发送到 listener 时不会阻塞
		return true
	})
}

func (con *SliverClient) triggerReactions(event *clientpb.Event) {
	reactions := core.Reactions.On(event.EventType)
	if len(reactions) == 0 {
		return
	}

	// We need some special handling for SessionOpenedEvent to
	// We 需要对 SessionOpenedEvent 进行一些特殊处理
	// set the new session as the active session
	// 将新的 session 设置为活动 session
	currentActiveSession, currentActiveBeacon := con.ActiveTarget.Get()
	defer func() {
		con.ActiveTarget.Set(currentActiveSession, currentActiveBeacon)
	}()

	switch event.EventType {
	case consts.SessionOpenedEvent:
		con.ActiveTarget.Set(nil, nil)

		con.ActiveTarget.Set(event.Session, nil)
	case consts.BeaconRegisteredEvent:
		con.ActiveTarget.Set(nil, nil)

		beacon := &clientpb.Beacon{}
		proto.Unmarshal(event.Data, beacon)
		con.ActiveTarget.Set(nil, beacon)
	}

	for _, reaction := range reactions {
		for _, line := range reaction.Commands {
			con.PrintInfof("%s '%s'", StyleBold.Render("Execute reaction:"), line)
			err := con.App.ActiveMenu().RunCommandLine(context.Background(), line)
			if err != nil {
				con.PrintErrorf("Reaction command error: %s\n", err)
			}
		}
	}
}

// triggerBeaconTaskCallback - Triggers the callback for a beacon task.
// triggerBeaconTaskCallback - Triggers callback 对应 beacon task.
func (con *SliverClient) triggerBeaconTaskCallback(data []byte) {
	task := &clientpb.BeaconTask{}
	err := proto.Unmarshal(data, task)
	if err != nil {
		con.PrintErrorf("\rCould not unmarshal beacon task: %s", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	beacon, _ := con.Rpc.GetBeacon(ctx, &clientpb.Beacon{ID: task.BeaconID})

	// If the callback is not in our map then we don't do anything, the beacon task
	// If callback 不在我们的地图中，那么我们什么也不做， beacon task
	// was either issued by another operator in multiplayer mode or the client process
	// 是由多人模式中的另一个 operator 或客户端进程发出的
	// was restarted between the time the task was created and when the server got the result
	// 在创建 task 和服务器获得结果之间重新启动
	con.BeaconTaskCallbacksMutex.Lock()
	defer con.BeaconTaskCallbacksMutex.Unlock()
	if callback, ok := con.BeaconTaskCallbacks[task.ID]; ok {
		if con.Settings.BeaconAutoResults {
			if beacon != nil {
				con.PrintSuccessf("%s completed task %s", beacon.Name, strings.Split(task.ID, "-")[0])
			}
			task_content, err := con.Rpc.GetBeaconTaskContent(ctx, &clientpb.BeaconTask{
				ID: task.ID,
			})
			con.Printf(Clearln + "\r")
			if err == nil {
				callback(task_content)
			} else {
				con.PrintErrorf("Could not get beacon task content: %s", err)
			}
		}
		delete(con.BeaconTaskCallbacks, task.ID)
	}
}

func (con *SliverClient) AddBeaconCallback(taskID string, callback BeaconTaskCallback) {
	con.BeaconTaskCallbacksMutex.Lock()
	defer con.BeaconTaskCallbacksMutex.Unlock()
	con.BeaconTaskCallbacks[taskID] = callback
}

func (con *SliverClient) GetPrompt() string {
	promptStyle := assets.PromptStyleHost
	if con.Settings != nil {
		promptStyle = assets.NormalizePromptStyle(con.Settings.PromptStyle)
	}

	if promptStyle == assets.PromptStyleCustom {
		src := assets.DefaultPromptTemplate
		if con.Settings != nil && strings.TrimSpace(con.Settings.PromptTemplate) != "" {
			src = con.Settings.PromptTemplate
		}
		out, err := renderPromptTemplate(con, src)
		if err != nil {
			return Clearln + " > "
		}
		if strings.TrimSpace(out) == "" {
			return Clearln + " > "
		}
		return Clearln + out
	}

	prompt := StyleUnderline.Render("sliver")
	if con.IsServer {
		if promptStyle != assets.PromptStyleBasic {
			prompt = StyleBold.Render("[server]") + " " + prompt
		}
	} else if promptStyle != assets.PromptStyleBasic {
		// sliver-client only: optionally include operator/host context.
		// 仅 sliver__PH0__：可选择包括 operator/host context.
		if details, _, ok := con.CurrentConnection(); ok && details != nil && details.Config != nil {
			operator := strings.TrimSpace(details.Config.Operator)
			host := strings.TrimSpace(details.Config.LHost)

			prefix := ""
			switch promptStyle {
			case assets.PromptStyleOperatorHost:
				if operator != "" && host != "" {
					prefix = operator + "@" + host
				} else if operator != "" {
					prefix = operator
				} else if host != "" {
					prefix = host
				}
			case assets.PromptStyleHost:
				if host != "" {
					prefix = host
				}
			}

			if prefix != "" {
				prompt = StyleBoldPrimary.Render("["+prefix+"]") + " " + prompt
			}
		}
	}

	if session := con.ActiveTarget.GetSession(); session != nil {
		prompt += StyleBoldRed.Render(" (" + session.Name + ")")
	} else if beacon := con.ActiveTarget.GetBeacon(); beacon != nil {
		prompt += StyleBoldBlue.Render(" (" + beacon.Name + ")")
	}
	prompt += " > "
	return Clearln + prompt
}

func (con *SliverClient) PrintLogo() {
	// Always show a logo even if the server connection is transiently unavailable.
	// 即使服务器连接是暂时的 Always 也会显示徽标 unavailable.
	logo := asciiLogos[util.Intn(len(asciiLogos))]
	fmt.Println(strings.ReplaceAll(logo.Render(), "\n", "\r\n"))
	fmt.Println("All hackers gain " + abilities[util.Intn(len(abilities))] + "\r")

	if con.Rpc == nil {
		fmt.Println(Warn + "Not connected to a server\r")
		fmt.Printf("%sClient %s\r\n", Info, version.FullVersion())
		fmt.Println(Info + "Welcome to the sliver shell, please type 'help' for options\r")
		fmt.Println("")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	serverVer, err := con.Rpc.GetVersion(ctx, &commonpb.Empty{})
	if err != nil {
		fmt.Printf("%sCould not query server version: %s\r\n", Warn, err)
		fmt.Printf("%sClient %s\r\n", Info, version.FullVersion())
		fmt.Println(Info + "Welcome to the sliver shell, please type 'help' for options\r")
		fmt.Println("")
		con.CheckLastUpdate()
		return
	}
	dirty := ""
	if serverVer.Dirty {
		dirty = " - " + StyleBold.Render("Dirty")
	}
	serverSemVer := fmt.Sprintf("%d.%d.%d", serverVer.Major, serverVer.Minor, serverVer.Patch)
	fmt.Printf("%sServer v%s - %s%s\r\n", Info, serverSemVer, serverVer.Commit, dirty)
	if version.GitCommit != serverVer.Commit {
		fmt.Printf("%sClient %s\r\n", Info, version.FullVersion())
	}
	fmt.Println(Info + "Welcome to the sliver shell, please type 'help' for options\r")
	if serverVer.Major != int32(version.SemanticVersion()[0]) {
		fmt.Print(Warn + "Warning: Client and server may be running incompatible versions.\r\n")
	}
	fmt.Println("")
	con.CheckLastUpdate()
}

func (con *SliverClient) CheckLastUpdate() {
	now := time.Now()
	lastUpdate := getLastUpdateCheck()
	compiledAt, err := version.Compiled()
	if err != nil {
		log.Printf("Failed to parse compiled at timestamp %s", err)
		return
	}

	day := 24 * time.Hour
	if compiledAt.Add(30 * day).Before(now) {
		if lastUpdate == nil || lastUpdate.Add(30*day).Before(now) {
			con.Printf("%sCheck for updates with the 'update' command\n\n", Info)
		}
	}
}

func getLastUpdateCheck() *time.Time {
	appDir := assets.GetRootAppDir()
	lastUpdateCheckPath := filepath.Join(appDir, consts.LastUpdateCheckFileName)
	data, err := os.ReadFile(lastUpdateCheckPath)
	if err != nil {
		log.Printf("Failed to read last update check %s", err)
		return nil
	}
	unixTime, err := strconv.Atoi(string(data))
	if err != nil {
		log.Printf("Failed to parse last update check %s", err)
		return nil
	}
	lastUpdate := time.Unix(int64(unixTime), 0)
	return &lastUpdate
}

func (con *SliverClient) GetSession(arg string) *clientpb.Session {
	sessions, err := con.Rpc.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintWarnf("%s", err)
		return nil
	}
	for _, session := range sessions.GetSessions() {
		if session.Name == arg || strings.HasPrefix(session.ID, arg) {
			return session
		}
	}
	return nil
}

// GetSessionsByName - Return all sessions for an Implant by name.
// GetSessionsByName - Return name. 的 Implant 的所有课程
func (con *SliverClient) GetSessionsByName(name string) []*clientpb.Session {
	sessions, err := con.Rpc.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf("%s%s\n", Warn, err)
		return nil
	}
	matched := []*clientpb.Session{}
	for _, session := range sessions.GetSessions() {
		if session.Name == name {
			matched = append(matched, session)
		}
	}
	return matched
}

// GetActiveSessionConfig - Get the active sessions's config
// GetActiveSessionConfig - Get 活动会话的配置
// TODO: Switch to query config based on ConfigID.
// TODO: Switch 根据 ConfigID. 查询配置
func (con *SliverClient) GetActiveSessionConfig() *clientpb.ImplantConfig {
	session := con.ActiveTarget.GetSession()
	if session == nil {
		return nil
	}
	c2s := []*clientpb.ImplantC2{}
	c2s = append(c2s, &clientpb.ImplantC2{
		URL:      session.GetActiveC2(),
		Priority: uint32(0),
	})
	config := &clientpb.ImplantConfig{
		ID:      session.ID,
		GOOS:    session.GetOS(),
		GOARCH:  session.GetArch(),
		Debug:   true,
		Evasion: session.GetEvasion(),

		MaxConnectionErrors: uint32(1000),
		ReconnectInterval:   int64(60),
		Format:              clientpb.OutputFormat_SHELLCODE,
		IsSharedLib:         true,
		C2:                  c2s,
	}
	/* If this config will be used to build an implant,
 If 此配置将用于构建 implant，
	we need to make sure to include the correct transport
	我们需要确保包含正确的交通工具
	for the build 
	用于构建*/
	switch session.Transport {
	case "mtls":
		config.IncludeMTLS = true
	case "http(s)":
		config.IncludeHTTP = true
	case "dns":
		config.IncludeDNS = true
	case "wg":
		config.IncludeWG = true
	case "namedpipe":
		config.IncludeNamePipe = true
	case "tcppivot":
		config.IncludeTCP = true
	}
	return config
}

func (con *SliverClient) GetActiveBeaconConfig() *clientpb.ImplantConfig {
	beacon := con.ActiveTarget.GetBeacon()
	if beacon == nil {
		return nil
	}

	c2s := []*clientpb.ImplantC2{}
	c2s = append(c2s, &clientpb.ImplantC2{
		URL:      beacon.ActiveC2,
		Priority: uint32(0),
	})

	config := &clientpb.ImplantConfig{
		ID:                  beacon.ID,
		GOOS:                beacon.OS,
		GOARCH:              beacon.Arch,
		Debug:               false,
		IsBeacon:            true,
		BeaconInterval:      beacon.Interval,
		BeaconJitter:        beacon.Jitter,
		Evasion:             beacon.Evasion,
		MaxConnectionErrors: uint32(1000),
		ReconnectInterval:   int64(60),
		Format:              clientpb.OutputFormat_SHELLCODE,
		IsSharedLib:         true,
		C2:                  c2s,
	}
	/* If this config will be used to build an implant,
 If 此配置将用于构建 implant，
	we need to make sure to include the correct transport
	我们需要确保包含正确的交通工具
	for the build 
	用于构建*/
	switch beacon.Transport {
	case "mtls":
		config.IncludeMTLS = true
	case "http":
		config.IncludeHTTP = true
	case "dns":
		config.IncludeDNS = true
	case "wg":
		config.IncludeWG = true
	case "namedpipe":
		config.IncludeNamePipe = true
	case "tcppivot":
		config.IncludeTCP = true
	}
	return config
}

// exitConsole prompts the user for confirmation to exit the console.
// exitConsole 提示用户确认退出 console.
func (c *SliverClient) exitConsole(_ *console.Console) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Confirm exit (Y/y, Ctrl-C): ")
	text, _ := reader.ReadString('\n')
	answer := strings.TrimSpace(text)

	if (answer == "Y") || (answer == "y") {
		c.FlushOutput()
		os.Exit(0)
	}
}

// exitImplantMenu uses the background command to detach from the implant menu.
// exitImplantMenu 使用后台命令与 implant menu. 分离
func (c *SliverClient) exitImplantMenu(_ *console.Console) {
	root := c.App.Menu(consts.ImplantMenu).Command
	root.SetArgs([]string{"background"})
	root.Execute()
}

func (con *SliverClient) SpinUntil(message string, ctrl chan bool) {
	go spin.Until(os.Stdout, message, ctrl)
}

// FormatDateDelta - Generate formatted date string of the time delta between then and now.
// FormatDateDelta - Generate 格式的日期字符串，表示当时和 now. 之间的时间增量
func (con *SliverClient) FormatDateDelta(t time.Time, includeDate bool, color bool) string {
	nextTime := t.Format(time.UnixDate)

	var interval string

	if t.Before(time.Now()) {
		if includeDate {
			interval = fmt.Sprintf("%s (%s ago)", nextTime, time.Since(t).Round(time.Second))
		} else {
			interval = time.Since(t).Round(time.Second).String()
		}
		if color {
			interval = StyleBoldRed.Render(interval)
		}
	} else {
		if includeDate {
			interval = fmt.Sprintf("%s (in %s)", nextTime, time.Until(t).Round(time.Second))
		} else {
			interval = time.Until(t).Round(time.Second).String()
		}
		if color {
			interval = StyleBoldGreen.Render(interval)
		}
	}
	return interval
}

// GrpcContext - Generate a context for a GRPC request, if no grumble context or an invalid flag is provided 60 seconds is used instead.
// GrpcContext - Generate GRPC 请求的上下文，如果没有提供 grumble 上下文或提供无效标志，则使用 60 秒 instead.
func (con *SliverClient) GrpcContext(cmd *cobra.Command) (context.Context, context.CancelFunc) {
	if cmd == nil {
		return context.WithTimeout(context.Background(), 60*time.Second)
	}

	timeOutF, _ := cmd.Flags().GetInt64("timeout")
	timeout := time.Duration(int64(time.Second) * timeOutF)
	if timeout < 1 {
		timeout = 60 * time.Second
	}
	return context.WithTimeout(context.Background(), timeout)
}

//
// -------------------------- [ Active Target ] --------------------------
//

type ActiveTarget struct {
	session    *clientpb.Session
	beacon     *clientpb.Beacon
	observers  map[int]Observer
	observerID int
	con        *SliverClient
}

// GetSessionInteractive - Get the active target(s).
// GetSessionInteractive - Get 活动目标。
func (s *ActiveTarget) GetInteractive() (*clientpb.Session, *clientpb.Beacon) {
	if s.session == nil && s.beacon == nil {
		fmt.Print(Warn + "Please select a session or beacon via `use`\n")
		return nil, nil
	}
	return s.session, s.beacon
}

// GetSessionInteractive - Get the active target(s).
// GetSessionInteractive - Get 活动目标。
func (s *ActiveTarget) Get() (*clientpb.Session, *clientpb.Beacon) {
	return s.session, s.beacon
}

// GetSessionInteractive - GetSessionInteractive the active session.
// GetSessionInteractive - GetSessionInteractive 活跃 session.
func (s *ActiveTarget) GetSessionInteractive() *clientpb.Session {
	if s.session == nil {
		fmt.Print(Warn + "Please select a session via `use`\n")
		return nil
	}
	return s.session
}

// GetSession - Same as GetSession() but doesn't print a warning.
// GetSession - Same 作为 GetSession() 但不打印 warning.
func (s *ActiveTarget) GetSession() *clientpb.Session {
	return s.session
}

// GetBeaconInteractive - Get beacon interactive the active session.
// GetBeaconInteractive - Get beacon 互动主动 session.
func (s *ActiveTarget) GetBeaconInteractive() *clientpb.Beacon {
	if s.beacon == nil {
		fmt.Print(Warn + "Please select a beacon via `use`\n")
		return nil
	}
	return s.beacon
}

// GetBeacon - Same as GetBeacon() but doesn't print a warning.
// GetBeacon - Same 作为 GetBeacon() 但不打印 warning.
func (s *ActiveTarget) GetBeacon() *clientpb.Beacon {
	return s.beacon
}

// IsSession - Is the current target a session?
// IsSession - Is 当前目标是 session？
func (s *ActiveTarget) IsSession() bool {
	return s.session != nil
}

// IsBeacon - Is the current target a beacon?
// IsBeacon - Is 当前目标是 beacon？
func (s *ActiveTarget) IsBeacon() bool {
	return s.beacon != nil
}

// AddObserver - Observers to notify when the active session changes
// AddObserver - Observers 在活动 session 发生变化时发出通知
func (s *ActiveTarget) AddObserver(observer Observer) int {
	s.observerID++
	s.observers[s.observerID] = observer
	return s.observerID
}

func (s *ActiveTarget) RemoveObserver(observerID int) {
	delete(s.observers, observerID)
}

func (s *ActiveTarget) Request(cmd *cobra.Command) *commonpb.Request {
	if s.session == nil && s.beacon == nil {
		return nil
	}

	// One less than the gRPC timeout so that the server should timeout first
	// One 小于 gRPC 超时，因此服务器应首先超时
	timeOutF := int64(defaultTimeout) - 1
	if cmd != nil {
		timeOutF, _ = cmd.Flags().GetInt64("timeout")
	}
	timeout := (int64(time.Second) * timeOutF) - 1

	req := &commonpb.Request{}
	req.Timeout = timeout

	if s.session != nil {
		req.Async = false
		req.SessionID = s.session.ID
	}
	if s.beacon != nil {
		req.Async = true
		req.BeaconID = s.beacon.ID
	}
	return req
}

// Set - Change the active session.
// Set - Change 活跃 session.
func (s *ActiveTarget) Set(session *clientpb.Session, beacon *clientpb.Beacon) {
	if session != nil && beacon != nil {
		s.con.PrintErrorf("cannot set both an active beacon and an active session")
		return
	}

	defer s.con.ExposeCommands()

	// Backgrounding
	if session == nil && beacon == nil {
		s.session = nil
		s.beacon = nil
		for _, observer := range s.observers {
			observer(s.session, s.beacon)
		}

		if s.con.IsCLI {
			return
		}

		// Switch back to server menu.
		// Switch 返回服务器 menu.
		if s.con.App.ActiveMenu().Name() == consts.ImplantMenu {
			s.con.App.SwitchMenu(consts.ServerMenu)
		}

		return
	}

	// Foreground
	if session != nil {
		s.session = session
		s.beacon = nil
		for _, observer := range s.observers {
			observer(s.session, s.beacon)
		}
	} else if beacon != nil {
		s.beacon = beacon
		s.session = nil
		for _, observer := range s.observers {
			observer(s.session, s.beacon)
		}
	}

	if s.con.IsCLI {
		return
	}

	// Update menus, prompts and commands
	// Update 菜单、提示和命令
	if s.con.App.ActiveMenu().Name() != consts.ImplantMenu {
		s.con.App.SwitchMenu(consts.ImplantMenu)
	}
}

// Background - Background the active session.
// Background - Background 活跃 session.
func (s *ActiveTarget) Background() {
	defer s.con.App.ShowCommands()

	s.session = nil
	s.beacon = nil
	for _, observer := range s.observers {
		observer(nil, nil)
	}

	// Switch back to server menu.
	// Switch 返回服务器 menu.
	if !s.con.IsCLI && s.con.App.ActiveMenu().Name() == consts.ImplantMenu {
		s.con.App.SwitchMenu(consts.ServerMenu)
	}
}

// GetHostUUID - Get the Host's UUID (ID in the database)
// GetHostUUID - Get Host 的 UUID （数据库中的 ID）
func (s *ActiveTarget) GetHostUUID() string {
	if s.IsSession() {
		return s.session.UUID
	} else if s.IsBeacon() {
		return s.beacon.UUID
	}

	return ""
}

// Expose or hide commands if the active target does support them (or not).
// Expose 或隐藏命令（如果活动目标支持（或不支持））。
// Ex; hide Windows commands on Linux implants, Wireguard tools on HTTP C2, etc.
// Ex；隐藏 Linux 种植体上的 Windows 命令，隐藏 HTTP C2、etc. 上的 Wireguard 工具
func (con *SliverClient) ExposeCommands() {
	con.App.ShowCommands()

	if con.ActiveTarget.session == nil && con.ActiveTarget.beacon == nil {
		return
	}

	filters := make([]string, 0)

	// Target type.
	switch {
	case con.ActiveTarget.session != nil:
		session := con.ActiveTarget.session
		filters = append(filters, consts.BeaconCmdsFilter)

		// Operating system
		// Operating系统
		if session.OS != "windows" {
			filters = append(filters, consts.WindowsCmdsFilter)
		}

		// C2 stack
		// C2 堆栈
		if session.Transport != "wg" {
			filters = append(filters, consts.WireguardCmdsFilter)
		}

	case con.ActiveTarget.beacon != nil:
		beacon := con.ActiveTarget.beacon
		filters = append(filters, consts.SessionCmdsFilter)

		// Operating system
		// Operating系统
		if beacon.OS != "windows" {
			filters = append(filters, consts.WindowsCmdsFilter)
		}

		// C2 stack
		// C2 堆栈
		if beacon.Transport != "wg" {
			filters = append(filters, consts.WireguardCmdsFilter)
		}
	}

	// Use all defined filters.
	// Use 全部定义 filters.
	con.App.HideCommands(filters...)
}

var abilities = []string{
	"first strike",
	"vigilance",
	"haste",
	"indestructible",
	"hexproof",
	"deathtouch",
	"fear",
	"epic",
	"ninjitsu",
	"recover",
	"persist",
	"conspire",
	"reinforce",
	"exalted",
	"annihilator",
	"infect",
	"undying",
	"living weapon",
	"miracle",
	"scavenge",
	"cipher",
	"evolve",
	"dethrone",
	"hidden agenda",
	"prowess",
	"dash",
	"exploit",
	"renown",
	"skulk",
	"improvise",
	"assist",
	"jump-start",
}

type asciiLogoStyle int

const (
	logoStyleRed asciiLogoStyle = iota
	logoStyleGreen
	logoStyleBoldGray
)

type asciiLogo struct {
	style asciiLogoStyle
	art   string
}

func (l asciiLogo) Render() string {
	switch l.style {
	case logoStyleRed:
		return StyleRed.Render(l.art)
	case logoStyleGreen:
		return StyleGreen.Render(l.art)
	case logoStyleBoldGray:
		return StyleBoldGray.Render(l.art)
	default:
		return l.art
	}
}

var asciiLogos = []asciiLogo{
	{style: logoStyleRed, art: `
	 	  ██████  ██▓     ██▓ ██▒   █▓▓█████  ██▀███
		▒██    ▒ ▓██▒    ▓██▒▓██░   █▒▓█   ▀ ▓██ ▒ ██▒
		░ ▓██▄   ▒██░    ▒██▒ ▓██  █▒░▒███   ▓██ ░▄█ ▒
		  ▒   ██▒▒██░    ░██░  ▒██ █░░▒▓█  ▄ ▒██▀▀█▄
		▒██████▒▒░██████▒░██░   ▒▀█░  ░▒████▒░██▓ ▒██▒
		▒ ▒▓▒ ▒ ░░ ▒░▓  ░░▓     ░ ▐░  ░░ ▒░ ░░ ▒▓ ░▒▓░
		░ ░▒  ░ ░░ ░ ▒  ░ ▒ ░   ░ ░░   ░ ░  ░  ░▒ ░ ▒░
		░  ░  ░    ░ ░    ▒ ░     ░░     ░     ░░   ░
			  ░      ░  ░ ░        ░     ░  ░   ░
	`},

	{style: logoStyleGreen, art: `
	    ███████╗██╗     ██╗██╗   ██╗███████╗██████╗
	    ██╔════╝██║     ██║██║   ██║██╔════╝██╔══██╗
	    ███████╗██║     ██║██║   ██║█████╗  ██████╔╝
	    ╚════██║██║     ██║╚██╗ ██╔╝██╔══╝  ██╔══██╗
	    ███████║███████╗██║ ╚████╔╝ ███████╗██║  ██║
	    ╚══════╝╚══════╝╚═╝  ╚═══╝  ╚══════╝╚═╝  ╚═╝
	`},

	{style: logoStyleBoldGray, art: `
.------..------..------..------..------..------.
|S.--. ||L.--. ||I.--. ||V.--. ||E.--. ||R.--. |
| :/\: || :/\: || (\/) || :(): || (\/) || :(): |
| :\/: || (__) || :\/: || ()() || :\/: || ()() |
| '--'S|| '--'L|| '--'I|| '--'V|| '--'E|| '--'R|
` + "`------'`------'`------'`------'`------'`------'" + `
	`},
}
