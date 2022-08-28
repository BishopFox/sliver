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
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/prelude"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/spf13/cobra"
)

var cmdPrelude = &cobra.Command{
	Use:   "prelude",
	Short: "Start client as a Prelude bridge",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		confFilePath, err := cmd.Flags().GetString("config")
		if err != nil {
			log.Printf("[!] %s", err)
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}
		config, err := assets.ReadConfig(confFilePath)
		if err != nil {
			log.Printf("[!] Failed to read config file: %s", err)
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}
		preludeServer, err := cmd.Flags().GetString(preludeServerFlagStr)
		if err != nil {
			log.Printf("[!] %s", err)
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}
		aesKey, err := cmd.Flags().GetString(aesKeyFlagStr)
		if err != nil {
			log.Printf("[!] %s", err)
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}

		fmt.Printf("Prelude relay mode, connecting to %s ... ", preludeServer)
		log.Printf("Prelude relay mode, connecting to %s ...", preludeServer)
		rpc, _, err := transport.MTLSConnect(config)
		if err != nil {
			log.Printf("[!] Failed to connect to server %s", err)
			fmt.Printf("%s\n", err)
			os.Exit(2)
		}
		fmt.Println("success!")
		log.Printf("Connection successful")
		preludeConfig := &prelude.OperatorConfig{
			OperatorURL: preludeServer,
			RPC:         rpc,
			AESKey:      aesKey,
		}
		implantMapper := prelude.InitImplantMapper(preludeConfig)
		sessions, err := rpc.GetSessions(context.Background(), &commonpb.Empty{})
		if err != nil {
			log.Printf("[!] Could not get session list: %s", err)
			os.Exit(3)
		}

		if len(sessions.Sessions) > 0 {
			log.Printf("Adding existing sessions ...")
			for _, session := range sessions.Sessions {
				if !session.IsDead {
					err = implantMapper.AddImplant(session, nil)
					if err != nil {
						log.Printf("[!] Could not add session %s to implant mapper: %s", session.Name, err)
					}
				}
			}
			log.Printf("Done !")
		}
		beacons, err := rpc.GetBeacons(context.Background(), &commonpb.Empty{})
		if err != nil {
			log.Printf("[!] Could not get beacon list: %s", err)
			os.Exit(3)
		}
		if len(beacons.Beacons) > 0 {
			log.Printf("Adding existing beacons ...")

			// I'm not sure this callback map is actually needed
			type BeaconTaskCallback func(*clientpb.BeaconTask)
			beaconTaskCallbacks := map[string]BeaconTaskCallback{}
			beaconTaskCallbacksMutex := &sync.Mutex{}

			for _, beacon := range beacons.Beacons {
				err = implantMapper.AddImplant(beacon, func(taskID string, cb func(task *clientpb.BeaconTask)) {
					beaconTaskCallbacksMutex.Lock()
					defer beaconTaskCallbacksMutex.Unlock()
					beaconTaskCallbacks[taskID] = cb
				})
				if err != nil {
					log.Printf("[!] Could not add beacon %s to implant mapper: %s", beacon.Name, err)
				}
			}
		}
	},
}
