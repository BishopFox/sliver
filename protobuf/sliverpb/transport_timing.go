package sliverpb

import "time"

// These timing values are shared by implant transports and server session
// liveness checks. Keep them beside the wire contract used by both sides.
const (
	MTLSPingInterval      = 2 * time.Minute
	WireGuardPingInterval = 2 * time.Minute
)
