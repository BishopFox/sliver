package e2e

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	clientopfor "github.com/bishopfox/sliver/client/command/opfor"
	clientconsole "github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

const (
	nativeBOFDownloadTimeout = 5 * time.Minute
	nativeBOFCompileTimeout  = 2 * time.Minute
	nativeBOFOutputLimit     = 1 << 20

	catReleasePublicKey = "RWTFh8KpxR0fDBsOK+FYYSo/SxW9hFEQwqqCa0hv1YLyjVXrbXR/hMdf"
	firefoxPublicKey    = "RWQT71u8OryWX4sIY2mwJq/41tKfI6SdZivwRlWLP9pBDn6ijresXI/H"

	catCNAHash        = "94c7bcaae209a6355dcc8c126019f6e19a681173680955166b8c30cf97fc66f7"
	firefoxCNAHash    = "c8c9ef28675d16dd1c8786f055c08ce2096eb0fedb16fad68b78f22c81b1bbde"
	findDotnetCNAHash = "897dbee453ed504be7e9ace1229e7070f131a9351fb09a723629dd8e0337aa03"
	findSysmonCNAHash = "1579dcca9e1c0ab7b20a646ff2db7705e05468c067a8ed6b3c35b553f10458d7"

	catX64ObjectHash     = "c96c07c85fc3809240f87ab37b1aba40c80dcf1c81caa669bde8f8bddd0815e2"
	firefoxX64ObjectHash = "86df6e65c759ce092d18274dfe17cde17c16d8194d609455c918d2dc0a9e78e5"
	firefoxX86ObjectHash = "b1a11e41c9f4d99bc9df94878853d29f30a24250169eaedcf71150f96a498dd0"
	findDotnetObjectHash = "1dcda8bf8db5851e9fc64b40690d3c03f1f86e59ae4b340520603774acff5508"
	findSysmonObjectHash = "c2924e690b4ba407fddcfefdba2632e649008084afb7a9dcf9e11f5853cf3050"

	typedCallbackMarker   = "OPFOR_E2E_TYPED_CALLBACK_OK"
	partialCallbackMarker = "OPFOR_E2E_PARTIAL_CALLBACK_OK"
	malformedMarker       = "OPFOR_E2E_MALFORMED_CALLBACK_OK"

	catScenario        = "OPFOR Cat CNA reads isolated test file"
	firefoxScenario    = "OPFOR FirefoxDump CNA finds no host profiles"
	findDotnetScenario = "OPFOR FindDotnet CNA read-only process inventory"
	findSysmonScenario = "OPFOR FindSysmon CNA read-only registry probe"
	typedScenario      = "OPFOR callback preserves ordered typed binary channels"
	partialScenario    = "typed BOF partial output retained on callback error"
	malformedScenario  = "malformed BOF returns bounded loader error"
	timeoutScenario    = "finite BOF deadline returns and target recovers"
)

var (
	catArchiveAsset = pinnedAsset{
		url:    "https://github.com/sliverarmory/bof_collection/releases/download/v0.0.1/beune-bof-collection.tar.gz",
		sha256: "d69747b567c69c7ed03ef4ae5b6c1f76e76ae88f40f68987f56c0276db1389d7",
	}
	catSignatureAsset = pinnedAsset{
		url:    "https://github.com/sliverarmory/bof_collection/releases/download/v0.0.1/beune-bof-collection.minisig",
		sha256: "56ef773025fb7cd4b397cedd2d68b73f99bd1c0bd5cbbcc8c73d376ebb765cfd",
	}
	// v0.0.1 is the exact source revision used for the x86 Cat build. The
	// release tag resolves directly to commit 2cb3fb1b39a96484c4c40b8710c1ca9f83e846ee.
	catSourceAsset = pinnedAsset{
		url:    "https://raw.githubusercontent.com/sliverarmory/bof_collection/2cb3fb1b39a96484c4c40b8710c1ca9f83e846ee/cat/entry.c",
		sha256: "a728f11fc10670a2435adebf78374878bbb3aaf16c445697d75819f6f9a3a578",
	}
	catBeaconHeaderAsset = pinnedAsset{
		url:    "https://raw.githubusercontent.com/sliverarmory/bof_collection/2cb3fb1b39a96484c4c40b8710c1ca9f83e846ee/beacon.h",
		sha256: "ff0d64312744d7934e633c604201391b35aef1f40051769d277b2205eb8aa6c2",
	}
	firefoxArchiveAsset = pinnedAsset{
		url:    "https://github.com/sliverarmory/firefoxdump/releases/download/v0.0.2/firefoxdump.tar.gz",
		sha256: "76260d331ebb57454d06caf9badb512bc21827146e85f425d6337096facfde7f",
	}
	firefoxSignatureAsset = pinnedAsset{
		url:    "https://github.com/sliverarmory/firefoxdump/releases/download/v0.0.2/firefoxdump.minisig",
		sha256: "22d2d5adfd40612fad82be927101ec2f0e38795b44e34d0a7b539d72f2ed74dd",
	}
	findDotnetAsset = pinnedAsset{
		url:    "https://raw.githubusercontent.com/sliverarmory/OperatorsKit/66368f4738528d26cc1ccc6d9a3c93d44d63edc1/KIT/FindDotnet/finddotnet.o",
		sha256: findDotnetObjectHash,
	}
	findSysmonAsset = pinnedAsset{
		url:    "https://raw.githubusercontent.com/sliverarmory/OperatorsKit/66368f4738528d26cc1ccc6d9a3c93d44d63edc1/KIT/FindSysmon/findsysmon.o",
		sha256: findSysmonObjectHash,
	}

	nativeBOFConsoleMu sync.Mutex
)

