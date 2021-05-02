package completion

import (
	"context"

	"github.com/maxlandon/readline"

	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// ServerInterfaceAddrs - All addresses (IPv4/v6) of the C2 Sliver Server.
func ServerInterfaceAddrs() (comps []*readline.CompletionGroup) {

	// Completions
	comp := &readline.CompletionGroup{
		Name:        "server addresses",
		MaxLength:   5,
		DisplayType: readline.TabDisplayGrid,
	}
	var suggestions []string

	resp, err := transport.RPC.GetServerInterfaces(context.Background(), &sliverpb.IfconfigReq{})
	if err != nil {
		return
	}

	for _, iface := range resp.NetInterfaces {
		for _, ip := range iface.IPAddresses {
			suggestions = append(suggestions, ip)
		}
	}

	comp.Suggestions = suggestions

	return []*readline.CompletionGroup{comp}
}
