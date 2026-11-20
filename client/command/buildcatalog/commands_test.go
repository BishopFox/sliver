package buildcatalog

import (
	"slices"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/jedib0t/go-pretty/v6/table"
)

func TestCatalogCommandFilterFlags(t *testing.T) {
	root := catalogCommand("cores", clientpb.BuildSpecKind_BUILD_SPEC_CORE, nil)
	if err := root.PersistentFlags().Set("target", "linux/amd64"); err != nil {
		t.Fatal(err)
	}
	if err := root.PersistentFlags().Set("availability", "unavailable"); err != nil {
		t.Fatal(err)
	}
	list, _, err := root.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	query, err := queryFromFlags(list, clientpb.BuildSpecKind_BUILD_SPEC_CORE)
	if err != nil {
		t.Fatal(err)
	}
	if query.TargetGOOS != "linux" || query.TargetGOARCH != "amd64" || query.Availability != clientpb.BuildSpecAvailability_BUILD_SPEC_UNAVAILABLE {
		t.Fatalf("inherited filters were not applied: %+v", query)
	}
	if err := root.PersistentFlags().Set("target", "linux"); err != nil {
		t.Fatal(err)
	}
	if _, err := queryFromFlags(list, clientpb.BuildSpecKind_BUILD_SPEC_CORE); err == nil {
		t.Fatal("expected a malformed target filter to fail")
	}
}

func TestCatalogDigestColumnFlag(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind clientpb.BuildSpecKind
	}{
		{name: "cores", kind: clientpb.BuildSpecKind_BUILD_SPEC_CORE},
		{name: "transforms", kind: clientpb.BuildSpecKind_BUILD_SPEC_TRANSFORM},
	} {
		for _, args := range [][]string{{}, {"list"}, {"--show-digests"}, {"list", "--show-digests"}} {
			t.Run(tc.name+"/"+strings.Join(args, "-"), func(t *testing.T) {
				root := catalogCommand(tc.name, tc.kind, nil)
				flag := root.PersistentFlags().Lookup("show-digests")
				if flag == nil || flag.DefValue != "false" {
					t.Fatalf("--show-digests must default to false: %v", flag)
				}
				root.SetArgs(args)
				if err := root.Execute(); err != nil {
					t.Fatal(err)
				}
				selected := root
				if len(args) > 0 && args[0] == "list" {
					var err error
					selected, _, err = root.Find([]string{"list"})
					if err != nil {
						t.Fatal(err)
					}
				}
				got := selected.Flag("show-digests")
				want := "false"
				if slices.Contains(args, "--show-digests") {
					want = "true"
				}
				if got == nil || got.Value.String() != want {
					t.Fatalf("--show-digests on %s %v = %v, want %s", tc.name, args, got, want)
				}
			})
		}
	}
}

func TestCatalogDigestColumnRendering(t *testing.T) {
	const digest = "sha256:unique-build-spec-digest"
	spec := &clientpb.BuildSpec{
		Ref:       &clientpb.BuildSpecRef{Name: "builtin/sliver", Version: "1", Digest: digest},
		Origins:   []*clientpb.BuildSpecOrigin{{Source: "builtin"}},
		Providers: []*clientpb.BuildSpecProvider{{Name: "server", Online: true}},
		Available: true,
	}
	for _, showDigests := range []bool{false, true} {
		tw := table.NewWriter()
		appendSpecTable(tw, []*clientpb.BuildSpec{spec}, showDigests)
		output := tw.RenderCSV()
		if got := strings.Contains(output, digest); got != showDigests {
			t.Fatalf("digest visible=%t with showDigests=%t: %q", got, showDigests, output)
		}
		header := strings.SplitN(output, "\n", 2)[0]
		if got := strings.Contains(strings.ToLower(header), "digest"); got != showDigests {
			t.Fatalf("digest column visible=%t with showDigests=%t: %q", got, showDigests, output)
		}
		for _, want := range []string{"builtin/sliver", "Version", "Source", "Available", "Providers"} {
			if !strings.Contains(output, want) {
				t.Fatalf("other column %q missing with showDigests=%t: %q", want, showDigests, output)
			}
		}
	}
}
