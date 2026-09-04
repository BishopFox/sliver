package generate

import (
	"reflect"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/gogo"
)

func TestImplantLDFlags(t *testing.T) {
	tests := []struct {
		name   string
		config *clientpb.ImplantConfig
		want   []string
	}{
		{
			name:   "non-debug linux",
			config: &clientpb.ImplantConfig{GOOS: LINUX},
			want:   []string{"-s -w -buildid="},
		},
		{
			name:   "non-debug windows",
			config: &clientpb.ImplantConfig{GOOS: WINDOWS},
			want:   []string{"-s -w -buildid= -H=windowsgui"},
		},
		{
			name:   "debug linux",
			config: &clientpb.ImplantConfig{GOOS: LINUX, Debug: true},
		},
		{
			name:   "debug windows",
			config: &clientpb.ImplantConfig{GOOS: WINDOWS, Debug: true},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := implantLDFlags(test.config); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("implantLDFlags() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestImplantLDFlagsReturnsIndependentSlices(t *testing.T) {
	config := &clientpb.ImplantConfig{GOOS: LINUX}
	first := implantLDFlags(config)
	first[0] = "modified"

	want := []string{nonDebugImplantLDFlags}
	if second := implantLDFlags(config); !reflect.DeepEqual(second, want) {
		t.Fatalf("implantLDFlags() reused mutable state: got %q, want %q", second, want)
	}
}

func TestIsZigCC(t *testing.T) {
	tests := []struct {
		cc   string
		want bool
	}{
		{cc: "", want: false},
		{cc: "gcc", want: false},
		{cc: "zig cc -target x86_64-linux-musl", want: true},
		{cc: "/tmp/zig cc -target x86_64-linux-musl", want: true},
		{cc: "/tmp/zig c++ -target x86_64-linux-musl", want: false},
		{cc: "something zig cc something", want: true},
	}
	for _, tt := range tests {
		if got := isZigCC(tt.cc); got != tt.want {
			t.Fatalf("isZigCC(%q) = %v, want %v", tt.cc, got, tt.want)
		}
	}
}

func TestApplyZigStaticLinking(t *testing.T) {
	cfg := &gogo.GoConfig{
		GOOS: "linux",
		CGO:  "1",
		CC:   "/tmp/zig cc -target x86_64-linux-musl",
	}
	ldflags, enabled := applyZigStaticLinking(cfg, "c-shared", []string{nonDebugImplantLDFlags})
	if enabled {
		t.Fatalf("did not expect -static flags for c-shared (zig emits an ar archive instead of a .so)")
	}
	if len(ldflags) != 1 {
		t.Fatalf("expected a single -ldflags string, got %d (%v)", len(ldflags), ldflags)
	}
	if ldflags[0] != nonDebugImplantLDFlags {
		t.Fatalf("expected ldflags unchanged, got %q", ldflags[0])
	}
}

func TestApplyLinuxShellcodeBuildID(t *testing.T) {
	ldflags := applyLinuxShellcodeBuildID([]string{nonDebugImplantLDFlags})
	if len(ldflags) != 1 {
		t.Fatalf("expected a single -ldflags string, got %d (%v)", len(ldflags), ldflags)
	}
	want := nonDebugImplantLDFlags + " " + linuxShellcodeBuildIDLDFlag
	if ldflags[0] != want {
		t.Fatalf("applyLinuxShellcodeBuildID() = %q, want %q", ldflags[0], want)
	}
}

func TestApplyZigStaticLinking_Executable(t *testing.T) {
	cfg := &gogo.GoConfig{
		GOOS: "linux",
		CGO:  "1",
		CC:   "/tmp/zig cc -target x86_64-linux-musl",
	}
	ldflags, enabled := applyZigStaticLinking(cfg, "", []string{nonDebugImplantLDFlags})
	if !enabled {
		t.Fatalf("expected static linking to be enabled for linux zig cc executable build")
	}
	if len(ldflags) != 1 {
		t.Fatalf("expected a single -ldflags string, got %d (%v)", len(ldflags), ldflags)
	}
	if !strings.Contains(ldflags[0], "-linkmode=external") {
		t.Fatalf("expected -linkmode=external in ldflags, got %q", ldflags[0])
	}
	if !strings.Contains(ldflags[0], "-extldflags=-static") {
		t.Fatalf("expected -extldflags=-static in ldflags, got %q", ldflags[0])
	}
	if !strings.Contains(ldflags[0], nonDebugImplantLDFlags) {
		t.Fatalf("expected production linker flags in ldflags, got %q", ldflags[0])
	}
}

func TestApplyZigStaticLinking_NotLinux(t *testing.T) {
	cfg := &gogo.GoConfig{
		GOOS: "windows",
		CGO:  "1",
		CC:   "zig cc -target x86_64-windows-gnu",
	}
	ldflags, enabled := applyZigStaticLinking(cfg, "c-shared", []string{nonDebugImplantLDFlags})
	if enabled {
		t.Fatalf("did not expect static linking to be enabled on %s", cfg.GOOS)
	}
	if len(ldflags) != 1 || ldflags[0] != nonDebugImplantLDFlags {
		t.Fatalf("expected ldflags unchanged, got %v", ldflags)
	}
}

func TestSanitizeZigForBuild_LinuxSharedObject(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	cfg := &gogo.GoConfig{
		GOOS: "linux",
		CGO:  "1",
		CC:   "/tmp/zig cc -target x86_64-linux-musl -static",
		CXX:  "/tmp/zig c++ -target x86_64-linux-musl -static",
	}
	sanitizeZigForBuild(cfg, "c-shared")

	if strings.Contains(cfg.CC, "-static") || strings.Contains(cfg.CXX, "-static") {
		t.Fatalf("expected -static to be stripped for shared objects, got CC=%q CXX=%q", cfg.CC, cfg.CXX)
	}
	if strings.Contains(cfg.CC, "-linux-musl") || strings.Contains(cfg.CXX, "-linux-musl") {
		t.Fatalf("expected linux-musl -> linux-gnu for shared objects, got CC=%q CXX=%q", cfg.CC, cfg.CXX)
	}
	if !strings.Contains(cfg.CC, "-linux-gnu") || !strings.Contains(cfg.CXX, "-linux-gnu") {
		t.Fatalf("expected linux-gnu target for shared objects, got CC=%q CXX=%q", cfg.CC, cfg.CXX)
	}

	// We should also add a debug prefix map for the Sliver-managed zig root to
	// avoid leaking absolute paths in DWARF.
	if !strings.Contains(cfg.CC, "-fdebug-prefix-map=") {
		t.Fatalf("expected -fdebug-prefix-map to be set, got CC=%q", cfg.CC)
	}
}

func TestSanitizeZigForBuild_Windows(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
	cfg := &gogo.GoConfig{
		GOOS: "windows",
		CGO:  "1",
		CC:   "/tmp/zig cc -target x86_64-windows-gnu",
		CXX:  "/tmp/zig c++ -target x86_64-windows-gnu",
	}
	sanitizeZigForBuild(cfg, "c-shared")

	if !strings.Contains(cfg.CC, "-fdebug-prefix-map=") {
		t.Fatalf("expected -fdebug-prefix-map to be set for zig builds, got CC=%q", cfg.CC)
	}
	if !strings.Contains(cfg.CXX, "-fdebug-prefix-map=") {
		t.Fatalf("expected -fdebug-prefix-map to be set for zig builds, got CXX=%q", cfg.CXX)
	}
}
