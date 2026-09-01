package e2e

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	clientassets "github.com/bishopfox/sliver/client/assets"
	consts "github.com/bishopfox/sliver/client/constants"
	clienttransport "github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	e2ecoverage "github.com/bishopfox/sliver/test/e2e/coverage"
	"github.com/bishopfox/sliver/test/e2e/rportfwdcoverage"
	"github.com/bishopfox/sliver/test/e2e/shellcodecoverage"
	"google.golang.org/protobuf/proto"
)

const (
	operatorName               = "e2e-operator"
	suiteScopeComprehensive    = "comprehensive"
	suiteScopeRportFwd         = "rportfwd"
	processLogTailBytes        = 1024 * 1024
	commandFailureLogTailBytes = 64 * 1024
	listenerPollInterval       = 250 * time.Millisecond
	listenerDialTimeout        = 500 * time.Millisecond
	cleanupGraceTimeout        = 2 * time.Second
	cleanupProcessTimeout      = 10 * time.Second
)

var transportOrder = []string{"mtls", "wg", "http"}

type options struct {
	repoPath       string
	serverPath     string
	serverArch     string
	targetOS       string
	targetArch     string
	resultsDir     string
	transportCSV   string
	modeCSV        string
	suiteScope     string
	transports     []string
	modes          []string
	timeout        time.Duration
	startupTimeout time.Duration
	connectTimeout time.Duration
	commandTimeout time.Duration
	beaconInterval time.Duration
	sgnSamples     int
	implantDebug   bool
}

type suite struct {
	t       *testing.T
	opts    options
	ctx     context.Context
	cancel  context.CancelFunc
	workDir string

	serverRoot string
	clientRoot string
	homeDir    string
	serverEnv  []string
	server     *managedProcess
	serverLog  string

	rpc              rpcpb.SliverRPCClient
	closeGRPC        func()
	hub              *eventHub
	coverage         *e2ecoverage.Recorder
	rportfwdCoverage *rportfwdcoverage.Recorder
	listeners        map[string]*listener
	armoryOnce       sync.Once
	armory           *armoryAssets
	armoryErr        error
	nativeBOFOnce    sync.Once
	nativeBOF        *nativeBOFAssets
	nativeBOFErr     error
	closeOnce        sync.Once
}

type listener struct {
	transport string
	jobID     uint32
	c2URL     string
	port      int
	nport     int
	keyPort   int
}

type implantTarget struct {
	session *clientpb.Session
	beacon  *clientpb.Beacon
}

func (target implantTarget) mode() string {
	if target.beacon != nil {
		return "beacon"
	}
	return "session"
}

func (target implantTarget) id() string {
	if target.beacon != nil {
		return target.beacon.ID
	}
	if target.session != nil {
		return target.session.ID
	}
	return ""
}

func (target implantTarget) name() string {
	if target.beacon != nil {
		return target.beacon.Name
	}
	if target.session != nil {
		return target.session.Name
	}
	return ""
}

func (target implantTarget) request(timeout time.Duration) *commonpb.Request {
	requestTimeout := timeout - time.Second
	if requestTimeout <= 0 {
		requestTimeout = timeout
	}
	request := &commonpb.Request{Timeout: int64(requestTimeout)}
	if target.beacon != nil {
		request.Async = true
		request.BeaconID = target.beacon.ID
	} else if target.session != nil {
		request.SessionID = target.session.ID
	}
	return request
}

type managedProcess struct {
	cmd      *exec.Cmd
	done     chan struct{}
	err      error
	logPath  string
	tree     processTree
	stopOnce sync.Once
	stopErr  error
}

