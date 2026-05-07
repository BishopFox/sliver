package rpc

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Binject/debug/pe"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/db"
	generatePkg "github.com/bishopfox/sliver/server/generate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

var (
	spoofGenerateSetupOnce sync.Once
	spoofGenerateSetupErr  error
)

func TestGenerateSpoofMetadataAppliesPETimestampOverBufnet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping generate/spoof integration test in -short mode")
	}

	setupGenerateSpoofTest(t)

	client, cleanup := newBufnetRPCClient(t)
	defer cleanup()

	buildName := fmt.Sprintf("rpc-spoof-metadata-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		_ = RemoveBuildByName(buildName)
		_ = os.RemoveAll(filepath.Join(generatePkg.GetSliversDir(), "windows", "amd64", buildName))
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	generateResp, err := client.Generate(ctx, &clientpb.GenerateReq{
		Name: buildName,
		Config: &clientpb.ImplantConfig{
			GOOS:             "windows",
			GOARCH:           "amd64",
			Format:           clientpb.OutputFormat_EXECUTABLE,
			Debug:            true,
			ObfuscateSymbols: false,
			C2: []*clientpb.ImplantC2{
				{URL: "http://127.0.0.1"},
			},
			HTTPC2ConfigName: consts.DefaultC2Profile,
		},
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	origData := generateResp.GetFile().GetData()
	if len(origData) == 0 {
		t.Fatal("Generate() returned empty file data")
	}
	origTimestamp, err := peFileTimestamp(origData)
	if err != nil {
		t.Fatalf("parse generated PE timestamp: %v", err)
	}

	// Create a donor PE by modifying only the PE header timestamp.
	spoofTimestamp := origTimestamp + 1337
	donorData, err := setPEFileTimestamp(origData, spoofTimestamp)
	if err != nil {
		t.Fatalf("set donor PE timestamp: %v", err)
	}
	donorTimestamp, err := peFileTimestamp(donorData)
	if err != nil {
		t.Fatalf("parse donor PE timestamp: %v", err)
	}
	if donorTimestamp != spoofTimestamp {
		t.Fatalf("donor timestamp mismatch: got=%d want=%d", donorTimestamp, spoofTimestamp)
	}

	spoofReq := &clientpb.GenerateSpoofMetadataReq{
		ImplantBuildID: generateResp.GetImplantBuildID(),
		SpoofMetadata: &clientpb.SpoofMetadataConfig{
			PE: &clientpb.PESpoofMetadataConfig{
				Source: &clientpb.SpoofMetadataFile{
					Name: "donor.exe",
					Data: donorData,
				},
			},
		},
	}
	if spoofReq.GetImplantBuildID() == "" {
		t.Fatal("Generate() returned empty ImplantBuildID")
	}

	if _, err := client.GenerateSpoofMetadata(ctx, spoofReq); err != nil {
		t.Fatalf("GenerateSpoofMetadata() error: %v", err)
	}

	regenerated, err := client.Regenerate(ctx, &clientpb.RegenerateReq{ImplantName: buildName})
	if err != nil {
		t.Fatalf("Regenerate() error: %v", err)
	}
	modifiedData := regenerated.GetFile().GetData()
	if len(modifiedData) == 0 {
		t.Fatal("Regenerate() returned empty file data")
	}

	modifiedTimestamp, err := peFileTimestamp(modifiedData)
	if err != nil {
		t.Fatalf("parse modified PE timestamp: %v", err)
	}
	if modifiedTimestamp != spoofTimestamp {
		t.Fatalf("spoofed PE timestamp mismatch: got=%d want=%d", modifiedTimestamp, spoofTimestamp)
	}
	if modifiedTimestamp == origTimestamp {
		t.Fatalf("expected PE timestamp to change from %d", origTimestamp)
	}
}

func setupGenerateSpoofTest(t *testing.T) {
	t.Helper()
	spoofGenerateSetupOnce.Do(func() {
		assets.Setup(false, false)
		certs.SetupCAs()
		_, err := db.LoadHTTPC2ConfigByName(consts.DefaultC2Profile)
		if err == nil {
			return
		}
		spoofGenerateSetupErr = db.SaveHTTPC2Config(configs.GenerateDefaultHTTPC2Config())
	})
	if spoofGenerateSetupErr != nil {
		t.Fatalf("setup generate/spoof test prerequisites: %v", spoofGenerateSetupErr)
	}
}

func newBufnetRPCClient(t *testing.T) (rpcpb.SliverRPCClient, func()) {
	t.Helper()

	ln := bufconn.Listen(2 * 1024 * 1024)
	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(64<<20),
		grpc.MaxSendMsgSize(64<<20),
	)
	rpcpb.RegisterSliverRPCServer(grpcServer, &Server{})

	go func() {
		_ = grpcServer.Serve(ln)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	conn, err := dialBufConn(ctx, ln)
	cancel()
	if err != nil {
		grpcServer.Stop()
		_ = ln.Close()
		t.Fatalf("dial grpc/bufconn: %v", err)
	}

	cleanup := func() {
		_ = conn.Close()
		grpcServer.Stop()
		_ = ln.Close()
	}
	return rpcpb.NewSliverRPCClient(conn), cleanup
}

func dialBufConn(ctx context.Context, ln *bufconn.Listener) (*grpc.ClientConn, error) {
	dialer := func(context.Context, string) (net.Conn, error) { return ln.Dial() }
	return grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(64<<20),
			grpc.MaxCallSendMsgSize(64<<20),
		),
	)
}

func peFileTimestamp(data []byte) (uint32, error) {
	peFile, err := pe.NewFile(bytes.NewReader(data))
	if err != nil {
		return 0, err
	}
	defer peFile.Close()
	return peFile.FileHeader.TimeDateStamp, nil
}

func setPEFileTimestamp(data []byte, timestamp uint32) ([]byte, error) {
	if len(data) < 0x40 {
		return nil, fmt.Errorf("invalid PE data length: %d", len(data))
	}
	cloned := append([]byte(nil), data...)
	peHeaderOffset := int(binary.LittleEndian.Uint32(cloned[0x3c:0x40]))
	if peHeaderOffset < 0 || peHeaderOffset+12 > len(cloned) {
		return nil, fmt.Errorf("invalid PE header offset: %d", peHeaderOffset)
	}
	binary.LittleEndian.PutUint32(cloned[peHeaderOffset+8:peHeaderOffset+12], timestamp)
	return cloned, nil
}
