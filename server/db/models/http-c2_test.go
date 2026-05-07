package models

import (
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestRandomSampleEmptyValues(t *testing.T) {
	sample := randomSample(nil, 1, 3)
	if len(sample) != 0 {
		t.Fatalf("expected empty sample, got %d entries", len(sample))
	}
}

func TestRandomPathSegmentsWithNoSegments(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("RandomPathSegments panicked with empty input: %v", recovered)
		}
	}()

	config := &clientpb.HTTPC2ImplantConfig{
		MinFileGen: 1,
		MaxFileGen: 3,
		MinPathGen: 0,
		MaxPathGen: 3,
	}
	segments := RandomPathSegments(config)
	if len(segments) != 0 {
		t.Fatalf("expected no path segments, got %d", len(segments))
	}
}