func newSuite(t *testing.T, opts options, recordCommandCoverage bool) (*suite, error) {
	if err := validateOptions(&opts); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), opts.timeout)
	workDir, err := os.MkdirTemp("", "sliver-comprehensive-e2e-")
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create isolated E2E directory: %w", err)
	}

	s := &suite{
		t:         t,
		opts:      opts,
		ctx:       ctx,
		cancel:    cancel,
		workDir:   workDir,
		listeners: map[string]*listener{},
	}
	if recordCommandCoverage {
		s.coverage, err = e2ecoverage.NewRecorder(e2ecoverage.Target{OS: opts.targetOS, Arch: opts.targetArch})
		if err != nil {
			s.close()
			return nil, fmt.Errorf("initialize E2E coverage recorder: %w", err)
		}
	}
	s.rportfwdCoverage, err = rportfwdcoverage.NewRecorder(e2ecoverage.Target{OS: opts.targetOS, Arch: opts.targetArch})
	if err != nil {
		s.close()
		return nil, fmt.Errorf("initialize reverse-port-forward E2E coverage recorder: %w", err)
	}
	if s.opts.resultsDir == "" {
		s.opts.resultsDir, err = os.MkdirTemp("", "sliver-comprehensive-e2e-results-")
		if err != nil {
			s.close()
			return nil, fmt.Errorf("create persistent E2E results directory: %w", err)
		}
		s.t.Logf("No -results directory supplied; preserving reports in %s", s.opts.resultsDir)
	}
	if err := os.MkdirAll(s.opts.resultsDir, 0o700); err != nil {
		s.close()
		return nil, fmt.Errorf("create E2E results directory: %w", err)
	}
	if err := s.startServer(); err != nil {
		s.close()
		return nil, err
	}
	return s, nil
}

func (s *suite) writeCoverage() error {
	var writeErrors []error
	if s.rportfwdCoverage != nil {
		paths, err := s.rportfwdCoverage.Write(s.opts.resultsDir)
		if err != nil {
			writeErrors = append(writeErrors, fmt.Errorf("write reverse-port-forward E2E coverage: %w", err))
		} else {
			s.t.Logf("Wrote reverse-port-forward E2E coverage reports %s and %s", paths.JSON, paths.Markdown)
			if err := s.rportfwdCoverage.ValidateComplete(s.opts.transports); err != nil {
				writeErrors = append(writeErrors, err)
			}
		}
	}
	if s.coverage != nil {
		paths, err := s.coverage.Write(s.opts.resultsDir)
		if err != nil {
			writeErrors = append(writeErrors, fmt.Errorf("write E2E coverage: %w", err))
		} else {
			s.t.Logf("Wrote E2E coverage reports %s and %s", paths.JSON, paths.Markdown)
		}
	}
	return errors.Join(writeErrors...)
}

func validateOptions(opts *options) error {
	if opts.serverPath == "" {
		return errors.New("-server is required")
	}

	var err error
	opts.repoPath, err = filepath.Abs(opts.repoPath)
	if err != nil {
		return fmt.Errorf("resolve repository path: %w", err)
	}
	opts.serverPath, err = filepath.Abs(opts.serverPath)
	if err != nil {
		return fmt.Errorf("resolve server path: %w", err)
	}
	if opts.resultsDir != "" {
		opts.resultsDir, err = filepath.Abs(opts.resultsDir)
		if err != nil {
			return fmt.Errorf("resolve results directory: %w", err)
		}
	}
	if stat, err := os.Stat(opts.serverPath); err != nil {
		return fmt.Errorf("stat server executable: %w", err)
	} else if stat.IsDir() {
		return fmt.Errorf("server executable %q is a directory", opts.serverPath)
	}
	if _, err := os.Stat(filepath.Join(opts.repoPath, "go.mod")); err != nil {
		return fmt.Errorf("repository does not contain go.mod: %w", err)
	}

	opts.targetOS = strings.ToLower(strings.TrimSpace(opts.targetOS))
	opts.targetArch = strings.ToLower(strings.TrimSpace(opts.targetArch))
	opts.serverArch = strings.ToLower(strings.TrimSpace(opts.serverArch))
	opts.suiteScope, err = normalizeSuiteScope(opts.suiteScope)
	if err != nil {
		return err
	}
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
		return fmt.Errorf("target %s is not in the comprehensive E2E matrix", target)
	}
	if runtime.GOOS != opts.targetOS || runtime.GOARCH != opts.targetArch {
		return fmt.Errorf("E2E driver must run as the implant target: driver is %s/%s, target is %s", runtime.GOOS, runtime.GOARCH, target)
	}
	if opts.serverArch != "386" && opts.serverArch != "amd64" && opts.serverArch != "arm64" {
		return fmt.Errorf("unsupported expected server architecture %q", opts.serverArch)
	}
	if opts.timeout <= 0 || opts.startupTimeout <= 0 || opts.connectTimeout <= 0 || opts.commandTimeout <= 0 || opts.beaconInterval <= 0 {
		return errors.New("all E2E timeouts and intervals must be positive")
	}
	if opts.sgnSamples < shellcodecoverage.MinimumSGNSamples {
		return fmt.Errorf("shellcode SGN samples must be at least %d", shellcodecoverage.MinimumSGNSamples)
	}
	opts.transports, err = parseSelection(opts.transportCSV, transportOrder, "transport")
	if err != nil {
		return err
	}
	opts.modes, err = parseSelection(opts.modeCSV, []string{"session", "beacon"}, "implant mode")
	if err != nil {
		return err
	}
	opts.modes = modesForSuiteScope(opts.suiteScope, opts.modes)
	return nil
}