type nativeBOFAssets struct {
	catScript        string
	firefoxScript    string
	findDotnetScript string
	findSysmonScript string
	callbackScript   string
	malformedScript  string
}

type nativeBOFRunner struct {
	suite   *suite
	console *clientconsole.SliverClient
	rpc     *nativeBOFCapturingRPC
	loaded  map[string]bool
}

type nativeBOFCall struct {
	taskID   string
	beaconID string
	async    bool
	isBOF    bool
	err      error
}

type nativeBOFCapturingRPC struct {
	rpcpb.SliverRPCClient

	mu    sync.Mutex
	calls []nativeBOFCall
}

type capturedOPFORCommand struct {
	output    string
	err       error
	truncated bool
}

type boundedCapture struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func (capture *boundedCapture) Write(data []byte) (int, error) {
	written := len(data)
	remaining := capture.limit - capture.buffer.Len()
	if remaining > 0 {
		if len(data) > remaining {
			_, _ = capture.buffer.Write(data[:remaining])
			capture.truncated = true
		} else {
			_, _ = capture.buffer.Write(data)
		}
	} else if len(data) != 0 {
		capture.truncated = true
	}
	return written, nil
}

func (s *suite) exerciseNativeBOFs(target implantTarget, remoteRoot string, transport string) error {
	if s.opts.targetOS != "windows" || (s.opts.targetArch != "386" && s.opts.targetArch != "amd64") {
		s.t.Logf("SKIP native OPFOR BOFs on %s/%s: no supported Reflektor target", s.opts.targetOS, s.opts.targetArch)
		return nil
	}

	if err := s.localStep("OPFORNativeFixtures", "pinned signed releases and exact-revision source build", s.prepareNativeBOFs); err != nil {
		return err
	}
	runner, err := s.newNativeBOFRunner(target)
	if err != nil {
		return err
	}

	var scenarioErrors []error
	scenarioErrors = appendIfError(scenarioErrors, s.exerciseNativeCat(runner, target, remoteRoot, transport))
	scenarioErrors = appendIfError(scenarioErrors, s.exerciseNativeFirefox(runner, target, transport))
	if s.opts.targetArch == "amd64" {
		scenarioErrors = appendIfError(scenarioErrors, s.exerciseNativeFindDotnet(runner, target, transport))
		scenarioErrors = appendIfError(scenarioErrors, s.exerciseNativeFindSysmon(runner, target, transport))
	}
	scenarioErrors = appendIfError(scenarioErrors, s.exerciseNativeTypedOutput(runner, target, transport))
	scenarioErrors = appendIfError(scenarioErrors, s.exerciseNativePartialError(runner, target, transport))
	scenarioErrors = appendIfError(scenarioErrors, s.exerciseNativeMalformed(runner, target, transport))
	scenarioErrors = appendIfError(scenarioErrors, s.exerciseNativeTimeout(runner, target, transport))
	return errors.Join(scenarioErrors...)
}

func (s *suite) exerciseNativeCat(runner *nativeBOFRunner, target implantTarget, remoteRoot string, transport string) error {
	return s.step(target, transport, "CallExtension", catScenario, func() error {
		const marker = "SLIVER_OPFOR_NATIVE_CAT_MARKER"
		path := filepath.Join(remoteRoot, "opfor native cat fixture.txt")
		if err := os.WriteFile(path, []byte(marker+"\n"), 0o600); err != nil {
			return fmt.Errorf("write isolated Cat fixture: %w", err)
		}
		result := runner.invoke(s.nativeBOF.catScript, "cat", path)
		if err := validateCapturedCommand(result); err != nil {
			return err
		}
		if !strings.Contains(result.output, marker) || !strings.Contains(result.output, path) {
			return fmt.Errorf("Cat output did not contain the isolated path and marker: %q", summarizeOutput(result.output))
		}
		return nil
	})
}

func (s *suite) exerciseNativeFirefox(runner *nativeBOFRunner, target implantTarget, transport string) error {
	return s.step(target, transport, "CallExtension", firefoxScenario, func() error {
		return withSafeEmptyFirefoxProfile(func() error {
			result := runner.invoke(s.nativeBOF.firefoxScript, "firefoxdump")
			fingerprint := outputFingerprint(result.output)
			if result.truncated {
				return fmt.Errorf("FirefoxDump output exceeded %d bytes; %s", nativeBOFOutputLimit, fingerprint)
			}
			if result.err != nil {
				return fmt.Errorf("FirefoxDump command failed (%T); %s", result.err, fingerprint)
			}
			var missing []string
			for _, marker := range []string{
				"Profile has none of the requested Firefox data - skipping",
				"Completed processing 1 profile(s)",
			} {
				if !strings.Contains(result.output, marker) {
					missing = append(missing, marker)
				}
			}
			if len(missing) != 0 {
				return fmt.Errorf("FirefoxDump output missing markers %q; %s", missing, fingerprint)
			}
			return nil
		})
	})
}

func (s *suite) exerciseNativeFindDotnet(runner *nativeBOFRunner, target implantTarget, transport string) error {
	return s.step(target, transport, "CallExtension", findDotnetScenario, func() error {
		result := runner.invoke(s.nativeBOF.findDotnetScript, "finddotnet")
		if err := validateCapturedCommand(result); err != nil {
			return err
		}
		if strings.Contains(result.output, "No .NET process found!") {
			return nil
		}
		if strings.Contains(result.output, "Process name") && strings.Contains(result.output, "PID") {
			return nil
		}
		return fmt.Errorf("FindDotnet output did not contain a stable inventory result: %q", summarizeOutput(result.output))
	})
}

