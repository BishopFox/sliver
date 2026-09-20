package crack

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/charmbracelet/x/ansi"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

func TestCrackBenchmarksCommandIsRegistered(t *testing.T) {
	root := Commands(nil)[0]
	benchmarks, _, err := root.Find([]string{"benchmarks"})
	if err != nil || benchmarks == root {
		t.Fatalf("find crack benchmarks: command=%v error=%v", benchmarks, err)
	}
	if benchmarks.Use != "benchmarks" {
		t.Fatalf("crack benchmarks use = %q, want benchmarks", benchmarks.Use)
	}
	if err := benchmarks.Args(benchmarks, nil); err != nil {
		t.Fatalf("crack benchmarks rejected no arguments: %v", err)
	}
	if err := benchmarks.Args(benchmarks, []string{"unexpected"}); err == nil {
		t.Fatal("crack benchmarks accepted a positional argument")
	}

	allFlag := benchmarks.Flags().Lookup("all")
	if allFlag == nil {
		t.Fatal("crack benchmarks is missing --all")
	}
	if allFlag.Shorthand != "a" {
		t.Fatalf("--all shorthand = %q, want %q", allFlag.Shorthand, "a")
	}
	if benchmarks.PersistentFlags().Lookup("all") != nil || root.PersistentFlags().Lookup("all") != nil {
		t.Fatal("--all must remain local to crack benchmarks")
	}
	if benchmarks.InheritedFlags().Lookup("timeout") == nil {
		t.Fatal("crack benchmarks must inherit --timeout")
	}
}

func TestRenderCrackBenchmarksSortsStationsAndRepresentativeModes(t *testing.T) {
	benchmarks := []*clientpb.CrackBenchmarkSnapshot{
		nil,
		{
			Name:                    "zeta",
			HostUUID:                "host-z",
			OperatorName:            "operator-z",
			CurrentHashcatVersion:   "v7.1.3",
			BenchmarkHashcatVersion: "v7.1.2",
			BenchmarkSchemaVersion:  1,
			BenchmarkedAt:           1_700_000_000,
			Online:                  true,
			Fresh:                   true,
			Benchmarks: map[int32]uint64{
				22000: 6_000_000_000_000,
				0:     999,
				3200:  5_000_000_000,
				100:   1_000,
				1400:  3_000_000,
				1000:  2_000_000,
				99999: 7_000_000,
			},
		},
		{Name: "alpha", HostUUID: "host-b", Benchmarks: map[int32]uint64{0: 1}},
		{Name: "alpha", HostUUID: "host-a", Benchmarks: map[int32]uint64{0: 2}},
	}

	raw := renderCrackBenchmarks(benchmarks, settings.SliverDefault, false)
	output := ansi.Strip(raw)
	requireOrderedCrackBenchmarkText(t, output, "host-a", "host-b", "host-z")
	zetaStart := strings.Index(output, ">>> Cached Crackstation 03 - zeta")
	if zetaStart < 0 {
		t.Fatalf("rendered benchmarks are missing sorted zeta section:\n%s", output)
	}
	requireOrderedCrackBenchmarkText(t, output[zetaStart:],
		"MD5", "SHA1", "NTLM", "SHA2-256", "bcrypt $2\\*$, Blowfish (Unix)", "WPA-PBKDF2-PMKID+EAPOL",
	)

	for _, expected := range []string{
		"Name", "Host UUID", "Operator", "Connection", "Online", "Cache", "Fresh",
		"Current Hashcat Version", "v7.1.3", "Benchmark Hashcat Version", "v7.1.2",
		"Benchmark Schema", "Benchmarked At", "2023-11-14T22:13:20Z", "Cached Modes",
		"operator-z", "Representative Rates", "Showing representative rates; use --all for every cached mode.",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("rendered benchmarks do not contain %q:\n%s", expected, output)
		}
	}
	requireCrackBenchmarkRow(t, output, 0, "MD5", "999 H/s")
	requireCrackBenchmarkRow(t, output, 100, "SHA1", "1.00 kH/s")
	requireCrackBenchmarkRow(t, output, 1000, "NTLM", "2.00 MH/s")
	requireCrackBenchmarkRow(t, output, 1400, "SHA2-256", "3.00 MH/s")
	requireCrackBenchmarkRow(t, output, 3200, "bcrypt $2\\*$, Blowfish (Unix)", "5.00 GH/s")
	requireCrackBenchmarkRow(t, output, 22000, "WPA-PBKDF2-PMKID+EAPOL", "6.00 TH/s")
	if hasCrackBenchmarkRow(output, 99999, "Plaintext", "7.00 MH/s") {
		t.Fatalf("default benchmark view unexpectedly contains non-representative mode 99999:\n%s", output)
	}

	for _, styled := range []string{
		console.StyleBoldPrimary.Render(">>> Cached Crackstation 01 - alpha"),
		console.StyleBoldPrimary.Render("host-a"),
		console.StyleBoldSuccess.Render("Online"),
		console.StyleBoldSuccess.Render("Fresh"),
		console.StyleBold.Render("Representative Rates"),
		console.StyleGray.Render("Showing representative rates; use --all for every cached mode."),
	} {
		if !strings.Contains(raw, styled) {
			t.Errorf("rendered benchmarks are missing shared Lip Gloss styling %q:\n%s", styled, raw)
		}
	}
}

