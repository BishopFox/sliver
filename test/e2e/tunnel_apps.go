package e2e

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func tunnelHTTPDestination(rawURL string) (*url.URL, string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, "", fmt.Errorf("parse tunnel target HTTP URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, "", fmt.Errorf("tunnel target URL scheme = %q, want http or https", parsed.Scheme)
	}
	if parsed.Hostname() == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, "", errors.New("tunnel target HTTP URL requires a host and must not contain credentials, a query, or a fragment")
	}
	port := parsed.Port()
	if port == "" {
		if parsed.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	return parsed, net.JoinHostPort(parsed.Hostname(), port), nil
}

func tunnelHTTPOrigin(rawURL string) string {
	parsed, _, err := tunnelHTTPDestination(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}

const (
	tunnelHTTPBodyCaptureBytes   = 16 * 1024 * 1024
	tunnelHTTPStderrCaptureBytes = 64 * 1024
	tunnelHTTPURLReadBytes       = 16 * 1024
	tunnelHTTPURLReadTimeout     = 10 * time.Second
)

func readTunnelHTTPURLFD(ctx context.Context, descriptor int) (string, error) {
	if descriptor < 0 {
		return "", nil
	}
	if ctx == nil {
		return "", errors.New("read tunnel HTTP URL without a context")
	}
	payload, err := readBoundedE2EFileDescriptor(ctx, descriptor, tunnelHTTPURLReadBytes+1)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", fmt.Errorf("read tunnel HTTP URL from descriptor %d: %w", descriptor, ctxErr)
		}
		return "", fmt.Errorf("read tunnel HTTP URL from descriptor %d: %w", descriptor, err)
	}
	if len(payload) > tunnelHTTPURLReadBytes {
		return "", fmt.Errorf("tunnel HTTP URL descriptor %d exceeds %d bytes", descriptor, tunnelHTTPURLReadBytes)
	}
	rawURL := strings.TrimSpace(string(payload))
	if rawURL == "" {
		return "", fmt.Errorf("tunnel HTTP URL descriptor %d is empty", descriptor)
	}
	if _, _, err := tunnelHTTPDestination(rawURL); err != nil {
		// url.Parse errors retain the complete input URL. Descriptor-backed
		// targets may contain a sensitive path, so do not wrap or otherwise
		// expose the parser error outside this boundary.
		return "", fmt.Errorf("tunnel HTTP URL descriptor %d contains an invalid HTTP(S) target", descriptor)
	}
	return rawURL, nil
}

type tunnelHTTPCurlRequest struct {
	targetURL  string
	noProxy    string
	connectTo  string
	socksProxy string
}

type boundedTunnelHTTPCapture struct {
	buffer     bytes.Buffer
	limit      int
	total      int64
	truncated  bool
	retainTail bool
}

func (capture *boundedTunnelHTTPCapture) Write(data []byte) (int, error) {
	capture.total += int64(len(data))
	if capture.limit <= 0 {
		capture.truncated = capture.truncated || len(data) > 0
		return len(data), nil
	}
	if !capture.retainTail {
		remaining := capture.limit - capture.buffer.Len()
		if remaining > 0 {
			if len(data) < remaining {
				remaining = len(data)
			}
			_, _ = capture.buffer.Write(data[:remaining])
		}
		capture.truncated = capture.truncated || int64(capture.buffer.Len()) < capture.total
		return len(data), nil
	}

	if len(data) >= capture.limit {
		capture.buffer.Reset()
		_, _ = capture.buffer.Write(data[len(data)-capture.limit:])
		capture.truncated = true
		return len(data), nil
	}
	overflow := capture.buffer.Len() + len(data) - capture.limit
	if overflow > 0 {
		retained := capture.buffer.Bytes()
		copy(retained, retained[overflow:])
		capture.buffer.Truncate(len(retained) - overflow)
		capture.truncated = true
	}
	_, _ = capture.buffer.Write(data)
	return len(data), nil
}

func curlConfigQuote(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		"\t", `\t`,
		"\n", `\n`,
		"\r", `\r`,
		"\v", `\v`,
	)
	return `"` + replacer.Replace(value) + `"`
}