func (s *suite) exerciseNativeFindSysmon(runner *nativeBOFRunner, target implantTarget, transport string) error {
	return s.step(target, transport, "CallExtension", findSysmonScenario, func() error {
		if err := requireSysmonChannelAbsent(s.ctx); err != nil {
			return err
		}
		result := runner.invoke(s.nativeBOF.findSysmonScript, "findsysmon", "reg")
		if err := validateCapturedCommand(result); err != nil {
			return err
		}
		if !strings.Contains(result.output, "No Sysmon service found") {
			return fmt.Errorf("FindSysmon output did not report the safety-gated absent service: %q", summarizeOutput(result.output))
		}
		return nil
	})
}

func (s *suite) exerciseNativeTypedOutput(runner *nativeBOFRunner, target implantTarget, transport string) error {
	return s.step(target, transport, "CallExtension", typedScenario, func() error {
		result := runner.invoke(s.nativeBOF.callbackScript, "opfor-e2e-output")
		if err := validateCapturedCommand(result); err != nil {
			return err
		}
		if !strings.Contains(result.output, typedCallbackMarker) {
			return fmt.Errorf("typed callback success marker missing: %q", summarizeOutput(result.output))
		}
		return nil
	})
}

func (s *suite) exerciseNativePartialError(runner *nativeBOFRunner, target implantTarget, transport string) error {
	return s.step(target, transport, "CallExtension", partialScenario, func() error {
		result := runner.invoke(s.nativeBOF.callbackScript, "opfor-e2e-partial")
		if result.truncated {
			return fmt.Errorf("OPFOR output exceeded %d bytes", nativeBOFOutputLimit)
		}
		if result.err == nil {
			return errors.New("partial-output fixture unexpectedly succeeded")
		}
		message := strings.ToLower(result.err.Error())
		if !strings.Contains(message, "beaconoutput") || !strings.Contains(message, "length 1") {
			return fmt.Errorf("partial-output error did not preserve the Reflektor callback failure: %w", result.err)
		}
		if !strings.Contains(result.output, partialCallbackMarker) {
			return fmt.Errorf("partial-output lifecycle callback marker missing: %q", summarizeOutput(result.output))
		}
		return nil
	})
}

func (s *suite) exerciseNativeMalformed(runner *nativeBOFRunner, target implantTarget, transport string) error {
	return s.step(target, transport, "CallExtension", malformedScenario, func() error {
		started := time.Now()
		result := runner.invoke(s.nativeBOF.malformedScript, "opfor-e2e-malformed")
		elapsed := time.Since(started)
		if result.truncated {
			return fmt.Errorf("OPFOR output exceeded %d bytes", nativeBOFOutputLimit)
		}
		if result.err == nil {
			return errors.New("malformed BOF unexpectedly succeeded")
		}
		if !strings.Contains(strings.ToLower(result.err.Error()), "load bof") {
			return fmt.Errorf("malformed object returned the wrong error: %w", result.err)
		}
		if !strings.Contains(result.output, malformedMarker) {
			return fmt.Errorf("malformed-object lifecycle callback marker missing: %q", summarizeOutput(result.output))
		}
		limit := 30 * time.Second
		if target.beacon != nil {
			limit += 2 * s.opts.beaconInterval
		}
		if elapsed > limit {
			return fmt.Errorf("malformed object took %s, bounded limit is %s", elapsed.Round(time.Millisecond), limit)
		}
		return s.nativeBOFRecoveryPing(target)
	})
}

func (s *suite) exerciseNativeTimeout(runner *nativeBOFRunner, target implantTarget, transport string) error {
	return s.step(target, transport, "CallExtension", timeoutScenario, func() error {
		timeoutSeconds := int64(1)
		sleepDuration := 2500 * time.Millisecond
		if target.beacon != nil {
			timeoutSeconds = int64((s.opts.beaconInterval+time.Second-1)/time.Second) + 2
			sleepDuration = time.Duration(timeoutSeconds+3) * time.Second
		}
		callCursor := runner.rpc.callCount()
		result := runner.invokeWithArguments(
			s.nativeBOF.callbackScript,
			[]string{"--timeout", strconv.FormatInt(timeoutSeconds, 10), "opfor-e2e-timeout", strconv.FormatInt(sleepDuration.Milliseconds(), 10)},
		)
		if result.truncated {
			return fmt.Errorf("OPFOR output exceeded %d bytes", nativeBOFOutputLimit)
		}
		if result.err == nil || !isDeadlineError(result.err) {
			return fmt.Errorf("finite timeout did not return a deadline error: %v", result.err)
		}
		if strings.Contains(result.output, "before-timeout") || strings.Contains(result.output, "after-timeout") {
			return fmt.Errorf("timed-out BOF leaked callback output: %q", summarizeOutput(result.output))
		}

		settle := sleepDuration + time.Second
		if err := waitForNativeBOF(s.ctx, settle); err != nil {
			return err
		}
		if target.beacon != nil {
			taskID, err := runner.rpc.beaconTaskAfter(callCursor, target.beacon.ID)
			if err != nil {
				return err
			}
			if err := s.verifyCompletedTimeoutBeaconTask(target, taskID); err != nil {
				return err
			}
		}
		return s.nativeBOFRecoveryPing(target)
	})
}

func (s *suite) nativeBOFRecoveryPing(target implantTarget) error {
	const nonce = int32(0x2f0b0f42)
	response, err := invokeRPC(s, target, "Ping after native BOF", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Ping, error) {
		return s.rpc.Ping(ctx, &sliverpb.Ping{Nonce: nonce, Request: request})
	}, func(response *sliverpb.Ping) *commonpb.Response { return response.GetResponse() })
	if err != nil {
		return fmt.Errorf("target did not recover after native BOF: %w", err)
	}
	if response.GetNonce() != nonce {
		return fmt.Errorf("recovery Ping nonce got %d, want %d", response.GetNonce(), nonce)
	}
	return nil
}

