package e2e

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const tunnelReportSchemaVersion = 3

var tunnelReportRequiredScenarios = map[string][]string{
	"portfwd": {
		"active-session-disconnect",
		"binary-boundaries",
		"http-curl",
		"idle-resume",
		"implant-dial-failure-recovery",
		"proxy-isolation",
		"sequential-concurrent",
		"stop-new-dial",
		"sustained-full-duplex",
	},
	"socks5": {
		"active-session-disconnect",
		"auth-no-auth-proxy-isolation",
		"authenticated-success-and-wrong-password",
		"boundary-and-random-binary",
		"go-http-and-curl",
		"idle-resume",
		"malformed-recovery-and-ping",
		"no-auth-ipv4-and-hostname",
		"sequential-and-concurrent",
		"stop-and-restart",
		"sustained-full-duplex",
		"two-proxy-stop-isolation",
	},
}

type tunnelReportTarget struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

type tunnelReportFuzz struct {
	Seed       int64 `json:"seed"`
	Cases      int   `json:"cases"`
	ReplayCase int   `json:"replay_case"`
}

type tunnelReportExternal struct {
	HTTPConfigured          bool   `json:"http_configured"`
	HTTPOrigin              string `json:"http_origin,omitempty"`
	RDPAddress              string `json:"rdp_address,omitempty"`
	RDPAuthenticationActive bool   `json:"rdp_authentication_active"`
}

type tunnelReportProvenance struct {
	ServerSHA256 string `json:"server_sha256"`
	DriverSHA256 string `json:"driver_sha256"`
	ServerCommit string `json:"server_commit"`
	ServerDirty  bool   `json:"server_dirty"`
}

type tunnelReportObservation struct {
	Transport  string `json:"transport"`
	Feature    string `json:"feature"`
	Scenario   string `json:"scenario"`
	Status     string `json:"status"`
	DurationMS int64  `json:"duration_ms"`
	Detail     string `json:"detail,omitempty"`
}

type tunnelReportEvidence struct {
	Transport        string `json:"transport"`
	Route            string `json:"route"`
	Kind             string `json:"kind"`
	Bytes            int    `json:"bytes,omitempty"`
	SHA256           string `json:"sha256,omitempty"`
	NegotiationType  uint32 `json:"negotiation_type,omitempty"`
	SelectedProtocol uint32 `json:"selected_protocol,omitempty"`
	DurationMS       int64  `json:"duration_ms,omitempty"`
}

type tunnelReportDocument struct {
	SchemaVersion     int                       `json:"schema_version"`
	Scope             string                    `json:"scope"`
	AcceptanceProfile string                    `json:"acceptance_profile"`
	Acceptance        bool                      `json:"acceptance"`
	StartedAt         time.Time                 `json:"started_at"`
	FinishedAt        time.Time                 `json:"finished_at"`
	Passed            bool                      `json:"passed"`
	Target            tunnelReportTarget        `json:"target"`
	Transports        []string                  `json:"transports"`
	Fuzz              tunnelReportFuzz          `json:"fuzz"`
	External          tunnelReportExternal      `json:"external"`
	Provenance        tunnelReportProvenance    `json:"provenance"`
	Observations      []tunnelReportObservation `json:"observations"`
	Evidence          []tunnelReportEvidence    `json:"evidence"`
}

type tunnelReportPaths struct {
	JSON     string
	Markdown string
}

type tunnelReportRecorder struct {
	mutex    sync.Mutex
	document tunnelReportDocument
}

