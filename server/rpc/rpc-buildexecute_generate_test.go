//go:build sliver_generate

package rpc

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/generate"
)

// TestExecuteBuildSliverNativeExecutable exercises the installed Sliver core
// through the neutral RPC and checks the bytes frozen for both new and legacy
// artifact retrieval.
func TestExecuteBuildSliverNativeExecutable(t *testing.T) {
	if !generate.SupportedCompilerTargets[runtime.GOOS+"/"+runtime.GOARCH] {
		t.Skip("native host is not a supported Sliver target")
	}
	setupGenerateSpoofTest(t)
	httpConfig, err := db.LoadHTTPC2ConfigByName("default")
	if err != nil || httpConfig == nil || httpConfig.Name == "" {
		if err := db.SaveHTTPC2Config(configs.GenerateDefaultHTTPC2Config()); err != nil {
			t.Fatalf("prepare default HTTP C2 configuration: %v", err)
		}
	}

	name := fmt.Sprintf("m3-native-rpc-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		_ = RemoveBuildByName(name)
		_ = os.RemoveAll(filepath.Join(generate.GetSliversDir(), runtime.GOOS, runtime.GOARCH, name))
	})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	rpc := &Server{}
	record, err := rpc.ExecuteBuild(ctx, &clientpb.BuildRequest{
		Name: name,
		Target: &clientpb.BuildTarget{
			GOOS: runtime.GOOS, GOARCH: runtime.GOARCH,
		},
		Mode:   "session",
		Format: "exe",
		C2: []*clientpb.BuildEndpoint{{
			Protocol: "mtls", Address: "mtls://127.0.0.1:8888",
		}},
		CoreOptions: []*clientpb.BuildOption{
			{Name: "debug", ValueJSON: []byte("true")},
			{Name: "obfuscateSymbols", ValueJSON: []byte("false")},
		},
	})
	if err != nil {
		t.Fatalf("ExecuteBuild: %v", err)
	}
	if record.State != clientpb.BuildRecordState_BUILD_RECORD_SUCCEEDED {
		t.Fatalf("Sliver build did not succeed: %+v", record)
	}
	workspaceParent := filepath.Join(generate.GetSliversDir(), runtime.GOOS, runtime.GOARCH, name)
	entries, err := os.ReadDir(workspaceParent)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("inspect private build workspace: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("private build workspace and material remain after artifact acceptance: %v", entries)
	}
	if record.ID == "" || record.Plan == nil || len(record.Plan.Steps) != 1 || record.Plan.Steps[0].Spec == nil ||
		record.Plan.Steps[0].Spec.Name != "builtin/sliver" || record.Plan.Steps[0].Spec.Digest == "" ||
		len(record.Steps) != 1 || record.Steps[0].ToolchainIdentity == "" {
		t.Fatalf("Sliver core executor or toolchain was not recorded: %+v", record)
	}
	wantName := name
	if runtime.GOOS == "windows" {
		wantName += ".exe"
	}
	if record.FinalArtifact == nil || record.FinalArtifact.FileName != wantName ||
		record.FinalArtifact.Type == nil || record.FinalArtifact.Type.Kind != "executable" ||
		record.FinalArtifact.Type.GOOS != runtime.GOOS || record.FinalArtifact.Type.GOARCH != runtime.GOARCH {
		t.Fatalf("incorrect native executable metadata: %+v", record.FinalArtifact)
	}

	artifact, err := rpc.GetBuildArtifact(ctx, &clientpb.BuildRecordRef{ID: record.ID})
	if err != nil {
		t.Fatalf("GetBuildArtifact: %v", err)
	}
	if artifact.File == nil || artifact.Metadata == nil || len(artifact.File.Data) == 0 ||
		artifact.File.Name != wantName {
		t.Fatalf("stored executable is missing or misnamed: %+v", artifact)
	}
	hash := sha256.Sum256(artifact.File.Data)
	wantHash := hex.EncodeToString(hash[:])
	if artifact.Metadata.SHA256 != wantHash || record.FinalArtifact.SHA256 != wantHash ||
		artifact.Metadata.SizeBytes != uint64(len(artifact.File.Data)) {
		t.Fatalf("stored executable checksum or size differs from the record: %+v", artifact.Metadata)
	}
	switch runtime.GOOS {
	case "linux":
		if _, err := elf.NewFile(bytes.NewReader(artifact.File.Data)); err != nil {
			t.Fatalf("native output is not ELF: %v", err)
		}
	case "darwin":
		if _, err := macho.NewFile(bytes.NewReader(artifact.File.Data)); err != nil {
			t.Fatalf("native output is not Mach-O: %v", err)
		}
	case "windows":
		if _, err := pe.NewFile(bytes.NewReader(artifact.File.Data)); err != nil {
			t.Fatalf("native output is not PE: %v", err)
		}
	}

	legacy, err := rpc.Regenerate(ctx, &clientpb.RegenerateReq{ImplantName: name})
	if err != nil {
		t.Fatalf("Regenerate: %v", err)
	}
	if legacy.File == nil || legacy.File.Name != wantName || !bytes.Equal(legacy.File.Data, artifact.File.Data) {
		t.Fatal("legacy regenerate did not return the exact recorded executable")
	}
}

