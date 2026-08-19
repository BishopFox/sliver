//go:build sliver_controlflow_e2e

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
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

const (
	operatorName          = "controlflowe2e"
	targetOS              = "darwin"
	targetArch            = "arm64"
	controlFlowCapability = "garble.control-flow/balanced-v1"
	processLogTailBytes   = 1024 * 1024
	listenerPollInterval  = 250 * time.Millisecond
	listenerDialTimeout   = 500 * time.Millisecond
	cleanupGraceTimeout   = 2 * time.Second
	cleanupProcessTimeout = 10 * time.Second
	sessionPingNonce      = int32(0x534553)
	beaconPingNonce       = int32(0x424543)
)

type options struct {
	repoPath        string
	serverPath      string
	timeout         time.Duration
	startupTimeout  time.Duration
	generateTimeout time.Duration
	callbackTimeout time.Duration
	keepWorkDir     bool
}

type testLayout struct {
	workDir        string
	serverRoot     string
	clientRoot     string
	homeDir        string
	tempDir        string
	logDir         string
	artifactDir    string
	garbleDebugDir string
}

type generatedArtifact struct {
	name string
	path string
	data []byte
}

func main() {
	opts := options{}
	flag.StringVar(&opts.repoPath, "repo", ".", "path to the Sliver repository")
	flag.StringVar(&opts.serverPath, "server", "", "prebuilt Sliver server executable; omitted builds the current checkout into the isolated work directory")
	flag.DurationVar(&opts.timeout, "timeout", 90*time.Minute, "overall integration timeout")
	flag.DurationVar(&opts.startupTimeout, "startup-timeout", 10*time.Minute, "Sliver daemon startup timeout")
	flag.DurationVar(&opts.generateTimeout, "generate-timeout", 40*time.Minute, "timeout for each Garble implant build")
	flag.DurationVar(&opts.callbackTimeout, "callback-timeout", 5*time.Minute, "timeout for each implant callback and Ping verification")
	flag.BoolVar(&opts.keepWorkDir, "keep-work-dir", false, "preserve the isolated work directory after success (failed runs are always preserved)")
	flag.Parse()

	if err := run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "control-flow integration failed: %v\n", err)
		os.Exit(1)
	}
}

