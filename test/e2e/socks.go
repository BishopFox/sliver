package e2e

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"time"

	clientcore "github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/things-go/go-socks5/statute"
	"golang.org/x/net/proxy"
)

const (
	socksE2EBinarySeed      int64 = 0x51_1e_7e_25
	socksE2EMutationSeed    int64 = 0x50_c5_5e_ed
	socksE2EIdleDuration          = 21 * time.Second
	socksE2EMalformedCases        = 32
	socksE2EMaxFuzzCases          = 10_000
	socksE2ESequentialCount       = 4
	socksE2EConcurrentCount       = 8
	socksE2ECaseTimeout           = 10 * time.Second
)

var socksE2EBoundarySizes = []int{
	1, 2, 3,
	254, 255, 256, 257,
	4095, 4096, 4097,
	4107, 4108, 4109,
	32767, 32768, 32769,
	65535, 65536, 65537,
}

type socksE2EProxy struct {
	id       uint64
	address  string
	listener net.Listener
	done     chan error

	stopOnce sync.Once
	stopDone chan struct{}
	stopErr  error
}

type socksE2EScenario struct {
	name string
	run  func(context.Context) error
}

type socksE2EBinaryCase struct {
	Index   int
	Length  int
	Payload []byte
}

type socksE2EMutation struct {
	Index int
	Kind  string
	Data  []byte
}

// exerciseSocks5 drives the same client/core implementation used by the
// interactive socks5 command. The fixture services bind to target loopback, so
// the only path exercised by the test client is client -> teamserver -> implant
// -> target service.
//
//nolint:gocyclo // One lifecycle keeps all required SOCKS scenarios and cleanup accounting together.
func (s *suite) exerciseSocks5(target implantTarget, transport string) (resultErr error) {
	if target.session == nil || target.beacon != nil {
		return errors.New("SOCKS5 E2E requires an interactive session target")
	}
	s.t.Logf(
		"SOCKS5 live fuzz configuration: seed=%#x cases=%d replay=%d",
		s.opts.socksFuzzSeed,
		s.opts.socksFuzzCases,
		s.opts.socksFuzzCase,
	)

	echoServer, err := startTCPEchoServer()
	if err != nil {
		return fmt.Errorf("start SOCKS5 target echo service: %w", err)
	}
	defer func() {
		if closeErr := echoServer.close(); closeErr != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("close SOCKS5 target echo service: %w", closeErr))
		}
	}()

	httpBody := socksE2EHTTPFixtureBody()
	httpServer := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/socks-fixture" {
			http.NotFound(response, request)
			return
		}
		response.Header().Set("Content-Type", "application/octet-stream")
		response.Header().Set("X-Sliver-Socks-Fixture", "v1")
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write(httpBody)
	}))
	defer httpServer.Close()
	hostnameHTTPURL, err := socksE2EHostnameURL(httpServer.URL, "/socks-fixture")
	if err != nil {
		return err
	}

	echoAddress := echoServer.address()
	hostnameEchoAddress, err := socksE2EHostnameAddress(echoAddress)
	if err != nil {
		return err
	}

	scenarios := []socksE2EScenario{
		{name: "no-auth-ipv4-and-hostname", run: func(ctx context.Context) error {
			return s.socksE2ENoAuthAddressing(ctx, target.session, echoAddress, hostnameEchoAddress)
		}},
		{name: "boundary-and-random-binary", run: func(ctx context.Context) error {
			return s.socksE2EBinaryTraffic(ctx, target.session, echoAddress)
		}},
		{name: "sustained-full-duplex", run: func(ctx context.Context) error {
			return s.socksE2EFullDuplex(ctx, target.session, echoAddress)
		}},
		{name: "sequential-and-concurrent", run: func(ctx context.Context) error {
			return s.socksE2EConnectionConcurrency(ctx, target.session, echoAddress)
		}},
		{name: "go-http-and-curl", run: func(ctx context.Context) error {
			return s.socksE2EHTTPClients(ctx, target.session, hostnameHTTPURL, httpBody)
		}},
		{name: "idle-resume", run: func(ctx context.Context) error {
			return s.socksE2EIdleResume(ctx, target.session, echoAddress)
		}},
		{name: "malformed-recovery-and-ping", run: func(ctx context.Context) error {
			return s.socksE2EMalformedRecovery(ctx, target, echoAddress, hostnameEchoAddress)
		}},
		{name: "stop-and-restart", run: func(ctx context.Context) error {
			return s.socksE2EStopAndRestart(ctx, target.session, echoAddress, echoServer)
		}},
		{name: "two-proxy-stop-isolation", run: func(ctx context.Context) error {
			return s.socksE2ETwoProxyStopIsolation(ctx, target.session, echoAddress)
		}},
		{name: "authenticated-success-and-wrong-password", run: func(ctx context.Context) error {
			return s.socksE2EAuthentication(ctx, target.session, echoAddress)
		}},
		{name: "auth-no-auth-proxy-isolation", run: func(ctx context.Context) error {
			return s.socksE2EAuthModeIsolation(ctx, target.session, echoAddress)
		}},
	}
	if s.opts.tunnelHTTPURL != "" {
		scenarios = append(scenarios, socksE2EScenario{name: "real-http-curl", run: func(ctx context.Context) error {
			return s.socksE2EExternalHTTP(ctx, target.session, transport)
		}})
	}
	if s.opts.tunnelRDPAddr != "" {
		scenarios = append(scenarios, socksE2EScenario{name: "rdp-negotiation", run: func(ctx context.Context) error {
			return s.socksE2ERDP(ctx, target.session, transport)
		}})
	}

	var scenarioErrors []error
	for _, scenario := range scenarios {
		started := time.Now()
		scenarioTimeout := s.opts.commandTimeout
		if scenario.name == "idle-resume" && scenarioTimeout < socksE2EIdleDuration+socksE2ECaseTimeout {
			scenarioTimeout = socksE2EIdleDuration + socksE2ECaseTimeout
		}
		if scenario.name == "malformed-recovery-and-ping" {
			fuzzTimeout := socksE2EMalformedScenarioTimeout(s.opts.socksFuzzCases, s.opts.socksFuzzCase)
			if scenarioTimeout < fuzzTimeout {
				scenarioTimeout = fuzzTimeout
			}
		}
		ctx, cancel := context.WithTimeout(s.ctx, scenarioTimeout)
		err := scenario.run(ctx)
		cancel()
		duration := time.Since(started).Round(time.Millisecond)
		s.recordTunnelScenario(transport, "socks5", scenario.name, duration, err)
		if err != nil {
			scenarioErrors = append(scenarioErrors, fmt.Errorf("SOCKS5 %s/%s (%s): %w", transport, scenario.name, duration, err))
			continue
		}
		s.t.Logf("PASS %s/%s %s session SOCKS5/%s (%s)", s.opts.targetOS, s.opts.targetArch, transport, scenario.name, duration)
	}
	return errors.Join(scenarioErrors...)
}

