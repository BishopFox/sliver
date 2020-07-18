package cloudflare

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
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/log"
	"github.com/cloudflare/cloudflare-go"
)

var (
	cloudflareLog = log.NamedLogger("infrastructure", "cloudflare")
)

// DNSConfigureForC2 - Confiture a Cloudflare domain for DNS C2
func DNSConfigureForC2(parentDomain string, force bool) error {
	cfConfig := configs.GetCloudflareConfig()
	// Construct a new API object
	api, err := cloudflare.New(cfConfig.Credentials())
	if err != nil {
		cloudflareLog.Error(err)
		return err
	}

	// Fetch user details on the account
	u, err := api.UserDetails()
	if err != nil {
		cloudflareLog.Error(err)
		return err
	}
	// Print user details
	cloudflareLog.Info(u)

	// Fetch the zone ID
	id, err := api.ZoneIDByName(parentDomain)
	if err != nil {
		cloudflareLog.Error(err)
		return err
	}

	// Fetch zone details
	zone, err := api.ZoneDetails(id)
	if err != nil {
		cloudflareLog.Error(err)
		return err
	}

	// Print zone details
	cloudflareLog.Info(zone)
	return nil
}
