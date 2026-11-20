package extensions

import (
	"reflect"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/spf13/cobra"
)

func newTestExtensionCommand(t *testing.T) (*cobra.Command, *cobra.Command) {
	t.Helper()

	previous := loadedExtensions
	loadedExtensions = map[string]*ExtCommand{}
	t.Cleanup(func() {
		loadedExtensions = previous
	})

	root := &cobra.Command{Use: "root", SilenceErrors: true, SilenceUsage: true}
	root.AddGroup(&cobra.Group{ID: consts.ExtensionHelpGroup, Title: "Extensions"})
	ExtensionRegisterCommand(&ExtCommand{
		CommandName: "delegationbof",
		Help:        "Test BOF",
		Arguments: []*extensionArgument{
			{Name: "type", Type: "int", Desc: "Delegation type"},
			{Name: "fqdn", Type: "string", Desc: "Child domain"},
		},
	}, root, &console.SliverClient{})

	extensionCmd, _, err := root.Find([]string{"delegationbof"})
	if err != nil {
		t.Fatal(err)
	}
	return root, extensionCmd
}

func TestExtensionRegisterCommandPassesThroughArguments(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		want        []string
		wantTimeout int64
		wantSave    bool
	}{
		{
			name:        "named flags without separator",
			args:        []string{"--type", "6", "--fqdn", "child.htb.local"},
			want:        []string{"--type", "6", "--fqdn", "child.htb.local"},
			wantTimeout: defaultTimeout,
		},
		{
			name:        "named flags with existing separator",
			args:        []string{"--", "--type", "6", "--fqdn", "child.htb.local"},
			want:        []string{"--type", "6", "--fqdn", "child.htb.local"},
			wantTimeout: defaultTimeout,
		},
		{
			name:        "quoted value containing spaces",
			args:        []string{"--type", "6", "--fqdn", "child domain.htb.local"},
			want:        []string{"--type", "6", "--fqdn", "child domain.htb.local"},
			wantTimeout: defaultTimeout,
		},
		{
			name:        "positional arguments",
			args:        []string{"6", "child.htb.local"},
			want:        []string{"6", "child.htb.local"},
			wantTimeout: defaultTimeout,
		},
		{
			name:        "single dash extension flags",
			args:        []string{"-type", "6", "-fqdn", "child.htb.local"},
			want:        []string{"-type", "6", "-fqdn", "child.htb.local"},
			wantTimeout: defaultTimeout,
		},
		{
			name:        "Sliver flags before extension arguments",
			args:        []string{"--timeout", "90", "--save", "--type", "6", "--fqdn", "child.htb.local"},
			want:        []string{"--type", "6", "--fqdn", "child.htb.local"},
			wantTimeout: 90,
			wantSave:    true,
		},
		{
			name:        "Sliver flags before existing separator",
			args:        []string{"-t90", "-s", "--", "--type", "6", "--fqdn", "child.htb.local"},
			want:        []string{"--type", "6", "--fqdn", "child.htb.local"},
			wantTimeout: 90,
			wantSave:    true,
		},
		{
			name:        "Sliver-looking flag after extension arguments",
			args:        []string{"--type", "6", "--timeout", "12"},
			want:        []string{"--type", "6", "--timeout", "12"},
			wantTimeout: defaultTimeout,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, extensionCmd := newTestExtensionCommand(t)

			var got []string
			var parseErr error
			var helpRequested bool
			extensionCmd.Run = func(cmd *cobra.Command, args []string) {
				got, helpRequested, parseErr = parseExtensionCommandArgs(cmd, args)
			}

			root.SetArgs(append([]string{"delegationbof"}, test.args...))
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			if helpRequested {
				t.Fatal("help unexpectedly requested")
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("extension args = %#v, want %#v", got, test.want)
			}

			timeout, err := extensionCmd.Flags().GetInt64("timeout")
			if err != nil {
				t.Fatal(err)
			}
			if timeout != test.wantTimeout {
				t.Fatalf("timeout = %d, want %d", timeout, test.wantTimeout)
			}
			save, err := extensionCmd.Flags().GetBool("save")
			if err != nil {
				t.Fatal(err)
			}
			if save != test.wantSave {
				t.Fatalf("save = %v, want %v", save, test.wantSave)
			}
		})
	}
}

func TestExtensionCommandHelpShowsDirectAndLegacySyntax(t *testing.T) {
	_, extensionCmd := newTestExtensionCommand(t)

	if !strings.Contains(extensionCmd.Use, "delegationbof --type TYPE --fqdn FQDN") {
		t.Fatalf("Use = %q", extensionCmd.Use)
	}
	if !strings.Contains(extensionCmd.Example, "delegationbof --type TYPE --fqdn FQDN") {
		t.Fatalf("Example does not show direct syntax: %q", extensionCmd.Example)
	}
	if !strings.Contains(extensionCmd.Example, "delegationbof -- --type TYPE --fqdn FQDN") {
		t.Fatalf("Example does not show legacy syntax: %q", extensionCmd.Example)
	}
}

func TestBOFSetupAndParseFlagsAcceptsNamedAndPositionalArguments(t *testing.T) {
	ext := &ExtCommand{
		Arguments: []*extensionArgument{
			{Name: "type", Type: "int"},
			{Name: "fqdn", Type: "string"},
		},
	}
	tests := []struct {
		name     string
		args     []string
		wantFQDN string
	}{
		{
			name:     "double dash flags",
			args:     []string{"--type", "6", "--fqdn", "child.htb.local"},
			wantFQDN: "child.htb.local",
		},
		{
			name:     "existing separator",
			args:     []string{"--", "--type", "6", "--fqdn", "child.htb.local"},
			wantFQDN: "child.htb.local",
		},
		{
			name:     "single dash flags",
			args:     []string{"-type", "6", "-fqdn", "child.htb.local"},
			wantFQDN: "child.htb.local",
		},
		{
			name:     "quoted value containing spaces",
			args:     []string{"--type", "6", "--fqdn", "child domain.htb.local"},
			wantFQDN: "child domain.htb.local",
		},
		{
			name:     "positional arguments",
			args:     []string{"6", "child.htb.local"},
			wantFQDN: "child.htb.local",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fs, stringValues, _, intValues, _, _, err := bofSetupAndParseFlags(test.args, ext)
			if err != nil {
				t.Fatal(err)
			}
			if !bofFlagWasProvided(fs, "type") || !bofFlagWasProvided(fs, "fqdn") {
				t.Fatal("normalized BOF flags were not marked as provided")
			}
			if got := *intValues["type"]; got != 6 {
				t.Fatalf("type = %d, want 6", got)
			}
			if got := *stringValues["fqdn"]; got != test.wantFQDN {
				t.Fatalf("fqdn = %q, want %q", got, test.wantFQDN)
			}
		})
	}
}
