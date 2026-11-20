package minimal

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bishopfox/sliver/server/buildcatalog"
	"github.com/bishopfox/sliver/server/buildcore/magebuild"
	"github.com/bishopfox/sliver/server/buildcore/workspace"
	"github.com/bishopfox/sliver/server/buildplan"
	"github.com/bishopfox/sliver/server/buildprofile"
)

func TestInstalledSpecAndMageBinding(t *testing.T) {
	catalog, err := buildcatalog.Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := catalog.Get(buildcatalog.Reference{Name: SpecName})
	if err != nil {
		t.Fatal(err)
	}
	if !spec.Available || spec.Core == nil || spec.Core.Executor.ID != ExecutorID ||
		spec.Core.Source == nil || spec.Core.Builder == nil ||
		spec.Core.Source.Mode != (Adapter{}).Binding().SourceMode ||
		spec.Core.Builder.Magefile != (Adapter{}).Binding().Magefile ||
		spec.Core.Builder.Target != (Adapter{}).Binding().Target ||
		len(spec.Core.Runtime.Commands) != 1 || spec.Core.Runtime.Commands[0] != "MsgPing" {
		t.Fatalf("installed minimal core binding differs from pinned spec: %+v", spec.Core)
	}
	example, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "build-pipeline", "examples", "core-minimal-sliver-v1.yaml"))
	if err != nil {
		t.Fatalf("read matching example: %v", err)
	}
	if !bytes.Equal(example, spec.RawYAML) {
		t.Fatal("documentation example differs from the installed core spec")
	}
}

