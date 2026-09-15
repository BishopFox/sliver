package e2e

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	clientcore "github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/tcpproxy"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

const (
	portForwardSequentialConnections = 4
	portForwardConcurrentConnections = 6
	portForwardRandomizedCases       = 12
	portForwardIdleDuration          = 6 * time.Second
	portForwardStopTimeout           = 10 * time.Second
	portForwardDialFailureTimeout    = 4 * time.Second
	portForwardServerCloseTimeout    = 20 * time.Second
)

type activePortForward struct {
	id          int
	bindAddress string
	stopOnce    sync.Once
	stopDone    chan struct{}
	stopErr     error
}

func (forward *activePortForward) stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), portForwardStopTimeout)
	defer cancel()
	return forward.stopWithin(ctx)
}

func (forward *activePortForward) stopWithin(ctx context.Context) error {
	if forward == nil {
		return nil
	}
	forward.stopOnce.Do(func() {
		if forward.stopDone == nil {
			forward.stopDone = make(chan struct{})
		}
		go func() {
			if !clientcore.Portfwds.Remove(forward.id) {
				forward.stopErr = fmt.Errorf("port forward %d was not registered", forward.id)
			}
			close(forward.stopDone)
		}()
	})
	select {
	case <-forward.stopDone:
		return forward.stopErr
	case <-ctx.Done():
		return fmt.Errorf("stop port forward %d: %w", forward.id, ctx.Err())
	}
}

type portForwardPayloadCase struct {
	name    string
	payload []byte
	chunks  []int
}

func portForwardPayloadCases() []portForwardPayloadCase {
	boundarySizes := []int{1, 2, 4095, 4096, 4108, 32767, 32768, 65535, 65536, 65537, 128*1024 + 17}
	cases := make([]portForwardPayloadCase, 0, len(boundarySizes)+portForwardRandomizedCases)
	for index, size := range boundarySizes {
		payload := deterministicTunnelPayload(fmt.Sprintf("portfwd-boundary-%d-%d", index, size), size)
		payload[0] = 0
		if len(payload) > 1 {
			payload[len(payload)-1] = 0xff
		}
		cases = append(cases, portForwardPayloadCase{
			name:    fmt.Sprintf("boundary-%d", size),
			payload: payload,
			chunks:  []int{1, 7, 257, 4095, 2, 8192},
		})
	}

	randomSource := rand.New(rand.NewSource(0x51_1e_2e))
	for index := 0; index < portForwardRandomizedCases; index++ {
		size := 1 + randomSource.Intn(128*1024)
		chunks := make([]int, 7)
		for chunkIndex := range chunks {
			chunks[chunkIndex] = 1 + randomSource.Intn(16*1024)
		}
		payload := deterministicTunnelPayload(fmt.Sprintf("portfwd-randomized-%d-%d", index, size), size)
		cases = append(cases, portForwardPayloadCase{
			name:    fmt.Sprintf("seeded-random-%02d-size-%d", index, size),
			payload: payload,
			chunks:  chunks,
		})
	}
	return cases
}