func tunnelHTTPCurlConfig(request tunnelHTTPCurlRequest) string {
	var config strings.Builder
	config.WriteString("fail\n")
	config.WriteString("silent\n")
	config.WriteString("show-error\n")
	fmt.Fprintf(&config, "max-time = %s\n", curlConfigQuote("60"))
	fmt.Fprintf(&config, "max-filesize = %s\n", curlConfigQuote(strconv.Itoa(tunnelHTTPBodyCaptureBytes)))
	fmt.Fprintf(&config, "noproxy = %s\n", curlConfigQuote(request.noProxy))
	if request.connectTo != "" {
		fmt.Fprintf(&config, "connect-to = %s\n", curlConfigQuote(request.connectTo))
	}
	if request.socksProxy != "" {
		fmt.Fprintf(&config, "socks5-hostname = %s\n", curlConfigQuote(request.socksProxy))
	}
	fmt.Fprintf(&config, "url = %s\n", curlConfigQuote(request.targetURL))
	return config.String()
}

func newTunnelHTTPCurlCommand(ctx context.Context, curlPath string, request tunnelHTTPCurlRequest, environment []string) *exec.Cmd {
	// -q must be curl's first argument to prevent loading any host .curlrc
	// before it consumes the complete, anonymous config supplied below.
	command := exec.CommandContext(ctx, curlPath, "-q", "--config", "-")
	command.Stdin = strings.NewReader(tunnelHTTPCurlConfig(request))
	command.Env = tunnelAppEnvironment(environment)
	return command
}

func tunnelAppEnvironment(environment []string) []string {
	allowed := map[string]bool{
		"COMSPEC":        true,
		"CURL_CA_BUNDLE": true,
		"HOME":           true,
		"LANG":           true,
		"LC_ALL":         true,
		"LC_CTYPE":       true,
		"PATH":           true,
		"PATHEXT":        true,
		"SSL_CERT_DIR":   true,
		"SSL_CERT_FILE":  true,
		"SYSTEMDRIVE":    true,
		"SYSTEMROOT":     true,
		"TEMP":           true,
		"TMP":            true,
		"TMPDIR":         true,
		"USERPROFILE":    true,
		"WINDIR":         true,
	}
	result := make([]string, 0, len(allowed))
	for _, entry := range environment {
		name, value, found := strings.Cut(entry, "=")
		upperName := strings.ToUpper(name)
		if !found || !allowed[upperName] {
			continue
		}
		result = envWith(result, upperName, value)
	}
	sort.Strings(result)
	return result
}

func sanitizeTunnelHTTPDiagnostic(diagnostic string, rawURL string) string {
	replacements := []string{rawURL}
	if parsed, err := url.Parse(rawURL); err == nil {
		replacements = append(replacements, parsed.RequestURI(), parsed.EscapedPath(), parsed.RawPath, parsed.Path)
	}
	for _, value := range replacements {
		if value == "" || value == "/" {
			continue
		}
		diagnostic = strings.ReplaceAll(diagnostic, value, "[target-url]")
	}
	return diagnostic
}

func (s *suite) curlTunnelHTTP(ctx context.Context, request tunnelHTTPCurlRequest) ([]byte, error) {
	curlPath, err := exec.LookPath("curl")
	if err != nil {
		return nil, fmt.Errorf("HTTP tunnel validation requires curl: %w", err)
	}
	// The configured URL, including its path, remains in the anonymous config
	// stream. It must not appear in the curl child process's argument vector.
	command := newTunnelHTTPCurlCommand(ctx, curlPath, request, s.serverEnv)
	command.Dir = s.opts.repoPath
	stdout := boundedTunnelHTTPCapture{limit: tunnelHTTPBodyCaptureBytes}
	stderr := boundedTunnelHTTPCapture{limit: tunnelHTTPStderrCaptureBytes, retainTail: true}
	command.Stdout = &stdout
	command.Stderr = &stderr
	runErr := command.Run()
	if stdout.truncated {
		return nil, fmt.Errorf(
			"curl HTTP response exceeded the %d-byte capture limit (received at least %d bytes)",
			tunnelHTTPBodyCaptureBytes,
			stdout.total,
		)
	}
	if runErr != nil {
		diagnostic := sanitizeTunnelHTTPDiagnostic(strings.TrimSpace(stderr.buffer.String()), request.targetURL)
		if stderr.truncated {
			diagnostic = "[earlier curl stderr truncated] " + diagnostic
		}
		if diagnostic == "" {
			return nil, fmt.Errorf("curl HTTP request: %w", runErr)
		}
		return nil, fmt.Errorf("curl HTTP request: %w: %s", runErr, diagnostic)
	}
	if stdout.buffer.Len() == 0 {
		return nil, errors.New("curl HTTP request returned an empty body")
	}
	return append([]byte(nil), stdout.buffer.Bytes()...), nil
}

