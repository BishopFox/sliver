package e2e

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

const testRDPCertificateFingerprint = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestTunnelHTTPDestination(t *testing.T) {
	tests := []struct {
		name        string
		rawURL      string
		wantAddress string
		wantError   bool
	}{
		{name: "http default", rawURL: "http://10.13.37.10/path", wantAddress: "10.13.37.10:80"},
		{name: "https default", rawURL: "https://example.test/path", wantAddress: "example.test:443"},
		{name: "custom port", rawURL: "http://example.test:8080/path", wantAddress: "example.test:8080"},
		{name: "credentials rejected", rawURL: "http://user:password@example.test/", wantError: true},
		{name: "query rejected", rawURL: "https://example.test/path?token=secret", wantError: true},
		{name: "fragment rejected", rawURL: "https://example.test/path#secret", wantError: true},
		{name: "unsupported scheme", rawURL: "ftp://example.test/", wantError: true},
		{name: "missing host", rawURL: "http:///path", wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, address, err := tunnelHTTPDestination(test.rawURL)
			if test.wantError {
				if err == nil {
					t.Fatalf("tunnelHTTPDestination(%q) succeeded", test.rawURL)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if address != test.wantAddress {
				t.Fatalf("destination = %q, want %q", address, test.wantAddress)
			}
		})
	}
}

//nolint:gocyclo // The fake curl matrix verifies argv secrecy, stdin config, output bounds, and cancellation together.
func TestTunnelHTTPCurlUsesConfigStdinAndBoundedCapture(t *testing.T) {
	secretURL := `https://example.test/private/path-token`
	request := tunnelHTTPCurlRequest{
		targetURL:  secretURL,
		noProxy:    "",
		socksProxy: "127.0.0.1:1080",
	}
	environment := []string{
		"HOME=/isolated/home",
		"PATH=/usr/bin",
		"SSL_CERT_FILE=/isolated/ca.pem",
		"CURL_HOME=/host/curl",
		"XDG_CONFIG_HOME=/host/config",
		"ALL_PROXY=http://operator-secret",
		"AWS_SECRET_ACCESS_KEY=operator-secret",
	}
	command := newTunnelHTTPCurlCommand(context.Background(), "/path/to/curl", request, environment)
	for _, argument := range command.Args {
		if strings.Contains(argument, secretURL) || strings.Contains(argument, "path-token") {
			t.Fatalf("curl argv exposed target URL: %q", command.Args)
		}
	}
	if got, want := strings.Join(command.Args, "\x00"), "/path/to/curl\x00-q\x00--config\x00-"; got != want {
		t.Fatalf("curl argv = %q, want %q", got, want)
	}
	environmentValues := map[string]string{}
	for _, entry := range command.Env {
		name, value, found := strings.Cut(entry, "=")
		if !found {
			t.Fatalf("curl environment contains malformed entry %q", entry)
		}
		environmentValues[strings.ToUpper(name)] = value
	}
	if environmentValues["HOME"] != "/isolated/home" || environmentValues["PATH"] != "/usr/bin" || environmentValues["SSL_CERT_FILE"] != "/isolated/ca.pem" {
		t.Fatalf("curl environment omitted isolated runtime values: %v", command.Env)
	}
	for _, forbidden := range []string{"CURL_HOME", "XDG_CONFIG_HOME", "ALL_PROXY", "AWS_SECRET_ACCESS_KEY"} {
		if _, ok := environmentValues[forbidden]; ok {
			t.Fatalf("curl environment retained %s: %v", forbidden, command.Env)
		}
	}
	config, err := io.ReadAll(command.Stdin)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(config), curlConfigQuote(secretURL)) {
		t.Fatal("curl stdin config omitted the target URL")
	}

	prefix := boundedTunnelHTTPCapture{limit: 4}
	_, _ = prefix.Write([]byte("abc"))
	_, _ = prefix.Write([]byte("def"))
	if got := prefix.buffer.String(); got != "abcd" || prefix.total != 6 || !prefix.truncated {
		t.Fatalf("bounded prefix capture = %q total=%d truncated=%t", got, prefix.total, prefix.truncated)
	}
	tail := boundedTunnelHTTPCapture{limit: 4, retainTail: true}
	_, _ = tail.Write([]byte("abc"))
	_, _ = tail.Write([]byte("def"))
	if got := tail.buffer.String(); got != "cdef" || tail.total != 6 || !tail.truncated {
		t.Fatalf("bounded tail capture = %q total=%d truncated=%t", got, tail.total, tail.truncated)
	}
}