func (s *suite) socksE2EFullDuplex(ctx context.Context, session *clientpb.Session, destination string) error {
	return s.withSocksE2EProxy(ctx, session, "", "", func(proxyServer *socksE2EProxy) error {
		connection, err := socksE2EDial(ctx, proxyServer.address, "", "", destination)
		if err != nil {
			return err
		}
		defer func() { _ = connection.Close() }()
		elapsed, err := tunnelFullDuplexEchoOnConn(
			ctx,
			connection,
			"socks-sustained-full-duplex",
			tunnelFullDuplexPayloadBytes,
			tunnelFullDuplexMinimumBytesPerSecond,
		)
		if err != nil {
			return err
		}
		s.t.Logf("SOCKS5 full-duplex: %d bytes each direction in %s", tunnelFullDuplexPayloadBytes, elapsed.Round(time.Millisecond))
		return nil
	})
}

func (s *suite) startSocksE2EProxy(session *clientpb.Session, bindAddress string, username string, password string) (*socksE2EProxy, error) {
	if session == nil {
		return nil, errors.New("start SOCKS5 proxy without a session")
	}
	if bindAddress == "" {
		bindAddress = "127.0.0.1:0"
	}
	listener, err := net.Listen("tcp4", bindAddress)
	if err != nil {
		return nil, fmt.Errorf("listen for SOCKS5 proxy on %s: %w", bindAddress, err)
	}
	tcpProxy := &clientcore.TcpProxy{
		Rpc:             s.rpc,
		Session:         session,
		Username:        username,
		Password:        password,
		BindAddr:        listener.Addr().String(),
		Listener:        listener,
		KeepAlivePeriod: 30 * time.Second,
		DialTimeout:     30 * time.Second,
	}
	registered := clientcore.SocksProxies.Add(tcpProxy)
	result := &socksE2EProxy{
		id:       registered.ID,
		address:  listener.Addr().String(),
		listener: listener,
		done:     make(chan error, 1),
		stopDone: make(chan struct{}),
	}
	go func() {
		result.done <- clientcore.SocksProxies.Start(tcpProxy)
	}()

	// A completed Start at this point means the production proxy failed before
	// it could accept traffic. The listener itself was created synchronously.
	select {
	case startErr := <-result.done:
		_ = listener.Close()
		_ = clientcore.SocksProxies.Remove(result.id)
		if startErr == nil {
			startErr = errors.New("SOCKS5 proxy stopped during startup")
		}
		return nil, startErr
	default:
	}
	return result, nil
}

func (proxyServer *socksE2EProxy) stop(ctx context.Context) error {
	proxyServer.stopOnce.Do(func() {
		if proxyServer.stopDone == nil {
			proxyServer.stopDone = make(chan struct{})
		}
		go func() {
			if !clientcore.SocksProxies.Remove(proxyServer.id) {
				select {
				case startErr := <-proxyServer.done:
					proxyServer.stopErr = errors.Join(
						fmt.Errorf("SOCKS5 proxy %d was not registered during stop", proxyServer.id),
						startErr,
					)
				default:
					proxyServer.stopErr = fmt.Errorf("SOCKS5 proxy %d was not registered during stop", proxyServer.id)
				}
				close(proxyServer.stopDone)
				return
			}
			proxyServer.stopErr = <-proxyServer.done
			close(proxyServer.stopDone)
		}()
	})
	select {
	case <-proxyServer.stopDone:
		return proxyServer.stopErr
	case <-ctx.Done():
		return fmt.Errorf("wait for SOCKS5 proxy %d stop: %w", proxyServer.id, ctx.Err())
	}
}