func TestRenderCrackBenchmarksFallsBackToFirstSixModes(t *testing.T) {
	benchmarks := []*clientpb.CrackBenchmarkSnapshot{{
		Name: "fallback",
		Benchmarks: map[int32]uint64{
			23: 23,
			10: 10,
			22: 22,
			12: 12,
			20: 20,
			11: 11,
			21: 21,
		},
	}}

	output := ansi.Strip(renderCrackBenchmarks(benchmarks, settings.SliverDefault, false))
	requireOrderedCrackBenchmarkText(t, output,
		"md5($pass.$salt)",
		"Joomla < 2.5.18",
		"PostgreSQL",
		"md5($salt.$pass)",
		"osCommerce, xt:Commerce",
		"Juniper NetScreen/SSG (ScreenOS)",
	)
	if hasCrackBenchmarkRow(output, 23, "Skype", "23 H/s") {
		t.Fatalf("fallback benchmark view contains more than six modes:\n%s", output)
	}
}

func TestRenderCrackBenchmarksAllShowsEveryModeAndUnknownFallback(t *testing.T) {
	benchmarks := []*clientpb.CrackBenchmarkSnapshot{{
		Name: "all-modes",
		Benchmarks: map[int32]uint64{
			424242: 1_234,
			36100:  3_600,
			99999:  7_000_000,
			0:      500,
		},
	}}

	raw := renderCrackBenchmarks(benchmarks, settings.SliverDefault, true)
	output := ansi.Strip(raw)
	for _, row := range []struct {
		mode     int32
		hashType string
		rate     string
	}{
		{mode: 0, hashType: "MD5", rate: "500 H/s"},
		{mode: 36100, hashType: "YESCRYPT", rate: "3.60 kH/s"},
		{mode: 99999, hashType: "Plaintext", rate: "7.00 MH/s"},
		{mode: 424242, hashType: "Unknown hash mode", rate: "1.23 kH/s"},
	} {
		requireCrackBenchmarkRow(t, output, row.mode, row.hashType, row.rate)
	}
	if !strings.Contains(raw, console.StyleBold.Render("All Cached Rates")) {
		t.Fatalf("all-modes view is missing styled title:\n%s", raw)
	}
	if strings.Contains(output, "Showing representative rates") {
		t.Fatalf("all-modes view unexpectedly contains representative-rate hint:\n%s", output)
	}
}

func TestRenderCrackBenchmarksSanitizesMetadata(t *testing.T) {
	benchmarks := []*clientpb.CrackBenchmarkSnapshot{{
		Name:                    "alpha\nbeta\x1b]52;c;payload\a",
		HostUUID:                "host\r\nuuid",
		OperatorName:            "op\tname",
		CurrentHashcatVersion:   "current-v7\x00.2",
		BenchmarkHashcatVersion: "benchmark-v7\x00.1",
		Benchmarks:              map[int32]uint64{0: 1},
	}}

	raw := renderCrackBenchmarks(benchmarks, settings.SliverDefault, false)
	output := ansi.Strip(raw)
	for _, expected := range []string{"alphabeta]52;c;payload", "hostuuid", "opname", "current-v7.2", "benchmark-v7.1"} {
		if !strings.Contains(output, expected) {
			t.Errorf("sanitized benchmark metadata does not contain %q:\n%s", expected, output)
		}
	}
	for _, control := range []string{"\x1b]52;", "\a", "\r", "\x00", "\t"} {
		if strings.Contains(raw, control) {
			t.Errorf("rendered benchmark metadata retained control text %q:\n%q", control, raw)
		}
	}
}

