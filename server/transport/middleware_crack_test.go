package transport

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestSanitizeAuditRequestRemovesCrackTaskSecretsWithoutMutation(t *testing.T) {
	request := &clientpb.CrackTask{
		ID:               "task-id",
		HostUUID:         "host-id",
		Attempt:          7,
		LeaseToken:       "super-secret-token",
		Command:          &clientpb.CrackCommand{Hashes: []string{"secret-hash"}, PositionalArguments: []string{"secret-target"}},
		Stdout:           []byte("secret-stdout"),
		Stderr:           []byte("secret-stderr"),
		LatestStatusJSON: []byte(`{"secret":"status"}`),
		RecoveredJSON:    []byte(`[{"hash":"secret-hash","plaintext":"c2VjcmV0LXBsYWludGV4dA=="}]`),
	}
	original := proto.Clone(request).(*clientpb.CrackTask)
	sanitized, ok := sanitizeAuditRequest("/rpcpb.SliverRPC/CrackTaskUpdate", request).(*clientpb.CrackTask)
	if !ok {
		t.Fatal("sanitized request has unexpected type")
	}
	if !proto.Equal(request, original) {
		t.Fatal("audit sanitization mutated the handler request")
	}
	serialized, err := json.Marshal(sanitized)
	if err != nil {
		t.Fatalf("marshal sanitized request: %v", err)
	}
	text := string(serialized)
	for _, secret := range []string{
		"super-secret-token", "secret-hash", "secret-target", "secret-stdout", "secret-stderr", "secret-plaintext",
		base64.StdEncoding.EncodeToString([]byte("secret-stdout")),
		base64.StdEncoding.EncodeToString([]byte("secret-stderr")),
		base64.StdEncoding.EncodeToString(request.LatestStatusJSON),
		base64.StdEncoding.EncodeToString(request.RecoveredJSON),
	} {
		if strings.Contains(text, secret) {
			t.Fatalf("sanitized audit request contains secret %q: %s", secret, text)
		}
	}
	if !strings.Contains(text, "task-id") || !strings.Contains(text, `"Attempt":7`) {
		t.Fatalf("sanitized audit request lost task identity: %s", text)
	}

	byID := sanitizeAuditRequest("/rpcpb.SliverRPC/CrackTaskByID", request).(*clientpb.CrackTask)
	if byID.LeaseToken != "" || byID.ID != request.ID || byID.Attempt != request.Attempt {
		t.Fatalf("CrackTaskByID sanitization = %#v", byID)
	}
}

func TestSanitizeAuditRequestClearsTaskStatusEventDataOnly(t *testing.T) {
	data := []byte(`{"task_id":"task-id","attempt":4,"lease_token":"secret-token","status":{"hash":"secret-hash","target":"secret-target","plaintext":"secret-plaintext"}}`)
	event := &clientpb.Event{EventType: consts.CrackTaskStatus, Data: data}
	original := proto.Clone(event).(*clientpb.Event)
	sanitized := sanitizeAuditRequest("/rpcpb.SliverRPC/CrackstationTrigger", event).(*clientpb.Event)
	if !proto.Equal(event, original) {
		t.Fatal("event audit sanitization mutated the handler request")
	}
	if sanitized.EventType != consts.CrackTaskStatus || len(sanitized.Data) != 0 {
		t.Fatalf("sanitized task status event = %#v", sanitized)
	}
	serialized, err := json.Marshal(sanitized)
	if err != nil {
		t.Fatalf("marshal sanitized event: %v", err)
	}
	if strings.Contains(string(serialized), "secret") || strings.Contains(string(serialized), base64.StdEncoding.EncodeToString(data)) {
		t.Fatalf("sanitized event leaked data: %s", serialized)
	}

	unrelated := &clientpb.Event{EventType: consts.CrackStatusEvent, Data: []byte("coarse-status")}
	if got := sanitizeAuditRequest("/rpcpb.SliverRPC/CrackstationTrigger", unrelated); got != unrelated {
		t.Fatal("unrelated trigger request was cloned or changed")
	}
	job := &clientpb.CrackJob{ID: "job-id"}
	if got := sanitizeAuditRequest("/rpcpb.SliverRPC/CrackJobByID", job); got != job {
		t.Fatal("unrelated request was cloned or changed")
	}
}