func newTunnelReportRecorder(opts options, authenticationActive bool) *tunnelReportRecorder {
	profile := opts.tunnelAcceptanceProfile
	if profile == "" {
		profile = tunnelAcceptanceProfileBase
	}
	return &tunnelReportRecorder{document: tunnelReportDocument{
		SchemaVersion:     tunnelReportSchemaVersion,
		Scope:             suiteScopePortfwdSocks5,
		AcceptanceProfile: profile,
		Acceptance:        false,
		StartedAt:         time.Now().UTC(),
		Target:            tunnelReportTarget{OS: opts.targetOS, Arch: opts.targetArch},
		Transports:        append([]string(nil), opts.transports...),
		Fuzz: tunnelReportFuzz{
			Seed:       opts.socksFuzzSeed,
			Cases:      opts.socksFuzzCases,
			ReplayCase: opts.socksFuzzCase,
		},
		External: tunnelReportExternal{
			HTTPConfigured:          opts.tunnelHTTPURL != "",
			HTTPOrigin:              tunnelHTTPOrigin(opts.tunnelHTTPURL),
			RDPAddress:              opts.tunnelRDPAddr,
			RDPAuthenticationActive: authenticationActive,
		},
		Observations: make([]tunnelReportObservation, 0),
		Evidence:     make([]tunnelReportEvidence, 0),
	}}
}