func run(opts options) (retErr error) {
	if err := validateOptions(&opts); err != nil {
		return err
	}

	layout, err := newTestLayout()
	if err != nil {
		return err
	}
	fmt.Printf("Control-flow integration work directory: %s\n", layout.workDir)
	defer func() {
		if opts.keepWorkDir || retErr != nil {
			fmt.Printf("Preserved integration work directory: %s\n", layout.workDir)
			return
		}
		if err := os.RemoveAll(layout.workDir); err != nil {
			fmt.Fprintf(os.Stderr, "warning: remove integration work directory: %v\n", err)
		}
	}()

	isolation := map[string]string{
		"HOME":                    layout.homeDir,
		"SLIVER_CLIENT_ROOT_DIR":  layout.clientRoot,
		"SLIVER_GARBLE_DEBUG_DIR": layout.garbleDebugDir,
		"SLIVER_ROOT_DIR":         layout.serverRoot,
		"TMPDIR":                  layout.tempDir,
		"USERPROFILE":             layout.homeDir,
	}
	restoreEnvironment, err := setProcessEnvironment(isolation)
	if err != nil {
		return fmt.Errorf("set isolated process environment: %w", err)
	}
	defer restoreEnvironment()
	isolatedEnv := environmentWith(os.Environ(), isolation)

	ctx, cancel := context.WithTimeout(context.Background(), opts.timeout)
	defer cancel()

	serverPath := opts.serverPath
	if serverPath == "" {
		serverPath, err = buildServer(ctx, opts.repoPath, layout, isolatedEnv)
		if err != nil {
			return err
		}
		fmt.Printf("Built Sliver server from current checkout: %s\n", serverPath)
	} else {
		fmt.Printf("Using prebuilt Sliver server: %s\n", serverPath)
	}

	multiplayerPort, implantPort, err := twoUnusedTCPPorts()
	if err != nil {
		return fmt.Errorf("select loopback ports: %w", err)
	}

	serverLog := filepath.Join(layout.logDir, "sliver-server.log")
	serverProcess, err := startProcess(
		serverPath,
		[]string{
			"daemon",
			"--lhost", "127.0.0.1",
			"--lport", fmt.Sprintf("%d", multiplayerPort),
			"--force",
		},
		opts.repoPath,
		isolatedEnv,
		serverLog,
	)
	if err != nil {
		return fmt.Errorf("start Sliver daemon: %w", err)
	}
	defer serverProcess.stop()
	defer func() {
		if retErr != nil {
			retErr = fmt.Errorf("%w\nSliver server log:\n%s", retErr, readLogTail(serverLog))
		}
	}()

	fmt.Printf("Started Sliver daemon on 127.0.0.1:%d\n", multiplayerPort)
	startupCtx, startupCancel := context.WithTimeout(ctx, opts.startupTimeout)
	err = waitForTCP(startupCtx, fmt.Sprintf("127.0.0.1:%d", multiplayerPort), serverProcess)
	startupCancel()
	if err != nil {
		return fmt.Errorf("wait for Sliver daemon: %w", err)
	}
	profilePath := filepath.Join(layout.clientRoot, operatorName+".cfg")
	profileCtx, profileCancel := context.WithTimeout(ctx, 2*time.Minute)
	profileOutput, profileErr := runOwnedCommand(
		profileCtx,
		opts.repoPath,
		isolatedEnv,
		serverPath,
		"operator",
		"--name", operatorName,
		"--lhost", "127.0.0.1",
		"--lport", fmt.Sprintf("%d", multiplayerPort),
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
	if clientConfig.Operator != operatorName || clientConfig.LHost != "127.0.0.1" || clientConfig.LPort != multiplayerPort {
		return fmt.Errorf(
			"operator profile mismatch: got operator=%q endpoint=%s:%d",
			clientConfig.Operator,
			clientConfig.LHost,
			clientConfig.LPort,
		)
	}

	clienttransport.SetMultiplayerConnectMode(clienttransport.MultiplayerConnectDirect)
	rpc, conn, err := clienttransport.MTLSConnect(clientConfig)
	if err != nil {
		return fmt.Errorf("connect to multiplayer gRPC: %w", err)
	}
	defer clienttransport.CloseGRPCConnection(conn)

	version, err := rpc.GetVersion(ctx, &commonpb.Empty{})
	if err != nil {
		return fmt.Errorf("query Sliver server version: %w", err)
	}
	fmt.Printf("Connected to Sliver server %s/%s over multiplayer mTLS\n", version.OS, version.Arch)
	if err := verifyCompilerCapability(ctx, rpc); err != nil {
		return err
	}

	eventsCtx, eventsCancel := context.WithCancel(ctx)
	defer eventsCancel()
	eventStream, err := rpc.Events(eventsCtx, &commonpb.Empty{})
	if err != nil {
		return fmt.Errorf("subscribe to Sliver events: %w", err)
	}
	pump := startEventPump(eventStream)

	listener, err := rpc.StartMTLSListener(ctx, &clientpb.MTLSListenerReq{
		Host: "127.0.0.1",
		Port: uint32(implantPort),
	})
	if err != nil {
		return fmt.Errorf("start implant mTLS listener: %w", err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = rpc.KillJob(cleanupCtx, &clientpb.KillJobReq{ID: listener.JobID})
	}()

	listenerCtx, listenerCancel := context.WithTimeout(ctx, 30*time.Second)
	err = waitForListenerReady(listenerCtx, pump, listener.JobID, fmt.Sprintf("127.0.0.1:%d", implantPort))
	listenerCancel()
	if err != nil {
		return fmt.Errorf("wait for implant mTLS listener: %w", err)
	}
	fmt.Printf("Started implant mTLS listener on 127.0.0.1:%d (job %d)\n", implantPort, listener.JobID)

	sessionArtifact, err := generateArtifact(ctx, opts.generateTimeout, rpc, layout, implantPort, "cfoe2esession", false)
	if err != nil {
		return fmt.Errorf("generate control-flow session: %w", err)
	}
	fmt.Printf("Generated control-flow session %s (%d bytes)\n", sessionArtifact.path, len(sessionArtifact.data))

	sessionLog := filepath.Join(layout.logDir, "session.log")
	sessionProcess, err := startProcess(sessionArtifact.path, nil, layout.artifactDir, isolatedEnv, sessionLog)
	if err != nil {
		return fmt.Errorf("start generated session: %w", err)
	}
	defer sessionProcess.stop()

	sessionCtx, sessionCancel := context.WithTimeout(ctx, opts.callbackTimeout)
	sessionEvent, err := waitForMatchingSession(
		sessionCtx,
		pump,
		sessionProcess,
		listener.JobID,
		sessionArtifact.name,
	)
	if err == nil {
		err = verifySessionPing(sessionCtx, rpc, sessionEvent.Session.ID)
	}
	if err == nil {
		err = verifySessionFilteredLs(sessionCtx, rpc, sessionEvent.Session.ID, layout.tempDir)
	}
	sessionCancel()
	if err != nil {
		return fmt.Errorf("verify generated session callback: %w\nsession log:\n%s", err, readLogTail(sessionLog))
	}
	fmt.Printf("Verified SessionOpened %s and synchronous Ping nonce %d\n", sessionEvent.Session.ID, sessionPingNonce)
	sessionProcess.stop()

	beaconArtifact, err := generateArtifact(ctx, opts.generateTimeout, rpc, layout, implantPort, "cfoe2ebeacon", true)
	if err != nil {
		return fmt.Errorf("generate control-flow beacon: %w", err)
	}
	fmt.Printf("Generated control-flow beacon %s (%d bytes)\n", beaconArtifact.path, len(beaconArtifact.data))

	beaconLog := filepath.Join(layout.logDir, "beacon.log")
	beaconProcess, err := startProcess(beaconArtifact.path, nil, layout.artifactDir, isolatedEnv, beaconLog)
	if err != nil {
		return fmt.Errorf("start generated beacon: %w", err)
	}
	defer beaconProcess.stop()

	beaconCtx, beaconCancel := context.WithTimeout(ctx, opts.callbackTimeout)
	beacon, err := waitForMatchingBeacon(
		beaconCtx,
		pump,
		beaconProcess,
		listener.JobID,
		beaconArtifact.name,
	)
	if err == nil {
		err = verifyBeaconPing(beaconCtx, rpc, beaconProcess, beacon.ID)
	}
	beaconCancel()
	if err != nil {
		return fmt.Errorf("verify generated beacon callback: %w\nbeacon log:\n%s", err, readLogTail(beaconLog))
	}
	fmt.Printf("Verified BeaconRegistered %s and completed asynchronous Ping nonce %d\n", beacon.ID, beaconPingNonce)
	beaconProcess.stop()

	fmt.Println("PASS: Darwin/arm64 control-flow session and beacon both connected and answered Ping")
	return nil
}

func validateOptions(opts *options) error {
	if runtime.GOOS != targetOS || runtime.GOARCH != targetArch {
		return fmt.Errorf("control-flow callback harness must run natively on %s/%s, current host is %s/%s", targetOS, targetArch, runtime.GOOS, runtime.GOARCH)
	}
	if opts.timeout <= 0 || opts.startupTimeout <= 0 || opts.generateTimeout <= 0 || opts.callbackTimeout <= 0 {
		return errors.New("timeouts must be positive")
	}

	repoPath, err := filepath.Abs(opts.repoPath)
	if err != nil {
		return fmt.Errorf("resolve repository path: %w", err)
	}
	if _, err := os.Stat(filepath.Join(repoPath, "go.mod")); err != nil {
		return fmt.Errorf("repository does not contain go.mod: %w", err)
	}
	opts.repoPath = repoPath

	if opts.serverPath == "" {
		return validateEmbeddedAssetInputs(repoPath)
	}
	serverPath, err := filepath.Abs(opts.serverPath)
	if err != nil {
		return fmt.Errorf("resolve server executable: %w", err)
	}
	stat, err := os.Stat(serverPath)
	if err != nil {
		return fmt.Errorf("stat server executable: %w", err)
	}
	if stat.IsDir() {
		return fmt.Errorf("server executable %q is a directory", serverPath)
	}
	if stat.Mode()&0o111 == 0 {
		return fmt.Errorf("server executable %q is not executable", serverPath)
	}
	opts.serverPath = serverPath
	return nil
}

func validateEmbeddedAssetInputs(repoPath string) error {
	required := []string{
		filepath.Join("server", "assets", "fs", "src.zip"),
		filepath.Join("server", "assets", "fs", targetOS, targetArch, "go.zip"),
		filepath.Join("server", "assets", "fs", targetOS, targetArch, "garble"),
	}
	for _, relativePath := range required {
		if _, err := os.Stat(filepath.Join(repoPath, relativePath)); err != nil {
			return fmt.Errorf("server build asset %q is unavailable: %w; run make once to populate embedded assets or pass --server", relativePath, err)
		}
	}
	return nil
}

func newTestLayout() (*testLayout, error) {
	workDir, err := os.MkdirTemp("", "sliver-controlflow-e2e-")
	if err != nil {
		return nil, fmt.Errorf("create integration work directory: %w", err)
	}
	layout := &testLayout{
		workDir:        workDir,
		serverRoot:     filepath.Join(workDir, "server-root"),
		clientRoot:     filepath.Join(workDir, "client-root"),
		homeDir:        filepath.Join(workDir, "home"),
		tempDir:        filepath.Join(workDir, "tmp"),
		logDir:         filepath.Join(workDir, "logs"),
		artifactDir:    filepath.Join(workDir, "artifacts"),
		garbleDebugDir: filepath.Join(workDir, "garble-debug"),
	}
	for _, dir := range []string{
		layout.serverRoot,
		layout.clientRoot,
		layout.homeDir,
		layout.tempDir,
		layout.logDir,
		layout.artifactDir,
	} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			_ = os.RemoveAll(workDir)
			return nil, fmt.Errorf("create isolated directory %q: %w", dir, err)
		}
	}
	return layout, nil
}