func normalizeSuiteScope(value string) (string, error) {
	scope := strings.TrimSpace(value)
	if scope != suiteScopeComprehensive && scope != suiteScopeRportFwd {
		return "", fmt.Errorf("unknown E2E suite scope %q (want %s or %s)", scope, suiteScopeComprehensive, suiteScopeRportFwd)
	}
	return scope, nil
}

func modesForSuiteScope(scope string, modes []string) []string {
	if scope == suiteScopeRportFwd {
		// Reverse port forwarding is an interactive session feature. A focused
		// run intentionally avoids generating beacons even when callers retain
		// the comprehensive suite's default selector.
		return []string{"session"}
	}
	return modes
}

func parseSelection(value string, allowed []string, label string) ([]string, error) {
	wanted := map[string]bool{}
	for _, item := range strings.Split(value, ",") {
		item = strings.ToLower(strings.TrimSpace(item))
		if item == "" {
			continue
		}
		wanted[item] = true
	}
	if len(wanted) == 0 {
		return nil, fmt.Errorf("at least one %s is required", label)
	}
	result := make([]string, 0, len(wanted))
	for _, item := range allowed {
		if wanted[item] {
			result = append(result, item)
			delete(wanted, item)
		}
	}
	if len(wanted) != 0 {
		unknown := make([]string, 0, len(wanted))
		for item := range wanted {
			unknown = append(unknown, item)
		}
		sort.Strings(unknown)
		return nil, fmt.Errorf("unknown %s selection: %s", label, strings.Join(unknown, ", "))
	}
	return result, nil
}

func (s *suite) startServer() error {
	s.serverRoot = filepath.Join(s.workDir, "server")
	s.clientRoot = filepath.Join(s.workDir, "client")
	s.homeDir = filepath.Join(s.workDir, "home")
	for _, dir := range []string{s.serverRoot, s.clientRoot, s.homeDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create isolated directory %q: %w", dir, err)
		}
	}

	grpcPort, err := unusedTCPPort()
	if err != nil {
		return fmt.Errorf("select multiplayer port: %w", err)
	}
	s.serverEnv = os.Environ()
	for name, value := range map[string]string{
		"HOME":                   s.homeDir,
		"SLIVER_CLIENT_ROOT_DIR": s.clientRoot,
		"SLIVER_ROOT_DIR":        s.serverRoot,
		"USERPROFILE":            s.homeDir,
	} {
		s.serverEnv = envWith(s.serverEnv, name, value)
	}

	s.serverLog = filepath.Join(s.workDir, "sliver-server.log")
	s.server, err = startProcess(
		s.opts.serverPath,
		[]string{"daemon", "--lhost", "127.0.0.1", "--lport", fmt.Sprintf("%d", grpcPort), "--force"},
		s.opts.repoPath,
		s.serverEnv,
		s.serverLog,
	)
	if err != nil {
		return fmt.Errorf("start Sliver daemon: %w", err)
	}
	s.t.Logf("Started unmodified Sliver daemon on 127.0.0.1:%d", grpcPort)

	startupCtx, startupCancel := context.WithTimeout(s.ctx, s.opts.startupTimeout)
	err = waitForTCP(startupCtx, fmt.Sprintf("127.0.0.1:%d", grpcPort), s.server)
	startupCancel()
	if err != nil {
		return fmt.Errorf("wait for Sliver daemon: %w", err)
	}

	profilePath := filepath.Join(s.workDir, "e2e-operator.cfg")
	profileCtx, profileCancel := context.WithTimeout(s.ctx, 2*time.Minute)
	profileOutput, profileErr := runCommand(
		profileCtx,
		s.opts.repoPath,
		s.serverEnv,
		s.opts.serverPath,
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
	config, err := clientassets.ReadConfig(profilePath)
	if err != nil {
		return fmt.Errorf("read operator profile: %w", err)
	}
	if config.Operator != operatorName || config.LHost != "127.0.0.1" || config.LPort != grpcPort {
		return fmt.Errorf("operator profile mismatch: got operator=%q endpoint=%s:%d", config.Operator, config.LHost, config.LPort)
	}
	s.t.Logf("Saved and validated %s operator profile", operatorName)

	rpc, conn, err := clienttransport.MTLSConnect(config)
	if err != nil {
		return fmt.Errorf("connect to multiplayer gRPC: %w", err)
	}
	s.rpc = rpc
	s.closeGRPC = func() { clienttransport.CloseGRPCConnection(conn) }

	version, err := s.rpc.GetVersion(s.ctx, &commonpb.Empty{})
	if err != nil {
		return fmt.Errorf("query Sliver server version: %w", err)
	}
	if version.OS != s.opts.targetOS || version.Arch != s.opts.serverArch {
		return fmt.Errorf("server host mismatch: got %s/%s, want %s/%s", version.OS, version.Arch, s.opts.targetOS, s.opts.serverArch)
	}
	s.t.Logf("Connected custom E2E client to Sliver server %s/%s over multiplayer gRPC", version.OS, version.Arch)

	events, err := s.rpc.Events(s.ctx, &commonpb.Empty{})
	if err != nil {
		return fmt.Errorf("subscribe to Sliver events: %w", err)
	}
	s.hub = newEventHub(events)
	return nil
}

