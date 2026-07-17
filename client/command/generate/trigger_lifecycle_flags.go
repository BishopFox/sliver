package generate

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

	parseTriggerLifecycleFlags is the operator-side parser for the
	Phase 2 implant lifecycle flags:

	  --ttl <go-duration>                  enable TTL deadman switch
	  --ttl-burn-extra-path <path>         repeatable; wipe these on burn
	  --ttl-burn-persistence <artifact>    repeatable; persistence artifact
	                                       to wipe on burn (systemd unit
	                                       path / launchd plist /
	                                       registry key)
	  --trigger-wake-bind <host:port>      enable passive UDP wake listener
	  --trigger-wake-secret-env <ENVVAR>   read HMAC secret from this env
	                                       var on the OPERATOR host
	                                       (avoids secret-in-argv)
	  --trigger-wake-secret <VALUE>        pass HMAC secret directly
	                                       (visible in ps; prefer -env)
	  --trigger-wake-allowed-client <id>   repeatable; client_id allowlist

	When neither --trigger-wake-secret-env nor --trigger-wake-secret is
	provided (and --trigger-wake-bind IS set), the operator is prompted
	for the secret on stdin (masked on TTY, single-line when piped).

	Output is a flat struct copied into the ImplantConfig at the call
	site. The struct is unexported because nothing outside this package
	consumes it directly; callers receive zero-value fields for disabled
	features so the template gates compile down to nothing.
*/

import (
	"errors"
	"fmt"
	"math"
	"net"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/secretinput"
	"github.com/spf13/cobra"
)

// triggerLifecycleFlags carries the parsed --ttl / --trigger-wake-* values.
// Field names match the ImplantConfig proto fields they populate.
type triggerLifecycleFlags struct {
	ttlEnabled                  bool
	ttlMinutes                  uint32
	burnExtraPaths              []string
	burnPersistence             []string
	includeTriggerWake          bool
	triggerWakeBindAddr         string
	triggerWakeSecret           []byte
	triggerWakeAllowedClientIDs []string
}

// parseTriggerLifecycleFlags reads the trigger / TTL flags off cmd and
// returns a validated triggerLifecycleFlags. Returns an error (and the
// caller surfaces it via con.PrintErrorf) for any combination that
// would produce an unusable implant — e.g. --trigger-wake-bind set
// without any secret source.
//
// Secret resolution uses the three-tier hierarchy:
//  1. --trigger-wake-secret-env ENVVAR (preferred; no secret in argv)
//  2. --trigger-wake-secret VALUE (direct; ps-visible warning emitted)
//  3. stdin prompt (masked on TTY, line-read when piped)
//
// All flags are optional; with none set the returned struct is the
// zero value and the corresponding template gates are off.
func parseTriggerLifecycleFlags(cmd *cobra.Command) (triggerLifecycleFlags, error) {
	out := triggerLifecycleFlags{}

	// ---- TTL ----
	ttlStr, _ := cmd.Flags().GetString("ttl")
	burnExtra, _ := cmd.Flags().GetStringSlice("ttl-burn-extra-path")
	burnPersist, _ := cmd.Flags().GetStringSlice("ttl-burn-persistence")

	if strings.TrimSpace(ttlStr) != "" {
		dur, err := time.ParseDuration(strings.TrimSpace(ttlStr))
		if err != nil {
			return out, fmt.Errorf("--ttl: %w (use Go duration syntax, e.g. 720h for 30 days)", err)
		}
		if dur <= 0 {
			return out, errors.New("--ttl must be a positive duration")
		}
		// Minimum cadence inside the watchdog is 1 minute; sub-minute TTLs
		// would over-shoot. Reject them at parse time so the operator sees
		// the error before the build kicks off.
		if dur < time.Minute {
			return out, fmt.Errorf("--ttl must be at least 1 minute (got %s)", dur)
		}
		mins := dur.Round(time.Minute) / time.Minute
		if mins > math.MaxUint32 {
			return out, fmt.Errorf("--ttl too large (max ~8000 years)")
		}
		out.ttlEnabled = true
		out.ttlMinutes = uint32(mins)
	}

	// Burn paths apply to both the TTL-fired and the operator-fired
	// self-destruct paths (triggerwake "self-destruct" task reuses
	// the same burn.Options). So they're allowed independently of --ttl.
	out.burnExtraPaths = filterEmpty(burnExtra)
	out.burnPersistence = filterEmpty(burnPersist)

	// ---- triggerwake ----
	bindAddr, _ := cmd.Flags().GetString("trigger-wake-bind")
	secretEnv, _ := cmd.Flags().GetString("trigger-wake-secret-env")
	directSecret, _ := cmd.Flags().GetString("trigger-wake-secret")
	directSecretChanged := cmd.Flags().Changed("trigger-wake-secret")
	allowedClients, _ := cmd.Flags().GetStringSlice("trigger-wake-allowed-client")

	bindAddr = strings.TrimSpace(bindAddr)
	secretEnv = strings.TrimSpace(secretEnv)

	// Determine if any triggerwake-related flag was explicitly set.
	anyTriggerFlag := bindAddr != "" || secretEnv != "" || directSecretChanged || len(allowedClients) > 0

	switch {
	case !anyTriggerFlag:
		// triggerwake fully off — leave zero values.
	case bindAddr == "":
		return out, errors.New("--trigger-wake-secret-env / --trigger-wake-secret / --trigger-wake-allowed-client require --trigger-wake-bind")
	default:
		if _, _, err := net.SplitHostPort(bindAddr); err != nil {
			return out, fmt.Errorf("--trigger-wake-bind: %w (want host:port)", err)
		}
		// Three-tier secret resolution via secretinput.Resolve.
		secret, err := secretinput.Resolve(
			secretEnv,
			directSecret,
			directSecretChanged,
			"--trigger-wake-secret",
			"trigger-wake HMAC secret",
			func(format string, args ...any) {
				fmt.Fprintf(cmd.ErrOrStderr(), format, args...)
			},
		)
		if err != nil {
			return out, err
		}
		if err := secretinput.ValidateForTemplate(secret); err != nil {
			return out, err
		}

		out.includeTriggerWake = true
		out.triggerWakeBindAddr = bindAddr
		out.triggerWakeSecret = secret
		out.triggerWakeAllowedClientIDs = filterEmpty(allowedClients)
	}

	return out, nil
}

// filterEmpty drops empty / whitespace-only entries. cobra's StringSlice
// can produce `[""]` when the user passes `--flag ""` explicitly; the
// template doesn't filter, so we do it here.
func filterEmpty(xs []string) []string {
	if len(xs) == 0 {
		return nil
	}
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		if s := strings.TrimSpace(x); s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
