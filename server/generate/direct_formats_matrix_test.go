//go:build sliver_generate

package generate

import (
	"bytes"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/configs"
)

// TestSliverDirectFormatsAdvertisedTargets checks the service and shared
// library routes omitted from the older direct-build tests. Each advertised
// tuple must compile in both session and beacon mode on this builder.
func TestSliverDirectFormatsAdvertisedTargets(t *testing.T) {
	certs.SetupCAs()
	for _, format := range []clientpb.OutputFormat{
		clientpb.OutputFormat_SERVICE,
		clientpb.OutputFormat_SHARED_LIB,
	} {
		targets := advertisedTargetsForFormat(format)
		if len(targets) == 0 {
			t.Logf("no %s compiler targets advertised by this builder", format)
			continue
		}
		for _, target := range targets {
			for _, beacon := range []bool{false, true} {
				mode := "session"
				if beacon {
					mode = "beacon"
				}
				t.Run(fmt.Sprintf("%s/%s/%s/%s", format, target.GOOS, target.GOARCH, mode), func(t *testing.T) {
					buildDirectFormatForTarget(t, format, target, beacon)
				})
			}
		}
	}
}

// TestSliverExecutablesAdvertisedTargets checks every executable tuple in the
// server's compiler inventory, including both implant modes.
func TestSliverExecutablesAdvertisedTargets(t *testing.T) {
	certs.SetupCAs()
	targets := advertisedTargetsForFormat(clientpb.OutputFormat_EXECUTABLE)
	if len(targets) == 0 {
		t.Fatal("server advertises no executable compiler targets")
	}
	for _, target := range targets {
		for _, beacon := range []bool{false, true} {
			mode := "session"
			if beacon {
				mode = "beacon"
			}
			t.Run(fmt.Sprintf("%s/%s/%s", target.GOOS, target.GOARCH, mode), func(t *testing.T) {
				buildExecutableForTarget(t, target, beacon)
			})
		}
	}
}

func buildExecutableForTarget(t *testing.T, target *clientpb.CompilerTarget, beacon bool) {
	t.Helper()
	mode := "session"
	if beacon {
		mode = "beacon"
	}
	extension := ""
	if target.GOOS == WINDOWS {
		extension = ".exe"
	}
	name := fmt.Sprintf("m3_exe_%s_%s_%s", target.GOOS, target.GOARCH, mode)
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Join(GetSliversDir(), target.GOOS, target.GOARCH, name)) })

	config := &clientpb.ImplantConfig{
		GOOS: target.GOOS, GOARCH: target.GOARCH,
		C2:    []*clientpb.ImplantC2{{URL: "mtls://1.example.com"}},
		Debug: true, Format: clientpb.OutputFormat_EXECUTABLE,
		IsBeacon: beacon, IncludeMTLS: true,
	}
	build, err := GenerateConfig(name, config)
	if err != nil {
		t.Fatalf("prepare executable: %v", err)
	}
	httpConfig := configs.GenerateDefaultHTTPC2Config()
	gotPath, err := SliverExecutable(name, build, config, httpConfig.ImplantConfig)
	if err != nil {
		t.Fatalf("compile executable %s/%s %s: %v", target.GOOS, target.GOARCH, mode, err)
	}
	assertGeneratedOutputPath(t, gotPath, target.GOOS, target.GOARCH, name, name+extension)
	info, err := os.Stat(gotPath)
	if err != nil || info.Size() == 0 {
		t.Fatalf("output file is missing or empty: info=%v, err=%v", info, err)
	}
	assertDirectFormatBinary(t, gotPath, target.GOOS)
}

func advertisedTargetsForFormat(format clientpb.OutputFormat) []*clientpb.CompilerTarget {
	seen := make(map[string]bool)
	var targets []*clientpb.CompilerTarget
	for _, target := range GetCompilerTargets() {
		if target.Format != format {
			continue
		}
		key := target.GOOS + "/" + target.GOARCH
		if seen[key] {
			continue
		}
		seen[key] = true
		targets = append(targets, target)
	}
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].GOOS == targets[j].GOOS {
			return targets[i].GOARCH < targets[j].GOARCH
		}
		return targets[i].GOOS < targets[j].GOOS
	})
	return targets
}

