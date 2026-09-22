package e2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTunnelReportTestRecorder(opts options, authenticationActive bool) *tunnelReportRecorder {
	recorder := newTunnelReportRecorder(opts, authenticationActive)
	recorder.setExecutableProvenance(tunnelReportProvenance{
		ServerSHA256: strings.Repeat("a", 64),
		DriverSHA256: strings.Repeat("b", 64),
	})
	recorder.setServerVersion("0123456789abcdef", true)
	return recorder
}

func populateCompleteTunnelReportTestDocument(document *tunnelReportDocument) {
	for _, transport := range document.Transports {
		for feature, scenarios := range tunnelReportRequiredScenarios {
			for _, scenario := range scenarios {
				document.Observations = append(document.Observations, tunnelReportObservation{
					Transport: transport,
					Feature:   feature,
					Scenario:  scenario,
					Status:    "pass",
				})
			}
			if document.External.HTTPConfigured {
				document.Observations = append(document.Observations, tunnelReportObservation{
					Transport: transport,
					Feature:   feature,
					Scenario:  "real-http-curl",
					Status:    "pass",
				})
			}
			if document.External.RDPAddress != "" {
				document.Observations = append(document.Observations, tunnelReportObservation{
					Transport: transport,
					Feature:   feature,
					Scenario:  "rdp-negotiation",
					Status:    "pass",
				})
			}
		}
		for _, route := range []string{"port-forward", "SOCKS5"} {
			if document.External.HTTPConfigured {
				document.Evidence = append(document.Evidence, tunnelReportEvidence{
					Transport: transport,
					Route:     route,
					Kind:      "http-body",
					Bytes:     149,
					SHA256:    strings.Repeat("c", 64),
				})
			}
			if document.External.RDPAddress != "" {
				document.Evidence = append(document.Evidence, tunnelReportEvidence{
					Transport:        transport,
					Route:            route,
					Kind:             "rdp-negotiation",
					Bytes:            len(rdpNegotiationRequest),
					NegotiationType:  0x02,
					SelectedProtocol: 0x02,
				})
				if document.External.RDPAuthenticationActive {
					document.Evidence = append(document.Evidence,
						tunnelReportEvidence{Transport: transport, Route: route, Kind: "rdp-nla"},
						tunnelReportEvidence{
							Transport:  transport,
							Route:      route,
							Kind:       "rdp-desktop",
							DurationMS: rdpDesktopAcceptanceDuration.Milliseconds(),
						},
					)
				}
			}
		}
	}
}

func TestTunnelReportEmptyCollectionsEncodeAsArrays(t *testing.T) {
	recorder := newTunnelReportTestRecorder(options{
		targetOS:       "linux",
		targetArch:     "amd64",
		transports:     []string{"mtls"},
		socksFuzzCases: 1,
		socksFuzzCase:  -1,
	}, false)

	assertEmptyArrays := func(report []byte, fields ...string) {
		t.Helper()
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(report, &raw); err != nil {
			t.Fatalf("decode tunnel report: %v", err)
		}
		for _, field := range fields {
			if !bytes.Equal(raw[field], []byte("[]")) {
				t.Errorf("tunnel report %s = %s, want []", field, raw[field])
			}
		}
	}

	initial, err := json.Marshal(recorder.document)
	if err != nil {
		t.Fatal(err)
	}
	assertEmptyArrays(initial, "observations", "evidence")

	for feature, scenarios := range tunnelReportRequiredScenarios {
		for _, scenario := range scenarios {
			recorder.recordScenario(tunnelReportObservation{
				Transport: "mtls",
				Feature:   feature,
				Scenario:  scenario,
				Status:    "pass",
			})
		}
	}
	paths, err := recorder.write(t.TempDir(), true)
	if err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(paths.JSON)
	if err != nil {
		t.Fatal(err)
	}
	assertEmptyArrays(written, "evidence")
}

