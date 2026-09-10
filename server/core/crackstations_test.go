package core

import (
	"sync"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestCrackstationSnapshotIsolatedDuringUpdates(t *testing.T) {
	station := NewCrackstation(&clientpb.Crackstation{
		HostUUID:   "11111111-1111-4111-8111-111111111111",
		Benchmarks: map[int32]uint64{0: 1},
	})
	station.UpdateStatus(&clientpb.CrackstationStatus{State: clientpb.States_IDLE})
	first := station.Snapshot()
	first.Benchmarks[0] = 999
	first.Status.State = clientpb.States_CRACKING
	second := station.Snapshot()
	if second.Benchmarks[0] != 1 || second.Status.State != clientpb.States_IDLE {
		t.Fatalf("snapshot mutation leaked into station: %#v", second)
	}

	var workers sync.WaitGroup
	for index := 0; index < 20; index++ {
		workers.Add(2)
		go func(rate uint64) {
			defer workers.Done()
			station.UpdateBenchmarks(map[int32]uint64{0: rate})
		}(uint64(index + 1))
		go func() {
			defer workers.Done()
			_ = station.Snapshot()
		}()
	}
	workers.Wait()
	final := station.Snapshot()
	if len(final.Benchmarks) != 1 || final.Status == nil {
		t.Fatalf("invalid final snapshot: %#v", final)
	}
}