func (s *suite) exercisePortForward(target implantTarget, transport string) error {
	if target.session == nil || target.beacon != nil {
		return errors.New("forward port forwarding E2E requires a session target")
	}

	var exerciseErrors []error
	exerciseErrors = appendIfError(exerciseErrors, s.portForwardStep(transport, "binary-boundaries", "relays binary boundary and seeded-random payloads", func() error {
		return s.exercisePortForwardPayloads(target)
	}))
	exerciseErrors = appendIfError(exerciseErrors, s.portForwardStep(transport, "sustained-full-duplex", "sustains exact simultaneous bidirectional MiB traffic above the throughput floor", func() error {
		return s.exercisePortForwardFullDuplex(target)
	}))
	exerciseErrors = appendIfError(exerciseErrors, s.portForwardStep(transport, "sequential-concurrent", "supports repeated and concurrent connections", func() error {
		return s.exercisePortForwardConnections(target)
	}))
	exerciseErrors = appendIfError(exerciseErrors, s.portForwardStep(transport, "http-curl", "carries deterministic HTTP and curl traffic", func() error {
		return s.exercisePortForwardHTTP(target)
	}))
	exerciseErrors = appendIfError(exerciseErrors, s.portForwardStep(transport, "idle-resume", "resumes a connection after a keepalive interval", func() error {
		return s.exercisePortForwardIdleResume(target)
	}))
	exerciseErrors = appendIfError(exerciseErrors, s.portForwardStep(transport, "implant-dial-failure-recovery", "propagates a real implant destination-dial failure and preserves the session", func() error {
		return s.exercisePortForwardImplantDialFailureRecovery(target)
	}))
	exerciseErrors = appendIfError(exerciseErrors, s.portForwardStep(transport, "stop-new-dial", "rejects new connections after stop", func() error {
		return s.exercisePortForwardStop(target)
	}))
	exerciseErrors = appendIfError(exerciseErrors, s.portForwardStep(transport, "proxy-isolation", "keeps independent proxies isolated", func() error {
		return s.exercisePortForwardIsolation(target)
	}))
	if s.opts.tunnelHTTPURL != "" {
		exerciseErrors = appendIfError(exerciseErrors, s.portForwardStep(transport, "real-http-curl", "carries a real curl request to the configured target", func() error {
			return s.exercisePortForwardExternalHTTP(target, transport)
		}))
	}
	if s.opts.tunnelRDPAddr != "" {
		exerciseErrors = appendIfError(exerciseErrors, s.portForwardStep(transport, "rdp-negotiation", "negotiates the real RDP service through a forward", func() error {
			return s.exercisePortForwardRDP(target, transport)
		}))
	}
	return errors.Join(exerciseErrors...)
}

func (s *suite) exercisePortForwardFullDuplex(target implantTarget) (resultErr error) {
	echoServer, err := startTCPEchoServer()
	if err != nil {
		return fmt.Errorf("start full-duplex target echo fixture: %w", err)
	}
	defer func() { resultErr = errors.Join(resultErr, echoServer.close()) }()
	forward, err := s.startPortForward(target, echoServer.address(), 30*time.Second)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, forward.stop()) }()
	ctx, cancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
	defer cancel()
	connection, err := (&net.Dialer{Timeout: socksE2ECaseTimeout}).DialContext(ctx, "tcp4", forward.bindAddress)
	if err != nil {
		return fmt.Errorf("dial full-duplex port forward: %w", err)
	}
	defer func() { _ = connection.Close() }()
	elapsed, err := tunnelFullDuplexEchoOnConn(
		ctx,
		connection,
		"portfwd-sustained-full-duplex",
		tunnelFullDuplexPayloadBytes,
		tunnelFullDuplexMinimumBytesPerSecond,
	)
	if err != nil {
		return err
	}
	s.t.Logf("Port forward full-duplex: %d bytes each direction in %s", tunnelFullDuplexPayloadBytes, elapsed.Round(time.Millisecond))
	return nil
}

func (s *suite) exercisePortForwardPayloads(target implantTarget) (resultErr error) {
	echoServer, err := startTCPEchoServer()
	if err != nil {
		return fmt.Errorf("start target-side TCP echo fixture: %w", err)
	}
	defer func() {
		if err := echoServer.close(); err != nil {
			resultErr = errors.Join(resultErr, err)
		}
	}()
	forward, err := s.startPortForward(target, echoServer.address(), 30*time.Second)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, forward.stop()) }()

	for _, testCase := range portForwardPayloadCases() {
		ctx, cancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
		err := tunnelEchoRoundTrip(ctx, forward.bindAddress, testCase.payload, testCase.chunks)
		cancel()
		if err != nil {
			return fmt.Errorf("%s (%d bytes, chunks %v): %w", testCase.name, len(testCase.payload), testCase.chunks, err)
		}
	}
	// CloseTunnel deliberately gives asynchronously delivered final frames a
	// quiet-period grace on the server. Drain the entire corpus once instead of
	// paying that grace after every short-lived connection.
	return s.waitForPortForwardEchoDrain(echoServer)
}

