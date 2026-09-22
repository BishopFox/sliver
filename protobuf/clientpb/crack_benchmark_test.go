package clientpb

import (
	"testing"

	"google.golang.org/protobuf/proto"
)

func TestCrackBenchmarkFreshnessFieldsRoundTrip(t *testing.T) {
	want := &CrackBenchmark{
		HostUUID:       "11111111-1111-4111-8111-111111111111",
		Benchmarks:     map[int32]uint64{0: 123},
		SchemaVersion:  1,
		HashcatVersion: "v7.1.2",
	}
	data, err := proto.Marshal(want)
	if err != nil {
		t.Fatalf("marshal benchmark: %v", err)
	}
	got := &CrackBenchmark{}
	if err := proto.Unmarshal(data, got); err != nil {
		t.Fatalf("unmarshal benchmark: %v", err)
	}
	if !proto.Equal(got, want) {
		t.Fatalf("benchmark round trip = %#v, want %#v", got, want)
	}
}
