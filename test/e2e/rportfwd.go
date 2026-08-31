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

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

const (
	reverseForwardSequentialConnections = 4
	reverseForwardConcurrentConnections = 6
	reverseForwardStopTimeout           = 20 * time.Second
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
	if listenerID == 0 {
		return errors.New("start response returned zero reverse-forward listener ID")
	}

	if err := validateRportFwdListener(listener, listenerID, bindAddress, forwardAddress, ""); err != nil {
		return fmt.Errorf("validate start metadata: %w", err)
	}
	authorizationID := listener.GetAuthorizationID()
	if authorizationID == "" {
		return errors.New("start response omitted server-issued reverse-forward authorization ID")
	}

	if err := s.localStep("RportFwd", transport+" authoritative listener metadata", func() error {
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

	if err := s.localStep("RportFwd", transport+" bidirectional echo", func() error {
		ctx, cancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
		defer cancel()
		payload := []byte("sliver-rportfwd-bidirectional-echo\x00\xff")
		return waitForReverseForwardRoundTrip(ctx, bindAddress, payload)
	}); err != nil {
		return err
	}

	if err := s.localStep("RportFwd", transport+" repeated sequential connections", func() error {
		for index := 0; index < reverseForwardSequentialConnections; index++ {
			ctx, cancel := context.WithTimeout(s.ctx, s.opts.commandTimeout)
			payload := bytes.Repeat([]byte(fmt.Sprintf("sequential-%d-", index)), 257)
			roundTripErr := reverseForwardRoundTrip(ctx, bindAddress, payload)
			cancel()
			if roundTripErr != nil {
				return fmt.Errorf("connection %d/%d: %w", index+1, reverseForwardSequentialConnections, roundTripErr)
			}
		}
		return nil
	}); err != nil {
		return err
	}

	if err := s.localStep("RportFwd", transport+" concurrent connections", func() error {
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
		return errors.Join(connectionErrors...)
	}); err != nil {
		return err
	}

	if err := s.localStep("RportFwd", transport+" stop revokes listener", func() error {
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
		return requireReverseForwardDialRejection(ctx, bindAddress)
	}); err != nil {
		return err
	}

	return nil
}

func (s *suite) startRportFwdListener(target implantTarget, bindAddress string, forwardAddress string) (*sliverpb.RportFwdListener, error) {
	return invokeRPC(s, target, "StartRportFwdListener", func(ctx context.Context, request *commonpb.Request) (*sliverpb.RportFwdListener, error) {
		return s.rpc.StartRportFwdListener(ctx, &sliverpb.RportFwdStartListenerReq{
			BindAddress:    bindAddress,
			ForwardAddress: forwardAddress,
			KeepAlive:      5,
			Request:        request,
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