func TestTunnelHTTPDiagnosticRedactsURLAndPath(t *testing.T) {
	rawURL := "https://example.test/private/%70ath-token"
	diagnostic := "request " + rawURL + " failed at /private/path-token"
	got := sanitizeTunnelHTTPDiagnostic(diagnostic, rawURL)
	for _, secret := range []string{rawURL, "/private/%70ath-token", "/private/path-token"} {
		if strings.Contains(got, secret) {
			t.Fatalf("sanitized diagnostic retained %q in %q", secret, got)
		}
	}
}

func TestReadTunnelHTTPURLFD(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("descriptor duplication is Unix-only")
	}
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	want := "https://example.test/private/path-token"
	go func() {
		_, _ = io.WriteString(writePipe, want+"\n")
		_ = writePipe.Close()
	}()
	got, err := readTunnelHTTPURLFD(context.Background(), int(readPipe.Fd()))
	_ = readPipe.Close()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("descriptor URL = %q, want %q", got, want)
	}
}

func TestReadTunnelHTTPURLFDBoundsAndDeadline(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("descriptor duplication is Unix-only")
	}
	t.Run("size", func(t *testing.T) {
		readPipe, writePipe, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		go func() {
			_, _ = writePipe.Write(bytes.Repeat([]byte{'x'}, tunnelHTTPURLReadBytes+1))
			_ = writePipe.Close()
		}()
		_, err = readTunnelHTTPURLFD(context.Background(), int(readPipe.Fd()))
		_ = readPipe.Close()
		if err == nil || !strings.Contains(err.Error(), "exceeds") {
			t.Fatalf("oversized URL descriptor error = %v", err)
		}
	})
	t.Run("invalid URL is redacted", func(t *testing.T) {
		readPipe, writePipe, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		secretURL := "https://example.test/private/path-token/%zz"
		go func() {
			_, _ = io.WriteString(writePipe, secretURL)
			_ = writePipe.Close()
		}()
		_, err = readTunnelHTTPURLFD(context.Background(), int(readPipe.Fd()))
		_ = readPipe.Close()
		if err == nil {
			t.Fatal("invalid descriptor URL succeeded")
		}
		for _, secret := range []string{secretURL, "path-token", "%zz"} {
			if strings.Contains(err.Error(), secret) {
				t.Fatalf("descriptor validation error exposed %q: %v", secret, err)
			}
		}
	})
	t.Run("deadline", func(t *testing.T) {
		readPipe, writePipe, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = readPipe.Close() }()
		defer func() { _ = writePipe.Close() }()
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		started := time.Now()
		_, err = readTunnelHTTPURLFD(ctx, int(readPipe.Fd()))
		runtime.KeepAlive(writePipe)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("open URL writer error = %v, want context deadline", err)
		}
		if elapsed := time.Since(started); elapsed > time.Second {
			t.Fatalf("open URL writer remained blocked for %s", elapsed)
		}
	})
}