func (s *suite) withSocksE2EProxy(_ context.Context, session *clientpb.Session, username string, password string, run func(*socksE2EProxy) error) (resultErr error) {
	proxyServer, err := s.startSocksE2EProxy(session, "", username, password)
	if err != nil {
		return err
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), socksE2ECaseTimeout)
		defer cancel()
		resultErr = errors.Join(resultErr, proxyServer.stop(stopCtx))
	}()
	return run(proxyServer)
}

func socksE2EContextDialer(proxyAddress string, username string, password string) (proxy.ContextDialer, error) {
	var authentication *proxy.Auth
	if username != "" || password != "" {
		authentication = &proxy.Auth{User: username, Password: password}
	}
	dialer, err := proxy.SOCKS5("tcp", proxyAddress, authentication, &net.Dialer{Timeout: socksE2ECaseTimeout})
	if err != nil {
		return nil, fmt.Errorf("construct SOCKS5 dialer for %s: %w", proxyAddress, err)
	}
	contextDialer, ok := dialer.(proxy.ContextDialer)
	if !ok {
		return nil, errors.New("SOCKS5 dialer does not implement proxy.ContextDialer")
	}
	return contextDialer, nil
}

func socksE2EDial(ctx context.Context, proxyAddress string, username string, password string, destination string) (net.Conn, error) {
	dialer, err := socksE2EContextDialer(proxyAddress, username, password)
	if err != nil {
		return nil, err
	}
	connection, err := dialer.DialContext(ctx, "tcp", destination)
	if err != nil {
		return nil, fmt.Errorf("SOCKS5 %s -> %s: %w", proxyAddress, destination, err)
	}
	return connection, nil
}

func socksE2ERoundTrip(ctx context.Context, proxyAddress string, username string, password string, destination string, payload []byte, chunks []int) error {
	connection, err := socksE2EDial(ctx, proxyAddress, username, password, destination)
	if err != nil {
		return err
	}
	defer func() { _ = connection.Close() }()
	return tunnelEchoRoundTripOnConn(ctx, connection, payload, chunks)
}

