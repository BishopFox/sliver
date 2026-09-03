package e2e

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/bishopfox/sliver/test/e2e/shellcodecoverage"
)

var testOptions options

func defaultSuiteScope() string {
	scope := strings.TrimSpace(os.Getenv("SLIVER_E2E_SCOPE"))
	if scope == "" {
		return suiteScopeComprehensive
	}
	return scope
}

func init() {
	flag.StringVar(&testOptions.repoPath, "repo", ".", "path to the Sliver repository")
	flag.StringVar(&testOptions.serverPath, "server", "", "path to the unmodified Sliver server executable")
	flag.StringVar(&testOptions.serverArch, "server-arch", runtime.GOARCH, "expected architecture reported by the server")
	flag.StringVar(&testOptions.targetOS, "target-os", runtime.GOOS, "implant target operating system")
	flag.StringVar(&testOptions.targetArch, "target-arch", runtime.GOARCH, "implant target architecture")
	flag.StringVar(&testOptions.resultsDir, "results", "", "directory for JSON and Markdown coverage reports")
	flag.StringVar(&testOptions.transportCSV, "transports", strings.Join(transportOrder, ","), "comma-separated transports to run (mtls,wg,http)")
	flag.StringVar(&testOptions.modeCSV, "implant-modes", "session,beacon", "comma-separated implant modes to run (session,beacon)")
	flag.StringVar(&testOptions.suiteScope, "suite-scope", defaultSuiteScope(), "E2E suite scope (comprehensive, rportfwd, or portfwd-socks5)")
	flag.DurationVar(&testOptions.timeout, "e2e-timeout", 4*time.Hour, "overall comprehensive E2E timeout")
	flag.DurationVar(&testOptions.startupTimeout, "startup-timeout", 10*time.Minute, "Sliver daemon startup timeout")
	flag.DurationVar(&testOptions.connectTimeout, "connect-timeout", 5*time.Minute, "implant connection timeout")
	flag.DurationVar(&testOptions.commandTimeout, "command-timeout", 2*time.Minute, "individual implant command timeout")
	flag.DurationVar(&testOptions.beaconInterval, "beacon-interval", 10*time.Second, "beacon callback interval")
	flag.IntVar(&testOptions.sgnSamples, "shellcode-sgn-samples", shellcodecoverage.MinimumSGNSamples, "required SGN execution samples per shellcode matrix cell")
	flag.Int64Var(&testOptions.socksFuzzSeed, "socks-fuzz-seed", socksE2EMutationSeed, "deterministic seed for the live SOCKS5 malformed-input corpus")
	flag.IntVar(&testOptions.socksFuzzCases, "socks-fuzz-cases", socksE2EMalformedCases, "number of live SOCKS5 malformed-input cases")
	flag.IntVar(&testOptions.socksFuzzCase, "socks-fuzz-replay-case", -1, "replay one zero-based SOCKS5 malformed-input case (-1 runs the corpus)")
	flag.StringVar(&testOptions.tunnelHTTPURL, "tunnel-target-http-url", "", "optional non-secret HTTP(S) URL fetched directly and through both tunnel features")
	flag.IntVar(&testOptions.tunnelHTTPURLFD, "tunnel-target-http-url-fd", -1, "optional descriptor containing a runtime-only HTTP(S) target URL")
	flag.StringVar(&testOptions.tunnelRDPAddr, "tunnel-target-rdp-address", "", "optional real RDP host:port negotiated through both tunnel features")
	flag.IntVar(&testOptions.rdpCredentialsFD, "tunnel-rdp-credentials-fd", -1, "optional descriptor containing one runtime-only RDP credential and certificate-pin JSON object")
	flag.StringVar(&testOptions.tunnelAcceptanceProfile, "tunnel-acceptance-profile", tunnelAcceptanceProfileBase, "focused tunnel evidence profile (base or proxmox)")
	flag.BoolVar(&testOptions.implantDebug, "implant-debug", false, "generate debug implants and capture their transport logs")
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
	os.Exit(m.Run())
}

func TestComprehensiveE2E(t *testing.T) {
	// Invalidate the machine-readable PASS marker before any validation or
	// startup work. A failed preflight must never leave a prior successful run
	// looking current to a manual runner or future CI wrapper.
	if strings.TrimSpace(testOptions.suiteScope) == suiteScopePortfwdSocks5 && strings.TrimSpace(testOptions.resultsDir) != "" {
		if err := invalidateTunnelReportArtifacts(testOptions.resultsDir); err != nil {
			t.Fatalf("invalidate prior portfwd/SOCKS5 E2E report: %v", err)
		}
	}
	if testOptions.serverPath == "" {
		t.Skip("comprehensive E2E requires -server")
	}

	recordCommandCoverage := strings.TrimSpace(testOptions.suiteScope) == suiteScopeComprehensive
	suite, err := newSuite(t, testOptions, recordCommandCoverage)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		// Cleanup can uncover listener, tunnel-loop, or process failures. Run it
		// before stamping the report so its top-level result reflects teardown.
		suite.close()
		if err := suite.writeCoverage(); err != nil {
			t.Errorf("%v", err)
		}
	}()

	if err := suite.run(); err != nil {
		t.Fatal(err)
	}
}
