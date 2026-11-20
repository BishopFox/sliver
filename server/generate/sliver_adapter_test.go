package generate

import (
	"context"
	"encoding/json"
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/buildcore"
	"github.com/bishopfox/sliver/server/buildplan"
)

func TestSliverCoreAdapterRegistered(t *testing.T) {
	adapter, ok := buildcore.DefaultAdapter(sliverExecutorID)
	if !ok || adapter.ExecutorID() != sliverExecutorID {
		t.Fatal("Sliver core adapter is not registered by executor ID")
	}
	if _, ok := adapter.(buildcore.Availability); !ok {
		t.Fatal("Sliver core adapter does not advertise target availability")
	}
}

func TestCompilerCommandAvailable(t *testing.T) {
	if !compilerCommandAvailable("go version") {
		t.Fatal("Go on the test runner PATH should be an installed compiler command")
	}
	if compilerCommandAvailable("/nonexistent/sliver-m3-compiler cc -target darwin") {
		t.Fatal("a missing compiler must not advertise a cross-build target")
	}
	if compilerCommandAvailable(" \t ") {
		t.Fatal("an empty compiler command must not be available")
	}
}

func TestSliverCoreAdapterSupports(t *testing.T) {
	adapter := SliverCoreAdapter{}
	if !adapter.Supports(buildplan.Target{OS: "windows", Arch: "amd64"}, buildplan.Toolchain{Driver: "go"}) {
		t.Fatal("Windows/amd64 executable should be available on every server")
	}
	if adapter.Supports(buildplan.Target{OS: "windows", Arch: "mips"}, buildplan.Toolchain{Driver: "go"}) {
		t.Fatal("unsupported architecture should be unavailable")
	}
	if adapter.Supports(buildplan.Target{OS: "windows", Arch: "amd64"}, buildplan.Toolchain{Driver: "other"}) {
		t.Fatal("Sliver core should reject a non-Go compiler")
	}
	if _, supported := SupportedCompilerTargets[runtime.GOOS+"/"+runtime.GOARCH]; supported {
		for _, requirement := range []string{"cgo", "native-or-zig-cc"} {
			if !adapter.Supports(buildplan.Target{OS: runtime.GOOS, Arch: runtime.GOARCH},
				buildplan.Toolchain{Driver: "go", Requirements: []string{requirement}}) {
				t.Fatalf("native %s compiler target should be available", requirement)
			}
		}
	}
}

func TestSliverCoreAdapterConfigFromPlan(t *testing.T) {
	plan := buildplan.Plan{
		Request: buildplan.Request{
			Target: buildplan.Target{OS: "windows", Arch: "amd64"}, Mode: "session", FinalFormat: "service",
			C2:               []buildplan.Endpoint{{Protocol: "wireguard", Address: "wireguard://192.0.2.10:53"}},
			IncludeProtocols: []string{"https"},
		},
		Steps: []buildplan.Step{{
			ID: "core", Kind: "core", Spec: buildplan.Reference{Name: "builtin/sliver"}, Provider: "server",
			Options: []buildplan.Option{
				{Name: "exports", ValueJSON: json.RawMessage(`["StartW","DllRegisterServer"]`)},
				{Name: "serviceName", ValueJSON: json.RawMessage(`"TestingService"`)},
				{Name: "trafficEncoders", ValueJSON: json.RawMessage(`["traffic.wasm"]`)},
			},
		}},
	}
	config, err := (SliverCoreAdapter{}).ConfigFromPlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	if config.Format != clientpb.OutputFormat_SERVICE || !config.IsService || !config.IncludeWG || !config.IncludeHTTP ||
		len(config.C2) != 1 || config.C2[0].URL != "wg://192.0.2.10:53" || config.ServiceName != "TestingService" {
		t.Fatalf("Sliver service config lost route settings: %+v", config)
	}
	if len(config.Exports) != 2 || !config.TrafficEncodersEnabled || len(config.Assets) != 1 || config.Assets[0].Name != "traffic.wasm" {
		t.Fatalf("Sliver adapter lost exports or traffic encoder: %+v", config)
	}
}