func buildServer(ctx context.Context, repoPath string, layout *testLayout, isolatedEnv []string) (string, error) {
	binDir := filepath.Join(layout.workDir, "bin")
	goCache := filepath.Join(layout.workDir, "server-build-cache")
	goPath := filepath.Join(layout.workDir, "server-build-gopath")
	goModCache := filepath.Join(layout.workDir, "server-build-modcache")
	for _, dir := range []string{binDir, goCache, goPath, goModCache} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return "", fmt.Errorf("create server build directory: %w", err)
		}
	}
	serverPath := filepath.Join(binDir, "sliver-server")
	buildEnv := environmentWith(isolatedEnv, map[string]string{
		"CGO_ENABLED": "0",
		"GOARCH":      targetArch,
		"GOCACHE":     goCache,
		"GOENV":       "off",
		"GOFLAGS":     "",
		"GOMODCACHE":  goModCache,
		"GOOS":        targetOS,
		"GOPATH":      goPath,
		"GOWORK":      "off",
	})
	output, err := runOwnedCommand(
		ctx,
		repoPath,
		buildEnv,
		"go",
		"build",
		"-buildvcs=false",
		"-mod=vendor",
		"-trimpath",
		"-tags=go_sqlite,server",
		"-o", serverPath,
		"./server",
	)
	if err != nil {
		return "", fmt.Errorf("build Sliver server: %w\n%s", err, output)
	}
	return serverPath, nil
}

