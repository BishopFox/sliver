//go:build 386 || amd64

package version

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"fmt"
	//{{if .Config.Debug}}
	"log"
	//{{end}}
	"runtime"

	"golang.org/x/sys/windows"
)

const VER_NT_WORKSTATION = 0x0000001

func getOSVersion() string {
	osVersion := windows.RtlGetVersion()

	var osName string
	if osVersion.MajorVersion == 6 {
		switch osVersion.MinorVersion {
		case 0:
			if osVersion.ProductType == VER_NT_WORKSTATION {
				osName = "Vista"
			} else {
				osName = "Server 2008"
			}
		case 1:
			if osVersion.ProductType == VER_NT_WORKSTATION {
				osName = "7"
			} else {
				osName = "Server 2008 R2"
			}
		case 2:
			if osVersion.ProductType == VER_NT_WORKSTATION {
				osName = "8"
			} else {
				osName = "Server 2012"
			}
		case 3:
			if osVersion.ProductType == VER_NT_WORKSTATION {
				osName = "8.1"
			} else {
				osName = "Server 2012 R2"
			}
		}
	} else {
		if osVersion.ProductType == VER_NT_WORKSTATION {
			osName = "10"
		} else {
			osName = "Server 2016"
		}
	}

	var servicePack string
	if osVersion.ServicePackMajor != 0 {
		servicePack = fmt.Sprintf(" Service Pack %d", osVersion.ServicePackMajor)
	}

	var arch string
	if runtime.GOARCH == "amd64" {
		arch = "x86_64"
	} else {
		var is64Bit bool
		pHandle := windows.CurrentProcess()
		if uint(pHandle) == 0 {
			//{{if .Config.Debug}}
			log.Printf("error getting OS version: error getting current process handle")
			//{{end}}
			arch = "<error getting arch>"
		}
		if err := windows.IsWow64Process(pHandle, &is64Bit); err != nil {
			//{{if .Config.Debug}}
			log.Printf("error getting OS version: error checking if running in WOW: %v", err)
			//{{end}}
			arch = "<error getting arch>"
		} else {
			if !is64Bit {
				arch = "x86"
			} else {
				arch = "x86_64"
			}
		}
	}

	return fmt.Sprintf("%s%s build %d %s", osName, servicePack, osVersion.BuildNumber, arch)
}

// GetVersion returns the os version information
func GetVersion() string {
	return getOSVersion()
}