func TestSliverCoreAdapterDirectOutputFormats(t *testing.T) {
	for _, test := range []struct {
		format    string
		want      clientpb.OutputFormat
		service   bool
		sharedLib bool
	}{
		{format: "exe", want: clientpb.OutputFormat_EXECUTABLE},
		{format: "service", want: clientpb.OutputFormat_SERVICE, service: true},
		{format: "shared", want: clientpb.OutputFormat_SHARED_LIB, sharedLib: true},
		{format: "archive", want: clientpb.OutputFormat_GO_ARCHIVE, sharedLib: true},
	} {
		t.Run(test.format, func(t *testing.T) {
			plan := buildplan.Plan{
				Request: buildplan.Request{Target: buildplan.Target{OS: "windows", Arch: "amd64"}, FinalFormat: test.format},
				Steps:   []buildplan.Step{{ID: "core", Kind: "core", Spec: buildplan.Reference{Name: "builtin/sliver"}, Provider: "server"}},
			}
			config, err := (SliverCoreAdapter{}).ConfigFromPlan(plan)
			if err != nil {
				t.Fatal(err)
			}
			if config.Format != test.want || config.IsService != test.service || config.IsSharedLib != test.sharedLib || config.IsShellcode {
				t.Fatalf("incorrect %s output settings: %+v", test.format, config)
			}
		})
	}
}

func TestSliverCoreAdapterAcceptsOrderedConverterAndEncoder(t *testing.T) {
	plan := buildplan.Plan{
		Request: buildplan.Request{Target: buildplan.Target{OS: "linux", Arch: "amd64"}, FinalFormat: "shellcode"},
		Steps: []buildplan.Step{
			{ID: "core", Kind: "core", Spec: buildplan.Reference{Name: "builtin/sliver"}, Provider: "server"},
			{ID: "transform-1", Kind: "transform", Spec: buildplan.Reference{Name: "builtin/malasada"}, Provider: "server"},
			{ID: "transform-2", Kind: "transform", Spec: buildplan.Reference{Name: "builtin/xor"}, Provider: "server"},
		},
	}
	config, err := (SliverCoreAdapter{}).ConfigFromPlan(plan)
	if err != nil || config.Format != clientpb.OutputFormat_SHELLCODE || !config.IsShellcode {
		t.Fatalf("ordered converter and encoder were rejected: config=%+v err=%v", config, err)
	}
	plan.Steps[2].Provider = "remote-builder"
	config, err = (SliverCoreAdapter{}).ConfigFromPlan(plan)
	if err != nil || config.Format != clientpb.OutputFormat_SHELLCODE {
		t.Fatalf("remote transform should preserve the core format: config=%+v err=%v", config, err)
	}
}

func TestSliverCoreAdapterCancelBeforeCredentials(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := (SliverCoreAdapter{}).BuildLegacy(ctx, "example", &clientpb.ImplantConfig{Format: clientpb.OutputFormat_EXECUTABLE})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled build should stop before credential generation: %v", err)
	}
	_, err = (SliverCoreAdapter{}).BuildLegacy(context.Background(), "example", nil)
	if err == nil || !strings.Contains(err.Error(), "missing implant config") {
		t.Fatalf("nil config should be rejected: %v", err)
	}
	_, err = (SliverCoreAdapter{}).BuildLegacy(nil, "example", &clientpb.ImplantConfig{}) //nolint:staticcheck // Verify nil context rejection (SA1012).
	if err == nil || !strings.Contains(err.Error(), "missing build context") {
		t.Fatalf("nil context should be rejected: %v", err)
	}
}

func TestSliverCoreAdapterNormalize(t *testing.T) {
	adapter := SliverCoreAdapter{}
	if err := adapter.Normalize(nil); err == nil {
		t.Fatal("nil config should be rejected")
	}
	config := &clientpb.ImplantConfig{
		Format: clientpb.OutputFormat_GO_ARCHIVE,
		C2:     []*clientpb.ImplantC2{{URL: "https://example.test"}, {URL: "wg://192.0.2.10:53"}},
	}
	if err := adapter.Normalize(config); err != nil {
		t.Fatal(err)
	}
	if !config.IncludeHTTP || !config.IncludeWG || !config.IsSharedLib || len(config.Exports) != 1 || config.Exports[0] != "StartW" {
		t.Fatalf("normalization lost Sliver defaults: %+v", config)
	}
	if err := adapter.Normalize(config); err != nil || len(config.Exports) != 1 {
		t.Fatalf("normalization should be idempotent: %+v, %v", config, err)
	}
}
