package e2e

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	e2ecoverage "github.com/bishopfox/sliver/test/e2e/coverage"
	"github.com/bishopfox/sliver/test/e2e/rportfwdcoverage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	reverseForwardSequentialConnections = 4
	reverseForwardConcurrentConnections = 6
	reverseForwardStopTimeout           = 20 * time.Second
	attackerSuppliedAuthorizationID     = "attacker-controlled-authorization"
)

type tcpEchoServer struct {
	listener net.Listener
	done     chan struct{}
	err      chan error

	mu          sync.Mutex
	connections map[net.Conn]struct{}
	workers     sync.WaitGroup
}

func startTCPEchoServer() (*tcpEchoServer, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	server := &tcpEchoServer{
		listener:    listener,
		done:        make(chan struct{}),
		err:         make(chan error, 1),
		connections: map[net.Conn]struct{}{},
	}
	go server.serve()
	return server, nil
}

func (server *tcpEchoServer) address() string {
	return server.listener.Addr().String()
}

func (server *tcpEchoServer) connectionCount() int {
	server.mu.Lock()
	defer server.mu.Unlock()
	return len(server.connections)
}

func (server *tcpEchoServer) serve() {
	defer close(server.done)
	for {
		connection, err := server.listener.Accept()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				select {
				case server.err <- err:
				default:
				}
			}
			return
		}

		server.mu.Lock()
		server.connections[connection] = struct{}{}
		server.mu.Unlock()
		server.workers.Add(1)
		go func() {
			defer server.workers.Done()
			defer func() {
				server.mu.Lock()
				delete(server.connections, connection)
				server.mu.Unlock()
				_ = connection.Close()
			}()
			_, _ = io.Copy(connection, connection)
		}()
	}
}

func (server *tcpEchoServer) close() error {
	listenerErr := server.listener.Close()
	<-server.done
	// Waiting for the accept loop first guarantees it cannot add a connection
	// after this snapshot. Closing the active set then unblocks every echo
	// worker before Wait.
	server.mu.Lock()
	for connection := range server.connections {
		_ = connection.Close()
	}
	server.mu.Unlock()
	server.workers.Wait()

	select {
	case serveErr := <-server.err:
		return errors.Join(listenerErr, fmt.Errorf("echo server accept: %w", serveErr))
	default:
		return listenerErr
	}
}