func requireSocksE2EAuthRejection(ctx context.Context, proxyAddress string, username string, password string) error {
	connection, err := (&net.Dialer{Timeout: socksE2ECaseTimeout}).DialContext(ctx, "tcp4", proxyAddress)
	if err != nil {
		return fmt.Errorf("connect to authenticated SOCKS5 proxy %s: %w", proxyAddress, err)
	}
	defer func() { _ = connection.Close() }()
	deadline := time.Now().Add(socksE2ECaseTimeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err := connection.SetDeadline(deadline); err != nil {
		return err
	}
	return requireSocksE2EAuthRejectionOnConn(connection, username, password)
}

func requireSocksE2EAuthRejectionOnConn(connection net.Conn, username string, password string) error {
	if len(username) == 0 || len(username) > 255 || len(password) == 0 || len(password) > 255 {
		return errors.New("SOCKS5 rejection probe credentials must contain 1 through 255 bytes")
	}
	if err := writeTunnelChunks(connection, []byte{0x05, 0x01, 0x02}, []int{3}); err != nil {
		return fmt.Errorf("write SOCKS5 authentication method greeting: %w", err)
	}
	methodReply := make([]byte, 2)
	if _, err := io.ReadFull(connection, methodReply); err != nil {
		return fmt.Errorf("read SOCKS5 authentication method reply: %w", err)
	}
	if !bytes.Equal(methodReply, []byte{0x05, 0x02}) {
		return fmt.Errorf("SOCKS5 proxy selected authentication method %x, want username/password", methodReply)
	}
	authRequest := make([]byte, 0, 3+len(username)+len(password))
	authRequest = append(authRequest, 0x01, byte(len(username)))
	authRequest = append(authRequest, username...)
	authRequest = append(authRequest, byte(len(password)))
	authRequest = append(authRequest, password...)
	if err := writeTunnelChunks(connection, authRequest, []int{len(authRequest)}); err != nil {
		return fmt.Errorf("write SOCKS5 username/password request: %w", err)
	}
	authReply := make([]byte, 2)
	if _, err := io.ReadFull(connection, authReply); err != nil {
		return fmt.Errorf("read SOCKS5 username/password reply: %w", err)
	}
	if authReply[0] != 0x01 {
		return fmt.Errorf("SOCKS5 username/password reply version = %#x, want 0x1", authReply[0])
	}
	if authReply[1] == 0x00 {
		return errors.New("SOCKS5 proxy explicitly accepted an incorrect password")
	}
	return nil
}

func (s *suite) socksE2ENoAuthAddressing(ctx context.Context, session *clientpb.Session, ipv4Address string, hostnameAddress string) error {
	return s.withSocksE2EProxy(ctx, session, "", "", func(proxyServer *socksE2EProxy) error {
		ipv4Err := socksE2ERoundTrip(ctx, proxyServer.address, "", "", ipv4Address, []byte("socks-ipv4\x00\xff"), []int{1, 3, 2})
		hostnameErr := socksE2ERoundTrip(ctx, proxyServer.address, "", "", hostnameAddress, []byte("socks-hostname\x00\xff"), []int{2, 1, 7})
		return errors.Join(ipv4Err, hostnameErr)
	})
}

func socksE2EBinaryCases() []socksE2EBinaryCase {
	cases := make([]socksE2EBinaryCase, 0, len(socksE2EBoundarySizes)+8)
	for index, size := range socksE2EBoundarySizes {
		payload := deterministicTunnelPayload(fmt.Sprintf("socks-boundary-%d-%d", index, size), size)
		payload[0] = 0
		if len(payload) > 1 {
			payload[len(payload)-1] = 0xff
		}
		cases = append(cases, socksE2EBinaryCase{Index: index, Length: size, Payload: payload})
	}
	randomSource := rand.New(rand.NewSource(socksE2EBinarySeed))
	for index := 0; index < 8; index++ {
		size := 1 + randomSource.Intn(96*1024)
		cases = append(cases, socksE2EBinaryCase{
			Index:   len(cases),
			Length:  size,
			Payload: deterministicTunnelPayload(fmt.Sprintf("socks-random-%d-%d", index, size), size),
		})
	}
	return cases
}

func (s *suite) socksE2EBinaryTraffic(ctx context.Context, session *clientpb.Session, destination string) error {
	return s.withSocksE2EProxy(ctx, session, "", "", func(proxyServer *socksE2EProxy) error {
		for _, testCase := range socksE2EBinaryCases() {
			chunks := []int{1, 7, 257, 4095, 4108, 8192}
			if err := socksE2ERoundTrip(ctx, proxyServer.address, "", "", destination, testCase.Payload, chunks); err != nil {
				return fmt.Errorf("binary seed=%#x case=%d length=%d chunks=%v: %w", socksE2EBinarySeed, testCase.Index, testCase.Length, chunks, err)
			}
		}
		return nil
	})
}

func (s *suite) socksE2EConnectionConcurrency(ctx context.Context, session *clientpb.Session, destination string) error {
	return s.withSocksE2EProxy(ctx, session, "", "", func(proxyServer *socksE2EProxy) error {
		for index := 0; index < socksE2ESequentialCount; index++ {
			payload := deterministicTunnelPayload(fmt.Sprintf("socks-sequential-%d", index), 2048+index*911)
			if err := socksE2ERoundTrip(ctx, proxyServer.address, "", "", destination, payload, []int{17, 1, 1024}); err != nil {
				return fmt.Errorf("sequential connection %d/%d: %w", index+1, socksE2ESequentialCount, err)
			}
		}

		start := make(chan struct{})
		errorsByConnection := make(chan error, socksE2EConcurrentCount)
		var workers sync.WaitGroup
		for index := 0; index < socksE2EConcurrentCount; index++ {
			index := index
			workers.Add(1)
			go func() {
				defer workers.Done()
				<-start
				payload := deterministicTunnelPayload(fmt.Sprintf("socks-concurrent-%d", index), 4096+index*733)
				if err := socksE2ERoundTrip(ctx, proxyServer.address, "", "", destination, payload, []int{1 + index, 511, 4096}); err != nil {
					errorsByConnection <- fmt.Errorf("concurrent connection %d/%d: %w", index+1, socksE2EConcurrentCount, err)
				}
			}()
		}
		close(start)
		workers.Wait()
		close(errorsByConnection)
		var connectionErrors []error
		for connectionErr := range errorsByConnection {
			connectionErrors = append(connectionErrors, connectionErr)
		}
		return errors.Join(connectionErrors...)
	})
}

func socksE2EHTTPFixtureBody() []byte {
	body := deterministicTunnelPayload("socks-http-fixture", 64*1024+17)
	digest := sha256.Sum256(body)
	return append(body, []byte("\nsha256="+hex.EncodeToString(digest[:])+"\n")...)
}

func socksE2EHostnameURL(rawURL string, path string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse SOCKS HTTP fixture URL: %w", err)
	}
	port := parsed.Port()
	if port == "" {
		return "", fmt.Errorf("SOCKS HTTP fixture URL %q has no port", rawURL)
	}
	parsed.Host = net.JoinHostPort("localhost", port)
	parsed.Path = path
	return parsed.String(), nil
}

func socksE2EHostnameAddress(address string) (string, error) {
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", fmt.Errorf("parse SOCKS destination %q: %w", address, err)
	}
	return net.JoinHostPort("localhost", port), nil
}

