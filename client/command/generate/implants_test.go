package generate

import (
	"strings"
	"testing"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestImplantsTableShowsRecordedCore(t *testing.T) {
	builds := &clientpb.ImplantBuilds{
		Configs: map[string]*clientpb.ImplantConfig{
			"CUSTOM_BUILD": {GOOS: "linux", GOARCH: "amd64", Format: clientpb.OutputFormat_EXECUTABLE},
			"LEGACY_BUILD": {GOOS: "windows", GOARCH: "amd64", Format: clientpb.OutputFormat_EXECUTABLE},
		},
		CoreSpecs: map[string]*clientpb.BuildSpecRef{
			"CUSTOM_BUILD": {Name: "example/custom", Version: "2.3.1"},
		},
		ResourceIDs: map[string]*clientpb.ResourceID{
			"CUSTOM_BUILD": {},
			"LEGACY_BUILD": {},
		},
	}
	con := &console.SliverClient{Settings: &assets.ClientSettings{TableStyle: "SliverDefault"}}
	rendered := renderImplantBuilds(builds, ImplantBuildFilter{}, con)
	if !strings.Contains(rendered, "Core") || strings.Contains(rendered, "Template") {
		t.Fatalf("implants table has wrong provenance header:\n%s", rendered)
	}
	for _, tc := range []struct{ build, core string }{
		{"CUSTOM_BUILD", "example/custom@2.3.1"},
		{"LEGACY_BUILD", "Unknown (legacy)"},
	} {
		found := false
		for _, line := range strings.Split(rendered, "\n") {
			if strings.Contains(line, tc.build) {
				found = strings.Contains(line, tc.core)
				break
			}
		}
		if !found {
			t.Fatalf("%s did not show core %q:\n%s", tc.build, tc.core, rendered)
		}
	}
}
