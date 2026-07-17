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

	generate trigger -- builds an implant with two operational modes,
	both always baked in:

	  1. Ad-hoc exec: bidirectional UDP command execution. The operator
	     fires a signed "exec" packet; the implant runs the command and
	     returns output over UDP. No C2 channel required.

	  2. Wake session: on receipt of a signed "wake" packet, the implant
	     establishes an interactive SESSION (not a beacon) over its
	     configured C2 transports. For maximum flexibility, specify both
	     --mtls (TCP) and --wg (UDP) when generating the implant.

	Trigger implants never use beacon mode. IsBeacon is always false.

	This is a UX convenience wrapper around `generate --trigger-wake-bind ...`
	that:

	  * Requires --trigger-wake-bind (it is the whole point).
	  * Still accepts all standard generate flags (--mtls, --os, --arch, etc.).
	  * Forces IsBeacon=false -- no beacon mode for trigger implants.
	  * Provides clear help text describing the two-mode architecture.
*/

import (
	"os"

	"github.com/bishopfox/sliver/client/console"
	"github.com/spf13/cobra"
)

// GenerateTriggerCmd generates a trigger implant with two modes: ad-hoc command
// execution (returns output via UDP) and wake (establishes interactive callback
// session). The trigger-wake bind address is required; the implant starts
// dormant and only activates its C2 channel when a valid wake packet arrives.
// Beacon mode is never used -- IsBeacon is always false for trigger implants.
func GenerateTriggerCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	// Validate that --trigger-wake-bind was provided. cobra's MarkFlagRequired
	// would normally handle this, but since the flag is inherited from the
	// shared coreImplantFlags set we enforce it here for clarity.
	bindAddr, _ := cmd.Flags().GetString("trigger-wake-bind")
	if bindAddr == "" {
		con.PrintErrorf("--trigger-wake-bind is required for trigger implants\n")
		con.PrintErrorf("Usage: generate trigger --trigger-wake-bind <host:port> --mtls <server> --wg <server:port> [flags]\n")
		return
	}

	name, config := parseCompileFlags(cmd, con)
	if config == nil {
		return
	}
	spoofMetadata, err := parseSpoofMetadataFlag(cmd, config)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	// Trigger implants always use session mode, never beacon mode.
	config.IsBeacon = false

	// Suggest both TCP and UDP transports if only one (or neither) is configured.
	hasMTLS, _ := cmd.Flags().GetString("mtls")
	hasWG, _ := cmd.Flags().GetString("wg")
	if hasMTLS == "" && hasWG != "" {
		con.PrintWarnf("Consider adding --mtls for TCP callback coverage alongside --wg (UDP)\n")
	} else if hasMTLS != "" && hasWG == "" {
		con.PrintWarnf("Consider adding --wg for UDP callback coverage alongside --mtls (TCP)\n")
	}

	// Sanity: parseCompileFlags already handled trigger lifecycle flags
	// via parseTriggerLifecycleFlags and set IncludeTriggerWake on the
	// config. Double-check the invariant.
	if !config.IncludeTriggerWake {
		con.PrintErrorf("Internal error: trigger-wake was not enabled on the config despite --trigger-wake-bind being set\n")
		return
	}

	save, _ := cmd.Flags().GetString("save")
	if save == "" {
		save, _ = os.Getwd()
	}
	if external, _ := cmd.Flags().GetBool("external-builder"); !external {
		compile(name, config, spoofMetadata, save, con)
	} else {
		_, err := externalBuild(name, config, spoofMetadata, save, con)
		if err != nil {
			switch err {
			case ErrNoExternalBuilder:
				con.PrintErrorf("There are no external builders currently connected to the server\n")
				con.PrintErrorf("See 'builders' command for more information\n")
			case ErrNoValidBuilders:
				con.PrintErrorf("There are external builders connected to the server, but none can build the target you specified\n")
				con.PrintErrorf("See 'builders' command for more information\n")
			default:
				con.PrintErrorf("%s\n", err)
			}
			return
		}
	}
}
