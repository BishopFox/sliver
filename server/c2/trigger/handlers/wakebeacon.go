package handlers

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

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
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/0x90pkt/trigger/pkg/intents"
	"github.com/gofrs/uuid"

	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/log"
)

var wakeSessionLog = log.NamedLogger("c2", "trigger-wake-session")

// beaconLookup and beaconSave are package-level indirections so tests
// can swap in fakes without touching the real GORM session.
var (
	beaconLookup = db.BeaconByID
	beaconSave   = func(b *models.Beacon) error {
		return db.Session().Save(b).Error
	}
)

// WakeSession is a task handler (intents.Handler) that fires a wake signal for
// a trigger implant. Trigger implants always establish interactive sessions
// (never beacons) when woken.
//
// For server-side trigger listeners that target legacy beacon implants, this
// handler updates the beacon's NextCheckin to time.Now() so the server treats
// the next check-in as "due immediately." For trigger implants (which use
// session mode), the wake packet is sent directly to the implant's UDP
// listener, and this handler logs the wake event for audit purposes.
//
// The handler retains the beacon UUID binding for backward compatibility with
// existing server-side trigger listener configurations that reference beacon
// UUIDs. New deployments targeting trigger implants may pass a placeholder
// UUID and rely on the UDP wake packet reaching the implant directly.
type WakeSession struct {
	intent   string
	beaconID string // uuid string (may be a placeholder for session-mode wakes)
}

// NewWakeSession constructs a WakeSession handler. The beacon UUID is
// validated at construction so an invalid binding fails fast at
// listener-start rather than at first trigger fire.
func NewWakeSession(intent, beaconID string) (*WakeSession, error) {
	if strings.TrimSpace(intent) == "" {
		return nil, errors.New("wake-session: task name must be set")
	}
	if strings.TrimSpace(beaconID) == "" {
		return nil, errors.New("wake-session: beacon ID must be set")
	}
	if _, err := uuid.FromString(beaconID); err != nil {
		return nil, fmt.Errorf("wake-session: invalid UUID %q: %w", beaconID, err)
	}
	return &WakeSession{intent: intent, beaconID: beaconID}, nil
}

// NewWakeBeacon is a backward-compatible alias for NewWakeSession.
// Deprecated: use NewWakeSession instead.
func NewWakeBeacon(intent, beaconID string) (*WakeSession, error) {
	return NewWakeSession(intent, beaconID)
}

// Name implements intents.Handler.
func (h *WakeSession) Name() string { return h.intent }

// Execute implements intents.Handler. If a beacon exists with the configured
// UUID, updates its NextCheckin to now for backward compatibility. For trigger
// implants (session mode), the real wake happens via the UDP packet -- this
// handler logs the event for server-side audit.
func (h *WakeSession) Execute(_ context.Context, evt intents.Event) error {
	beacon, err := beaconLookup(h.beaconID)
	if err != nil {
		return fmt.Errorf("wake-session %s: lookup %s: %w", h.intent, h.beaconID, err)
	}
	if beacon == nil {
		// No beacon found -- this is expected for session-mode trigger implants.
		// Log the wake event and return success.
		wakeSessionLog.Infof("wake-session fired (session mode): intent=%s target=%s triggered_by=%s source_ip=%s nonce=%s",
			h.intent, h.beaconID, evt.ClientID, evt.SourceIP, evt.Nonce)
		return nil
	}
	// Legacy beacon path: update NextCheckin for backward compat.
	beacon.NextCheckin = time.Now().Unix()
	if err := beaconSave(beacon); err != nil {
		return fmt.Errorf("wake-session %s: save %s: %w", h.intent, h.beaconID, err)
	}
	wakeSessionLog.Infof("wake-session fired (beacon compat): intent=%s beacon=%s triggered_by=%s source_ip=%s nonce=%s",
		h.intent, h.beaconID, evt.ClientID, evt.SourceIP, evt.Nonce)
	return nil
}
