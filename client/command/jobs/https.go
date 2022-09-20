package jobs

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
	"io/ioutil"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
)

// HTTPSListenerCmd - Start an HTTPS listener
func HTTPSListenerCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	domain := ctx.Flags.String("domain")
	website := ctx.Flags.String("website")
	lhost := ctx.Flags.String("lhost")
	lport := uint16(ctx.Flags.Int("lport"))

	disableOTP := ctx.Flags.Bool("disable-otp")
	longPollTimeout, err := time.ParseDuration(ctx.Flags.String("long-poll-timeout"))
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	longPollJitter, err := time.ParseDuration(ctx.Flags.String("long-poll-jitter"))
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	cert, key, err := getLocalCertificatePair(ctx)
	if err != nil {
		con.Println()
		con.PrintErrorf("Failed to load local certificate %s\n", err)
		return
	}

	con.PrintInfof("Starting HTTPS %s:%d listener ...\n", domain, lport)
	https, err := con.Rpc.StartHTTPSListener(context.Background(), &clientpb.HTTPListenerReq{
		Domain:          domain,
		Website:         website,
		Host:            lhost,
		Port:            uint32(lport),
		Secure:          true,
		Cert:            cert,
		Key:             key,
		ACME:            ctx.Flags.Bool("lets-encrypt"),
		Persistent:      ctx.Flags.Bool("persistent"),
		EnforceOTP:      !disableOTP,
		LongPollTimeout: int64(longPollTimeout),
		LongPollJitter:  int64(longPollJitter),
		RandomizeJARM:   !ctx.Flags.Bool("disable-randomized-jarm"),
	})
	con.Println()
	if err != nil {
		con.PrintErrorf("%s\n", err)
	} else {
		con.PrintInfof("Successfully started job #%d\n", https.JobID)
	}
}

func getLocalCertificatePair(ctx *grumble.Context) ([]byte, []byte, error) {
	if ctx.Flags.String("cert") == "" && ctx.Flags.String("key") == "" {
		return nil, nil, nil
	}
	cert, err := ioutil.ReadFile(ctx.Flags.String("cert"))
	if err != nil {
		return nil, nil, err
	}
	key, err := ioutil.ReadFile(ctx.Flags.String("key"))
	if err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}
