package rpc

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"google.golang.org/protobuf/proto"
)

const (
	writeTimeout                = 5 * time.Second
	socksClientBindTimeout      = 10 * time.Second
	socksClientBindPollInterval = 10 * time.Millisecond
	socksLifecycleCheckInterval = 4 * time.Second
	socksFirstPayloadTimeout    = 10 * time.Second
	socksLegacyIdleTimeout      = 15 * time.Second
	// Legacy HTTP implants send data and an unsequenced terminal through
	// independently in-flight POSTs. Match the generic tunnel reorder grace so
	// a delayed payload cannot be overtaken under a loaded or latent C2 path.
	socksLegacyTerminalReorderGrace = 10 * time.Second
	// Current clients use an empty frame at an unreachable sequence as an
	// ownership and terminal-support marker. An old server retains this one
	// out-of-order frame but still processes real SOCKS data from sequence zero,
	// making the extension backward compatible without protobuf field churn.
	socksLifecycleBindSequence = ^uint64(0)
)

func isSocksOwnershipFrame(frame *sliverpb.SocksData) bool {
	if frame == nil || frame.Ack != 0 || frame.CloseConn || len(frame.Data) != 0 {
		return false
	}
	// Sequence zero is retained for compatibility with the first iteration of
	// lifecycle-aware clients. MaxUint64 is the backward-safe wire encoding.
	return frame.Sequence == 0 || frame.Sequence == socksLifecycleBindSequence
}

func isCanonicalClientSocksAcknowledgement(frame *sliverpb.SocksData) bool {
	return frame != nil && frame.Ack != 0 && len(frame.Data) == 0 && !frame.CloseConn &&
		frame.Sequence == 0 && frame.Capabilities == 0 && frame.Username == "" && frame.Password == ""
}

func isSocksTunnelResourcePressure(err error) bool {
	return errors.Is(err, core.ErrTunnelIngressLimit) || errors.Is(err, core.ErrTunnelPendingBytes)
}

// SocksProxy relays SOCKS5 frames between one operator stream and implant tunnels.
func (s *Server) SocksProxy(stream rpcpb.SliverRPC_SocksProxyServer) error {
	return s.socksProxy(stream, core.SocksTunnels.Get, socksLegacyTerminalReorderGrace, nil, nil)
}

