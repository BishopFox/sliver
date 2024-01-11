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
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/c2"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/console"
	"github.com/bishopfox/sliver/server/cryptography"
	"github.com/bishopfox/sliver/server/daemon"
	"github.com/bishopfox/sliver/server/db"
)

const (

	// Unpack flags
	forceFlagStr = "force"

	// Operator flags
	nameFlagStr        = "name"
	lhostFlagStr       = "lhost"
	lportFlagStr       = "lport"
	saveFlagStr        = "save"
	outputFlagStr      = "output"
	permissionsFlagStr = "permissions"
	tailscaleFlagStr   = "tailscale"

	// Cert flags
	caTypeFlagStr = "type"
	loadFlagStr   = "load"

	// console log file name
	logFileName = "console.log"
)

// Initialize logging
func initConsoleLogging(appDir string) *os.File {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	logFile, err := os.OpenFile(filepath.Join(appDir, "logs", logFileName), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	log.SetOutput(logFile)
	return logFile
}

func init() {
	// Unpack
	unpackCmd.Flags().BoolP(forceFlagStr, "f", false, "Force unpack and overwrite")
	rootCmd.AddCommand(unpackCmd)

	// Operator
	operatorCmd.Flags().StringP(nameFlagStr, "n", "", "operator name")
	operatorCmd.Flags().StringP(lhostFlagStr, "l", "", "multiplayer listener host")
	operatorCmd.Flags().Uint16P(lportFlagStr, "p", uint16(31337), "multiplayer listener port")
	operatorCmd.Flags().StringP(saveFlagStr, "s", "", "save file to ...")
	operatorCmd.Flags().StringP(outputFlagStr, "o", "file", "output format (file, stdout)")
	operatorCmd.Flags().StringSliceP(permissionsFlagStr, "P", []string{}, "grant permissions to the operator profile (all, builder, crackstation)")
	rootCmd.AddCommand(operatorCmd)

	// Certs
	cmdExportCA.Flags().StringP(saveFlagStr, "s", "", "save CA to file ...")
	cmdExportCA.Flags().StringP(caTypeFlagStr, "t", "", fmt.Sprintf("ca type (%s)", strings.Join(validCATypes(), ", ")))
	rootCmd.AddCommand(cmdExportCA)

	cmdImportCA.Flags().StringP(loadFlagStr, "l", "", "load CA from file ...")
	cmdImportCA.Flags().StringP(caTypeFlagStr, "t", "", fmt.Sprintf("ca type (%s)", strings.Join(validCATypes(), ", ")))
	rootCmd.AddCommand(cmdImportCA)

	// Daemon
	daemonCmd.Flags().StringP(lhostFlagStr, "l", daemon.BlankHost, "multiplayer listener host")
	daemonCmd.Flags().Uint16P(lportFlagStr, "p", daemon.BlankPort, "multiplayer listener port")
	daemonCmd.Flags().BoolP(forceFlagStr, "f", false, "force unpack and overwrite static assets")
	daemonCmd.Flags().BoolP(tailscaleFlagStr, "t", false, "enable tailscale")
	rootCmd.AddCommand(daemonCmd)

	// Builder
	rootCmd.AddCommand(initBuilderCmd())

	// Version
	rootCmd.AddCommand(versionCmd)
}

var rootCmd = &cobra.Command{
	Use:   "sliver-server",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		// Root command starts the server normally

		appDir := assets.GetRootAppDir()
		logFile := initConsoleLogging(appDir)
		defer logFile.Close()

		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic:\n%s", debug.Stack())
				fmt.Println("stacktrace from panic: \n" + string(debug.Stack()))
				os.Exit(99)
			}
		}()

		assets.Setup(false, true)
		certs.SetupCAs()
		certs.SetupWGKeys()
		cryptography.AgeServerKeyPair()
		cryptography.MinisignServerPrivateKey()
		c2.SetupDefaultC2Profiles()

		serverConfig := configs.GetServerConfig()
		listenerJobs, err := db.ListenerJobs()
		if err != nil {
			fmt.Println(err)
		}

		err = StartPersistentJobs(listenerJobs)
		if err != nil {
			fmt.Println(err)
		}
		if serverConfig.DaemonMode {
			daemon.Start(daemon.BlankHost, daemon.BlankPort, serverConfig.DaemonConfig.Tailscale)
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
