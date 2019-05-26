package console

import (
	"fmt"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/transport"

	"github.com/desertbit/grumble"
)

// StartClientConsole - Start the client console
func StartClientConsole() error {

	configs := assets.GetConfigs()
	if len(configs) == 0 {
		fmt.Printf(Warn+"No config files found at %s or -config\n", assets.GetConfigDir())
		return nil
	}
	config := selectConfig()
	if config == nil {
		return nil
	}
	fmt.Printf(Info+"Connecting to %s:%d ...\n", config.LHost, config.LPort)
	send, recv, err := transport.MTLSConnect(config)
	if err != nil {
		fmt.Printf(Warn+"Connection to server failed %v", err)
		return nil
	}

	sliverServer := core.BindSliverServer(send, recv)
	go sliverServer.ResponseMapper()

	return Start(sliverServer, func(*grumble.App, *core.SliverServer) {})
}
