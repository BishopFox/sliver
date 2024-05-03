package statute

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

// Request represents the SOCKS5 request, it contains everything that is not payload
// The SOCKS5 request is formed as follows:
//
// +-----+-----+-------+------+----------+----------+
// | VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
// +-----+-----+-------+------+----------+----------+
// |  1  |  1  | X'00' |  1   | Variable |    2     |
// +-----+-----+-------+------+----------+----------+
type Request struct {
	// Version of socks protocol for message
	Version byte
	// Socks Command "connect","bind","associate"
	Command byte
	// Reserved byte
	Reserved byte
	// DstAddr in socks message
	DstAddr AddrSpec
}

// ParseRequest to request from io.Reader
func ParseRequest(r io.Reader) (req Request, err error) {
	// Read the version and command
	tmp := []byte{0, 0}
	if _, err = io.ReadFull(r, tmp); err != nil {
		return req, fmt.Errorf("failed to get request version and command, %v", err)
	}
	req.Version, req.Command = tmp[0], tmp[1]
	if req.Version != VersionSocks5 {
		return req, fmt.Errorf("unrecognized SOCKS version[%d]", req.Version)
	}

	// Read reserved and address type
	if _, err = io.ReadFull(r, tmp); err != nil {
		return req, fmt.Errorf("failed to get request RSV and address type, %v", err)
	}
	req.Reserved, req.DstAddr.AddrType = tmp[0], tmp[1]

	switch req.DstAddr.AddrType {
	case ATYPIPv4:
		addr := make([]byte, net.IPv4len+2)
		if _, err = io.ReadFull(r, addr); err != nil {
			return req, fmt.Errorf("failed to get request, %v", err)
		}
		req.DstAddr.IP = net.IPv4(addr[0], addr[1], addr[2], addr[3])
		req.DstAddr.Port = int(binary.BigEndian.Uint16(addr[net.IPv4len:]))
	case ATYPIPv6:
		addr := make([]byte, net.IPv6len+2)
		if _, err = io.ReadFull(r, addr); err != nil {
			return req, fmt.Errorf("failed to get request, %v", err)
		}
		req.DstAddr.IP = addr[:net.IPv6len]
		req.DstAddr.Port = int(binary.BigEndian.Uint16(addr[net.IPv6len:]))
	case ATYPDomain:
		if _, err = io.ReadFull(r, tmp[:1]); err != nil {
			return req, fmt.Errorf("failed to get request, %v", err)
		}
		domainLen := int(tmp[0])
		addr := make([]byte, domainLen+2)
		if _, err = io.ReadFull(r, addr); err != nil {
			return req, fmt.Errorf("failed to get request, %v", err)
		}
		req.DstAddr.FQDN = string(addr[:domainLen])
		req.DstAddr.Port = int(binary.BigEndian.Uint16(addr[domainLen:]))
	default:
		return req, ErrUnrecognizedAddrType
	}
	return req, nil
}

// Bytes returns a slice of request
func (h Request) Bytes() (b []byte) {
	var addr []byte

	length := 6
	if h.DstAddr.AddrType == ATYPIPv4 {
		length += net.IPv4len
		addr = h.DstAddr.IP.To4()
	} else if h.DstAddr.AddrType == ATYPIPv6 {
		length += net.IPv6len
		addr = h.DstAddr.IP.To16()
	} else { // ATYPDomain
		length += 1 + len(h.DstAddr.FQDN)
		addr = []byte(h.DstAddr.FQDN)
	}

	b = make([]byte, 0, length)
	b = append(b, h.Version, h.Command, h.Reserved, h.DstAddr.AddrType)
	if h.DstAddr.AddrType == ATYPDomain {
		b = append(b, byte(len(h.DstAddr.FQDN)))
	}
	b = append(b, addr...)
	b = append(b, byte(h.DstAddr.Port>>8), byte(h.DstAddr.Port))
	return b
}