func (s *suite) exerciseReversePortForward(target implantTarget, transport string) (resultErr error) {
	if target.session == nil || target.beacon != nil {
		return errors.New("reverse port forwarding E2E requires a session target")
	}
	if err := s.rportfwdStep(transport, rportfwdcoverage.ScenarioInvalidDestination, "rejects invalid operator destinations", func() error {
		return s.requireInvalidReverseForwardRejected(target)
	}); err != nil {
		return err
	}

	echoServer, err := startTCPEchoServer()
	if err != nil {
		return fmt.Errorf("start teamserver-side TCP echo service: %w", err)
	}
	defer func() {
		if err := echoServer.close(); err != nil {
			resultErr = errors.Join(resultErr, err)
		}
	}()

	bindPort, err := unusedTCPPort()
	if err != nil {
		return fmt.Errorf("select implant reverse-forward bind port: %w", err)
	}
	bindAddress := fmt.Sprintf("127.0.0.1:%d", bindPort)
	forwardAddress := echoServer.address()

	listener, err := s.startRportFwdListener(target, bindAddress, forwardAddress)
	if err != nil {
		return fmt.Errorf("start reverse-forward listener: %w", err)
	}
	listenerID := listener.GetID()
	stopped := false
	defer func() {
		if stopped {
			return
		}
		_, cleanupErr := s.stopRportFwdListener(target, listenerID)
		if cleanupErr != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("cleanup reverse-forward listener %d: %w", listenerID, cleanupErr))
		}
	}()
	authorizationID := listener.GetAuthorizationID()
	if err := s.rportfwdStep(transport, rportfwdcoverage.ScenarioMetadataAuthority, "returns authoritative listener metadata", func() error {
		if listenerID == 0 {
			return errors.New("start response returned zero reverse-forward listener ID")
		}
		if err := validateRportFwdListener(listener, listenerID, bindAddress, forwardAddress, ""); err != nil {
			return fmt.Errorf("validate start metadata: %w", err)
		}
		if authorizationID == "" {
			return errors.New("start response omitted server-issued reverse-forward authorization ID")
		}
		if authorizationID == attackerSuppliedAuthorizationID {
			return errors.New("teamserver accepted the caller-supplied reverse-forward authorization ID")
		}
		listeners, err := s.getRportFwdListeners(target)
		if err != nil {
			return err
		}
		listed := findRportFwdListener(listeners.GetListeners(), listenerID)
		if listed == nil {
			return fmt.Errorf("listener %d missing from inventory", listenerID)
		}
		return validateRportFwdListener(listed, listenerID, bindAddress, forwardAddress, authorizationID)
	}); err != nil {
		return err
	}

	if err := s.rportfwdStep(transport, rportfwdcoverage.ScenarioInvalidSessionStopIsolation, "invalid session cannot affect an active listener", func() error {
		return s.requireInvalidSessionStopIsolation(target, listenerID, bindAddress, forwardAddress, authorizationID)
	}); err != nil {
		return err
	}

	if err := s.rportfwdStep(transport, rportfwdcoverage.ScenarioBidirectionalEcho, "relays bidirectional echo", func() error {
		ctx, cancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
		defer cancel()
		payload := []byte("sliver-rportfwd-bidirectional-echo\x00\xff")
		if err := waitForReverseForwardRoundTrip(ctx, bindAddress, payload); err != nil {
			return err
		}
		return waitForEchoConnectionCount(ctx, echoServer, 0)
	}); err != nil {
		return err
	}

	if err := s.rportfwdStep(transport, rportfwdcoverage.ScenarioSequentialConnections, "supports repeated sequential connections", func() error {
		for index := 0; index < reverseForwardSequentialConnections; index++ {
			ctx, cancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
			payload := bytes.Repeat([]byte(fmt.Sprintf("sequential-%d-", index)), 257)
			roundTripErr := reverseForwardRoundTrip(ctx, bindAddress, payload)
			if roundTripErr != nil {
				cancel()
				return fmt.Errorf("connection %d/%d: %w", index+1, reverseForwardSequentialConnections, roundTripErr)
			}
			cleanupErr := waitForEchoConnectionCount(ctx, echoServer, 0)
			cancel()
			if cleanupErr != nil {
				return fmt.Errorf("connection %d/%d cleanup: %w", index+1, reverseForwardSequentialConnections, cleanupErr)
			}
		}
		return nil
	}); err != nil {
		return err
	}

	if err := s.rportfwdStep(transport, rportfwdcoverage.ScenarioConcurrentConnections, "supports concurrent connections", func() error {
		ctx, cancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
		defer cancel()
		start := make(chan struct{})
		errorsByConnection := make(chan error, reverseForwardConcurrentConnections)
		var workers sync.WaitGroup
		for index := 0; index < reverseForwardConcurrentConnections; index++ {
			index := index
			workers.Add(1)
			go func() {
				defer workers.Done()
				<-start
				payload := bytes.Repeat([]byte(fmt.Sprintf("concurrent-%d-", index)), 521)
				if err := reverseForwardRoundTrip(ctx, bindAddress, payload); err != nil {
					errorsByConnection <- fmt.Errorf("connection %d: %w", index+1, err)
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
		if err := errors.Join(connectionErrors...); err != nil {
			return err
		}
		return waitForEchoConnectionCount(ctx, echoServer, 0)
	}); err != nil {
		return err
	}

	var establishedConnection net.Conn
	defer func() {
		if establishedConnection != nil {
			_ = establishedConnection.Close()
		}
	}()
	if err := s.rportfwdStep(transport, rportfwdcoverage.ScenarioEstablishedBeforeStop, "establishes a persistent relay before stop", func() error {
		ctx, cancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
		defer cancel()
		connection, err := (&net.Dialer{}).DialContext(ctx, "tcp", bindAddress)
		if err != nil {
			return fmt.Errorf("dial established relay: %w", err)
		}
		establishedConnection = connection
		deadline, ok := ctx.Deadline()
		if ok {
			_ = establishedConnection.SetDeadline(deadline)
		}
		payload := []byte("established-before-stop\x00\xff")
		if _, err := establishedConnection.Write(payload); err != nil {
			return fmt.Errorf("write established relay: %w", err)
		}
		echoed := make([]byte, len(payload))
		if _, err := io.ReadFull(establishedConnection, echoed); err != nil {
			return fmt.Errorf("read established relay: %w", err)
		}
		if !bytes.Equal(echoed, payload) {
			return fmt.Errorf("established relay echo mismatch: got %x, want %x", echoed, payload)
		}
		_ = establishedConnection.SetDeadline(time.Time{})
		return waitForEchoConnectionCount(ctx, echoServer, 1)
	}); err != nil {
		return err
	}

	if err := s.rportfwdStep(transport, rportfwdcoverage.ScenarioStopClosesEstablished, "stop revokes listener and closes established relay", func() error {
		response, err := s.stopRportFwdListener(target, listenerID)
		if err != nil {
			return err
		}
		stopped = true
		if err := validateRportFwdListener(response, listenerID, bindAddress, forwardAddress, authorizationID); err != nil {
			return fmt.Errorf("validate stop metadata: %w", err)
		}

		listeners, err := s.getRportFwdListeners(target)
		if err != nil {
			return err
		}
		if listed := findRportFwdListener(listeners.GetListeners(), listenerID); listed != nil {
			return fmt.Errorf("stopped listener %d remains in inventory", listenerID)
		}

		ctx, cancel := context.WithTimeout(s.ctx, reverseForwardStopTimeout)
		defer cancel()
		if establishedConnection == nil {
			return errors.New("established relay disappeared before stop validation")
		}
		if deadline, ok := ctx.Deadline(); ok {
			_ = establishedConnection.SetReadDeadline(deadline)
		}
		buffer := make([]byte, 1)
		count, readErr := establishedConnection.Read(buffer)
		if readErr == nil || count != 0 {
			return fmt.Errorf("established relay remained readable after stop (%d bytes, error %v)", count, readErr)
		}
		if networkErr, ok := readErr.(net.Error); ok && networkErr.Timeout() {
			return fmt.Errorf("established relay did not close before stop deadline: %w", readErr)
		}
		_ = establishedConnection.Close()
		establishedConnection = nil
		if err := waitForEchoConnectionCount(ctx, echoServer, 0); err != nil {
			return fmt.Errorf("target side remained open after stop: %w", err)
		}
		return requireReverseForwardDialRejection(ctx, bindAddress)
	}); err != nil {
		return err
	}

	return nil
}

func (s *suite) requireInvalidReverseForwardRejected(target implantTarget) error {
	before, err := s.getRportFwdListeners(target)
	if err != nil {
		return fmt.Errorf("inventory before invalid start: %w", err)
	}
	ctx, cancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
	defer cancel()
	_, err = s.rpc.StartRportFwdListener(ctx, &sliverpb.RportFwdStartListenerReq{
		BindAddress:     "127.0.0.1:0",
		ForwardAddress:  "127.0.0.1:0",
		AuthorizationID: attackerSuppliedAuthorizationID,
		Request:         target.request(s.opts.commandTimeout),
	})
	if status.Code(err) != codes.InvalidArgument {
		return fmt.Errorf("invalid destination error = %v, want %s", err, codes.InvalidArgument)
	}
	after, err := s.getRportFwdListeners(target)
	if err != nil {
		return fmt.Errorf("inventory after invalid start: %w", err)
	}
	if len(after.GetListeners()) != len(before.GetListeners()) {
		return fmt.Errorf("invalid start changed inventory from %d to %d listeners", len(before.GetListeners()), len(after.GetListeners()))
	}
	return nil
}

func (s *suite) requireInvalidSessionStopIsolation(target implantTarget, listenerID uint32, bindAddress string, forwardAddress string, authorizationID string) error {
	ctx, cancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
	defer cancel()
	request := target.request(s.opts.commandTimeout)
	request.SessionID = "00000000-0000-4000-8000-000000000001"
	_, err := s.rpc.StopRportFwdListener(ctx, &sliverpb.RportFwdStopListenerReq{
		ID:      listenerID,
		Request: request,
	})
	if err == nil {
		return errors.New("stop for an invalid session unexpectedly succeeded")
	}
	listeners, inventoryErr := s.getRportFwdListeners(target)
	if inventoryErr != nil {
		return fmt.Errorf("owner inventory after invalid-session stop: %w", inventoryErr)
	}
	listed := findRportFwdListener(listeners.GetListeners(), listenerID)
	if listed == nil {
		return errors.New("invalid-session stop revoked the active listener")
	}
	return validateRportFwdListener(listed, listenerID, bindAddress, forwardAddress, authorizationID)
}

func (s *suite) exerciseReversePortForwardDisconnect(target implantTarget, transport string, process *managedProcess) (resultErr error) {
	if target.session == nil || target.beacon != nil || process == nil {
		return errors.New("reverse-forward disconnect E2E requires a managed session process")
	}
	echoServer, err := startTCPEchoServer()
	if err != nil {
		return fmt.Errorf("start disconnect TCP echo service: %w", err)
	}
	defer func() {
		if err := echoServer.close(); err != nil {
			resultErr = errors.Join(resultErr, err)
		}
	}()
	bindPort, err := unusedTCPPort()
	if err != nil {
		return err
	}
	bindAddress := fmt.Sprintf("127.0.0.1:%d", bindPort)
	listener, err := s.startRportFwdListener(target, bindAddress, echoServer.address())
	if err != nil {
		return fmt.Errorf("start disconnect reverse-forward listener: %w", err)
	}
	listenerID := listener.GetID()
	if listenerID == 0 || listener.GetAuthorizationID() == "" || listener.GetAuthorizationID() == attackerSuppliedAuthorizationID {
		return fmt.Errorf("invalid disconnect listener metadata: %+v", listener)
	}
	disconnected := false
	defer func() {
		if disconnected {
			return
		}
		if _, cleanupErr := s.stopRportFwdListener(target, listenerID); cleanupErr != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("cleanup disconnect listener: %w", cleanupErr))
		}
	}()

	ctx, cancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
	if err := waitForReverseForwardRoundTrip(ctx, bindAddress, []byte("disconnect-listener-ready")); err != nil {
		cancel()
		return err
	}
	connection, err := (&net.Dialer{}).DialContext(ctx, "tcp", bindAddress)
	cancel()
	if err != nil {
		return fmt.Errorf("dial disconnect relay: %w", err)
	}
	defer connection.Close()
	if err := connection.SetDeadline(time.Now().Add(s.opts.commandTimeout)); err != nil {
		return fmt.Errorf("set disconnect relay deadline: %w", err)
	}
	payload := []byte("active-relay-before-session-close\x00\xff")
	if _, err := connection.Write(payload); err != nil {
		return fmt.Errorf("write disconnect relay: %w", err)
	}
	echoed := make([]byte, len(payload))
	if _, err := io.ReadFull(connection, echoed); err != nil {
		return fmt.Errorf("read disconnect relay: %w", err)
	}
	if !bytes.Equal(echoed, payload) {
		return errors.New("disconnect relay echo mismatch")
	}
	_ = connection.SetDeadline(time.Time{})
	readyCtx, readyCancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
	if err := waitForEchoConnectionCount(readyCtx, echoServer, 1); err != nil {
		readyCancel()
		return err
	}
	readyCancel()

	return s.rportfwdStep(transport, rportfwdcoverage.ScenarioDisconnectClosesEstablished, "session teardown closes established relay", func() error {
		cursor := s.hub.cursor()
		if transport == "http" || transport == "wg" {
			// HTTP polling and WireGuard's userspace UDP/netstack transport cannot
			// deterministically deliver a kernel TCP FIN/RST when the implant is
			// killed. Close the authoritative server session first so this scenario
			// tests relay cleanup rather than a transport keepalive interval.
			closeCtx, closeCancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
			_, closeErr := s.rpc.CloseSession(closeCtx, &sliverpb.CloseSession{Request: target.request(s.opts.commandTimeout)})
			closeCancel()
			if closeErr != nil {
				return fmt.Errorf("request %s session close: %w", transport, closeErr)
			}
			if err := process.stop(); err != nil {
				return fmt.Errorf("stop %s implant after server close: %w", transport, err)
			}
		} else if err := process.stop(); err != nil {
			return fmt.Errorf("abruptly stop %s implant: %w", transport, err)
		}
		disconnected = true

		teardownCtx, teardownCancel := context.WithTimeout(s.ctx, reverseForwardStopTimeout)
		defer teardownCancel()
		if _, _, err := s.hub.wait(teardownCtx, cursor, func(event *clientpb.Event) bool {
			return event.EventType == consts.SessionClosedEvent && event.Session != nil && event.Session.ID == target.session.ID
		}); err != nil {
			return fmt.Errorf("wait for session closed event: %w", err)
		}
		if deadline, ok := teardownCtx.Deadline(); ok {
			_ = connection.SetReadDeadline(deadline)
		}
		buffer := make([]byte, 1)
		count, readErr := connection.Read(buffer)
		if readErr == nil || count != 0 {
			return fmt.Errorf("relay survived session teardown (%d bytes, error %v)", count, readErr)
		}
		if networkErr, ok := readErr.(net.Error); ok && networkErr.Timeout() {
			return fmt.Errorf("relay did not close before teardown deadline: %w", readErr)
		}
		if err := waitForEchoConnectionCount(teardownCtx, echoServer, 0); err != nil {
			return err
		}
		if err := requireReverseForwardDialRejection(teardownCtx, bindAddress); err != nil {
			return err
		}
		listeners, err := s.getRportFwdListeners(target)
		if err != nil {
			return err
		}
		if len(listeners.GetListeners()) != 0 {
			return fmt.Errorf("session teardown retained %d listeners", len(listeners.GetListeners()))
		}
		return nil
	})
}

