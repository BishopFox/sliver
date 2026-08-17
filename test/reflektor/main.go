package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	clientassets "github.com/bishopfox/sliver/client/assets"
	consts "github.com/bishopfox/sliver/client/constants"
	clienttransport "github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"golang.org/x/mod/modfile"
)

const (
	operatorName          = "testuser"
	reflektorModule       = "github.com/sliverarmory/reflektor"
	reflektorExport       = "StartW"
	processLogTailBytes   = 32 * 1024
	eventBufferSize       = 128
	eventSyncTimeout      = 30 * time.Second
	listenerPollInterval  = 250 * time.Millisecond
	listenerDialTimeout   = 500 * time.Millisecond
	cleanupGraceTimeout   = 2 * time.Second
	cleanupProcessTimeout = 10 * time.Second
)

type options struct {
	repoPath       string
	serverPath     string
	targetOS       string
	targetArch     string
	timeout        time.Duration
	startupTimeout time.Duration
	sessionTimeout time.Duration
}

type managedProcess struct {
	cmd     *exec.Cmd
	done    chan struct{}
	err     error
	logPath string
}

type eventPump struct {
	events chan *clientpb.Event
	err    error
}

func main() {
	opts := options{}
	flag.StringVar(&opts.repoPath, "repo", ".", "path to the Sliver repository")
	flag.StringVar(&opts.serverPath, "server", "", "path to the Sliver server executable")
	flag.StringVar(&opts.targetOS, "target-os", runtime.GOOS, "shared library target operating system")
	flag.StringVar(&opts.targetArch, "target-arch", runtime.GOARCH, "shared library target architecture")
	flag.DurationVar(&opts.timeout, "timeout", 60*time.Minute, "overall integration test timeout")
	flag.DurationVar(&opts.startupTimeout, "startup-timeout", 10*time.Minute, "Sliver daemon startup timeout")
	flag.DurationVar(&opts.sessionTimeout, "session-timeout", 3*time.Minute, "session connection timeout after Reflektor starts")
	flag.Parse()

	if err := run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "reflektor integration test failed: %v\n", err)
		os.Exit(1)
	}
}

