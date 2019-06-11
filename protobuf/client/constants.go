package clientpb

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

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
