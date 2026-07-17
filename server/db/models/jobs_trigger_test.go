package models

/*
	Tests for the TriggerListener model's proto-marshal-to-bytes
	persistence scheme. The TriggerListenerReq carries a discriminated-
	union of task bindings (WakeBeacon | StopJob | Exec | ReverseShell)
	plus opaque secret bytes — we store the proto.Marshal'd bytes in
	one Conf column rather than normalizing into ~5 child tables.

	These tests verify the round-trip:
	  TriggerListenerReq → ListenerJobFromProtobuf → TriggerListener.Conf
	                                                      ↓
	                                                  proto.Unmarshal
	                                                      ↓
	  matches original proto.Marshal output (every field preserved)
*/

import (
	"bytes"
	"testing"

	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"google.golang.org/protobuf/proto"
)

// makeTriggerReq builds a TriggerListenerReq exercising every shape
// we'd realistically persist: per-client keys, task bindings with all
// four kinds, source allowlist, tunables.
func makeTriggerReq() *clientpb.TriggerListenerReq {
	return &clientpb.TriggerListenerReq{
		Host:         "0.0.0.0",
		Port:         14629,
		SharedSecret: []byte("default-secret-32-bytes-of-data!"),
		PerClientKeys: map[string][]byte{
			"ops-alice": []byte("alice-secret-bytes"),
			"ops-bob":   []byte("bob-secret-bytes"),
		},
		Strict:           true,
		AllowedClientIDs: []string{"ops-alice", "ops-bob"},
		AllowedSources:   []string{"10.0.0.0/8", "192.168.1.42"},
		ServerID:         "test-trigger-server",
		Intents: []*clientpb.TriggerIntentBinding{
			{
				Name: "wake-beacon-alpha",
				Config: &clientpb.TriggerIntentBinding_WakeBeacon{
					WakeBeacon: &clientpb.WakeBeaconConfig{
						BeaconID: "deadbeef-dead-beef-dead-beefdeadbeef",
					},
				},
			},
			{
				Name: "stop-mtls",
				Config: &clientpb.TriggerIntentBinding_StopJob{
					StopJob: &clientpb.StopJobConfig{
						JobName: "mtls",
					},
				},
			},
			{
				Name: "run-ls",
				Config: &clientpb.TriggerIntentBinding_Exec{
					Exec: &clientpb.ExecConfig{
						Cmd:  "/bin/ls",
						Args: []string{"-la", "/tmp"},
					},
				},
			},
			{
				Name: "callback-shell",
				Config: &clientpb.TriggerIntentBinding_ReverseShell{
					ReverseShell: &clientpb.ReverseShellConfig{
						OperatorAddr: "10.0.0.1:4444",
					},
				},
			},
		},
		Workers:                    4,
		MaxClockSkewSeconds:        60,
		ReplayTTLSeconds:           300,
		MaxMessageBytes:            4096,
		GlobalRatePerSecond:        100,
		PerClientRequestsPerMinute: 60,
		MaxReplayEntries:           10000,
		MaxRateLimitEntries:        1000,
		HandlerTimeoutMs:           5000,
	}
}