func buildDirectFormatForTarget(t *testing.T, format clientpb.OutputFormat, target *clientpb.CompilerTarget, beacon bool) {
	t.Helper()
	mode := "session"
	if beacon {
		mode = "beacon"
	}
	kind := "service"
	extension := ".exe"
	if format == clientpb.OutputFormat_SHARED_LIB {
		kind = "shared"
		switch target.GOOS {
		case WINDOWS:
			extension = ".dll"
		case DARWIN:
			extension = ".dylib"
		case LINUX:
			extension = ".so"
		default:
			t.Fatalf("advertised shared-library target has unexpected OS %q", target.GOOS)
		}
	}
	name := fmt.Sprintf("m3_%s_%s_%s_%s", kind, target.GOOS, target.GOARCH, mode)
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Join(GetSliversDir(), target.GOOS, target.GOARCH, name)) })

	config := &clientpb.ImplantConfig{
		GOOS: target.GOOS, GOARCH: target.GOARCH,
		C2:    []*clientpb.ImplantC2{{URL: "mtls://1.example.com"}},
		Debug: true, Format: format, IsBeacon: beacon,
		IsService:   format == clientpb.OutputFormat_SERVICE,
		IsSharedLib: format == clientpb.OutputFormat_SHARED_LIB,
		IncludeMTLS: true,
		Exports:     []string{"StartW"},
		ServiceName: "M3Service",
	}
	build, err := GenerateConfig(name, config)
	if err != nil {
		t.Fatalf("prepare %s: %v", kind, err)
	}
	httpConfig := configs.GenerateDefaultHTTPC2Config()
	var gotPath string
	if format == clientpb.OutputFormat_SERVICE {
		gotPath, err = SliverExecutable(name, build, config, httpConfig.ImplantConfig)
	} else {
		gotPath, err = SliverSharedLibrary(name, build, config, httpConfig.ImplantConfig)
	}
	if err != nil {
		t.Fatalf("compile %s %s/%s %s: %v", kind, target.GOOS, target.GOARCH, mode, err)
	}
	assertGeneratedOutputPath(t, gotPath, target.GOOS, target.GOARCH, name, name+extension)
	info, err := os.Stat(gotPath)
	if err != nil || info.Size() == 0 {
		t.Fatalf("output file is missing or empty: info=%v, err=%v", info, err)
	}
	assertDirectFormatBinary(t, gotPath, target.GOOS)
	if format == clientpb.OutputFormat_SERVICE {
		assertServiceEntrypoint(t, gotPath)
	} else {
		assertSharedEntrypoint(t, gotPath, name)
	}
}

func assertDirectFormatBinary(t *testing.T, path, goos string) {
	t.Helper()
	switch goos {
	case WINDOWS:
		file, err := pe.Open(path)
		if err != nil {
			t.Fatalf("output is not PE: %v", err)
		}
		if err := file.Close(); err != nil {
			t.Errorf("close PE output: %v", err)
		}
	case DARWIN:
		file, err := macho.Open(path)
		if err != nil {
			t.Fatalf("output is not Mach-O: %v", err)
		}
		if err := file.Close(); err != nil {
			t.Errorf("close Mach-O output: %v", err)
		}
	case LINUX:
		file, err := elf.Open(path)
		if err != nil {
			t.Fatalf("output is not ELF: %v", err)
		}
		if err := file.Close(); err != nil {
			t.Errorf("close ELF output: %v", err)
		}
	default:
		t.Fatalf("unexpected target OS %q", goos)
	}
}

func assertServiceEntrypoint(t *testing.T, binaryPath string) {
	t.Helper()
	runnerPath := filepath.Join(filepath.Dir(filepath.Dir(binaryPath)), "src", "implant", "sliver", "runner", "runner.go")
	runner, err := os.ReadFile(runnerPath)
	if err != nil {
		t.Fatalf("read generated service runner: %v", err)
	}
	if !bytes.Contains(runner, []byte(`svc.Run("M3Service", &sliverService{})`)) {
		t.Fatalf("generated service runner is missing its service entrypoint")
	}
}

func assertSharedEntrypoint(t *testing.T, binaryPath, name string) {
	t.Helper()
	headerPath := filepath.Join(filepath.Dir(binaryPath), name+".h")
	header, err := os.ReadFile(headerPath)
	if err != nil {
		t.Fatalf("read generated shared-library header: %v", err)
	}
	if !bytes.Contains(header, []byte("StartW")) {
		t.Fatalf("generated shared-library header is missing the StartW export")
	}
}