func TestReadRDPCredentialsFD(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		if _, err := readRDPCredentialsFD(context.Background(), 0); err == nil {
			t.Fatal("unsupported platform accepted an RDP credential descriptor")
		}
		return
	}
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(rdpCredentials{
		Domain:                 "north",
		Username:               "range-user",
		Password:               "runtime-secret",
		ServerName:             "win11-test.north.local",
		CertificateFingerprint: testRDPCertificateFingerprint,
	})
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		_, _ = writePipe.Write(payload)
		_ = writePipe.Close()
	}()
	credentials, err := readRDPCredentialsFD(context.Background(), int(readPipe.Fd()))
	_ = readPipe.Close()
	if err != nil {
		t.Fatal(err)
	}
	if credentials.Domain != "north" || credentials.Username != "range-user" || credentials.Password != "runtime-secret" ||
		credentials.ServerName != "win11-test.north.local" ||
		credentials.CertificateFingerprint != testRDPCertificateFingerprint {
		t.Fatalf("decoded credentials = domain %q username %q password length %d", credentials.Domain, credentials.Username, len(credentials.Password))
	}
	redacted := redactFreeRDPOutput("failure NORTH RANGE-USER RUNTIME-SECRET", credentials)
	for label, value := range map[string]string{
		"domain":   strings.ToUpper(credentials.Domain),
		"username": strings.ToUpper(credentials.Username),
		"password": strings.ToUpper(credentials.Password),
	} {
		if strings.Contains(redacted, value) {
			t.Fatalf("FreeRDP output redaction retained the %s", label)
		}
	}
}

func TestReadRDPCredentialsFDBoundsOpenWriter(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("descriptor duplication is Unix-only")
	}
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = readPipe.Close() }()
	defer func() { _ = writePipe.Close() }()
	payload, err := json.Marshal(rdpCredentials{
		Username:               "range-user",
		Password:               "runtime-secret",
		ServerName:             "win11-test.north.local",
		CertificateFingerprint: testRDPCertificateFingerprint,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writePipe.Write(payload); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err = readRDPCredentialsFD(ctx, int(readPipe.Fd()))
	runtime.KeepAlive(writePipe)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("open credential writer error = %v, want context deadline", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("open credential writer remained blocked for %s", elapsed)
	}
}

func TestCanonicalTunnelTCPAddress(t *testing.T) {
	valid := map[string]string{
		"10.13.37.22:03389":     "10.13.37.22:3389",
		"[::1]:3389":            "[::1]:3389",
		"rdp.example.test:3389": "rdp.example.test:3389",
	}
	for address, want := range valid {
		canonical, err := canonicalTunnelTCPAddress(address)
		if err != nil {
			t.Errorf("canonicalize %q: %v", address, err)
		} else if canonical != want {
			t.Errorf("canonicalize %q = %q, want %q", address, canonical, want)
		}
	}
	for _, address := range []string{
		"missing-port",
		":3389",
		"host:0",
		"host:65536",
		"[rdp.example.test\n/proxy:socks5://attacker]:3389",
		"[rdp.example.test\x00]:3389",
	} {
		if canonical, err := canonicalTunnelTCPAddress(address); err == nil {
			t.Errorf("canonicalize %q = %q, want error", address, canonical)
		}
	}
}

func TestCanonicalRDPCertificateFingerprint(t *testing.T) {
	colonSeparated := "SHA256:01:23:45:67:89:AB:CD:EF:01:23:45:67:89:AB:CD:EF:01:23:45:67:89:AB:CD:EF:01:23:45:67:89:AB:CD:EF"
	if canonical, err := canonicalRDPCertificateFingerprint(colonSeparated); err != nil {
		t.Fatal(err)
	} else if canonical != testRDPCertificateFingerprint {
		t.Fatalf("canonical RDP certificate fingerprint = %q, want %q", canonical, testRDPCertificateFingerprint)
	}
	for _, fingerprint := range []string{
		"",
		"sha1:0123456789abcdef0123456789abcdef01234567",
		"sha256:0123",
		"sha256:zz23456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		testRDPCertificateFingerprint + "\n/cert:ignore",
	} {
		if canonical, err := canonicalRDPCertificateFingerprint(fingerprint); err == nil {
			t.Errorf("canonicalize RDP certificate fingerprint %q = %q, want error", fingerprint, canonical)
		}
	}
}

func TestValidateRDPCommandTimeout(t *testing.T) {
	minimum := minimumRDPCommandTimeout()
	if err := validateRDPCommandTimeout(-1, time.Millisecond); err != nil {
		t.Fatalf("credential-free RDP rejected a short command timeout: %v", err)
	}
	if err := validateRDPCommandTimeout(0, minimum); err != nil {
		t.Fatalf("authenticated RDP rejected the minimum command timeout: %v", err)
	}
	if err := validateRDPCommandTimeout(0, minimum-time.Millisecond); err == nil || !strings.Contains(err.Error(), minimum.String()) {
		t.Fatalf("authenticated RDP short-timeout error = %v, want minimum %s", err, minimum)
	}
}

func TestFreeRDPDesktopArgumentsKeepAcceptanceOnTCPTunnel(t *testing.T) {
	s := &suite{rdpCredentials: &rdpCredentials{
		Domain:                 "NORTH",
		Username:               "range-user",
		Password:               "runtime-secret",
		ServerName:             "win11-test.north.local",
		CertificateFingerprint: testRDPCertificateFingerprint,
	}}
	tests := []struct {
		name string
		args []string
	}{
		{name: "port forward", args: s.freeRDPDesktopArguments("127.0.0.1:3389", "")},
		{name: "SOCKS5", args: s.freeRDPDesktopArguments("10.13.37.22:3389", "127.0.0.1:1080")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if count := countString(test.args, "-multitransport"); count != 1 {
				t.Fatalf("TCP-only multitransport disable count = %d, want 1", count)
			}
			if count := countString(test.args, "+auth-only"); count != 0 {
				t.Fatalf("redundant auth-only count = %d, want 0", count)
			}
			if count := countString(test.args, "/cert:ignore"); count != 0 {
				t.Fatalf("certificate-ignore count = %d, want 0", count)
			}
			certificateArgument := "/cert:deny,fingerprint:" + testRDPCertificateFingerprint
			if count := countString(test.args, certificateArgument); count != 1 {
				t.Fatalf("certificate pin count = %d, want 1", count)
			}
			if count := countString(test.args, "/server-name:win11-test.north.local"); count != 1 {
				t.Fatalf("RDP server-name count = %d, want 1", count)
			}
		})
	}
}