func (s *suite) run() error {
	if err := s.startListeners(); err != nil {
		return err
	}

	var runErrors []error
	for _, transport := range s.opts.transports {
		listener := s.listeners[transport]
		for _, mode := range s.opts.modes {
			beacon := mode == "beacon"
			if err := s.runImplant(listener, beacon); err != nil {
				runErrors = append(runErrors, err)
			}
		}
	}
	return errors.Join(runErrors...)
}

func (s *suite) startListeners() error {
	if s.transportSelected("mtls") {
		mtlsPort, err := unusedTCPPort()
		if err != nil {
			return fmt.Errorf("select mTLS port: %w", err)
		}
		mtlsJob, err := s.rpc.StartMTLSListener(s.ctx, &clientpb.MTLSListenerReq{Host: "127.0.0.1", Port: uint32(mtlsPort)})
		if err != nil {
			return fmt.Errorf("start mTLS listener: %w", err)
		}
		s.listeners["mtls"] = &listener{transport: "mtls", jobID: mtlsJob.JobID, c2URL: fmt.Sprintf("mtls://127.0.0.1:%d", mtlsPort), port: mtlsPort}
		if err := s.waitForListener("mTLS", mtlsPort); err != nil {
			return err
		}
	}

	if s.transportSelected("wg") {
		wgPort, err := unusedUDPPort()
		if err != nil {
			return fmt.Errorf("select WireGuard port: %w", err)
		}
		wgNPort, err := unusedTCPPort()
		if err != nil {
			return fmt.Errorf("select WireGuard TCP comms port: %w", err)
		}
		wgKeyPort, err := unusedTCPPort()
		if err != nil {
			return fmt.Errorf("select WireGuard key exchange port: %w", err)
		}
		for wgKeyPort == wgNPort {
			wgKeyPort, err = unusedTCPPort()
			if err != nil {
				return fmt.Errorf("reselect WireGuard key exchange port: %w", err)
			}
		}
		wgJob, err := s.rpc.StartWGListener(s.ctx, &clientpb.WGListenerReq{Host: "127.0.0.1", Port: uint32(wgPort), NPort: uint32(wgNPort), KeyPort: uint32(wgKeyPort)})
		if err != nil {
			return fmt.Errorf("start WireGuard listener: %w", err)
		}
		s.listeners["wg"] = &listener{transport: "wg", jobID: wgJob.JobID, c2URL: fmt.Sprintf("wg://127.0.0.1:%d", wgPort), port: wgPort, nport: wgNPort, keyPort: wgKeyPort}
	}

	if s.transportSelected("http") {
		httpPort, err := unusedTCPPort()
		if err != nil {
			return fmt.Errorf("select HTTP port: %w", err)
		}
		httpJob, err := s.rpc.StartHTTPListener(s.ctx, &clientpb.HTTPListenerReq{
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
		s.listeners["http"] = &listener{transport: "http", jobID: httpJob.JobID, c2URL: fmt.Sprintf("http://127.0.0.1:%d?force-http=true&poll-timeout=1s", httpPort), port: httpPort}
		if err := s.waitForListener("HTTP", httpPort); err != nil {
			return err
		}
	}

	for _, name := range s.opts.transports {
		listener := s.listeners[name]
		s.t.Logf("Started %s listener job %d (%s)", name, listener.jobID, listener.c2URL)
	}
	return nil
}

func (s *suite) transportSelected(name string) bool {
	for _, selected := range s.opts.transports {
		if selected == name {
			return true
		}
	}
	return false
}

func (s *suite) waitForListener(name string, port int) error {
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()
	if err := waitForTCP(ctx, fmt.Sprintf("127.0.0.1:%d", port), s.server); err != nil {
		return fmt.Errorf("wait for %s listener: %w", name, err)
	}
	return nil
}

func (s *suite) runImplant(listener *listener, beacon bool) error {
	mode := "session"
	if beacon {
		mode = "beacon"
	}
	implantName := fmt.Sprintf("e2e-%s-%s-%s-%s", listener.transport, mode, s.opts.targetOS, s.opts.targetArch)

	config := &clientpb.ImplantConfig{
		GOOS:                s.opts.targetOS,
		GOARCH:              s.opts.targetArch,
		TemplateName:        "sliver",
		Debug:               s.opts.implantDebug,
		ObfuscateSymbols:    false,
		IsBeacon:            beacon,
		BeaconInterval:      int64(s.opts.beaconInterval),
		BeaconJitter:        0,
		Format:              clientpb.OutputFormat_EXECUTABLE,
		C2:                  []*clientpb.ImplantC2{{URL: listener.c2URL}},
		HTTPC2ConfigName:    consts.DefaultC2Profile,
		ConnectionStrategy:  "s",
		ReconnectInterval:   int64(time.Second),
		PollTimeout:         int64(time.Second),
		MaxConnectionErrors: 20,
		NetGoEnabled:        true,
	}
	switch listener.transport {
	case "mtls":
		config.IncludeMTLS = true
	case "wg":
		uniqueIP, err := s.rpc.GenerateUniqueIP(s.ctx, &commonpb.Empty{})
		if err != nil {
			return fmt.Errorf("generate WireGuard peer IP for %s: %w", implantName, err)
		}
		config.IncludeWG = true
		config.WGPeerTunIP = uniqueIP.IP
		config.WGKeyExchangePort = uint32(listener.keyPort)
		config.WGTcpCommsPort = uint32(listener.nport)
	case "http":
		config.IncludeHTTP = true
	default:
		return fmt.Errorf("unknown transport %q", listener.transport)
	}

	s.t.Logf("Generating %s/%s %s %s implant over %s", s.opts.targetOS, s.opts.targetArch, mode, implantName, listener.transport)
	generated, err := s.rpc.Generate(s.ctx, &clientpb.GenerateReq{Name: implantName, Config: config})
	if err != nil {
		return fmt.Errorf("generate %s: %w", implantName, err)
	}
	if generated.File == nil || generated.File.Name == "" || len(generated.File.Data) == 0 {
		return fmt.Errorf("generate %s returned an empty executable", implantName)
	}

	remoteRoot := filepath.Join(s.workDir, "targets", implantName)
	if err := createKnownTree(remoteRoot); err != nil {
		return fmt.Errorf("prepare known filesystem tree for %s: %w", implantName, err)
	}
	implantHome := filepath.Join(remoteRoot, "home")
	implantTemp := filepath.Join(remoteRoot, "tmp")
	for _, dir := range []string{implantHome, implantTemp} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("prepare isolated implant directory %s: %w", dir, err)
		}
	}
	implantPath := filepath.Join(remoteRoot, filepath.Base(generated.File.Name))
	if err := os.WriteFile(implantPath, generated.File.Data, 0o700); err != nil {
		return fmt.Errorf("save %s: %w", implantName, err)
	}
	if err := os.Chmod(implantPath, 0o700); err != nil {
		return fmt.Errorf("make %s executable: %w", implantName, err)
	}

	implantLog := filepath.Join(s.workDir, implantName+".log")
	implantEnv := sanitizedImplantEnv(os.Environ(), implantHome, implantTemp, implantName)
	cursor := s.hub.cursor()
	process, err := startProcess(implantPath, nil, remoteRoot, implantEnv, implantLog)
	if err != nil {
		return fmt.Errorf("start %s: %w", implantName, err)
	}
	defer func() {
		if err := process.stop(); err != nil {
			s.t.Errorf("stop %s: %v", implantName, err)
		}
	}()

	connectCtx, connectCancel := context.WithTimeout(s.ctx, s.opts.connectTimeout)
	go func() {
		select {
		case <-process.done:
			connectCancel()
		case <-connectCtx.Done():
		}
	}()
	target, err := s.waitForImplant(connectCtx, cursor, process, listener, generated.ImplantName, mode)
	connectCancel()
	if err != nil {
		return fmt.Errorf("wait for %s connection: %w\nimplant log:\n%s\nserver log:\n%s", implantName, err, readLogTail(implantLog), readLogTail(s.serverLog))
	}
	s.t.Logf("Verified %s %s connection %s over %s", mode, target.id(), s.opts.targetOS+"/"+s.opts.targetArch, listener.transport)

	var exerciseErrors []error
	if target.session != nil {
		exerciseErrors = appendIfError(exerciseErrors, s.exerciseReversePortForward(target, listener.transport))
	}
	if s.opts.suiteScope == suiteScopeComprehensive {
		exerciseErrors = appendIfError(exerciseErrors, s.exerciseCommands(target, remoteRoot, listener.transport))
	}
	if target.session != nil {
		exerciseErrors = appendIfError(exerciseErrors, s.exerciseReversePortForwardDisconnect(target, listener.transport, process))
	}
	if err := errors.Join(exerciseErrors...); err != nil {
		return fmt.Errorf(
			"exercise %s scope for %s over %s: %w\n%s",
			s.opts.suiteScope,
			mode,
			listener.transport,
			err,
			commandFailureDiagnostics(process, implantLog, s.serverLog),
		)
	}
	return nil
}