func run(opts options) error {
	if err := validateOptions(&opts); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), opts.timeout)
	defer cancel()

	workDir, err := os.MkdirTemp("", "sliver-reflektor-e2e-")
	if err != nil {
		return fmt.Errorf("create test directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	serverRoot := filepath.Join(workDir, "server")
	clientRoot := filepath.Join(workDir, "client")
	homeDir := filepath.Join(workDir, "home")
	for _, dir := range []string{serverRoot, clientRoot, homeDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create isolated test directory %q: %w", dir, err)
		}
	}

	grpcPort, err := unusedTCPPort()
	if err != nil {
		return fmt.Errorf("select multiplayer port: %w", err)
	}

	serverEnv := os.Environ()
	for name, value := range map[string]string{
		"HOME":                   homeDir,
		"SLIVER_CLIENT_ROOT_DIR": clientRoot,
		"SLIVER_ROOT_DIR":        serverRoot,
		"USERPROFILE":            homeDir,
	} {
		serverEnv = envWith(serverEnv, name, value)
	}
	serverLog := filepath.Join(workDir, "sliver-server.log")
	serverProcess, err := startProcess(
		opts.serverPath,
		[]string{
			"daemon",
			"--lhost", "127.0.0.1",
			"--lport", fmt.Sprintf("%d", grpcPort),
			"--force",
		},
		opts.repoPath,
		serverEnv,
		serverLog,
	)
	if err != nil {
		return fmt.Errorf("start Sliver daemon: %w", err)
	}
	defer serverProcess.stop()

	fmt.Printf("Started Sliver daemon on 127.0.0.1:%d\n", grpcPort)
	startupCtx, startupCancel := context.WithTimeout(ctx, opts.startupTimeout)
	err = waitForTCP(startupCtx, fmt.Sprintf("127.0.0.1:%d", grpcPort), serverProcess)
	startupCancel()
	if err != nil {
		return fmt.Errorf("wait for Sliver daemon: %w", err)
	}

	profilePath := filepath.Join(workDir, "testuser.cfg")
	profileCtx, profileCancel := context.WithTimeout(ctx, 2*time.Minute)
	profileOutput, profileErr := runCommand(
		profileCtx,
		opts.repoPath,
		serverEnv,
		opts.serverPath,
		"operator",
		"--name", operatorName,
		"--lhost", "127.0.0.1",
		"--lport", fmt.Sprintf("%d", grpcPort),
		"--permissions", "all",
		"--save", profilePath,
	)
	profileCancel()
	if profileErr != nil {
		return fmt.Errorf("generate operator profile: %w\n%s", profileErr, profileOutput)
	}
	if _, err := os.Stat(profilePath); err != nil {
		return fmt.Errorf("operator command did not create %q: %w\n%s", profilePath, err, profileOutput)
	}

	clientConfig, err := clientassets.ReadConfig(profilePath)
	if err != nil {
		return fmt.Errorf("read operator profile: %w", err)
	}
	if clientConfig.Operator != operatorName || clientConfig.LHost != "127.0.0.1" || clientConfig.LPort != grpcPort {
		return fmt.Errorf(
			"operator profile mismatch: got operator=%q endpoint=%s:%d",
			clientConfig.Operator,
			clientConfig.LHost,
			clientConfig.LPort,
		)
	}
	fmt.Printf("Saved %s multiplayer profile to disk\n", operatorName)

	rpc, conn, err := clienttransport.MTLSConnect(clientConfig)
	if err != nil {
		return fmt.Errorf("connect to multiplayer gRPC: %w", err)
	}
	defer clienttransport.CloseGRPCConnection(conn)

	version, err := rpc.GetVersion(ctx, &commonpb.Empty{})
	if err != nil {
		return fmt.Errorf("query Sliver server version: %w", err)
	}
	fmt.Printf("Connected to Sliver server %s/%s over gRPC\n", version.OS, version.Arch)

	eventsCtx, eventsCancel := context.WithCancel(ctx)
	defer eventsCancel()
	events, err := rpc.Events(eventsCtx, &commonpb.Empty{})
	if err != nil {
		return fmt.Errorf("subscribe to Sliver events: %w", err)
	}
	pump := startEventPump(events)

	httpPort, err := unusedTCPPort()
	if err != nil {
		return fmt.Errorf("select HTTP port: %w", err)
	}

	listener, err := rpc.StartHTTPListener(ctx, &clientpb.HTTPListenerReq{
		Domain:          "localhost",
		Host:            "127.0.0.1",
		Port:            uint32(httpPort),
		Secure:          false,
		EnforceOTP:      false,
		LongPollTimeout: int64(time.Second),
	})
	if err != nil {
		return fmt.Errorf("start HTTP listener: %w", err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = rpc.KillJob(cleanupCtx, &clientpb.KillJobReq{ID: listener.JobID})
	}()
	listenerCtx, listenerCancel := context.WithTimeout(ctx, eventSyncTimeout)
	err = waitForListenerReady(listenerCtx, pump, listener.JobID, fmt.Sprintf("127.0.0.1:%d", httpPort))
	listenerCancel()
	if err != nil {
		return fmt.Errorf("wait for HTTP listener: %w", err)
	}
	fmt.Printf("Started HTTP listener on 127.0.0.1:%d (job %d)\n", httpPort, listener.JobID)

	implantName := fmt.Sprintf("reflektor%s%s", opts.targetOS, opts.targetArch)
	generated, err := rpc.Generate(ctx, &clientpb.GenerateReq{
		Name: implantName,
		Config: &clientpb.ImplantConfig{
			GOOS:                opts.targetOS,
			GOARCH:              opts.targetArch,
			TemplateName:        "sliver",
			Debug:               false,
			ObfuscateSymbols:    false,
			IsBeacon:            false,
			Format:              clientpb.OutputFormat_SHARED_LIB,
			IsSharedLib:         true,
			RunAtLoad:           false,
			Exports:             []string{reflektorExport},
			C2:                  []*clientpb.ImplantC2{{URL: fmt.Sprintf("http://127.0.0.1:%d?force-http=true&poll-timeout=1s", httpPort)}},
			IncludeHTTP:         true,
			HTTPC2ConfigName:    consts.DefaultC2Profile,
			ConnectionStrategy:  "s",
			ReconnectInterval:   int64(time.Second),
			PollTimeout:         int64(time.Second),
			MaxConnectionErrors: 10,
			NetGoEnabled:        true,
		},
	})
	if err != nil {
		return fmt.Errorf("generate %s/%s shared library: %w", opts.targetOS, opts.targetArch, err)
	}
	if generated.File == nil || generated.File.Name == "" || len(generated.File.Data) == 0 {
		return errors.New("Sliver returned an empty shared library")
	}

	buildEventCtx, buildEventCancel := context.WithTimeout(ctx, eventSyncTimeout)
	buildEvent, buildEventErr := waitForEvent(buildEventCtx, pump, func(event *clientpb.Event) bool {
		return (event.EventType == consts.BuildCompletedEvent && string(event.Data) == generated.File.Name) ||
			isListenerStopped(event, listener.JobID)
	})
	buildEventCancel()
	if buildEventErr != nil {
		return fmt.Errorf("wait for shared-library build event: %w", buildEventErr)
	}
	if err := listenerStoppedError(buildEvent, listener.JobID); err != nil {
		return err
	}

	libraryPath := filepath.Join(workDir, filepath.Base(generated.File.Name))
	if err := os.WriteFile(libraryPath, generated.File.Data, 0o600); err != nil {
		return fmt.Errorf("save generated shared library: %w", err)
	}
	fmt.Printf("Generated %s/%s session shared library %s\n", opts.targetOS, opts.targetArch, filepath.Base(libraryPath))

	reflektorPath, reflektorVersion, err := buildReflektorCLI(ctx, opts.repoPath, workDir, opts.targetOS, opts.targetArch)
	if err != nil {
		return err
	}
	fmt.Printf("Built Reflektor CLI %s from Sliver go.mod\n", reflektorVersion)

	reflektorLog := filepath.Join(workDir, "reflektor.log")
	reflektorProcess, err := startProcess(
		reflektorPath,
		[]string{"--call-export", reflektorExport, libraryPath},
		opts.repoPath,
		os.Environ(),
		reflektorLog,
	)
	if err != nil {
		return fmt.Errorf("start Reflektor CLI: %w", err)
	}
	defer reflektorProcess.stop()

	sessionCtx, sessionCancel := context.WithTimeout(ctx, opts.sessionTimeout)
	defer sessionCancel()
	sessionEvent, err := waitForSession(
		sessionCtx,
		pump,
		reflektorProcess,
		listener.JobID,
		generated.ImplantName,
		opts.targetOS,
		opts.targetArch,
	)
	if err != nil {
		return err
	}

	fmt.Printf(
		"Reflektor loaded Sliver and connected session %s (%s, %s/%s)\n",
		sessionEvent.Session.ID,
		sessionEvent.Session.Name,
		sessionEvent.Session.OS,
		sessionEvent.Session.Arch,
	)
	return nil
}

func validateOptions(opts *options) error {
	if opts.serverPath == "" {
		return errors.New("-server is required")
	}

	repoPath, err := filepath.Abs(opts.repoPath)
	if err != nil {
		return fmt.Errorf("resolve repository path: %w", err)
	}
	serverPath, err := filepath.Abs(opts.serverPath)
	if err != nil {
		return fmt.Errorf("resolve server path: %w", err)
	}
	if stat, err := os.Stat(serverPath); err != nil {
		return fmt.Errorf("stat server executable: %w", err)
	} else if stat.IsDir() {
		return fmt.Errorf("server executable %q is a directory", serverPath)
	}
	if _, err := os.Stat(filepath.Join(repoPath, "go.mod")); err != nil {
		return fmt.Errorf("repository does not contain go.mod: %w", err)
	}

	opts.repoPath = repoPath
	opts.serverPath = serverPath
	opts.targetOS = strings.ToLower(strings.TrimSpace(opts.targetOS))
	opts.targetArch = strings.ToLower(strings.TrimSpace(opts.targetArch))

	supported := map[string]bool{
		"darwin/amd64":  true,
		"darwin/arm64":  true,
		"linux/386":     true,
		"linux/amd64":   true,
		"linux/arm64":   true,
		"windows/386":   true,
		"windows/amd64": true,
		"windows/arm64": true,
	}
	target := opts.targetOS + "/" + opts.targetArch
	if !supported[target] {
		return fmt.Errorf("target %s is not in Reflektor v0.0.2's test matrix", target)
	}
	if opts.targetOS != runtime.GOOS || opts.targetArch != runtime.GOARCH {
		return fmt.Errorf(
			"Reflektor must run as the target architecture: driver is %s/%s, target is %s",
			runtime.GOOS,
			runtime.GOARCH,
			target,
		)
	}
	if opts.timeout <= 0 || opts.startupTimeout <= 0 || opts.sessionTimeout <= 0 {
		return errors.New("timeouts must be positive")
	}
	return nil
}

func unusedTCPPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func waitForTCP(ctx context.Context, address string, process *managedProcess) error {
	ticker := time.NewTicker(listenerPollInterval)
	defer ticker.Stop()

	for {
		conn, err := net.DialTimeout("tcp", address, listenerDialTimeout)
		if err == nil {
			_ = conn.Close()
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-process.done:
			return process.failure("Sliver daemon exited before accepting connections")
		case <-ticker.C:
		}
	}
}

func startProcess(path string, args []string, dir string, env []string, logPath string) (*managedProcess, error) {
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(path, args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	prepareCommand(cmd)
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, err
	}

	process := &managedProcess{
		cmd:     cmd,
		done:    make(chan struct{}),
		logPath: logPath,
	}
	go func() {
		process.err = cmd.Wait()
		_ = logFile.Close()
		close(process.done)
	}()
	return process, nil
}

func (process *managedProcess) stop() {
	select {
	case <-process.done:
		return
	default:
	}

	terminateProcessTree(process.cmd)
	select {
	case <-process.done:
		return
	case <-time.After(cleanupGraceTimeout):
	}

	killProcessTree(process.cmd)
	select {
	case <-process.done:
	case <-time.After(cleanupProcessTimeout):
		fmt.Fprintf(os.Stderr, "timed out cleaning up process tree for %s (pid %d)\n", process.cmd.Path, process.cmd.Process.Pid)
	}
}

func (process *managedProcess) failure(message string) error {
	processErr := process.err
	if processErr == nil {
		processErr = errors.New("process exited")
	}
	return fmt.Errorf("%s: %w\n%s", message, processErr, readLogTail(process.logPath))
}

func readLogTail(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("(could not read %s: %v)", path, err)
	}
	if len(data) > processLogTailBytes {
		data = data[len(data)-processLogTailBytes:]
	}
	return strings.TrimSpace(string(data))
}

