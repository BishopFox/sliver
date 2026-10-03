package console

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	consts "github.com/bishopfox/sliver/client/constants"
	reefconsole "github.com/reeflective/console"
	"github.com/spf13/cobra"
)

func pipedTestCommands(got *[]string) reefconsole.Commands {
	return func() *cobra.Command {
		root := &cobra.Command{Use: "root", SilenceErrors: true, SilenceUsage: true}
		root.AddCommand(&cobra.Command{
			Use:  "record value",
			Args: cobra.ExactArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				*got = append(*got, args[0])
				return nil
			},
		})
		root.AddCommand(&cobra.Command{
			Use: "fail",
			RunE: func(_ *cobra.Command, _ []string) error {
				return errors.New("command failed")
			},
		})
		return root
	}
}

func withPipedStdin(t *testing.T, input string) *os.File {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(writer, input); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reader.Close() })

	oldStdin := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() { os.Stdin = oldStdin })
	oldIsTerminal := isTerminal
	isTerminal = func(int) bool { return false }
	t.Cleanup(func() { isTerminal = oldIsTerminal })
	return reader
}

func TestStartClientRunsPipedCommandsInOrder(t *testing.T) {
	withPipedStdin(t, "\r\nrecord 'one two'\r\nrecord three\n  record four  ")
	var got []string
	con := NewConsole(false)
	con.Settings.ConsoleLogs = false
	cmds := pipedTestCommands(&got)

	if err := StartClient(con, nil, nil, nil, cmds, cmds, true, ""); err != nil {
		t.Fatalf("run piped commands: %v", err)
	}
	if want := []string{"one two", "three", "four"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("commands = %q, want %q", got, want)
	}
	if con.IsCLI {
		t.Fatal("piped console should retain console mode for menu switching")
	}
}

func TestStartClientPipedCommandsFollowMenuSwitch(t *testing.T) {
	withPipedStdin(t, "use\nrecord implant\n")
	var got []string
	con := NewConsole(false)
	con.Settings.ConsoleLogs = false
	serverCmds := func() *cobra.Command {
		root := &cobra.Command{Use: "root", SilenceErrors: true, SilenceUsage: true}
		root.AddCommand(&cobra.Command{Use: "use", Run: func(_ *cobra.Command, _ []string) {
			con.App.SwitchMenu(consts.ImplantMenu)
		}})
		return root
	}
	implantCmds := pipedTestCommands(&got)

	if err := StartClient(con, nil, nil, nil, serverCmds, implantCmds, true, ""); err != nil {
		t.Fatalf("run piped commands across menus: %v", err)
	}
	if want := []string{"implant"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("commands = %q, want %q", got, want)
	}
}

func TestStartClientRCPrecedesPipedStdin(t *testing.T) {
	reader := withPipedStdin(t, "record piped\n")
	var got []string
	con := NewConsole(false)
	con.Settings.ConsoleLogs = false
	cmds := pipedTestCommands(&got)

	if err := StartClient(con, nil, nil, nil, cmds, cmds, true, "record rc\n"); err != nil {
		t.Fatalf("run rc script: %v", err)
	}
	if want := []string{"rc"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("commands = %q, want %q", got, want)
	}
	remaining, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if string(remaining) != "record piped\n" {
		t.Fatalf("piped stdin was consumed after --rc: %q", remaining)
	}
}

func TestStartClientEmptyPipeExitsAtEOF(t *testing.T) {
	withPipedStdin(t, "")
	con := NewConsole(false)
	con.Settings.ConsoleLogs = false
	var got []string
	if err := StartClient(con, nil, nil, nil, pipedTestCommands(&got), nil, true, ""); err != nil {
		t.Fatalf("empty stdin: %v", err)
	}
}

func TestRunCommandScriptContinuesAndReportsErrors(t *testing.T) {
	var got []string
	con := NewConsole(false)
	var output bytes.Buffer
	con.printf = func(format string, args ...any) (int, error) {
		return fmt.Fprintf(&output, format, args...)
	}
	cmds := pipedTestCommands(&got)
	input := strings.NewReader("record first\nfail\nrecord second\nrecord 'unterminated\nrecord third\n")

	err := con.runCommandScript(cmds, cmds, input, "stdin")
	if err == nil || !strings.Contains(err.Error(), "2 error(s)") {
		t.Fatalf("error = %v, want two failures", err)
	}
	if want := []string{"first", "second", "third"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("commands = %q, want %q", got, want)
	}
	for _, line := range []string{"stdin line 2", "stdin line 4"} {
		if !strings.Contains(output.String(), line) {
			t.Fatalf("missing %q in output %q", line, output.String())
		}
	}
}