//nolint:gocyclo // One artifact round trip verifies schema, redaction, completeness, and deterministic rendering.
func TestTunnelReportWritesStructuredCredentialFreeEvidence(t *testing.T) {
	httpPathSecret := "private-range-path-token"
	opts := options{
		targetOS:       "linux",
		targetArch:     "amd64",
		transports:     []string{"mtls"},
		socksFuzzSeed:  0x50c55eed,
		socksFuzzCases: 64,
		socksFuzzCase:  -1,
		tunnelHTTPURL:  "http://example.test/" + httpPathSecret,
	}
	recorder := newTunnelReportTestRecorder(opts, false)
	s := &suite{
		t:            t,
		tunnelReport: recorder,
		rdpCredentials: &rdpCredentials{
			Domain:   "SECRET-DOMAIN",
			Username: "SECRET-USER",
			Password: "SECRET-PASSWORD",
		},
	}
	for feature, scenarios := range tunnelReportRequiredScenarios {
		for _, scenario := range scenarios {
			var scenarioErr error
			if feature == "portfwd" && scenario == "binary-boundaries" {
				scenarioErr = errors.New("SECRET-DOMAIN SECRET-USER SECRET-PASSWORD failed")
			}
			s.recordTunnelScenario("mtls", feature, scenario, 1500*time.Millisecond, scenarioErr)
		}
		s.recordTunnelScenario("mtls", feature, "real-http-curl", time.Second, nil)
	}
	recorder.recordEvidence(tunnelReportEvidence{
		Transport: "mtls",
		Route:     "port-forward",
		Kind:      "http-body",
		Bytes:     149,
		SHA256:    strings.Repeat("a", 64),
	})
	recorder.recordEvidence(tunnelReportEvidence{
		Transport: "mtls",
		Route:     "SOCKS5",
		Kind:      "http-body",
		Bytes:     149,
		SHA256:    strings.Repeat("a", 64),
	})

	paths, err := recorder.write(t.TempDir(), true)
	if err != nil {
		t.Fatal(err)
	}
	jsonBytes, err := os.ReadFile(paths.JSON)
	if err != nil {
		t.Fatal(err)
	}
	markdownBytes, err := os.ReadFile(paths.Markdown)
	if err != nil {
		t.Fatal(err)
	}
	combined := string(jsonBytes) + string(markdownBytes)
	for _, secret := range []string{"SECRET-DOMAIN", "SECRET-USER", "SECRET-PASSWORD"} {
		if strings.Contains(combined, secret) {
			t.Fatalf("tunnel report serialized credential value %q", secret)
		}
	}
	if strings.Contains(combined, httpPathSecret) || strings.Contains(combined, opts.tunnelHTTPURL) {
		t.Fatal("tunnel report serialized the external HTTP target path")
	}

	var document tunnelReportDocument
	if err := json.Unmarshal(jsonBytes, &document); err != nil {
		t.Fatalf("decode tunnel JSON report: %v", err)
	}
	if document.Passed {
		t.Fatal("report with a failed scenario was marked passed")
	}
	if !document.External.HTTPConfigured || document.External.HTTPOrigin != "http://example.test" {
		t.Fatalf("sanitized external HTTP report metadata = %+v", document.External)
	}
	if document.AcceptanceProfile != tunnelAcceptanceProfileBase || document.Acceptance {
		t.Fatalf("base report acceptance metadata = profile %q acceptance=%t", document.AcceptanceProfile, document.Acceptance)
	}
	if document.Provenance.ServerSHA256 != strings.Repeat("a", 64) || document.Provenance.DriverSHA256 != strings.Repeat("b", 64) ||
		document.Provenance.ServerCommit != "0123456789abcdef" || !document.Provenance.ServerDirty {
		t.Fatalf("report provenance = %+v", document.Provenance)
	}
	failures := 0
	for _, observation := range document.Observations {
		if observation.Status == "fail" {
			failures++
		}
	}
	if failures != 1 {
		t.Fatalf("report failure observations = %d, want one", failures)
	}
	if len(document.Evidence) != 2 {
		t.Fatalf("report evidence = %+v, want two HTTP digest rows", document.Evidence)
	}
	if !strings.Contains(string(markdownBytes), "Portfwd and SOCKS5 E2E report") {
		t.Fatal("Markdown report is missing its title")
	}
	if !strings.Contains(string(markdownBytes), "NO — base diagnostic profile") {
		t.Fatal("base Markdown report is not clearly labeled non-acceptance")
	}
}

