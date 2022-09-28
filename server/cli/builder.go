package cli

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"os"
	"runtime/debug"
	"strings"

	clientAssets "github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/version"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/builder"
	"github.com/bishopfox/sliver/server/log"
	"github.com/spf13/cobra"
)

var (
	builderLog = log.NamedLogger("cli", "builder")
)

var builderCmd = &cobra.Command{
	Use:   "builder",
	Short: "Start the process as an external builder",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		quiet, err := cmd.Flags().GetBool(quietFlagStr)
		if err != nil {
			builderLog.Fatalf("Failed to parse --%s flag %s\n", quietFlagStr, err)
		}
		if !quiet {
			log.RootLogger.AddHook(log.NewStdoutHook(log.RootLoggerName))
		}
		builderLog.Infof("Starting Sliver external builder - %s", version.FullVersion())

		defer func() {
			if r := recover(); r != nil {
				builderLog.Printf("panic:\n%s", debug.Stack())
				builderLog.Fatalf("stacktrace from panic: \n" + string(debug.Stack()))
				os.Exit(99)
			}
		}()

		configPath, err := cmd.Flags().GetString(operatorConfigFlagStr)
		if err != nil {
			builderLog.Fatalf("Failed to parse --%s flag %s\n", operatorConfigFlagStr, err)
			return
		}

		builderConfig := parseBuilderConfigFlags(cmd)

		// load the client configuration from the filesystem
		config, err := clientAssets.ReadConfig(configPath)
		if err != nil {
			builderLog.Fatalf("Invalid config file: %s", err)
		}
		// connect to the server
		rpc, ln, err := transport.MTLSConnect(config)
		if err != nil {
			builderLog.Fatalf("Failed to connect to server: %s", err)
		}
		defer ln.Close()
		builder.StartBuilder(rpc, builderConfig)
	},
}

func parseBuilderConfigFlags(cmd *cobra.Command) builder.Config {
	conf := builder.Config{}
	var err error

	conf.GOOSs, err = cmd.Flags().GetStringSlice(goosFlagStr)
	if err != nil {
		builderLog.Fatalf("Failed to parse --%s flag %s\n", goosFlagStr, err)
	}
	builderLog.Debugf("GOOS enabled: %v", conf.GOOSs)
	conf.GOARCHs, err = cmd.Flags().GetStringSlice(goarchFlagStr)
	if err != nil {
		builderLog.Fatalf("Failed to parse --%s flag %s\n", goarchFlagStr, err)
	}
	builderLog.Debugf("GOARCH enabled: %v", conf.GOARCHs)
	rawFormats, err := cmd.Flags().GetStringSlice(formatFlagStr)
	if err != nil {
		builderLog.Fatalf("Failed to parse --%s flag %s\n", formatFlagStr, err)
	}

	for _, rawFormat := range rawFormats {
		switch strings.ToLower(rawFormat) {
		case "exe", "executable", "pe":
			builderLog.Debugf("Executable format enabled (%d)", clientpb.OutputFormat_EXECUTABLE)
			conf.Formats = append(conf.Formats, clientpb.OutputFormat_EXECUTABLE)
		case "dll", "so", "shared", "dylib", "lib", "library":
			builderLog.Debugf("Library format enabled (%d)", clientpb.OutputFormat_SHARED_LIB)
			conf.Formats = append(conf.Formats, clientpb.OutputFormat_SHARED_LIB)
		case "service":
			builderLog.Debugf("Service format enabled (%d)", clientpb.OutputFormat_SERVICE)
			conf.Formats = append(conf.Formats, clientpb.OutputFormat_SERVICE)
		case "bin", "shellcode":
			builderLog.Debugf("Shellcode format enabled (%d)", clientpb.OutputFormat_SHELLCODE)
			conf.Formats = append(conf.Formats, clientpb.OutputFormat_SHELLCODE)
		}
	}

	return conf
}