func createKnownTree(root string) error {
	knownNested := filepath.Join(root, "known", "nested")
	if err := os.MkdirAll(knownNested, 0o700); err != nil {
		return err
	}
	files := map[string]string{
		filepath.Join(root, "known", "seed.txt"):         "alpha\nbeta\ngamma\n",
		filepath.Join(knownNested, "child.txt"):          "child-marker\n",
		filepath.Join(knownNested, "another.log"):        "log-marker\n",
		filepath.Join(root, "outside-test-sentinel.txt"): "must-survive\n",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			return err
		}
	}
	return nil
}

func (s *suite) waitForImplant(ctx context.Context, cursor int, process *managedProcess, listener *listener, implantName string, mode string) (implantTarget, error) {
	expectedPID := int32(process.cmd.Process.Pid)
	for {
		event, next, err := s.hub.wait(ctx, cursor, func(event *clientpb.Event) bool {
			if event.EventType == consts.JobStoppedEvent && event.Job != nil && event.Job.ID == listener.jobID {
				return true
			}
			if mode == "session" {
				return event.EventType == consts.SessionOpenedEvent && event.Session != nil && event.Session.Name == implantName && event.Session.PID == expectedPID
			}
			if event.EventType != consts.BeaconRegisteredEvent {
				return false
			}
			beacon := &clientpb.Beacon{}
			return proto.Unmarshal(event.Data, beacon) == nil && beacon.Name == implantName && beacon.PID == expectedPID
		})
		cursor = next
		if err != nil {
			select {
			case <-process.done:
				return implantTarget{}, process.failure("implant exited before connecting")
			default:
				return implantTarget{}, err
			}
		}
		if event.EventType == consts.JobStoppedEvent {
			return implantTarget{}, fmt.Errorf("%s listener job %d stopped: %s", listener.transport, listener.jobID, event.Err)
		}
		if mode == "session" {
			if event.Session.OS != s.opts.targetOS || event.Session.Arch != s.opts.targetArch {
				return implantTarget{}, fmt.Errorf("session target mismatch: got %s/%s", event.Session.OS, event.Session.Arch)
			}
			if err := validateConnectedTransport(event.Session.Transport, event.Session.ActiveC2, listener.transport); err != nil {
				return implantTarget{}, fmt.Errorf("session connection mismatch: %w", err)
			}
			return implantTarget{session: event.Session}, nil
		}
		beacon := &clientpb.Beacon{}
		if err := proto.Unmarshal(event.Data, beacon); err != nil {
			return implantTarget{}, fmt.Errorf("decode beacon registration: %w", err)
		}
		if beacon.OS != s.opts.targetOS || beacon.Arch != s.opts.targetArch {
			return implantTarget{}, fmt.Errorf("beacon target mismatch: got %s/%s", beacon.OS, beacon.Arch)
		}
		if err := validateConnectedTransport(beacon.Transport, beacon.ActiveC2, listener.transport); err != nil {
			return implantTarget{}, fmt.Errorf("beacon connection mismatch: %w", err)
		}
		return implantTarget{beacon: beacon}, nil
	}
}