// Reply represents the SOCKS5 reply, it contains everything that is not payload
// The SOCKS5 reply is formed as follows:
//
//	+-----+-----+-------+------+----------+-----------+
//	| VER | REP |  RSV  | ATYP | BND.ADDR | BND].PORT |
//	+-----+-----+-------+------+----------+-----------+
//	|  1  |  1  | X'00' |  1   | Variable |    2      |
//	+-----+-----+-------+------+----------+-----------+
type Reply struct {
	// Version of socks protocol for message
	Version byte
	// Socks Response status"
	Response byte
	// Reserved byte
	Reserved byte
	// Bind Address in socks message
	BndAddr AddrSpec
}

// Bytes returns a slice of request
func (sf Reply) Bytes() (b []byte) {
	var addr []byte

	length := 6
	if sf.BndAddr.AddrType == ATYPIPv4 {
		length += net.IPv4len
		addr = sf.BndAddr.IP.To4()
	} else if sf.BndAddr.AddrType == ATYPIPv6 {
		length += net.IPv6len
		addr = sf.BndAddr.IP.To16()
	} else { // ATYPDomain
		length += 1 + len(sf.BndAddr.FQDN)
		addr = []byte(sf.BndAddr.FQDN)
	}

	b = make([]byte, 0, length)
	b = append(b, sf.Version, sf.Response, sf.Reserved, sf.BndAddr.AddrType)
	if sf.BndAddr.AddrType == ATYPDomain {
		b = append(b, byte(len(sf.BndAddr.FQDN)))
	}
	b = append(b, addr...)
	b = append(b, byte(sf.BndAddr.Port>>8), byte(sf.BndAddr.Port))
	return b
}

// ParseReply parse to reply from io.Reader
func ParseReply(r io.Reader) (rep Reply, err error) {
	// Read the version and command
	tmp := []byte{0, 0}
	if _, err = io.ReadFull(r, tmp); err != nil {
		return rep, fmt.Errorf("failed to get reply version and command, %v", err)
	}
	rep.Version, rep.Response = tmp[0], tmp[1]
	if rep.Version != VersionSocks5 {
		return rep, fmt.Errorf("unrecognized SOCKS version[%d]", rep.Version)
	}
	// Read reserved and address type
	if _, err = io.ReadFull(r, tmp); err != nil {
		return rep, fmt.Errorf("failed to get reply RSV and address type, %v", err)
	}
	rep.Reserved, rep.BndAddr.AddrType = tmp[0], tmp[1]

	switch rep.BndAddr.AddrType {
	case ATYPDomain:
		if _, err = io.ReadFull(r, tmp[:1]); err != nil {
			return rep, fmt.Errorf("failed to get reply, %v", err)
		}
		domainLen := int(tmp[0])
		addr := make([]byte, domainLen+2)
		if _, err = io.ReadFull(r, addr); err != nil {
			return rep, fmt.Errorf("failed to get reply, %v", err)
		}
		rep.BndAddr.FQDN = string(addr[:domainLen])
		rep.BndAddr.Port = int(binary.BigEndian.Uint16(addr[domainLen:]))
	case ATYPIPv4:
		addr := make([]byte, net.IPv4len+2)
		if _, err = io.ReadFull(r, addr); err != nil {
			return rep, fmt.Errorf("failed to get reply, %v", err)
		}
		rep.BndAddr.IP = net.IPv4(addr[0], addr[1], addr[2], addr[3])
		rep.BndAddr.Port = int(binary.BigEndian.Uint16(addr[net.IPv4len:]))
	case ATYPIPv6:
		addr := make([]byte, net.IPv6len+2)
		if _, err = io.ReadFull(r, addr); err != nil {
			return rep, fmt.Errorf("failed to get reply, %v", err)
		}
		rep.BndAddr.IP = addr[:net.IPv6len]
		rep.BndAddr.Port = int(binary.BigEndian.Uint16(addr[net.IPv6len:]))
	default:
		return rep, ErrUnrecognizedAddrType
	}
	return rep, nil
}