func (s *suite) directTunnelHTTPBaseline(ctx context.Context) ([]byte, error) {
	s.tunnelHTTPBaselineOnce.Do(func() {
		s.tunnelHTTPBaseline, s.tunnelHTTPBaselineErr = s.curlTunnelHTTP(ctx, tunnelHTTPCurlRequest{
			targetURL: s.opts.tunnelHTTPURL,
			noProxy:   "*",
		})
	})
	return append([]byte(nil), s.tunnelHTTPBaseline...), s.tunnelHTTPBaselineErr
}

func requireTunnelHTTPMatch(route string, baseline []byte, tunneled []byte) error {
	if !bytes.Equal(tunneled, baseline) {
		wantDigest := sha256.Sum256(baseline)
		gotDigest := sha256.Sum256(tunneled)
		return fmt.Errorf(
			"%s body mismatch: got %d bytes sha256=%s, want %d bytes sha256=%s",
			route,
			len(tunneled),
			hex.EncodeToString(gotDigest[:]),
			len(baseline),
			hex.EncodeToString(wantDigest[:]),
		)
	}
	return nil
}

func (s *suite) logTunnelHTTPMatch(transport string, route string, body []byte) {
	digest := sha256.Sum256(body)
	digestText := hex.EncodeToString(digest[:])
	s.t.Logf("Validated %s real HTTP body: bytes=%d sha256=%s", route, len(body), digestText)
	if s.tunnelReport != nil {
		s.tunnelReport.recordEvidence(tunnelReportEvidence{
			Transport: transport,
			Route:     route,
			Kind:      "http-body",
			Bytes:     len(body),
			SHA256:    digestText,
		})
	}
}

func (s *suite) exercisePortForwardExternalHTTP(target implantTarget, transport string) (resultErr error) {
	parsed, destination, err := tunnelHTTPDestination(s.opts.tunnelHTTPURL)
	if err != nil {
		return err
	}
	baselineCtx, baselineCancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
	baseline, err := s.directTunnelHTTPBaseline(baselineCtx)
	baselineCancel()
	if err != nil {
		return fmt.Errorf("direct real HTTP baseline: %w", err)
	}

	forward, err := s.startPortForward(target, destination, 30*time.Second)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, forward.stop()) }()
	localHost, localPort, err := net.SplitHostPort(forward.bindAddress)
	if err != nil {
		return err
	}
	targetPort := parsed.Port()
	if targetPort == "" {
		if parsed.Scheme == "https" {
			targetPort = "443"
		} else {
			targetPort = "80"
		}
	}
	connectTo := fmt.Sprintf("%s:%s:%s:%s", parsed.Hostname(), targetPort, localHost, localPort)
	requestCtx, requestCancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
	tunneled, err := s.curlTunnelHTTP(requestCtx, tunnelHTTPCurlRequest{
		targetURL: s.opts.tunnelHTTPURL,
		noProxy:   "*",
		connectTo: connectTo,
	})
	requestCancel()
	if err != nil {
		return fmt.Errorf("port-forward real HTTP request: %w", err)
	}
	if err := requireTunnelHTTPMatch("port-forward", baseline, tunneled); err != nil {
		return err
	}
	s.logTunnelHTTPMatch(transport, "port-forward", tunneled)
	return nil
}

func (s *suite) socksE2EExternalHTTP(ctx context.Context, session *clientpb.Session, transport string) error {
	baseline, err := s.directTunnelHTTPBaseline(ctx)
	if err != nil {
		return fmt.Errorf("direct real HTTP baseline: %w", err)
	}
	return s.withSocksE2EProxy(ctx, session, "", "", func(proxyServer *socksE2EProxy) error {
		tunneled, err := s.curlTunnelHTTP(ctx, tunnelHTTPCurlRequest{
			targetURL:  s.opts.tunnelHTTPURL,
			noProxy:    "",
			socksProxy: proxyServer.address,
		})
		if err != nil {
			return fmt.Errorf("SOCKS5 real HTTP request: %w", err)
		}
		if err := requireTunnelHTTPMatch("SOCKS5", baseline, tunneled); err != nil {
			return err
		}
		s.logTunnelHTTPMatch(transport, "SOCKS5", tunneled)
		return nil
	})
}

var rdpNegotiationRequest = []byte{
	0x03, 0x00, 0x00, 0x13,
	0x0e, 0xe0, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x01, 0x00, 0x08, 0x00, 0x03, 0x00, 0x00, 0x00,
}

type rdpNegotiationResult struct {
	responseBytes    int
	selectedProtocol uint32
	negotiationType  byte
}

const (
	rdpCredentialReadTimeout     = 10 * time.Second
	rdpDesktopActivationTimeout  = 60 * time.Second
	rdpDesktopAcceptanceDuration = 15 * time.Second
	rdpDesktopStartupAllowance   = 10 * time.Second
	freeRDPLogFilter             = "com.freerdp.core.rdp:DEBUG,com.freerdp.gdi:INFO"
	freeRDPOutputCaptureBytes    = 256 * 1024
)

