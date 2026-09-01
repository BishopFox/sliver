package handlers

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/core/rtunnels"
	"google.golang.org/protobuf/proto"
)

func TestLegacyCapabilityReverseTunnelQueuesFinalDataWithoutRacingTerminalEnvelope(t *testing.T) {
	registry, dialer, broker := newRecordingReverseForwardBroker(t)
	connection, session := addBufferedTestSession(t)
	session.Capabilities = 0 // A pre-terminal-sequence implant.
	authorizationID := beginActiveAuthorization(t, registry, session.ID, "127.0.0.1:4701", 31)
	tunnelID := core.NewTunnelID()

	sendReverseTunnelRequest(t, connection, broker, &sliverpb.TunnelData{
		TunnelID:      tunnelID,
		CreateReverse: true,
		Rportfwd: &sliverpb.RPortfwd{
			AuthorizationID: authorizationID.String(),
		},
	})
	peer := recordingReverseTunnelPeer(t, dialer)
	if err := peer.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set target write deadline: %v", err)
	}
	finalData := []byte("final bytes before target EOF")
	if _, err := peer.Write(finalData); err != nil {
		t.Fatalf("write final target data: %v", err)
	}
	if err := peer.Close(); err != nil {
		t.Fatalf("close target peer: %v", err)
	}

	envelope := receiveReverseTunnelEnvelope(t, connection)
	if envelope.Type != sliverpb.MsgTunnelData {
		t.Fatalf("first envelope type = %d, want %d", envelope.Type, sliverpb.MsgTunnelData)
	}
	data := &sliverpb.TunnelData{}
	if err := proto.Unmarshal(envelope.Data, data); err != nil {
		t.Fatalf("unmarshal final tunnel data: %v", err)
	}
	if data.TunnelID != tunnelID || data.Sequence != 0 || !bytes.Equal(data.Data, finalData) {
		t.Fatalf("final tunnel data = %+v, want tunnel=%d sequence=0 data=%q", data, tunnelID, finalData)
	}

	waitForReverseTunnelRemoval(t, tunnelID)
	select {
	case unexpected := <-connection.Send:
		t.Fatalf("legacy implant received racing terminal envelope type %d after final data", unexpected.Type)
	default:
	}
}

func TestTunnelTerminalCapabilityQueuesSequencedCloseAfterFinalData(t *testing.T) {
	registry, dialer, broker := newRecordingReverseForwardBroker(t)
	connection, session := addBufferedTestSession(t)
	session.Capabilities = sliverpb.CapabilityTunnelTerminalV1
	authorizationID := beginActiveAuthorization(t, registry, session.ID, "127.0.0.1:4702", 32)
	tunnelID := core.NewTunnelID()

	sendReverseTunnelRequest(t, connection, broker, &sliverpb.TunnelData{
		TunnelID:      tunnelID,
		CreateReverse: true,
		Rportfwd: &sliverpb.RPortfwd{
			AuthorizationID: authorizationID.String(),
		},
	})
	peer := recordingReverseTunnelPeer(t, dialer)
	if err := peer.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set target write deadline: %v", err)
	}
	finalData := []byte("sequenced final bytes")
	if _, err := peer.Write(finalData); err != nil {
		t.Fatalf("write final target data: %v", err)
	}
	if err := peer.Close(); err != nil {
		t.Fatalf("close target peer: %v", err)
	}

	dataEnvelope := receiveReverseTunnelEnvelope(t, connection)
	closeEnvelope := receiveReverseTunnelEnvelope(t, connection)
	if dataEnvelope.Type != sliverpb.MsgTunnelData {
		t.Fatalf("first envelope type = %d, want %d", dataEnvelope.Type, sliverpb.MsgTunnelData)
	}
	if closeEnvelope.Type != sliverpb.MsgTunnelClose {
		t.Fatalf("second envelope type = %d, want %d", closeEnvelope.Type, sliverpb.MsgTunnelClose)
	}
	data := &sliverpb.TunnelData{}
	if err := proto.Unmarshal(dataEnvelope.Data, data); err != nil {
		t.Fatalf("unmarshal tunnel data: %v", err)
	}
	closed := &sliverpb.TunnelData{}
	if err := proto.Unmarshal(closeEnvelope.Data, closed); err != nil {
		t.Fatalf("unmarshal tunnel close: %v", err)
	}
	if data.TunnelID != tunnelID || data.Sequence != 0 || !bytes.Equal(data.Data, finalData) {
		t.Fatalf("final tunnel data = %+v, want tunnel=%d sequence=0 data=%q", data, tunnelID, finalData)
	}
	if !closed.Closed || closed.TunnelID != tunnelID || closed.Sequence != 1 {
		t.Fatalf("terminal envelope = %+v, want closed tunnel=%d sequence=1", closed, tunnelID)
	}
	waitForReverseTunnelRemoval(t, tunnelID)
}

func recordingReverseTunnelPeer(t *testing.T, dialer *recordingReverseForwardDialer) net.Conn {
	t.Helper()
	dialer.mutex.Lock()
	defer dialer.mutex.Unlock()
	if len(dialer.peers) != 1 {
		t.Fatalf("recording dialer peers = %d, want 1", len(dialer.peers))
	}
	return dialer.peers[0]
}

func receiveReverseTunnelEnvelope(t *testing.T, connection *core.ImplantConnection) *sliverpb.Envelope {
	t.Helper()
	select {
	case envelope := <-connection.Send:
		return envelope
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for reverse-tunnel envelope")
		return nil
	}
}

func waitForReverseTunnelRemoval(t *testing.T, tunnelID uint64) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for rtunnels.GetRTunnel(tunnelID) != nil && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if tunnel := rtunnels.GetRTunnel(tunnelID); tunnel != nil {
		closeTestReverseTunnel(tunnelID)
		t.Fatalf("reverse tunnel %d remained published after target EOF", tunnelID)
	}
}