func (s *suite) socksE2EHTTPClients(ctx context.Context, session *clientpb.Session, targetURL string, wantBody []byte) error {
	return s.withSocksE2EProxy(ctx, session, "", "", func(proxyServer *socksE2EProxy) error {
		dialer, err := socksE2EContextDialer(proxyServer.address, "", "")
		if err != nil {
			return err
		}
		transport := &http.Transport{Proxy: nil, DialContext: dialer.DialContext, DisableKeepAlives: true}
		defer transport.CloseIdleConnections()
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
		if err != nil {
			return err
		}
		response, err := (&http.Client{Transport: transport}).Do(request)
		if err != nil {
			return fmt.Errorf("request through SOCKS5 with Go client: %w", err)
		}
		body, readErr := io.ReadAll(response.Body)
		closeErr := response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected Go HTTP status through SOCKS5: %s", response.Status)
		}
		if err := errors.Join(readErr, closeErr); err != nil {
			return fmt.Errorf("read Go HTTP SOCKS5 response: %w", err)
		}
		if !bytes.Equal(body, wantBody) {
			return fmt.Errorf("body mismatch through SOCKS5 Go HTTP client: got %d bytes, want %d", len(body), len(wantBody))
		}

		curlCtx, cancel := context.WithTimeout(ctx, s.opts.commandTimeout)
		defer cancel()
		output, err := s.curlTunnelHTTP(curlCtx, tunnelHTTPCurlRequest{
			targetURL:  targetURL,
			noProxy:    "",
			socksProxy: proxyServer.address,
		})
		if err != nil {
			return fmt.Errorf("curl through SOCKS5: %w", err)
		}
		if !bytes.Equal(output, wantBody) {
			return fmt.Errorf("curl SOCKS5 body mismatch: got %d bytes, want %d", len(output), len(wantBody))
		}
		return nil
	})
}

func (s *suite) socksE2EIdleResume(ctx context.Context, session *clientpb.Session, destination string) error {
	return s.withSocksE2EProxy(ctx, session, "", "", func(proxyServer *socksE2EProxy) error {
		connection, err := socksE2EDial(ctx, proxyServer.address, "", "", destination)
		if err != nil {
			return err
		}
		defer func() { _ = connection.Close() }()
		if err := tunnelEchoRoundTripOnConn(ctx, connection, []byte("socks-before-idle\x00\xff"), []int{1, 5, 2}); err != nil {
			return fmt.Errorf("exchange before idle: %w", err)
		}
		if err := connection.SetDeadline(time.Time{}); err != nil {
			return fmt.Errorf("clear SOCKS5 idle deadline: %w", err)
		}
		timer := time.NewTimer(socksE2EIdleDuration)
		select {
		case <-timer.C:
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		}
		return tunnelEchoRoundTripOnConn(ctx, connection, deterministicTunnelPayload("socks-after-idle", 32*1024+3), []int{4095, 1, 8192})
	})
}

func socksE2EMutations(seed int64, caseCount int) []socksE2EMutation {
	if caseCount <= 0 {
		return nil
	}
	seeds := []socksE2EMutation{
		{Kind: "empty"},
		{Kind: "truncated-version", Data: []byte{0x05}},
		{Kind: "zero-methods", Data: []byte{0x05, 0x00}},
		{Kind: "wrong-version", Data: []byte{0x04, 0x01, 0x00}},
		{Kind: "unsupported-method", Data: []byte{0x05, 0x01, 0xff}},
		{Kind: "truncated-request", Data: []byte{0x05, 0x01, 0x00, 0x05, 0x01}},
		{Kind: "unsupported-command", Data: []byte{0x05, 0x01, 0x00, 0x05, 0x7f, 0x00, 0x01, 127, 0, 0, 1, 0, 1}},
		{Kind: "bad-address-type", Data: []byte{0x05, 0x01, 0x00, 0x05, 0x01, 0x00, 0x7f}},
		{Kind: "truncated-domain", Data: []byte{0x05, 0x01, 0x00, 0x05, 0x01, 0x00, 0x03, 0xff, 'a'}},
		{Kind: "truncated-ipv6", Data: []byte{0x05, 0x01, 0x00, 0x05, 0x01, 0x00, 0x04, 0, 1}},
		// Complete BIND/UDP requests belong in the isolated implant unit harness.
		// Keeping the live cases truncated guarantees a rule regression cannot
		// create a listener or UDP association on the E2E driver host.
		{Kind: "truncated-bind", Data: []byte{0x05, 0x01, 0x00, 0x05, 0x02, 0x00, 0x01}},
		{Kind: "truncated-udp-associate", Data: []byte{0x05, 0x01, 0x00, 0x05, 0x03, 0x00, 0x01}},
	}
	mutations := make([]socksE2EMutation, 0, caseCount)
	for _, seed := range seeds {
		if len(mutations) == caseCount {
			return mutations
		}
		seed.Index = len(mutations)
		seed.Data = append([]byte(nil), seed.Data...)
		mutations = append(mutations, seed)
	}
	randomSource := rand.New(rand.NewSource(seed))
	for len(mutations) < caseCount {
		base := seeds[randomSource.Intn(len(seeds))]
		data := append([]byte(nil), base.Data...)
		switch randomSource.Intn(4) {
		case 0:
			if len(data) == 0 {
				data = append(data, byte(randomSource.Intn(256)))
			} else {
				data[randomSource.Intn(len(data))] ^= byte(1 + randomSource.Intn(255))
			}
		case 1:
			if len(data) > 0 {
				data = data[:randomSource.Intn(len(data))]
			}
		case 2:
			data = append(data, byte(randomSource.Intn(256)), byte(randomSource.Intn(256)))
		case 3:
			length := 1 + randomSource.Intn(48)
			data = make([]byte, length)
			_, _ = randomSource.Read(data)
			// Raw random input must never accidentally become an accepted
			// SOCKS5 greeting followed by an arbitrary network destination.
			data[0] = 0x04
		}
		if err := validateSocksE2EMutationEgress(data); err != nil {
			// Keep the deterministic mutation while making it unambiguously an
			// invalid protocol version. The live fuzz corpus is a parser/lifecycle
			// test and must never expand its configured loopback target scope.
			if len(data) == 0 {
				data = []byte{0x04}
			} else {
				data[0] = 0x04
			}
		}
		mutations = append(mutations, socksE2EMutation{Index: len(mutations), Kind: "mutated-" + base.Kind, Data: data})
	}
	return mutations
}