func TestTunnelAppEnvironmentDropsOperatorSecrets(t *testing.T) {
	environment := tunnelAppEnvironment([]string{
		"Path=/untrusted-first-path",
		"HOME=/tmp/e2e-home",
		"LANG=C.UTF-8",
		"PATH=/usr/bin",
		"TMPDIR=/tmp/e2e-tmp",
		"GH_TOKEN=github-secret",
		"SSH_AUTH_SOCK=/tmp/operator-agent.sock",
		"OP_SERVICE_ACCOUNT_TOKEN=one-password-secret",
		"HTTP_PROXY=http://proxy-user:proxy-password@example.invalid",
		"HTTPS_PROXY=https://proxy-user:proxy-password@example.invalid",
		"ALL_PROXY=socks5://proxy-user:proxy-password@example.invalid",
		"NO_PROXY=metadata.internal",
		"LD_PRELOAD=/tmp/hostile.so",
		"DYLD_INSERT_LIBRARIES=/tmp/hostile.dylib",
		"KRB5CCNAME=/tmp/operator-kerberos-cache",
		"XDG_CONFIG_HOME=/tmp/operator-config",
		"DISPLAY=:99",
		"XAUTHORITY=/tmp/operator-xauthority",
	})
	want := []string{"HOME=/tmp/e2e-home", "LANG=C.UTF-8", "PATH=/usr/bin", "TMPDIR=/tmp/e2e-tmp"}
	if strings.Join(environment, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("filtered tunnel app environment = %q, want deterministic %q", environment, want)
	}
	joined := strings.Join(environment, "\n")
	for _, secret := range []string{"GH_TOKEN", "github-secret", "SSH_AUTH_SOCK", "operator-agent.sock", "OP_SERVICE_ACCOUNT_TOKEN", "one-password-secret"} {
		if strings.Contains(joined, secret) {
			t.Errorf("filtered tunnel app environment retained %q: %q", secret, environment)
		}
	}
	for _, hostile := range []string{"PROXY", "proxy-password", "LD_PRELOAD", "DYLD_INSERT_LIBRARIES", "KRB5CCNAME", "XDG_CONFIG_HOME", "DISPLAY", "XAUTHORITY"} {
		if strings.Contains(joined, hostile) {
			t.Errorf("filtered tunnel app environment retained hostile host value %q: %q", hostile, environment)
		}
	}
}