func verifyCompilerCapability(ctx context.Context, rpc rpcpb.SliverRPCClient) error {
	compiler, err := rpc.GetCompiler(ctx, &commonpb.Empty{})
	if err != nil {
		return fmt.Errorf("query server compiler: %w", err)
	}
	hasCapability := false
	for _, capability := range compiler.Capabilities {
		if capability == controlFlowCapability {
			hasCapability = true
			break
		}
	}
	if !hasCapability {
		return fmt.Errorf("server does not advertise required capability %q", controlFlowCapability)
	}
	hasTarget := false
	for _, target := range compiler.Targets {
		if target.GOOS == targetOS && target.GOARCH == targetArch && target.Format == clientpb.OutputFormat_EXECUTABLE {
			hasTarget = true
			break
		}
	}
	if !hasTarget {
		return fmt.Errorf("server compiler does not advertise %s/%s executable support", targetOS, targetArch)
	}
	fmt.Printf("Server advertises %s and %s/%s executable support\n", controlFlowCapability, targetOS, targetArch)
	return nil
}

func generateArtifact(
	ctx context.Context,
	timeout time.Duration,
	rpc rpcpb.SliverRPCClient,
	layout *testLayout,
	implantPort int,
	name string,
	isBeacon bool,
) (*generatedArtifact, error) {

	config := &clientpb.ImplantConfig{
		GOOS:                targetOS,
		GOARCH:              targetArch,
		TemplateName:        "sliver",
		Debug:               false,
		ObfuscateSymbols:    true,
		ControlFlow:         clientpb.ControlFlowPolicy_CONTROL_FLOW_BALANCED_V1,
		IsBeacon:            isBeacon,
		BeaconInterval:      int64(2 * time.Second),
		BeaconJitter:        0,
		Format:              clientpb.OutputFormat_EXECUTABLE,
		C2:                  []*clientpb.ImplantC2{{URL: fmt.Sprintf("mtls://127.0.0.1:%d", implantPort)}},
		IncludeMTLS:         true,
		HTTPC2ConfigName:    consts.DefaultC2Profile,
		ConnectionStrategy:  "rd",
		ReconnectInterval:   int64(time.Second),
		PollTimeout:         int64(time.Second),
		MaxConnectionErrors: 10,
		NetGoEnabled:        true,
	}

	buildCtx, buildCancel := context.WithTimeout(ctx, timeout)
	generated, err := rpc.Generate(buildCtx, &clientpb.GenerateReq{Name: name, Config: config})
	buildCancel()
	if err != nil {
		return nil, err
	}
	if generated.File == nil || generated.File.Name == "" || len(generated.File.Data) == 0 {
		return nil, errors.New("Sliver returned an empty executable")
	}
	if generated.ImplantName != name {
		return nil, fmt.Errorf("generated implant name = %q, want %q", generated.ImplantName, name)
	}

	if err := verifyRenderedControlFlow(layout.serverRoot, generated.ImplantName); err != nil {
		return nil, err
	}
	transformedFiles, err := verifyGarbleControlFlowDebugDir(layout.garbleDebugDir)
	if err != nil {
		return nil, fmt.Errorf("inspect direct Garble transformation output for %s: %w", name, err)
	}
	fmt.Printf("Verified three exact rendered directives and %d transformed %s files for %s\n", transformedFiles, controlFlowGeneratedFile, name)

	artifactPath := filepath.Join(layout.artifactDir, filepath.Base(generated.File.Name))
	if err := os.WriteFile(artifactPath, generated.File.Data, 0o700); err != nil {
		return nil, fmt.Errorf("save generated executable: %w", err)
	}
	return &generatedArtifact{
		name: generated.ImplantName,
		path: artifactPath,
		data: generated.File.Data,
	}, nil
}

