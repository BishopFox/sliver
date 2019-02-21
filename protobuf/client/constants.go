package clientpb

const (

	// MsgEvent - Initial message from sliver with metadata
	MsgEvent = uint32(1 + iota)

	// MsgSessions - Sessions message
	MsgSessions

	// MsgJobs - Jobs message
	MsgJobs

	// MsgMtls - MTLS message
	MsgMtls

	// MsgDns - DNS message
	MsgDns

	// MsgMsf - MSF message
	MsgMsf

	// MsgTunnelCreate - Create tunnel message
	MsgTunnelCreate

	// MsgGenerate - Generate message
	MsgGenerate
	MsgNewProfile
	MsgProfiles
)
