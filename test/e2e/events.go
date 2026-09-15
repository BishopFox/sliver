package e2e

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
)

const eventHistorySize = 512

type eventHub struct {
	mu      sync.Mutex
	base    int
	history []*clientpb.Event
	wake    chan struct{}
	done    chan struct{}
	err     error
}

func newEventHub(stream rpcpb.SliverRPC_EventsClient) *eventHub {
	hub := &eventHub{
		wake: make(chan struct{}, 1),
		done: make(chan struct{}),
	}
	go hub.receive(stream)
	return hub
}

func (hub *eventHub) receive(stream rpcpb.SliverRPC_EventsClient) {
	defer close(hub.done)
	for {
		event, err := stream.Recv()
		if err != nil {
			hub.mu.Lock()
			hub.err = err
			hub.mu.Unlock()
			return
		}

		hub.mu.Lock()
		hub.history = append(hub.history, event)
		if len(hub.history) > eventHistorySize {
			dropped := len(hub.history) - eventHistorySize
			hub.history = append([]*clientpb.Event(nil), hub.history[dropped:]...)
			hub.base += dropped
		}
		hub.mu.Unlock()
		select {
		case hub.wake <- struct{}{}:
		default:
		}
	}
}

func (hub *eventHub) wait(ctx context.Context, after int, match func(*clientpb.Event) bool) (*clientpb.Event, int, error) {
	for {
		hub.mu.Lock()
		if after < hub.base {
			base := hub.base
			hub.mu.Unlock()
			return nil, base, fmt.Errorf("Sliver event history overflow: cursor %d is older than retained base %d", after, base)
		}
		end := hub.base + len(hub.history)
		if after > end {
			after = end
		}
		for index := after - hub.base; index < len(hub.history); index++ {
			if match(hub.history[index]) {
				event := hub.history[index]
				next := hub.base + index + 1
				hub.mu.Unlock()
				return event, next, nil
			}
		}
		next := hub.base + len(hub.history)
		streamErr := hub.err
		hub.mu.Unlock()

		if streamErr != nil {
			return nil, next, fmt.Errorf("Sliver event stream ended: %w", streamErr)
		}

		select {
		case <-ctx.Done():
			return nil, next, ctx.Err()
		case <-hub.done:
			hub.mu.Lock()
			streamErr = hub.err
			hub.mu.Unlock()
			if streamErr == nil {
				streamErr = errors.New("event stream closed")
			}
			return nil, next, fmt.Errorf("Sliver event stream ended: %w", streamErr)
		case <-hub.wake:
			after = next
		}
	}
}

func (hub *eventHub) cursor() int {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	return hub.base + len(hub.history)
}
