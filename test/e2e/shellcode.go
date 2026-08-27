package e2e

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	e2ecoverage "github.com/bishopfox/sliver/test/e2e/coverage"
	"github.com/bishopfox/sliver/test/e2e/shellcodecoverage"
)

const shellcodeFailureDetailBytes = 16 * 1024

var shellcodeEncoderEnums = map[string]clientpb.ShellcodeEncoder{
	shellcodecoverage.EncoderNone:         clientpb.ShellcodeEncoder_NONE,
	shellcodecoverage.EncoderShikataGaNai: clientpb.ShellcodeEncoder_SHIKATA_GA_NAI,
	shellcodecoverage.EncoderXOR:          clientpb.ShellcodeEncoder_XOR,
	shellcodecoverage.EncoderXORDynamic:   clientpb.ShellcodeEncoder_XOR_DYNAMIC,
}

func (s *suite) runShellcode(recorder *shellcodecoverage.Recorder) error {
	target := e2ecoverage.Target{OS: s.opts.targetOS, Arch: s.opts.targetArch}
	if !shellcodeTargetSupported(target) {
		return fmt.Errorf("target %s/%s does not support shellcode generation", target.OS, target.Arch)
	}

	runnerPath, err := s.buildShellcodeRunner(s.ctx)
	if err != nil {
		recordErr := s.recordAllShellcodeFailures(recorder, fmt.Sprintf("build C runner: %v", err))
		return errors.Join(fmt.Errorf("build shellcode C runner: %w", err), recordErr)
	}
	s.t.Logf("Built %s/%s C shellcode runner %s", target.OS, target.Arch, filepath.Base(runnerPath))

	encoders, err := s.discoverShellcodeEncoders(target)
	if err != nil {
		recordErr := s.recordAllShellcodeFailures(recorder, fmt.Sprintf("discover shellcode encoders: %v", err))
		return errors.Join(err, recordErr)
	}

	if err := s.startListeners(); err != nil {
		recordErr := s.recordAllShellcodeFailures(recorder, fmt.Sprintf("start localhost listeners: %v", err))
		return errors.Join(err, recordErr)
	}

	var runErrors []error
	for _, transport := range s.opts.transports {
		listener := s.listeners[transport]
		for _, mode := range s.opts.modes {
			for _, compression := range shellcodecoverage.Compressions() {
				if err := s.runShellcodeBuild(recorder, runnerPath, listener, mode, compression, encoders); err != nil {
					runErrors = append(runErrors, err)
				}
			}
		}
	}
	return errors.Join(runErrors...)
}

