package generate

import (
	"net"

	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/log"
	"github.com/sirupsen/logrus"
)

var (
	wgipsLog = log.RootLogger.WithFields(logrus.Fields{
		"pkg":    "generate",
		"stream": "wgips",
	})
)

// GenerateUniqueIP generates and returns an available IP which can then
// be assigned to a Wireguard interface
func GenerateUniqueIP() (net.IP, error) {
	dbWireguardIPs, err := db.WGPeerIPs()
	if err != nil {
		wgipsLog.Errorf("Failed to retrieve list of WG Peers IPs with error: %s", err)
		return nil, err
	}

	// Use the 100.64.0.1/16 range for TUN ips.
	// This range chosen due to Tailscale also using it (Cut down to /16 instead of /10)
	// https://tailscale.com/kb/1015/100.x-addresses
	addressPool, err := hosts("100.64.0.1/16")
	if err != nil {
		wgipsLog.Errorf("Failed to generate host address pool for WG Peers IPs %s", err)
		return nil, err
	}

	for _, address := range addressPool {
		for _, ip := range dbWireguardIPs {
			if ip == address {
				addressPool = remove(addressPool, []string{ip})
				break
			}
		}
	}

	return net.ParseIP(addressPool[0]), nil
}

// Reserve use of 100.64.0.{0|1} addresses
var reservedAddresses = []string{"100.64.0.0", "100.64.0.1"}

func hosts(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incrementIP(ip) {
		ips = append(ips, ip.String())
	}

	ips = remove(ips, reservedAddresses)
	return ips, nil
}

func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func remove(stringSlice []string, remove []string) []string {
	var result []string
	for _, v := range stringSlice {
		shouldAppend := true
		for _, value := range remove {
			if v == value {
				shouldAppend = false
			}
		}
		if shouldAppend {
			result = append(result, v)
		}
	}
	return result
}
