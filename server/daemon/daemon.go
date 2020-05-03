package daemon

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/server/transport"
)

var (
	serverConfig = configs.GetServerConfig()
	daemonLog    = log.NamedLogger("daemon", "main")
)

// Start - Start as daemon process
func Start() {
	daemonLog.Infof("Starting Sliver daemon ...")
	host := serverConfig.DaemonConfig.Host
	port := uint16(serverConfig.DaemonConfig.Port)
	_, ln, err := transport.StartClientListener(host, port)
	if err != nil {
		fmt.Printf("[!] Failed to start daemon %s", err)
		daemonLog.Errorf("Error starting client listener %s", err)
		return
	}

	job := &core.Job{
		ID:          core.NextJobID(),
		Name:        "grpc (daemon)",
		Description: "client listener",
		Protocol:    "tcp",
		Port:        port,
		JobCtrl:     make(chan bool),
	}
	core.Jobs.Add(job)

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
