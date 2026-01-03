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
	"golang.org/x/sys/windows/registry"
	"strings"	
	"net"
)

func GetVirtualizationInfo() string {
	
	if isVirtualizedByMac() {
		return "Vbox/VMware"
	}
	virt:= virtualizationbyreg()
	if virt != "" {
		return virt
	}
	return "none"
}

func virtualizationbyreg() string {
	registryPath := `SYSTEM\CurrentControlSet\Services`
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, registryPath, registry.READ)	
	if err != nil {
		return ""
	}
	subKeys, err := key.ReadSubKeyNames(-1)
	if err != nil {
		return ""
	}
	for _, subKey := range subKeys {
		lowerSubKey := strings.ToLower(subKey)
		if (strings.Contains(lowerSubKey, "vmware") || strings.Contains(lowerSubKey, "vmhgfs") || strings.Contains(lowerSubKey, "vm3dmp") || strings.Contains(lowerSubKey, "vmci") || strings.Contains(lowerSubKey, "vmsrvc") || strings.Contains(lowerSubKey, "vboxguest") || strings.Contains(lowerSubKey, "vboxmouse") || strings.Contains(lowerSubKey, "vboxservice") || strings.Contains(lowerSubKey, "vboxwddm") || strings.Contains(lowerSubKey, "vboxsf") || strings.Contains(lowerSubKey, "vmx86") || strings.Contains(lowerSubKey, "vmxnet") || strings.Contains(lowerSubKey, "vmkbd") || strings.Contains(lowerSubKey, "vmtools") || strings.Contains(lowerSubKey, "vmicheartbeat") || strings.Contains(lowerSubKey, "vmicvss") || strings.Contains(lowerSubKey, "vmicguestinterface") || strings.Contains(lowerSubKey, "vmickvpexchange") || strings.Contains(lowerSubKey, "vmicshutdown")) {
			return "Vbox/VMware"
		}
	}
	defer key.Close()
	return ""
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