func (s *suite) rportfwdStep(transport string, scenario string, label string, fn func() error) error {
	started := time.Now()
	err := fn()
	duration := time.Since(started)
	status := e2ecoverage.StatusPass
	detail := ""
	if err != nil {
		status = e2ecoverage.StatusFail
		detail = err.Error()
	}
	if s.rportfwdCoverage == nil {
		recordErr := errors.New("reverse-port-forward coverage recorder is not initialized")
		if err != nil {
			return errors.Join(err, recordErr)
		}
		return recordErr
	}
	recordErr := s.rportfwdCoverage.Add(rportfwdcoverage.Observation{
		Transport: transport,
		Scenario:  scenario,
		Status:    status,
		Duration:  duration,
		Detail:    detail,
	})
	if recordErr != nil {
		if err != nil {
			return errors.Join(err, fmt.Errorf("record reverse-port-forward E2E coverage: %w", recordErr))
		}
		return fmt.Errorf("record reverse-port-forward E2E coverage for %s/%s: %w", transport, scenario, recordErr)
	}
	if err != nil {
		return fmt.Errorf("RportFwd (%s, %s/%s, %s): %w", label, transport, scenario, duration.Round(time.Millisecond), err)
	}
	s.t.Logf("PASS %s/%s %s session RportFwd/%s (%s)", s.opts.targetOS, s.opts.targetArch, transport, scenario, duration.Round(time.Millisecond))
	return nil
}

