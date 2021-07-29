package registry

import (
	"context"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

func RegListSubKeysCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	path := ctx.Args.String("registry-path")
	hive := ctx.Flags.String("hive")
	hostname := ctx.Flags.String("hostname")

	regList, err := con.Rpc.RegistryListSubKeys(context.Background(), &sliverpb.RegistrySubKeyListReq{
		Hive:     hive,
		Hostname: hostname,
		Path:     path,
		Request:  con.ActiveSession.Request(ctx),
	})

	if err != nil {
		con.PrintErrorf("Error: %s\n", err.Error())
		return
	}

	if regList.Response != nil && regList.Response.Err != "" {
		con.PrintErrorf("Error: %s\n", regList.Response.Err)
		return
	}
	if len(regList.Subkeys) > 0 {
		con.PrintInfof("Sub keys under %s:\\%s:\n", hive, path)
	}
	for _, subKey := range regList.Subkeys {
		con.Println(subKey)
	}
}

func RegListValuesCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	regPath := ctx.Args.String("registry-path")
	hive := ctx.Flags.String("hive")
	hostname := ctx.Flags.String("hostname")

	regList, err := con.Rpc.RegistryListValues(context.Background(), &sliverpb.RegistryListValuesReq{
		Hive:     hive,
		Hostname: hostname,
		Path:     regPath,
		Request:  con.ActiveSession.Request(ctx),
	})

	if err != nil {
		con.PrintErrorf("Error: %s\n", err.Error())
		return
	}

	if regList.Response != nil && regList.Response.Err != "" {
		con.PrintErrorf("Error: %s\n", regList.Response.Err)
		return
	}
	if len(regList.ValueNames) > 0 {
		con.PrintInfof("Values under %s:\\%s:\n", hive, regPath)
	}
	for _, val := range regList.ValueNames {
		con.Println(val)
	}
}