func (s *suite) runShellcodeBuild(
	recorder *shellcodecoverage.Recorder,
	runnerPath string,
	listener *listener,
	mode string,
	compression string,
	encoders map[string]clientpb.ShellcodeEncoder,
) error {
	target := e2ecoverage.Target{OS: s.opts.targetOS, Arch: s.opts.targetArch}
	buildName := fmt.Sprintf("shellcode-%s-%s-%s-%s-%s", listener.transport, mode, compression, target.OS, target.Arch)
	config, err := s.shellcodeImplantConfig(listener, mode, compression)
	if err != nil {
		detail := fmt.Sprintf("prepare implant config: %v", err)
		recordErr := s.recordShellcodeBuildFailures(recorder, listener.transport, mode, compression, 0, detail)
		return errors.Join(fmt.Errorf("%s: %w", buildName, err), recordErr)
	}

	s.t.Logf("Generating %s shellcode with compression=%s", buildName, compression)
	generateStart := time.Now()
	generated, err := s.rpc.Generate(s.ctx, &clientpb.GenerateReq{Name: buildName, Config: config})
	generateDuration := time.Since(generateStart)
	if err != nil {
		detail := fmt.Sprintf("Generate RPC failed: %v", err)
		recordErr := s.recordShellcodeBuildFailures(recorder, listener.transport, mode, compression, generateDuration, detail)
		return errors.Join(fmt.Errorf("generate %s: %w", buildName, err), recordErr)
	}
	if generated.File == nil || generated.File.Name == "" || len(generated.File.Data) == 0 {
		err = errors.New("Generate RPC returned an empty .bin payload")
		recordErr := s.recordShellcodeBuildFailures(recorder, listener.transport, mode, compression, generateDuration, err.Error())
		return errors.Join(fmt.Errorf("generate %s: %w", buildName, err), recordErr)
	}
	if generated.ImplantName == "" {
		err = errors.New("Generate RPC returned an empty implant name")
		recordErr := s.recordShellcodeBuildFailures(recorder, listener.transport, mode, compression, generateDuration, err.Error())
		return errors.Join(fmt.Errorf("generate %s: %w", buildName, err), recordErr)
	}
	if !strings.EqualFold(filepath.Ext(generated.File.Name), ".bin") {
		err = fmt.Errorf("Generate RPC returned %q, want a .bin payload", generated.File.Name)
		recordErr := s.recordShellcodeBuildFailures(recorder, listener.transport, mode, compression, generateDuration, err.Error())
		return errors.Join(fmt.Errorf("generate %s: %w", buildName, err), recordErr)
	}

	var runErrors []error
	for _, encoder := range shellcodecoverage.Encoders() {
		if !shellcodecoverage.EncoderSupported(target, encoder) {
			continue
		}
		caseStart := time.Now()
		requiredSamples := shellcodeEncoderSamples(encoder, s.opts.sgnSamples)
		var encodeSample func([]byte) ([]byte, error)
		if encoder != shellcodecoverage.EncoderNone {
			encodeSample = func(payload []byte) ([]byte, error) {
				return s.encodeShellcode(encoder, encoders[encoder], payload)
			}
		}
		sampleResult, caseErr := runShellcodeSamples(
			generated.File.Data,
			requiredSamples,
			encodeSample,
			func(payload []byte, sample int, phase shellcodeExecutionPhase) error {
				return s.executeShellcodeCase(
					runnerPath,
					listener,
					mode,
					compression,
					encoder,
					generated.ImplantName,
					payload,
					sample,
					phase,
				)
			},
			func(sample int, payload []byte, payloadHash [sha256.Size]byte) {
				s.t.Logf(
					"Verified %s/%s %s %s shellcode over %s with compression=%s encoder=%s sample=%d/%d (%d bytes, sha256=%x)",
					target.OS,
					target.Arch,
					mode,
					generated.ImplantName,
					listener.transport,
					compression,
					encoder,
					sample,
					requiredSamples,
					len(payload),
					payloadHash,
				)
			},
		)

		status := e2ecoverage.StatusPass
		detail := ""
		if caseErr != nil {
			status = e2ecoverage.StatusFail
			detail = shellcodeFailureDetail(caseErr.Error())
		}
		recordErr := recorder.Add(shellcodecoverage.Observation{
			Transport:        listener.transport,
			ImplantMode:      mode,
			Compression:      compression,
			Encoder:          encoder,
			Status:           status,
			Duration:         generateDuration + time.Since(caseStart),
			Detail:           detail,
			PayloadBytes:     sampleResult.payloadBytes,
			RequiredSamples:  requiredSamples,
			CompletedSamples: sampleResult.completedSamples,
		})
		if caseErr != nil || recordErr != nil {
			resultErr := recordErr
			if caseErr != nil {
				resultErr = errors.Join(fmt.Errorf("%s encoder %s: %w", buildName, encoder, caseErr), recordErr)
			}
			runErrors = append(runErrors, resultErr)
		}
	}
	return errors.Join(runErrors...)
}

func (s *suite) shellcodeImplantConfig(listener *listener, mode string, compression string) (*clientpb.ImplantConfig, error) {
	compress := uint32(1)
	if compression == shellcodecoverage.CompressionAPLib {
		compress = 2
	}
	config := &clientpb.ImplantConfig{
		GOOS:                s.opts.targetOS,
		GOARCH:              s.opts.targetArch,
		TemplateName:        "sliver",
		Debug:               s.opts.implantDebug,
		ObfuscateSymbols:    false,
		IsBeacon:            mode == shellcodecoverage.ImplantModeBeacon,
		BeaconInterval:      int64(s.opts.beaconInterval),
		BeaconJitter:        0,
		Format:              clientpb.OutputFormat_SHELLCODE,
		IsShellcode:         true,
		Exports:             []string{"StartW"},
		ShellcodeConfig:     &clientpb.ShellcodeConfig{Compress: compress},
		C2:                  []*clientpb.ImplantC2{{URL: listener.c2URL}},
		HTTPC2ConfigName:    consts.DefaultC2Profile,
		ConnectionStrategy:  "s",
		ReconnectInterval:   int64(time.Second),
		PollTimeout:         int64(time.Second),
		MaxConnectionErrors: 20,
		NetGoEnabled:        true,
	}
	switch listener.transport {
	case shellcodecoverage.TransportMTLS:
		config.IncludeMTLS = true
	case shellcodecoverage.TransportWG:
		uniqueIP, err := s.rpc.GenerateUniqueIP(s.ctx, &commonpb.Empty{})
		if err != nil {
			return nil, fmt.Errorf("generate WireGuard peer IP: %w", err)
		}
		config.IncludeWG = true
		config.WGPeerTunIP = uniqueIP.IP
		config.WGKeyExchangePort = uint32(listener.keyPort)
		config.WGTcpCommsPort = uint32(listener.nport)
	case shellcodecoverage.TransportHTTP:
		config.IncludeHTTP = true
	default:
		return nil, fmt.Errorf("unknown transport %q", listener.transport)
	}
	return config, nil
}

