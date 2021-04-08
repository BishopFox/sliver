package cli

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"

	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/c2"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/daemon"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Start server in daemon mode",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		appDir := assets.GetRootAppDir()
		logFile := initLogging(appDir)
		defer logFile.Close()

		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic:\n%s", debug.Stack())
				fmt.Println("stacktrace from panic: \n" + string(debug.Stack()))
				os.Exit(99)
			}
		}()

		assets.Setup(false)
		certs.SetupCAs()
		certs.SetupWGKeys()

		serverConfig := configs.GetServerConfig()
		c2.StartPersistentJobs(serverConfig)

		daemon.Start()
	},
}
