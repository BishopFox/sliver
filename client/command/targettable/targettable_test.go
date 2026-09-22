package targettable

import (
	"strings"
	"testing"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func TestFitSelectsLayoutsInOrder(t *testing.T) {
	tests := []struct {
		name         string
		width        int
		hasIntegrity bool
		wantWidth    int
		wantCalls    []Layout
	}{
		{
			name:         "full path exact fit",
			width:        50,
			hasIntegrity: true,
			wantWidth:    50,
			wantCalls: []Layout{
				{IncludeIntegrity: true, IncludeLocale: true},
			},
		},
		{
			name:         "process basename",
			width:        40,
			hasIntegrity: true,
			wantWidth:    40,
			wantCalls: []Layout{
				{IncludeIntegrity: true, IncludeLocale: true},
				{ProcessBasename: true, IncludeIntegrity: true, IncludeLocale: true},
			},
		},
		{
			name:         "omit integrity",
			width:        30,
			hasIntegrity: true,
			wantWidth:    30,
			wantCalls: []Layout{
				{IncludeIntegrity: true, IncludeLocale: true},
				{ProcessBasename: true, IncludeIntegrity: true, IncludeLocale: true},
				{ProcessBasename: true, IncludeLocale: true},
			},
		},
		{
			name:         "omit locale",
			width:        20,
			hasIntegrity: true,
			wantWidth:    20,
			wantCalls: []Layout{
				{IncludeIntegrity: true, IncludeLocale: true},
				{ProcessBasename: true, IncludeIntegrity: true, IncludeLocale: true},
				{ProcessBasename: true, IncludeLocale: true},
				{ProcessBasename: true},
			},
		},
		{
			name:         "compact fallback",
			width:        19,
			hasIntegrity: true,
			wantWidth:    10,
			wantCalls: []Layout{
				{IncludeIntegrity: true, IncludeLocale: true},
				{ProcessBasename: true, IncludeIntegrity: true, IncludeLocale: true},
				{ProcessBasename: true, IncludeLocale: true},
				{ProcessBasename: true},
				{Compact: true},
			},
		},
		{
			name:         "skip absent integrity",
			width:        20,
			hasIntegrity: false,
			wantWidth:    20,
			wantCalls: []Layout{
				{IncludeLocale: true},
				{ProcessBasename: true, IncludeLocale: true},
				{ProcessBasename: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := []Layout{}
			got := Fit(tt.width, tt.hasIntegrity, func(layout Layout) table.Writer {
				calls = append(calls, layout)
				return tableWithWidth(layoutWidth(layout))
			})

			if gotWidth := text.LongestLineLen(got.Render()); gotWidth != tt.wantWidth {
				t.Fatalf("selected table width = %d, want %d", gotWidth, tt.wantWidth)
			}
			if len(calls) != len(tt.wantCalls) {
				t.Fatalf("build calls = %#v, want %#v", calls, tt.wantCalls)
			}
			for index := range calls {
				if calls[index] != tt.wantCalls[index] {
					t.Fatalf("build call %d = %#v, want %#v", index, calls[index], tt.wantCalls[index])
				}
			}
		})
	}
}

func TestFitUsesANSIVisibleWidthAndAcceptsExactFit(t *testing.T) {
	const colored = "\x1b[31m12345\x1b[0m"
	buildCalls := 0
	build := func(Layout) table.Writer {
		buildCalls++
		tw := table.NewWriter()
		tw.SetStyle(singleCellStyle())
		tw.AppendRow(table.Row{colored})
		return tw
	}

	exactWidth := text.LongestLineLen(build(Layout{}).Render())
	buildCalls = 0
	got := Fit(exactWidth, true, build)

	if buildCalls != 1 {
		t.Fatalf("build calls = %d, want 1 for an ANSI-aware exact fit", buildCalls)
	}
	if gotWidth := text.LongestLineLen(got.Render()); gotWidth != exactWidth {
		t.Fatalf("selected table width = %d, want %d", gotWidth, exactWidth)
	}
}

func TestFitBoundsOversizedCompactFallback(t *testing.T) {
	const width = 20
	got := Fit(width, true, func(layout Layout) table.Writer {
		if layout.Compact {
			return tableWithWidth(40)
		}
		return tableWithWidth(50)
	})

	if gotWidth := text.LongestLineLen(got.Render()); gotWidth > width {
		t.Fatalf("compact table width = %d, want at most %d", gotWidth, width)
	}
}

func TestFitTreatsUnknownWidthAsUnlimited(t *testing.T) {
	buildCalls := 0
	got := Fit(0, true, func(layout Layout) table.Writer {
		buildCalls++
		return tableWithWidth(layoutWidth(layout))
	})

	if buildCalls != 1 {
		t.Fatalf("build calls = %d, want 1 for an unknown terminal width", buildCalls)
	}
	if gotWidth := text.LongestLineLen(got.Render()); gotWidth != 50 {
		t.Fatalf("selected table width = %d, want full-width table of 50", gotWidth)
	}
}

func TestProcessName(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "windows", path: `C:\Users\vagrant\AppData\Local\Temp\sliver.exe`, want: "sliver.exe"},
		{name: "UNC", path: `\\server\share\sliver.exe`, want: "sliver.exe"},
		{name: "POSIX", path: "/usr/local/bin/sliver", want: "sliver"},
		{name: "mixed separators", path: `C:\Temp/subdir\sliver.exe`, want: "sliver.exe"},
		{name: "base name", path: "sliver.exe", want: "sliver.exe"},
		{name: "empty", path: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ProcessName(tt.path); got != tt.want {
				t.Fatalf("ProcessName(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func layoutWidth(layout Layout) int {
	switch {
	case layout.Compact:
		return 10
	case !layout.ProcessBasename:
		return 50
	case layout.IncludeIntegrity:
		return 40
	case layout.IncludeLocale:
		return 30
	default:
		return 20
	}
}

func tableWithWidth(width int) table.Writer {
	tw := table.NewWriter()
	tw.SetStyle(singleCellStyle())
	tw.AppendRow(table.Row{strings.Repeat("x", width)})
	return tw
}

func singleCellStyle() table.Style {
	return table.Style{
		Format: table.FormatOptions{
			Footer: text.FormatDefault,
			Header: text.FormatDefault,
			Row:    text.FormatDefault,
		},
	}
}
