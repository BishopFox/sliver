package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
)

// consoleCmd generates the console with required pre/post runners
func consoleCmd(con *console.SliverConsoleClient) *cobra.Command {
	consoleCmd := &cobra.Command{
		Use:   "console",
		Short: "Start the sliver client console",
	}

	consoleCmd.RunE, consoleCmd.PersistentPostRunE = consoleRunnerCmd(con, true)

	return consoleCmd
}

func consoleRunnerCmd(con *console.SliverConsoleClient, run bool) (pre, post func(cmd *cobra.Command, args []string) error) {
	var ln *grpc.ClientConn

	pre = func(_ *cobra.Command, _ []string) error {
		appDir := assets.GetRootAppDir()
		logFile := initLogging(appDir)
		defer logFile.Close()

		configs := assets.GetConfigs()
		if len(configs) == 0 {
			fmt.Printf("No config files found at %s (see --help)\n", assets.GetConfigDir())
			return nil
		}
		config := selectConfig()
		if config == nil {
			return nil
		}

		// Don't clobber output when simply running an implant command from system shell.
		if run {
			fmt.Printf("Connecting to %s:%d ...\n", config.LHost, config.LPort)
		}

		var rpc rpcpb.SliverRPCClient
		var err error

		rpc, ln, err = transport.MTLSConnect(config)
		if err != nil {
			fmt.Printf("Connection to server failed %s", err)
			return nil
		}

		return console.StartClient(con, rpc, command.ServerCommands(con, nil), command.SliverCommands(con), run)
	}

	// Close the RPC connection once exiting
	post = func(_ *cobra.Command, _ []string) error {
		if ln != nil {
			return ln.Close()
		}

		return nil
	}

	return pre, post
}
