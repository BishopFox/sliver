package console

import (
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestRemoveEventListenerUnblocksInFlightDispatchAndClosesReceiver(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		con := &SliverClient{EventListeners: &sync.Map{}}
		listenerID, events := con.CreateEventListener()
		value, ok := con.EventListeners.Load(listenerID)
		if !ok {
			t.Fatal("created listener is not registered")
		}
		listener := value.(*consoleEventListener)
		buffered := &clientpb.Event{EventType: "buffered"}
		for index := 0; index < cap(listener.events); index++ {
			listener.events <- buffered
		}

		dispatchDone := make(chan struct{})
		go func() {
			event := &clientpb.Event{EventType: "in-flight"}
			con.triggerEventListeners(event)
			close(dispatchDone)
		}()

		// The full buffer and Wait establish that send is blocked before removal.
		synctest.Wait()
		select {
		case <-dispatchDone:
			t.Fatal("dispatch completed before listener removal")
		default:
		}

		removeDone := make(chan struct{})
		go func() {
			con.RemoveEventListener(listenerID)
			// Dispatch and repeated removal after closure must remain harmless.
			con.triggerEventListeners(&clientpb.Event{EventType: "removed"})
			listener.send(&clientpb.Event{EventType: "closed"})
			con.RemoveEventListener(listenerID)
			close(removeDone)
		}()
		for _, done := range []<-chan struct{}{dispatchDone, removeDone} {
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("listener removal and in-flight dispatch did not complete")
			}
		}
		if _, stillRegistered := con.EventListeners.Load(listenerID); stillRegistered {
			t.Fatal("removed listener remains registered")
		}

		deadline := time.After(time.Second)
		for received := 0; ; received++ {
			select {
			case event, open := <-events:
				if !open {
					if received != cap(listener.events) {
						t.Fatalf("received %d buffered events, want %d", received, cap(listener.events))
					}
					return
				}
				if event != buffered {
					t.Fatalf("unexpected event after removal: %v", event)
				}
			case <-deadline:
				t.Fatal("removed listener receiver did not close")
			}
		}
	})
}