//nolint:gocyclo // SOCKS framing, ownership, and teardown form one stream state machine.
func (s *Server) socksProxy(
	stream rpcpb.SliverRPC_SocksProxyServer,
	getTunnel func(uint64) *core.TcpTunnel,
	legacyTerminalGrace time.Duration,
	legacyTerminalExpiry <-chan time.Time,
	legacyTerminalWaitStarted chan<- uint64,
) (resultErr error) {
	ctx, cancel := context.WithCancel(stream.Context())
	client := core.NewSocksClient(stream)
	var workers sync.WaitGroup
	var workerErr error
	var workerErrMu sync.Mutex
	reportWorkerError := func(err error) {
		if err == nil {
			return
		}
		workerErrMu.Lock()
		if workerErr == nil {
			workerErr = err
		}
		workerErrMu.Unlock()
		cancel()
	}
	getWorkerError := func() error {
		workerErrMu.Lock()
		defer workerErrMu.Unlock()
		return workerErr
	}

	// Cancellation must happen before waiting for workers. Tunnel cleanup runs
	// after they stop, and CloseIf makes it safe to race other lifecycle owners.
	defer func() {
		cancel()
		workers.Wait()
		for _, tunnel := range core.SocksTunnels.List() {
			if tunnel.Client() == client {
				rpcLog.Infof("Cleaning up tunnel %d on proxy closure", tunnel.ID)
				s.closeSocksTunnelForClientTeardown(tunnel)
			}
		}
		if resultErr == nil {
			resultErr = getWorkerError()
		}
	}()

	receiveResults := receiveStreamFrames[*sliverpb.SocksData](ctx, stream)
	for {
		var fromClient *sliverpb.SocksData
		select {
		case <-ctx.Done():
			if err := getWorkerError(); err != nil {
				return err
			}
			return ctx.Err()
		case <-client.Done():
			if err := client.Err(); err != nil {
				return fmt.Errorf("SOCKS client stream failed: %w", err)
			}
			return errors.New("SOCKS client stream failed")
		case received, ok := <-receiveResults:
			if !ok {
				if err := getWorkerError(); err != nil {
					return err
				}
				if err := ctx.Err(); err != nil {
					return err
				}
				return nil
			}
			if received.err == io.EOF {
				return nil
			}
			if received.err != nil {
				rpcLog.Warnf("Error on stream recv %s", received.err)
				return rpcError(received.err)
			}
			fromClient = received.data
		}
		if fromClient == nil {
			continue
		}
		if fromClient.Request == nil || fromClient.Request.SessionID == "" {
			rpcLog.Warnf("Ignoring SOCKS frame with missing request metadata for tunnel %d", fromClient.TunnelID)
			continue
		}

		tunnel := getTunnel(fromClient.TunnelID)
		if tunnel == nil {
			// CreateSocks can race session teardown before the client's zero-byte
			// ownership bind arrives. Return an exact terminal so that late bind
			// cannot strand its local TCP connection waiting for a tunnel that no
			// longer exists.
			if err := notifySocksClient(client, &sliverpb.SocksData{
				TunnelID:  fromClient.TunnelID,
				CloseConn: true,
			}, writeTimeout); err != nil {
				return fmt.Errorf("notify client of missing SOCKS tunnel %d: %w", fromClient.TunnelID, err)
			}
			if len(fromClient.Data) > core.MaxSocksFrameBytes {
				return fmt.Errorf("SOCKS operator frame: %w: got %d bytes, limit %d", core.ErrTunnelFrameTooLarge, len(fromClient.Data), core.MaxSocksFrameBytes)
			}
			continue
		}
		if fromClient.Request.SessionID != tunnel.SessionID {
			rpcLog.Warnf("Ignoring SOCKS frame for tunnel %d owned by session %s from session %s", fromClient.TunnelID, tunnel.SessionID, fromClient.Request.SessionID)
			continue
		}
		currentSession := core.Sessions.Get(tunnel.SessionID)
		if currentSession == nil || currentSession.Connection != tunnel.ImplantConnection() {
			rpcLog.Warnf("Closing SOCKS tunnel %d because its exact owning session generation is gone", fromClient.TunnelID)
			if notifyErr := s.rejectSocksTunnel(tunnel, client); notifyErr != nil {
				return fmt.Errorf("notify client after SOCKS session teardown for tunnel %d: %w", tunnel.ID, notifyErr)
			}
			continue
		}
		if fromClient.Ack != 0 {
			if tunnel.Client() != client {
				rpcLog.Warnf("Ignoring SOCKS acknowledgement for tunnel %d owned by another proxy stream", tunnel.ID)
				continue
			}
			if !isCanonicalClientSocksAcknowledgement(fromClient) {
				rpcLog.Warnf("Closing SOCKS tunnel %d after malformed operator acknowledgement", tunnel.ID)
				if notifyErr := s.rejectSocksTunnel(tunnel, client); notifyErr != nil {
					return fmt.Errorf("malformed operator SOCKS acknowledgement for tunnel %d (client terminal: %v)", tunnel.ID, notifyErr)
				}
				continue
			}
			if err := tunnel.RelayClientAcknowledgement(client, fromClient.Ack); err != nil {
				if errors.Is(err, core.ErrTunnelClosed) || errors.Is(err, core.ErrSocksOwner) {
					continue
				}
				rpcLog.Warnf("Closing SOCKS tunnel %d after invalid operator acknowledgement: %v", tunnel.ID, err)
				if notifyErr := s.rejectSocksTunnel(tunnel, client); notifyErr != nil {
					return fmt.Errorf("invalid operator SOCKS acknowledgement for tunnel %d: %w (client terminal: %v)", tunnel.ID, err, notifyErr)
				}
			}
			continue
		}
		ownershipFrame := isSocksOwnershipFrame(fromClient)
		owned, newlyBound, bindErr := tunnel.BindClientWithNegotiatedCapabilities(client, fromClient.Username, fromClient.Password, ownershipFrame, fromClient.Capabilities)
		if bindErr != nil {
			if errors.Is(bindErr, core.ErrTunnelClosed) {
				// Another lifecycle owner retired this exact tunnel after lookup.
				// The race is tunnel-local and must not tear down sibling tunnels on
				// the shared operator stream. An unbound client still needs an exact
				// terminal because the winning close owner could not discover it.
				if tunnel.Client() != client {
					if notifyErr := notifySocksClient(client, &sliverpb.SocksData{TunnelID: tunnel.ID, CloseConn: true}, writeTimeout); notifyErr != nil {
						return fmt.Errorf("notify client after closed SOCKS bind race for tunnel %d: %w", tunnel.ID, notifyErr)
					}
				}
				continue
			}
			rpcLog.Warnf("Closing SOCKS tunnel %d after invalid ownership bind: %v", tunnel.ID, bindErr)
			if notifyErr := s.rejectSocksTunnel(tunnel, client); notifyErr != nil {
				return fmt.Errorf("invalid SOCKS ownership bind for tunnel %d: %w (client terminal: %v)", tunnel.ID, bindErr, notifyErr)
			}
			return fmt.Errorf("invalid SOCKS ownership bind for tunnel %d: %w", tunnel.ID, bindErr)
		}
		if !owned {
			rpcLog.Warnf("Ignoring SOCKS frame for tunnel %d owned by another proxy stream", fromClient.TunnelID)
			continue
		}
		if !ownershipFrame && fromClient.Capabilities != 0 {
			rpcLog.Warnf("Closing SOCKS tunnel %d after capability metadata outside ownership bind", tunnel.ID)
			if notifyErr := s.rejectSocksTunnel(tunnel, client); notifyErr != nil {
				return fmt.Errorf("invalid SOCKS capability metadata for tunnel %d (client terminal: %v)", tunnel.ID, notifyErr)
			}
			continue
		}
		if len(fromClient.Data) > core.MaxSocksFrameBytes {
			rpcLog.Warnf("Closing SOCKS tunnel %d after oversized operator frame (%d bytes)", tunnel.ID, len(fromClient.Data))
			if notifyErr := s.rejectSocksTunnel(tunnel, client); notifyErr != nil {
				return fmt.Errorf("SOCKS operator frame: %w: got %d bytes, limit %d (client terminal: %v)", core.ErrTunnelFrameTooLarge, len(fromClient.Data), core.MaxSocksFrameBytes, notifyErr)
			}
			return fmt.Errorf("SOCKS operator frame: %w: got %d bytes, limit %d", core.ErrTunnelFrameTooLarge, len(fromClient.Data), core.MaxSocksFrameBytes)
		}
		if newlyBound {
			s.startSocksTunnelWorkers(
				ctx,
				&workers,
				tunnel,
				client,
				reportWorkerError,
				legacyTerminalGrace,
				legacyTerminalExpiry,
				legacyTerminalWaitStarted,
			)
		}
		// Ownership frames are out-of-band and do not consume a data sequence.
		if ownershipFrame {
			continue
		}
		if err := tunnel.AdmitToImplantContext(ctx, fromClient); err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			if errors.Is(err, core.ErrTunnelClosed) {
				// Close can wake a capacity waiter after another tunnel has already
				// queued work on this shared stream. Continue so the sibling frame is
				// not converted into a stream-wide failure.
				continue
			}
			if isSocksTunnelResourcePressure(err) {
				rpcLog.Warnf("Closing SOCKS tunnel %d after operator ingress resource pressure: %v", tunnel.ID, err)
				if notifyErr := s.rejectSocksTunnel(tunnel, client); notifyErr != nil {
					return fmt.Errorf("operator SOCKS resource pressure on tunnel %d: %w (client terminal: %v)", tunnel.ID, err, notifyErr)
				}
				continue
			}
			rpcLog.Warnf("Closing SOCKS tunnel %d after invalid operator frame: %v", tunnel.ID, err)
			if notifyErr := s.rejectSocksTunnel(tunnel, client); notifyErr != nil {
				return fmt.Errorf("invalid operator SOCKS frame for tunnel %d: %w (client terminal: %v)", tunnel.ID, err, notifyErr)
			}
			return fmt.Errorf("invalid operator SOCKS frame for tunnel %d: %w", tunnel.ID, err)
		}
	}
}