func (s *suite) exercisePortForwardConnections(target implantTarget) (resultErr error) {
	echoServer, err := startTCPEchoServer()
	if err != nil {
		return fmt.Errorf("start connection TCP echo fixture: %w", err)
	}
	defer func() { resultErr = errors.Join(resultErr, echoServer.close()) }()
	forward, err := s.startPortForward(target, echoServer.address(), 30*time.Second)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, forward.stop()) }()

	for index := 0; index < portForwardSequentialConnections; index++ {
		ctx, cancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
		payload := deterministicTunnelPayload(fmt.Sprintf("portfwd-sequential-%d", index), 4096+index*977)
		err := tunnelEchoRoundTrip(ctx, forward.bindAddress, payload, []int{17, 1024, 3, 4096})
		cancel()
		if err != nil {
			return fmt.Errorf("sequential connection %d/%d: %w", index+1, portForwardSequentialConnections, err)
		}
	}

	ctx, cancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
	start := make(chan struct{})
	errorsByConnection := make(chan error, portForwardConcurrentConnections)
	var workers sync.WaitGroup
	for index := 0; index < portForwardConcurrentConnections; index++ {
		index := index
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			payload := deterministicTunnelPayload(fmt.Sprintf("portfwd-concurrent-%d", index), 8192+index*1231)
			if err := tunnelEchoRoundTrip(ctx, forward.bindAddress, payload, []int{1 + index, 4096, 31}); err != nil {
				errorsByConnection <- fmt.Errorf("concurrent connection %d/%d: %w", index+1, portForwardConcurrentConnections, err)
			}
		}()
	}
	close(start)
	workers.Wait()
	close(errorsByConnection)
	var connectionErrors []error
	for err := range errorsByConnection {
		connectionErrors = append(connectionErrors, err)
	}
	cancel()
	if err := errors.Join(connectionErrors...); err != nil {
		return err
	}
	return s.waitForPortForwardEchoDrain(echoServer)
}

func (s *suite) exercisePortForwardHTTP(target implantTarget) (resultErr error) {
	fixture, err := startDeterministicHTTPServer("portfwd-http")
	if err != nil {
		return fmt.Errorf("start deterministic HTTP fixture: %w", err)
	}
	defer func() { resultErr = errors.Join(resultErr, fixture.close()) }()
	forward, err := s.startPortForward(target, fixture.address(), 30*time.Second)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, forward.stop()) }()

	ctx, cancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
	if err := requestDeterministicHTTP(ctx, forward.bindAddress, fixture); err != nil {
		cancel()
		return err
	}
	cancel()

	ctx, cancel = context.WithTimeout(s.ctx, s.opts.commandTimeout)
	output, err := s.curlTunnelHTTP(ctx, tunnelHTTPCurlRequest{
		targetURL: fixture.url(forward.bindAddress),
		noProxy:   "*",
	})
	cancel()
	if err != nil {
		return fmt.Errorf("curl deterministic HTTP fixture through port forward: %w", err)
	}
	if !bytes.Equal(output, fixture.body) {
		return fmt.Errorf("curl body mismatch: got %d bytes, want exact %d-byte body", len(output), len(fixture.body))
	}
	return nil
}

func (s *suite) exercisePortForwardIdleResume(target implantTarget) (resultErr error) {
	echoServer, err := startTCPEchoServer()
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, echoServer.close()) }()
	forward, err := s.startPortForward(target, echoServer.address(), 30*time.Second)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, forward.stop()) }()

	dialCtx, dialCancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
	connection, err := (&net.Dialer{Timeout: 2 * time.Second}).DialContext(dialCtx, "tcp4", forward.bindAddress)
	dialCancel()
	if err != nil {
		return fmt.Errorf("dial idle/resume port forward: %w", err)
	}
	defer func() { _ = connection.Close() }()

	initialCtx, initialCancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
	err = tunnelEchoRoundTripOnConn(initialCtx, connection, []byte("portfwd-before-idle\x00\xff"), []int{1, 5, 2})
	initialCancel()
	if err != nil {
		return fmt.Errorf("initial idle/resume exchange: %w", err)
	}
	if err := connection.SetDeadline(time.Time{}); err != nil {
		return fmt.Errorf("clear idle/resume deadline: %w", err)
	}
	idleTimer := time.NewTimer(portForwardIdleDuration)
	select {
	case <-idleTimer.C:
	case <-s.ctx.Done():
		if !idleTimer.Stop() {
			<-idleTimer.C
		}
		return s.ctx.Err()
	}

	resumeCtx, resumeCancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
	err = tunnelEchoRoundTripOnConn(resumeCtx, connection, deterministicTunnelPayload("portfwd-after-idle", 32*1024+3), []int{4095, 1, 8192})
	resumeCancel()
	if err != nil {
		return fmt.Errorf("resumed idle connection: %w", err)
	}
	if err := connection.Close(); err != nil {
		return fmt.Errorf("close idle/resume connection: %w", err)
	}
	return s.waitForPortForwardEchoDrain(echoServer)
}