func verifySessionPing(ctx context.Context, rpc rpcpb.SliverRPCClient, sessionID string) error {
	if sessionID == "" {
		return errors.New("SessionOpened event has an empty session ID")
	}
	response, err := rpc.Ping(ctx, &sliverpb.Ping{
		Nonce: sessionPingNonce,
		Request: &commonpb.Request{
			SessionID: sessionID,
			Timeout:   int64(30 * time.Second),
		},
	})
	if err != nil {
		return fmt.Errorf("synchronous session Ping: %w", err)
	}
	if response.Nonce != sessionPingNonce {
		return fmt.Errorf("session Ping nonce = %d, want %d", response.Nonce, sessionPingNonce)
	}
	return nil
}

func verifySessionFilteredLs(ctx context.Context, rpc rpcpb.SliverRPCClient, sessionID string, tempDir string) error {
	const proofFileName = "controlflow-filter-proof.txt"
	proofPath := filepath.Join(tempDir, proofFileName)
	if err := os.WriteFile(proofPath, []byte("control-flow directory filter proof\n"), 0o600); err != nil {
		return fmt.Errorf("create filtered Ls proof file: %w", err)
	}
	wildcardPath := filepath.Join(tempDir, "controlflow-filter-*")
	listing, err := rpc.Ls(ctx, &sliverpb.LsReq{
		Path: wildcardPath,
		Request: &commonpb.Request{
			SessionID: sessionID,
			Timeout:   int64(30 * time.Second),
		},
	})
	if err != nil {
		return fmt.Errorf("filtered session Ls(%q): %w", wildcardPath, err)
	}
	if listing.Response != nil && listing.Response.Err != "" {
		return fmt.Errorf("filtered session Ls(%q) returned implant error: %s", wildcardPath, listing.Response.Err)
	}
	if !listing.Exists {
		return fmt.Errorf("filtered session Ls(%q) reported that its directory does not exist", wildcardPath)
	}
	foundProof := false
	for _, file := range listing.Files {
		if file.Name == proofFileName {
			foundProof = true
			break
		}
	}
	if !foundProof {
		return fmt.Errorf("filtered session Ls(%q) did not return proof file %q", wildcardPath, proofFileName)
	}
	fmt.Printf(
		"Verified transformed determineDirPathFilter via Ls(%q): matched %s in %s (%d result(s))\n",
		wildcardPath,
		proofFileName,
		listing.Path,
		len(listing.Files),
	)
	return nil
}

