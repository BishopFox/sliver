package cli

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
	"log"
	"os"
	"path"
	"strings"

	"github.com/bishopfox/sliver/client/version"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/c2"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/console"
	"github.com/bishopfox/sliver/server/daemon"

	"github.com/spf13/cobra"
)

var (
	sliverServerVersion = fmt.Sprintf("v%s", version.FullVersion())
)

const (

	// Unpack flags
	forceFlagStr = "force"

	// Operator flags
	nameFlagStr  = "name"
	lhostFlagStr = "lhost"
	lportFlagStr = "lport"
	saveFlagStr  = "save"

	// Cert flags
	caTypeFlagStr = "type"
	loadFlagStr   = "load"

	logFileName = "console.log"
)

// Initialize logging
func initLogging(appDir string) *os.File {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	logFile, err := os.OpenFile(path.Join(appDir, "logs", logFileName), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	log.SetOutput(logFile)
	return logFile
}

func init() {

	// Unpack
	cmdUnpack.Flags().BoolP(forceFlagStr, "f", false, "Force unpack and overwrite")
	rootCmd.AddCommand(cmdUnpack)

	// Operator
	cmdOperator.Flags().StringP(nameFlagStr, "n", "", "operator name")
	cmdOperator.Flags().StringP(lhostFlagStr, "l", "", "listener host")
	cmdOperator.Flags().Uint16P(lportFlagStr, "p", uint16(1337), "listener port")
	cmdOperator.Flags().StringP(saveFlagStr, "s", "", "save file to ...")
	rootCmd.AddCommand(cmdOperator)

	// Certs
	cmdExportCA.Flags().StringP(saveFlagStr, "s", "", "save CA to file ...")
	cmdExportCA.Flags().StringP(caTypeFlagStr, "t", "",
		fmt.Sprintf("ca type (%s)", strings.Join(validCATypes(), ", ")))
	rootCmd.AddCommand(cmdExportCA)

	cmdImportCA.Flags().StringP(loadFlagStr, "l", "", "load CA from file ...")
	cmdImportCA.Flags().StringP(caTypeFlagStr, "t", "",
		fmt.Sprintf("ca type (%s)", strings.Join(validCATypes(), ", ")))
	rootCmd.AddCommand(cmdImportCA)

	// Version
	rootCmd.AddCommand(cmdVersion)
}

var rootCmd = &cobra.Command{
	Use:   "sliver-server",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		// Root command starts the server normally

		appDir := assets.GetRootAppDir()
		logFile := initLogging(appDir)
		defer logFile.Close()

		assets.Setup(false)
		certs.SetupCAs()

		serverConfig := configs.GetServerConfig()
		c2.StartPersistentJobs(serverConfig)
		if serverConfig.DaemonMode {
			daemon.Start()
		} else {
			os.Args = os.Args[:1] // Hide cli from grumble console
			console.Start()
		}

	},
}

// Execute - Execute root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