func (s *suite) encodeShellcode(name string, encoder clientpb.ShellcodeEncoder, payload []byte) ([]byte, error) {
	response, err := s.rpc.ShellcodeEncoder(s.ctx, &clientpb.ShellcodeEncodeReq{
		Encoder:      encoder,
		Architecture: s.opts.targetArch,
		Iterations:   1,
		Data:         payload,
	})
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, errors.New("empty response")
	}
	if response.Response != nil && response.Response.Err != "" {
		return nil, errors.New(response.Response.Err)
	}
	if len(response.Data) == 0 {
		return nil, fmt.Errorf("encoder %s returned an empty payload", name)
	}
	return response.Data, nil
}

func (s *suite) executeShellcodeCase(
	runnerPath string,
	listener *listener,
	mode string,
	compression string,
	encoder string,
	implantName string,
	payload []byte,
	sample int,
	phase shellcodeExecutionPhase,
) error {
	caseName := strings.Join([]string{listener.transport, mode, compression, encoder}, "-")
	caseRoot := filepath.Join(
		s.workDir,
		"shellcode-cases",
		caseName,
		fmt.Sprintf("sample-%02d", sample),
		string(phase),
	)
	homeDir := filepath.Join(caseRoot, "home")
	tempDir := filepath.Join(caseRoot, "tmp")
	for _, dir := range []string{caseRoot, homeDir, tempDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create shellcode case directory %s: %w", dir, err)
		}
	}
	payloadPath := filepath.Join(caseRoot, "payload.bin")
	if err := os.WriteFile(payloadPath, payload, 0o600); err != nil {
		return fmt.Errorf("write shellcode payload: %w", err)
	}

	runnerLog := filepath.Join(caseRoot, "runner.log")
	runnerMarker := fmt.Sprintf("shellcode-%s-sample-%02d-%s", caseName, sample, phase)
	runnerEnv := sanitizedImplantEnv(os.Environ(), homeDir, tempDir, runnerMarker)
	cursor := s.hub.cursor()
	protection := shellcodeExecutionProtection(s.opts.targetArch, encoder)
	process, err := startProcess(runnerPath, []string{payloadPath, protection}, caseRoot, runnerEnv, runnerLog)
	if err != nil {
		return fmt.Errorf("start C shellcode runner: %w", err)
	}

	connectCtx, connectCancel := context.WithTimeout(s.ctx, s.opts.connectTimeout)
	go func() {
		select {
		case <-process.done:
			connectCancel()
		case <-connectCtx.Done():
		}
	}()
	_, waitErr := s.waitForImplant(connectCtx, cursor, process, listener, implantName, mode)
	connectCancel()
	stopErr := process.stop()
	if waitErr != nil {
		return errors.Join(
			fmt.Errorf("wait for shellcode callback: %w\nrunner log:\n%s\nserver log:\n%s", waitErr, readLogTail(runnerLog), readLogTail(s.serverLog)),
			stopErr,
		)
	}
	if stopErr != nil {
		return fmt.Errorf("stop C shellcode runner: %w", stopErr)
	}
	return nil
}

func shellcodeEncoderSamples(encoder string, sgnSamples int) int {
	if encoder == shellcodecoverage.EncoderShikataGaNai {
		return sgnSamples
	}
	return 1
}

type shellcodeSampleResult struct {
	completedSamples int
	payloadBytes     int64
}

type shellcodeExecutionPhase string

const (
	shellcodeExecutionPrimary          shellcodeExecutionPhase = "primary"
	shellcodeExecutionDiagnosticReplay shellcodeExecutionPhase = "diagnostic-replay"
)

