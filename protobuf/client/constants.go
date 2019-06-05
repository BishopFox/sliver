package clientpb

const (
	clientOffset = 10000 // To avoid duplications with sliverpb constants

	// MsgEvent - Initial message from sliver with metadata
	MsgEvent = uint32(clientOffset + iota)

	// MsgSessions - Sessions message
	MsgSessions

	// MsgPlayers - List active players
	MsgPlayers

	// MsgJobs - Jobs message
	MsgJobs
	MsgJobKill
	// MsgTcp - TCP message
	MsgTcp
	// MsgMtls - MTLS message
	MsgMtls

	// MsgDns - DNS message
	MsgDns

	// MsgHttp - HTTP(S) listener  message
	MsgHttp
	// MsgHttps - HTTP(S) listener  message
	MsgHttps

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
	MsgMigrate
	MsgGetSystemReq
	MsgEggReq

	MsgRegenerate

	MsgListSliverBuilds
	MsgListCanaries

	// Website related messages
	MsgWebsiteList
	MsgWebsiteAddContent
	MsgWebsiteRemoveContent
)
