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
	"github.com/bishopfox/sliver/server/sgn"
)

// ShellcodeEncode - Encode a piece shellcode
func (rpc *Server) ShellcodeEncoder(ctx context.Context, req *clientpb.ShellcodeEncodeReq) (*clientpb.ShellcodeEncode, error) {

	resp := &clientpb.ShellcodeEncode{Response: &commonpb.Response{}}
	var err error

	switch req.Encoder {
	case clientpb.ShellcodeEncoder_SHIKATA_GA_NAI:
		rpcLog.Infof("[rpc] Shellcode encoder request for: SHIKATA_GA_NAI")
		resp.Data, err = sgn.EncodeShellcode(req.Data, req.Architecture, int(req.Iterations), req.BadChars)
		if err != nil {
			resp.Response.Err = err.Error()
		}
	default:
		resp.Response.Err = "Unknown encoder"
	}

	rpcLog.Infof("[rpc] Successfully encoded shellcode (%d bytes)", len(resp.Data))

	return resp, nil
}

// ShellcodeEncoderMap - Get a map of support shellcode encoders <human readable/enum>
func (rpc *Server) ShellcodeEncoderMap(ctx context.Context, _ *commonpb.Empty) (*clientpb.ShellcodeEncoderMap, error) {
	resp := &clientpb.ShellcodeEncoderMap{
		Encoders: map[string]clientpb.ShellcodeEncoder{
			// Human Readable: enum value
			"shikata-ga-nai": clientpb.ShellcodeEncoder_SHIKATA_GA_NAI,
		},
	}
	return resp, nil
}
