// SPDX-License-Identifier: GPL-3.0-or-later
//
// Package handlers provides built-in intent handlers for the trigger
// listener. The set is intentionally small: noop, exec, webhook.
// Anything more domain-specific (wake-beacon for Sliver, page-on-call
// for PagerDuty, etc.) is implemented by the consumer in their own
// package.
package handlers

import (
	"context"

	"github.com/0x90pkt/trigger/pkg/intents"
)

// Noop is the simplest possible handler: accept the intent, do
// nothing, return nil. The listener still emits an "accepted" audit
// event so operators see the trigger fired.
//
// Use cases:
//   - Liveness / health-check intents where you only want a signed
//     audit trail of "this client_id reached me at this time".
//   - Smoke-testing the dispatch pipeline before wiring real handlers.
//   - Acting as the default for intents that should be audit-only.
type Noop struct {
	name string
}

// NewNoop returns a Noop handler bound to the given intent name.
func NewNoop(name string) *Noop {
	return &Noop{name: name}
}

// Name implements intents.Handler.
func (h *Noop) Name() string { return h.name }

// Execute implements intents.Handler. Always returns nil.
func (h *Noop) Execute(_ context.Context, _ intents.Event) error {
	return nil
}
