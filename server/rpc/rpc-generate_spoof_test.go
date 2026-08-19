package rpc

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
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
	"github.com/bishopfox/sliver/server/db/models"
	generatePkg "github.com/bishopfox/sliver/server/generate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
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
	persistedBuild, err := db.ImplantBuildByName(buildName)
	if err != nil {
		t.Fatalf("load generated implant build: %v", err)
	}
	if persistedBuild.ImplantConfigID == "" {
		t.Fatal("generated implant build has an empty config ID")
	}
	persistedConfig, err := db.ImplantConfigByID(persistedBuild.ImplantConfigID)
	if err != nil {
		t.Fatalf("load generated implant config: %v", err)
	}
	if persistedConfig.ID != persistedBuild.ImplantConfigID {
		t.Fatalf("persisted config ID = %q, build config ID = %q", persistedConfig.ID, persistedBuild.ImplantConfigID)
	}
	if !bytes.Contains(origData, []byte(persistedConfig.ID)) {
		t.Fatalf("generated artifact does not contain persisted config ID %q", persistedConfig.ID)
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

func TestGenerateCompileFailureDoesNotPersistPreselectedConfig(t *testing.T) {
	setupGenerateSpoofTest(t)
	var configCountBefore int64
	if err := db.Session().Model(&models.ImplantConfig{}).Count(&configCountBefore).Error; err != nil {
		t.Fatalf("count implant configs before Generate(): %v", err)
	}

	buildName := fmt.Sprintf("rpc-config-id-failure-%d", time.Now().UnixNano())
	config := &clientpb.ImplantConfig{
		GOOS:             "unsupported",
		GOARCH:           "amd64",
		Format:           clientpb.OutputFormat_EXECUTABLE,
		Debug:            true,
		ObfuscateSymbols: false,
		C2: []*clientpb.ImplantC2{
			{URL: "http://127.0.0.1"},
		},
		HTTPC2ConfigName: consts.DefaultC2Profile,
	}

	_, err := (&Server{}).Generate(context.Background(), &clientpb.GenerateReq{
		Name:   buildName,
		Config: config,
	})
	if err == nil {
		t.Fatal("Generate() unexpectedly succeeded for an unsupported compiler target")
	}
	if !strings.Contains(err.Error(), "invalid compiler target") {
		t.Fatalf("Generate() error = %v, want invalid compiler target", err)
	}
	var configCountAfter int64
	if err := db.Session().Model(&models.ImplantConfig{}).Count(&configCountAfter).Error; err != nil {
		t.Fatalf("count implant configs after Generate(): %v", err)
	}
	if configCountAfter != configCountBefore {
		t.Fatalf("implant config count after failed Generate() = %d, want %d", configCountAfter, configCountBefore)
	}
}

func TestGenerateUsesPersistedProfileBuildSettings(t *testing.T) {
	setupGenerateSpoofTest(t)
	profileName := fmt.Sprintf("rpc-profile-authority-%d", time.Now().UnixNano())
	profile, err := generatePkg.SaveImplantProfile(&clientpb.ImplantProfile{
		Name: profileName,
		Config: &clientpb.ImplantConfig{
			GOOS:               "linux",
			GOARCH:             "amd64",
			Format:             clientpb.OutputFormat(99),
			TemplateName:       generatePkg.SliverTemplateName,
			HTTPC2ConfigName:   consts.DefaultC2Profile,
			ObfuscateSymbols:   true,
			ConnectionStrategy: "s",
			C2: []*clientpb.ImplantC2{
				{URL: "http://127.0.0.1"},
			},
		},
	})
	if err != nil {
		t.Fatalf("SaveImplantProfile() error = %v", err)
	}
	t.Cleanup(func() { _ = db.DeleteProfile(profileName) })

	_, err = (&Server{}).Generate(context.Background(), &clientpb.GenerateReq{
		Name: fmt.Sprintf("rpc-profile-authority-build-%d", time.Now().UnixNano()),
		Config: &clientpb.ImplantConfig{
			ID:               profile.Config.ID,
			Format:           clientpb.OutputFormat_EXECUTABLE,
			HTTPC2ConfigName: "does-not-exist",
		},
	})
	if status.Code(err) != codes.InvalidArgument || !strings.Contains(err.Error(), "invalid output format") {
		t.Fatalf("Generate() error = %v, code = %s; want persisted profile's invalid output format", err, status.Code(err))
	}
}

func TestGenerateRejectsUnpersistedProfileAssociation(t *testing.T) {
	_, err := (&Server{}).Generate(context.Background(), &clientpb.GenerateReq{
		Name: fmt.Sprintf("rpc-profile-trust-%d", time.Now().UnixNano()),
		Config: &clientpb.ImplantConfig{
			ImplantProfileID: "502f6ad5-1ff1-49cc-bbb5-8a6ea4533661",
		},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("Generate() error = %v, code = %s; want %s", err, status.Code(err), codes.InvalidArgument)
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
