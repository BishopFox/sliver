package console

import (
	"sync"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestRemoveEventListenerUnblocksInFlightDispatchAndClosesReceiver(t *testing.T) {
	con := &SliverClient{EventListeners: &sync.Map{}}
	listenerID, events := con.CreateEventListener()
	value, ok := con.EventListeners.Load(listenerID)
	if !ok {
		t.Fatal("created listener is not registered")
	}
	listener := value.(*consoleEventListener)
	for index := 0; index < cap(listener.events); index++ {
		listener.events <- &clientpb.Event{EventType: "buffered"}
	}

	dispatchDone := make(chan struct{})
	go func() {
		listener.send(&clientpb.Event{EventType: "in-flight"})
		close(dispatchDone)
	}()

	con.RemoveEventListener(listenerID)
	select {
	case <-dispatchDone:
	case <-time.After(time.Second):
		t.Fatal("listener removal did not unblock in-flight dispatch")
	}
	if _, stillRegistered := con.EventListeners.Load(listenerID); stillRegistered {
		t.Fatal("removed listener remains registered")
	}

	for range events {
	}
}

func TestConcurrentEventDispatchAndRemovalDoesNotPanic(t *testing.T) {
	for iteration := 0; iteration < 1_000; iteration++ {
		con := &SliverClient{EventListeners: &sync.Map{}}
		listenerID, events := con.CreateEventListener()
		dispatchDone := make(chan struct{})
		go func() {
			con.triggerEventListeners(&clientpb.Event{EventType: "race"})
			close(dispatchDone)
		}()
		con.RemoveEventListener(listenerID)
		<-dispatchDone
		for range events {
		}
	}
}
