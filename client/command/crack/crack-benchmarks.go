package crack

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

var representativeCrackBenchmarkModes = map[int32]struct{}{
	0:     {}, // MD5
	100:   {}, // SHA1
	1000:  {}, // NTLM
	1400:  {}, // SHA2-256
	3200:  {}, // bcrypt
	22000: {}, // WPA-PBKDF2
}

const (
	crackBenchmarkOutputWidth        = 80
	crackBenchmarkHashTypeWidth      = 48
	crackBenchmarkMetadataValueWidth = 48
)

// CrackBenchmarksCmd displays benchmark results persisted by the server.
func CrackBenchmarksCmd(cmd *cobra.Command, con *console.SliverClient, _ []string) {
	ctx, cancel := crackCommandContext(cmd.Context(), cmd)
	defer cancel()

	response, err := con.Rpc.CrackstationBenchmarks(ctx, &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	showAll, _ := cmd.Flags().GetBool("all")
	benchmarks := response.GetSnapshots()
	rendered := renderCrackBenchmarks(benchmarks, settings.GetTableStyle(con), showAll)
	if rendered == "" {
		con.PrintInfof("No cached crackstation benchmarks\n")
		return
	}
	con.Printf("%s\n", rendered)
}

func renderCrackBenchmarks(benchmarks []*clientpb.CrackBenchmarkSnapshot, style table.Style, showAll bool) string {
	ordered := make([]*clientpb.CrackBenchmarkSnapshot, 0, len(benchmarks))
	for _, benchmark := range benchmarks {
		if benchmark != nil {
			ordered = append(ordered, benchmark)
		}
	}
	if len(ordered) == 0 {
		return ""
	}

	sort.SliceStable(ordered, func(i, j int) bool {
		leftName := safeCrackCell(ordered[i].GetName())
		rightName := safeCrackCell(ordered[j].GetName())
		if leftName == rightName {
			return safeCrackCell(ordered[i].GetHostUUID()) < safeCrackCell(ordered[j].GetHostUUID())
		}
		return leftName < rightName
	})

	sections := make([]string, 0, len(ordered)*2+1)
	for index, benchmark := range ordered {
		identity := cachedBenchmarkIdentity(benchmark)
		title := renderCachedBenchmarkTitle(index, identity)
		metadata := table.NewWriter()
		metadata.SetStyle(boundedCrackBenchmarkStyle(style))
		metadata.SetColumnConfigs([]table.ColumnConfig{
			{Number: 2, WidthMax: crackBenchmarkMetadataValueWidth},
		})
		metadata.AppendRow(table.Row{console.StyleBold.Render("Name"), valueOrDash(benchmark.GetName())})
		metadata.AppendRow(table.Row{console.StyleBold.Render("Host UUID"), console.StyleBoldPrimary.Render(valueOrDash(benchmark.GetHostUUID()))})
		metadata.AppendRow(table.Row{console.StyleBold.Render("Operator"), valueOrDash(benchmark.GetOperatorName())})
		metadata.AppendRow(table.Row{console.StyleBold.Render("Connection"), renderCachedBenchmarkConnection(benchmark.GetOnline())})
		metadata.AppendRow(table.Row{console.StyleBold.Render("Cache"), renderCachedBenchmarkFreshness(benchmark.GetFresh())})
		metadata.AppendRow(table.Row{console.StyleBold.Render("Current Hashcat Version"), valueOrDash(benchmark.GetCurrentHashcatVersion())})
		metadata.AppendRow(table.Row{console.StyleBold.Render("Benchmark Hashcat Version"), valueOrDash(benchmark.GetBenchmarkHashcatVersion())})
		metadata.AppendRow(table.Row{console.StyleBold.Render("Benchmark Schema"), strconv.FormatUint(uint64(benchmark.GetBenchmarkSchemaVersion()), 10)})
		metadata.AppendRow(table.Row{console.StyleBold.Render("Benchmarked At"), formatCachedBenchmarkTime(benchmark.GetBenchmarkedAt())})
		metadata.AppendRow(table.Row{console.StyleBold.Render("Cached Modes"), strconv.Itoa(len(benchmark.GetBenchmarks()))})
		sections = append(sections, title+"\n"+metadata.Render())

		rates := table.NewWriter()
		rates.SetStyle(boundedCrackBenchmarkStyle(style))
		rates.SetColumnConfigs([]table.ColumnConfig{
			{Name: "Hash Type", WidthMax: crackBenchmarkHashTypeWidth},
		})
		if showAll {
			rates.SetTitle(console.StyleBold.Render("All Cached Rates"))
		} else {
			rates.SetTitle(console.StyleBold.Render("Representative Rates"))
		}
		rates.AppendHeader(table.Row{"Mode", "Hash Type", "Rate"})
		modes := cachedBenchmarkModes(benchmark.GetBenchmarks(), showAll)
		if len(modes) == 0 {
			rates.AppendRow(table.Row{"-", "No benchmark rates cached", "-"})
		} else {
			for _, mode := range modes {
				rates.AppendRow(table.Row{
					strconv.FormatInt(int64(mode), 10),
					crackBenchmarkModeName(mode),
					humanizeHashRate(benchmark.GetBenchmarks()[mode]),
				})
			}
		}
		sections = append(sections, rates.Render())
	}
	if !showAll {
		sections = append(sections, console.StyleGray.Render("Showing representative rates; use --all for every cached mode."))
	}
	return strings.Join(sections, "\n\n")
}

func boundedCrackBenchmarkStyle(style table.Style) table.Style {
	style.Size.WidthMax = crackBenchmarkOutputWidth
	return style
}

func renderCachedBenchmarkTitle(index int, identity string) string {
	prefix := fmt.Sprintf(">>> Cached Crackstation %02d - ", index+1)
	maximumIdentityWidth := crackBenchmarkOutputWidth - ansi.StringWidth(prefix)
	if maximumIdentityWidth < 1 {
		return console.StyleBoldPrimary.Render(ansi.Truncate(prefix, crackBenchmarkOutputWidth, "…"))
	}
	identity = ansi.Truncate(identity, maximumIdentityWidth, "…")
	return console.StyleBoldPrimary.Render(prefix + identity)
}

func cachedBenchmarkIdentity(benchmark *clientpb.CrackBenchmarkSnapshot) string {
	if benchmark == nil {
		return "Unknown crackstation"
	}
	if name := safeCrackCell(benchmark.GetName()); name != "" {
		return name
	}
	if hostUUID := safeCrackCell(benchmark.GetHostUUID()); hostUUID != "" {
		return hostUUID
	}
	return "Unknown crackstation"
}

func renderCachedBenchmarkConnection(online bool) string {
	if online {
		return console.StyleBoldSuccess.Render("Online")
	}
	return console.StyleBoldGray.Render("Offline")
}

func renderCachedBenchmarkFreshness(fresh bool) string {
	if fresh {
		return console.StyleBoldSuccess.Render("Fresh")
	}
	return console.StyleBoldWarning.Render("Stale")
}

func formatCachedBenchmarkTime(unix int64) string {
	if unix == 0 {
		return "-"
	}
	return time.Unix(unix, 0).UTC().Format(time.RFC3339)
}

func cachedBenchmarkModes(benchmarks map[int32]uint64, showAll bool) []int32 {
	modes := make([]int32, 0, len(benchmarks))
	for mode := range benchmarks {
		if showAll {
			modes = append(modes, mode)
			continue
		}
		if _, representative := representativeCrackBenchmarkModes[mode]; representative {
			modes = append(modes, mode)
		}
	}
	sort.Slice(modes, func(i, j int) bool { return modes[i] < modes[j] })
	if showAll || len(modes) != 0 {
		return modes
	}

	for mode := range benchmarks {
		modes = append(modes, mode)
	}
	sort.Slice(modes, func(i, j int) bool { return modes[i] < modes[j] })
	if len(modes) > 6 {
		modes = modes[:6]
	}
	return modes
}

func crackBenchmarkModeName(mode int32) string {
	if name, ok := hashcatHashTypeName(mode); ok {
		return name
	}
	if enumName := clientpb.HashType(mode).String(); enumName != strconv.FormatInt(int64(mode), 10) {
		return enumName
	}
	return "Unknown hash mode"
}
