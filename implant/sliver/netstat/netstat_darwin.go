package netstat

import (
	"encoding/binary"
	"net"
	"unsafe"

	"golang.org/x/sys/unix"
)

var skStates = [...]string{
	"CLOSED",
	"LISTEN",
	"SYN_SENT",
	"SYN_RCVD",
	"ESTABLISHED",
	"CLOSE_WAIT",
	"FIN_WAIT_1",
	"CLOSING",
	"LAST_ACK",
	"FIN_WAIT_2",
	"TIME_WAIT",
}

// Socket states
const (
	Closed SkState = 0 << iota
	Listen
	SynSent
	SynRecvd
	Established
	CloseWait
	FinWait1
	Closing
	LastAck
	FinWait2
	TimeWait
)

const (
	TCP = 1 << iota
	UDP
)

func osTCPSocks(accept AcceptFn) ([]SockTabEntry, error) {
	return parseTCP(INP_IPV4)
}

func osTCP6Socks(accept AcceptFn) ([]SockTabEntry, error) {
	return parseTCP(INP_IPV6)
}

func osUDPSocks(accept AcceptFn) ([]SockTabEntry, error) {
	return parseUDP(INP_IPV4)
}

func osUDP6Socks(accept AcceptFn) ([]SockTabEntry, error) {
	return parseUDP(INP_IPV6)
}

const (
	INP_IPV4 = 1 << iota
	INP_IPV6
)

func ntohs(v uint16) uint16 {
	bs := make([]byte, 2)
	binary.LittleEndian.PutUint16(bs, v)
	return binary.BigEndian.Uint16(bs)
}

func getIP(data [16]byte, family int) net.IP {
	var ip net.IP
	switch family {
	case INP_IPV4:
		in4addr := (*InAddr4in6)(unsafe.Pointer(&data[0]))
		ip = net.IP(in4addr.Addr4[:]).To4()
	case INP_IPV6:
		in6addr := (*In6Addr)(unsafe.Pointer(&data[0]))
		ip = net.IP(in6addr.X__u6_addr[:])
	}
	return ip
}

func parseTCP(version uint8) ([]SockTabEntry, error) {
	entries := make([]SockTabEntry, 0)
	buf, err := unix.SysctlRaw("net.inet.tcp.pcblist64")
	if err != nil {
		return nil, err
	}
	xinpgen := (*Xinpgen)(unsafe.Pointer(&buf[0]))
	current := xinpgen
	for i := 1; i < int(xinpgen.Count); i++ {
		xig := (*XTCPcb64)(unsafe.Pointer(current))
		inp := (*Xinpcb64)(unsafe.Pointer(&xig.Pad_cgo_0))
		if inp.Inp_vflag&version != 0 {
			entry := parseXinpcb64(inp, TCP, version, xig)
			entries = append(entries, entry)
		}
		current = (*Xinpgen)(unsafe.Pointer(uintptr(unsafe.Pointer(current)) + uintptr(current.Len)))
	}
	return entries, nil
}

func parseUDP(version uint8) ([]SockTabEntry, error) {
	entries := make([]SockTabEntry, 0)
	buf, err := unix.SysctlRaw("net.inet.udp.pcblist64")
	if err != nil {
		return nil, err
	}
	xinpgen := (*Xinpgen)(unsafe.Pointer(&buf[0]))
	current := xinpgen
	for i := 1; i < int(xinpgen.Count); i++ {
		inp := (*Xinpcb64)(unsafe.Pointer(current))
		if inp.Inp_vflag&version != 0 {
			entry := parseXinpcb64(inp, UDP, version, nil)
			entries = append(entries, entry)
		}
		current = (*Xinpgen)(unsafe.Pointer(uintptr(unsafe.Pointer(current)) + uintptr(current.Len)))
	}
	return entries, nil
}

func parseXinpcb64(inp *Xinpcb64, transport int, ipVersion uint8, xig *XTCPcb64) SockTabEntry {
	var result SockTabEntry

	lport := ntohs(inp.Inp_lport)
	fport := ntohs(inp.Inp_fport)

	switch ipVersion {
	case INP_IPV4:
		result.LocalAddr = &SockAddr{
			IP:   getIP(inp.Inp_dependladdr, INP_IPV4),
			Port: lport,
		}
		result.RemoteAddr = &SockAddr{
			IP:   getIP(inp.Inp_dependfaddr, INP_IPV4),
			Port: fport,
		}
	case INP_IPV6:
		result.LocalAddr = &SockAddr{
			IP:   getIP(inp.Inp_dependladdr, INP_IPV6),
			Port: lport,
		}
		result.RemoteAddr = &SockAddr{
			IP:   getIP(inp.Inp_dependfaddr, INP_IPV6),
			Port: fport,
		}
	}
	if xig != nil {
		result.State = SkState(xig.T_state)
	}
	return result
}