var freeRDPDesktopActivationMarkers = []string{
	"CONNECTION_STATE_NEGO --> CONNECTION_STATE_NLA",
	"Local framebuffer format",
	"Remote framebuffer format",
	"--> CONNECTION_STATE_ACTIVE",
}

type rdpCredentials struct {
	Domain                 string `json:"domain"`
	Username               string `json:"username"`
	Password               string `json:"password"`
	ServerName             string `json:"server_name"`
	CertificateFingerprint string `json:"certificate_fingerprint"`
}

func readRDPCredentialsFD(ctx context.Context, descriptor int) (*rdpCredentials, error) {
	if descriptor < 0 {
		return nil, nil
	}
	if ctx == nil {
		return nil, errors.New("read RDP credentials without a context")
	}
	credentialReadError := func(action string, readErr error) error {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf("%s RDP credentials from descriptor %d: %w", action, descriptor, ctxErr)
		}
		return fmt.Errorf("%s RDP credentials from descriptor %d: %w", action, descriptor, readErr)
	}
	payload, err := readBoundedE2EFileDescriptor(ctx, descriptor, 64*1024+1)
	if err != nil {
		return nil, credentialReadError("read", err)
	}
	if len(payload) > 64*1024 {
		return nil, fmt.Errorf("RDP credential descriptor %d exceeds %d bytes", descriptor, 64*1024)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	credentials := &rdpCredentials{}
	if err := decoder.Decode(credentials); err != nil {
		return nil, credentialReadError("decode", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, fmt.Errorf("RDP credential descriptor %d contains more than one JSON value", descriptor)
		}
		return nil, credentialReadError("validate", err)
	}
	if credentials.Username == "" || credentials.Password == "" || credentials.ServerName == "" || credentials.CertificateFingerprint == "" {
		return nil, errors.New("RDP credentials require non-empty username, password, server_name, and certificate_fingerprint fields")
	}
	for label, value := range map[string]string{
		"domain":      credentials.Domain,
		"username":    credentials.Username,
		"password":    credentials.Password,
		"server_name": credentials.ServerName,
	} {
		if strings.ContainsAny(value, "\r\n") {
			return nil, fmt.Errorf("RDP credential %s contains a newline", label)
		}
	}
	if strings.IndexFunc(credentials.ServerName, unicode.IsSpace) >= 0 || strings.IndexFunc(credentials.ServerName, unicode.IsControl) >= 0 {
		return nil, errors.New("RDP server_name contains whitespace or a control character")
	}
	credentials.CertificateFingerprint, err = canonicalRDPCertificateFingerprint(credentials.CertificateFingerprint)
	if err != nil {
		return nil, err
	}
	return credentials, nil
}

func canonicalRDPCertificateFingerprint(value string) (string, error) {
	if strings.IndexFunc(value, unicode.IsSpace) >= 0 || strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return "", errors.New("RDP certificate fingerprint contains whitespace or a control character")
	}
	parts := strings.Split(strings.ToLower(value), ":")
	if len(parts) < 2 || parts[0] != "sha256" {
		return "", errors.New("RDP certificate fingerprint must use sha256:<64 hexadecimal digits>")
	}
	digestText := strings.Join(parts[1:], "")
	digest, err := hex.DecodeString(digestText)
	if err != nil || len(digest) != sha256.Size {
		return "", errors.New("RDP certificate fingerprint must use sha256:<64 hexadecimal digits>")
	}
	return "sha256:" + hex.EncodeToString(digest), nil
}

func canonicalTunnelTCPAddress(address string) (string, error) {
	host, rawPort, err := net.SplitHostPort(address)
	if err != nil {
		return "", err
	}
	if host == "" {
		return "", errors.New("TCP target host is empty")
	}
	if strings.IndexFunc(host, unicode.IsControl) >= 0 {
		return "", errors.New("TCP target host contains a control character")
	}
	port, err := strconv.Atoi(rawPort)
	if err != nil || port < 1 || port > 65535 {
		return "", fmt.Errorf("TCP target port %q is outside 1-65535", rawPort)
	}
	return net.JoinHostPort(host, strconv.Itoa(port)), nil
}

func validateRDPCommandTimeout(credentialsDescriptor int, commandTimeout time.Duration) error {
	if credentialsDescriptor < 0 {
		return nil
	}
	minimum := minimumRDPCommandTimeout()
	if commandTimeout < minimum {
		return fmt.Errorf("authenticated RDP requires -command-timeout of at least %s", minimum)
	}
	return nil
}

