package forwarder

// {{if .Config.WGc2Enabled}}
var (
	tcpForwarders map[int]*WGTCPForwarder
	socksServers  map[int]*WGSocksServer
)

// GetTCPForwarders - Returns a map of WireGuard TCP forwarders
func GetTCPForwarders() map[int]*WGTCPForwarder {
	return tcpForwarders
}

// GetSocksServers - Returns a map of WireGuard SOCKS proxies
func GetSocksServers() map[int]*WGSocksServer {
	return socksServers
}

// GetTCPForwarder - Returns a WireGuard TCP forwarder by id
func GetTCPForwarder(id int) *WGTCPForwarder {
	if f, ok := tcpForwarders[id]; ok {
		return f
	}
	return nil
}

// RemoteTCPForwarder - Remove a TCP forwarder by id
func RemoveTCPForwarder(id int) {
	delete(tcpForwarders, id)
}

// GetSocksServer - Returns a WireGuard SOCKS proxy by id
func GetSocksServer(id int) *WGSocksServer {
	if s, ok := socksServers[id]; ok {
		return s
	}
	return nil
}

// RemoveSocksServer - Remove a SOCKS proxy by id
func RemoveSocksServer(id int) {
	delete(socksServers, id)
}

func init() {
	tcpForwarders = make(map[int]*WGTCPForwarder, 0)
	socksServers = make(map[int]*WGSocksServer, 0)
}

// {{end}}