func runCommand(ctx context.Context, dir string, env []string, path string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Dir = dir
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

func envWith(env []string, name string, value string) []string {
	prefix := name + "="
	updated := make([]string, 0, len(env)+1)
	for _, entry := range env {
		if !strings.HasPrefix(entry, prefix) {
			updated = append(updated, entry)
		}
	}
	return append(updated, prefix+value)
}

func startEventPump(stream rpcpb.SliverRPC_EventsClient) *eventPump {
	pump := &eventPump{
		events: make(chan *clientpb.Event, eventBufferSize),
	}
	go func() {
		defer close(pump.events)
		for {
			event, err := stream.Recv()
			if err != nil {
				pump.err = err
				return
			}
			select {
			case pump.events <- event:
			case <-stream.Context().Done():
				pump.err = stream.Context().Err()
				return
			}
		}
	}()
	return pump
}

func waitForListenerReady(ctx context.Context, pump *eventPump, jobID uint32, address string) error {
	ticker := time.NewTicker(listenerPollInterval)
	defer ticker.Stop()
	started := false

	for {
		if started {
			conn, err := net.DialTimeout("tcp", address, listenerDialTimeout)
			if err == nil {
				_ = conn.Close()
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-pump.events:
			if !ok {
				return eventStreamError(pump, "event stream ended while starting HTTP listener")
			}
			if event.Job == nil || event.Job.ID != jobID {
				continue
			}
			switch event.EventType {
			case consts.JobStartedEvent:
				started = true
			case consts.JobStoppedEvent:
				if event.Err != "" {
					return fmt.Errorf("HTTP listener job %d stopped: %s", jobID, event.Err)
				}
				return fmt.Errorf("HTTP listener job %d stopped before accepting connections", jobID)
			}
		case <-ticker.C:
		}
	}
}

func waitForEvent(ctx context.Context, pump *eventPump, match func(*clientpb.Event) bool) (*clientpb.Event, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case event, ok := <-pump.events:
			if !ok {
				return nil, eventStreamError(pump, "event stream ended")
			}
			if match(event) {
				return event, nil
			}
		}
	}
}

