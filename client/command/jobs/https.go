package jobs

import (
	"context"
	"io/ioutil"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
)

func HTTPSListenerCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	domain := ctx.Flags.String("domain")
	website := ctx.Flags.String("website")
	lport := uint16(ctx.Flags.Int("lport"))

	cert, key, err := getLocalCertificatePair(ctx)
	if err != nil {
		con.Println()
		con.PrintErrorf("Failed to load local certificate %s\n", err)
		return
	}

	con.PrintInfof("Starting HTTPS %s:%d listener ...\n", domain, lport)
	https, err := con.Rpc.StartHTTPSListener(context.Background(), &clientpb.HTTPListenerReq{
		Domain:     domain,
		Website:    website,
		Port:       uint32(lport),
		Secure:     true,
		Cert:       cert,
		Key:        key,
		ACME:       ctx.Flags.Bool("lets-encrypt"),
		Persistent: ctx.Flags.Bool("persistent"),
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
