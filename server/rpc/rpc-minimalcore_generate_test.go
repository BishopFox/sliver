//go:build sliver_generate

package rpc

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/elf"
	"debug/macho"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/c2"
	"github.com/bishopfox/sliver/server/core"
)

// TestM7MinimalCoreLiveMTLSPing proves that an installed core outside the
// Sliver adapter can build, register over the existing wire protocol, and
// answer a real operator RPC. It runs only on the fixture's native targets.
func TestM7MinimalCoreLiveMTLSPing(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("minimal core fixture supports Darwin and Linux")
	}
	if runtime.GOOS == "darwin" && runtime.GOARCH != "arm64" ||
		runtime.GOOS == "linux" && runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		t.Skip("minimal core fixture does not support this architecture")
	}
	setupGenerateSpoofTest(t)
	port := startM7MinimalCoreLoopbackMTLS(t)

	name := fmt.Sprintf("m7-minimal-%d", time.Now().UnixNano())
	t.Cleanup(func() { _ = RemoveBuildByName(name) })
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Minute)
	defer cancel()
	rpc := &Server{}
	record, err := rpc.ExecuteBuild(ctx, &clientpb.BuildRequest{
		Name: name, Target: &clientpb.BuildTarget{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH},
		Mode: "session", Format: "exe", Core: &clientpb.BuildSpecRef{Name: "fixture/minimal-sliver-v1"},
		C2:       []*clientpb.BuildEndpoint{{Protocol: "mtls", Address: fmt.Sprintf("mtls://127.0.0.1:%d", port)}},
		Provider: &clientpb.BuildProviderPolicy{Mode: "pinned", Builder: "server"},
	})
	if err != nil {
		t.Fatalf("build minimal core: %v", err)
	}
	if record.State != clientpb.BuildRecordState_BUILD_RECORD_SUCCEEDED || record.Plan == nil ||
		len(record.Plan.Steps) != 1 || record.Plan.Steps[0].Spec == nil ||
		record.Plan.Steps[0].Spec.Name != "fixture/minimal-sliver-v1" ||
		record.FinalArtifact == nil || record.FinalArtifact.Type == nil ||
		record.FinalArtifact.Type.Kind != "executable" {
		t.Fatalf("minimal core did not produce a frozen executable record: %+v", record)
	}
	artifact, err := rpc.GetBuildArtifact(ctx, &clientpb.BuildRecordRef{ID: record.ID})
	if err != nil || artifact.File == nil || artifact.Metadata == nil || len(artifact.File.Data) == 0 {
		t.Fatalf("get minimal core artifact: %+v, %v", artifact, err)
	}
	if artifact.File.Name == "" || filepath.Base(artifact.File.Name) != artifact.File.Name {
		t.Fatalf("unsafe artifact filename %q", artifact.File.Name)
	}
	hash := sha256.Sum256(artifact.File.Data)
	if want := hex.EncodeToString(hash[:]); artifact.Metadata.SHA256 != want || record.FinalArtifact.SHA256 != want {
		t.Fatalf("minimal core artifact SHA-256 differs: record=%s retrieved=%s computed=%s",
			record.FinalArtifact.SHA256, artifact.Metadata.SHA256, want)
	}
	assertM7MinimalCoreLivePing(t, name, artifact.File.Name, artifact.File.Data, artifact.Metadata.SHA256)
}

// TestM7MinimalCoreAdvertisedTargets compiles every tuple in the installed
// fixture spec. Cross-target outputs are build-only evidence; the native live
// tests separately prove the wire contract.
func TestM7MinimalCoreAdvertisedTargets(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("cross-target fixture matrix runs on Linux")
	}
	setupGenerateSpoofTest(t)
	for _, target := range []struct {
		goos, goarch, container string
	}{
		{"linux", "amd64", "elf"},
		{"linux", "arm64", "elf"},
		{"darwin", "arm64", "macho"},
	} {
		t.Run(target.goos+"/"+target.goarch, func(t *testing.T) {
			name := fmt.Sprintf("m7-matrix-%s-%s-%d", target.goos, target.goarch, time.Now().UnixNano())
			t.Cleanup(func() { _ = RemoveBuildByName(name) })
			ctx, cancel := context.WithTimeout(t.Context(), 20*time.Minute)
			defer cancel()
			rpc := &Server{}
			record, err := rpc.ExecuteBuild(ctx, &clientpb.BuildRequest{
				Name: name, Target: &clientpb.BuildTarget{GOOS: target.goos, GOARCH: target.goarch},
				Mode: "session", Format: "exe", Core: &clientpb.BuildSpecRef{Name: "fixture/minimal-sliver-v1"},
				C2:       []*clientpb.BuildEndpoint{{Protocol: "mtls", Address: "mtls://127.0.0.1:8888"}},
				Provider: &clientpb.BuildProviderPolicy{Mode: "pinned", Builder: "server"},
			})
			if err != nil || record.State != clientpb.BuildRecordState_BUILD_RECORD_SUCCEEDED ||
				record.FinalArtifact == nil || record.FinalArtifact.Type == nil ||
				record.FinalArtifact.Type.Container != target.container {
				t.Fatalf("build advertised target %s/%s: record=%+v err=%v", target.goos, target.goarch, record, err)
			}
			artifact, err := rpc.GetBuildArtifact(ctx, &clientpb.BuildRecordRef{ID: record.ID})
			if err != nil || artifact.File == nil || artifact.Metadata == nil {
				t.Fatalf("get advertised target artifact: %+v, %v", artifact, err)
			}
			hash := sha256.Sum256(artifact.File.Data)
			if want := hex.EncodeToString(hash[:]); artifact.Metadata.SHA256 != want || record.FinalArtifact.SHA256 != want {
				t.Fatalf("advertised target SHA-256 mismatch: record=%s retrieved=%s computed=%s",
					record.FinalArtifact.SHA256, artifact.Metadata.SHA256, want)
			}
			switch target.container {
			case "elf":
				parsed, err := elf.NewFile(bytes.NewReader(artifact.File.Data))
				if err != nil {
					t.Fatalf("parse ELF: %v", err)
				}
				defer parsed.Close()
				wantMachine := elf.EM_AARCH64
				if target.goarch == "amd64" {
					wantMachine = elf.EM_X86_64
				}
				if parsed.Machine != wantMachine {
					t.Fatalf("ELF machine = %s, want %s", parsed.Machine, wantMachine)
				}
			case "macho":
				parsed, err := macho.NewFile(bytes.NewReader(artifact.File.Data))
				if err != nil {
					t.Fatalf("parse Mach-O: %v", err)
				}
				defer parsed.Close()
				if parsed.Cpu != macho.CpuArm64 {
					t.Fatalf("Mach-O CPU = %s, want arm64", parsed.Cpu)
				}
			}
			t.Logf("%s/%s build-only artifact: %d bytes, SHA-256 %s", target.goos, target.goarch,
				len(artifact.File.Data), artifact.Metadata.SHA256)
		})
	}
}