func verifyBeaconPing(ctx context.Context, rpc rpcpb.SliverRPCClient, process *managedProcess, beaconID string) error {
	if beaconID == "" {
		return errors.New("BeaconRegistered event has an empty beacon ID")
	}
	response, err := rpc.Ping(ctx, &sliverpb.Ping{
		Nonce: beaconPingNonce,
		Request: &commonpb.Request{
			Async:    true,
			BeaconID: beaconID,
			Timeout:  int64(30 * time.Second),
		},
	})
	if err != nil {
		return fmt.Errorf("queue asynchronous beacon Ping: %w", err)
	}
	if response.Response == nil || response.Response.TaskID == "" {
		return fmt.Errorf("asynchronous beacon Ping returned no task ID: %#v", response.Response)
	}
	task, err := waitForCompletedBeaconTask(ctx, rpc, process, response.Response.TaskID)
	if err != nil {
		return err
	}
	if len(task.Response) == 0 {
		return fmt.Errorf("completed beacon Ping task %s has an empty response", task.ID)
	}
	pong := &sliverpb.Ping{}
	if err := proto.Unmarshal(task.Response, pong); err != nil {
		return fmt.Errorf("unmarshal completed beacon Ping response: %w", err)
	}
	if pong.Nonce != beaconPingNonce {
		return fmt.Errorf("beacon Ping nonce = %d, want %d", pong.Nonce, beaconPingNonce)
	}
	return nil
}