func TestRenderCrackBenchmarksStylesConnectionAndFreshness(t *testing.T) {
	raw := renderCrackBenchmarks([]*clientpb.CrackBenchmarkSnapshot{
		{Name: "fresh", Online: true, Fresh: true, Benchmarks: map[int32]uint64{0: 1}},
		{Name: "stale", Online: false, Fresh: false, Benchmarks: map[int32]uint64{0: 1}},
	}, settings.SliverDefault, false)

	for _, styled := range []string{
		console.StyleBoldSuccess.Render("Online"),
		console.StyleBoldSuccess.Render("Fresh"),
		console.StyleBoldGray.Render("Offline"),
		console.StyleBoldWarning.Render("Stale"),
	} {
		if !strings.Contains(raw, styled) {
			t.Errorf("rendered benchmarks are missing status styling %q:\n%s", styled, raw)
		}
	}
}

func TestRenderCrackBenchmarksBoundsEveryLine(t *testing.T) {
	raw := renderCrackBenchmarks([]*clientpb.CrackBenchmarkSnapshot{{
		Name:                    strings.Repeat("very-long-crackstation-name-", 8),
		HostUUID:                "12345678-1234-1234-1234-123456789abc",
		OperatorName:            strings.Repeat("operator-", 16),
		CurrentHashcatVersion:   strings.Repeat("current-version-", 10),
		BenchmarkHashcatVersion: strings.Repeat("benchmark-version-", 10),
		BenchmarkedAt:           1_700_000_000,
		Benchmarks:              map[int32]uint64{6211: 1_000_000},
	}}, settings.SliverDefault, true)

	for lineNumber, line := range strings.Split(raw, "\n") {
		if width := ansi.StringWidth(line); width > crackBenchmarkOutputWidth {
			t.Errorf("rendered line %d width = %d, want <= %d:\n%s", lineNumber+1, width, crackBenchmarkOutputWidth, line)
		}
	}
	if !strings.Contains(ansi.Strip(raw), "…") {
		t.Fatalf("long benchmark heading was not truncated:\n%s", raw)
	}
}

func TestFormatCachedBenchmarkTime(t *testing.T) {
	if got := formatCachedBenchmarkTime(0); got != "-" {
		t.Fatalf("zero benchmark time = %q, want -", got)
	}
	if got := formatCachedBenchmarkTime(1_700_000_000); got != "2023-11-14T22:13:20Z" {
		t.Fatalf("benchmark time = %q, want UTC RFC3339", got)
	}
}

func TestRenderCrackBenchmarksHandlesNilAndEmptyData(t *testing.T) {
	for _, benchmarks := range [][]*clientpb.CrackBenchmarkSnapshot{nil, {nil, nil}} {
		if got := renderCrackBenchmarks(benchmarks, settings.SliverDefault, false); got != "" {
			t.Fatalf("renderCrackBenchmarks(%#v) = %q, want empty output", benchmarks, got)
		}
	}

	raw := renderCrackBenchmarks([]*clientpb.CrackBenchmarkSnapshot{{
		HostUUID: "host-only",
	}}, settings.SliverDefault, false)
	output := ansi.Strip(raw)
	for _, expected := range []string{
		">>> Cached Crackstation 01 - host-only",
		"No benchmark rates cached",
		"Representative Rates",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("empty benchmark snapshot does not contain %q:\n%s", expected, output)
		}
	}

	unknown := ansi.Strip(renderCrackBenchmarks([]*clientpb.CrackBenchmarkSnapshot{{}}, settings.SliverDefault, false))
	if !strings.Contains(unknown, ">>> Cached Crackstation 01 - Unknown crackstation") {
		t.Fatalf("missing unknown-station identity fallback:\n%s", unknown)
	}
}

