package notifications

import (
	"context"
	"time"

	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/log"
)

const (
	notificationQueueSize = 100
	notificationTimeout   = 10 * time.Second
)

var (
	notificationsLog = log.NamedLogger("notifications", "manager")
)

type Manager struct {
	enabled bool
	entries []notifierEntry

	queue   chan core.Event
	cancel  context.CancelFunc
	started bool
}

func (m *Manager) Start() {
	if m == nil || m.started || !m.enabled || len(m.entries) == 0 {
		return
	}
	m.started = true
	m.queue = make(chan core.Event, notificationQueueSize)

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	events := core.EventBroker.Subscribe()
	go func() {
		defer core.EventBroker.Unsubscribe(events)
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				select {
				case m.queue <- event:
				default:
					notificationsLog.Warnf("Dropping notification event %q (queue full)", event.EventType)
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-m.queue:
				m.dispatch(ctx, event)
			}
		}
	}()
}

func (m *Manager) Stop() {
	if m == nil || !m.started {
		return
	}
	if m.cancel != nil {
		m.cancel()
	}
	m.started = false
}

func (m *Manager) dispatch(ctx context.Context, event core.Event) {
	subject, message := formatEvent(event)
	for _, entry := range m.entries {
		if !entry.allows(event.EventType) {
			continue
		}
		sendCtx, cancel := context.WithTimeout(ctx, notificationTimeout)
		if err := entry.notifier.Send(sendCtx, subject, message); err != nil {
			notificationsLog.Warnf("Notification %s failed: %v", entry.name, err)
		}
		cancel()
	}
}