func (s *suite) verifyCompletedTimeoutBeaconTask(target implantTarget, taskID string) error {
	ctx, cancel := context.WithTimeout(s.ctx, 2*s.opts.beaconInterval+10*time.Second)
	defer cancel()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		task, err := s.rpc.GetBeaconTaskContent(ctx, &clientpb.BeaconTask{ID: taskID})
		if err != nil {
			return fmt.Errorf("fetch timed-out BOF beacon task %s: %w", taskID, err)
		}
		if task.GetBeaconID() != target.beacon.ID {
			return fmt.Errorf("timed-out BOF task %s beacon %q, want %q", taskID, task.GetBeaconID(), target.beacon.ID)
		}
		switch strings.ToLower(task.GetState()) {
		case "completed":
			response := &sliverpb.CallExtension{}
			if len(task.GetResponse()) == 0 {
				return fmt.Errorf("timed-out BOF task %s completed without a response", taskID)
			}
			if err := proto.Unmarshal(task.GetResponse(), response); err != nil {
				return fmt.Errorf("decode timed-out BOF task %s: %w", taskID, err)
			}
			if response.GetResponse().GetErr() != "" {
				return fmt.Errorf("timed-out BOF task %s completed with an implant error", taskID)
			}
			records := response.GetBOFOutputs()
			if len(records) != 2 || records[0].GetType() != 0 || records[1].GetType() != 0 ||
				!bytes.Equal(records[0].GetData(), []byte("before-timeout")) ||
				!bytes.Equal(records[1].GetData(), []byte("after-timeout")) {
				return fmt.Errorf("timed-out BOF task %s did not retain its two expected typed records", taskID)
			}
			return nil
		case "failed", "canceled", "cancelled":
			return fmt.Errorf("timed-out BOF task %s entered terminal state %q", taskID, task.GetState())
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for timed-out BOF task %s completion: %w", taskID, ctx.Err())
		case <-ticker.C:
		}
	}
}

func (s *suite) prepareNativeBOFs() error {
	s.nativeBOFOnce.Do(func() {
		s.nativeBOF, s.nativeBOFErr = s.buildNativeBOFAssets(s.ctx)
	})
	return s.nativeBOFErr
}

func (s *suite) buildNativeBOFAssets(ctx context.Context) (*nativeBOFAssets, error) {
	root := filepath.Join(s.workDir, "native-opfor-bofs")
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("create native OPFOR fixture root: %w", err)
	}
	httpClient := &http.Client{Timeout: nativeBOFDownloadTimeout}
	barch, machine, zigTarget, err := nativeBOFTarget(s.opts.targetArch)
	if err != nil {
		return nil, err
	}

	catScript, err := copyPinnedCNA(
		filepath.Join(s.opts.repoPath, "client", "command", "opfor", "testdata", "corpus", "bof_collection", "cat.cna"),
		catCNAHash,
		filepath.Join(root, "cat", "cat.cna"),
	)
	if err != nil {
		return nil, err
	}
	catObjectPath := filepath.Join(filepath.Dir(catScript), "dist", "cat."+barch+".o")
	if s.opts.targetArch == "amd64" {
		files, err := downloadVerifiedArchive(ctx, httpClient, "bof_collection v0.0.1", catArchiveAsset, catSignatureAsset, catReleasePublicKey)
		if err != nil {
			return nil, err
		}
		object, err := archiveFileExact(files, "cat.x64.o")
		if err != nil {
			return nil, err
		}
		if err := verifyPinnedBytes("signed Cat x64 object", object, catX64ObjectHash); err != nil {
			return nil, err
		}
		if err := verifyCOFFMachine("signed Cat x64 object", object, machine); err != nil {
			return nil, err
		}
		if err := writeNativeFixture(catObjectPath, object); err != nil {
			return nil, err
		}
	} else {
		if err := s.buildPinnedCatX86(ctx, httpClient, catObjectPath, machine, zigTarget, root); err != nil {
			return nil, err
		}
	}

	firefoxScript, err := copyPinnedCNA(
		filepath.Join(s.opts.repoPath, "client", "command", "opfor", "testdata", "corpus", "firefoxdump", "firefoxdump.cna"),
		firefoxCNAHash,
		filepath.Join(root, "firefoxdump", "firefoxdump.cna"),
	)
	if err != nil {
		return nil, err
	}
	firefoxFiles, err := downloadVerifiedArchive(ctx, httpClient, "firefoxdump v0.0.2", firefoxArchiveAsset, firefoxSignatureAsset, firefoxPublicKey)
	if err != nil {
		return nil, err
	}
	firefoxObjectName := "firefoxdump." + barch + ".o"
	firefoxObject, err := archiveFileExact(firefoxFiles, firefoxObjectName)
	if err != nil {
		return nil, err
	}
	firefoxHash := firefoxX64ObjectHash
	if s.opts.targetArch == "386" {
		firefoxHash = firefoxX86ObjectHash
	}
	if err := verifyPinnedBytes("signed "+firefoxObjectName, firefoxObject, firefoxHash); err != nil {
		return nil, err
	}
	if err := verifyCOFFMachine("signed "+firefoxObjectName, firefoxObject, machine); err != nil {
		return nil, err
	}
	if err := writeNativeFixture(filepath.Join(filepath.Dir(firefoxScript), "bin", firefoxObjectName), firefoxObject); err != nil {
		return nil, err
	}

	assets := &nativeBOFAssets{catScript: catScript, firefoxScript: firefoxScript}
	if s.opts.targetArch == "amd64" {
		assets.findDotnetScript, err = materializeOperatorsKitFixture(
			ctx, httpClient, s.opts.repoPath, root, "finddotnet", findDotnetCNAHash, findDotnetAsset, machine,
		)
		if err != nil {
			return nil, err
		}
		assets.findSysmonScript, err = materializeOperatorsKitFixture(
			ctx, httpClient, s.opts.repoPath, root, "findsysmon", findSysmonCNAHash, findSysmonAsset, machine,
		)
		if err != nil {
			return nil, err
		}
	}

	assets.callbackScript, err = s.buildSyntheticNativeBOF(ctx, root, barch, machine, zigTarget)
	if err != nil {
		return nil, err
	}
	assets.malformedScript, err = buildMalformedNativeBOF(root, barch)
	if err != nil {
		return nil, err
	}
	return assets, nil
}