func TestTunnelReportCompletenessEnforcesProxmoxAcceptanceContract(t *testing.T) {
	recorder := newTunnelReportTestRecorder(options{
		targetOS:                "linux",
		targetArch:              "amd64",
		transports:              append([]string(nil), transportOrder...),
		socksFuzzCases:          1,
		socksFuzzCase:           -1,
		tunnelHTTPURL:           "https://range.example.test/resource",
		tunnelRDPAddr:           "windows.example.test:3389",
		tunnelAcceptanceProfile: tunnelAcceptanceProfileProxmox,
	}, true)
	document := recorder.document
	if document.AcceptanceProfile != tunnelAcceptanceProfileProxmox {
		t.Fatal("proxmox report did not persist its acceptance profile")
	}
	if err := validateTunnelReportCompleteness(document); err == nil {
		t.Fatal("empty proxmox report unexpectedly passed completeness")
	} else {
		for _, unexpected := range []string{
			"proxmox acceptance requires exactly transports",
			"proxmox acceptance requires external HTTP",
			"proxmox acceptance requires an RDP target",
			"proxmox acceptance requires authenticated certificate-pinned RDP",
		} {
			if strings.Contains(err.Error(), unexpected) {
				t.Fatalf("valid proxmox profile contract produced %q: %v", unexpected, err)
			}
		}
	}

	document.Transports = []string{"mtls"}
	document.External.HTTPConfigured = false
	document.External.HTTPOrigin = ""
	document.External.RDPAddress = ""
	document.External.RDPAuthenticationActive = false
	err := validateTunnelReportCompleteness(document)
	for _, want := range []string{
		"proxmox acceptance requires exactly transports",
		"proxmox acceptance requires external HTTP",
		"proxmox acceptance requires an RDP target",
		"proxmox acceptance requires authenticated certificate-pinned RDP",
	} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("incomplete proxmox report error = %v, want %q", err, want)
		}
	}
}

func TestTunnelReportMarksOnlyCompleteProxmoxProfileAsAcceptance(t *testing.T) {
	recorder := newTunnelReportTestRecorder(options{
		targetOS:                "linux",
		targetArch:              "amd64",
		transports:              append([]string(nil), transportOrder...),
		socksFuzzCases:          1,
		socksFuzzCase:           -1,
		tunnelHTTPURL:           "https://range.example.test/resource",
		tunnelRDPAddr:           "windows.example.test:3389",
		tunnelAcceptanceProfile: tunnelAcceptanceProfileProxmox,
	}, true)
	populateCompleteTunnelReportTestDocument(&recorder.document)
	paths, err := recorder.write(t.TempDir(), true)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(paths.JSON)
	if err != nil {
		t.Fatal(err)
	}
	var document tunnelReportDocument
	if err := json.Unmarshal(payload, &document); err != nil {
		t.Fatal(err)
	}
	if !document.Passed || !document.Acceptance || document.AcceptanceProfile != tunnelAcceptanceProfileProxmox {
		t.Fatalf("proxmox acceptance result = passed=%t acceptance=%t profile=%q", document.Passed, document.Acceptance, document.AcceptanceProfile)
	}
}

func TestTunnelReportCompletenessRejectsMissingOrZeroExecutableHashes(t *testing.T) {
	document := newTunnelReportTestRecorder(options{
		targetOS:       "linux",
		targetArch:     "amd64",
		transports:     []string{"mtls"},
		socksFuzzCases: 1,
		socksFuzzCase:  -1,
	}, false).document
	document.Provenance.ServerSHA256 = ""
	document.Provenance.DriverSHA256 = strings.Repeat("0", 64)
	err := validateTunnelReportCompleteness(document)
	for _, want := range []string{"server executable SHA-256", "driver executable SHA-256"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("invalid provenance error = %v, want %q", err, want)
		}
	}
}