func runShellcodeSamples(
	basePayload []byte,
	requiredSamples int,
	encode func([]byte) ([]byte, error),
	execute func([]byte, int, shellcodeExecutionPhase) error,
	verified func(int, []byte, [sha256.Size]byte),
) (shellcodeSampleResult, error) {
	result := shellcodeSampleResult{}
	seenPayloads := map[[sha256.Size]byte]int{}
	for sample := 1; sample <= requiredSamples; sample++ {
		payload := basePayload
		result.payloadBytes = 0
		if encode != nil {
			var err error
			payload, err = encode(payload)
			if err != nil {
				return result, fmt.Errorf("sample %d/%d ShellcodeEncoder RPC failed: %w", sample, requiredSamples, err)
			}
		}
		result.payloadBytes = int64(len(payload))

		payloadHash := sha256.Sum256(payload)
		if previous, duplicate := seenPayloads[payloadHash]; duplicate {
			return result, fmt.Errorf(
				"sample %d/%d duplicated sample %d payload sha256=%x",
				sample,
				requiredSamples,
				previous,
				payloadHash,
			)
		}
		seenPayloads[payloadHash] = sample
		primaryErr := execute(payload, sample, shellcodeExecutionPrimary)
		if primaryErr != nil {
			primaryFailure := fmt.Errorf("primary execution failed: %w", primaryErr)
			replayErr := execute(payload, sample, shellcodeExecutionDiagnosticReplay)
			var executionFailure error
			if replayErr == nil {
				executionFailure = errors.Join(
					errors.New("same-byte diagnostic replay succeeded; original failure remains authoritative"),
					primaryFailure,
				)
			} else {
				executionFailure = errors.Join(
					fmt.Errorf("same-byte diagnostic replay failed: %w", replayErr),
					primaryFailure,
				)
			}
			return result, fmt.Errorf(
				"sample %d/%d payload sha256=%x: %w",
				sample,
				requiredSamples,
				payloadHash,
				executionFailure,
			)
		}
		result.completedSamples++
		if verified != nil {
			verified(sample, payload, payloadHash)
		}
	}
	return result, nil
}

func (s *suite) discoverShellcodeEncoders(target e2ecoverage.Target) (map[string]clientpb.ShellcodeEncoder, error) {
	response, err := s.rpc.ShellcodeEncoderMap(s.ctx, &commonpb.Empty{})
	if err != nil {
		return nil, fmt.Errorf("query ShellcodeEncoderMap: %w", err)
	}
	if response == nil || response.Encoders == nil || response.Encoders[target.Arch] == nil {
		return nil, fmt.Errorf("ShellcodeEncoderMap has no entry for architecture %s", target.Arch)
	}
	actual := response.Encoders[target.Arch].Encoders
	result := map[string]clientpb.ShellcodeEncoder{shellcodecoverage.EncoderNone: clientpb.ShellcodeEncoder_NONE}
	known := map[string]bool{}
	for _, name := range shellcodecoverage.Encoders() {
		known[name] = true
		expected := shellcodecoverage.EncoderSupported(target, name)
		if name == shellcodecoverage.EncoderNone {
			continue
		}
		value, present := actual[name]
		if expected != present {
			return nil, fmt.Errorf("ShellcodeEncoderMap support mismatch for %s/%s encoder %s: expected=%t actual=%t", target.OS, target.Arch, name, expected, present)
		}
		if !present {
			continue
		}
		if value != shellcodeEncoderEnums[name] {
			return nil, fmt.Errorf("ShellcodeEncoderMap enum mismatch for %s: got %s, want %s", name, value, shellcodeEncoderEnums[name])
		}
		result[name] = value
	}
	var unknown []string
	for name := range actual {
		if !known[name] {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) != 0 {
		sort.Strings(unknown)
		return nil, fmt.Errorf("ShellcodeEncoderMap contains untested encoders for %s: %s", target.Arch, strings.Join(unknown, ", "))
	}
	return result, nil
}

func (s *suite) buildShellcodeRunner(ctx context.Context) (string, error) {
	sourcePath := filepath.Join(s.opts.repoPath, "test", "e2e", "fixtures", "shellcode_runner.c")
	if _, err := os.Stat(sourcePath); err != nil {
		return "", fmt.Errorf("stat C runner source: %w", err)
	}

	outputName := "shellcode-runner"
	compilerPath := ""
	compilerArgs := []string{}
	switch s.opts.targetOS {
	case "darwin", "linux":
		compilerPath, _ = exec.LookPath("cc")
		compilerArgs = []string{"-std=c11", "-O2", "-Wall", "-Wextra", "-Werror", sourcePath}
	case "windows":
		outputName += ".exe"
		compilerPath = filepath.Join(s.serverRoot, "zig", "zig.exe")
		zigTarget := map[string]string{
			"386":   "x86-windows-gnu",
			"amd64": "x86_64-windows-gnu",
		}[s.opts.targetArch]
		if zigTarget == "" {
			return "", fmt.Errorf("no Zig C target for windows/%s", s.opts.targetArch)
		}
		compilerArgs = []string{"cc", "-target", zigTarget, "-std=c11", "-O2", "-Wall", "-Wextra", "-Werror", sourcePath}
	default:
		return "", fmt.Errorf("cannot build a C shellcode runner for %s/%s", s.opts.targetOS, s.opts.targetArch)
	}
	if compilerPath == "" {
		return "", fmt.Errorf("could not find a native C compiler for %s/%s", s.opts.targetOS, s.opts.targetArch)
	}
	if _, err := os.Stat(compilerPath); err != nil {
		return "", fmt.Errorf("stat C compiler %q: %w", compilerPath, err)
	}

	outputPath := filepath.Join(s.workDir, outputName)
	compilerArgs = append(compilerArgs, "-o", outputPath)
	command := exec.CommandContext(ctx, compilerPath, compilerArgs...)
	command.Dir = s.opts.repoPath
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("compile C shellcode runner with %s: %w\n%s", compilerPath, err, strings.TrimSpace(string(output)))
	}
	stat, err := os.Stat(outputPath)
	if err != nil {
		return "", fmt.Errorf("stat compiled C shellcode runner: %w", err)
	}
	if stat.Size() == 0 {
		return "", errors.New("C compiler returned an empty shellcode runner")
	}
	if err := os.Chmod(outputPath, 0o700); err != nil {
		return "", fmt.Errorf("make C shellcode runner executable: %w", err)
	}
	return outputPath, nil
}

