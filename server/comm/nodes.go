package comm

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
	"errors"

	"github.com/golang/protobuf/proto"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
)

func sendNodeAdd(n *core.Session, r *sliverpb.Route) (err error) {
	addRouteReq := &sliverpb.AddRouteReq{
		Request: &commonpb.Request{SessionID: n.ID},
		Route:   r,
	}
	data, _ := proto.Marshal(addRouteReq)

	// Process response
	addRoute := &sliverpb.AddRoute{}
	resp, err := n.Request(sliverpb.MsgNumber(addRouteReq), defaultNetTimeout, data)
	if err != nil {
		return err
	}
	proto.Unmarshal(resp, addRoute)

	// If there is an error with a node, we return it and the caller will be in
	// charge of asking previously ordered nodes to delete this orphaned route.
	if addRoute.Success == false {
		return errors.New(addRoute.Response.Err)
	}
	return
}

func sendNodeDel(n *core.Session, r *sliverpb.Route) (err error) {
	closeRouteReq := &sliverpb.RmRouteReq{
		Request: &commonpb.Request{SessionID: n.ID},
		Route:   r,
	}
	data, _ := proto.Marshal(closeRouteReq)

	// Process response
	closeRoute := &sliverpb.RmRoute{}
	resp, err := n.Request(sliverpb.MsgNumber(closeRouteReq), defaultNetTimeout, data)
	if err != nil {
		return err
	}
	proto.Unmarshal(resp, closeRoute)

	// If there is an error with a node, we return it and the caller will be in
	// charge of asking previously ordered nodes to delete this orphaned route.
	if closeRoute.Success == false {
		return errors.New(closeRoute.Response.Err)
	}
	return
}
