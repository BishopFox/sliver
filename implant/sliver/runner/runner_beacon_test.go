//go:build windows

package runner

import (
	"errors"
	"testing"
	"time"

	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

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