func minimumRDPCommandTimeout() time.Duration {
	return rdpDesktopActivationTimeout +
		rdpDesktopAcceptanceDuration +
		cleanupGraceTimeout +
		cleanupProcessTimeout +
		socksE2ECaseTimeout +
		rdpDesktopStartupAllowance
}

func wrapIfError(message string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

//nolint:gocyclo // Parsing the bounded TPKT/X.224 negotiation response requires explicit protocol branches.
func rdpNegotiationRoundTrip(ctx context.Context, connection net.Conn) (rdpNegotiationResult, error) {
	if deadline, ok := ctx.Deadline(); ok {
		if err := connection.SetDeadline(deadline); err != nil {
			return rdpNegotiationResult{}, err
		}
	}
	written, err := connection.Write(rdpNegotiationRequest)
	if err != nil {
		return rdpNegotiationResult{}, fmt.Errorf("write RDP negotiation request: %w", err)
	}
	if written != len(rdpNegotiationRequest) {
		return rdpNegotiationResult{}, fmt.Errorf("write RDP negotiation request: %w (%d of %d bytes)", io.ErrShortWrite, written, len(rdpNegotiationRequest))
	}
	header := make([]byte, 4)
	if _, err := io.ReadFull(connection, header); err != nil {
		return rdpNegotiationResult{}, fmt.Errorf("read RDP TPKT header: %w", err)
	}
	if header[0] != 0x03 || header[1] != 0x00 {
		return rdpNegotiationResult{}, fmt.Errorf("RDP TPKT header = %x, want version 03 and reserved 00", header)
	}
	packetLength := int(binary.BigEndian.Uint16(header[2:]))
	if packetLength < 11 || packetLength > 4096 {
		return rdpNegotiationResult{}, fmt.Errorf("RDP TPKT packet length = %d, want 11-4096", packetLength)
	}
	payload := make([]byte, packetLength-len(header))
	if _, err := io.ReadFull(connection, payload); err != nil {
		return rdpNegotiationResult{}, fmt.Errorf("read RDP TPKT payload: %w", err)
	}
	if len(payload) < 7 || payload[1] != 0xd0 {
		return rdpNegotiationResult{}, fmt.Errorf("RDP X.224 response = %x, want connection confirm", payload)
	}
	if len(payload) < 15 {
		return rdpNegotiationResult{}, fmt.Errorf("RDP X.224 confirm omitted the requested 8-byte negotiation response")
	}
	result := rdpNegotiationResult{responseBytes: packetLength}
	negotiation := payload[7:15]
	result.negotiationType = negotiation[0]
	if binary.LittleEndian.Uint16(negotiation[2:4]) != 8 {
		return rdpNegotiationResult{}, fmt.Errorf("RDP negotiation structure length = %d, want 8", binary.LittleEndian.Uint16(negotiation[2:4]))
	}
	if result.negotiationType == 0x03 {
		return rdpNegotiationResult{}, fmt.Errorf("RDP server returned negotiation failure code %#x", binary.LittleEndian.Uint32(negotiation[4:8]))
	}
	if result.negotiationType != 0x02 {
		return rdpNegotiationResult{}, fmt.Errorf("RDP negotiation response type = %#x, want 0x02", result.negotiationType)
	}
	result.selectedProtocol = binary.LittleEndian.Uint32(negotiation[4:8])
	if result.selectedProtocol != 0x01 && result.selectedProtocol != 0x02 {
		return rdpNegotiationResult{}, fmt.Errorf("RDP selected protocol = %#x, want requested TLS (0x1) or HYBRID/NLA (0x2)", result.selectedProtocol)
	}
	return result, nil
}

func (s *suite) logRDPNegotiation(transport string, route string, result rdpNegotiationResult) {
	s.t.Logf(
		"Validated %s RDP negotiation: response-bytes=%d response-type=%#x selected-protocol=%#x",
		route,
		result.responseBytes,
		result.negotiationType,
		result.selectedProtocol,
	)
	if s.tunnelReport != nil {
		s.tunnelReport.recordEvidence(tunnelReportEvidence{
			Transport:        transport,
			Route:            route,
			Kind:             "rdp-negotiation",
			Bytes:            result.responseBytes,
			NegotiationType:  uint32(result.negotiationType),
			SelectedProtocol: result.selectedProtocol,
		})
	}
}

func (s *suite) freeRDPArguments(endpoint string, socksProxy string) []string {
	arguments := []string{
		"/v:" + endpoint,
		"/u:" + s.rdpCredentials.Username,
		"/p:" + s.rdpCredentials.Password,
		"-multitransport",
		"/cert:deny,fingerprint:" + s.rdpCredentials.CertificateFingerprint,
		"/server-name:" + s.rdpCredentials.ServerName,
		"/sec:nla",
		"/timeout:30000",
		"/network:lan",
		"/log-level:INFO",
	}
	if s.rdpCredentials.Domain != "" {
		arguments = append(arguments, "/d:"+s.rdpCredentials.Domain)
	}
	if socksProxy != "" {
		arguments = append(arguments, "/proxy:socks5://"+socksProxy)
	}
	return arguments
}

func (s *suite) freeRDPDesktopArguments(endpoint string, socksProxy string) []string {
	arguments := s.freeRDPArguments(endpoint, socksProxy)
	return append(arguments,
		"/size:1024x768",
		"/bpp:16",
		"/audio-mode:2",
		"-clipboard",
	)
}

func redactFreeRDPOutput(output string, credentials *rdpCredentials) string {
	if credentials == nil {
		return output
	}
	redacted := output
	for _, value := range []string{credentials.Password, credentials.Username, credentials.Domain} {
		if value != "" {
			// FreeRDP and its authentication libraries can normalize account and
			// realm text to a different case before logging it. Redact every case
			// variant so runtime-only credential fields cannot enter a report.
			pattern := regexp.MustCompile("(?i:" + regexp.QuoteMeta(value) + ")")
			redacted = pattern.ReplaceAllLiteralString(redacted, "[REDACTED]")
		}
	}
	return redacted
}

func requireFreeRDPDesktopActivation(output string) error {
	searchFrom := 0
	for _, marker := range freeRDPDesktopActivationMarkers {
		markerOffset := strings.Index(output[searchFrom:], marker)
		if markerOffset < 0 {
			return fmt.Errorf("FreeRDP output did not contain desktop activation marker %q", marker)
		}
		searchFrom += markerOffset + len(marker)
	}
	return nil
}

type freeRDPActivationCapture struct {
	mutex           sync.Mutex
	output          []byte
	markerRemainder []byte
	markerIndex     int
	truncated       bool
	activated       chan struct{}
	once            sync.Once
}

func newFreeRDPActivationCapture() *freeRDPActivationCapture {
	return &freeRDPActivationCapture{activated: make(chan struct{})}
}

func (capture *freeRDPActivationCapture) Write(data []byte) (int, error) {
	capture.mutex.Lock()
	capture.appendOutputTail(data)
	capture.scanActivationMarkers(data)
	active := capture.markerIndex == len(freeRDPDesktopActivationMarkers)
	capture.mutex.Unlock()
	if active {
		capture.once.Do(func() { close(capture.activated) })
	}
	return len(data), nil
}

func (capture *freeRDPActivationCapture) String() string {
	capture.mutex.Lock()
	defer capture.mutex.Unlock()
	if capture.truncated {
		return "[earlier FreeRDP output truncated]\n" + string(capture.output)
	}
	return string(capture.output)
}

func (capture *freeRDPActivationCapture) appendOutputTail(data []byte) {
	if len(data) >= freeRDPOutputCaptureBytes {
		capture.output = append(capture.output[:0], data[len(data)-freeRDPOutputCaptureBytes:]...)
		capture.truncated = true
		return
	}
	overflow := len(capture.output) + len(data) - freeRDPOutputCaptureBytes
	if overflow > 0 {
		copy(capture.output, capture.output[overflow:])
		capture.output = capture.output[:len(capture.output)-overflow]
		capture.truncated = true
	}
	capture.output = append(capture.output, data...)
}

func (capture *freeRDPActivationCapture) scanActivationMarkers(data []byte) {
	if capture.markerIndex == len(freeRDPDesktopActivationMarkers) {
		return
	}
	window := make([]byte, 0, len(capture.markerRemainder)+len(data))
	window = append(window, capture.markerRemainder...)
	window = append(window, data...)
	for capture.markerIndex < len(freeRDPDesktopActivationMarkers) {
		marker := []byte(freeRDPDesktopActivationMarkers[capture.markerIndex])
		offset := bytes.Index(window, marker)
		if offset < 0 {
			keep := len(marker) - 1
			if keep > len(window) {
				keep = len(window)
			}
			capture.markerRemainder = append(capture.markerRemainder[:0], window[len(window)-keep:]...)
			return
		}
		window = window[offset+len(marker):]
		capture.markerIndex++
	}
	capture.markerRemainder = capture.markerRemainder[:0]
}

func freeRDPLineBufferedArguments(freeRDPPath string) []string {
	// FreeRDP's console logger is block buffered when stdout is a pipe. The
	// DEBUG state-transition stream can fill one 8 KiB block immediately before
	// CONNECTION_STATE_ACTIVE, leaving the activation marker buffered while the
	// desktop is already live. stdbuf keeps the production process output
	// observable without weakening the ordered activation oracle.
	return []string{"-oL", "-eL", freeRDPPath, "/args-from:fd:3"}
}

//nolint:gocyclo // Process, credential, activation, framebuffer, and cleanup checks form one acceptance lifecycle.
func (s *suite) runFreeRDPDesktop(ctx context.Context, transport string, route string, endpoint string, socksProxy string) (resultErr error) {
	if s.rdpCredentials == nil {
		return nil
	}
	freeRDPPath, err := exec.LookPath("xfreerdp3")
	if err != nil {
		return fmt.Errorf("%s RDP desktop validation requires FreeRDP 3: %w", route, err)
	}
	xvfbRunPath, err := exec.LookPath("xvfb-run")
	if err != nil {
		return fmt.Errorf("%s RDP desktop validation requires xvfb-run: %w", route, err)
	}
	stdbufPath, err := exec.LookPath("stdbuf")
	if err != nil {
		return fmt.Errorf("%s RDP desktop validation requires stdbuf: %w", route, err)
	}
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("create %s FreeRDP desktop argument pipe: %w", route, err)
	}

	commandArguments := append([]string{"-a", stdbufPath}, freeRDPLineBufferedArguments(freeRDPPath)...)
	command := exec.CommandContext(ctx, xvfbRunPath, commandArguments...)
	command.Dir = s.opts.repoPath
	command.Env = envWith(tunnelAppEnvironment(s.serverEnv), "WLOG_FILTER", freeRDPLogFilter)
	command.ExtraFiles = []*os.File{readPipe}
	capture := newFreeRDPActivationCapture()
	command.Stdout = capture
	command.Stderr = capture
	prepareCommand(command)
	if err := command.Start(); err != nil {
		return errors.Join(
			fmt.Errorf("start %s FreeRDP desktop session: %w", route, err),
			readPipe.Close(),
			writePipe.Close(),
		)
	}
	if err := readPipe.Close(); err != nil {
		_ = writePipe.Close()
		_ = killPreparedProcessTree(command)
		_ = command.Wait()
		return fmt.Errorf("close parent %s FreeRDP desktop argument reader: %w", route, err)
	}
	tree, err := attachProcessTree(command)
	if err != nil {
		_ = killPreparedProcessTree(command)
		_ = command.Wait()
		_ = writePipe.Close()
		return fmt.Errorf("attach %s FreeRDP desktop process tree: %w", route, err)
	}
	process := &managedProcess{cmd: command, done: make(chan struct{}), tree: tree}
	go func() {
		process.err = command.Wait()
		close(process.done)
	}()
	cleanupPending := true
	defer func() {
		if cleanupPending {
			resultErr = errors.Join(resultErr, process.stop())
		}
	}()
	argumentBytes := []byte(strings.Join(s.freeRDPDesktopArguments(endpoint, socksProxy), "\n") + "\n")
	defer func() {
		for index := range argumentBytes {
			argumentBytes[index] = 0
		}
	}()
	writeDone := make(chan error, 1)
	go func() {
		written, writeErr := writePipe.Write(argumentBytes)
		if writeErr == nil && written != len(argumentBytes) {
			writeErr = io.ErrShortWrite
		}
		closeErr := writePipe.Close()
		writeDone <- errors.Join(writeErr, closeErr)
	}()
	select {
	case writeErr := <-writeDone:
		if writeErr != nil {
			return fmt.Errorf("write %s FreeRDP desktop runtime arguments: %w", route, writeErr)
		}
	case <-process.done:
		_ = writePipe.Close()
		writeErr := <-writeDone
		runErr := process.err
		if runErr == nil {
			runErr = errors.New("FreeRDP exited before reading its runtime arguments")
		}
		return fmt.Errorf("%s FreeRDP desktop argument delivery: %w", route, errors.Join(runErr, writeErr))
	case <-ctx.Done():
		_ = writePipe.Close()
		writeErr := <-writeDone
		return fmt.Errorf("%s FreeRDP desktop argument delivery: %w", route, errors.Join(ctx.Err(), writeErr))
	}
	activationTimer := time.NewTimer(rdpDesktopActivationTimeout)
	defer activationTimer.Stop()
	select {
	case <-capture.activated:
	case <-process.done:
		runErr := process.err
		if runErr == nil {
			runErr = errors.New("FreeRDP exited before desktop activation")
		}
		return fmt.Errorf("%s FreeRDP desktop activation: %w: %s", route, runErr, redactFreeRDPOutput(strings.TrimSpace(capture.String()), s.rdpCredentials))
	case <-activationTimer.C:
		return fmt.Errorf("%s FreeRDP did not activate a desktop within %s: %s", route, rdpDesktopActivationTimeout, redactFreeRDPOutput(strings.TrimSpace(capture.String()), s.rdpCredentials))
	case <-ctx.Done():
		return fmt.Errorf("%s FreeRDP desktop activation: %w", route, ctx.Err())
	}
	activatedAt := time.Now()
	acceptanceTimer := time.NewTimer(rdpDesktopAcceptanceDuration)
	defer acceptanceTimer.Stop()
	select {
	case <-acceptanceTimer.C:
		select {
		case <-process.done:
			runErr := process.err
			if runErr == nil {
				runErr = errors.New("FreeRDP exited at the desktop acceptance boundary")
			}
			return fmt.Errorf("%s FreeRDP desktop session: %w: %s", route, runErr, redactFreeRDPOutput(strings.TrimSpace(capture.String()), s.rdpCredentials))
		default:
		}
	case <-process.done:
		runErr := process.err
		if runErr == nil {
			runErr = errors.New("FreeRDP exited before the post-activation acceptance window")
		}
		return fmt.Errorf("%s FreeRDP desktop session: %w after %s: %s", route, runErr, time.Since(activatedAt).Round(time.Millisecond), redactFreeRDPOutput(strings.TrimSpace(capture.String()), s.rdpCredentials))
	case <-ctx.Done():
		return fmt.Errorf("%s FreeRDP desktop acceptance: %w", route, ctx.Err())
	}
	heldDuration := time.Since(activatedAt)
	stopErr := process.stop()
	cleanupPending = false
	if stopErr != nil {
		return fmt.Errorf("stop accepted %s FreeRDP desktop session: %w", route, stopErr)
	}
	s.t.Logf(
		"Validated %s FreeRDP NLA desktop activation and %s established session",
		route,
		rdpDesktopAcceptanceDuration,
	)
	if s.tunnelReport != nil {
		s.tunnelReport.recordEvidence(tunnelReportEvidence{
			Transport: transport,
			Route:     route,
			Kind:      "rdp-nla",
		})
		s.tunnelReport.recordEvidence(tunnelReportEvidence{
			Transport:  transport,
			Route:      route,
			Kind:       "rdp-desktop",
			DurationMS: heldDuration.Round(time.Millisecond).Milliseconds(),
		})
	}
	return nil
}