func (s *Server) startSocksTunnelWorkers(
	ctx context.Context,
	workers *sync.WaitGroup,
	tunnel *core.TcpTunnel,
	client *core.SocksClient,
	report func(error),
	legacyTerminalGrace time.Duration,
	legacyTerminalExpiry <-chan time.Time,
	legacyTerminalWaitStarted chan<- uint64,
) {
	start := func(name string, run func()) {
		workers.Add(1)
		go func(workerName string, worker func()) {
			defer workers.Done()
			defer func() {
				if recovered := recover(); recovered != nil {
					report(fmt.Errorf("%s panic: %v", workerName, recovered))
				}
			}()
			worker()
		}(name, run)
	}
	start("SOCKS lifecycle monitor", func() {
		s.monitorSocksTunnel(ctx, tunnel)
	})
	start("SOCKS client sender", func() {
		s.sendSocksDataToClient(ctx, tunnel, client, report)
	})
	start("SOCKS implant sender", func() {
		s.sendSocksDataToImplant(ctx, tunnel, report)
	})
	start("SOCKS legacy terminal scheduler", func() {
		s.scheduleLegacySocksTerminal(ctx, tunnel, legacyTerminalGrace, legacyTerminalExpiry, legacyTerminalWaitStarted)
	})
}

type legacySocksTerminalWaitResult uint8