func (s *suite) buildPinnedCatX86(
	ctx context.Context,
	httpClient *http.Client,
	outputPath string,
	machine uint16,
	zigTarget string,
	root string,
) error {
	entry, err := downloadPinned(ctx, httpClient, catSourceAsset)
	if err != nil {
		return fmt.Errorf("download bof_collection v0.0.1 cat/entry.c: %w", err)
	}
	header, err := downloadPinned(ctx, httpClient, catBeaconHeaderAsset)
	if err != nil {
		return fmt.Errorf("download bof_collection v0.0.1 beacon.h: %w", err)
	}
	sourceRoot := filepath.Join(root, "cat-v0.0.1-source")
	entryPath := filepath.Join(sourceRoot, "cat", "entry.c")
	if err := writeNativeFixture(entryPath, entry); err != nil {
		return err
	}
	if err := writeNativeFixture(filepath.Join(sourceRoot, "beacon.h"), header); err != nil {
		return err
	}
	// The bundled Zig distribution has no MinGW/Windows SDK headers. Supply
	// only the ABI declarations required by these exact pinned sources.
	compatibilityHeader, err := os.ReadFile(filepath.Join(s.opts.repoPath, "test", "e2e", "fixtures", "opfor_cat_windows.h"))
	if err != nil {
		return fmt.Errorf("read Cat Windows compatibility header: %w", err)
	}
	if err := writeNativeFixture(filepath.Join(sourceRoot, "cat", "windows.h"), compatibilityHeader); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(sourceRoot, "cat", "common"), 0o700); err != nil {
		return fmt.Errorf("create Cat common include directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o700); err != nil {
		return fmt.Errorf("create source-built Cat output directory: %w", err)
	}

	compileCtx, cancel := context.WithTimeout(ctx, nativeBOFCompileTimeout)
	defer cancel()
	compiler := filepath.Join(s.serverRoot, "zig", "zig.exe")
	arguments := []string{
		"cc", "-target", zigTarget, "-nostdlib", "-o", outputPath,
		"-I.", "-Icommon", "-Os", "-c", "-DBOF", "-fno-builtin",
		"-D__USE_MINGW_ANSI_STDIO=0", "entry.c",
	}
	command := exec.CommandContext(compileCtx, compiler, arguments...)
	command.Dir = filepath.Dir(entryPath)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compile bof_collection v0.0.1 Cat x86 with bundled Zig: %w\n%s", err, strings.TrimSpace(string(output)))
	}
	object, err := os.ReadFile(outputPath)
	if err != nil {
		return fmt.Errorf("read source-built Cat x86 object: %w", err)
	}
	return verifyCOFFMachine("source-built bof_collection v0.0.1 Cat x86 object", object, machine)
}

func (s *suite) buildSyntheticNativeBOF(ctx context.Context, root, barch string, machine uint16, zigTarget string) (string, error) {
	directory := filepath.Join(root, "synthetic")
	script, err := copyPinnedCNA(
		filepath.Join(s.opts.repoPath, "test", "e2e", "fixtures", "opfor_callback.cna"),
		"",
		filepath.Join(directory, "opfor_callback.cna"),
	)
	if err != nil {
		return "", err
	}
	source := filepath.Join(s.opts.repoPath, "test", "e2e", "fixtures", "opfor_bof_fixture.c")
	if _, err := os.Stat(source); err != nil {
		return "", fmt.Errorf("stat synthetic OPFOR BOF source: %w", err)
	}
	objectPath := filepath.Join(directory, "opfor_bof_fixture."+barch+".o")
	compileCtx, cancel := context.WithTimeout(ctx, nativeBOFCompileTimeout)
	defer cancel()
	compiler := filepath.Join(s.serverRoot, "zig", "zig.exe")
	command := exec.CommandContext(
		compileCtx,
		compiler,
		"cc", "-target", zigTarget, "-Oz", "-fno-stack-protector", "-fno-builtin", "-c", source, "-o", objectPath,
	)
	command.Dir = s.opts.repoPath
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("compile synthetic OPFOR BOF with bundled Zig: %w\n%s", err, strings.TrimSpace(string(output)))
	}
	object, err := os.ReadFile(objectPath)
	if err != nil {
		return "", fmt.Errorf("read synthetic OPFOR BOF: %w", err)
	}
	if err := verifyCOFFMachine("synthetic OPFOR BOF", object, machine); err != nil {
		return "", err
	}
	return script, nil
}

