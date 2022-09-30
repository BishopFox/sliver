package rpc

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

func (rpc *Server) Webhooks(ctx context.Context, _ *commonpb.Empty) (*clientpb.Webhooks, error) {

	return &clientpb.Webhooks{}, nil
}

func (rpc *Server) StartSlackWebhook(ctx context.Context, req *clientpb.SlackWebhook) (*commonpb.Empty, error) {

	//config := configs.SlackWebhookConfig{}

	return &commonpb.Empty{}, nil
}

func (rpc *Server) StopSlackWebhook(ctx context.Context, req *commonpb.Empty) (*commonpb.Empty, error) {

	return &commonpb.Empty{}, nil
}
