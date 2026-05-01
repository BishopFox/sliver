package core

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"time"

	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/log"
)

var (
	eventsLog = log.NamedLogger("core", "events")
)

const (
	// Size is arbitrary, just want to avoid weird cases where we'd block on channel sends
	eventBufSize = 64
)

// Event - An event is fired when there's a state change involving a
//
//	session, job, or client.
type Event struct {
	Session *Session
	Job     *Job
	Client  *Client
	Beacon  *models.Beacon

	EventType string

	Data []byte
	Err  error
}

type eventBroker struct {
	stop        chan struct{}
	publish     chan Event
	subscribe   chan chan Event
	unsubscribe chan chan Event
}

// Start - Start a broker channel
func (broker *eventBroker) Start() {
	subscribers := map[chan Event]struct{}{}
	for {
		select {
		case <-broker.stop:
			for sub := range subscribers {
				close(sub)
			}
			return
		case sub := <-broker.subscribe:
			subscribers[sub] = struct{}{}
		case sub := <-broker.unsubscribe:
			if _, ok := subscribers[sub]; ok {
				delete(subscribers, sub)
				close(sub)
			}
		case event := <-broker.publish:
			for sub := range subscribers {
				select {
				case sub <- event:
				case <-time.After(time.Millisecond * 50):
					eventsLog.Warnf("EventBroker: skipping slow subscriber for event %s", event.EventType)
				}
			}
		}
	}
}

// Stop - Close the broker channel
func (broker *eventBroker) Stop() {
	close(broker.stop)
}

// Subscribe - Generate a new subscription channel
func (broker *eventBroker) Subscribe() chan Event {
	events := make(chan Event, eventBufSize)
	broker.subscribe <- events
	return events
}

// Unsubscribe - Remove a subscription channel
func (broker *eventBroker) Unsubscribe(events chan Event) {
	broker.unsubscribe <- events
}

// Publish - Push a message to all subscribers
func (broker *eventBroker) Publish(event Event) {
	select {
	case broker.publish <- event:
	default:
		eventsLog.Warnf("EventBroker: publish channel full, dropping event %s", event.EventType)
	}
}

func newBroker() *eventBroker {
	broker := &eventBroker{
		stop:        make(chan struct{}),
		publish:     make(chan Event, eventBufSize),
		subscribe:   make(chan chan Event, eventBufSize),
		unsubscribe: make(chan chan Event, eventBufSize),
	}
	go broker.Start()
	return broker
}

var (
	// EventBroker - Distributes event messages
	EventBroker = newBroker()
)
