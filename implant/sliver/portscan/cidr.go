package portscan

import (
	"encoding/binary"
	"net"
)

func explodeCidr(cidr string) []net.IP {
	var ret []net.IP
	_, ipv4Net, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil
	}

	mask := binary.BigEndian.Uint32(ipv4Net.Mask)
	start := binary.BigEndian.Uint32(ipv4Net.IP)

	finish := (start & mask) | (mask ^ 0xffffffff)

	for i := start; i <= finish; i++ {
		ip := make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, i)
		ret = append(ret, ip)
	}

	return ret
}
