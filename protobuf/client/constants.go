package clientpb

const (
	clientOffset = 10000 // To avoid duplications with sliverpb constants

	// MsgEvent - Initial message from sliver with metadata
	MsgEvent = uint32(clientOffset + iota)

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

	// MsgMsfInject - MSF injection message
	MsgMsfInject

	// MsgTunnelCreate - Create tunnel message
	MsgTunnelCreate
	MsgTunnelClose

	// MsgGenerate - Generate message
	MsgGenerate
	MsgNewProfile
	MsgProfiles
	MsgTask
)