// startM7MinimalCoreLoopbackMTLS creates only a test-owned localhost listener.
// Callers initialize server CAs before invoking it.
func startM7MinimalCoreLoopbackMTLS(t *testing.T) int {
	t.Helper()
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := probe.Addr().(*net.TCPAddr).Port
	if err := probe.Close(); err != nil {
		t.Fatal(err)
	}
	job, err := c2.StartMTLSListenerJob(&clientpb.MTLSListenerReq{Host: "127.0.0.1", Port: uint32(port)})
	if err != nil {
		t.Fatalf("start loopback mTLS listener: %v", err)
	}
	t.Cleanup(func() {
		select {
		case job.JobCtrl <- true:
		case <-time.After(2 * time.Second):
			t.Error("mTLS listener did not stop")
		}
	})
	return port
}

// assertM7MinimalCoreLivePing runs an accepted artifact with a private process
// environment and proves its declared Ping capability over the live listener.
func assertM7MinimalCoreLivePing(t *testing.T, name string, fileName string, data []byte, sha256Hex string) {
	t.Helper()
	if fileName == "" || filepath.Base(fileName) != fileName {
		t.Fatalf("unsafe artifact filename %q", fileName)
	}
	path := filepath.Join(t.TempDir(), fileName)
	if err := os.WriteFile(path, data, 0700); err != nil {
		t.Fatal(err)
	}

	prior := make(map[string]bool)
	for _, session := range core.Sessions.All() {
		prior[session.ID] = true
	}
	registeredSessionID := ""
	t.Cleanup(func() {
		if registeredSessionID != "" {
			core.Sessions.Remove(registeredSessionID)
		}
	})
	processCtx, stopProcess := context.WithCancel(context.Background())
	command := exec.CommandContext(processCtx, path)
	command.Dir = t.TempDir()
	command.Env = []string{"HOME=" + t.TempDir(), "TMPDIR=" + t.TempDir()}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		stopProcess()
		t.Fatalf("start minimal core artifact: %v", err)
	}
	processDone := make(chan error, 1)
	go func() { processDone <- command.Wait() }()
	processExited := false
	t.Cleanup(func() {
		stopProcess()
		if processExited {
			return
		}
		select {
		case <-processDone:
		case <-time.After(5 * time.Second):
			t.Error("minimal core process did not stop")
		}
	})

	deadline := time.NewTimer(30 * time.Second)
	defer deadline.Stop()
	var session *core.Session
	for session == nil {
		for _, candidate := range core.Sessions.All() {
			if !prior[candidate.ID] && candidate.Name == name {
				session = candidate
				break
			}
		}
		if session != nil {
			break
		}
		select {
		case err := <-processDone:
			processExited = true
			t.Fatalf("minimal core exited before registration: %v, stderr=%s", err, stderr.String())
		case <-deadline.C:
			t.Fatalf("minimal core did not register a session; stderr=%s", stderr.String())
		case <-time.After(50 * time.Millisecond):
		}
	}
	registeredSessionID = session.ID
	pingCtx, pingCancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer pingCancel()
	const nonce = int32(0x5a17c0de)
	rpc := &Server{}
	response, err := rpc.Ping(pingCtx, &sliverpb.Ping{
		Nonce: nonce, Request: &commonpb.Request{SessionID: session.ID, Timeout: int64(5 * time.Second)},
	})
	if err != nil || response == nil || response.Nonce != nonce {
		t.Fatalf("minimal core sliver-v1 Ping failed: response=%+v err=%v stderr=%s", response, err, stderr.String())
	}
	t.Logf("minimal core %s registered session %s and answered Ping; artifact SHA-256 %s",
		runtime.GOOS+"/"+runtime.GOARCH, session.ID, sha256Hex)
}
