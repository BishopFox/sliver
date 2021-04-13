package forwarder

// {{if .Config.WGc2Enabled}}
var (
	tcpForwarders map[int]*WGTCPForwarder
	socksServers  map[int]*WGSocksServer
)

func GetTCPForwarders() map[int]*WGTCPForwarder {
	return tcpForwarders
}

func GetSocksServers() map[int]*WGSocksServer {
	return socksServers
}

func GetTCPForwarder(id int) *WGTCPForwarder {
	if f, ok := tcpForwarders[id]; ok {
		return f
	}
	return nil
}

func RemoveTCPForwarder(id int) {
	delete(tcpForwarders, id)
}

func GetSocksServer(id int) *WGSocksServer {
	if s, ok := socksServers[id]; ok {
		return s
	}
	return nil
}

func RemoveSocksServer(id int) {
	delete(socksServers, id)
}

func init() {
	tcpForwarders = make(map[int]*WGTCPForwarder, 0)
	socksServers = make(map[int]*WGSocksServer, 0)
}

// {{end}}
