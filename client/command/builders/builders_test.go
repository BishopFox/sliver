package builders

import (
	"strings"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestBuilderStatusColumns(t *testing.T) {
	now := time.Unix(1_700_000_120, 0)
	cpu, memory, age, availability := builderStatusColumns(&clientpb.BuilderStatus{
		EffectiveCPUCores:    8,
		AvailableMemoryBytes: 1_499_999,
		ReportedUnix:         now.Add(-90 * time.Second).Unix(),
		Online:               true,
		Available:            true,
	}, now)
	if cpu != "8" || memory != "1" || age != "1m30s" || availability != "available" {
		t.Fatalf("unexpected status columns: %q %q %q %q", cpu, memory, age, availability)
	}

	_, memory, _, availability = builderStatusColumns(&clientpb.BuilderStatus{
		AvailableMemoryBytes: 1_500_000,
		ReportedUnix:         now.Unix(),
		Online:               true,
	}, now)
	if memory != "2" || availability != "unavailable" {
		t.Fatalf("rounded MB or availability mismatch: %q %q", memory, availability)
	}

	_, _, age, availability = builderStatusColumns(nil, now)
	if age != "never" || availability != "unavailable" {
		t.Fatalf("missing report columns: %q %q", age, availability)
	}
}

func TestBuilderSpecSupportIncludesExactDigests(t *testing.T) {
	first := &clientpb.BuildSpecRef{Name: "builtin/sliver", Version: "1", Digest: "sha256:aaaaaaaa"}
	second := &clientpb.BuildSpecRef{Name: "builtin/donut", Version: "1", Digest: "sha256:bbbbbbbb"}
	got := builderSpecSupport([]*clientpb.BuildSpecRef{first, second})
	if !strings.Contains(got, first.Digest) || !strings.Contains(got, second.Digest) ||
		strings.Index(got, second.Name) > strings.Index(got, first.Name) {
		t.Fatalf("unexpected spec support: %q", got)
	}
}
