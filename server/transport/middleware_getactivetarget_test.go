package transport

import (
	"encoding/json"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

// TestGetActiveTargetDoesNotPanicOnByteRequestField reproduces CVE-2026-XXXX:
// clientpb.BeaconTask.Request is []byte, which encoding/json marshals as a
// base64 string. getActiveTarget previously asserted that the Request field
// was a map[string]interface{}, panicking and terminating the whole process.
func TestGetActiveTargetDoesNotPanicOnByteRequestField(t *testing.T) {
	task := &clientpb.BeaconTask{
		ID:      "00000000-0000-0000-0000-000000000000",
		Request: []byte("boom"),
	}
	rawRequest, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("json.Marshal(BeaconTask) failed: %v", err)
	}

	session, beacon, err := getActiveTarget(rawRequest)
	if err != nil {
		t.Fatalf("getActiveTarget returned unexpected error: %v", err)
	}
	if session != nil || beacon != nil {
		t.Fatalf("expected no active target, got session=%v beacon=%v", session, beacon)
	}
}

// TestGetActiveTargetHandlesNonStringBeaconID ensures a non-string BeaconID
// inside the Request object is treated as "no active target" instead of
// panicking on the type assertion.
func TestGetActiveTargetHandlesNonStringBeaconID(t *testing.T) {
	request := map[string]interface{}{
		"Request": map[string]interface{}{
			"BeaconID": 12345, // not a string
		},
	}
	rawRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	session, beacon, err := getActiveTarget(rawRequest)
	if err != nil {
		t.Fatalf("getActiveTarget returned unexpected error: %v", err)
	}
	if session != nil || beacon != nil {
		t.Fatalf("expected no active target, got session=%v beacon=%v", session, beacon)
	}
}

// TestGetActiveTargetHandlesNonStringSessionID ensures a non-string SessionID
// inside the Request object is treated as "no active target" instead of
// panicking on the type assertion.
func TestGetActiveTargetHandlesNonStringSessionID(t *testing.T) {
	request := map[string]interface{}{
		"Request": map[string]interface{}{
			"SessionID": []byte("not-a-string"),
		},
	}
	rawRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	session, beacon, err := getActiveTarget(rawRequest)
	if err != nil {
		t.Fatalf("getActiveTarget returned unexpected error: %v", err)
	}
	if session != nil || beacon != nil {
		t.Fatalf("expected no active target, got session=%v beacon=%v", session, beacon)
	}
}

// TestGetActiveTargetNoRequestField ensures a request without a Request field
// yields no active target (existing behavior, regression guard).
func TestGetActiveTargetNoRequestField(t *testing.T) {
	rawRequest, err := json.Marshal(map[string]interface{}{"ID": "abc"})
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	session, beacon, err := getActiveTarget(rawRequest)
	if err != nil {
		t.Fatalf("getActiveTarget returned unexpected error: %v", err)
	}
	if session != nil || beacon != nil {
		t.Fatalf("expected no active target, got session=%v beacon=%v", session, beacon)
	}
}

// TestGetActiveTargetInvalidJSON ensures malformed request bytes produce an
// error rather than a panic.
func TestGetActiveTargetInvalidJSON(t *testing.T) {
	_, _, err := getActiveTarget([]byte("not-json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