func (r *tunnelReportRecorder) setExecutableProvenance(provenance tunnelReportProvenance) {
	if r == nil {
		return
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.document.Provenance.ServerSHA256 = provenance.ServerSHA256
	r.document.Provenance.DriverSHA256 = provenance.DriverSHA256
}

func (r *tunnelReportRecorder) setServerVersion(commit string, dirty bool) {
	if r == nil {
		return
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.document.Provenance.ServerCommit = commit
	r.document.Provenance.ServerDirty = dirty
}

func (r *tunnelReportRecorder) recordScenario(observation tunnelReportObservation) {
	if r == nil {
		return
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.document.Observations = append(r.document.Observations, observation)
}

func (r *tunnelReportRecorder) recordEvidence(evidence tunnelReportEvidence) {
	if r == nil {
		return
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.document.Evidence = append(r.document.Evidence, evidence)
}

//nolint:gocyclo // Atomic two-artifact publication validates and records every failure path explicitly.
func (r *tunnelReportRecorder) write(resultsDir string, passed bool) (tunnelReportPaths, error) {
	paths := tunnelReportPaths{
		JSON:     filepath.Join(resultsDir, "portfwd-socks5-e2e.json"),
		Markdown: filepath.Join(resultsDir, "portfwd-socks5-e2e.md"),
	}
	// The JSON document is the machine-readable PASS marker. Invalidate it
	// before validation, marshaling, or staging can fail.
	if err := removeTunnelReportArtifact(paths.JSON); err != nil {
		return paths, err
	}

	if r == nil {
		return paths, errors.New("write nil tunnel report recorder")
	}
	r.mutex.Lock()
	document := r.document
	document.FinishedAt = time.Now().UTC()
	document.Passed = passed
	document.Transports = append([]string(nil), document.Transports...)
	document.Observations = append([]tunnelReportObservation{}, document.Observations...)
	document.Evidence = append([]tunnelReportEvidence{}, document.Evidence...)
	r.mutex.Unlock()
	completenessErr := validateTunnelReportCompleteness(document)
	if completenessErr != nil {
		document.Passed = false
		document.Observations = append(document.Observations, tunnelReportObservation{
			Transport: "*",
			Feature:   "suite",
			Scenario:  "report-completeness",
			Status:    "fail",
			Detail:    completenessErr.Error(),
		})
	}
	for _, observation := range document.Observations {
		if observation.Status != "pass" {
			document.Passed = false
			break
		}
	}
	document.Acceptance = document.Passed && document.AcceptanceProfile == tunnelAcceptanceProfileProxmox

	sort.SliceStable(document.Observations, func(left, right int) bool {
		a := document.Observations[left]
		b := document.Observations[right]
		if a.Transport != b.Transport {
			return a.Transport < b.Transport
		}
		if a.Feature != b.Feature {
			return a.Feature < b.Feature
		}
		return a.Scenario < b.Scenario
	})
	sort.SliceStable(document.Evidence, func(left, right int) bool {
		a := document.Evidence[left]
		b := document.Evidence[right]
		if a.Transport != b.Transport {
			return a.Transport < b.Transport
		}
		if a.Route != b.Route {
			return a.Route < b.Route
		}
		return a.Kind < b.Kind
	})

	jsonBytes, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return tunnelReportPaths{}, fmt.Errorf("marshal tunnel E2E JSON report: %w", err)
	}
	jsonBytes = append(jsonBytes, '\n')
	markdownBytes := []byte(renderTunnelReportMarkdown(document))
	jsonTemp, err := writeTunnelReportTemp(resultsDir, ".portfwd-socks5-e2e-json-*", jsonBytes)
	if err != nil {
		return tunnelReportPaths{}, fmt.Errorf("stage tunnel E2E JSON report: %w", err)
	}
	defer func() { _ = os.Remove(jsonTemp) }()
	markdownTemp, err := writeTunnelReportTemp(resultsDir, ".portfwd-socks5-e2e-markdown-*", markdownBytes)
	if err != nil {
		return tunnelReportPaths{}, fmt.Errorf("stage tunnel E2E Markdown report: %w", err)
	}
	defer func() { _ = os.Remove(markdownTemp) }()
	// Publish JSON only after Markdown succeeds, so an interrupted pair update
	// can never leave a plausible PASS JSON beside a failed write.
	if err := removeTunnelReportArtifact(paths.Markdown); err != nil {
		return tunnelReportPaths{}, err
	}
	if err := os.Rename(markdownTemp, paths.Markdown); err != nil {
		return tunnelReportPaths{}, fmt.Errorf("publish tunnel E2E Markdown report: %w", err)
	}
	if err := os.Rename(jsonTemp, paths.JSON); err != nil {
		_ = os.Remove(paths.Markdown)
		return tunnelReportPaths{}, fmt.Errorf("publish tunnel E2E JSON report: %w", err)
	}
	if completenessErr != nil {
		return paths, completenessErr
	}
	return paths, nil
}

func writeTunnelReportTemp(directory string, pattern string, data []byte) (string, error) {
	file, err := os.CreateTemp(directory, pattern)
	if err != nil {
		return "", err
	}
	path := file.Name()
	fail := func(cause error) (string, error) {
		_ = file.Close()
		_ = os.Remove(path)
		return "", cause
	}
	if err := file.Chmod(0o600); err != nil {
		return fail(err)
	}
	if _, err := file.Write(data); err != nil {
		return fail(err)
	}
	if err := file.Sync(); err != nil {
		return fail(err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

func removeTunnelReportArtifact(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove stale tunnel E2E report %s: %w", filepath.Base(path), err)
	}
	return nil
}

func invalidateTunnelReportArtifacts(resultsDir string) error {
	if strings.TrimSpace(resultsDir) == "" {
		return nil
	}
	// Remove JSON first because it is the machine-readable PASS marker. If the
	// Markdown removal fails, automation still cannot consume a stale success.
	return errors.Join(
		removeTunnelReportArtifact(filepath.Join(resultsDir, "portfwd-socks5-e2e.json")),
		removeTunnelReportArtifact(filepath.Join(resultsDir, "portfwd-socks5-e2e.md")),
	)
}

//nolint:gocyclo // Completeness is one contract spanning scenario, fuzz, HTTP, and RDP evidence matrices.
func validateTunnelReportCompleteness(document tunnelReportDocument) error {
	var invalid []string
	if document.Scope != suiteScopePortfwdSocks5 {
		invalid = append(invalid, fmt.Sprintf("scope is %q, want %q", document.Scope, suiteScopePortfwdSocks5))
	}
	profile, profileErr := normalizeTunnelAcceptanceProfile(document.AcceptanceProfile)
	if profileErr != nil || profile != document.AcceptanceProfile {
		invalid = append(invalid, fmt.Sprintf("acceptance profile is invalid: %q", document.AcceptanceProfile))
	}
	wantAcceptance := profileErr == nil && profile == tunnelAcceptanceProfileProxmox
	if document.Acceptance && !wantAcceptance {
		invalid = append(invalid, fmt.Sprintf("base profile cannot carry an acceptance marker for profile %q", document.AcceptanceProfile))
	}
	if !validTunnelReportSHA256(document.Provenance.ServerSHA256) {
		invalid = append(invalid, "server executable SHA-256 is missing, zero, or malformed")
	}
	if !validTunnelReportSHA256(document.Provenance.DriverSHA256) {
		invalid = append(invalid, "driver executable SHA-256 is missing, zero, or malformed")
	}
	if wantAcceptance {
		if len(document.Transports) != len(transportOrder) {
			invalid = append(invalid, "proxmox acceptance requires exactly transports mtls,wg,http")
		} else {
			for index, transport := range transportOrder {
				if document.Transports[index] != transport {
					invalid = append(invalid, "proxmox acceptance requires exactly transports mtls,wg,http")
					break
				}
			}
		}
		if !document.External.HTTPConfigured {
			invalid = append(invalid, "proxmox acceptance requires external HTTP traffic")
		}
		if document.External.RDPAddress == "" {
			invalid = append(invalid, "proxmox acceptance requires an RDP target")
		}
		if !document.External.RDPAuthenticationActive {
			invalid = append(invalid, "proxmox acceptance requires authenticated certificate-pinned RDP")
		}
	}
	if document.External.HTTPConfigured {
		parsed, _, err := tunnelHTTPDestination(document.External.HTTPOrigin)
		if err != nil || parsed.Path != "" || parsed.RawPath != "" {
			invalid = append(invalid, "external HTTP origin is missing or contains non-origin URL components")
		}
	} else if document.External.HTTPOrigin != "" {
		invalid = append(invalid, "external HTTP origin is present while HTTP traffic is not configured")
	}

	expectedObservations := map[string]struct{}{}
	for _, transport := range document.Transports {
		for feature, scenarios := range tunnelReportRequiredScenarios {
			for _, scenario := range scenarios {
				expectedObservations[tunnelReportKey(transport, feature, scenario)] = struct{}{}
			}
			if document.External.HTTPConfigured {
				expectedObservations[tunnelReportKey(transport, feature, "real-http-curl")] = struct{}{}
			}
			if document.External.RDPAddress != "" {
				expectedObservations[tunnelReportKey(transport, feature, "rdp-negotiation")] = struct{}{}
			}
		}
	}
	seenObservations := map[string]int{}
	for _, observation := range document.Observations {
		key := tunnelReportKey(observation.Transport, observation.Feature, observation.Scenario)
		seenObservations[key]++
		if _, ok := expectedObservations[key]; !ok {
			invalid = append(invalid, "unexpected scenario "+key)
		}
		if observation.Status != "pass" && observation.Status != "fail" {
			invalid = append(invalid, fmt.Sprintf("scenario %s has invalid status %q", key, observation.Status))
		}
	}
	for key := range expectedObservations {
		switch seenObservations[key] {
		case 0:
			invalid = append(invalid, "missing scenario "+key)
		case 1:
		default:
			invalid = append(invalid, fmt.Sprintf("duplicate scenario %s count=%d", key, seenObservations[key]))
		}
	}

	expectedEvidence := map[string]struct{}{}
	for _, transport := range document.Transports {
		for _, route := range []string{"port-forward", "SOCKS5"} {
			if document.External.HTTPConfigured {
				expectedEvidence[tunnelReportKey(transport, route, "http-body")] = struct{}{}
			}
			if document.External.RDPAddress != "" {
				expectedEvidence[tunnelReportKey(transport, route, "rdp-negotiation")] = struct{}{}
				if document.External.RDPAuthenticationActive {
					expectedEvidence[tunnelReportKey(transport, route, "rdp-nla")] = struct{}{}
					expectedEvidence[tunnelReportKey(transport, route, "rdp-desktop")] = struct{}{}
				}
			}
		}
	}
	seenEvidence := map[string]int{}
	var expectedHTTPBytes int
	var expectedHTTPDigest string
	for _, evidence := range document.Evidence {
		key := tunnelReportKey(evidence.Transport, evidence.Route, evidence.Kind)
		seenEvidence[key]++
		if _, ok := expectedEvidence[key]; !ok {
			invalid = append(invalid, "unexpected evidence "+key)
			continue
		}
		switch evidence.Kind {
		case "http-body":
			digest, err := hex.DecodeString(evidence.SHA256)
			if evidence.Bytes <= 0 || err != nil || len(digest) != sha256.Size || evidence.SHA256 != strings.ToLower(evidence.SHA256) {
				invalid = append(invalid, fmt.Sprintf("evidence %s has invalid HTTP body bytes=%d sha256=%q", key, evidence.Bytes, evidence.SHA256))
			} else if expectedHTTPDigest == "" {
				expectedHTTPBytes = evidence.Bytes
				expectedHTTPDigest = evidence.SHA256
			} else if evidence.Bytes != expectedHTTPBytes || evidence.SHA256 != expectedHTTPDigest {
				invalid = append(invalid, fmt.Sprintf(
					"evidence %s HTTP body bytes=%d sha256=%s disagrees with shared baseline bytes=%d sha256=%s",
					key, evidence.Bytes, evidence.SHA256, expectedHTTPBytes, expectedHTTPDigest,
				))
			}
		case "rdp-negotiation":
			if evidence.Bytes < len(rdpNegotiationRequest) || evidence.NegotiationType != 0x02 ||
				(evidence.SelectedProtocol != 0x01 && evidence.SelectedProtocol != 0x02) {
				invalid = append(invalid, fmt.Sprintf(
					"evidence %s has invalid RDP negotiation bytes=%d type=%#x protocol=%#x",
					key, evidence.Bytes, evidence.NegotiationType, evidence.SelectedProtocol,
				))
			}
		case "rdp-desktop":
			if evidence.DurationMS < rdpDesktopAcceptanceDuration.Milliseconds() {
				invalid = append(invalid, fmt.Sprintf(
					"evidence %s has desktop duration %dms below %dms",
					key, evidence.DurationMS, rdpDesktopAcceptanceDuration.Milliseconds(),
				))
			}
		}
	}
	for key := range expectedEvidence {
		switch seenEvidence[key] {
		case 0:
			invalid = append(invalid, "missing evidence "+key)
		case 1:
		default:
			invalid = append(invalid, fmt.Sprintf("duplicate evidence %s count=%d", key, seenEvidence[key]))
		}
	}
	if len(invalid) == 0 {
		return nil
	}
	sort.Strings(invalid)
	return fmt.Errorf("portfwd/SOCKS5 report is incomplete: %s", strings.Join(invalid, "; "))
}

func validTunnelReportSHA256(value string) bool {
	digest, err := hex.DecodeString(value)
	if err != nil || len(digest) != sha256.Size || value != strings.ToLower(value) {
		return false
	}
	for _, octet := range digest {
		if octet != 0 {
			return true
		}
	}
	return false
}

func tunnelReportKey(first string, second string, third string) string {
	return first + "/" + second + "/" + third
}

func renderTunnelReportMarkdown(document tunnelReportDocument) string {
	var report strings.Builder
	fmt.Fprintf(&report, "# Portfwd and SOCKS5 E2E report\n\n")
	fmt.Fprintf(&report, "- Result: %s\n", map[bool]string{true: "PASS", false: "FAIL"}[document.Passed])
	fmt.Fprintf(&report, "- Acceptance profile: `%s`\n", markdownCell(document.AcceptanceProfile))
	if document.Acceptance {
		report.WriteString("- Acceptance result: **PASS — Proxmox acceptance**\n")
	} else if document.AcceptanceProfile == tunnelAcceptanceProfileProxmox {
		report.WriteString("- Acceptance result: **FAIL — Proxmox acceptance criteria not satisfied**\n")
	} else {
		report.WriteString("- Acceptance result: **NO — base diagnostic profile**\n")
	}
	fmt.Fprintf(&report, "- Target: `%s/%s`\n", markdownCell(document.Target.OS), markdownCell(document.Target.Arch))
	fmt.Fprintf(&report, "- Transports: `%s`\n", markdownCell(strings.Join(document.Transports, ",")))
	fmt.Fprintf(&report, "- SOCKS fuzz: seed `%#x`, cases `%d`, replay `%d`\n", document.Fuzz.Seed, document.Fuzz.Cases, document.Fuzz.ReplayCase)
	fmt.Fprintf(&report, "- Server executable SHA-256: `%s`\n", markdownCell(document.Provenance.ServerSHA256))
	fmt.Fprintf(&report, "- Driver executable SHA-256: `%s`\n", markdownCell(document.Provenance.DriverSHA256))
	serverCommit := document.Provenance.ServerCommit
	if serverCommit == "" {
		serverCommit = "unreported"
	}
	fmt.Fprintf(&report, "- RPC server commit: `%s`\n", markdownCell(serverCommit))
	fmt.Fprintf(&report, "- RPC server dirty: `%t`\n", document.Provenance.ServerDirty)
	fmt.Fprintf(&report, "- Started: `%s`\n", document.StartedAt.Format(time.RFC3339Nano))
	fmt.Fprintf(&report, "- Finished: `%s`\n\n", document.FinishedAt.Format(time.RFC3339Nano))

	report.WriteString("## Scenarios\n\n")
	report.WriteString("| Transport | Feature | Scenario | Status | Duration | Detail |\n")
	report.WriteString("|---|---|---|---:|---:|---|\n")
	for _, observation := range document.Observations {
		fmt.Fprintf(
			&report,
			"| %s | %s | %s | %s | %d ms | %s |\n",
			markdownCell(observation.Transport),
			markdownCell(observation.Feature),
			markdownCell(observation.Scenario),
			markdownCell(strings.ToUpper(observation.Status)),
			observation.DurationMS,
			markdownCell(observation.Detail),
		)
	}

	report.WriteString("\n## External traffic evidence\n\n")
	if len(document.Evidence) == 0 {
		if document.External.HTTPConfigured || document.External.RDPAddress != "" {
			report.WriteString("External traffic targets were configured, but no successful evidence was recorded.\n")
		} else {
			report.WriteString("No external traffic targets were configured.\n")
		}
		return report.String()
	}
	report.WriteString("| Transport | Route | Evidence | Detail |\n")
	report.WriteString("|---|---|---|---|\n")
	for _, evidence := range document.Evidence {
		detail := "verified"
		switch evidence.Kind {
		case "http-body":
			detail = fmt.Sprintf("bytes=%d sha256=%s", evidence.Bytes, evidence.SHA256)
		case "rdp-negotiation":
			detail = fmt.Sprintf("bytes=%d type=%#x protocol=%#x", evidence.Bytes, evidence.NegotiationType, evidence.SelectedProtocol)
		case "rdp-desktop":
			detail = fmt.Sprintf("ordered NLA/framebuffer/ACTIVE markers observed; process alive post-activation=%d ms", evidence.DurationMS)
		}
		fmt.Fprintf(
			&report,
			"| %s | %s | %s | %s |\n",
			markdownCell(evidence.Transport),
			markdownCell(evidence.Route),
			markdownCell(evidence.Kind),
			markdownCell(detail),
		)
	}
	return report.String()
}

func markdownCell(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\r\n", "<br>")
	value = strings.ReplaceAll(value, "\n", "<br>")
	value = strings.ReplaceAll(value, "\r", "<br>")
	return value
}
