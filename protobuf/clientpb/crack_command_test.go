package clientpb

import (
	"testing"

	"google.golang.org/protobuf/proto"
)

func TestCrackCommandIgnoreLocalCacheRoundTrip(t *testing.T) {
	field := (&CrackCommand{}).ProtoReflect().Descriptor().Fields().ByNumber(174)
	if field == nil || field.Name() != "IgnoreLocalCache" {
		t.Fatalf("field 174 = %v, want IgnoreLocalCache", field)
	}

	want := &CrackCommand{IgnoreLocalCache: true}
	data, err := proto.Marshal(want)
	if err != nil {
		t.Fatalf("marshal crack command: %v", err)
	}
	got := &CrackCommand{}
	if err := proto.Unmarshal(data, got); err != nil {
		t.Fatalf("unmarshal crack command: %v", err)
	}
	if !got.GetIgnoreLocalCache() {
		t.Fatal("IgnoreLocalCache was not preserved")
	}
}