func (s *suite) waitForPortForwardEchoDrain(echoServer *tcpEchoServer) error {
	ctx, cancel := context.WithTimeout(s.ctx, portForwardServerCloseTimeout)
	defer cancel()
	return waitForEchoConnectionCount(ctx, echoServer, 0)
}

func (s *suite) exercisePortForwardImplantDialFailureRecovery(target implantTarget) (resultErr error) {
	// Select and release a valid nonzero loopback port. The destination passes
	// client-side parsing, so accepting a connection on the local proxy drives
	// the production request through to the implant's net.Dialer. Nothing is
	// listening after unusedTCPPort returns, making connection refusal the
	// expected implant-side setup result.
	closedPort, err := unusedTCPPort()
	if err != nil {
		return fmt.Errorf("select closed implant destination port: %w", err)
	}
	dialFailureAddress := fmt.Sprintf("127.0.0.1:%d", closedPort)
	if _, _, err := (&clientcore.ChannelProxy{RemoteAddr: dialFailureAddress}).ValidatedHostPort(); err != nil {
		return fmt.Errorf("closed implant destination is not locally valid: %w", err)
	}
	dialFailureForward, err := s.startPortForward(target, dialFailureAddress, portForwardDialFailureTimeout)
	if err != nil {
		return fmt.Errorf("start implant-dial-failure port forward: %w", err)
	}
	dialFailureErr := s.requireImplantDialFailureTermination(dialFailureForward.bindAddress)
	stopErr := dialFailureForward.stop()
	if stopErr != nil {
		stopErr = fmt.Errorf("stop implant-dial-failure port forward: %w", stopErr)
	}

	pingErr := s.requireTunnelSessionPing(target)
	if pingErr != nil {
		pingErr = fmt.Errorf("session Ping after implant dial failure: %w", pingErr)
	}

	echoServer, echoErr := startTCPEchoServer()
	if echoErr != nil {
		return errors.Join(dialFailureErr, stopErr, pingErr, fmt.Errorf("start recovery echo fixture: %w", echoErr))
	}
	defer func() { resultErr = errors.Join(resultErr, echoServer.close()) }()
	recoveryForward, startErr := s.startPortForward(target, echoServer.address(), 30*time.Second)
	if startErr != nil {
		return errors.Join(dialFailureErr, stopErr, pingErr, fmt.Errorf("start recovery port forward: %w", startErr))
	}
	defer func() { resultErr = errors.Join(resultErr, recoveryForward.stop()) }()
	ctx, cancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
	recoveryErr := tunnelEchoRoundTrip(ctx, recoveryForward.bindAddress, deterministicTunnelPayload("portfwd-implant-dial-failure-recovery", 16*1024+1), []int{7, 4096, 1})
	cancel()
	if recoveryErr != nil {
		recoveryErr = fmt.Errorf("valid relay after implant dial failure: %w", recoveryErr)
	}
	return errors.Join(dialFailureErr, stopErr, pingErr, recoveryErr)
}

func (s *suite) exercisePortForwardStop(target implantTarget) (resultErr error) {
	echoServer, err := startTCPEchoServer()
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, echoServer.close()) }()
	forward, err := s.startPortForward(target, echoServer.address(), 30*time.Second)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, forward.stop()) }()
	ctx, cancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
	connection, err := (&net.Dialer{}).DialContext(ctx, "tcp4", forward.bindAddress)
	if err != nil {
		cancel()
		return fmt.Errorf("dial active port forward before stop: %w", err)
	}
	defer func() { _ = connection.Close() }()
	if err := tunnelEchoRoundTripOnConn(ctx, connection, []byte("portfwd-before-stop"), []int{3, 1, 8}); err != nil {
		cancel()
		return err
	}
	if err := connection.SetDeadline(time.Time{}); err != nil {
		cancel()
		return err
	}
	if err := waitForEchoConnectionCount(ctx, echoServer, 1); err != nil {
		cancel()
		return fmt.Errorf("active port-forward target relay readiness: %w", err)
	}
	cancel()
	stopCtx, stopCancel := context.WithTimeout(s.ctx, portForwardStopTimeout)
	if err := forward.stopWithin(stopCtx); err != nil {
		stopCancel()
		return err
	}
	if err := requireTunnelConnectionClosed(stopCtx, connection, "port-forward proxy stop"); err != nil {
		stopCancel()
		return err
	}
	stopCancel()
	// The generic CloseTunnel protocol retains a ten-second ordering grace so
	// the unary close cannot overtake final stream data. Give the independently
	// observed implant-side target socket its own bounded drain window.
	targetCloseCtx, targetCloseCancel := context.WithTimeout(s.ctx, portForwardServerCloseTimeout)
	if err := waitForEchoConnectionCount(targetCloseCtx, echoServer, 0); err != nil {
		targetCloseCancel()
		return fmt.Errorf("port-forward target relay after proxy stop: %w", err)
	}
	targetCloseCancel()
	ctx, cancel = context.WithTimeout(s.ctx, portForwardStopTimeout)
	defer cancel()
	if err := requireTCPDialRejection(ctx, forward.bindAddress); err != nil {
		return err
	}
	if findPortForwardMetadata(forward.id) != nil {
		return fmt.Errorf("stopped port forward %d remains in client inventory", forward.id)
	}
	return nil
}

