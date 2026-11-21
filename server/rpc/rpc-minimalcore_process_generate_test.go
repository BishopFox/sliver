//go:build sliver_generate

package rpc

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/buildcatalog"
	"github.com/bishopfox/sliver/server/builder"
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/buildscheduler"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/db"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/proto"
)

const m7WorkerChildEnv = "SLIVER_M7_REMOTE_WORKER_CHILD"

type m7ProcessOutput struct {
	sync.Mutex
	bytes.Buffer
}

func (output *m7ProcessOutput) Write(data []byte) (int, error) {
	output.Lock()
	defer output.Unlock()
	return output.Buffer.Write(data)
}

func (output *m7ProcessOutput) String() string {
	output.Lock()
	defer output.Unlock()
	return output.Buffer.String()
}

// TestM7RemoteMinimalCoreSeparateProcessWorker verifies the assigned core over
// an actual mTLS gRPC connection to a worker running in a different process.

//nolint:gocyclo // The test owns a separate process and checks its full registration and execution lifecycle.
func TestM7RemoteMinimalCoreSeparateProcessWorker(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" ||
		runtime.GOOS == "darwin" && runtime.GOARCH != "arm64" ||
		runtime.GOOS == "linux" && runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		t.Skip("minimal core fixture does not support this native target")
	}
	setupGenerateSpoofTest(t)
	port := startM7MinimalCoreLoopbackMTLS(t)
	rpc := &Server{}
	operatorName := fmt.Sprintf("m7-process-operator-%d", time.Now().UnixNano())
	workerName := fmt.Sprintf("m7-process-worker-%d", time.Now().UnixNano())
	caPEM, _, err := certs.GetCertificateAuthorityPEM(certs.OperatorCA)
	if err != nil {
		t.Fatal(err)
	}
	serverPEM, serverKeyPEM, err := certs.OperatorServerGetCertificate("127.0.0.1")
	if errors.Is(err, certs.ErrCertDoesNotExist) {
		serverPEM, serverKeyPEM, err = certs.OperatorServerGenerateCertificate("127.0.0.1")
	}
	if err != nil {
		t.Fatal(err)
	}
	clientPEM, clientKeyPEM, err := certs.OperatorClientGenerateCertificate(operatorName)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = certs.OperatorClientRemoveCertificate(operatorName) })
	serverCert, err := tls.X509KeyPair(serverPEM, serverKeyPEM)
	if err != nil {
		t.Fatal(err)
	}
	clientCert, err := tls.X509KeyPair(clientPEM, clientKeyPEM)
	if err != nil {
		t.Fatal(err)
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		t.Fatal("operator CA certificate is invalid")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	serverTLS := &tls.Config{
		Certificates: []tls.Certificate{serverCert}, ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs: caPool, MinVersion: tls.VersionTLS13,
		VerifyConnection: func(state tls.ConnectionState) error {
			return certs.ValidateOperatorClientCertificate(state.PeerCertificates)
		},
	}
	grpcServer := grpc.NewServer(grpc.Creds(credentials.NewTLS(serverTLS)),
		grpc.MaxRecvMsgSize(64<<20), grpc.MaxSendMsgSize(64<<20))
	rpcpb.RegisterSliverRPCServer(grpcServer, rpc)
	serveDone := make(chan error, 1)
	go func() { serveDone <- grpcServer.Serve(listener) }()
	t.Cleanup(func() {
		grpcServer.Stop()
		_ = listener.Close()
		select {
		case err := <-serveDone:
			if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
				t.Errorf("worker gRPC server stopped: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("worker gRPC server did not stop")
		}
	})

	childDir := t.TempDir()
	caPath := writeM7ProcessPEM(t, childDir, "ca.pem", caPEM)
	certPath := writeM7ProcessPEM(t, childDir, "client.pem", clientPEM)
	keyPath := writeM7ProcessPEM(t, childDir, "client-key.pem", clientKeyPEM)
	address := listener.Addr().String()
	childCtx, stopChild := context.WithCancel(context.Background())
	child := exec.CommandContext(childCtx, os.Args[0], "-test.run=^TestM7RemoteMinimalCoreSeparateProcessChild$", "-test.timeout=20m")
	child.Env = append(os.Environ(), m7WorkerChildEnv+"=1", "SLIVER_M7_WORKER_ADDR="+address,
		"SLIVER_M7_WORKER_NAME="+workerName, "SLIVER_M7_WORKER_CA="+caPath,
		"SLIVER_M7_WORKER_CERT="+certPath, "SLIVER_M7_WORKER_KEY="+keyPath)
	var childOutput m7ProcessOutput
	child.Stdout, child.Stderr = &childOutput, &childOutput
	if err := child.Start(); err != nil {
		stopChild()
		t.Fatalf("start separate worker: %v", err)
	}
	childDone := make(chan struct{})
	var childErr error
	go func() {
		childErr = child.Wait()
		close(childDone)
	}()
	t.Cleanup(func() {
		stopChild()
		select {
		case <-childDone:
		case <-time.After(5 * time.Second):
			_ = child.Process.Kill()
			<-childDone
			t.Error("separate worker did not stop promptly")
		}
	})

	clientTLS := &tls.Config{Certificates: []tls.Certificate{clientCert}, RootCAs: caPool,
		ServerName: "127.0.0.1", MinVersion: tls.VersionTLS13}
	dialCtx, cancelDial := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancelDial()
	conn, err := connectWorkerTestClient(dialCtx, address, clientTLS)
	if err != nil {
		t.Fatalf("dial authenticated worker RPC transport: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	client := rpcpb.NewSliverRPCClient(conn)

	catalog, err := rpc.buildCatalog()
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := catalog.Get(buildcatalog.Reference{Name: "fixture/minimal-sliver-v1"})
	if err != nil || snapshot.Core == nil {
		t.Fatalf("installed minimal core spec is missing: %v", err)
	}
	target := buildplan.Target{OS: runtime.GOOS, Arch: runtime.GOARCH}
	var artifact buildplan.Artifact
	var toolchain buildplan.Toolchain
	for _, capability := range snapshot.Core.Capabilities {
		if capability.Mode != "session" || !slices.Contains(capability.Protocols, "mtls") {
			continue
		}
		for _, candidate := range capability.Targets {
			if candidate.OS == target.OS && candidate.Arch == target.Arch {
				artifact = buildplan.Artifact{Kind: capability.Output.Artifact.Kind,
					Container: capability.Output.Artifact.Container, GoBuildMode: capability.Output.Artifact.GoBuildMode,
					EntrySymbols: slices.Clone(capability.Output.Artifact.EntrySymbols), Target: target}
				toolchain = buildplan.Toolchain{Driver: capability.Toolchain.Driver,
					Requirements: slices.Clone(capability.Toolchain.Requirements)}
				break
			}
		}
	}
	if artifact.Kind != "executable" || toolchain.Driver == "" {
		t.Fatalf("minimal core does not advertise native executable for %s/%s", target.OS, target.Arch)
	}
	ref := buildplan.Reference{Name: snapshot.Reference.Name, Version: snapshot.Reference.Version, Digest: snapshot.Reference.Digest}
	requirement := buildscheduler.Requirement{Spec: ref, Target: target, Output: artifact,
		Mode: "session", Protocols: []string{"mtls"}, Toolchain: buildscheduler.Toolchain{
			Driver: toolchain.Driver, Requirements: slices.Clone(toolchain.Requirements)}}
	scheduler, err := rpc.stepScheduler()
	if err != nil {
		t.Fatal(err)
	}
	readyDeadline := time.NewTimer(30 * time.Second)
	defer readyDeadline.Stop()
	for len(scheduler.Candidates(requirement, workerName, time.Now())) != 1 {
		select {
		case <-childDone:
			t.Fatalf("separate worker exited before registration: %v; output: %s", childErr, childOutput.String())
		case <-readyDeadline.C:
			t.Fatalf("separate worker did not advertise exact minimal-core tuple; output: %s", childOutput.String())
		case <-time.After(20 * time.Millisecond):
		}
	}

	name := fmt.Sprintf("m7-minimal-process-%d", time.Now().UnixNano())
	t.Cleanup(func() { _ = RemoveBuildByName(name) })
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Minute)
	defer cancel()
	record, err := client.ExecuteBuild(ctx, &clientpb.BuildRequest{
		Name: name, Target: &clientpb.BuildTarget{GOOS: target.OS, GOARCH: target.Arch},
		Mode: "session", Format: "exe", Core: &clientpb.BuildSpecRef{Name: ref.Name, Version: ref.Version, Digest: ref.Digest},
		C2:       []*clientpb.BuildEndpoint{{Protocol: "mtls", Address: fmt.Sprintf("mtls://127.0.0.1:%d", port)}},
		Provider: &clientpb.BuildProviderPolicy{Mode: "pinned", Builder: workerName},
	})
	if err != nil {
		t.Fatalf("execute separate-process minimal core: %v; worker output: %s", err, childOutput.String())
	}
	if record == nil || record.State != clientpb.BuildRecordState_BUILD_RECORD_SUCCEEDED || record.Plan == nil || record.FinalArtifact == nil ||
		len(record.Plan.Steps) != 1 || record.Plan.Steps[0].Provider != workerName || record.Plan.Steps[0].Spec == nil ||
		record.Plan.Steps[0].Spec.Name != ref.Name || record.Plan.Steps[0].Spec.Version != ref.Version ||
		record.Plan.Steps[0].Spec.Digest != ref.Digest || len(record.Steps) != 1 ||
		record.Steps[0].Selection == nil || record.Steps[0].Selection.Builder != workerName {
		t.Fatalf("minimal core was not built by the separate worker with exact spec: %+v", record)
	}
	result, err := client.GetBuildArtifact(ctx, &clientpb.BuildRecordRef{ID: record.ID})
	if err != nil || result == nil || result.File == nil || result.Metadata == nil || len(result.File.Data) == 0 {
		t.Fatalf("get separate-process artifact: %+v, %v", result, err)
	}
	hash := sha256.Sum256(result.File.Data)
	checksum := hex.EncodeToString(hash[:])
	if result.File.Name == "" || filepath.Base(result.File.Name) != result.File.Name ||
		result.File.Name != result.Metadata.FileName || result.File.Name != record.FinalArtifact.FileName ||
		result.Metadata.SizeBytes != uint64(len(result.File.Data)) || record.FinalArtifact.SizeBytes != uint64(len(result.File.Data)) ||
		result.Metadata.SHA256 != checksum || record.FinalArtifact.SHA256 != checksum ||
		result.Metadata.Type == nil || record.FinalArtifact.Type == nil ||
		!proto.Equal(result.Metadata.Type, record.FinalArtifact.Type) {
		t.Fatalf("separate-process artifact differs from record or metadata: record=%+v metadata=%+v", record.FinalArtifact, result.Metadata)
	}
	build, err := db.ImplantBuildByName(name)
	if err != nil || build == nil || build.PeerPrivateKey == "" || build.ID != record.ID {
		t.Fatalf("remote core did not register runtime authentication: %+v, %v", build, err)
	}
	assertM7MinimalCoreLivePing(t, name, result.File.Name, result.File.Data, checksum)
	t.Logf("separate-process worker PID %d used spec %s@%s digest %s; artifact SHA-256 %s",
		child.Process.Pid, ref.Name, ref.Version, ref.Digest, checksum)
}

func writeM7ProcessPEM(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// The test binary's child mode runs only the installed worker. It never uses
// the parent process's direct-RPC adapter or in-memory scheduler state.
func TestM7RemoteMinimalCoreSeparateProcessChild(t *testing.T) {
	if os.Getenv(m7WorkerChildEnv) != "1" {
		t.Skip("worker child mode only")
	}
	assets.Setup(false, false)
	caPEM, err := os.ReadFile(os.Getenv("SLIVER_M7_WORKER_CA"))
	if err != nil {
		t.Fatal(err)
	}
	certPEM, err := os.ReadFile(os.Getenv("SLIVER_M7_WORKER_CERT"))
	if err != nil {
		t.Fatal(err)
	}
	keyPEM, err := os.ReadFile(os.Getenv("SLIVER_M7_WORKER_KEY"))
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		t.Fatal("worker CA certificate is invalid")
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{certificate}, RootCAs: caPool,
		ServerName: "127.0.0.1", MinVersion: tls.VersionTLS13}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	conn, err := connectWorkerTestClient(ctx, os.Getenv("SLIVER_M7_WORKER_ADDR"), tlsConfig)
	if err != nil {
		t.Fatalf("dial worker RPC: %v", err)
	}
	defer func() { _ = conn.Close() }()
	config := &clientpb.Builder{Name: os.Getenv("SLIVER_M7_WORKER_NAME")}
	if err := builder.ConfigureInstalledCapabilities(config); err != nil {
		t.Fatalf("configure installed worker capabilities: %v", err)
	}
	config.Status.AvailableMemoryBytes = 4_000_000_000
	worker := builder.NewBuilder(config, &sync.Mutex{}, rpcpb.NewSliverRPCClient(conn), conn)
	worker.Start()
}

func connectWorkerTestClient(ctx context.Context, address string, config *tls.Config) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient("passthrough:///"+address,
		grpc.WithTransportCredentials(credentials.NewTLS(config)),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(64<<20), grpc.MaxCallSendMsgSize(64<<20)))
	if err != nil {
		return nil, err
	}
	conn.Connect()
	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			return conn, nil
		}
		if state == connectivity.Idle {
			conn.Connect()
		}
		if !conn.WaitForStateChange(ctx, state) {
			_ = conn.Close()
			return nil, ctx.Err()
		}
	}
}