const (
	legacySocksTerminalWaitStopped legacySocksTerminalWaitResult = iota
	legacySocksTerminalWaitRefreshed
	legacySocksTerminalWaitExpired
)

func waitForLegacySocksTerminalExpiry(
	ctx context.Context,
	tunnel *core.TcpTunnel,
	delay time.Duration,
	expiry <-chan time.Time,
	changed <-chan struct{},
) legacySocksTerminalWaitResult {
	if expiry != nil {
		select {
		case <-ctx.Done():
			return legacySocksTerminalWaitStopped
		case <-tunnel.Done():
			return legacySocksTerminalWaitStopped
		case <-changed:
			return legacySocksTerminalWaitRefreshed
		case <-expiry:
			return legacySocksTerminalWaitExpired
		}
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return legacySocksTerminalWaitStopped
	case <-tunnel.Done():
		return legacySocksTerminalWaitStopped
	case <-changed:
		return legacySocksTerminalWaitRefreshed
	case <-timer.C:
		return legacySocksTerminalWaitExpired
	}
}

// scheduleLegacySocksTerminal owns one exact capability-zero tunnel's bounded
// close-reorder window. An expiry only materializes terminal EOF when no later
// implant data advanced the observed generation; queue pressure waits for the
// relay worker rather than dropping already accepted bytes.
//
//nolint:gocyclo // Generation refresh, bounded expiry, pressure, and teardown form one actor.
func (s *Server) scheduleLegacySocksTerminal(
	ctx context.Context,
	tunnel *core.TcpTunnel,
	quietPeriod time.Duration,
	expiry <-chan time.Time,
	waitStarted chan<- uint64,
) {
	if tunnel == nil {
		return
	}
	if quietPeriod <= 0 {
		quietPeriod = socksLegacyTerminalReorderGrace
	}
	select {
	case <-ctx.Done():
		return
	case <-tunnel.Done():
		return
	case <-tunnel.LegacyImplantTerminalPending():
	}
terminalLoop:
	for {
		pending, generation, changed := tunnel.LegacyImplantTerminalState()
		if !pending {
			return
		}
		if waitStarted != nil {
			select {
			case <-ctx.Done():
				return
			case <-tunnel.Done():
				return
			case waitStarted <- generation:
			}
		}
		switch waitForLegacySocksTerminalExpiry(ctx, tunnel, quietPeriod, expiry, changed) {
		case legacySocksTerminalWaitStopped:
			return
		case legacySocksTerminalWaitRefreshed:
			continue
		case legacySocksTerminalWaitExpired:
		}
		for {
			spaceChanged := tunnel.FromImplantSpaceChange()
			done, err := tunnel.TryFlushLegacyImplantTerminal(generation)
			if done || errors.Is(err, core.ErrTunnelClosed) {
				return
			}
			if err == nil {
				// Data refreshed the generation while this expiry was pending.
				break
			}
			if !isSocksTunnelResourcePressure(err) {
				rpcLog.Warnf("Closing SOCKS tunnel %d after legacy terminal scheduling failed: %v", tunnel.ID, err)
				s.closeSocksTunnel(tunnel)
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-tunnel.Done():
				return
			case <-changed:
				continue terminalLoop
			case <-spaceChanged:
			}
		}
	}
}