func TestM4GenerateShellcodeFinalParity(t *testing.T) {
	setupGenerateSpoofTest(t)
	targets := []struct{ os, arch string }{}
	switch runtime.GOOS {
	case "linux":
		targets = append(targets, struct{ os, arch string }{"linux", "amd64"})
	case "darwin":
		if runtime.GOARCH == "arm64" {
			targets = append(targets, struct{ os, arch string }{"darwin", "arm64"})
		}
	}
	// Windows PIE is cross-compiled by Go without a C compiler.
	targets = append(targets, struct{ os, arch string }{"windows", "amd64"})
	for _, target := range targets {
		t.Run(target.os+"/"+target.arch, func(t *testing.T) {
			name := fmt.Sprintf("m4-shellcode-%s-%d", target.os, time.Now().UnixNano())
			t.Cleanup(func() {
				_ = RemoveBuildByName(name)
				_ = os.RemoveAll(filepath.Join(generate.GetSliversDir(), target.os, target.arch, name))
			})
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
			defer cancel()
			rpc := &Server{}
			generated, err := rpc.Generate(ctx, &clientpb.GenerateReq{Name: name, Config: &clientpb.ImplantConfig{
				GOOS: target.os, GOARCH: target.arch, Format: clientpb.OutputFormat_SHELLCODE,
				Debug: true, ObfuscateSymbols: false, ShellcodeEncoder: clientpb.ShellcodeEncoder_XOR,
				C2: []*clientpb.ImplantC2{{URL: "mtls://127.0.0.1:8888"}}, HTTPC2ConfigName: "default",
			}})
			if err != nil {
				t.Fatal(err)
			}
			if generated.GetFile() == nil || len(generated.GetFile().GetData()) == 0 || generated.GetImplantBuildID() == "" {
				t.Fatalf("missing generated shellcode artifact: %+v", generated)
			}
			record, err := rpc.GetBuildRecord(ctx, &clientpb.BuildRecordRef{ID: generated.ImplantBuildID})
			if err != nil {
				t.Fatal(err)
			}
			wantConverter := map[string]string{"windows": "builtin/donut", "linux": "builtin/malasada", "darwin": "builtin/beignet"}[target.os]
			if record.State != clientpb.BuildRecordState_BUILD_RECORD_SUCCEEDED || len(record.Steps) != 3 ||
				record.Plan.Steps[1].Spec.Name != wantConverter || record.Plan.Steps[2].Spec.Name != "builtin/xor" {
				t.Fatalf("unexpected typed shellcode route: %+v", record.Plan)
			}
			hash := sha256.Sum256(generated.File.Data)
			wantHash := hex.EncodeToString(hash[:])
			if record.FinalArtifact.SHA256 != wantHash || record.Steps[2].Artifact.SHA256 != wantHash ||
				record.Steps[0].Artifact == nil || record.Steps[1].Artifact == nil {
				t.Fatalf("record omits observed step or final checksum: %+v", record)
			}
			stored, err := rpc.GetBuildArtifact(ctx, &clientpb.BuildRecordRef{ID: generated.ImplantBuildID})
			if err != nil || stored.File == nil || !bytes.Equal(stored.File.Data, generated.File.Data) {
				t.Fatalf("download differs from generated bytes: %v", err)
			}
			regenerated, err := rpc.Regenerate(ctx, &clientpb.RegenerateReq{ImplantName: name})
			if err != nil || regenerated.File == nil || !bytes.Equal(regenerated.File.Data, generated.File.Data) {
				t.Fatalf("regenerated bytes differ from final artifact: %v", err)
			}
		})
	}
}