func TestInstalledMageDigestAndLiteralSource(t *testing.T) {
	work, err := workspace.New(t.TempDir(), "linux", "amd64", "minimal-test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := work.Remove(); err != nil {
			t.Errorf("remove workspace: %v", err)
		}
	})
	adapter := mageAdapter(nil)
	if err := work.CopySource(adapter.Source); err != nil {
		t.Fatal(err)
	}
	if err := work.CopyMagefiles(adapter.Magefiles); err != nil {
		t.Fatal(err)
	}
	digest, err := magebuild.DigestBundle(filepath.Join(work.MagefilesDir, "magefiles", "minimal"))
	if err != nil {
		t.Fatal(err)
	}
	if digest != adapter.BundleDigest {
		t.Fatalf("staged Magefile digest %q differs from installed %q", digest, adapter.BundleDigest)
	}
	staged, err := os.ReadFile(filepath.Join(work.SourceDir, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	original, err := files.ReadFile("source/main.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(staged, original) || !bytes.Contains(staged, []byte("{{.Build.Name}}")) {
		t.Fatal("source was interpreted or changed while copying")
	}
}

func TestMaterialMustMatchFrozenPlan(t *testing.T) {
	plan := fixturePlan()
	m := material{
		Name: plan.Request.Name, Endpoint: plan.Request.C2[0].Address,
		ReconnectSeconds: reconnectSeconds(plan),
		TargetOS:         plan.Request.Target.OS, TargetArch: plan.Request.Target.Arch,
		PeerPrivateKey: "private", MtlsCert: "cert", MtlsKey: "key", MtlsCACert: "ca",
		MinisignServerPublicKey: "server-key",
	}
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseMaterial(plan, raw); err != nil {
		t.Fatalf("valid material rejected: %v", err)
	}
	changed := plan
	changed.Request.C2 = []buildplan.Endpoint{{Protocol: "mtls", Address: "mtls://127.0.0.1:4444"}}
	if _, err := parseMaterial(changed, raw); err == nil {
		t.Fatal("material for another endpoint accepted")
	}
	if _, err := (Adapter{}).BuildRemote(context.Background(), changed, raw); err == nil {
		t.Fatal("remote worker accepted material for another plan")
	}
	changed = plan
	changed.Steps = append([]buildplan.Step(nil), plan.Steps...)
	changed.Steps[0].Output.Kind = "shellcode"
	if _, err := parseMaterial(changed, raw); err == nil {
		t.Fatal("unsupported final core type accepted")
	}
	changed = plan
	changed.Request.C2Settings = &buildplan.C2Settings{
		ConnectionStrategy: "sequential", ReconnectSeconds: 5, HTTPC2Profile: "default",
	}
	if _, err := parseMaterial(changed, raw); err == nil {
		t.Fatal("material for another reconnect interval accepted")
	}
	m.ReconnectSeconds = 5
	authored, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseMaterial(changed, authored); err != nil {
		t.Fatalf("material for an authored reconnect interval rejected: %v", err)
	}
}

func TestMinimalC2SettingsContract(t *testing.T) {
	plan := fixturePlan()
	if err := validatePlan(plan); err != nil {
		t.Fatalf("legacy request rejected: %v", err)
	}
	if got := reconnectSeconds(plan); got != 2 {
		t.Fatalf("legacy reconnect = %d, want 2", got)
	}
	settings := buildplan.C2Settings{ConnectionStrategy: "sequential", ReconnectSeconds: 5, HTTPC2Profile: "default"}
	plan.Request.C2Settings = &settings
	if err := validatePlan(plan); err != nil {
		t.Fatalf("authored minimal C2 settings rejected: %v", err)
	}
	if got := reconnectSeconds(plan); got != 5 {
		t.Fatalf("authored reconnect = %d, want 5", got)
	}
	for _, tc := range []struct {
		name   string
		change func(*buildplan.C2Settings)
	}{
		{"random strategy", func(s *buildplan.C2Settings) { s.ConnectionStrategy = "random" }},
		{"zero reconnect", func(s *buildplan.C2Settings) { s.ReconnectSeconds = 0 }},
		{"overflow reconnect", func(s *buildplan.C2Settings) { s.ReconnectSeconds = maxReconnectSeconds + 1 }},
		{"poll timeout", func(s *buildplan.C2Settings) { s.PollTimeoutSeconds = 1 }},
		{"max errors", func(s *buildplan.C2Settings) { s.MaxConnectionErrors = 1 }},
		{"HTTP C2 profile", func(s *buildplan.C2Settings) { s.HTTPC2Profile = "custom" }},
		{"WireGuard key exchange", func(s *buildplan.C2Settings) { s.WireguardKeyExchangePort = 53 }},
		{"WireGuard TCP", func(s *buildplan.C2Settings) { s.WireguardTCPCommsPort = 53 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			unsupported := settings
			tc.change(&unsupported)
			plan.Request.C2Settings = &unsupported
			if err := validatePlan(plan); err == nil {
				t.Fatal("unsupported C2 settings accepted")
			}
		})
	}
}

func TestAuthoredMinimalProfilePlans(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "build-pipeline", "examples", "profile-minimal-sliver-v1.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	profile, err := buildprofile.Parse(raw)
	if err != nil {
		t.Fatalf("parse minimal profile: %v", err)
	}
	catalog, err := buildcatalog.Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	planner := buildplan.NewPlanner(catalog, func(provider string, spec buildcatalog.Snapshot, _ buildplan.Target, toolchain buildplan.Toolchain) bool {
		return provider == "server" && spec.Reference.Name == SpecName && toolchain.Driver == "go"
	})
	plan, err := planner.Plan(context.Background(), profile.Request("minimal-profile-test"))
	if err != nil {
		t.Fatalf("plan minimal profile: %v", err)
	}
	if err := validatePlan(plan); err != nil {
		t.Fatalf("minimal adapter rejected authored profile: %v", err)
	}
	if got := reconnectSeconds(plan); got != 5 {
		t.Fatalf("profile reconnect = %d, want 5", got)
	}
}

func fixturePlan() buildplan.Plan {
	target := buildplan.Target{OS: "linux", Arch: "amd64"}
	output := buildplan.Artifact{Kind: "executable", Container: "elf", Target: target, GoBuildMode: "default"}
	return buildplan.Plan{
		Request: buildplan.Request{
			Name: "minimal-test", Target: target, Mode: "session",
			Core: buildplan.Reference{Name: SpecName},
			C2:   []buildplan.Endpoint{{Protocol: "mtls", Address: "mtls://127.0.0.1:4433"}},
		},
		Steps: []buildplan.Step{{Kind: "core", Spec: buildplan.Reference{Name: SpecName},
			Provider: "server", Output: output,
			Toolchain: buildplan.Toolchain{Driver: "go", Requirements: []string{"go"}}}},
		FinalArtifact: output,
	}
}
