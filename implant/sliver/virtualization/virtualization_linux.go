//go:build linux

package virtualization
/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"os"
	"net"
	"strings"
)

func GetVirtualizationInfo() string {
	if isVirtualizedByMac() {
		return "vmware/virtualbox"
	}
	virt := virtualizationbydmi()
	if virt != "" {
		return virt
	}
	return "none"
}

func virtualizationbydmi() string {
	data, err := os.ReadFile("/sys/class/dmi/id/product_name")
	if err != nil {
		return ""
	}
	productName := string(data)

	data, err = os.ReadFile("/sys/class/dmi/id/sys_vendor")
	if err != nil {
		return ""
	}
	sysVendor := string(data)

	switch {
		case containsIgnoreCase(productName, "KVM"):
			return "kvm"
		case containsIgnoreCase(productName, "VirtualBox"):
			return "virtualbox"
		case containsIgnoreCase(productName, "VMware"):
			return "vmware"
		case containsIgnoreCase(sysVendor, "Microsoft Corporation") && containsIgnoreCase(productName, "Virtual Machine"):
			return "hyper-v"
		case containsIgnoreCase(sysVendor, "Xen"):
			return "xen"
		case containsIgnoreCase(sysVendor, "Amazon EC2"):
			return "aws"
		default:
			return ""
	}
}

var virtualizationMacs = []string{
	"00:05:69", // VMware	
	"00:0C:29", // VMware
	"00:1C:14", // VMware
	"00:50:56", // VMware
	"08:00:27", // VirtualBox
	"0A:00:27", // VirtualBox
}

func isVirtualizedByMac() bool {
	interfaces, err := net.Interfaces()
	if err != nil {
		return false
	}
	for _, iface := range interfaces {
		mac := iface.HardwareAddr.String()
		for _, prefix := range virtualizationMacs {
			if strings.HasPrefix(mac, prefix) {
				return true
			}
		}
	}
	return false
}



func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s[:len(substr)] == substr || containsIgnoreCase(s[1:], substr))
}