func TestSanitizeAuditRequestRemovesCrackCommandAndChunkContentsWithoutMutation(t *testing.T) {
	brainPasswordV7 := "secret-brain-password-v7"
	hashMode := uint32(100)
	request := &clientpb.CrackCommand{
		AttackMode:                clientpb.CrackAttackMode_HYBRID_WORDLIST_MASK,
		HashType:                  clientpb.HashType_SHA1,
		HashMode:                  &hashMode,
		Hashes:                    []string{"secret-hash"},
		MarkovHcstat2:             []byte("secret-markov"),
		Session:                   "secret-session",
		RestoreFile:               []byte("secret-restore"),
		Outfile:                   "secret-outfile",
		Potfile:                   []byte("secret-potfile"),
		DebugFile:                 "secret-debug-file",
		InductionDir:              "secret-induction-dir",
		OutfileCheckDir:           "secret-outfile-check-dir",
		KeyboardLayoutMapping:     []byte("secret-keyboard-map"),
		Separator:                 "!",
		TruecryptKeyfiles:         "secret-truecrypt-keyfile",
		VeracryptKeyfiles:         "secret-veracrypt-keyfile",
		RuleLeft:                  "secret-left-rule",
		RuleRight:                 "secret-right-rule",
		RulesFile:                 []byte("secret-rules-file"),
		CustomCharset1:            "secret-charset-1",
		CustomCharset2:            "secret-charset-2",
		CustomCharset3:            "secret-charset-3",
		CustomCharset4:            "secret-charset-4",
		CustomCharset5:            "secret-charset-5",
		CustomCharset6:            "secret-charset-6",
		CustomCharset7:            "secret-charset-7",
		CustomCharset8:            "secret-charset-8",
		Identify:                  "secret-legacy-operand",
		BrainHost:                 "secret-brain-host",
		BrainPassword:             "secret-brain-password",
		BrainPasswordV7:           &brainPasswordV7,
		BrainSession:              "secret-brain-session",
		BrainSessionWhitelist:     "secret-brain-whitelist",
		BrainClientFeatures:       "secret-brain-features",
		BrainSessionWhitelistV7:   []uint32{12345},
		SeekDBPath:                "secret-seek-db",
		BridgeParameter1:          "secret-bridge-1",
		BridgeParameter2:          "secret-bridge-2",
		BridgeParameter3:          "secret-bridge-3",
		BridgeParameter4:          "secret-bridge-4",
		Lookup:                    "secret-lookup",
		GenerateRulesFuncSel:      "secret-generate-rules-functions",
		EncryptWithPubkey:         "secret-public-key",
		PositionalArguments:       []string{"secret-wordlist", "secret-mask"},
		RulesFilesV7:              [][]byte{[]byte("secret-rules-v7")},
		EncodingFromName:          "secret-source-encoding",
		EncodingToName:            "secret-target-encoding",
		Stdin:                     []byte("secret-stdin"),
		CredentialIDs:             []string{"secret-credential-id"},
		CredentialCollection:      "secret-credential-collection",
		BackendDevices:            []uint32{1, 3},
		IncludeCrackedCredentials: true,
	}
	original := proto.Clone(request).(*clientpb.CrackCommand)
	sanitized, ok := sanitizeAuditRequest("/rpcpb.SliverRPC/Crack", request).(*clientpb.CrackCommand)
	if !ok {
		t.Fatal("sanitized crack request has unexpected type")
	}
	if !proto.Equal(request, original) {
		t.Fatal("crack audit sanitization mutated the handler request")
	}
	serialized, err := json.Marshal(sanitized)
	if err != nil {
		t.Fatalf("marshal sanitized crack request: %v", err)
	}
	text := string(serialized)
	if strings.Contains(text, "secret-") {
		t.Fatalf("sanitized crack request contains plaintext content: %s", text)
	}
	sanitized.ProtoReflect().Range(func(field protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		if field.Kind() == protoreflect.StringKind || field.Kind() == protoreflect.BytesKind {
			t.Errorf("sanitized crack request retained content field %s", field.FullName())
		}
		return true
	})
	for _, content := range [][]byte{
		request.MarkovHcstat2,
		request.RestoreFile,
		request.Potfile,
		request.KeyboardLayoutMapping,
		request.RulesFile,
		request.RulesFilesV7[0],
		request.Stdin,
	} {
		if encoded := base64.StdEncoding.EncodeToString(content); strings.Contains(text, encoded) {
			t.Fatalf("sanitized crack request contains base64 content %q: %s", encoded, text)
		}
	}
	if sanitized.AttackMode != request.AttackMode || sanitized.HashType != request.HashType || sanitized.HashMode == nil || *sanitized.HashMode != *request.HashMode || !proto.Equal(&clientpb.CrackCommand{BackendDevices: sanitized.BackendDevices}, &clientpb.CrackCommand{BackendDevices: request.BackendDevices}) {
		t.Fatalf("sanitized crack request lost nonsecret execution metadata: %#v", sanitized)
	}

	chunk := &clientpb.CrackFileChunk{ID: "chunk-id", CrackFileID: "file-id", N: 7, Data: []byte("secret-chunk-data")}
	originalChunk := proto.Clone(chunk).(*clientpb.CrackFileChunk)
	sanitizedChunk := sanitizeAuditRequest("/rpcpb.SliverRPC/CrackFileChunkUpload", chunk).(*clientpb.CrackFileChunk)
	if !proto.Equal(chunk, originalChunk) {
		t.Fatal("chunk audit sanitization mutated the handler request")
	}
	if sanitizedChunk.ID != chunk.ID || sanitizedChunk.CrackFileID != chunk.CrackFileID || sanitizedChunk.N != chunk.N || len(sanitizedChunk.Data) != 0 {
		t.Fatalf("sanitized chunk request = %#v", sanitizedChunk)
	}
	chunkJSON, err := json.Marshal(sanitizedChunk)
	if err != nil {
		t.Fatalf("marshal sanitized chunk request: %v", err)
	}
	if strings.Contains(string(chunkJSON), "secret-chunk-data") || strings.Contains(string(chunkJSON), base64.StdEncoding.EncodeToString(chunk.Data)) {
		t.Fatalf("sanitized chunk request leaked data: %s", chunkJSON)
	}
}