func TestM4GenerateStageFinalParity(t *testing.T) {
	setupGenerateSpoofTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	rpc := &Server{}
	profileName := fmt.Sprintf("m4-stage-profile-%d", time.Now().UnixNano())
	buildName := fmt.Sprintf("m4-stage-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		_ = RemoveBuildByName(buildName)
		_, _ = rpc.DeleteImplantProfile(context.Background(), &clientpb.DeleteReq{Name: profileName})
		_ = os.RemoveAll(filepath.Join(generate.GetSliversDir(), "windows", "amd64", buildName))
	})
	_, err := rpc.SaveImplantProfile(ctx, &clientpb.ImplantProfile{
		Name: profileName,
		Config: &clientpb.ImplantConfig{
			GOOS: "windows", GOARCH: "amd64", Format: clientpb.OutputFormat_EXECUTABLE,
			Debug: true, ObfuscateSymbols: false, HTTPC2ConfigName: "default",
			C2: []*clientpb.ImplantC2{{URL: "mtls://127.0.0.1:8888"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	generated, err := rpc.GenerateStage(ctx, &clientpb.GenerateStageReq{
		Profile: profileName, Name: buildName, PrependSize: true, Compress: "zlib",
		AESEncryptKey: "0123456789abcdef",
	})
	if err != nil {
		t.Fatal(err)
	}
	if generated.GetFile() == nil || len(generated.File.Data) == 0 || generated.GetImplantBuildID() == "" ||
		generated.File.Name != buildName+".bin" {
		t.Fatalf("stage response lacks final bytes, name, or build ID: %+v", generated)
	}
	record, err := rpc.GetBuildRecord(ctx, &clientpb.BuildRecordRef{ID: generated.ImplantBuildID})
	if err != nil {
		t.Fatal(err)
	}
	wantSteps := []string{"builtin/sliver", "builtin/stage-frame", "builtin/stage-compress", "builtin/stage-aes"}
	if record.State != clientpb.BuildRecordState_BUILD_RECORD_SUCCEEDED || len(record.Steps) != len(wantSteps) ||
		record.FinalArtifact == nil || record.FinalArtifact.Type.Kind != "stage" {
		t.Fatalf("stage plan or final artifact is incomplete: %+v", record)
	}
	for index, name := range wantSteps {
		if record.Plan.Steps[index].Spec.Name != name || record.Steps[index].Artifact == nil {
			t.Fatalf("stage step %d missing observed artifact: %+v", index, record)
		}
	}
	hash := sha256.Sum256(generated.File.Data)
	if record.FinalArtifact.SHA256 != hex.EncodeToString(hash[:]) ||
		record.Steps[len(record.Steps)-1].Artifact.SHA256 != record.FinalArtifact.SHA256 {
		t.Fatal("stage final checksum differs from returned bytes")
	}
	stored, err := rpc.GetBuildArtifact(ctx, &clientpb.BuildRecordRef{ID: generated.ImplantBuildID})
	if err != nil || stored.File == nil || !bytes.Equal(stored.File.Data, generated.File.Data) {
		t.Fatalf("stage download differs from returned bytes: %v", err)
	}
	build, err := db.ImplantBuildByName(buildName)
	if err != nil {
		t.Fatal(err)
	}
	legacyBytes, err := generate.ImplantFileFromBuild(build)
	if err != nil || !bytes.Equal(legacyBytes, generated.File.Data) || build.SHA256 != record.FinalArtifact.SHA256 {
		t.Fatalf("legacy stored stage differs from final artifact: %v", err)
	}
	regenerated, err := rpc.Regenerate(ctx, &clientpb.RegenerateReq{ImplantName: buildName})
	if err != nil || regenerated.File == nil || !bytes.Equal(regenerated.File.Data, generated.File.Data) {
		t.Fatalf("regenerated stage differs from final artifact: %v", err)
	}
	if _, err := rpc.DeleteImplantProfile(ctx, &clientpb.DeleteReq{Name: profileName}); err != nil {
		t.Fatalf("delete profile while preserving completed stage: %v", err)
	}
	if _, err := db.ImplantBuildByName(buildName); err != nil {
		t.Fatalf("profile deletion removed completed build: %v", err)
	}
	if _, err := db.ImplantConfigByID(build.ImplantConfigID); err != nil {
		t.Fatalf("profile deletion removed build configuration: %v", err)
	}
	regenerated, err = rpc.Regenerate(ctx, &clientpb.RegenerateReq{ImplantName: buildName})
	if err != nil || regenerated.File == nil || !bytes.Equal(regenerated.File.Data, generated.File.Data) {
		t.Fatalf("profile deletion changed regenerated bytes: %v", err)
	}
	stored, err = rpc.GetBuildArtifact(ctx, &clientpb.BuildRecordRef{ID: generated.ImplantBuildID})
	if err != nil || stored.File == nil || !bytes.Equal(stored.File.Data, generated.File.Data) {
		t.Fatalf("profile deletion changed frozen build artifact: %v", err)
	}
}