func waitForEchoConnectionCount(ctx context.Context, server *tcpEchoServer, want int) error {
	ticker := time.NewTicker(listenerPollInterval)
	defer ticker.Stop()
	for {
		if got := server.connectionCount(); got == want {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("echo connection count got %d, want %d: %w", server.connectionCount(), want, ctx.Err())
		case <-ticker.C:
		}
	}
}

func (s *suite) startRportFwdListener(target implantTarget, bindAddress string, forwardAddress string) (*sliverpb.RportFwdListener, error) {
	return invokeRPC(s, target, "StartRportFwdListener", func(ctx context.Context, request *commonpb.Request) (*sliverpb.RportFwdListener, error) {
		return s.rpc.StartRportFwdListener(ctx, &sliverpb.RportFwdStartListenerReq{
			BindAddress:     bindAddress,
			ForwardAddress:  forwardAddress,
			KeepAlive:       5,
			AuthorizationID: attackerSuppliedAuthorizationID,
			Request:         request,
		})
	}, func(response *sliverpb.RportFwdListener) *commonpb.Response { return response.GetResponse() })
}

func (s *suite) getRportFwdListeners(target implantTarget) (*sliverpb.RportFwdListeners, error) {
	return invokeRPC(s, target, "GetRportFwdListeners", func(ctx context.Context, request *commonpb.Request) (*sliverpb.RportFwdListeners, error) {
		return s.rpc.GetRportFwdListeners(ctx, &sliverpb.RportFwdListenersReq{Request: request})
	}, func(response *sliverpb.RportFwdListeners) *commonpb.Response { return response.GetResponse() })
}

