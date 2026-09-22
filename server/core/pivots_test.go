package core

import (
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/cryptography"
)

func TestPivotStartStopsWhenEitherConnectionCloses(t *testing.T) {
	tests := []struct {
		name                    string
		close                   func(*Pivot)
		wantImmediateConnection bool
	}{
		{name: "pivot", close: func(pivot *Pivot) { pivot.ImplantConn.Close() }, wantImmediateConnection: true},
		{name: "immediate", close: func(pivot *Pivot) { pivot.ImmediateImplantConn.Close() }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pivot := NewPivotSession([]*sliverpb.PivotPeer{{PeerID: 1}, {PeerID: 2}})
			pivot.ImplantConn = NewImplantConnection(PivotTransportName, "pivot")
			pivot.ImmediateImplantConn = NewImplantConnection("mtls", "immediate")
			pivot.CipherCtx = cryptography.NewCipherContext(cryptography.RandomSymmetricKey())
			PivotSessions.Store(pivot.ID, pivot)
			t.Cleanup(func() {
				pivot.ImplantConn.Close()
				pivot.ImmediateImplantConn.Close()
				PivotSessions.CompareAndDelete(pivot.ID, pivot)
			})
			pivot.Start()

			// Exercise the guarded forwarding send without leaving an unbounded
			// producer behind if the pivot shuts down first.
			enqueued := make(chan bool, 1)
			go func() {
				select {
				case pivot.ImplantConn.Send <- &sliverpb.Envelope{Type: sliverpb.MsgPing}:
					enqueued <- true
				case <-pivot.ImplantConn.Done():
					enqueued <- false
				}
			}()
			select {
			case sent := <-enqueued:
				if !sent {
					t.Fatal("pivot closed before the send loop received the envelope")
				}
			case <-time.After(time.Second):
				t.Fatal("pivot send loop did not receive envelope")
			}

			test.close(pivot)
			select {
			case <-pivot.ImplantConn.Done():
			case <-time.After(time.Second):
				t.Fatal("pivot connection survived lifecycle close")
			}
			select {
			case <-pivot.ImmediateImplantConn.Done():
				if test.wantImmediateConnection {
					t.Fatal("closing a synthetic pivot closed its shared immediate transport connection")
				}
			default:
				if !test.wantImmediateConnection {
					t.Fatal("immediate transport connection did not close")
				}
			}
			deadline := time.Now().Add(time.Second)
			for time.Now().Before(deadline) {
				if _, ok := PivotSessions.Load(pivot.ID); !ok {
					return
				}
				time.Sleep(time.Millisecond)
			}
			t.Fatal("pivot map entry survived lifecycle close")
		})
	}
}