func TestFreeRDPLineBufferedArguments(t *testing.T) {
	arguments := freeRDPLineBufferedArguments("/usr/bin/xfreerdp3")
	want := []string{"-oL", "-eL", "/usr/bin/xfreerdp3", "/args-from:fd:3"}
	if strings.Join(arguments, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("line-buffered FreeRDP arguments = %q, want %q", arguments, want)
	}
}

func countString(values []string, wanted string) int {
	count := 0
	for _, value := range values {
		if value == wanted {
			count++
		}
	}
	return count
}

func TestRequireFreeRDPDesktopActivation(t *testing.T) {
	valid := "CONNECTION_STATE_NEGO --> CONNECTION_STATE_NLA\n" +
		"[INFO][com.freerdp.gdi] - Local framebuffer format PIXEL_FORMAT_BGRX32\n" +
		"[INFO][com.freerdp.gdi] - Remote framebuffer format PIXEL_FORMAT_RGB16\n" +
		"FINALIZATION_CLIENT_FONT_MAP --> CONNECTION_STATE_ACTIVE\n"
	if err := requireFreeRDPDesktopActivation(valid); err != nil {
		t.Fatalf("valid FreeRDP desktop activation output: %v", err)
	}
	for _, output := range []string{
		"Local framebuffer format PIXEL_FORMAT_BGRX32",
		"CONNECTION_STATE_NEGO --> CONNECTION_STATE_NLA\nLocal framebuffer format\nRemote framebuffer format",
		"CONNECTION_STATE_NEGO --> CONNECTION_STATE_NLA\nLocal framebuffer format\n--> CONNECTION_STATE_ACTIVE",
		"--> CONNECTION_STATE_ACTIVE\nCONNECTION_STATE_NEGO --> CONNECTION_STATE_NLA\nLocal framebuffer format\nRemote framebuffer format",
		"CONNECTION_STATE_NEGO --> CONNECTION_STATE_NLA\nRemote framebuffer format\nLocal framebuffer format\n--> CONNECTION_STATE_ACTIVE",
		"",
	} {
		if err := requireFreeRDPDesktopActivation(output); err == nil {
			t.Fatalf("incomplete FreeRDP desktop output %q was accepted", output)
		}
	}
}

func TestFreeRDPActivationCaptureSignalsOnlyAfterAllMarkers(t *testing.T) {
	capture := newFreeRDPActivationCapture()
	if _, err := capture.Write([]byte("CONNECTION_STATE_NEGO --> CONNECTION_STATE_NLA\nLocal framebuffer for")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-capture.activated:
		t.Fatal("capture activated after only a partial marker")
	default:
	}
	if _, err := capture.Write([]byte("mat PIXEL_FORMAT_BGRX32\nRemote framebuffer format PIXEL_FORMAT_RGB16\n")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-capture.activated:
		t.Fatal("capture activated before the RDP state became active")
	default:
	}
	if _, err := capture.Write([]byte("FINALIZATION_CLIENT_FONT_MAP --> CONNECTION_STATE_ACTIVE\n")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-capture.activated:
	case <-time.After(time.Second):
		t.Fatal("capture did not signal after both activation markers")
	}
	if err := requireFreeRDPDesktopActivation(capture.String()); err != nil {
		t.Fatal(err)
	}
}

