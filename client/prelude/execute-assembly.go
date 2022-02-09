package prelude

import (
	"context"
	"errors"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

const (
	execAsmName = "execute-assembly"
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