func (s *Server) monitorSocksTunnel(ctx context.Context, tunnel *core.TcpTunnel) {
	s.monitorSocksTunnelWithTimeouts(ctx, tunnel, socksFirstPayloadTimeout, socksLegacyIdleTimeout, socksLifecycleCheckInterval)
}

//nolint:gocyclo // Keep lifecycle deadlines and generation checks in one monitor loop.
func (s *Server) monitorSocksTunnelWithTimeouts(ctx context.Context, tunnel *core.TcpTunnel, firstPayloadTimeout time.Duration, legacyIdleTimeout time.Duration, checkInterval time.Duration) {
	if tunnel == nil {
		return
	}
	implantConnection := tunnel.ImplantConnection()
	session := core.Sessions.Get(tunnel.SessionID)
	if session == nil || implantConnection == nil || session.Connection != implantConnection {
		s.closeSocksTunnel(tunnel)
		return
	}
	if firstPayloadTimeout <= 0 {
		firstPayloadTimeout = socksFirstPayloadTimeout
	}
	if legacyIdleTimeout <= 0 {
		legacyIdleTimeout = socksLegacyIdleTimeout
	}
	if checkInterval <= 0 {
		checkInterval = socksLifecycleCheckInterval
	}
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tunnel.Done():
			return
		case <-implantConnection.Done():
			s.closeSocksTunnel(tunnel)
			return
		case now := <-ticker.C:
			if core.SocksTunnels.Get(tunnel.ID) != tunnel {
				return
			}
			currentSession := core.Sessions.Get(tunnel.SessionID)
			if currentSession != session || currentSession.Connection != implantConnection {
				s.closeSocksTunnel(tunnel)
				return
			}
			lifecycle := tunnel.ClientLifecycle()
			if lifecycle.BoundAt.IsZero() {
				continue
			}
			if !lifecycle.ReceivedPayload && now.Sub(lifecycle.BoundAt) >= firstPayloadTimeout {
				rpcLog.Debugf("Closing SOCKS tunnel %d because no payload arrived within %s", tunnel.ID, firstPayloadTimeout)
				s.closeSocksTunnel(tunnel)
				return
			}
			if lifecycle.ReceivedPayload && (!lifecycle.SendsTerminal || !tunnel.FlowControlEnabled()) && now.Sub(lifecycle.LastActivity) >= legacyIdleTimeout {
				rpcLog.Debugf("Closing legacy-compatible SOCKS tunnel %d after %s without activity", tunnel.ID, legacyIdleTimeout)
				s.closeSocksTunnel(tunnel)
				return
			}
		}
	}
}

func (s *Server) sendSocksDataToClient(ctx context.Context, tunnel *core.TcpTunnel, client *core.SocksClient, report func(error)) {
	s.sendSocksDataToClientWithTimeout(ctx, tunnel, client, report, writeTimeout)
}

//nolint:gocyclo // Data, acknowledgements, terminal handling, and cancellation form one send loop.
func (s *Server) sendSocksDataToClientWithTimeout(ctx context.Context, tunnel *core.TcpTunnel, client *core.SocksClient, report func(error), sendTimeout time.Duration) {
	for {
		var tunnelData *sliverpb.SocksData
		select {
		case <-ctx.Done():
			return
		case <-tunnel.Done():
			return
		case ack := <-tunnel.AcknowledgementsToClient():
			tunnel.FromImplantMux.Lock()
			select {
			case <-ctx.Done():
				tunnel.FromImplantMux.Unlock()
				return
			case <-tunnel.Done():
				tunnel.FromImplantMux.Unlock()
				return
			default:
			}
			err := notifySocksClient(client, &sliverpb.SocksData{TunnelID: tunnel.ID, Ack: ack}, sendTimeout)
			tunnel.FromImplantMux.Unlock()
			if err != nil {
				report(fmt.Errorf("send SOCKS tunnel %d acknowledgement to client: %w", tunnel.ID, err))
				return
			}
			continue
		case tunnelData = <-tunnel.FromImplant():
		}
		if tunnelData == nil {
			continue
		}
		terminal := false
		func() {
			defer tunnel.CompleteFromImplant(tunnelData)
			tunnel.FromImplantMux.Lock()
			defer tunnel.FromImplantMux.Unlock()
			select {
			case <-ctx.Done():
				return
			case <-tunnel.Done():
				return
			default:
			}
			if tunnelData.CloseConn {
				terminal = true
				return
			}
			if err := notifySocksClient(client, &sliverpb.SocksData{
				TunnelID: tunnelData.TunnelID,
				Sequence: tunnelData.Sequence,
				Data:     tunnelData.Data,
			}, sendTimeout); err != nil {
				report(fmt.Errorf("send SOCKS tunnel %d data to client: %w", tunnel.ID, err))
				return
			}
			atomic.AddUint64(&tunnel.FromImplantSequence, 1)
		}()
		if terminal {
			// The implant has already emitted its terminal. Finalize and notify
			// the operator after releasing the directional send barrier.
			s.finishSocksTunnelAfterImplantTerminal(tunnel)
			return
		}
		if err := client.Err(); err != nil {
			return
		}
	}
}

