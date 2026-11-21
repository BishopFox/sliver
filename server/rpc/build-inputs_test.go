package rpc

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestBuildInputsRemainEphemeralAndTyped(t *testing.T) {
	secret := []byte("private-stage-key")
	request := &clientpb.BuildRequest{
		Target: &clientpb.BuildTarget{GOOS: "windows", GOARCH: "amd64"}, Mode: "session", Format: "stage",
		C2:     []*clientpb.BuildEndpoint{{Protocol: "mtls", Address: "mtls://example.invalid:8888"}},
		Inputs: []*clientpb.BuildInput{{Kind: "secret", Name: "stage/key", Data: secret}},
	}
	neutral, err := buildRequestFromProto(request)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(neutral)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, secret) || bytes.Contains(encoded, []byte("stage/key")) {
		t.Fatal("per-build material leaked into the neutral plan request")
	}
	resolve, cleanup, err := buildInputResolver(request.Inputs)
	if err != nil {
		t.Fatal(err)
	}
	secret[0] = '!'
	value, err := resolve("secret", "stage/key")
	if err != nil || string(value) != "private-stage-key" {
		t.Fatalf("resolver did not isolate input bytes: %v", err)
	}
	value[0] = '?'
	value, err = resolve("secret", "stage/key")
	if err != nil || string(value) != "private-stage-key" {
		t.Fatalf("resolver returned mutable shared bytes: %v", err)
	}
	if _, err := resolve("input-artifact", "stage/key"); err == nil {
		t.Fatal("resolver accepted wrong input kind")
	}
	cleanup()
	if _, err := resolve("secret", "stage/key"); err == nil {
		t.Fatal("cleaned-up input remained resolvable")
	}
}

func TestBuildInputsRejectDuplicatesAndInvalidMaterial(t *testing.T) {
	for _, inputs := range [][]*clientpb.BuildInput{
		{{Kind: "secret", Name: "key", Data: []byte("first")}, {Kind: "secret", Name: "key", Data: []byte("second")}},
		{{Kind: "secret", Name: "key", Data: nil}},
		{{Kind: "secret", Name: "key", Data: bytes.Repeat([]byte("x"), 257)}},
		{{Kind: "command", Name: "key", Data: []byte("run")}},
		{{Kind: "secret", Name: "\nkey", Data: []byte("value")}},
	} {
		if _, _, err := buildInputResolver(inputs); err == nil {
			t.Fatalf("invalid material was accepted: %+v", inputs)
		}
	}
}