func TestCrackPayloadLoggingDecidersAlwaysSuppressSensitiveRPCs(t *testing.T) {
	originalUnary := serverConfig.Logs.GRPCUnaryPayloads
	originalStream := serverConfig.Logs.GRPCStreamPayloads
	serverConfig.Logs.GRPCUnaryPayloads = true
	serverConfig.Logs.GRPCStreamPayloads = true
	t.Cleanup(func() {
		serverConfig.Logs.GRPCUnaryPayloads = originalUnary
		serverConfig.Logs.GRPCStreamPayloads = originalStream
	})
	for _, method := range []string{
		"/rpcpb.SliverRPC/Crack",
		"/rpcpb.SliverRPC/CrackJobByID",
		"/rpcpb.SliverRPC/CrackTaskByID",
		"/rpcpb.SliverRPC/CrackTaskUpdate",
		"/rpcpb.SliverRPC/CrackstationTrigger",
		"/rpcpb.SliverRPC/CrackFileChunkUpload",
		"/rpcpb.SliverRPC/CrackFileChunkDownload",
	} {
		if deciderUnary(context.Background(), method, nil) {
			t.Fatalf("payload logging enabled for %s", method)
		}
	}
	if deciderStream(context.Background(), "/rpcpb.SliverRPC/CrackstationRegister", nil) {
		t.Fatal("payload logging enabled for CrackstationRegister stream")
	}
	for _, method := range []string{
		"/rpcpb.SliverRPC/CrackJobs",
		"/rpcpb.SliverRPC/CrackFilesList",
		"/rpcpb.SliverRPC/Crackstations",
	} {
		if !deciderUnary(context.Background(), method, nil) {
			t.Fatalf("metadata-only payload logging was disabled for %s", method)
		}
	}
	if !deciderStream(context.Background(), "/rpcpb.SliverRPC/Events", nil) {
		t.Fatal("unrelated stream payload logging was disabled")
	}
}