func (s *suite) exercisePortForwardIsolation(target implantTarget) (resultErr error) {
	fixtureA, err := startDeterministicHTTPServer("portfwd-isolation-a")
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, fixtureA.close()) }()
	fixtureB, err := startDeterministicHTTPServer("portfwd-isolation-b")
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, fixtureB.close()) }()
	forwardA, err := s.startPortForward(target, fixtureA.address(), 30*time.Second)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, forwardA.stop()) }()
	forwardB, err := s.startPortForward(target, fixtureB.address(), 30*time.Second)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, forwardB.stop()) }()

	ctx, cancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
	requestAErr := requestDeterministicHTTP(ctx, forwardA.bindAddress, fixtureA)
	requestBErr := requestDeterministicHTTP(ctx, forwardB.bindAddress, fixtureB)
	cancel()
	if err := errors.Join(requestAErr, requestBErr); err != nil {
		return fmt.Errorf("initial isolated requests: %w", err)
	}
	if err := forwardA.stop(); err != nil {
		return fmt.Errorf("stop first isolated proxy: %w", err)
	}
	ctx, cancel = context.WithTimeout(s.ctx, portForwardStopTimeout)
	rejectionErr := requireTCPDialRejection(ctx, forwardA.bindAddress)
	cancel()
	ctx, cancel = context.WithTimeout(s.ctx, s.opts.commandTimeout)
	survivorErr := requestDeterministicHTTP(ctx, forwardB.bindAddress, fixtureB)
	cancel()
	if findPortForwardMetadata(forwardA.id) != nil {
		rejectionErr = errors.Join(rejectionErr, fmt.Errorf("stopped isolated proxy %d remains in inventory", forwardA.id))
	}
	if findPortForwardMetadata(forwardB.id) == nil {
		survivorErr = errors.Join(survivorErr, fmt.Errorf("surviving isolated proxy %d is missing from inventory", forwardB.id))
	}
	return errors.Join(rejectionErr, survivorErr)
}

func (s *suite) startPortForward(target implantTarget, remoteAddress string, dialTimeout time.Duration) (*activePortForward, error) {
	if target.session == nil || target.beacon != nil {
		return nil, errors.New("start port forward requires a session target")
	}
	bindAddress := "127.0.0.1:0"
	actualBindAddress := ""
	tcpProxy := &tcpproxy.Proxy{}
	tcpProxy.ListenFunc = func(network string, address string) (net.Listener, error) {
		listener, listenErr := net.Listen(network, address)
		if listenErr == nil {
			actualBindAddress = listener.Addr().String()
		}
		return listener, listenErr
	}
	channelProxy := &clientcore.ChannelProxy{
		Rpc:             s.rpc,
		Session:         target.session,
		RemoteAddr:      remoteAddress,
		BindAddr:        bindAddress,
		KeepAlivePeriod: 5 * time.Second,
		DialTimeout:     dialTimeout,
	}
	tcpProxy.AddRoute(bindAddress, channelProxy)
	if err := tcpProxy.Start(); err != nil {
		channelProxy.Stop()
		_ = tcpProxy.Close()
		return nil, fmt.Errorf("start production port forward %s -> %s: %w", bindAddress, remoteAddress, err)
	}
	if actualBindAddress == "" {
		channelProxy.Stop()
		_ = tcpProxy.Close()
		return nil, errors.New("production port forward listener did not publish its bound address")
	}
	bindAddress = actualBindAddress
	channelProxy.BindAddr = bindAddress
	registered := clientcore.Portfwds.Add(tcpProxy, channelProxy)
	forward := &activePortForward{id: registered.ID, bindAddress: bindAddress, stopDone: make(chan struct{})}
	metadata := findPortForwardMetadata(registered.ID)
	if metadata == nil {
		_ = forward.stop()
		return nil, fmt.Errorf("port forward %d missing from client inventory", registered.ID)
	}
	if metadata.SessionID != target.session.ID || metadata.BindAddr != bindAddress || metadata.RemoteAddr != remoteAddress {
		_ = forward.stop()
		return nil, fmt.Errorf("port forward metadata mismatch: got %+v, want session=%q bind=%q remote=%q", metadata, target.session.ID, bindAddress, remoteAddress)
	}
	return forward, nil
}