func materializeOperatorsKitFixture(
	ctx context.Context,
	httpClient *http.Client,
	repoPath string,
	root string,
	name string,
	cnaHash string,
	objectAsset pinnedAsset,
	machine uint16,
) (string, error) {
	directory := filepath.Join(root, name)
	script, err := copyPinnedCNA(
		filepath.Join(repoPath, "client", "command", "opfor", "testdata", "corpus", "operatorskit", name+".cna"),
		cnaHash,
		filepath.Join(directory, name+".cna"),
	)
	if err != nil {
		return "", err
	}
	object, err := downloadPinned(ctx, httpClient, objectAsset)
	if err != nil {
		return "", fmt.Errorf("download pinned OperatorsKit %s object: %w", name, err)
	}
	if err := verifyCOFFMachine("OperatorsKit "+name+" object", object, machine); err != nil {
		return "", err
	}
	if err := writeNativeFixture(filepath.Join(directory, name+".o"), object); err != nil {
		return "", err
	}
	return script, nil
}

func buildMalformedNativeBOF(root, barch string) (string, error) {
	directory := filepath.Join(root, "malformed")
	script := filepath.Join(directory, "opfor_malformed.cna")
	object := filepath.Join(directory, "opfor_malformed."+barch+".o")
	if err := writeNativeFixture(object, []byte{0x00, 0x01, 0x02}); err != nil {
		return "", err
	}
	content := `beacon_command_register(
    "opfor-e2e-malformed",
    "Exercise a bounded malformed-object loader failure through OPFOR.",
    "Internal Sliver comprehensive E2E fixture."
);

sub opfor_e2e_malformed_callback {
    local('$key');
    foreach $key (keys($3)) {
        if ($key eq "chunk_num" || $key eq "type_id" || $key eq "is_final") {
            throw "OPFOR E2E malformed lifecycle callback exposed data-only metadata";
        }
    }
    if (!($3["type"] eq "error" && indexOf($2, "load BOF") >= 0)) {
        throw "OPFOR E2E malformed lifecycle callback mismatch";
    }
    blog($1, "OPFOR_E2E_MALFORMED_CALLBACK_OK");
}

alias opfor-e2e-malformed {
    local('$handle $object');
    $handle = openf(script_resource("opfor_malformed." . barch($1) . ".o"));
    $object = readb($handle, -1);
    closef($handle);
    beacon_inline_execute($1, $object, "go", $null, &opfor_e2e_malformed_callback);
}
`
	if err := writeNativeFixture(script, []byte(content)); err != nil {
		return "", err
	}
	return script, nil
}

func downloadVerifiedArchive(
	ctx context.Context,
	httpClient *http.Client,
	name string,
	archiveAsset pinnedAsset,
	signatureAsset pinnedAsset,
	publicKey string,
) (map[string][]byte, error) {
	archive, err := downloadPinned(ctx, httpClient, archiveAsset)
	if err != nil {
		return nil, fmt.Errorf("download %s archive: %w", name, err)
	}
	signature, err := downloadPinned(ctx, httpClient, signatureAsset)
	if err != nil {
		return nil, fmt.Errorf("download %s signature: %w", name, err)
	}
	if err := verifySignature(publicKey, archive, signature); err != nil {
		return nil, fmt.Errorf("verify %s archive signature: %w", name, err)
	}
	files, err := readTarGzip(archive)
	if err != nil {
		return nil, fmt.Errorf("read %s archive: %w", name, err)
	}
	return files, nil
}

func copyPinnedCNA(source string, expectedHash string, destination string) (string, error) {
	content, err := os.ReadFile(source)
	if err != nil {
		return "", fmt.Errorf("read CNA fixture %s: %w", source, err)
	}
	if expectedHash != "" {
		if err := verifyPinnedBytes(filepath.Base(source), content, expectedHash); err != nil {
			return "", err
		}
	}
	if err := writeNativeFixture(destination, content); err != nil {
		return "", err
	}
	return destination, nil
}

func writeNativeFixture(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create native fixture directory for %s: %w", path, err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return fmt.Errorf("write native fixture %s: %w", path, err)
	}
	return nil
}

func verifyPinnedBytes(name string, content []byte, expected string) error {
	digest := sha256.Sum256(content)
	actual := hex.EncodeToString(digest[:])
	if actual != expected {
		return fmt.Errorf("%s SHA-256 %s, want %s", name, actual, expected)
	}
	return nil
}

func verifyCOFFMachine(name string, object []byte, expected uint16) error {
	if len(object) < 20 {
		return fmt.Errorf("%s is shorter than a COFF header", name)
	}
	machine := binary.LittleEndian.Uint16(object[:2])
	sections := binary.LittleEndian.Uint16(object[2:4])
	if machine != expected {
		return fmt.Errorf("%s COFF machine 0x%04x, want 0x%04x", name, machine, expected)
	}
	if sections == 0 {
		return fmt.Errorf("%s COFF header has no sections", name)
	}
	return nil
}

func nativeBOFTarget(arch string) (barch string, machine uint16, zigTarget string, err error) {
	switch arch {
	case "386":
		return "x86", 0x014c, "x86-windows-gnu", nil
	case "amd64":
		return "x64", 0x8664, "x86_64-windows-gnu", nil
	default:
		return "", 0, "", fmt.Errorf("unsupported native BOF target windows/%s", arch)
	}
}