// validateSocksE2EMutationEgress proves that a malformed-input case cannot
// direct the live SOCKS server outside the suite's loopback fixtures. Truncated
// handshakes and requests are safe because they cannot reach a dial. A complete
// request that can perform a network operation is rejected. BIND and UDP
// ASSOCIATE are exercised by the network-isolated implant unit harness, while
// valid CONNECT recovery is driven separately against an exact owned fixture.
func validateSocksE2EMutationEgress(data []byte) error {
	if len(data) < 2 || data[0] != statute.VersionSocks5 {
		return nil
	}
	methodCount := int(data[1])
	requestOffset := 2 + methodCount
	if requestOffset > len(data) {
		return nil
	}
	noAuthentication := false
	for _, method := range data[2:requestOffset] {
		if method == statute.MethodNoAuth {
			noAuthentication = true
			break
		}
	}
	if !noAuthentication {
		return nil
	}
	request, err := statute.ParseRequest(bytes.NewReader(data[requestOffset:]))
	if err != nil {
		return nil
	}
	switch request.Command {
	case statute.CommandConnect, statute.CommandBind, statute.CommandAssociate:
		return fmt.Errorf("mutation contains complete network command %#x", request.Command)
	default:
		return nil
	}
}

//nolint:gocyclo // The mutation switch intentionally encodes the complete bounded wire corpus in one sender.
func sendSocksE2EMalformedCase(ctx context.Context, proxyAddress string, mutation socksE2EMutation) error {
	if err := validateSocksE2EMutationEgress(mutation.Data); err != nil {
		return fmt.Errorf("refuse unsafe live SOCKS5 mutation: %w", err)
	}
	connection, err := (&net.Dialer{Timeout: socksE2ECaseTimeout}).DialContext(ctx, "tcp4", proxyAddress)
	if err != nil {
		return err
	}
	defer func() { _ = connection.Close() }()
	deadline := time.Now().Add(2 * time.Second)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err := connection.SetDeadline(deadline); err != nil {
		return err
	}
	if len(mutation.Data) > 0 {
		if _, err := connection.Write(mutation.Data); err != nil {
			if networkErr, ok := err.(net.Error); ok && networkErr.Timeout() {
				return fmt.Errorf("write timed out: %w", err)
			}
			return nil
		}
	}
	const maximumResponseBytes = 256
	response := make([]byte, 0, 32)
	buffer := make([]byte, 64)
	for len(response) < maximumResponseBytes {
		count, readErr := connection.Read(buffer)
		if count > 0 {
			response = append(response, buffer[:count]...)
			if socksE2EReplyContains(response, 0x00) {
				return fmt.Errorf("malformed SOCKS5 case %s received a successful CONNECT reply: %x", mutation.Kind, response)
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) || errors.Is(readErr, net.ErrClosed) {
				return requireSocksE2EMalformedResponse(mutation, response)
			}
			if networkErr, ok := readErr.(net.Error); ok && networkErr.Timeout() {
				// A truncated request may remain pending until the client closes;
				// the per-case deadline proves the parser is bounded.
				return requireSocksE2EMalformedResponse(mutation, response)
			}
			// A reset or parser-side transport error is an acceptable rejection,
			// but any response already received still must satisfy its oracle.
			return requireSocksE2EMalformedResponse(mutation, response)
		}
		if count == 0 {
			return io.ErrNoProgress
		}
	}
	return fmt.Errorf("malformed SOCKS5 case returned at least %d bytes", maximumResponseBytes)
}

