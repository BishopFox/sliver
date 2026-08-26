package e2e

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"testing"
	"time"
)

var (
	testOptions          options
	goroutineLeakProfile bool
)

func init() {
	flag.StringVar(&testOptions.repoPath, "repo", ".", "path to the Sliver repository")
	flag.StringVar(&testOptions.serverPath, "server", "", "path to the unmodified Sliver server executable")
	flag.StringVar(&testOptions.serverArch, "server-arch", runtime.GOARCH, "expected architecture reported by the server")
	flag.StringVar(&testOptions.targetOS, "target-os", runtime.GOOS, "implant target operating system")
	flag.StringVar(&testOptions.targetArch, "target-arch", runtime.GOARCH, "implant target architecture")
	flag.StringVar(&testOptions.resultsDir, "results", "", "directory for JSON and Markdown coverage reports")
	flag.StringVar(&testOptions.transportCSV, "transports", strings.Join(transportOrder, ","), "comma-separated transports to run (mtls,wg,http)")
	flag.StringVar(&testOptions.modeCSV, "implant-modes", "session,beacon", "comma-separated implant modes to run (session,beacon)")
	flag.DurationVar(&testOptions.timeout, "e2e-timeout", 4*time.Hour, "overall comprehensive E2E timeout")
	flag.DurationVar(&testOptions.startupTimeout, "startup-timeout", 10*time.Minute, "Sliver daemon startup timeout")
	flag.DurationVar(&testOptions.connectTimeout, "connect-timeout", 5*time.Minute, "implant connection timeout")
	flag.DurationVar(&testOptions.commandTimeout, "command-timeout", 2*time.Minute, "individual implant command timeout")
	flag.DurationVar(&testOptions.beaconInterval, "beacon-interval", 10*time.Second, "beacon callback interval")
	flag.BoolVar(&testOptions.implantDebug, "implant-debug", false, "generate debug implants and capture their transport logs")
	flag.BoolVar(&goroutineLeakProfile, "goroutine-leak-profile", false, "write the goroutine leak profile at comprehensive E2E shutdown")
}

func TestMain(m *testing.M) {
	switch os.Getenv("SLIVER_E2E_HELPER") {
	case "sync":
		fmt.Printf("stdout:%s\n", os.Getenv("SLIVER_E2E_EXEC_MARKER"))
		fmt.Fprintf(os.Stderr, "stderr:%s\n", os.Getenv("SLIVER_E2E_EXEC_MARKER"))
		os.Exit(7)
	case "child":
		fmt.Printf("child:%s\n", os.Getenv("SLIVER_E2E_EXEC_MARKER"))
		time.Sleep(30 * time.Minute)
		os.Exit(0)
	}
	exitCode := m.Run()
	if goroutineLeakProfile {
		fmt.Fprintln(os.Stderr, "=== comprehensive E2E goroutine leak profile ===")
		profile := pprof.Lookup("goroutineleak")
		if profile == nil {
			fmt.Fprintln(os.Stderr, "goroutineleak profile is unavailable")
		} else if err := profile.WriteTo(os.Stderr, 1); err != nil {
			fmt.Fprintf(os.Stderr, "write goroutineleak profile: %v\n", err)
		}
	}
	os.Exit(exitCode)
}

func TestComprehensiveE2E(t *testing.T) {
	if testOptions.serverPath == "" {
		t.Skip("comprehensive E2E requires -server")
	}

	suite, err := newSuite(t, testOptions, true)
	if err != nil {
		t.Fatal(err)
	}
	defer suite.close()
	defer func() {
		if err := suite.writeCoverage(); err != nil {
			t.Errorf("%v", err)
		}
	}()

	if err := suite.run(); err != nil {
		t.Fatal(err)
	}
}
