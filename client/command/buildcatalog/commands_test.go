package buildcatalog

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/carapace-sh/carapace"
	completer "github.com/carapace-sh/carapace/pkg/x"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

type catalogCompletionClient struct {
	rpcpb.SliverRPCClient
	names []string
	kinds []clientpb.BuildSpecKind
}

func (client *catalogCompletionClient) ListBuildSpecs(_ context.Context, query *clientpb.BuildSpecQuery, _ ...grpc.CallOption) (*clientpb.BuildSpecList, error) {
	client.kinds = append(client.kinds, query.GetKind())
	specs := make([]*clientpb.BuildSpec, 0, len(client.names)+2)
	for _, name := range client.names {
		specs = append(specs, &clientpb.BuildSpec{Ref: &clientpb.BuildSpecRef{Name: name}})
	}
	return &clientpb.BuildSpecList{Specs: append(specs, nil, &clientpb.BuildSpec{})}, nil
}

func TestCatalogInfoConsoleCompletion(t *testing.T) {
	for _, tc := range []struct {
		command string
		kind    clientpb.BuildSpecKind
		prefix  string
		names   []string
		want    []string
	}{
		{
			command: "cores",
			kind:    clientpb.BuildSpecKind_BUILD_SPEC_CORE,
			prefix:  "builtin/",
			names:   []string{"fixture/minimal-sliver-v1", "builtin/sliver", "builtin/sliver"},
			want:    []string{"builtin/sliver"},
		},
		{
			command: "transforms",
			kind:    clientpb.BuildSpecKind_BUILD_SPEC_TRANSFORM,
			prefix:  "builtin/stage-",
			names:   []string{"builtin/stage-frame", "builtin/xor", "builtin/stage-compress", "builtin/stage-frame"},
			want:    []string{"builtin/stage-compress", "builtin/stage-frame"},
		},
	} {
		t.Run(tc.command, func(t *testing.T) {
			rpc := &catalogCompletionClient{names: tc.names}
			menu := &cobra.Command{Use: "sliver"}
			menu.AddCommand(Commands(&console.SliverClient{Rpc: rpc})...)
			carapace.Gen(menu)
			t.Cleanup(completer.ClearStorage)

			result, err := completer.Complete(menu, "sliver", "_carapace", tc.command, "info", tc.prefix)
			if err != nil {
				t.Fatal(err)
			}
			var names []string
			for _, value := range result.Values {
				names = append(names, value.Value)
			}
			if !slices.Equal(names, tc.want) {
				t.Fatalf("name completions = %q, want %q", names, tc.want)
			}
			if !slices.Equal(rpc.kinds, []clientpb.BuildSpecKind{tc.kind}) {
				t.Fatalf("catalog queries = %v, want one %s query", rpc.kinds, tc.kind)
			}
		})
	}
}

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
