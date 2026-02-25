package generate

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/spf13/cobra"
)

func TestParseSpoofMetadataFlagDefaultOff(t *testing.T) {
	cmd := newSpoofMetadataTestCommand(t)
	config := &clientpb.ImplantConfig{
		GOOS:   "windows",
		GOARCH: "amd64",
		Format: clientpb.OutputFormat_EXECUTABLE,
	}

	spoofMetadata, err := parseSpoofMetadataFlag(cmd, config)
	if err != nil {
		t.Fatalf("parseSpoofMetadataFlag() unexpected error: %v", err)
	}
	if spoofMetadata != nil {
		t.Fatal("expected spoof metadata to be nil when flag is not set")
	}
}

func TestParseSpoofMetadataFlagNoArgUsesConfig(t *testing.T) {
	t.Setenv("SLIVER_CLIENT_ROOT_DIR", t.TempDir())
	cmd := newSpoofMetadataTestCommand(t)
	if err := cmd.ParseFlags([]string{"--" + spoofMetadataFlagName}); err != nil {
		t.Fatalf("ParseFlags() error: %v", err)
	}
	rootDir := os.Getenv("SLIVER_CLIENT_ROOT_DIR")
	configPath := filepath.Join(rootDir, "spoof-metadata.yaml")
	sourceData := []byte{0x01, 0x02, 0x03}
	configData := "pe:\n  source:\n    name: donor.exe\n    base64: " + base64.StdEncoding.EncodeToString(sourceData) + "\n"
	if err := os.WriteFile(configPath, []byte(configData), 0o600); err != nil {
		t.Fatalf("write spoof metadata config: %v", err)
	}

	config := &clientpb.ImplantConfig{
		GOOS:   "windows",
		GOARCH: "amd64",
		Format: clientpb.OutputFormat_EXECUTABLE,
	}
	spoofMetadata, err := parseSpoofMetadataFlag(cmd, config)
	if err != nil {
		t.Fatalf("parseSpoofMetadataFlag() unexpected error: %v", err)
	}
	if spoofMetadata == nil || spoofMetadata.GetPE() == nil || spoofMetadata.GetPE().GetSource() == nil {
		t.Fatal("expected spoof metadata source from config file")
	}
	if got := spoofMetadata.GetPE().GetSource().GetData(); string(got) != string(sourceData) {
		t.Fatalf("source data mismatch: got=%x want=%x", got, sourceData)
	}
}

func TestParseSpoofMetadataFlagPathFromPositionalArg(t *testing.T) {
	cmd := newSpoofMetadataTestCommand(t)
	sourcePath := goPEFixturePath(t, "gcc-amd64-mingw-exec")
	if err := cmd.ParseFlags([]string{"--" + spoofMetadataFlagName, sourcePath}); err != nil {
		t.Fatalf("ParseFlags() error: %v", err)
	}

	config := &clientpb.ImplantConfig{
		GOOS:   "windows",
		GOARCH: "amd64",
		Format: clientpb.OutputFormat_EXECUTABLE,
	}
	spoofMetadata, err := parseSpoofMetadataFlag(cmd, config)
	if err != nil {
		t.Fatalf("parseSpoofMetadataFlag() unexpected error: %v", err)
	}
	if spoofMetadata == nil || spoofMetadata.GetPE() == nil || spoofMetadata.GetPE().GetSource() == nil {
		t.Fatal("expected spoof metadata source from positional path")
	}
	if spoofMetadata.GetPE().GetSource().GetName() != filepath.Base(sourcePath) {
		t.Fatalf("source name mismatch: got=%q want=%q", spoofMetadata.GetPE().GetSource().GetName(), filepath.Base(sourcePath))
	}
}