func TestTunnelReportMarkdownDistinguishesConfiguredTargetsWithoutEvidence(t *testing.T) {
	document := newTunnelReportTestRecorder(options{
		targetOS:       "linux",
		targetArch:     "amd64",
		transports:     []string{"mtls"},
		socksFuzzCases: 1,
		socksFuzzCase:  -1,
		tunnelHTTPURL:  "https://range.example.test/resource",
	}, false).document
	markdown := renderTunnelReportMarkdown(document)
	if !strings.Contains(markdown, "configured, but no successful evidence was recorded") {
		t.Fatalf("configured target Markdown = %q", markdown)
	}
	if strings.Contains(markdown, "No external traffic targets were configured") {
		t.Fatal("configured target Markdown claims no target was configured")
	}
}

func TestTunnelReportCompletenessRequiresSharedHTTPDigest(t *testing.T) {
	recorder := newTunnelReportTestRecorder(options{
		targetOS:       "linux",
		targetArch:     "amd64",
		transports:     []string{"mtls"},
		socksFuzzCases: 1,
		socksFuzzCase:  -1,
		tunnelHTTPURL:  "http://example.test/range-token",
	}, false)
	document := recorder.document
	for feature, scenarios := range tunnelReportRequiredScenarios {
		for _, scenario := range append(append([]string(nil), scenarios...), "real-http-curl") {
			document.Observations = append(document.Observations, tunnelReportObservation{
				Transport: "mtls",
				Feature:   feature,
				Scenario:  scenario,
				Status:    "pass",
			})
		}
	}
	document.Evidence = append(document.Evidence, []tunnelReportEvidence{
		{Transport: "mtls", Route: "port-forward", Kind: "http-body", Bytes: 149, SHA256: strings.Repeat("a", 64)},
		{Transport: "mtls", Route: "SOCKS5", Kind: "http-body", Bytes: 149, SHA256: strings.Repeat("b", 64)},
	}...)
	if err := validateTunnelReportCompleteness(document); err == nil || !strings.Contains(err.Error(), "disagrees with shared baseline") {
		t.Fatalf("mismatched HTTP evidence error = %v", err)
	}
	document.Evidence[1].SHA256 = document.Evidence[0].SHA256
	if err := validateTunnelReportCompleteness(document); err != nil {
		t.Fatalf("matching HTTP evidence: %v", err)
	}
}

func TestInvalidateTunnelReportArtifactsRemovesPriorPassPair(t *testing.T) {
	directory := t.TempDir()
	jsonPath := filepath.Join(directory, "portfwd-socks5-e2e.json")
	markdownPath := filepath.Join(directory, "portfwd-socks5-e2e.md")
	for _, path := range []string{jsonPath, markdownPath} {
		if err := os.WriteFile(path, []byte("stale PASS"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := invalidateTunnelReportArtifacts(directory); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{jsonPath, markdownPath} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("stale report %s still exists: %v", filepath.Base(path), err)
		}
	}
	// Repeated invalidation and a not-yet-created result directory are both
	// valid startup states.
	if err := invalidateTunnelReportArtifacts(directory); err != nil {
		t.Fatal(err)
	}
	if err := invalidateTunnelReportArtifacts(filepath.Join(directory, "not-created")); err != nil {
		t.Fatal(err)
	}
}

func TestTunnelReportCompletenessFailureWritesFailingArtifact(t *testing.T) {
	recorder := newTunnelReportTestRecorder(options{
		targetOS:       "linux",
		targetArch:     "amd64",
		transports:     []string{"mtls"},
		socksFuzzCases: 1,
		socksFuzzCase:  -1,
	}, false)
	paths, err := recorder.write(t.TempDir(), true)
	if err == nil || !strings.Contains(err.Error(), "missing scenario") {
		t.Fatalf("incomplete report error = %v, want missing scenario", err)
	}
	jsonBytes, readErr := os.ReadFile(paths.JSON)
	if readErr != nil {
		t.Fatal(readErr)
	}
	var document tunnelReportDocument
	if err := json.Unmarshal(jsonBytes, &document); err != nil {
		t.Fatal(err)
	}
	if document.Passed {
		t.Fatal("incomplete report artifact was marked passed")
	}
	found := false
	for _, observation := range document.Observations {
		if observation.Feature == "suite" && observation.Scenario == "report-completeness" && observation.Status == "fail" {
			found = true
		}
	}
	if !found {
		t.Fatal("incomplete report artifact omitted completeness failure observation")
	}
}

func TestTunnelReportWriteFailureCannotLeaveStalePassingJSON(t *testing.T) {
	directory := t.TempDir()
	jsonPath := filepath.Join(directory, "portfwd-socks5-e2e.json")
	markdownPath := filepath.Join(directory, "portfwd-socks5-e2e.md")
	if err := os.WriteFile(jsonPath, []byte("{\"passed\":true}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(markdownPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(markdownPath, "block-removal"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	recorder := newTunnelReportTestRecorder(options{
		targetOS:       "linux",
		targetArch:     "amd64",
		transports:     []string{"mtls"},
		socksFuzzCases: 1,
		socksFuzzCase:  -1,
	}, false)
	if _, err := recorder.write(directory, true); err == nil {
		t.Fatal("report write unexpectedly replaced a non-empty Markdown directory")
	}
	if _, err := os.Stat(jsonPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale passing JSON remained after pair failure: %v", err)
	}
}

func TestTunnelReportMarshalFailureCannotLeaveStalePassingJSON(t *testing.T) {
	directory := t.TempDir()
	jsonPath := filepath.Join(directory, "portfwd-socks5-e2e.json")
	if err := os.WriteFile(jsonPath, []byte("{\"passed\":true}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	recorder := newTunnelReportTestRecorder(options{
		targetOS:       "linux",
		targetArch:     "amd64",
		transports:     []string{"mtls"},
		socksFuzzCases: 1,
		socksFuzzCase:  -1,
	}, false)
	recorder.document.StartedAt = time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC)

	if _, err := recorder.write(directory, true); err == nil || !strings.Contains(err.Error(), "marshal tunnel E2E JSON report") {
		t.Fatalf("write error = %v, want JSON marshal failure", err)
	}
	if _, err := os.Stat(jsonPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale passing JSON remained after marshal failure: %v", err)
	}
}

