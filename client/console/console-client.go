package console

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
	"fmt"

	"github.com/maxlandon/gonsole"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/transport"
)

// StartClientConsole - Start the client console
func StartClientConsole() error {
	configs := assets.GetConfigs()
	if len(configs) == 0 {
		fmt.Printf(Warn+"No config files found at %s or -import\n", assets.GetConfigDir())
		return nil
	}
	config := selectConfig()
	if config == nil {
		return nil
	}

	fmt.Printf(Info+"Connecting to %s:%d ...\n", config.LHost, config.LPort)
	rpc, ln, err := transport.MTLSConnect(config)
	if err != nil {
		fmt.Printf(Warn+"Connection to server failed %v", err)
		return nil
	}
	defer ln.Close()

	// Pass the server configuration, that is accessed by the prompt and the console.
	return Start(rpc, func(menu *gonsole.Menu) {}, config)
}