func (s *suite) exercisePortForwardRDP(target implantTarget, transport string) (resultErr error) {
	forward, err := s.startPortForward(target, s.opts.tunnelRDPAddr, 30*time.Second)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, forward.stop()) }()
	ctx, cancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
	defer cancel()
	if s.rdpCredentials != nil {
		if err := s.runFreeRDPDesktop(ctx, transport, "port-forward", forward.bindAddress, ""); err != nil {
			return err
		}
	}
	connection, err := (&net.Dialer{Timeout: socksE2ECaseTimeout}).DialContext(ctx, "tcp4", forward.bindAddress)
	if err != nil {
		return fmt.Errorf("dial RDP port forward: %w", err)
	}
	result, negotiationErr := rdpNegotiationRoundTrip(ctx, connection)
	closeErr := connection.Close()
	if err := errors.Join(
		wrapIfError("port-forward RDP negotiation", negotiationErr),
		wrapIfError("close port-forward RDP negotiation connection", closeErr),
	); err != nil {
		return err
	}
	s.logRDPNegotiation(transport, "port-forward", result)
	return nil
}

func (s *suite) socksE2ERDP(ctx context.Context, session *clientpb.Session, transport string) error {
	return s.withSocksE2EProxy(ctx, session, "", "", func(proxyServer *socksE2EProxy) error {
		if s.rdpCredentials != nil {
			if err := s.runFreeRDPDesktop(ctx, transport, "SOCKS5", s.opts.tunnelRDPAddr, proxyServer.address); err != nil {
				return err
			}
		}
		connection, err := socksE2EDial(ctx, proxyServer.address, "", "", s.opts.tunnelRDPAddr)
		if err != nil {
			return err
		}
		result, negotiationErr := rdpNegotiationRoundTrip(ctx, connection)
		closeErr := connection.Close()
		if err := errors.Join(
			wrapIfError("SOCKS5 RDP negotiation", negotiationErr),
			wrapIfError("close SOCKS5 RDP negotiation connection", closeErr),
		); err != nil {
			return err
		}
		s.logRDPNegotiation(transport, "SOCKS5", result)
		return nil
	})
}
