package console

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/version"
	reefconsole "github.com/reeflective/console"
)

func TestCheckLastUpdateBeforeReadline(t *testing.T) {
	now := time.Now()
	old := now.Add(-31 * 24 * time.Hour)
	recent := now.Add(-24 * time.Hour)

	for _, test := range []struct {
		name         string
		compiledAt   time.Time
		lastCheck    *time.Time
		wantReminder bool
	}{
		{name: "old build without update check", compiledAt: old, wantReminder: true},
		{name: "old build with old update check", compiledAt: old, lastCheck: &old, wantReminder: true},
		{name: "old build with recent update check", compiledAt: old, lastCheck: &recent},
		{name: "recent build without update check", compiledAt: recent},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("SLIVER_CLIENT_ROOT_DIR", root)
			t.Setenv("READLINE_CURSOR_POS", "1")
			if test.lastCheck != nil {
				lastCheck := strconv.FormatInt(test.lastCheck.Unix(), 10)
				if err := os.WriteFile(filepath.Join(root, consts.LastUpdateCheckFileName), []byte(lastCheck), 0600); err != nil {
					t.Fatal(err)
				}
			}

			previousCompiledAt := version.CompiledAt
			version.CompiledAt = strconv.FormatInt(test.compiledAt.Unix(), 10)
			t.Cleanup(func() { version.CompiledAt = previousCompiledAt })

			// Interactive startup selects the transient printer before the logo
			// and update reminder run, but Readline has not acquired stdin yet.
			app := reefconsole.New("sliver")
			con := &SliverClient{
				App:         app,
				printf:      app.TransientPrintf,
				jsonHandler: slog.NewJSONHandler(io.Discard, nil),
			}
			output := captureStartupOutput(t, con.CheckLastUpdate)
			if strings.Contains(output, "\x1b[6n") {
				t.Fatalf("startup output queried the cursor before readline: %q", output)
			}
			const reminder = "Check for updates with the 'update' command"
			if got := strings.Contains(output, reminder); got != test.wantReminder {
				t.Fatalf("update reminder present = %t, want %t; output: %q", got, test.wantReminder, output)
			}
		})
	}
}

func captureStartupOutput(t *testing.T, print func()) string {
	t.Helper()

	// An unexpected cursor query must fail the assertion rather than wait
	// indefinitely for a terminal response while the test is running.
	stdin, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stdin.Close() })
	previousStdin := os.Stdin
	os.Stdin = stdin
	t.Cleanup(func() { os.Stdin = previousStdin })

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reader.Close() })
	t.Cleanup(func() { _ = writer.Close() })
	previousStdout := os.Stdout
	os.Stdout = writer
	t.Cleanup(func() { os.Stdout = previousStdout })

	print()
	os.Stdout = previousStdout
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return string(output)
}
