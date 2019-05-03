package console

import (
	"fmt"
	"sliver/client/assets"
	"sliver/client/core"
	"sliver/client/transport"

	"log"

	"github.com/desertbit/grumble"
)

// StartClientConsole - Start the client console
func StartClientConsole() {
	log.Printf("Console starting ...")
	configs := assets.GetConfigs()
	if len(configs) == 0 {
		fmt.Printf(Warn+"No config files found at %s or -config\n", assets.GetConfigDir())
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

	Start(sliverServer, func(*grumble.App, *core.SliverServer) {})
}