func (s *suite) recordAllShellcodeFailures(recorder *shellcodecoverage.Recorder, detail string) error {
	var recordErrors []error
	target := e2ecoverage.Target{OS: s.opts.targetOS, Arch: s.opts.targetArch}
	for _, transport := range s.opts.transports {
		for _, mode := range s.opts.modes {
			for _, compression := range shellcodecoverage.Compressions() {
				for _, encoder := range shellcodecoverage.Encoders() {
					if !shellcodecoverage.EncoderSupported(target, encoder) {
						continue
					}
					if err := recorder.Add(shellcodecoverage.Observation{
						Transport:        transport,
						ImplantMode:      mode,
						Compression:      compression,
						Encoder:          encoder,
						Status:           e2ecoverage.StatusFail,
						Detail:           shellcodeFailureDetail(detail),
						RequiredSamples:  shellcodeEncoderSamples(encoder, s.opts.sgnSamples),
						CompletedSamples: 0,
					}); err != nil {
						recordErrors = append(recordErrors, err)
					}
				}
			}
		}
	}
	return errors.Join(recordErrors...)
}

func (s *suite) recordShellcodeBuildFailures(
	recorder *shellcodecoverage.Recorder,
	transport string,
	mode string,
	compression string,
	duration time.Duration,
	detail string,
) error {
	var recordErrors []error
	target := e2ecoverage.Target{OS: s.opts.targetOS, Arch: s.opts.targetArch}
	for _, encoder := range shellcodecoverage.Encoders() {
		if !shellcodecoverage.EncoderSupported(target, encoder) {
			continue
		}
		if err := recorder.Add(shellcodecoverage.Observation{
			Transport:        transport,
			ImplantMode:      mode,
			Compression:      compression,
			Encoder:          encoder,
			Status:           e2ecoverage.StatusFail,
			Duration:         duration,
			Detail:           shellcodeFailureDetail(detail),
			RequiredSamples:  shellcodeEncoderSamples(encoder, s.opts.sgnSamples),
			CompletedSamples: 0,
		}); err != nil {
			recordErrors = append(recordErrors, err)
		}
	}
	return errors.Join(recordErrors...)
}

func shellcodeTargetSupported(target e2ecoverage.Target) bool {
	for _, supported := range shellcodecoverage.Targets() {
		if supported == target {
			return true
		}
	}
	return false
}

func shellcodeFailureDetail(detail string) string {
	detail = strings.TrimSpace(strings.ToValidUTF8(detail, "\uFFFD"))
	if len(detail) <= shellcodeFailureDetailBytes {
		return detail
	}
	end := shellcodeFailureDetailBytes
	for end > 0 && !utf8.ValidString(detail[:end]) {
		end--
	}
	return detail[:end] + "...(truncated)"
}

func shellcodeExecutionProtection(arch string, encoder string) string {
	// The 386 and amd64 SGN/XOR decoder stubs decode in place. ARM64
	// encoders allocate a separate writable buffer before transferring control,
	// so their outer payload mapping can remain W^X-safe.
	if encoder != shellcodecoverage.EncoderNone && (arch == "386" || arch == "amd64") {
		return "rwx"
	}
	return "rx"
}
