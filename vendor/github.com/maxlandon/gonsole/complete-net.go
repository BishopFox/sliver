package gonsole

import (
	"net"
	"strings"

	"github.com/maxlandon/readline"
)

// ClientInterfaceAddrs - All addresses (IPv4/v6) of the console-running host.
func (c *CommandCompleter) ClientInterfaceAddrs() (comps []*readline.CompletionGroup) {

	// Completions
	comp := &readline.CompletionGroup{
		Name:        "client addresses",
		MaxLength:   5,
		DisplayType: readline.TabDisplayGrid,
	}
	var suggestions []string

	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}

		for _, a := range addrs {
			ip, _, err := net.ParseCIDR(a.String())
			if err != nil {
				continue
			}
			suggestions = append(suggestions, ip.String())
		}
	}

	comp.Suggestions = suggestions

	return []*readline.CompletionGroup{comp}
}

// ClientInterfaceNetworks - All networks (IPv4/v6, CIDR notation) to which the console-running host belongs.
func (c *CommandCompleter) ClientInterfaceNetworks() (comps []*readline.CompletionGroup) {

	// Completions
	comp := &readline.CompletionGroup{
		Name:        "client networks",
		MaxLength:   5,
		DisplayType: readline.TabDisplayGrid,
	}
	var suggestions []string

	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			suggestions = append(suggestions, a.String())
		}
	}

	comp.Suggestions = suggestions

	return []*readline.CompletionGroup{comp}
}

func isLoopback(ip string) bool {
	if strings.HasPrefix(ip, "127") || strings.HasPrefix(ip, "::1") {
		return true
	}
	return false
}
