package generate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func previewTestCommand(t *testing.T, beacon bool) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "generate"}
	coreImplantFlags("session", cmd)
	compileImplantFlags("session", cmd)
	if beacon {
		coreBeaconFlags("beacon", cmd)
	}
	return cmd
}

func TestPlanRequestFromFlagsDoesNotAllocateWireGuardIP(t *testing.T) {
	cmd := previewTestCommand(t, false)
	for name, value := range map[string]string{
		"os": "win", "arch": "x64", "format": "shellcode",
		"wg": "example.test", "shellcode-encoder": "xor", "include-http": "true",
	} {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatal(err)
		}
	}
	request, err := planRequestFromFlags(cmd, false)
	if err != nil {
		t.Fatal(err)
	}
	if request.Target.GOOS != "windows" || request.Target.GOARCH != "amd64" || request.Mode != "session" {
		t.Fatalf("wrong normalized target or mode: %+v", request)
	}
	if request.C2[0].Protocol != "wireguard" || !strings.HasPrefix(request.C2[0].Address, "wg://") {
		t.Fatalf("wrong WireGuard endpoint: %+v", request.C2)
	}
	if len(request.Transforms) != 2 || request.Transforms[0].Spec.Name != "builtin/donut" ||
		request.Transforms[1].Spec.Name != "builtin/xor" {
		t.Fatalf("wrong converter or requested encoder: %+v", request.Transforms)
	}
	if len(request.IncludeProtocols) != 2 || request.IncludeProtocols[0] != "http" || request.IncludeProtocols[1] != "https" {
		t.Fatalf("HTTP include should cover HTTP and HTTPS: %+v", request.IncludeProtocols)
	}
}

func TestPlanRequestFromFlagsRejectsUnknownFormat(t *testing.T) {
	cmd := previewTestCommand(t, false)
	_ = cmd.Flags().Set("mtls", "example.test")
	_ = cmd.Flags().Set("format", "unknown")
	_, err := planRequestFromFlags(cmd, false)
	if err == nil || !strings.Contains(err.Error(), "no core capability") {
		t.Fatalf("unknown format should be rejected before planning: %v", err)
	}
}

func TestPlanRequestFromFlagsBeacon(t *testing.T) {
	cmd := previewTestCommand(t, true)
	_ = cmd.Flags().Set("mtls", "example.test")
	_ = cmd.Flags().Set("seconds", "30")
	request, err := planRequestFromFlags(cmd, true)
	if err != nil {
		t.Fatal(err)
	}
	if request.Mode != "beacon" || request.Format != "exe" || request.GetBeaconSettings().GetIntervalSeconds() != 30 ||
		request.GetBeaconSettings().GetJitterSeconds() != 30 {
		t.Fatalf("wrong beacon request: %+v", request)
	}
	_ = cmd.Flags().Set("seconds", "1")
	if _, err := planRequestFromFlags(cmd, true); err == nil || !strings.Contains(err.Error(), "beacon interval") {
		t.Fatalf("short beacon interval should be rejected: %v", err)
	}
	_ = cmd.Flags().Set("seconds", "90")
	_ = cmd.Flags().Set("jitter", "15")
	request, err = planRequestFromFlags(cmd, true)
	if err != nil || request.GetBeaconSettings().GetIntervalSeconds() != 90 || request.GetBeaconSettings().GetJitterSeconds() != 15 {
		t.Fatalf("custom beacon settings were not preserved: request=%+v err=%v", request, err)
	}
}

func TestPlanRequestFromFlagsPinsRemoteBuilder(t *testing.T) {
	cmd := previewTestCommand(t, false)
	_ = cmd.Flags().Set("mtls", "example.test")
	_ = cmd.Flags().Set("builder", "builder-b")
	request, err := planRequestFromFlags(cmd, false)
	if err != nil {
		t.Fatal(err)
	}
	if request.GetProvider().GetMode() != "pinned" || request.GetProvider().GetBuilder() != "builder-b" {
		t.Fatalf("builder pin not retained: %+v", request.GetProvider())
	}
	_ = cmd.Flags().Set("builder", " ")
	if _, err := planRequestFromFlags(cmd, false); err == nil || !strings.Contains(err.Error(), "builder pin cannot be empty") {
		t.Fatalf("blank builder pin was accepted: %v", err)
	}
}

