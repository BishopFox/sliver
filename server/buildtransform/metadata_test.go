package buildtransform

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/pe"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/buildplan"
	"google.golang.org/protobuf/proto"
)

func TestPEMetadataCopiesDonorTimestampAndCleansScratch(t *testing.T) {
	root := t.TempDir()
	outputPath, input := compilePEMetadataFixture(t, root)
	donor := append([]byte(nil), input...)
	peOffset := int(binary.LittleEndian.Uint32(donor[0x3c:0x40]))
	if peOffset+12 > len(donor) {
		t.Fatal("compiled PE has truncated file header")
	}
	wantTimestamp := uint32(0x542ec8ab)
	binary.LittleEndian.PutUint32(donor[peOffset+8:peOffset+12], wantTimestamp)
	target := buildplan.Target{OS: "windows", Arch: "amd64"}
	artifact := buildplan.Artifact{Kind: "executable", Container: "pe", Target: target, GoBuildMode: "default"}
	step := testStageStep(PEMetadataSpec, artifact, artifact,
		testStageOption("donorArtifact", `{"kind":"input-artifact","name":"donor-pe"}`))
	scratch := filepath.Join(root, "scratch")
	if err := os.Mkdir(scratch, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", scratch)
	got, err := ApplyPEMetadata(context.Background(), step, input, donor)
	if err != nil {
		t.Fatal(err)
	}
	file, err := pe.NewFile(bytes.NewReader(got))
	if err != nil {
		t.Fatalf("spoofed output is not PE: %v", err)
	}
	if file.TimeDateStamp != wantTimestamp || file.Machine != pe.IMAGE_FILE_MACHINE_AMD64 {
		t.Fatalf("spoofed output metadata differs: timestamp=%#x machine=%#x", file.TimeDateStamp, file.Machine)
	}
	_ = file.Close()
	entries, err := os.ReadDir(scratch)
	if err != nil || len(entries) != 0 {
		t.Fatalf("private metadata scratch remains: entries=%v err=%v", entries, err)
	}
	if bytes.Equal(input, got) {
		t.Fatal("metadata transform did not change PE bytes")
	}
	if original, err := os.ReadFile(outputPath); err != nil || !bytes.Equal(original, input) {
		t.Fatalf("metadata transform changed original input: %v", err)
	}

	assertConfiguredPEMetadata(t, step, input, donor, wantTimestamp)
}

func compilePEMetadataFixture(t *testing.T, root string) (string, []byte) {
	t.Helper()
	goCmd, err := exec.LookPath("go")
	if err != nil {
		t.Skip("Go compiler unavailable")
	}
	source := filepath.Join(root, "main.go")
	if err := os.WriteFile(source, []byte("package main\nfunc main() {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(root, "fixture.exe")
	compileCtx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	command := exec.CommandContext(compileCtx, goCmd, "build", "-trimpath", "-o", outputPath, source)
	command.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0", "GOFLAGS=", "GO111MODULE=off", "GOTOOLCHAIN=local")
	if result, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile PE fixture: %v\n%s", err, result)
	}
	input, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	return outputPath, input
}

func assertConfiguredPEMetadata(t *testing.T, step buildplan.Step, input, donor []byte, wantTimestamp uint32) {
	t.Helper()
	fullConfig := &clientpb.SpoofMetadataConfig{PE: &clientpb.PESpoofMetadataConfig{
		Source: &clientpb.SpoofMetadataFile{Name: "donor.exe", Data: donor},
	}}
	configJSON, err := proto.MarshalOptions{Deterministic: true}.Marshal(fullConfig)
	if err != nil {
		t.Fatal(err)
	}
	configHash := sha256.Sum256(configJSON)
	refName := metadataDigestPrefix + hex.EncodeToString(configHash[:])
	step.Options = []buildplan.Option{testStageOption("donorArtifact", `{"kind":"input-artifact","name":"`+refName+`"}`)}
	configured, err := ApplyPEMetadataConfig(context.Background(), step, input, donor, configJSON)
	if err != nil {
		t.Fatalf("full metadata configuration failed: %v", err)
	}
	configuredPE, err := pe.NewFile(bytes.NewReader(configured))
	if err != nil {
		t.Fatal(err)
	}
	if configuredPE.TimeDateStamp != wantTimestamp {
		t.Fatalf("configured metadata did not copy donor timestamp: %#x", configuredPE.TimeDateStamp)
	}
	_ = configuredPE.Close()
	wrongConfig := append([]byte(nil), configJSON...)
	wrongConfig = append(wrongConfig, ' ')
	if _, err := ApplyPEMetadataConfig(context.Background(), step, input, donor, wrongConfig); err == nil ||
		!strings.Contains(err.Error(), "frozen reference") {
		t.Fatalf("accepted unfrozen metadata config: %v", err)
	}
	// This fixture lacks .rsrc. An override must reach the legacy mutator and
	// report its precise failure instead of being silently dropped.
	fullConfig.PE.ResourceDirectory = &clientpb.IMAGE_RESOURCE_DIRECTORY{TimeDateStamp: wantTimestamp}
	resourceConfig, err := proto.MarshalOptions{Deterministic: true}.Marshal(fullConfig)
	if err != nil {
		t.Fatal(err)
	}
	resourceHash := sha256.Sum256(resourceConfig)
	step.Options = []buildplan.Option{testStageOption("donorArtifact", `{"kind":"input-artifact","name":"`+
		metadataDigestPrefix+hex.EncodeToString(resourceHash[:])+`"}`)}
	if _, err := ApplyPEMetadataConfig(context.Background(), step, input, donor, resourceConfig); err == nil ||
		!strings.Contains(err.Error(), "target PE has no .rsrc section") {
		t.Fatalf("resource override was not applied: %v", err)
	}
}

func TestPEMetadataRejectsWrongEdgeAndReference(t *testing.T) {
	target := buildplan.Target{OS: "windows", Arch: "amd64"}
	artifact := buildplan.Artifact{Kind: "executable", Container: "pe", Target: target, GoBuildMode: "default"}
	step := testStageStep(PEMetadataSpec, artifact, artifact,
		testStageOption("donorArtifact", `{"kind":"secret","name":"donor-pe"}`))
	if _, err := ApplyPEMetadata(context.Background(), step, []byte("target"), []byte("donor")); err == nil || !strings.Contains(err.Error(), "donor") {
		t.Fatalf("accepted secret as donor artifact: %v", err)
	}
	step.Options = []buildplan.Option{testStageOption("donorArtifact", `{"kind":"input-artifact","name":"donor-pe"}`)}
	step.Output.Target.Arch = "386"
	if _, err := ApplyPEMetadata(context.Background(), step, []byte("target"), []byte("donor")); err == nil {
		t.Fatal("accepted target-changing metadata transform")
	}
	step.Output.Target.Arch = "amd64"
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ApplyPEMetadata(ctx, step, []byte("target"), []byte("donor")); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled metadata transform proceeded: %v", err)
	}
}