func TestFreeRDPActivationCaptureIsBoundedAndScansAcrossWrites(t *testing.T) {
	capture := newFreeRDPActivationCapture()
	largePrefix := bytes.Repeat([]byte("x"), freeRDPOutputCaptureBytes+8192)
	if count, err := capture.Write(largePrefix); err != nil || count != len(largePrefix) {
		t.Fatalf("large capture write = %d, %v", count, err)
	}
	for _, marker := range freeRDPDesktopActivationMarkers {
		split := len(marker) / 2
		if _, err := capture.Write([]byte(marker[:split])); err != nil {
			t.Fatal(err)
		}
		if _, err := capture.Write([]byte(marker[split:] + "\n")); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-capture.activated:
	case <-time.After(time.Second):
		t.Fatal("bounded capture did not detect split ordered markers")
	}
	output := capture.String()
	if !strings.HasPrefix(output, "[earlier FreeRDP output truncated]\n") {
		t.Fatal("bounded capture did not label truncated diagnostics")
	}
	if len(output) > freeRDPOutputCaptureBytes+64 {
		t.Fatalf("bounded capture diagnostics length = %d", len(output))
	}
}

func TestRDPNegotiationRoundTrip(t *testing.T) {
	server, client := net.Pipe()
	defer func() { _ = client.Close() }()
	serverDone := make(chan error, 1)
	go func() {
		defer func() { _ = server.Close() }()
		request := make([]byte, len(rdpNegotiationRequest))
		if _, err := io.ReadFull(server, request); err != nil {
			serverDone <- err
			return
		}
		if string(request) != string(rdpNegotiationRequest) {
			serverDone <- io.ErrUnexpectedEOF
			return
		}
		response := []byte{
			0x03, 0x00, 0x00, 0x13,
			0x0e, 0xd0, 0x00, 0x00, 0x12, 0x34, 0x00,
			0x02, 0x00, 0x08, 0x00, 0x02, 0x00, 0x00, 0x00,
		}
		_, err := server.Write(response)
		serverDone <- err
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result, err := rdpNegotiationRoundTrip(ctx, client)
	if err != nil {
		t.Fatal(err)
	}
	if result.responseBytes != 19 || result.negotiationType != 0x02 || result.selectedProtocol != 0x02 {
		t.Fatalf("RDP negotiation result = %+v", result)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}
}

func TestRDPNegotiationRejectsFailure(t *testing.T) {
	server, client := net.Pipe()
	defer func() { _ = client.Close() }()
	go func() {
		defer func() { _ = server.Close() }()
		request := make([]byte, len(rdpNegotiationRequest))
		_, _ = io.ReadFull(server, request)
		response := []byte{
			0x03, 0x00, 0x00, 0x13,
			0x0e, 0xd0, 0x00, 0x00, 0x12, 0x34, 0x00,
			0x03, 0x00, 0x08, 0x00, 0x05, 0x00, 0x00, 0x00,
		}
		binary.BigEndian.PutUint16(response[2:4], uint16(len(response)))
		_, _ = server.Write(response)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := rdpNegotiationRoundTrip(ctx, client); err == nil {
		t.Fatal("RDP negotiation failure response was accepted")
	}
}

func TestRDPNegotiationRejectsMissingOrUnrequestedSecurityResponse(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		response []byte
	}{
		{
			name: "missing negotiation response",
			response: []byte{
				0x03, 0x00, 0x00, 0x0b,
				0x06, 0xd0, 0x00, 0x00, 0x12, 0x34, 0x00,
			},
		},
		{
			name: "unrequested legacy protocol",
			response: []byte{
				0x03, 0x00, 0x00, 0x13,
				0x0e, 0xd0, 0x00, 0x00, 0x12, 0x34, 0x00,
				0x02, 0x00, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			server, client := net.Pipe()
			defer func() { _ = client.Close() }()
			go func() {
				defer func() { _ = server.Close() }()
				request := make([]byte, len(rdpNegotiationRequest))
				_, _ = io.ReadFull(server, request)
				_, _ = server.Write(testCase.response)
			}()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if _, err := rdpNegotiationRoundTrip(ctx, client); err == nil {
				t.Fatal("invalid RDP negotiation response was accepted")
			}
		})
	}
}
