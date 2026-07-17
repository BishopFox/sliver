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

// Package handlers provides Sliver-specific task handlers that
// implement the github.com/0x90pkt/trigger/pkg/intents.Handler
// interface. The trigger listener (imported as a library) dispatches
// authenticated triggers to these handlers; each handler binds a
// task label to a real Sliver server action.
//
// Current handlers:
//
//   - WakeSession: fires a wake signal for a trigger implant. Trigger
//     implants always establish interactive sessions (not beacons).
//     For backward compatibility with legacy beacon-based configs,
//     also updates the beacon's NextCheckin if the target exists.
//     (NewWakeBeacon is retained as a deprecated alias.)
//
//   - StopJob: stops a Sliver job by name (e.g., an mTLS listener,
//     an HTTP listener) using a non-blocking JobCtrl send.
//
//   - ReverseShell: dials a pre-bound operator endpoint and plumbs an
//     interactive shell over the connection. Fire-and-forget with a
//     concurrency semaphore (default 10 concurrent sessions) to prevent
//     resource exhaustion.
//
// Construction is config-time: the operator binds a task label to
// a target (beacon UUID, job name) when the trigger listener starts.
// Runtime task context (client_id, source_ip, nonce, timestamp)
// flows into Execute via intents.Event for audit/logging purposes
// but does NOT change the target -- that's locked at construction so
// crafted client_ids can't redirect handler effects.
