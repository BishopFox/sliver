package console

import (
	"fmt"
	"sliver/client/assets"
	"sliver/client/core"
	"sliver/client/transport"
)

// StartClientConsole - Start the client console
func StartClientConsole() {
	configs := assets.GetConfigs()
	if len(configs) == 0 {
		fmt.Printf(Warn+"No config files found at %s\n", assets.GetConfigDir())
		return
	}
	config := selectConfig()
	if config == nil {
		return
	}
	send, recv, err := transport.Connect(config)
	if err != nil {
		fmt.Printf(Warn+"Connection to server failed %v", err)
		return
	}

	sliverServer := core.BindSliverServer(send, recv)
	go sliverServer.ResponseMapper()

	var nop ExtraCmds
	Start(sliverServer, nop)
}
