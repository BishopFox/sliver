package core

// Event - Sliver connect/disconnect
type Event struct {
	Sliver    *Sliver
	Job       *Job
	Client    *Client
	EventType string
	Err       error
}

type eventBroker struct {
	stop        chan struct{}
	publish     chan Event
	subscribe   chan chan Event
	unsubscribe chan chan Event
	send        chan Event
}

func (b *eventBroker) Start() {
	subscribers := map[chan Event]struct{}{}
	for {
		select {
		case <-b.stop:
			for sub := range subscribers {
				close(sub)
			}
			return
		case sub := <-b.subscribe:
			subscribers[sub] = struct{}{}
		case sub := <-b.unsubscribe:
			delete(subscribers, sub)
		case event := <-b.publish:
			for sub := range subscribers {
				sub <- event
			}
		}
	}
}

func (b *eventBroker) Stop() {
	close(b.stop)
}

// Subscribe - Generate a new subscription channel
func (b *eventBroker) Subscribe() chan Event {
	events := make(chan Event, 5)
	b.subscribe <- events
	return events
}

// Unsubscribe - Remove a subscription channel
func (b *eventBroker) Unsubscribe(events chan Event) {
	b.unsubscribe <- events
	close(events)
}

// Publish - Push a message to all subscribers
func (b *eventBroker) Publish(event Event) {
	b.publish <- event
}

func newBroker() *eventBroker {
	broker := &eventBroker{
		stop:        make(chan struct{}),
		publish:     make(chan Event, 1),
		subscribe:   make(chan chan Event, 1),
		unsubscribe: make(chan chan Event, 1),
		send:        make(chan Event, 1),
	}
	go broker.Start()
	return broker
}

var (
	// EventBroker - Distributes event messages
	EventBroker = newBroker()
)
