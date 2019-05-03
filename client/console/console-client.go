package console

import (
	"fmt"
	"log"
	"sliver/client/assets"
	"sliver/client/core"
	"sliver/client/transport"

	"github.com/desertbit/grumble"
)

// StartClientConsole - Start the client console
func StartClientConsole() error {
	log.Printf("Console starting ...")
	configs := assets.GetConfigs()
	if len(configs) == 0 {
		fmt.Printf(Warn+"No config files found at %s or -config\n", assets.GetConfigDir())
		return nil
	}
	config := selectConfig()
	if config == nil {
		return nil
	}
	send, recv, err := transport.Connect(config)
	if err != nil {
		fmt.Printf(Warn+"Connection to server failed %v", err)
		return nil
	}

	sliverServer := core.BindSliverServer(send, recv)
	go sliverServer.ResponseMapper()

	return Start(sliverServer, func(*grumble.App, *core.SliverServer) {})
}