//nolint:gocyclo // Sending, credential injection, and terminal sequencing are one state machine.
func (s *Server) sendSocksDataToImplant(ctx context.Context, tunnel *core.TcpTunnel, report func(error)) {
	for {
		var recv *sliverpb.SocksData
		select {
		case <-ctx.Done():
			return
		case <-tunnel.Done():
			return
		case ack := <-tunnel.AcknowledgementsToImplant():
			tunnel.ToImplantMux.Lock()
			select {
			case <-ctx.Done():
				tunnel.ToImplantMux.Unlock()
				return
			case <-tunnel.Done():
				tunnel.ToImplantMux.Unlock()
				return
			default:
			}
			connection := tunnel.ImplantConnection()
			data, err := proto.Marshal(&sliverpb.SocksData{TunnelID: tunnel.ID, Ack: ack})
			if err == nil && connection == nil {
				err = errors.New("missing creating connection")
			}
			if err == nil {
				err = connection.SendEnvelopeUntil(&sliverpb.Envelope{Type: sliverpb.MsgSocksData, Data: data}, tunnel.Done(), writeTimeout)
			}
			tunnel.ToImplantMux.Unlock()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				case <-tunnel.Done():
					return
				default:
				}
				if connection != nil {
					connection.Close()
				}
				report(fmt.Errorf("send SOCKS tunnel %d acknowledgement to implant: %w", tunnel.ID, err))
				return
			}
			continue
		case recv = <-tunnel.ToImplant():
		}
		if recv == nil {
			continue
		}
		stop := func() bool {
			defer tunnel.CompleteToImplant(recv)
			tunnel.ToImplantMux.Lock()
			defer tunnel.ToImplantMux.Unlock()
			select {
			case <-ctx.Done():
				return true
			case <-tunnel.Done():
				return true
			default:
			}
			connection := tunnel.ImplantConnection()
			if connection == nil {
				report(fmt.Errorf("send SOCKS tunnel %d data to implant: missing creating connection", tunnel.ID))
				return true
			}
			outbound := recv
			if recv.Sequence == 0 {
				outbound = &sliverpb.SocksData{
					TunnelID:     recv.TunnelID,
					Sequence:     recv.Sequence,
					Data:         recv.Data,
					CloseConn:    recv.CloseConn,
					Capabilities: tunnel.Capabilities(),
				}
				if !recv.CloseConn {
					outbound.Username, outbound.Password = tunnel.Credentials()
				}
			}
			data, err := proto.Marshal(outbound)
			if err != nil {
				report(fmt.Errorf("marshal SOCKS tunnel %d data: %w", tunnel.ID, err))
				return true
			}
			if err := connection.SendEnvelopeUntil(&sliverpb.Envelope{
				Type: sliverpb.MsgSocksData,
				Data: data,
			}, tunnel.Done(), writeTimeout); err != nil {
				select {
				case <-tunnel.Done():
					return true
				default:
				}
				connection.Close()
				report(fmt.Errorf("send SOCKS tunnel %d data to implant: %w", tunnel.ID, err))
				return true
			}
			if !recv.CloseConn {
				atomic.AddUint64(&tunnel.ToImplantSequence, 1)
			}
			return recv.CloseConn
		}()
		if stop {
			if recv.CloseConn {
				s.finishSocksTunnelAfterImplantTerminal(tunnel)
			} else {
				s.closeSocksTunnel(tunnel)
			}
			return
		}
	}
}

