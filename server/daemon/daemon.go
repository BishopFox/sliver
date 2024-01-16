package daemon

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
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/server/transport"
)

var (
	serverConfig = configs.GetServerConfig()
	daemonLog    = log.NamedLogger("daemon", "main")

	// BlankHost is a blank hostname
	BlankHost = "-"
	// BlankPort is a blank port number
	BlankPort = uint16(0)
)

// Start - Start as daemon process
func Start(host string, port uint16, tailscale bool) {
	var (
		ln  net.Listener
		err error
	)
	// cli args take president over config
	if host == BlankHost {
		daemonLog.Info("No cli lhost, using config file or default value")
		host = serverConfig.DaemonConfig.Host
	}
	if port == BlankPort {
		daemonLog.Info("No cli lport, using config file or default value")
		port = uint16(serverConfig.DaemonConfig.Port)
	}

	daemonLog.Infof("Starting Sliver daemon %s:%d ...", host, port)
	if tailscale {
		_, ln, err = transport.StartTsNetClientListener(host, port)
	} else {
		_, ln, err = transport.StartMtlsClientListener(host, port)
	}
	if err != nil {
		fmt.Printf("[!] Failed to start daemon %s", err)
		daemonLog.Errorf("Error starting client listener %s", err)
		os.Exit(1)
	}

	done := make(chan bool)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	go func() {
		<-signals
		daemonLog.Infof("Received SIGTERM, exiting ...")
		ln.Close()
		done <- true
	}()
	<-done
}
