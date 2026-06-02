package c2

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
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/0x90pkt/trigger/pkg/protocol"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/core"
)

// TestSmoke_TriggerListenerDispatchesStopJob exercises the full
// Sliver-side path end-to-end:
//
//	factory  ->  bind UDP  ->  receive trigger  ->  HMAC verify  ->
//	task dispatch  ->  StopJob handler  ->  non-blocking JobCtrl send.
//
// It uses a fake core.Job as the StopJob target. A successful run
// proves the imported listener + Sliver-side glue work together.
//
// Pinned port (no t.Parallel), tag gates: server go_sqlite.
func TestSmoke_TriggerListenerDispatchesStopJob(t *testing.T) {
	const (
		port   = 14629
		secret = "smoke-test-secret-xxxxxxxxxxxxxx"
	)

	// 1. Fake target job. The StopJob handler will look it up by name
	//    via core.Jobs.All() and signal its JobCtrl.
	targetJob := &core.Job{
		ID:      core.NextJobID(),
		Name:    "smoke-target",
		JobCtrl: make(chan bool, 1),
	}
	core.Jobs.Add(targetJob)
	t.Cleanup(func() { core.Jobs.Remove(targetJob) })

	// 2. Start the trigger listener with a stop-job binding for the
	//    fake target.
	req := &clientpb.TriggerListenerReq{
		Host:         "127.0.0.1",
		Port:         port,
		SharedSecret: []byte(secret),
		ServerID:     "smoke-server",
		Intents: []*clientpb.TriggerIntentBinding{{
			Name: "kill-target",
			Config: &clientpb.TriggerIntentBinding_StopJob{
				StopJob: &clientpb.StopJobConfig{JobName: targetJob.Name},
			},
		}},
	}
	triggerJob, err := StartTriggerListenerJob(req)
	if err != nil {
		t.Fatalf("StartTriggerListenerJob: %v", err)
	}
	t.Cleanup(func() {
		// Non-blocking — defensive in case the listener already shut down.
		select {
		case triggerJob.JobCtrl <- true:
		default:
		}
	})

	// 3. Give the listener a moment to bind (the factory spawns its
	//    Start goroutine async).
	time.Sleep(150 * time.Millisecond)

	// 4. Sign + send a real UDP trigger packet.
	if err := sendSmokeTrigger(t, fmt.Sprintf("127.0.0.1:%d", port), secret, "smoke-operator", "kill-target"); err != nil {
		t.Fatalf("sendSmokeTrigger: %v", err)
	}

	// 5. The StopJob handler should fire and signal the target's
	//    JobCtrl. Wait for it.
	select {
	case got := <-targetJob.JobCtrl:
		if !got {
			t.Fatalf("JobCtrl received %v, want true", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("target job's JobCtrl never fired — handler dispatch broken")
	}
}

func sendSmokeTrigger(t *testing.T, addr, secret, clientID, intent string) error {
	t.Helper()
	nonce, err := protocol.GenerateNonce()
	if err != nil {
		return fmt.Errorf("nonce: %w", err)
	}
	msg := protocol.TriggerMessage{
		Version:   protocol.ProtocolVersion,
		ClientID:  clientID,
		Nonce:     nonce,
		Timestamp: protocol.NowUTC(),
		Intent:    intent,
	}
	sig, err := protocol.Sign(msg, secret)
	if err != nil {
		return fmt.Errorf("sign: %w", err)
	}
	msg.Signature = sig
	payload, err := protocol.EncodeWire(msg)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("resolve: %w", err)
	}
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()
	if _, err := conn.Write(payload); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}