// CreateSocks5 - Create requests we close a Socks
func (s *Server) CreateSocks(ctx context.Context, req *sliverpb.Socks) (*sliverpb.Socks, error) {
	if req == nil || req.SessionID == "" {
		return nil, ErrInvalidSessionID
	}
	session := core.Sessions.Get(req.SessionID)
	if session == nil {
		return nil, ErrInvalidSessionID
	}
	capabilities := req.GetCapabilities() & session.Capabilities & sliverpb.CapabilitySocksFlowControlV1
	tunnel, err := core.SocksTunnels.CreateForSession(session, capabilities, func(tunnel *core.TcpTunnel) {
		// Sessions.Remove already won registry ownership and closed the tunnel.
		// Notify the connection stored on this exact tunnel generation before
		// finishing the operator side synchronously. The session is no longer in
		// the registry, but its transport may still be live and own the remote socket.
		s.finishClosedSocksTunnel(tunnel, tunnel.ImplantConnection(), true)
	})
	if err != nil {
		return nil, rpcError(err)
	}
	if tunnel == nil {
		return nil, ErrTunnelInitFailure
	}
	go s.monitorUnboundSocksTunnel(tunnel, session, socksClientBindTimeout)

	return &sliverpb.Socks{
		SessionID:    session.ID,
		TunnelID:     tunnel.ID,
		Capabilities: capabilities,
	}, nil
}

// monitorUnboundSocksTunnel owns the short lifecycle gap between the unary
// CreateSocks RPC and the zero-byte bind frame on SocksProxy. Once bound, the
// proxy stream and its workers become the lifecycle owners. A lease is still
// required because an operator transport can fail before the bind reaches the
// server, leaving no stream identity to associate with the new tunnel.
func (s *Server) monitorUnboundSocksTunnel(tunnel *core.TcpTunnel, session *core.Session, bindTimeout time.Duration) {
	if tunnel == nil || session == nil || session.Connection == nil {
		s.closeSocksTunnel(tunnel)
		return
	}
	if bindTimeout <= 0 {
		bindTimeout = socksClientBindTimeout
	}
	timer := time.NewTimer(bindTimeout)
	defer timer.Stop()
	poll := time.NewTicker(socksClientBindPollInterval)
	defer poll.Stop()
	for {
		select {
		case <-tunnel.Done():
			return
		case <-session.Connection.Done():
			s.closeSocksTunnel(tunnel)
			return
		case <-timer.C:
			if tunnel.Client() == nil {
				rpcLog.Debugf("Closing SOCKS tunnel %d because it was not bound within %s", tunnel.ID, bindTimeout)
				s.closeSocksTunnel(tunnel)
			}
			return
		case <-poll.C:
			if core.SocksTunnels.Get(tunnel.ID) != tunnel {
				return
			}
			if core.Sessions.Get(tunnel.SessionID) != session {
				s.closeSocksTunnel(tunnel)
				return
			}
			if tunnel.Client() != nil {
				return
			}
		}
	}
}

// CloseSocks - Client requests we close a Socks
func (s *Server) CloseSocks(_ context.Context, req *sliverpb.Socks) (*commonpb.Empty, error) {
	if req == nil {
		return &commonpb.Empty{}, nil
	}
	tunnel := core.SocksTunnels.Get(req.TunnelID)
	if tunnel == nil {
		// An already-closed tunnel is success. Ordering state belongs to the
		// detached generation and was released by its close path.
		return &commonpb.Empty{}, nil
	}
	if req.SessionID != "" && req.SessionID != tunnel.SessionID {
		return nil, ErrInvalidSessionID
	}
	s.closeSocksTunnel(tunnel)
	return &commonpb.Empty{}, nil
}

func (s *Server) closeSocksTunnel(tunnel *core.TcpTunnel) {
	s.finishSocksTunnel(tunnel, true, true)
}

func (s *Server) finishSocksTunnelAfterImplantTerminal(tunnel *core.TcpTunnel) {
	s.finishSocksTunnel(tunnel, false, true)
}

func (s *Server) closeSocksTunnelForClientTeardown(tunnel *core.TcpTunnel) {
	s.finishSocksTunnel(tunnel, true, false)
}