func validateConnectedTransport(transport string, activeC2 string, expected string) error {
	wantTransport := expected
	if expected == "http" {
		wantTransport = "http(s)"
	}
	if !strings.EqualFold(strings.TrimSpace(transport), wantTransport) {
		return fmt.Errorf("transport got %q, want %q", transport, wantTransport)
	}
	if strings.TrimSpace(activeC2) == "" {
		return nil
	}
	scheme, _, found := strings.Cut(strings.ToLower(strings.TrimSpace(activeC2)), "://")
	if !found || scheme != expected {
		return fmt.Errorf("ActiveC2 got %q, want %s scheme", activeC2, expected)
	}
	return nil
}

func (s *suite) close() {
	s.closeOnce.Do(func() {
		if s.rpc != nil {
			names := make([]string, 0, len(s.listeners))
			for name := range s.listeners {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				_, err := s.rpc.KillJob(ctx, &clientpb.KillJobReq{ID: s.listeners[name].jobID})
				cancel()
				if err != nil {
					s.t.Errorf("stop %s listener job %d: %v", name, s.listeners[name].jobID, err)
				}
			}
		}
		if s.cancel != nil {
			s.cancel()
		}
		if s.closeGRPC != nil {
			s.closeGRPC()
		}
		if s.server != nil {
			if err := s.server.stop(); err != nil {
				s.t.Errorf("stop Sliver daemon: %v", err)
			}
		}
		if s.workDir != "" {
			if err := os.RemoveAll(s.workDir); err != nil {
				s.t.Errorf("remove isolated E2E directory %s: %v", s.workDir, err)
			}
		}
	})
}

func unusedTCPPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func unusedUDPPort() (int, error) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).Port, nil
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
			return process.failure("process exited before accepting connections")
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
	tree, err := attachProcessTree(cmd)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		_ = logFile.Close()
		return nil, fmt.Errorf("attach process %d to managed tree: %w", cmd.Process.Pid, err)
	}
	process := &managedProcess{cmd: cmd, done: make(chan struct{}), logPath: logPath, tree: tree}
	go func() {
		process.err = cmd.Wait()
		_ = logFile.Close()
		close(process.done)
	}()
	return process, nil
}

func (process *managedProcess) stop() error {
	process.stopOnce.Do(func() {
		process.stopErr = process.stopProcess()
	})
	return process.stopErr
}

func (process *managedProcess) stopProcess() (stopErr error) {
	defer func() {
		if err := closeProcessTree(process.tree); err != nil {
			stopErr = errors.Join(stopErr, fmt.Errorf("close process tree %d (%s): %w", process.cmd.Process.Pid, process.cmd.Path, err))
		}
	}()
	select {
	case <-process.done:
		if err := killProcessTree(process.tree, process.cmd); err != nil {
			return fmt.Errorf("clean descendants of exited process %d (%s): %w", process.cmd.Process.Pid, process.cmd.Path, err)
		}
		return nil
	default:
	}
	if err := terminateProcessTree(process.tree, process.cmd); err != nil {
		return fmt.Errorf("terminate process tree %d (%s): %w", process.cmd.Process.Pid, process.cmd.Path, err)
	}
	select {
	case <-process.done:
		if err := killProcessTree(process.tree, process.cmd); err != nil {
			return fmt.Errorf("clean descendants of process %d (%s): %w", process.cmd.Process.Pid, process.cmd.Path, err)
		}
		return nil
	case <-time.After(cleanupGraceTimeout):
	}
	if err := killProcessTree(process.tree, process.cmd); err != nil {
		return fmt.Errorf("kill process tree %d (%s): %w", process.cmd.Process.Pid, process.cmd.Path, err)
	}
	select {
	case <-process.done:
		return nil
	case <-time.After(cleanupProcessTimeout):
		return fmt.Errorf("process %d (%s) remained alive after terminate and kill", process.cmd.Process.Pid, process.cmd.Path)
	}
}

