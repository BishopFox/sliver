package beacons

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command/targettable"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

var beaconTableTestNow = time.Date(2026, time.September, 3, 21, 25, 41, 0, time.UTC)

func TestRenderBeaconsUsesFullProcessPathWhenItFits(t *testing.T) {
	beacon := testWindowsBeacon()
	layout := targettable.Layout{IncludeIntegrity: true, IncludeLocale: true}
	width := beaconCandidateWidth(t, []*clientpb.Beacon{beacon}, layout, true)

	rendered := renderBeaconTableAtWidth(t, []*clientpb.Beacon{beacon}, "", nil, width)
	if !strings.Contains(rendered, beacon.Filename+" (2020)") {
		t.Fatalf("full process path missing from table:\n%s", rendered)
	}
	if !strings.Contains(rendered, "(36s ago)") || !strings.Contains(rendered, "(in 24s)") {
		t.Fatalf("wide check-in timestamps changed unexpectedly:\n%s", rendered)
	}
	assertBeaconTableFits(t, rendered, width)
}

func TestRenderBeaconsUsesPortableProcessBasenameWhenRequired(t *testing.T) {
	tests := []struct {
		name         string
		beacon       *clientpb.Beacon
		hasIntegrity bool
		wantProcess  string
	}{
		{
			name:         "windows path",
			beacon:       testWindowsBeacon(),
			hasIntegrity: true,
			wantProcess:  "winterfell.exe (2020)",
		},
		{
			name:         "posix path",
			beacon:       testLinuxBeacon(),
			hasIntegrity: false,
			wantProcess:  "winterfell-agent (2020)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beacons := []*clientpb.Beacon{tt.beacon}
			fullLayout := targettable.Layout{
				IncludeIntegrity: tt.hasIntegrity,
				IncludeLocale:    true,
			}
			basenameLayout := fullLayout
			basenameLayout.ProcessBasename = true
			fullWidth := beaconCandidateWidth(t, beacons, fullLayout, tt.hasIntegrity)
			basenameWidth := beaconCandidateWidth(t, beacons, basenameLayout, tt.hasIntegrity)
			if basenameWidth >= fullWidth {
				t.Fatalf("basename candidate width %d is not narrower than full-path width %d", basenameWidth, fullWidth)
			}

			rendered := renderBeaconTableAtWidth(t, beacons, "", nil, basenameWidth)
			if !strings.Contains(rendered, tt.wantProcess) {
				t.Fatalf("process basename missing from table:\n%s", rendered)
			}
			if strings.Contains(rendered, tt.beacon.Filename) {
				t.Fatalf("full process path remained in basename layout:\n%s", rendered)
			}
			assertBeaconTableFits(t, rendered, basenameWidth)
		})
	}
}

func TestRenderBeaconsOmitsIntegrityBeforeLocale(t *testing.T) {
	beacon := testWindowsBeacon()
	beacons := []*clientpb.Beacon{beacon}
	withAll := targettable.Layout{
		ProcessBasename:  true,
		IncludeIntegrity: true,
		IncludeLocale:    true,
	}
	withoutIntegrity := withAll
	withoutIntegrity.IncludeIntegrity = false
	withoutLocale := withoutIntegrity
	withoutLocale.IncludeLocale = false

	allWidth := beaconCandidateWidth(t, beacons, withAll, true)
	withoutIntegrityWidth := beaconCandidateWidth(t, beacons, withoutIntegrity, true)
	withoutLocaleWidth := beaconCandidateWidth(t, beacons, withoutLocale, true)
	if !(withoutLocaleWidth < withoutIntegrityWidth && withoutIntegrityWidth < allWidth) {
		t.Fatalf("candidate widths did not shrink in omission order: all=%d no-integrity=%d no-locale=%d", allWidth, withoutIntegrityWidth, withoutLocaleWidth)
	}

	withoutIntegrityRendered := renderBeaconTableAtWidth(t, beacons, "", nil, withoutIntegrityWidth)
	assertHeaderPresence(t, withoutIntegrityRendered, "Integrity", false)
	assertHeaderPresence(t, withoutIntegrityRendered, "Locale", true)
	assertBeaconTableFits(t, withoutIntegrityRendered, withoutIntegrityWidth)

	withoutLocaleRendered := renderBeaconTableAtWidth(t, beacons, "", nil, withoutLocaleWidth)
	assertHeaderPresence(t, withoutLocaleRendered, "Integrity", false)
	assertHeaderPresence(t, withoutLocaleRendered, "Locale", false)
	assertBeaconTableFits(t, withoutLocaleRendered, withoutLocaleWidth)
}