func (s *Server) finishSocksTunnel(tunnel *core.TcpTunnel, notifyImplant bool, notifyClient bool) {
	if tunnel == nil {
		return
	}
	var implantConnection *core.ImplantConnection
	if notifyImplant {
		implantConnection = tunnel.ImplantConnection()
	}
	if !core.SocksTunnels.CloseIf(tunnel) {
		return
	}
	s.finishClosedSocksTunnel(tunnel, implantConnection, notifyClient)
}

// finishClosedSocksTunnel completes stream protocol after core has closed and
// released the exact tunnel generation's ordering state. Session removal uses
// this entry point because core owns the registry while RPC owns notifications.
func (s *Server) finishClosedSocksTunnel(tunnel *core.TcpTunnel, implantConnection *core.ImplantConnection, notifyClient bool) {
	tunnel.ToImplantMux.Lock()
	implantClose := &sliverpb.SocksData{
		TunnelID:  tunnel.ID,
		CloseConn: true,
		Sequence:  atomic.LoadUint64(&tunnel.ToImplantSequence),
	}
	if implantClose.Sequence == 0 {
		implantClose.Capabilities = tunnel.Capabilities()
	}
	// The implant owns the remote socket, so enqueue its bounded close before
	// attempting the operator stream notification, whose Send may be stalled.
	if err := notifySocksImplant(implantConnection, implantClose, writeTimeout); err != nil {
		rpcLog.Debugf("Failed to notify implant that SOCKS tunnel %d closed: %v", tunnel.ID, err)
		// An unqueued terminal leaves the remote socket owner indeterminate. Fail
		// the exact creating transport generation closed; never rediscover it by
		// session ID, which may now identify a replacement connection.
		if implantConnection != nil {
			implantConnection.Close()
		}
	}
	tunnel.ToImplantMux.Unlock()
	if client := tunnel.Client(); notifyClient && client != nil {
		clientClose := &sliverpb.SocksData{
			TunnelID:  tunnel.ID,
			CloseConn: true,
			Sequence:  atomic.LoadUint64(&tunnel.FromImplantSequence),
		}
		if err := notifySocksClient(client, clientClose, writeTimeout); err != nil {
			rpcLog.Debugf("Failed to notify SOCKS client that tunnel %d closed: %v", tunnel.ID, err)
		}
	}
}

// rejectSocksTunnel terminates server-side protocol/capacity failures and
// always attempts an exact client terminal, including failures that occur
// before credentials can be bound onto the tunnel.
func (s *Server) rejectSocksTunnel(tunnel *core.TcpTunnel, client *core.SocksClient) error {
	if tunnel == nil {
		return nil
	}
	implantConnection := tunnel.ImplantConnection()
	if core.SocksTunnels.CloseIf(tunnel) {
		s.finishClosedSocksTunnel(tunnel, implantConnection, false)
	}
	return notifySocksClient(client, &sliverpb.SocksData{
		TunnelID:  tunnel.ID,
		CloseConn: true,
		Sequence:  atomic.LoadUint64(&tunnel.FromImplantSequence),
	}, writeTimeout)
}

// notifySocksImplant delivers a bounded terminal to the exact transport that
// owns the remote socket. Session teardown removes the registry entry before
// tunnel finalizers run, so callers must capture the connection beforehand.
func notifySocksImplant(connection *core.ImplantConnection, data *sliverpb.SocksData, timeout time.Duration) error {
	if connection == nil {
		return nil
	}
	payload, err := proto.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal SOCKS terminal: %w", err)
	}
	return connection.SendEnvelope(&sliverpb.Envelope{
		Type: sliverpb.MsgSocksData,
		Data: payload,
	}, timeout)
}

// notifySocksClient makes close notification best effort. The stream itself
// has no per-Send context, so a helper goroutine is necessary to keep a stalled
// operator from holding tunnel and session cleanup indefinitely. The gRPC
// stream context releases the sender when its owning SocksProxy exits.
func notifySocksClient(client *core.SocksClient, data *sliverpb.SocksData, timeout time.Duration) error {
	if client == nil {
		return nil
	}
	if timeout <= 0 {
		timeout = writeTimeout
	}
	result := make(chan error, 1)
	go func() {
		result <- client.Send(data)
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case err := <-result:
		return err
	case <-timer.C:
		client.Fail(context.DeadlineExceeded)
		return context.DeadlineExceeded
	}
}