func TestTriggerListener_RoundTripViaModel(t *testing.T) {
	original := makeTriggerReq()

	listenerJob := &clientpb.ListenerJob{
		Type:        constants.TriggerStr,
		JobID:       42,
		TriggerConf: original,
	}

	// Marshal via the model — same path SaveC2Listener uses
	dbModel := ListenerJobFromProtobuf(listenerJob)
	if dbModel.Type != constants.TriggerStr {
		t.Fatalf("Type = %q, want %q", dbModel.Type, constants.TriggerStr)
	}
	if len(dbModel.TriggerListener.Conf) == 0 {
		t.Fatalf("TriggerListener.Conf is empty — marshaling failed silently")
	}

	// Unmarshal via the model — same path daemon-restore uses
	roundTripped := dbModel.TriggerListener.ToProtobuf()
	if roundTripped == nil {
		t.Fatal("ToProtobuf returned nil after a successful marshal")
	}

	// Use proto.Equal for semantic comparison. proto.Marshal output is
	// NOT deterministic when the message contains maps (PerClientKeys)
	// because Go map iteration order is randomized. bytes.Equal was
	// flaky here; proto.Equal compares field-by-field correctly.
	if !proto.Equal(original, roundTripped) {
		t.Fatalf("round-trip not proto-equal")
	}

	// Spot-check specific fields survived
	if roundTripped.Host != "0.0.0.0" || roundTripped.Port != 14629 {
		t.Errorf("host:port lost: %q:%d", roundTripped.Host, roundTripped.Port)
	}
	if !bytes.Equal(roundTripped.SharedSecret, []byte("default-secret-32-bytes-of-data!")) {
		t.Errorf("SharedSecret bytes mangled")
	}
	if len(roundTripped.Intents) != 4 {
		t.Fatalf("Intents lost: got %d, want 4", len(roundTripped.Intents))
	}
	// Each task kind preserved
	if roundTripped.Intents[0].GetWakeBeacon().GetBeaconID() != "deadbeef-dead-beef-dead-beefdeadbeef" {
		t.Errorf("WakeBeacon binding lost")
	}
	if roundTripped.Intents[1].GetStopJob().GetJobName() != "mtls" {
		t.Errorf("StopJob binding lost")
	}
	if roundTripped.Intents[2].GetExec().GetCmd() != "/bin/ls" {
		t.Errorf("Exec binding lost")
	}
	if roundTripped.Intents[3].GetReverseShell().GetOperatorAddr() != "10.0.0.1:4444" {
		t.Errorf("ReverseShell binding lost")
	}
}

func TestTriggerListener_ToProtobufEmptyConfReturnsNil(t *testing.T) {
	// A non-trigger ListenerJob (e.g. mtls) has a zero-value
	// TriggerListener attached. ToProtobuf must return nil so the
	// resulting *clientpb.ListenerJob has TriggerConf=nil (matching
	// the existing pattern for HTTPConf, MTLSConf, etc.).
	tl := &TriggerListener{Conf: nil}
	if got := tl.ToProtobuf(); got != nil {
		t.Fatalf("ToProtobuf with empty Conf returned non-nil: %+v", got)
	}
}

func TestTriggerListener_ToProtobufMalformedConfReturnsNil(t *testing.T) {
	// Defensive: if Conf is somehow malformed, return nil rather than
	// panic. (This shouldn't happen — we wrote the bytes via
	// proto.Marshal — but be robust.)
	tl := &TriggerListener{Conf: []byte("not a valid proto message")}
	if got := tl.ToProtobuf(); got != nil {
		t.Fatalf("ToProtobuf with malformed Conf returned non-nil: %+v", got)
	}
}

func TestListenerJobFromProtobuf_TriggerType(t *testing.T) {
	// Verify the switch-case for TriggerStr is wired and produces a
	// non-empty Conf.
	req := makeTriggerReq()
	pb := &clientpb.ListenerJob{
		Type:        constants.TriggerStr,
		JobID:       7,
		TriggerConf: req,
	}
	model := ListenerJobFromProtobuf(pb)
	if model.Type != constants.TriggerStr {
		t.Fatalf("Type = %q, want %q", model.Type, constants.TriggerStr)
	}
	if model.JobID != 7 {
		t.Errorf("JobID = %d, want 7", model.JobID)
	}
	if len(model.TriggerListener.Conf) == 0 {
		t.Fatalf("TriggerListener.Conf is empty")
	}
}

func TestListenerJobFromProtobuf_TriggerTypeNilConf(t *testing.T) {
	// If TriggerConf is nil (e.g. someone constructed a Type=trigger
	// listener without populating the config), the resulting model
	// has an empty Conf — restore path will then fail with "missing
	// Trigger listener configuration", which is the intended outcome.
	pb := &clientpb.ListenerJob{
		Type:        constants.TriggerStr,
		JobID:       7,
		TriggerConf: nil,
	}
	model := ListenerJobFromProtobuf(pb)
	if len(model.TriggerListener.Conf) != 0 {
		t.Errorf("expected empty Conf when input TriggerConf is nil, got %d bytes", len(model.TriggerListener.Conf))
	}
}
