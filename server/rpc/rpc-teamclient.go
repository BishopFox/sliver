package rpc

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
	"context"
	"runtime"

	"github.com/bishopfox/sliver/client/version"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/core"
)

// GetVersion - Get the server version
func (rpc *Server) GetVersion(ctx context.Context, _ *commonpb.Empty) (*clientpb.Version, error) {
	dirty := version.GitDirty != ""
	semVer := version.SemanticVersion()
	compiled, _ := version.Compiled()
	return &clientpb.Version{
		Major:      int32(semVer[0]),
		Minor:      int32(semVer[1]),
		Patch:      int32(semVer[2]),
		Commit:     version.GitCommit,
		Dirty:      dirty,
		CompiledAt: compiled.Unix(),
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
	}, nil
}

// GetUsers returns the list of teamserver users and their status.
func (ts *Server) GetUsers(context.Context, *commonpb.Empty) (*clientpb.Users, error) {
	// Fetch users from the teamserver user database.
	users, err := ts.team.Users()

	userspb := make([]*clientpb.User, len(users))
	for i, user := range users {
		userspb[i] = &clientpb.User{
			Name:     user.Name,
			Online:   isOperatorOnline(user.Name),
			LastSeen: user.LastSeen.Unix(),
			Clients:  int32(user.Clients),
		}
	}

	return &clientpb.Users{Users: userspb}, err
}

func isOperatorOnline(commonName string) bool {
	for _, operator := range core.Clients.ActiveOperators() {
		if commonName == operator {
			return true
		}
	}
	return false
}