func (s *suite) stopRportFwdListener(target implantTarget, listenerID uint32) (*sliverpb.RportFwdListener, error) {
	return invokeRPC(s, target, "StopRportFwdListener", func(ctx context.Context, request *commonpb.Request) (*sliverpb.RportFwdListener, error) {
		return s.rpc.StopRportFwdListener(ctx, &sliverpb.RportFwdStopListenerReq{ID: listenerID, Request: request})
	}, func(response *sliverpb.RportFwdListener) *commonpb.Response { return response.GetResponse() })
}

func validateRportFwdListener(listener *sliverpb.RportFwdListener, listenerID uint32, bindAddress string, forwardAddress string, authorizationID string) error {
	if listener == nil {
		return errors.New("empty listener metadata")
	}
	if listener.GetID() != listenerID {
		return fmt.Errorf("ID got %d, want %d", listener.GetID(), listenerID)
	}
	if listener.GetBindAddress() != bindAddress {
		return fmt.Errorf("bind address got %q, want %q", listener.GetBindAddress(), bindAddress)
	}
	if listener.GetForwardAddress() != forwardAddress {
		return fmt.Errorf("forward address got %q, want %q", listener.GetForwardAddress(), forwardAddress)
	}
	if authorizationID != "" && listener.GetAuthorizationID() != authorizationID {
		return fmt.Errorf("authorization ID got %q, want %q", listener.GetAuthorizationID(), authorizationID)
	}
	return nil
}