func TestParseSpoofMetadataFlagPathRejectsNonWindowsTarget(t *testing.T) {
	cmd := newSpoofMetadataTestCommand(t)
	sourcePath := writeTempFile(t, "not-pe.bin", []byte("abc"))
	if err := cmd.Flags().Set(spoofMetadataFlagName, sourcePath); err != nil {
		t.Fatalf("set flag: %v", err)
	}

	config := &clientpb.ImplantConfig{
		GOOS:   "linux",
		GOARCH: "amd64",
		Format: clientpb.OutputFormat_EXECUTABLE,
	}
	_, err := parseSpoofMetadataFlag(cmd, config)
	if err == nil {
		t.Fatal("expected error for non-windows target")
	}
	if !strings.Contains(err.Error(), "supports only windows PE targets") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseSpoofMetadataFlagPathChecksMachine(t *testing.T) {
	cmd := newSpoofMetadataTestCommand(t)
	sourcePath := goPEFixturePath(t, "gcc-386-mingw-exec")
	if err := cmd.Flags().Set(spoofMetadataFlagName, sourcePath); err != nil {
		t.Fatalf("set flag: %v", err)
	}

	config := &clientpb.ImplantConfig{
		GOOS:   "windows",
		GOARCH: "amd64",
		Format: clientpb.OutputFormat_EXECUTABLE,
	}
	_, err := parseSpoofMetadataFlag(cmd, config)
	if err == nil {
		t.Fatal("expected machine mismatch error")
	}
	if !strings.Contains(err.Error(), "expected amd64") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseSpoofMetadataFlagPathSuccess(t *testing.T) {
	cmd := newSpoofMetadataTestCommand(t)
	sourcePath := goPEFixturePath(t, "gcc-amd64-mingw-exec")
	if err := cmd.Flags().Set(spoofMetadataFlagName, sourcePath); err != nil {
		t.Fatalf("set flag: %v", err)
	}

	config := &clientpb.ImplantConfig{
		GOOS:   "windows",
		GOARCH: "amd64",
		Format: clientpb.OutputFormat_EXECUTABLE,
	}
	spoofMetadata, err := parseSpoofMetadataFlag(cmd, config)
	if err != nil {
		t.Fatalf("parseSpoofMetadataFlag() unexpected error: %v", err)
	}
	if spoofMetadata == nil || spoofMetadata.GetPE() == nil || spoofMetadata.GetPE().GetSource() == nil {
		t.Fatal("expected spoof metadata source from path")
	}
	if spoofMetadata.GetPE().GetSource().GetName() != filepath.Base(sourcePath) {
		t.Fatalf("source name mismatch: got=%q want=%q", spoofMetadata.GetPE().GetSource().GetName(), filepath.Base(sourcePath))
	}
	if len(spoofMetadata.GetPE().GetSource().GetData()) == 0 {
		t.Fatal("expected non-empty source data")
	}
}

func TestSpoofMetadataFlagUsageIsClean(t *testing.T) {
	cmd := newSpoofMetadataTestCommand(t)
	usage := cmd.Flags().FlagUsages()
	if !strings.Contains(usage, "--"+spoofMetadataFlagName) {
		t.Fatalf("expected usage to include spoof metadata flag, got: %s", usage)
	}
	if strings.Contains(usage, "string[=") {
		t.Fatalf("unexpected optional-string marker in usage: %s", usage)
	}
	if strings.Contains(usage, `[="`) {
		t.Fatalf("unexpected no-opt default rendering in usage: %s", usage)
	}
}

func newSpoofMetadataTestCommand(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "test"}
	bindSpoofMetadataFlag(cmd.Flags())
	return cmd
}

func writeTempFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func goPEFixturePath(t *testing.T, fixtureName string) string {
	t.Helper()
	fixturePath := filepath.Join(runtime.GOROOT(), "src", "debug", "pe", "testdata", fixtureName)
	if _, err := os.Stat(fixturePath); err != nil {
		t.Skipf("missing PE fixture %s: %v", fixturePath, err)
	}
	return fixturePath
}