func TestTunnelReportCompletenessRequiresAuthenticatedRDPEvidence(t *testing.T) {
	recorder := newTunnelReportTestRecorder(options{
		targetOS:       "linux",
		targetArch:     "amd64",
		transports:     []string{"mtls"},
		socksFuzzCases: 1,
		socksFuzzCase:  -1,
		tunnelRDPAddr:  "10.13.37.22:3389",
	}, true)
	document := recorder.document
	for feature, scenarios := range tunnelReportRequiredScenarios {
		for _, scenario := range scenarios {
			document.Observations = append(document.Observations, tunnelReportObservation{
				Transport: "mtls",
				Feature:   feature,
				Scenario:  scenario,
				Status:    "pass",
			})
		}
		document.Observations = append(document.Observations, tunnelReportObservation{
			Transport: "mtls",
			Feature:   feature,
			Scenario:  "rdp-negotiation",
			Status:    "pass",
		})
	}
	for _, route := range []string{"port-forward", "SOCKS5"} {
		for _, kind := range []string{"rdp-negotiation", "rdp-nla", "rdp-desktop"} {
			document.Evidence = append(document.Evidence, tunnelReportEvidence{
				Transport: "mtls",
				Route:     route,
				Kind:      kind,
			})
		}
	}
	if err := validateTunnelReportCompleteness(document); err == nil || !strings.Contains(err.Error(), "invalid RDP negotiation") {
		t.Fatalf("zero-valued authenticated RDP evidence error = %v", err)
	}
	for index := range document.Evidence {
		switch document.Evidence[index].Kind {
		case "rdp-negotiation":
			document.Evidence[index].Bytes = len(rdpNegotiationRequest)
			document.Evidence[index].NegotiationType = 0x02
			document.Evidence[index].SelectedProtocol = 0x02
		case "rdp-desktop":
			document.Evidence[index].DurationMS = rdpDesktopAcceptanceDuration.Milliseconds()
		}
	}
	if err := validateTunnelReportCompleteness(document); err != nil {
		t.Fatalf("complete authenticated RDP report: %v", err)
	}
	document.Evidence = document.Evidence[:len(document.Evidence)-1]
	if err := validateTunnelReportCompleteness(document); err == nil || !strings.Contains(err.Error(), "mtls/SOCKS5/rdp-desktop") {
		t.Fatalf("missing authenticated RDP evidence error = %v", err)
	}
}
