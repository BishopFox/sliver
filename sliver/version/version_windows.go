package version

import (
	"fmt"
	//{{if .Debug}}
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
		pHandle, _ := windows.GetCurrentProcess()
		if uint(pHandle) == 0 {
			//{{if .Debug}}
			log.Printf("error getting OS version: error getting current process handle: %v")
			//{{end}}
			arch = "<error getting arch>"
		}
		if err := windows.IsWow64Process(pHandle, &is64Bit); err != nil {
			//{{if .Debug}}
			log.Printf("error getting OS version: error checking if running in WOW: %v")
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