func waitForSession(
	ctx context.Context,
	pump *eventPump,
	reflektor *managedProcess,
	listenerJobID uint32,
	implantName string,
	targetOS string,
	targetArch string,
) (*clientpb.Event, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for Reflektor-loaded session: %w\n%s", ctx.Err(), readLogTail(reflektor.logPath))
		case <-reflektor.done:
			for {
				select {
				case event, ok := <-pump.events:
					if !ok {
						return nil, eventStreamError(pump, "event stream ended while waiting for session")
					}
					if err := listenerStoppedError(event, listenerJobID); err != nil {
						return nil, err
					}
					if matched, err := matchingSession(event, implantName, targetOS, targetArch); matched || err != nil {
						return event, err
					}
				default:
					return nil, reflektor.failure("Reflektor exited before Sliver connected")
				}
			}
		case event, ok := <-pump.events:
			if !ok {
				return nil, eventStreamError(pump, "event stream ended while waiting for session")
			}
			if err := listenerStoppedError(event, listenerJobID); err != nil {
				return nil, err
			}
			if matched, err := matchingSession(event, implantName, targetOS, targetArch); matched || err != nil {
				return event, err
			}
		}
	}
}

func isListenerStopped(event *clientpb.Event, jobID uint32) bool {
	return event.EventType == consts.JobStoppedEvent && event.Job != nil && event.Job.ID == jobID
}

