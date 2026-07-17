package transports

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

	------------------------------------------------------------------------

	Wake hook: a tiny package-level signal that lets out-of-band
	transports (currently triggerwake; future: anything that can wake
	a sleeping beacon) short-circuit the beacon main loop's sleep
	between check-ins.

	Design: a single coalescing channel (buffer of 1). WakeNow() is
	non-blocking — multiple concurrent wakes collapse to a single
	signal. The runner's beacon select reads from WakeChannel()
	alongside its own internal shortCircuit and the standard
	time.After branch.

	The channel carries a transport hint string (e.g. "mtls", "wg",
	"http", "dns"). An empty string means "try all transports in the
	configured order". For beacon wake (short-circuit sleep), the hint
	is typically ignored. For trigger implants, the hint selects which
	C2 transport to use when establishing the session.

	When the implant is built without any wake-capable transport (the
	template directive guards the triggerwake import), the channel
	exists but never gets a sender; the runner's select harmlessly
	ignores its branch.
*/

var wakeNow = make(chan string, 1)

// WakeNow signals the beacon/session loop to wake up. The
// transportHint parameter carries the operator's preferred C2
// transport scheme (e.g. "mtls", "wg", "http", "dns"). Pass ""
// for "try all". Safe to call from any goroutine. Non-blocking --
// if a wake is already pending, the new call is silently coalesced.
func WakeNow(transportHint string) {
	select {
	case wakeNow <- transportHint:
	default:
		// already signaled; coalesce.
	}
}

// WakeChannel returns the read end of the wake signal. The beacon
// main loop's select uses this to be roused early. For trigger
// implants, the received string is the transport hint. Returned as
// <-chan to discourage accidental sends from other call sites -- use
// WakeNow() for that.
func WakeChannel() <-chan string {
	return wakeNow
}

// ResetWake drains any pending wake signal so the channel can
// receive a fresh one. Called by the trigger-implant loop in
// runner.go after a session ends, before blocking on WakeChannel()
// again. Safe to call when no signal is pending.
func ResetWake() {
	select {
	case <-wakeNow:
	default:
	}
}
