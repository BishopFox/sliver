package reconfig

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
	"context"
	"net/url"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

// ReconfigCmd - Reconfigure metadata about a sessions.
func ReconfigCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	var err error
	var reconnectInterval time.Duration
	interval, _ := cmd.Flags().GetString("reconnect-interval")

	if interval != "" {
		reconnectInterval, err = time.ParseDuration(interval)
		if err != nil {
			con.PrintErrorf("Invalid reconnect interval: %s\n", err)
			return
		}
	}

	var beaconInterval time.Duration
	var beaconJitter time.Duration
	binterval, _ := cmd.Flags().GetString("beacon-interval")
	bjitter, _ := cmd.Flags().GetString("beacon-jitter")
	C2URI, _ := cmd.Flags().GetString("c2-uri")

	if beacon != nil {
		if binterval != "" {
			beaconInterval, err = time.ParseDuration(binterval)
			if err != nil {
				con.PrintErrorf("Invalid beacon interval: %s\n", err)
				return
			}
		}
		if bjitter != "" {
			beaconJitter, err = time.ParseDuration(bjitter)
			if err != nil {
				con.PrintErrorf("Invalid beacon jitter: %s\n", err)
				return
			}
			if beaconInterval == 0 && beaconJitter != 0 {
				con.PrintInfof("Modified beacon jitter will take effect after next check-in\n")
			}
		}
	}

	if C2URI != "" {
		// validate uri
		newURI, err := url.Parse(C2URI)
		if err != nil || newURI.Scheme == "" || newURI.Host == "" {
			con.PrintErrorf("Invalid C2 URI: %s (expected format: protocol://host:port)\n", C2URI)
			return
		}

		// validate the protocol is the same as the beacon (cuz it compiled with only one protocol)
		var activeC2 string
		if beacon != nil {
			activeC2 = beacon.ActiveC2
		} else if session != nil {
			activeC2 = session.ActiveC2
		}

		if activeC2 != "" {
			currentURI, err := url.Parse(activeC2)
			if err == nil {
				// must be the same transport
				isHTTP := func(s string) bool { return s == "http" || s == "https" }
				if newURI.Scheme != currentURI.Scheme && !(isHTTP(newURI.Scheme) && isHTTP(currentURI.Scheme)) {
					con.PrintErrorf("Cannot switch protocol from %s to %s (protocol must match compiled implant)\n",
						currentURI.Scheme, newURI.Scheme)
					return
				}
				// for now only support the same Host/C2 Server (becuase the implant crypto keys problem)
				if newURI.Hostname() != currentURI.Hostname() {
					con.PrintWarnf("Changing C2 host: %s -> %s. (THIS SESSION WILL END ON THIS SERVER)\n", currentURI.Hostname(), newURI.Hostname())
					con.PrintInfof("Ensure build identities are migrated via 'implants export' to avoid losing the session.\n")

					confirm := false
					_ = forms.Confirm("Are you sure you want to change the C2-URI?", &confirm)
					if !confirm {
						return
					}
				}
			}
		}

		con.PrintInfof("Updating C2 endpoint to: %s\n", C2URI)
	}

	reconfig, err := con.Rpc.Reconfigure(context.Background(), &sliverpb.ReconfigureReq{
		ReconnectInterval: int64(reconnectInterval),
		BeaconInterval:    int64(beaconInterval),
		BeaconJitter:      int64(beaconJitter),
		C2URI:             C2URI,
		Request:           con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		con.PrintWarnf("%s\n", err)
		return
	}
	if reconfig.Response != nil && reconfig.Response.Async {
		con.AddBeaconCallback(reconfig.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, reconfig)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			con.PrintInfof("Reconfigured beacon\n")
		})
		con.PrintAsyncResponse(reconfig.Response)
	} else {
		con.PrintInfof("Reconfiguration complete\n")
	}
}