func (process *managedProcess) failure(message string) error {
	processErr := process.err
	if processErr == nil {
		processErr = errors.New("process exited")
	}
	return fmt.Errorf("%s: %w\n%s", message, processErr, readLogTail(process.logPath))
}

func commandFailureDiagnostics(process *managedProcess, implantLog string, serverLog string) string {
	return fmt.Sprintf(
		"implant process status: %s\nimplant log tail (last %d bytes):\n%s\nserver log tail (last %d bytes):\n%s",
		managedProcessStatus(process),
		commandFailureLogTailBytes,
		readLogTailBytes(implantLog, commandFailureLogTailBytes),
		commandFailureLogTailBytes,
		readLogTailBytes(serverLog, commandFailureLogTailBytes),
	)
}

func managedProcessStatus(process *managedProcess) string {
	if process == nil {
		return "unavailable"
	}
	select {
	case <-process.done:
		if process.err != nil {
			return fmt.Sprintf("exited with error: %v", process.err)
		}
		return "exited without error"
	default:
		return "remains running"
	}
}

func readLogTail(path string) string {
	return readLogTailBytes(path, processLogTailBytes)
}

func readLogTailBytes(path string, limit int64) string {
	if limit <= 0 {
		return "(log tail disabled)"
	}
	logFile, err := os.Open(path)
	if err != nil {
		return fmt.Sprintf("(could not read %s: %v)", path, err)
	}
	defer logFile.Close()

	end, err := logFile.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Sprintf("(could not inspect %s: %v)", path, err)
	}
	start := end - limit
	if start < 0 {
		start = 0
	}
	if _, err := logFile.Seek(start, io.SeekStart); err != nil {
		return fmt.Sprintf("(could not seek %s: %v)", path, err)
	}
	data, err := io.ReadAll(io.LimitReader(logFile, limit))
	if err != nil {
		return fmt.Sprintf("(could not read %s: %v)", path, err)
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
	updated := make([]string, 0, len(env)+1)
	for _, entry := range env {
		entryName, _, found := strings.Cut(entry, "=")
		if !found || !strings.EqualFold(entryName, name) {
			updated = append(updated, entry)
		}
	}
	return append(updated, name+"="+value)
}

func sanitizedImplantEnv(hostEnv []string, home string, temp string, marker string) []string {
	allowed := map[string]bool{
		"COMSPEC": true, "PATH": true, "PATHEXT": true, "SYSTEMDRIVE": true,
		"SYSTEMROOT": true, "WINDIR": true,
	}
	result := make([]string, 0, len(allowed)+7)
	for _, entry := range hostEnv {
		name, value, found := strings.Cut(entry, "=")
		if found && allowed[strings.ToUpper(name)] {
			result = envWith(result, strings.ToUpper(name), value)
		}
	}
	for name, value := range map[string]string{
		"HOME": home, "USERPROFILE": home, "TMP": temp, "TEMP": temp, "TMPDIR": temp,
		"SLIVER_E2E_PARENT_MARKER": marker, "SLIVER_E2E_MARKER": marker,
	} {
		result = envWith(result, name, value)
	}
	sort.Strings(result)
	return result
}
