//go:build windows

package runner

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

func setBeaconTimingForTest(t *testing.T, interval time.Duration) {
	t.Helper()
	oldInterval := transports.GetInterval()
	oldJitter := transports.GetJitter()
	oldC2URI := transports.GetC2URI()
	transports.SetInterval(int64(interval))
	transports.SetJitter(1) // Int63n(1) always yields zero jitter.
	transports.SetC2URI("")
	t.Cleanup(func() {
		transports.SetInterval(oldInterval)
		transports.SetJitter(oldJitter)
		transports.SetC2URI(oldC2URI)
	})
}

func TestBeaconCheckinLoopSerializesSlowCheckins(t *testing.T) {
	const interval = 10 * time.Millisecond
	setBeaconTimingForTest(t, interval)

	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseFirst) }) }
	defer release()

	stopErr := errors.New("stop check-in loop")
	var stateMu sync.Mutex
	active := 0
	maxActive := 0
	calls := 0
	checkin := func(*transports.Beacon, time.Time, *beaconResultQueue) error {
		stateMu.Lock()
		active++
		calls++
		call := calls
		if maxActive < active {
			maxActive = active
		}
		stateMu.Unlock()
		defer func() {
			stateMu.Lock()
			active--
			stateMu.Unlock()
		}()

		switch call {
		case 1:
			close(firstStarted)
			<-releaseFirst
			return nil
		case 2:
			close(secondStarted)
			return stopErr
		default:
			return errors.New("unexpected extra check-in")
		}
	}

	done := make(chan error, 1)
	go func() {
		done <- beaconCheckinLoop(&transports.Beacon{}, &beaconResultQueue{}, checkin)
	}()

	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first check-in did not start")
	}
	overlapped := false
	select {
	case <-secondStarted:
		overlapped = true
	case <-time.After(5 * interval):
	}

	release()
	select {
	case err := <-done:
		if !errors.Is(err, stopErr) {
			t.Fatalf("got error %v, want %v", err, stopErr)
		}
	case <-time.After(time.Second):
		t.Fatal("check-in loop did not stop")
	}

	stateMu.Lock()
	defer stateMu.Unlock()
	if overlapped {
		t.Error("second check-in overlapped the first")
	}
	if calls != 2 {
		t.Fatalf("check-in called %d times, want 2", calls)
	}
	if maxActive != 1 {
		t.Fatalf("maximum concurrent check-ins = %d, want 1", maxActive)
	}
}

func TestBeaconCheckinLoopShortCircuitsReconfiguredInterval(t *testing.T) {
	const originalInterval = 500 * time.Millisecond
	setBeaconTimingForTest(t, originalInterval)

	stopErr := errors.New("stop check-in loop")
	calls := 0
	start := time.Now()
	err := beaconCheckinLoop(&transports.Beacon{}, &beaconResultQueue{}, func(*transports.Beacon, time.Time, *beaconResultQueue) error {
		calls++
		if calls == 1 {
			transports.SetInterval(int64(time.Millisecond))
			return nil
		}
		return stopErr
	})
	if !errors.Is(err, stopErr) {
		t.Fatalf("got error %v, want %v", err, stopErr)
	}
	if elapsed := time.Since(start); originalInterval/2 <= elapsed {
		t.Fatalf("reconfigured interval did not short circuit sleep: %s", elapsed)
	}
}

func TestBeaconResultQueuePrependPreservesOrder(t *testing.T) {
	queue := &beaconResultQueue{}
	first := &sliverpb.Envelope{ID: 1}
	second := &sliverpb.Envelope{ID: 2}
	third := &sliverpb.Envelope{ID: 3}
	queue.add(third)
	queue.prepend([]*sliverpb.Envelope{first, second})

	actual := queue.drain()
	if len(actual) != 3 || actual[0] != first || actual[1] != second || actual[2] != third {
		t.Fatalf("unexpected result order: %+v", actual)
	}
	if remaining := queue.drain(); len(remaining) != 0 {
		t.Fatalf("drain left %d result(s)", len(remaining))
	}
}

func TestBeaconMainResultCheckinDoesNotWaitForTasks(t *testing.T) {
	queue := &beaconResultQueue{}
	queue.add(&sliverpb.Envelope{ID: 41, Data: []byte("result")})

	var recvCalled bool
	beacon := &transports.Beacon{
		Start: func() error { return nil },
		Send:  func(*sliverpb.Envelope) error { return nil },
		Recv: func() (*sliverpb.Envelope, error) {
			recvCalled = true
			return nil, errors.New("unexpected receive")
		},
		Close: func() error { return nil },
	}

	if err := beaconMain(beacon, time.Now(), queue); err != nil {
		t.Fatal(err)
	}
	if recvCalled {
		t.Fatal("result-only check-in waited for a server task response")
	}
	if remaining := queue.drain(); len(remaining) != 0 {
		t.Fatalf("successful check-in retained %d result(s)", len(remaining))
	}
}

func TestBeaconMainEmptyCheckinWaitsForTasks(t *testing.T) {
	queue := &beaconResultQueue{}
	var recvCalled bool
	beacon := &transports.Beacon{
		Start: func() error { return nil },
		Send:  func(*sliverpb.Envelope) error { return nil },
		Recv: func() (*sliverpb.Envelope, error) {
			recvCalled = true
			return nil, nil
		},
		Close: func() error { return nil },
	}

	if err := beaconMain(beacon, time.Now(), queue); err != nil {
		t.Fatal(err)
	}
	if !recvCalled {
		t.Fatal("empty check-in did not wait for a server task response")
	}
}

func TestBeaconMainRequeuesResultsAfterSendFailure(t *testing.T) {
	wantErr := errors.New("send failed")
	result := &sliverpb.Envelope{ID: 73, Data: []byte("result")}
	queue := &beaconResultQueue{}
	queue.add(result)
	beacon := &transports.Beacon{
		Start: func() error { return nil },
		Send:  func(*sliverpb.Envelope) error { return wantErr },
		Recv:  func() (*sliverpb.Envelope, error) { return nil, nil },
		Close: func() error { return nil },
	}

	if err := beaconMain(beacon, time.Now(), queue); !errors.Is(err, wantErr) {
		t.Fatalf("got error %v, want %v", err, wantErr)
	}
	actual := queue.drain()
	if len(actual) != 1 || actual[0] != result {
		t.Fatalf("send failure did not preserve exact result: %+v", actual)
	}
}
