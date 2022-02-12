package prelude

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
	"errors"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

type execasmArgs struct {
	IsDLL        bool   `json:"isDLL"`
	Process      string `json:"process"`
	Arguments    string `json:"arguments"`
	Architecture string `json:"architecture"`
	Method       string `json:"method"`
	Class        string `json:"class"`
	AppDomain    string `json:"appDomain"`
}

func execAsm(session *clientpb.Session, rpc rpcpb.SliverRPCClient, asm []byte, args execasmArgs) (output string, err error) {
	if !isLoaderLoaded(session, rpc) {
		err = registerLoader(session, rpc)
		if err != nil {
			return
		}
	}

	extResp, err := rpc.ExecuteAssembly(context.Background(), &sliverpb.ExecuteAssemblyReq{
		Request:   MakeRequest(session),
		IsDLL:     args.IsDLL,
		Process:   args.Process,
		Arguments: args.Arguments,
		Assembly:  asm,
		Arch:      args.Architecture,
		Method:    args.Method,
		ClassName: args.Class,
		AppDomain: args.AppDomain,
	})

	if err != nil {
		return
	}

	if extResp.Response != nil && extResp.Response.Err != "" {
		err = errors.New(extResp.Response.Err)
	}
	output = string(extResp.Output)

	return
}
