package transport

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

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

func grpcKeepaliveOptions() []grpc.ServerOption {
	if serverConfig == nil || serverConfig.GRPC == nil || serverConfig.GRPC.Keepalive == nil {
		return nil
	}

	minTimeSeconds := serverConfig.GRPC.Keepalive.MinTimeSeconds
	if minTimeSeconds <= 0 {
		return nil
	}

	permitWithoutStream := true
	if serverConfig.GRPC.Keepalive.PermitWithoutStream != nil {
		permitWithoutStream = *serverConfig.GRPC.Keepalive.PermitWithoutStream
	}

	policy := keepalive.EnforcementPolicy{
		MinTime:             time.Duration(minTimeSeconds) * time.Second,
		PermitWithoutStream: permitWithoutStream,
	}
	return []grpc.ServerOption{
		grpc.KeepaliveEnforcementPolicy(policy),
	}
}