func findRportFwdListener(listeners []*sliverpb.RportFwdListener, listenerID uint32) *sliverpb.RportFwdListener {
	for _, listener := range listeners {
		if listener.GetID() == listenerID {
			return listener
		}
	}
	return nil
}

func waitForReverseForwardRoundTrip(ctx context.Context, address string, payload []byte) error {
	ticker := time.NewTicker(listenerPollInterval)
	defer ticker.Stop()

	var lastErr error
	for {
		attemptCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		err := reverseForwardRoundTrip(attemptCtx, address, payload)
		cancel()
		if err == nil {
			return nil
		}
		lastErr = err

		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for reverse-forward round trip to %s: %w (last error: %v)", address, ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

func reverseForwardRoundTrip(ctx context.Context, address string, payload []byte) error {
	dialer := net.Dialer{Timeout: 2 * time.Second}
	connection, err := dialer.DialContext(ctx, "tcp4", address)
	if err != nil {
		return fmt.Errorf("dial implant listener %s: %w", address, err)
	}
	defer connection.Close()
	if deadline, ok := ctx.Deadline(); ok {
		if err := connection.SetDeadline(deadline); err != nil {
			return fmt.Errorf("set reverse-forward deadline: %w", err)
		}
	}

	written, err := connection.Write(payload)
	if err != nil {
		return fmt.Errorf("write %d-byte payload: %w", len(payload), err)
	}
	if written != len(payload) {
		return fmt.Errorf("write payload: %w (%d of %d bytes)", io.ErrShortWrite, written, len(payload))
	}
	echoed := make([]byte, len(payload))
	if _, err := io.ReadFull(connection, echoed); err != nil {
		return fmt.Errorf("read %d-byte echo: %w", len(payload), err)
	}
	if !bytes.Equal(echoed, payload) {
		return fmt.Errorf("echo mismatch: got %d bytes, want exact %d-byte payload", len(echoed), len(payload))
	}
	return nil
}

func requireReverseForwardDialRejection(ctx context.Context, address string) error {
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
			return fmt.Errorf("implant listener %s still accepted connections after stop: %w", address, ctx.Err())
		case <-ticker.C:
		}
	}
}