func waitForCompletedBeaconTask(
	ctx context.Context,
	rpc rpcpb.SliverRPCClient,
	process *managedProcess,
	taskID string,
) (*clientpb.BeaconTask, error) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for beacon Ping task %s: %w", taskID, ctx.Err())
		case <-process.done:
			return nil, process.failure("beacon exited before its Ping task completed")
		case <-ticker.C:
			pollCtx, pollCancel := context.WithTimeout(ctx, 3*time.Second)
			task, err := rpc.GetBeaconTaskContent(pollCtx, &clientpb.BeaconTask{ID: taskID})
			pollCancel()
			if err != nil {
				continue
			}
			switch task.State {
			case "completed":
				return task, nil
			case "failed", "canceled":
				return nil, fmt.Errorf("beacon Ping task %s entered terminal state %q", taskID, task.State)
			}
		}
	}
}

func twoUnusedTCPPorts() (int, int, error) {
	first, err := unusedTCPPort()
	if err != nil {
		return 0, 0, err
	}
	for attempts := 0; attempts < 8; attempts++ {
		second, err := unusedTCPPort()
		if err != nil {
			return 0, 0, err
		}
		if first != second {
			return first, second, nil
		}
	}
	return 0, 0, errors.New("could not select two distinct TCP ports")
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
		connection, err := net.DialTimeout("tcp", address, listenerDialTimeout)
		if err == nil {
			_ = connection.Close()
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-process.done:
			return process.failure("Sliver daemon exited before accepting multiplayer connections")
		case <-ticker.C:
		}
	}
}

func setProcessEnvironment(values map[string]string) (func(), error) {
	type previousValue struct {
		value string
		set   bool
	}
	previous := make(map[string]previousValue, len(values))
	for name, value := range values {
		oldValue, set := os.LookupEnv(name)
		previous[name] = previousValue{value: oldValue, set: set}
		if err := os.Setenv(name, value); err != nil {
			for restoreName, restoreValue := range previous {
				if restoreValue.set {
					_ = os.Setenv(restoreName, restoreValue.value)
				} else {
					_ = os.Unsetenv(restoreName)
				}
			}
			return nil, err
		}
	}
	return func() {
		for name, value := range previous {
			if value.set {
				_ = os.Setenv(name, value.value)
			} else {
				_ = os.Unsetenv(name)
			}
		}
	}, nil
}

func environmentWith(environment []string, values map[string]string) []string {
	updated := append([]string{}, environment...)
	for name, value := range values {
		updated = environmentValue(updated, name, value)
	}
	return updated
}

func environmentValue(environment []string, name string, value string) []string {
	prefix := name + "="
	updated := make([]string, 0, len(environment)+1)
	for _, entry := range environment {
		if !strings.HasPrefix(entry, prefix) {
			updated = append(updated, entry)
		}
	}
	return append(updated, prefix+value)
}

func runOwnedCommand(ctx context.Context, dir string, env []string, path string, args ...string) (string, error) {
	cmd := exec.Command(path, args...)
	cmd.Dir = dir
	cmd.Env = env
	var output strings.Builder
	cmd.Stdout = &output
	cmd.Stderr = &output
	prepareCommand(cmd)
	if err := cmd.Start(); err != nil {
		return output.String(), err
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case err := <-done:
		return strings.TrimSpace(output.String()), err
	case <-ctx.Done():
		terminateProcessTree(cmd)
		select {
		case <-done:
		case <-time.After(cleanupGraceTimeout):
			killProcessTree(cmd)
			select {
			case <-done:
			case <-time.After(cleanupProcessTimeout):
			}
		}
		return strings.TrimSpace(output.String()), ctx.Err()
	}
}