func (s *suite) newNativeBOFRunner(target implantTarget) (*nativeBOFRunner, error) {
	// NewConsole consults SLIVER_CLIENT_ROOT_DIR while constructing its output
	// sink. Keep that one-time setup inside the suite's isolated client root.
	nativeBOFConsoleMu.Lock()
	previousRoot, hadRoot := os.LookupEnv("SLIVER_CLIENT_ROOT_DIR")
	if err := os.Setenv("SLIVER_CLIENT_ROOT_DIR", s.clientRoot); err != nil {
		nativeBOFConsoleMu.Unlock()
		return nil, fmt.Errorf("set isolated Sliver client root: %w", err)
	}
	console := clientconsole.NewConsole(false)
	console.Settings.ConsoleLogs = false
	if hadRoot {
		_ = os.Setenv("SLIVER_CLIENT_ROOT_DIR", previousRoot)
	} else {
		_ = os.Unsetenv("SLIVER_CLIENT_ROOT_DIR")
	}
	nativeBOFConsoleMu.Unlock()

	capturingRPC := &nativeBOFCapturingRPC{SliverRPCClient: s.rpc}
	if err := clientconsole.StartClient(console, capturingRPC, nil, nil, nil, nil, false, ""); err != nil {
		return nil, fmt.Errorf("initialize OPFOR E2E console: %w", err)
	}
	console.ActiveTarget.Set(target.session, target.beacon)
	return &nativeBOFRunner{suite: s, console: console, rpc: capturingRPC, loaded: map[string]bool{}}, nil
}

func (rpc *nativeBOFCapturingRPC) CallExtension(
	ctx context.Context,
	request *sliverpb.CallExtensionReq,
	options ...grpc.CallOption,
) (*sliverpb.CallExtension, error) {
	response, err := rpc.SliverRPCClient.CallExtension(ctx, request, options...)
	call := nativeBOFCall{err: err}
	if request != nil {
		call.isBOF = request.GetIsBOF()
		call.beaconID = request.GetRequest().GetBeaconID()
	}
	if response != nil {
		call.taskID = response.GetResponse().GetTaskID()
		call.async = response.GetResponse().GetAsync()
	}
	rpc.mu.Lock()
	rpc.calls = append(rpc.calls, call)
	rpc.mu.Unlock()
	return response, err
}

func (rpc *nativeBOFCapturingRPC) callCount() int {
	rpc.mu.Lock()
	defer rpc.mu.Unlock()
	return len(rpc.calls)
}

func (rpc *nativeBOFCapturingRPC) beaconTaskAfter(cursor int, beaconID string) (string, error) {
	rpc.mu.Lock()
	defer rpc.mu.Unlock()
	if cursor < 0 || cursor > len(rpc.calls) {
		return "", fmt.Errorf("invalid native BOF CallExtension cursor %d", cursor)
	}
	calls := rpc.calls[cursor:]
	if len(calls) != 1 {
		return "", fmt.Errorf("native BOF timeout issued %d CallExtension requests, want 1", len(calls))
	}
	call := calls[0]
	if call.err != nil {
		return "", fmt.Errorf("native BOF timeout CallExtension dispatch failed before task correlation: %w", call.err)
	}
	if !call.isBOF || !call.async || call.beaconID != beaconID || call.taskID == "" {
		return "", fmt.Errorf(
			"native BOF timeout task metadata mismatch: bof=%v async=%v beacon=%q task=%q",
			call.isBOF, call.async, call.beaconID, call.taskID,
		)
	}
	return call.taskID, nil
}

func (runner *nativeBOFRunner) invoke(script string, alias string, arguments ...string) capturedOPFORCommand {
	return runner.invokeWithArguments(script, append([]string{alias}, arguments...))
}

func (runner *nativeBOFRunner) invokeWithArguments(script string, arguments []string) capturedOPFORCommand {
	if !runner.loaded[script] {
		loaded := runner.execute([]string{"load", script})
		if loaded.err != nil || loaded.truncated {
			return loaded
		}
		runner.loaded[script] = true
	}
	return runner.execute(arguments)
}

func (runner *nativeBOFRunner) execute(arguments []string) capturedOPFORCommand {
	nativeBOFConsoleMu.Lock()
	defer nativeBOFConsoleMu.Unlock()

	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		return capturedOPFORCommand{err: fmt.Errorf("capture OPFOR stdout: %w", err)}
	}
	previousStdout := os.Stdout
	os.Stdout = writePipe
	capture := &boundedCapture{limit: nativeBOFOutputLimit}
	copyDone := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(capture, readPipe)
		copyDone <- copyErr
	}()

	commands := clientopfor.Commands(runner.console)
	var commandErr error
	if len(commands) != 1 || commands[0] == nil {
		commandErr = errors.New("OPFOR command is unavailable in the E2E client build")
	} else {
		command := commands[0]
		command.SilenceErrors = true
		command.SilenceUsage = true
		ctx, cancel := context.WithTimeout(runner.suite.ctx, runner.suite.opts.commandTimeout)
		defer cancel()
		command.SetContext(ctx)
		command.SetArgs(arguments)
		_, commandErr = command.ExecuteC()
	}

	_ = writePipe.Close()
	os.Stdout = previousStdout
	copyErr := <-copyDone
	_ = readPipe.Close()
	if copyErr != nil {
		commandErr = errors.Join(commandErr, fmt.Errorf("read captured OPFOR stdout: %w", copyErr))
	}
	return capturedOPFORCommand{output: capture.buffer.String(), err: commandErr, truncated: capture.truncated}
}

func validateCapturedCommand(result capturedOPFORCommand) error {
	if result.truncated {
		return fmt.Errorf("OPFOR output exceeded %d bytes", nativeBOFOutputLimit)
	}
	if result.err != nil {
		return result.err
	}
	return nil
}

func summarizeOutput(output string) string {
	const limit = 2048
	if len(output) <= limit {
		return output
	}
	return output[:limit] + "...[truncated summary]"
}

func outputFingerprint(output string) string {
	digest := sha256.Sum256([]byte(output))
	return fmt.Sprintf("captured_output_length=%d captured_output_sha256=%s", len(output), hex.EncodeToString(digest[:]))
}

func isDeadlineError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "deadline exceeded") || strings.Contains(message, "context deadline")
}