func requireSocksE2EMalformedResponse(mutation socksE2EMutation, response []byte) error {
	if socksE2EReplyContains(response, 0x00) {
		return fmt.Errorf("malformed SOCKS5 case %s received a successful CONNECT reply: %x", mutation.Kind, response)
	}
	var expectedReply byte
	switch mutation.Kind {
	case "unsupported-method":
		if !bytes.Contains(response, []byte{0x05, 0xff}) {
			return fmt.Errorf("unsupported SOCKS5 method response = %x, want 05ff", response)
		}
		return nil
	case "unsupported-command":
		expectedReply = 0x07
	case "bad-address-type":
		expectedReply = 0x08
	default:
		return nil
	}
	if !socksE2EReplyContains(response, expectedReply) {
		return fmt.Errorf("SOCKS5 case %s response = %x, want reply code %#x", mutation.Kind, response, expectedReply)
	}
	return nil
}

func socksE2EReplyContains(response []byte, replyCode byte) bool {
	for index := 0; index+4 <= len(response); index++ {
		if response[index] == 0x05 && response[index+1] == replyCode && response[index+2] == 0x00 {
			switch response[index+3] {
			case 0x01, 0x03, 0x04:
				return true
			}
		}
	}
	return false
}

func waitForSocksE2EClientTunnelDrain(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		count := 0
		clientcore.SocksConnPool.Range(func(_, _ any) bool {
			count++
			return true
		})
		if count == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for %d SOCKS5 client tunnel(s) to drain: %w", count, ctx.Err())
		case <-ticker.C:
		}
	}
}

func (s *suite) socksE2EMalformedRecovery(ctx context.Context, target implantTarget, ipv4Destination string, hostnameDestination string) error {
	return s.withSocksE2EProxy(ctx, target.session, "", "", func(proxyServer *socksE2EProxy) error {
		generatedCases := socksE2EGeneratedCaseCount(s.opts.socksFuzzCases, s.opts.socksFuzzCase)
		for _, mutation := range socksE2EMutations(s.opts.socksFuzzSeed, generatedCases) {
			if s.opts.socksFuzzCase >= 0 && mutation.Index != s.opts.socksFuzzCase {
				continue
			}
			if err := sendSocksE2EMalformedCase(ctx, proxyServer.address, mutation); err != nil {
				return fmt.Errorf("mutation seed=%#x case=%d kind=%s bytes=%x: %w", s.opts.socksFuzzSeed, mutation.Index, mutation.Kind, mutation.Data, err)
			}
			drainCtx, drainCancel := context.WithTimeout(ctx, socksE2ECaseTimeout)
			drainErr := waitForSocksE2EClientTunnelDrain(drainCtx)
			drainCancel()
			if drainErr != nil {
				return fmt.Errorf("mutation seed=%#x case=%d kind=%s leaked a client tunnel: %w", s.opts.socksFuzzSeed, mutation.Index, mutation.Kind, drainErr)
			}
			if s.opts.socksFuzzCase >= 0 || (mutation.Index+1)%8 == 0 {
				destination := ipv4Destination
				if mutation.Index%16 != 0 {
					destination = hostnameDestination
				}
				payload := deterministicTunnelPayload(fmt.Sprintf("socks-fuzz-recovery-%d", mutation.Index), 2048+mutation.Index)
				if err := socksE2ERoundTrip(ctx, proxyServer.address, "", "", destination, payload, []int{1, 257, 4096}); err != nil {
					return fmt.Errorf("post-mutation recovery seed=%#x after case=%d: %w", s.opts.socksFuzzSeed, mutation.Index, err)
				}
			}
		}
		if err := s.requireTunnelSessionPing(target); err != nil {
			return fmt.Errorf("session Ping after malformed SOCKS5 corpus: %w", err)
		}
		return nil
	})
}

func socksE2EGeneratedCaseCount(caseCount int, replayCase int) int {
	if replayCase >= caseCount {
		return replayCase + 1
	}
	return caseCount
}

func socksE2EMalformedScenarioTimeout(caseCount int, replayCase int) time.Duration {
	executedCases := caseCount
	if replayCase >= 0 {
		executedCases = 1
	}
	if executedCases < 1 {
		executedCases = 1
	}
	recoveryChecks := (executedCases + 7) / 8
	if replayCase >= 0 {
		recoveryChecks = 1
	}
	// Every mutation can consume its two-second parser deadline and the full
	// client-tunnel drain deadline. Recovery round trips have their own case
	// deadline. Account for all three so the outer scenario cannot expire before
	// an inner bounded operation reports the reproducible case that failed.
	return time.Duration(executedCases)*(2*time.Second+socksE2ECaseTimeout) +
		time.Duration(recoveryChecks)*socksE2ECaseTimeout + 30*time.Second
}

func findSocksE2EMetadata(id uint64) *clientcore.SocksProxyMeta {
	for _, metadata := range clientcore.SocksProxies.List() {
		if metadata.ID == id {
			return metadata
		}
	}
	return nil
}