func findPortForwardMetadata(id int) *clientcore.PortfwdMeta {
	for _, metadata := range clientcore.Portfwds.List() {
		if metadata.ID == id {
			return metadata
		}
	}
	return nil
}

func (s *suite) requireImplantDialFailureTermination(address string) error {
	ctx, cancel := context.WithTimeout(s.ctx, 2*portForwardDialFailureTimeout)
	defer cancel()
	connection, err := (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, "tcp4", address)
	if err != nil {
		return fmt.Errorf("dial implant-failure port forward: %w", err)
	}
	defer func() { _ = connection.Close() }()
	if deadline, ok := ctx.Deadline(); ok {
		_ = connection.SetDeadline(deadline)
	}
	if _, err := connection.Write([]byte("implant-dial-failure-probe")); err != nil {
		if networkErr, ok := err.(net.Error); ok && networkErr.Timeout() {
			return fmt.Errorf("implant-dial-failure write did not terminate before deadline: %w", err)
		}
		return nil
	}
	buffer := make([]byte, 1)
	count, readErr := connection.Read(buffer)
	if readErr == nil || count != 0 {
		return fmt.Errorf("failed implant destination returned unexpected data (%d bytes, error %v)", count, readErr)
	}
	if networkErr, ok := readErr.(net.Error); ok && networkErr.Timeout() {
		return fmt.Errorf("implant-dial-failure connection did not close before deadline: %w", readErr)
	}
	return nil
}

func (s *suite) requireTunnelSessionPing(target implantTarget) error {
	const nonce = int32(0x3f17a20d)
	response, err := invokeRPC(s, target, "Ping", func(ctx context.Context, request *commonpb.Request) (*sliverpb.Ping, error) {
		return s.rpc.Ping(ctx, &sliverpb.Ping{Nonce: nonce, Request: request})
	}, func(response *sliverpb.Ping) *commonpb.Response { return response.GetResponse() })
	if err != nil {
		return err
	}
	if response.GetNonce() != nonce {
		return fmt.Errorf("ping nonce = %d, want %d", response.GetNonce(), nonce)
	}
	return nil
}

func (s *suite) portForwardStep(transport string, scenario string, label string, run func() error) error {
	started := time.Now()
	err := run()
	duration := time.Since(started)
	s.recordTunnelScenario(transport, "portfwd", scenario, duration, err)
	if err != nil {
		return fmt.Errorf("portfwd (%s, %s/%s, %s): %w", label, transport, scenario, duration.Round(time.Millisecond), err)
	}
	s.t.Logf("PASS %s/%s %s session Portfwd/%s (%s)", s.opts.targetOS, s.opts.targetArch, transport, scenario, duration.Round(time.Millisecond))
	return nil
}

func requireTCPDialRejection(ctx context.Context, address string) error {
	const requiredConsecutiveFailures = 3
	ticker := time.NewTicker(listenerPollInterval)
	defer ticker.Stop()
	consecutiveFailures := 0
	for {
		attemptCtx, cancel := context.WithTimeout(ctx, listenerDialTimeout)
		connection, err := (&net.Dialer{}).DialContext(attemptCtx, "tcp4", address)
		cancel()
		if err != nil {
			consecutiveFailures++
			if consecutiveFailures == requiredConsecutiveFailures {
				return nil
			}
		} else {
			consecutiveFailures = 0
			_ = connection.Close()
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("local proxy %s still accepted connections after stop: %w", address, ctx.Err())
		case <-ticker.C:
		}
	}
}
