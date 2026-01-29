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

	templates map[string]templateSpec
	renderer  *templateRenderer

	queue   chan core.Event
	cancel  context.CancelFunc
	started bool
}

func (m *Manager) Start() {
	if m == nil || m.started || !m.enabled || len(m.entries) == 0 {
		if m != nil && !m.enabled {
			notificationsLog.Infof("Notifications disabled")
		}
		return
	}
	m.started = true
	m.queue = make(chan core.Event, notificationQueueSize)

	notificationsLog.Infof("Starting notifications with %d service(s)", len(m.entries))
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
	notificationsLog.Infof("Notifications stopped")
}

func (m *Manager) dispatch(ctx context.Context, event core.Event) {
	defer func() {
		if recovered := recover(); recovered != nil {
			notificationsLog.Errorf("Notification dispatch panic for event %q: %v", event.EventType, recovered)
		}
	}()
	subject, message := formatEvent(event)
	if m.renderer != nil && len(m.templates) > 0 {
		if spec, ok := m.templates[event.EventType]; ok {
			rendered, err := m.renderer.render(spec, buildTemplateData(event, subject, message))
			if err != nil {
				notificationsLog.Warnf("Failed to render template %q for event %q: %v", spec.name, event.EventType, err)
			} else {
				message = rendered
				notificationsLog.Debugf("Rendered %s template %q for event %q", spec.typ, spec.name, event.EventType)
			}
		}
	}
	for _, entry := range m.entries {
		if !entry.allows(event.EventType) {
			notificationsLog.Debugf("Skipping notification %s for event %q (filtered)", entry.name, event.EventType)
			continue
		}
		sendCtx, cancel := context.WithTimeout(ctx, notificationTimeout)
		if err := entry.notifier.Send(sendCtx, subject, message); err != nil {
			notificationsLog.Warnf("Notification %s failed: %v", entry.name, err)
		} else {
			notificationsLog.Debugf("Notification %s delivered for event %q", entry.name, event.EventType)
		}
		cancel()
	}
}
