package console

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
	"errors"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/reeflective/team"
	"google.golang.org/grpc/status"
)

// Users returns a list of all users registered with the app teamserver.
// If the gRPC teamclient is not connected or does not have an RPC client,
// an ErrNoRPC is returned.
func (h *SliverClient) Users() (users []team.User, err error) {
	if h.Rpc == nil {
		return nil, errors.New("No Sliver client RPC")
	}

	res, err := h.Rpc.GetUsers(context.Background(), &commonpb.Empty{})
	if err != nil {
		return nil, h.UnwrapServerErr(err)
	}

	for _, user := range res.GetUsers() {
		users = append(users, team.User{
			Name:     user.Name,
			Online:   user.Online,
			LastSeen: time.Unix(user.LastSeen, 0),
		})
	}

	return
}

// ServerVersion returns the version information of the server to which
// the client is connected, or nil and an error if it could not retrieve it.
func (h *SliverClient) Version() (version team.Version, err error) {
	if h.Rpc == nil {
		return version, errors.New("No Sliver client RPC")
	}

	ver, err := h.Rpc.GetVersion(context.Background(), &commonpb.Empty{})
	if err != nil {
		return version, errors.New(status.Convert(err).Message())
	}

	return team.Version{
		Major:      ver.Major,
		Minor:      ver.Minor,
		Patch:      ver.Patch,
		Commit:     ver.Commit,
		Dirty:      ver.Dirty,
		CompiledAt: ver.CompiledAt,
		OS:         ver.OS,
		Arch:       ver.Arch,
	}, nil
}
