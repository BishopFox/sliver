package clientpb

import (
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestCrackQueryProtocolFieldNumbersAndRoundTrip(t *testing.T) {
	capabilitiesField := (&Crackstation{}).ProtoReflect().Descriptor().Fields().ByName("Capabilities")
	if capabilitiesField == nil || capabilitiesField.Number() != protoreflect.FieldNumber(104) ||
		!capabilitiesField.IsList() || capabilitiesField.Kind() != protoreflect.StringKind {
		t.Fatalf("Crackstation.Capabilities field = %v, want repeated string tag 104", capabilitiesField)
	}
	if CrackstationCapabilityCrackQueryV1 != "crack-query-v1" {
		t.Fatalf("crack query capability = %q, want crack-query-v1", CrackstationCapabilityCrackQueryV1)
	}
	commandField := (&CrackCommand{}).ProtoReflect().Descriptor().Fields().ByName("Crackstation")
	if commandField == nil || commandField.Number() != protoreflect.FieldNumber(173) {
		t.Fatalf("CrackCommand.Crackstation field = %v, want tag 173", commandField)
	}
	queryField := (&CrackResponse{}).ProtoReflect().Descriptor().Fields().ByName("Query")
	if queryField == nil || queryField.Number() != protoreflect.FieldNumber(3) {
		t.Fatalf("CrackResponse.Query field = %v, want tag 3", queryField)
	}
	if got := int32(CrackTaskKind_CRACK_TASK_QUERY); got != 4 {
		t.Fatalf("CRACK_TASK_QUERY = %d, want 4", got)
	}

	want := &CrackResponse{Query: &CrackQueryResult{
		Mode:                 CrackQueryMode_CRACK_QUERY_LOOKUP,
		CrackstationHostUUID: "3efbc42b-89d7-45d9-9746-fe8445d05cd4",
		CrackstationName:     "gpu-one",
		HashcatVersion:       "v7.1.2",
		Value:                "candidate: hashcat\nindex: 42",
		Stderr:               "rule lookup may be incomplete",
	}}
	raw, err := proto.Marshal(want)
	if err != nil {
		t.Fatalf("marshal crack query response: %v", err)
	}
	got := &CrackResponse{}
	if err := proto.Unmarshal(raw, got); err != nil {
		t.Fatalf("unmarshal crack query response: %v", err)
	}
	if !proto.Equal(got, want) {
		t.Fatalf("crack query response round trip = %v, want %v", got, want)
	}

	wantStation := &Crackstation{Capabilities: []string{CrackstationCapabilityCrackQueryV1, "future-capability"}}
	raw, err = proto.Marshal(wantStation)
	if err != nil {
		t.Fatalf("marshal crackstation capabilities: %v", err)
	}
	gotStation := &Crackstation{}
	if err := proto.Unmarshal(raw, gotStation); err != nil {
		t.Fatalf("unmarshal crackstation capabilities: %v", err)
	}
	if !proto.Equal(gotStation, wantStation) {
		t.Fatalf("crackstation capability round trip = %v, want %v", gotStation, wantStation)
	}
}
