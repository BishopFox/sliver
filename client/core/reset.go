package core

import "net"

// ResetClientState tears down singleton client-side state that is tied to a
// specific server connection (tunnels, portfwds, SOCKS, cursed procs, etc).
//
// This is primarily used to support switching servers inside a single
// sliver-client process.
func ResetClientState() {
	// Port forwards
	for _, meta := range Portfwds.List() {
		Portfwds.Remove(meta.ID)
	}

	// SOCKS
	for _, meta := range SocksProxies.List() {
		SocksProxies.Remove(meta.ID)
	}
	SocksConnPool.Range(func(key, value any) bool {
		if conn, ok := value.(net.Conn); ok {
			_ = conn.Close()
		}
		SocksConnPool.Delete(key)
		return true
	})

	// Tunnels
	GetTunnels().Reset()

	// Cursed processes
	CursedProcesses.Range(func(key, value any) bool {
		if cp, ok := value.(*CursedProcess); ok && cp != nil && cp.PortFwd != nil {
			Portfwds.Remove(cp.PortFwd.ID)
		}
		CursedProcesses.Delete(key)
		return true
	})
}