type crackBenchmarksRPCCapture struct {
	rpcpb.SliverRPCClient
	response           *clientpb.CrackBenchmarkSnapshots
	err                error
	benchmarkCalls     int
	crackstationsCalls int
	hadDeadline        bool
}

func (capture *crackBenchmarksRPCCapture) CrackstationBenchmarks(ctx context.Context, _ *commonpb.Empty, _ ...grpc.CallOption) (*clientpb.CrackBenchmarkSnapshots, error) {
	capture.benchmarkCalls++
	_, capture.hadDeadline = ctx.Deadline()
	return capture.response, capture.err
}

func (capture *crackBenchmarksRPCCapture) Crackstations(_ context.Context, _ *commonpb.Empty, _ ...grpc.CallOption) (*clientpb.Crackstations, error) {
	capture.crackstationsCalls++
	return &clientpb.Crackstations{}, nil
}

func TestCrackBenchmarksCommandUsesDedicatedRPCWithDeadline(t *testing.T) {
	capture := &crackBenchmarksRPCCapture{response: &clientpb.CrackBenchmarkSnapshots{Snapshots: []*clientpb.CrackBenchmarkSnapshot{{
		Name:       "station",
		HostUUID:   "host-id",
		Benchmarks: map[int32]uint64{0: 1},
	}}}}
	con := console.NewConsole(false)
	emptyCommands := func() *cobra.Command { return &cobra.Command{Use: "test"} }
	if err := console.StartClient(con, capture, nil, nil, emptyCommands, emptyCommands, false, ""); err != nil {
		t.Fatal(err)
	}

	root := Commands(con)[0]
	root.SetArgs([]string{"benchmarks", "--timeout=1", "-a"})
	root.SilenceErrors = true
	root.SilenceUsage = true
	if err := root.Execute(); err != nil {
		t.Fatalf("execute crack benchmarks: %v", err)
	}
	if capture.benchmarkCalls != 1 {
		t.Fatalf("CrackstationBenchmarks calls = %d, want 1", capture.benchmarkCalls)
	}
	if capture.crackstationsCalls != 0 {
		t.Fatalf("Crackstations calls = %d, want 0", capture.crackstationsCalls)
	}
	if !capture.hadDeadline {
		t.Fatal("CrackstationBenchmarks context did not inherit --timeout")
	}
}

func TestCrackBenchmarksCommandHandlesNilResponse(t *testing.T) {
	capture := &crackBenchmarksRPCCapture{}
	con := console.NewConsole(false)
	emptyCommands := func() *cobra.Command { return &cobra.Command{Use: "test"} }
	if err := console.StartClient(con, capture, nil, nil, emptyCommands, emptyCommands, false, ""); err != nil {
		t.Fatal(err)
	}

	root := Commands(con)[0]
	root.SetArgs([]string{"benchmarks", "--timeout=1"})
	root.SilenceErrors = true
	root.SilenceUsage = true
	if err := root.Execute(); err != nil {
		t.Fatalf("execute crack benchmarks with nil response: %v", err)
	}
	if capture.benchmarkCalls != 1 {
		t.Fatalf("CrackstationBenchmarks calls = %d, want 1", capture.benchmarkCalls)
	}
}

func requireOrderedCrackBenchmarkText(t *testing.T, output string, values ...string) {
	t.Helper()
	remaining := output
	for _, value := range values {
		index := strings.Index(remaining, value)
		if index < 0 {
			t.Fatalf("rendered benchmarks do not contain %q in order:\n%s", value, output)
		}
		remaining = remaining[index+len(value):]
	}
}

func requireCrackBenchmarkRow(t *testing.T, output string, mode int32, hashType string, rate string) {
	t.Helper()
	if !hasCrackBenchmarkRow(output, mode, hashType, rate) {
		t.Fatalf("rendered benchmarks are missing mode %d, hash type %q, rate %q:\n%s", mode, hashType, rate, output)
	}
}

func hasCrackBenchmarkRow(output string, mode int32, hashType string, rate string) bool {
	pattern := `(?m)^[ \t]*` + regexp.QuoteMeta(strconv.FormatInt(int64(mode), 10)) + `[ \t]+` +
		regexp.QuoteMeta(hashType) + `[ \t]+` + regexp.QuoteMeta(rate) + `[ \t]*$`
	return regexp.MustCompile(pattern).MatchString(output)
}
