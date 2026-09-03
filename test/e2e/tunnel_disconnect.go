package e2e

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

const tunnelSessionDisconnectTimeout = 20 * time.Second

// exercisePortfwdSocks5Disconnect is deliberately the last focused scenario
// for a transport: it destroys the session while both tunnel features have an
// established relay, then proves remote teardown drains those relays before
// the local listeners are explicitly removed.
func (s *suite) exercisePortfwdSocks5Disconnect(target implantTarget, transport string, process *managedProcess) error {
	started := time.Now()
	err := s.exercisePortfwdSocks5DisconnectLifecycle(target, transport, process)
	duration := time.Since(started)
	s.recordTunnelScenario(transport, "portfwd", "active-session-disconnect", duration, err)
	s.recordTunnelScenario(transport, "socks5", "active-session-disconnect", duration, err)
	if err != nil {
		return fmt.Errorf("Portfwd/SOCKS5 (active session disconnect, %s, %s): %w", transport, duration.Round(time.Millisecond), err)
	}
	s.t.Logf("PASS %s/%s %s session Portfwd+SOCKS5/active-session-disconnect (%s)", s.opts.targetOS, s.opts.targetArch, transport, duration.Round(time.Millisecond))
	return nil
}

//nolint:gocyclo // Portfwd and SOCKS teardown must be observed within the same session-disconnect lifecycle.
func (s *suite) exercisePortfwdSocks5DisconnectLifecycle(target implantTarget, transport string, process *managedProcess) (resultErr error) {
	if target.session == nil || target.beacon != nil || process == nil {
		return errors.New("portfwd/SOCKS5 disconnect E2E requires a managed session process")
	}
	portForwardEcho, err := startTCPEchoServer()
	if err != nil {
		return fmt.Errorf("start port-forward disconnect echo service: %w", err)
	}
	defer func() { resultErr = errors.Join(resultErr, portForwardEcho.close()) }()
	socksEcho, err := startTCPEchoServer()
	if err != nil {
		return fmt.Errorf("start SOCKS5 disconnect echo service: %w", err)
	}
	defer func() { resultErr = errors.Join(resultErr, socksEcho.close()) }()

	forward, err := s.startPortForward(target, portForwardEcho.address(), 30*time.Second)
	if err != nil {
		return fmt.Errorf("start disconnect port forward: %w", err)
	}
	defer func() { resultErr = errors.Join(resultErr, forward.stop()) }()
	proxyServer, err := s.startSocksE2EProxy(target.session, "", "", "")
	if err != nil {
		return fmt.Errorf("start disconnect SOCKS5 proxy: %w", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), socksE2ECaseTimeout)
		defer cancel()
		resultErr = errors.Join(resultErr, proxyServer.stop(stopCtx))
	}()

	setupCtx, setupCancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
	portForwardConnection, err := (&net.Dialer{}).DialContext(setupCtx, "tcp4", forward.bindAddress)
	if err != nil {
		setupCancel()
		return fmt.Errorf("dial active port-forward disconnect relay: %w", err)
	}
	defer func() { _ = portForwardConnection.Close() }()
	socksConnection, err := socksE2EDial(setupCtx, proxyServer.address, "", "", socksEcho.address())
	if err != nil {
		setupCancel()
		return fmt.Errorf("dial active SOCKS5 disconnect relay: %w", err)
	}
	defer func() { _ = socksConnection.Close() }()
	if err := tunnelEchoRoundTripOnConn(setupCtx, portForwardConnection, []byte("portfwd-active-before-session-disconnect\x00\xff"), []int{1, 7, 4096}); err != nil {
		setupCancel()
		return fmt.Errorf("port-forward active relay before disconnect: %w", err)
	}
	if err := tunnelEchoRoundTripOnConn(setupCtx, socksConnection, []byte("socks-active-before-session-disconnect\x00\xff"), []int{3, 1, 4096}); err != nil {
		setupCancel()
		return fmt.Errorf("SOCKS5 active relay before disconnect: %w", err)
	}
	_ = portForwardConnection.SetDeadline(time.Time{})
	_ = socksConnection.SetDeadline(time.Time{})
	if err := waitForEchoConnectionCount(setupCtx, portForwardEcho, 1); err != nil {
		setupCancel()
		return fmt.Errorf("port-forward target relay readiness: %w", err)
	}
	if err := waitForEchoConnectionCount(setupCtx, socksEcho, 1); err != nil {
		setupCancel()
		return fmt.Errorf("SOCKS5 target relay readiness: %w", err)
	}
	setupCancel()

	cursor := s.hub.cursor()
	if transport == "http" || transport == "wg" {
		closeCtx, closeCancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
		_, closeErr := s.rpc.CloseSession(closeCtx, &sliverpb.CloseSession{Request: target.request(s.opts.commandTimeout)})
		closeCancel()
		if closeErr != nil {
			return fmt.Errorf("request %s session close with active tunnels: %w", transport, closeErr)
		}
		if err := process.stop(); err != nil {
			return fmt.Errorf("stop %s implant after server close: %w", transport, err)
		}
	} else if err := process.stop(); err != nil {
		return fmt.Errorf("abruptly stop %s implant with active tunnels: %w", transport, err)
	}

	teardownCtx, teardownCancel := context.WithTimeout(s.ctx, tunnelSessionDisconnectTimeout)
	defer teardownCancel()
	if _, _, err := s.hub.wait(teardownCtx, cursor, func(event *clientpb.Event) bool {
		return event.EventType == consts.SessionClosedEvent && event.Session != nil && event.Session.ID == target.session.ID
	}); err != nil {
		return fmt.Errorf("wait for active-tunnel session closed event: %w", err)
	}
	if err := requireTunnelConnectionClosed(teardownCtx, portForwardConnection, "port-forward"); err != nil {
		return err
	}
	if err := requireTunnelConnectionClosed(teardownCtx, socksConnection, "SOCKS5"); err != nil {
		return err
	}
	if err := waitForEchoConnectionCount(teardownCtx, portForwardEcho, 0); err != nil {
		return fmt.Errorf("port-forward target relay drain: %w", err)
	}
	if err := waitForEchoConnectionCount(teardownCtx, socksEcho, 0); err != nil {
		return fmt.Errorf("SOCKS5 target relay drain: %w", err)
	}
	if err := waitForSocksE2EClientTunnelDrain(teardownCtx); err != nil {
		return fmt.Errorf("SOCKS5 client tunnel drain after session disconnect: %w", err)
	}

	// The established relays above must close because the session disappeared.
	// Local listener lifecycle remains operator-owned, so remove each exact proxy
	// only after proving that remote teardown path.
	if err := forward.stop(); err != nil {
		return fmt.Errorf("stop disconnect port forward: %w", err)
	}
	if err := proxyServer.stop(teardownCtx); err != nil {
		return fmt.Errorf("stop disconnect SOCKS5 proxy: %w", err)
	}
	if err := requireTCPDialRejection(teardownCtx, forward.bindAddress); err != nil {
		return fmt.Errorf("port-forward listener after disconnect cleanup: %w", err)
	}
	if err := requireTCPDialRejection(teardownCtx, proxyServer.address); err != nil {
		return fmt.Errorf("SOCKS5 listener after disconnect cleanup: %w", err)
	}
	if findPortForwardMetadata(forward.id) != nil {
		return fmt.Errorf("disconnect cleanup retained port forward %d", forward.id)
	}
	if findSocksE2EMetadata(proxyServer.id) != nil {
		return fmt.Errorf("disconnect cleanup retained SOCKS5 proxy %d", proxyServer.id)
	}
	return nil
}

func requireTunnelConnectionClosed(ctx context.Context, connection net.Conn, label string) error {
	if deadline, ok := ctx.Deadline(); ok {
		_ = connection.SetReadDeadline(deadline)
	}
	buffer := make([]byte, 1)
	count, readErr := connection.Read(buffer)
	if readErr == nil || count != 0 {
		return fmt.Errorf("%s relay survived expected teardown (%d bytes, error %v)", label, count, readErr)
	}
	if networkErr, ok := readErr.(net.Error); ok && networkErr.Timeout() {
		return fmt.Errorf("%s relay did not close before expected teardown deadline: %w", label, readErr)
	}
	return nil
}