func TestRenderBeaconsDoesNotAddIntegrityForNonWindowsTargets(t *testing.T) {
	beacon := testLinuxBeacon()
	beacons := []*clientpb.Beacon{beacon}
	layout := targettable.Layout{IncludeLocale: true}
	width := beaconCandidateWidth(t, beacons, layout, false)

	rendered := renderBeaconTableAtWidth(t, beacons, "", nil, width)
	assertHeaderPresence(t, rendered, "Integrity", false)
	assertHeaderPresence(t, rendered, "Locale", true)
	if !strings.Contains(rendered, beacon.Filename+" (2020)") {
		t.Fatalf("non-Windows full process path missing from table:\n%s", rendered)
	}
	assertBeaconTableFits(t, rendered, width)
}

func TestRenderBeaconsFiltersCanonicalFullRows(t *testing.T) {
	beacon := testWindowsBeacon()
	beacons := []*clientpb.Beacon{beacon}
	basenameLayout := targettable.Layout{
		ProcessBasename:  true,
		IncludeIntegrity: true,
		IncludeLocale:    true,
	}
	basenameWidth := beaconCandidateWidth(t, beacons, basenameLayout, true)
	withoutLocale := basenameLayout
	withoutLocale.IncludeIntegrity = false
	withoutLocale.IncludeLocale = false
	withoutLocaleWidth := beaconCandidateWidth(t, beacons, withoutLocale, true)

	t.Run("full process path survives basename projection", func(t *testing.T) {
		rendered := renderBeaconTableAtWidth(t, beacons, `AppData\Local\Temp`, nil, basenameWidth)
		if !strings.Contains(rendered, beacon.Name) {
			t.Fatalf("filter matching canonical process path removed beacon:\n%s", rendered)
		}
		if strings.Contains(rendered, `AppData\Local\Temp`) {
			t.Fatalf("basename layout unexpectedly displayed filtered directory:\n%s", rendered)
		}
	})

	t.Run("hidden integrity survives column projection", func(t *testing.T) {
		rendered := renderBeaconTableAtWidth(t, beacons, "", regexp.MustCompile(`^High$`), withoutLocaleWidth)
		if !strings.Contains(rendered, beacon.Name) {
			t.Fatalf("filter matching hidden integrity removed beacon:\n%s", rendered)
		}
		assertHeaderPresence(t, rendered, "Integrity", false)
	})
}

func TestRenderBeaconsDoesNotMutateEmptyIntegrity(t *testing.T) {
	beacon := testWindowsBeacon()
	beacon.Integrity = ""
	layout := targettable.Layout{IncludeIntegrity: true, IncludeLocale: true}
	width := beaconCandidateWidth(t, []*clientpb.Beacon{beacon}, layout, true)

	_ = renderBeaconTableAtWidth(t, []*clientpb.Beacon{beacon}, "", nil, width)
	if beacon.Integrity != "" {
		t.Fatalf("beacon integrity mutated to %q", beacon.Integrity)
	}
}