func listenerStoppedError(event *clientpb.Event, jobID uint32) error {
	if !isListenerStopped(event, jobID) {
		return nil
	}
	if event.Err != "" {
		return fmt.Errorf("HTTP listener job %d stopped: %s", jobID, event.Err)
	}
	return fmt.Errorf("HTTP listener job %d stopped before the session connected", jobID)
}

func matchingSession(event *clientpb.Event, implantName string, targetOS string, targetArch string) (bool, error) {
	if event.EventType != consts.SessionOpenedEvent || event.Session == nil {
		return false, nil
	}
	if event.Session.Name != implantName {
		return false, nil
	}
	if event.Session.OS != targetOS || event.Session.Arch != targetArch {
		return true, fmt.Errorf(
			"session target mismatch: got %s/%s, want %s/%s",
			event.Session.OS,
			event.Session.Arch,
			targetOS,
			targetArch,
		)
	}
	return true, nil
}

func eventStreamError(pump *eventPump, message string) error {
	if pump.err == nil {
		return errors.New(message)
	}
	return fmt.Errorf("%s: %w", message, pump.err)
}

func buildReflektorCLI(
	ctx context.Context,
	repoPath string,
	workDir string,
	targetOS string,
	targetArch string,
) (string, string, error) {
	goMod, err := os.ReadFile(filepath.Join(repoPath, "go.mod"))
	if err != nil {
		return "", "", fmt.Errorf("read go.mod: %w", err)
	}
	parsed, err := modfile.Parse("go.mod", goMod, nil)
	if err != nil {
		return "", "", fmt.Errorf("parse go.mod: %w", err)
	}

	version := ""
	for _, requirement := range parsed.Require {
		if requirement.Mod.Path == reflektorModule {
			version = requirement.Mod.Version
			break
		}
	}
	if version == "" {
		return "", "", fmt.Errorf("%s is not required by go.mod", reflektorModule)
	}

	binDir := filepath.Join(workDir, "reflektor-bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		return "", "", fmt.Errorf("create Reflektor output directory: %w", err)
	}
	name := "reflektor"
	if targetOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(binDir, name)

	cgoEnabled := "0"
	if targetOS == "darwin" || targetOS == "linux" {
		cgoEnabled = "1"
	}
	buildEnv := os.Environ()
	for name, value := range map[string]string{
		"CGO_ENABLED": cgoEnabled,
		"GOARCH":      targetArch,
		"GOENV":       "off",
		"GOFLAGS":     "",
		"GOOS":        targetOS,
		"GOWORK":      "off",
	} {
		buildEnv = envWith(buildEnv, name, value)
	}

	listCommand := exec.CommandContext(ctx, "go", "list", "-mod=readonly", "-m", "-f={{.Version}}", reflektorModule)
	listCommand.Dir = repoPath
	listCommand.Env = buildEnv
	output, err := listCommand.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("resolve Reflektor CLI version: %w\n%s", err, output)
	}
	if selected := strings.TrimSpace(string(output)); selected != version {
		return "", "", fmt.Errorf("Reflektor version mismatch: go.mod requires %s, module graph selected %s", version, selected)
	}

	buildCommand := exec.CommandContext(
		ctx,
		"go",
		"build",
		"-mod=readonly",
		"-trimpath",
		"-o", path,
		reflektorModule+"/cli",
	)
	buildCommand.Dir = repoPath
	buildCommand.Env = buildEnv
	output, err = buildCommand.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("build Reflektor CLI %s for %s/%s: %w\n%s", version, targetOS, targetArch, err, output)
	}
	return path, version, nil
}