func waitForNativeBOF(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func requireSysmonChannelAbsent(ctx context.Context) error {
	const key = `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\WINEVT\Channels\Microsoft-Windows-Sysmon/Operational`
	queryCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	command := exec.CommandContext(queryCtx, "reg.exe", "query", key)
	output, err := command.CombinedOutput()
	if err == nil {
		return fmt.Errorf("refusing FindSysmon BOF: safety gate found registry key %s", key)
	}
	if queryCtx.Err() != nil {
		return fmt.Errorf("query Sysmon safety key: %w", queryCtx.Err())
	}
	exitError := &exec.ExitError{}
	message := strings.ToLower(string(output))
	if !errors.As(err, &exitError) || exitError.ExitCode() != 1 ||
		(!strings.Contains(message, "unable to find") && !strings.Contains(message, "cannot find")) {
		return fmt.Errorf("refusing FindSysmon BOF: registry absence was not proven: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func withSafeEmptyFirefoxProfile(run func() error) error {
	profileRoot, created, err := prepareSafeEmptyFirefoxProfile()
	if err != nil {
		return err
	}
	runErr := run()
	cleanupErr := removeCreatedFirefoxPaths(created)
	if cleanupErr != nil {
		cleanupErr = fmt.Errorf("remove synthetic Firefox profile %s: %w", profileRoot, cleanupErr)
	}
	return errors.Join(runErr, cleanupErr)
}

func prepareSafeEmptyFirefoxProfile() (string, []string, error) {
	configDirectory, err := os.UserConfigDir()
	if err != nil {
		return "", nil, fmt.Errorf("resolve current Windows user configuration directory: %w", err)
	}
	profileRoot := filepath.Join(configDirectory, "Mozilla", "Firefox", "Profiles")
	if err := requireAllFirefoxProfileRootsAbsent(configDirectory, profileRoot); err != nil {
		return "", nil, err
	}

	profile := filepath.Join(profileRoot, "sliver-opfor-e2e.empty")
	created, err := createTrackedDirectories(configDirectory, profile)
	if err != nil {
		_ = removeCreatedFirefoxPaths(created)
		return "", nil, err
	}
	return profile, created, nil
}

func requireAllFirefoxProfileRootsAbsent(configDirectory string, currentProfileRoot string) error {
	volume := filepath.VolumeName(configDirectory)
	if volume == "" {
		return errors.New("refusing FirefoxDump BOF: could not resolve the current configuration volume")
	}
	candidates := []string{currentProfileRoot}
	usersRoots := []string{
		filepath.Clean(`C:\Users`),
		filepath.Join(volume+string(os.PathSeparator), "Users"),
	}
	seenRoots := map[string]bool{}
	for _, usersRoot := range usersRoots {
		key := strings.ToLower(filepath.Clean(usersRoot))
		if seenRoots[key] {
			continue
		}
		seenRoots[key] = true
		if err := requireDirectoryNotLink(usersRoot); err != nil {
			return fmt.Errorf("refusing FirefoxDump BOF: literal/configuration users tree was not proven safe: %w", err)
		}
		users, err := os.ReadDir(usersRoot)
		if err != nil {
			return fmt.Errorf("refusing FirefoxDump BOF: enumerate %s: %w", usersRoot, err)
		}
		for _, user := range users {
			candidates = append(candidates, filepath.Join(usersRoot, user.Name(), "AppData", "Roaming", "Mozilla", "Firefox", "Profiles"))
		}
	}
	seen := map[string]bool{}
	for _, candidate := range candidates {
		key := strings.ToLower(filepath.Clean(candidate))
		if seen[key] {
			continue
		}
		seen[key] = true
		_, err := os.Lstat(candidate)
		switch {
		case err == nil:
			return fmt.Errorf("refusing FirefoxDump BOF: preexisting Firefox profile root %s", candidate)
		case os.IsNotExist(err):
			continue
		default:
			return fmt.Errorf("refusing FirefoxDump BOF: inspect %s: %w", candidate, err)
		}
	}
	return nil
}

func createTrackedDirectories(base string, target string) ([]string, error) {
	base = filepath.Clean(base)
	target = filepath.Clean(target)
	relative, err := filepath.Rel(base, target)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return nil, fmt.Errorf("synthetic Firefox profile %s is outside configuration root %s", target, base)
	}
	if err := requireDirectoryNotLink(base); err != nil {
		return nil, err
	}
	created := []string{}
	current := base
	for _, component := range strings.Split(relative, string(os.PathSeparator)) {
		current = filepath.Join(current, component)
		info, statErr := os.Lstat(current)
		if statErr == nil {
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return created, fmt.Errorf("refusing FirefoxDump BOF: path component %s is not a plain directory", current)
			}
			continue
		}
		if !os.IsNotExist(statErr) {
			return created, fmt.Errorf("inspect Firefox profile component %s: %w", current, statErr)
		}
		if err := os.Mkdir(current, 0o700); err != nil {
			return created, fmt.Errorf("create Firefox profile component %s: %w", current, err)
		}
		created = append(created, current)
		if err := requireDirectoryNotLink(current); err != nil {
			return created, err
		}
	}
	return created, nil
}

func requireDirectoryNotLink(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect Firefox profile path %s: %w", path, err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing FirefoxDump BOF: path %s is not a plain directory", path)
	}
	return nil
}

func removeCreatedFirefoxPaths(created []string) error {
	var cleanupErrors []error
	for index := len(created) - 1; index >= 0; index-- {
		if err := os.Remove(created[index]); err != nil && !os.IsNotExist(err) {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("remove %s non-recursively: %w", created[index], err))
		}
	}
	return errors.Join(cleanupErrors...)
}