func TestAssignedStepCompatibilityRejectsUnrepresentedSettings(t *testing.T) {
	cmd := previewTestCommand(t, false)
	config := &clientpb.ImplantConfig{HTTPC2ConfigName: "default"}
	if err := remoteStepCompatibility(cmd, config); err != nil {
		t.Fatalf("simple direct build should use assigned steps: %v", err)
	}
	_ = cmd.Flags().Set("c2profile", "custom")
	if err := remoteStepCompatibility(cmd, config); err == nil {
		t.Fatal("custom C2 profile would be lost in assigned-step conversion")
	}
	cmd = previewTestCommand(t, false)
	config.IsBeacon = true
	config.BeaconInterval = int64(30 * time.Second)
	config.BeaconJitter = int64(15 * time.Second)
	if err := remoteStepCompatibility(cmd, config); err != nil {
		t.Fatalf("custom beacon timing is represented in assigned steps: %v", err)
	}
	config.IsBeacon = false
	config.Format = clientpb.OutputFormat_SHELLCODE
	if err := remoteStepCompatibility(cmd, config); err != nil {
		t.Fatalf("shellcode should use assigned steps: %v", err)
	}
	cmd = previewTestCommand(t, false)
	if err := cmd.Flags().Set("traffic-encoders", "custom"); err != nil {
		t.Fatal(err)
	}
	config.TrafficEncodersEnabled = false // The legacy parser ignores this without HTTP C2.
	if err := remoteStepCompatibility(cmd, config); err == nil || !strings.Contains(err.Error(), "traffic encoder") {
		t.Fatalf("explicit traffic encoder flag would be sent to the remote plan: %v", err)
	}
}

func TestPlanRequestFromFlagsDebugDisablesSymbolObfuscation(t *testing.T) {
	cmd := previewTestCommand(t, false)
	_ = cmd.Flags().Set("mtls", "example.test")
	_ = cmd.Flags().Set("debug", "true")
	_ = cmd.Flags().Set("skip-symbols", "false")
	request, err := planRequestFromFlags(cmd, false)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, option := range request.CoreOptions {
		if option.Name == "obfuscateSymbols" {
			count++
			if string(option.ValueJSON) != "false" {
				t.Fatalf("debug build should disable symbol obfuscation: %s", option.ValueJSON)
			}
		}
	}
	if count != 1 {
		t.Fatalf("expected one resolved obfuscation option, got %d", count)
	}
}

func TestPreviewMetadataTransformBindsFullConfigWithoutBytes(t *testing.T) {
	metadata := &clientpb.SpoofMetadataConfig{PE: &clientpb.PESpoofMetadataConfig{
		Source:            &clientpb.SpoofMetadataFile{Name: "donor.exe", Data: []byte("donor bytes")},
		ResourceDirectory: &clientpb.IMAGE_RESOURCE_DIRECTORY{TimeDateStamp: 1234},
	}}
	step, err := previewMetadataTransform(metadata)
	if err != nil {
		t.Fatal(err)
	}
	if step.GetSpec().GetName() != "builtin/pe-metadata" || len(step.GetOptions()) != 1 {
		t.Fatalf("wrong metadata transform: %+v", step)
	}
	var ref struct{ Kind, Name string }
	if err := json.Unmarshal(step.Options[0].ValueJSON, &ref); err != nil {
		t.Fatal(err)
	}
	serialized, err := proto.MarshalOptions{Deterministic: true}.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(serialized)
	if ref.Kind != "input-artifact" || ref.Name != "metadata-sha256:"+hex.EncodeToString(hash[:]) {
		t.Fatalf("metadata preview does not match generation reference: %+v", ref)
	}
	stepJSON, err := json.Marshal(step)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stepJSON), "donor bytes") || strings.Contains(string(stepJSON), "donor.exe") {
		t.Fatal("donor material leaked into preview plan")
	}
}

func TestPlanRequestFromFlagsIncludesMetadataStep(t *testing.T) {
	goCmd, err := exec.LookPath("go")
	if err != nil {
		t.Skip("Go compiler unavailable")
	}
	root := t.TempDir()
	source := filepath.Join(root, "main.go")
	if err := os.WriteFile(source, []byte("package main\nfunc main() {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	donor := filepath.Join(root, "donor.exe")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, goCmd, "build", "-trimpath", "-o", donor, source)
	command.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0", "GOFLAGS=", "GO111MODULE=off", "GOTOOLCHAIN=local")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile donor PE: %v\n%s", err, output)
	}
	cmd := previewTestCommand(t, false)
	for flag, value := range map[string]string{
		"os": "windows", "arch": "amd64", "format": "exe", "mtls": "example.test",
		"spoof-metadata": donor,
	} {
		if err := cmd.Flags().Set(flag, value); err != nil {
			t.Fatal(err)
		}
	}
	request, err := planRequestFromFlags(cmd, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(request.Transforms) != 1 || request.Transforms[0].Spec.Name != "builtin/pe-metadata" ||
		!strings.Contains(string(request.Transforms[0].Options[0].ValueJSON), "metadata-sha256:") {
		t.Fatalf("metadata preview did not resolve the installed step: %+v", request.Transforms)
	}
}