func TestRenderBeaconsCompactFallbackPreservesLegacyColumnsAndTimes(t *testing.T) {
	beacon := testWindowsBeacon()
	beacons := []*clientpb.Beacon{beacon}
	compactWidth := beaconCandidateWidth(t, beacons, targettable.Layout{Compact: true}, true)
	withoutOptionalWidth := beaconCandidateWidth(t, beacons, targettable.Layout{ProcessBasename: true}, true)
	if compactWidth >= withoutOptionalWidth {
		t.Fatalf("test fixture does not require compact fallback: compact=%d detailed=%d", compactWidth, withoutOptionalWidth)
	}
	rendered := renderBeaconTableAtWidth(t, beacons, "", nil, compactWidth)

	for _, omitted := range []string{"Tasks", "Remote Address", "Process (PID)", "Integrity", "Locale"} {
		assertHeaderPresence(t, rendered, omitted, false)
	}
	if !strings.Contains(rendered, "36s") || !strings.Contains(rendered, "24s") {
		t.Fatalf("compact relative check-in times missing:\n%s", rendered)
	}
	if strings.Contains(rendered, "36s ago") || strings.Contains(rendered, "in 24s") {
		t.Fatalf("compact fallback did not preserve legacy timestamp format:\n%s", rendered)
	}
	assertBeaconTableFits(t, rendered, compactWidth)
}

func testWindowsBeacon() *clientpb.Beacon {
	return &clientpb.Beacon{
		ID:                  "0c9f6b27-1111-2222-3333-444444444444",
		Name:                "winterfell-session",
		TasksCount:          3,
		TasksCountCompleted: 2,
		Transport:           "mtls",
		RemoteAddress:       "10.13.37.11:49760",
		Hostname:            "winterfell",
		Username:            `winterfell\vagrant`,
		Filename:            `C:\Users\vagrant\AppData\Local\Temp\sliver-range\winterfell.exe`,
		PID:                 2020,
		Integrity:           "High",
		OS:                  "windows",
		Arch:                "amd64",
		Locale:              "en-US",
		LastCheckin:         beaconTableTestNow.Add(-36 * time.Second).Unix(),
		NextCheckin:         beaconTableTestNow.Add(24 * time.Second).Unix(),
	}
}

func testLinuxBeacon() *clientpb.Beacon {
	beacon := testWindowsBeacon()
	beacon.OS = "linux"
	beacon.Hostname = "winterfell-linux"
	beacon.Username = "vagrant"
	beacon.Filename = "/opt/sliver/temporary/build/output/winterfell-agent"
	beacon.Integrity = ""
	return beacon
}

func testBeaconConsole() *console.SliverClient {
	return &console.SliverClient{
		Settings: &assets.ClientSettings{TableStyle: "SliverDefault"},
	}
}

func beaconCandidateWidth(t *testing.T, beacons []*clientpb.Beacon, layout targettable.Layout, hasIntegrity bool) int {
	t.Helper()
	tw := buildBeaconTable(beacons, "", nil, testBeaconConsole(), layout, hasIntegrity, beaconTableTestNow)
	return text.LongestLineLen(tw.Render())
}

func renderBeaconTableAtWidth(t *testing.T, beacons []*clientpb.Beacon, filter string, filterRegex *regexp.Regexp, width int) string {
	t.Helper()
	tw := renderBeaconsAtWidth(beacons, filter, filterRegex, testBeaconConsole(), width, beaconTableTestNow)
	return text.StripEscape(tw.Render())
}

func assertBeaconTableFits(t *testing.T, rendered string, width int) {
	t.Helper()
	if got := text.LongestLineLen(rendered); got > width {
		t.Fatalf("rendered table width %d exceeds terminal width %d:\n%s", got, width, rendered)
	}
}

func assertHeaderPresence(t *testing.T, rendered string, header string, want bool) {
	t.Helper()
	headerLine := strings.SplitN(rendered, "\n", 2)[0]
	if got := strings.Contains(strings.ToLower(headerLine), strings.ToLower(header)); got != want {
		t.Fatalf("header %q presence = %v, want %v; header line: %s", header, got, want, headerLine)
	}
}