func (s *suite) socksE2EStopAndRestart(ctx context.Context, session *clientpb.Session, destination string, echoServer *tcpEchoServer) (resultErr error) {
	first, err := s.startSocksE2EProxy(session, "", "", "")
	if err != nil {
		return err
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), socksE2ECaseTimeout)
		defer cancel()
		resultErr = errors.Join(resultErr, first.stop(stopCtx))
	}()
	connection, err := socksE2EDial(ctx, first.address, "", "", destination)
	if err != nil {
		return err
	}
	defer func() { _ = connection.Close() }()
	if err := tunnelEchoRoundTripOnConn(ctx, connection, []byte("socks-before-stop"), []int{3, 1, 8}); err != nil {
		return err
	}
	if err := connection.SetDeadline(time.Time{}); err != nil {
		return err
	}
	if err := waitForEchoConnectionCount(ctx, echoServer, 1); err != nil {
		return fmt.Errorf("active SOCKS5 target relay readiness: %w", err)
	}
	if err := first.stop(ctx); err != nil {
		return err
	}
	if err := requireTunnelConnectionClosed(ctx, connection, "SOCKS5 proxy stop"); err != nil {
		return err
	}
	if err := waitForEchoConnectionCount(ctx, echoServer, 0); err != nil {
		return fmt.Errorf("SOCKS5 target relay after proxy stop: %w", err)
	}
	if err := requireTCPDialRejection(ctx, first.address); err != nil {
		return err
	}
	if findSocksE2EMetadata(first.id) != nil {
		return fmt.Errorf("stopped SOCKS5 proxy %d remains in inventory", first.id)
	}
	return s.withSocksE2EProxy(ctx, session, "", "", func(second *socksE2EProxy) error {
		return socksE2ERoundTrip(ctx, second.address, "", "", destination, []byte("socks-after-restart"), []int{1, 4, 2})
	})
}

func (s *suite) socksE2ETwoProxyStopIsolation(ctx context.Context, session *clientpb.Session, destination string) (resultErr error) {
	first, err := s.startSocksE2EProxy(session, "", "", "")
	if err != nil {
		return err
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), socksE2ECaseTimeout)
		defer cancel()
		resultErr = errors.Join(resultErr, first.stop(stopCtx))
	}()
	second, err := s.startSocksE2EProxy(session, "", "", "")
	if err != nil {
		return err
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), socksE2ECaseTimeout)
		defer cancel()
		resultErr = errors.Join(resultErr, second.stop(stopCtx))
	}()

	connection, err := socksE2EDial(ctx, second.address, "", "", destination)
	if err != nil {
		return err
	}
	defer func() { _ = connection.Close() }()
	if err := tunnelEchoRoundTripOnConn(ctx, connection, []byte("socks-isolation-before-stop"), []int{5, 1, 9}); err != nil {
		return err
	}
	if err := first.stop(ctx); err != nil {
		return fmt.Errorf("stop first isolated SOCKS5 proxy: %w", err)
	}
	if err := connection.SetDeadline(time.Time{}); err != nil {
		return err
	}
	if err := tunnelEchoRoundTripOnConn(ctx, connection, []byte("socks-isolation-after-stop"), []int{1, 7, 3}); err != nil {
		return fmt.Errorf("second proxy connection did not survive first proxy stop: %w", err)
	}
	return nil
}

func (s *suite) socksE2EAuthentication(ctx context.Context, session *clientpb.Session, destination string) error {
	const username = "sliver-e2e-user"
	const password = "sliver-e2e-password"
	return s.withSocksE2EProxy(ctx, session, username, password, func(proxyServer *socksE2EProxy) error {
		if err := socksE2ERoundTrip(ctx, proxyServer.address, username, password, destination, []byte("socks-auth-success"), []int{1, 8}); err != nil {
			return err
		}
		if err := requireSocksE2EAuthRejection(ctx, proxyServer.address, username, "wrong-password"); err != nil {
			return fmt.Errorf("verify explicit SOCKS5 wrong-password rejection: %w", err)
		}
		return socksE2ERoundTrip(
			ctx,
			proxyServer.address,
			username,
			password,
			destination,
			[]byte("socks-auth-recovery"),
			[]int{2, 1, 7},
		)
	})
}

func (s *suite) socksE2EAuthModeIsolation(ctx context.Context, session *clientpb.Session, destination string) (resultErr error) {
	const username = "sliver-e2e-isolation"
	const password = "sliver-e2e-isolation-password"
	authenticated, err := s.startSocksE2EProxy(session, "", username, password)
	if err != nil {
		return err
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), socksE2ECaseTimeout)
		defer cancel()
		resultErr = errors.Join(resultErr, authenticated.stop(stopCtx))
	}()
	noAuth, err := s.startSocksE2EProxy(session, "", "", "")
	if err != nil {
		return err
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), socksE2ECaseTimeout)
		defer cancel()
		resultErr = errors.Join(resultErr, noAuth.stop(stopCtx))
	}()
	authErr := socksE2ERoundTrip(ctx, authenticated.address, username, password, destination, []byte("socks-auth-isolated"), []int{1, 4})
	noAuthErr := socksE2ERoundTrip(ctx, noAuth.address, "", "", destination, []byte("socks-no-auth-isolated"), []int{2, 3})
	return errors.Join(authErr, noAuthErr)
}
