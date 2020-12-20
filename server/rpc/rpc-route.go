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

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/comm"
)

// Routes - Get active network routes
func (rpc *Server) Routes(ctx context.Context, req *sliverpb.RoutesReq) (*sliverpb.Routes, error) {

	resp := &sliverpb.Routes{}
	for _, route := range comm.Routes.Registered {
		resp.Active = append(resp.Active, route.ToProtobuf())
	}

	return resp, nil
}

// AddRoute - Add a network route.
func (rpc *Server) AddRoute(ctx context.Context, req *sliverpb.AddRouteReq) (*sliverpb.AddRoute, error) {

	resp := &sliverpb.AddRoute{Response: &commonpb.Response{}}

	// Task server to setup route and request nodes to implement it
	route, err := comm.Routes.Add(req.Route)
	if err != nil {
		resp.Success = false
		resp.Response.Err = err.Error()
		return resp, nil
	}
	if route == nil {
		resp.Success = false
		resp.Response.Err = "Route returned from route setup is nil: an unidentified error occured"
		return resp, nil
	}

	// If we have a route and no errors, return success
	resp.Success = true

	return resp, nil
}

// RemoveRoute - Delete an active network route.
func (rpc *Server) RemoveRoute(ctx context.Context, req *sliverpb.RmRouteReq) (*sliverpb.RmRoute, error) {

	resp := &sliverpb.RmRoute{Response: &commonpb.Response{}}

	err := comm.Routes.Remove(req.Route.ID, req.Close)
	if err != nil {
		resp.Success = false
		resp.Response.Err = err.Error()
		return resp, nil
	}

	// If we have a route and no errors, return success
	resp.Success = true

	return resp, nil
}
