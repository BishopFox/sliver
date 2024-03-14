//go:build windows
// +build windows

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox

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

package mount

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
	"unsafe"

	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"golang.org/x/sys/windows"
)

const (
	pathLength             = windows.MAX_PATH + 1
	volumePathsLength      = 32769 // 32K + 1 for a null terminator
	universalNameInfoLevel = 1
	remoteNameInfoLevel    = 2
	remoteNameMaxLength    = 1024
)

type UniversalNameInfo struct {
	UniversalName [remoteNameMaxLength]uint16
}

func findAllVolumes() ([]string, error) {
	var volumes []string

	volumeGuid := make([]uint16, pathLength)
	handle, err := windows.FindFirstVolume(&volumeGuid[0], pathLength)
	defer windows.FindVolumeClose(handle)
	if err != nil {
		return volumes, err
	}

	volumes = append(volumes, windows.UTF16ToString(volumeGuid))

	// Keep gathering volumes until we cannot find anymore
	for {
		err = windows.FindNextVolume(handle, &volumeGuid[0], pathLength)
		if err != nil {
			if err == windows.ERROR_NO_MORE_FILES {
				// Then we are done
				break
			} else {
				return volumes, err
			}
		}
		volumes = append(volumes, windows.UTF16ToString(volumeGuid))
	}

	return volumes, nil
}

func findMountPointsForVolume(volumeName string) ([]string, error) {
	const nullByte uint16 = 0
	var volumeMounts []string

	mountsForVolume := make([]uint16, volumePathsLength)
	var returnedInformationLength uint32 = 0
	volNameUint, err := windows.UTF16PtrFromString(volumeName)
	if err != nil {
		return volumeMounts, err
	}
	err = windows.GetVolumePathNamesForVolumeName(volNameUint, &mountsForVolume[0], volumePathsLength, &returnedInformationLength)
	if err != nil {
		if err == windows.ERROR_MORE_DATA {
			// Then we need to make the buffer bigger and try again
			// We will try one more time
			mountsForVolume := make([]uint16, returnedInformationLength)
			err = windows.GetVolumePathNamesForVolumeName(volNameUint, &mountsForVolume[0], volumePathsLength, &returnedInformationLength)
			if err != nil {
				return volumeMounts, err
			}
		} else {
			return volumeMounts, err
		}
	}

	/*
		Because the buffer contains multiple strings separated by a null terminator, we will read the
		buffer in chunks and separate the strings
	*/
	volumeMounts = splitStringBuffer(mountsForVolume)

	return volumeMounts, nil
}

func splitStringBuffer(buffer []uint16) []string {
	bufferString := string(utf16.Decode(buffer))
	return strings.Split(strings.TrimRight(bufferString, "\x00"), "\x00")
}

func getAllDrives() ([]string, error) {
	var logicalDrives []string

	// Get necessary buffer size
	n, err := windows.GetLogicalDriveStrings(0, nil)
	driveStrings := make([]uint16, n)
	_, err = windows.GetLogicalDriveStrings(n, &driveStrings[0])
	if err != nil {
		return logicalDrives, err
	}

	logicalDrives = splitStringBuffer(driveStrings)

	return logicalDrives, nil
}

func getDriveType(driveSpec string) string {
	driveUTF16, err := windows.UTF16PtrFromString(driveSpec)
	if err != nil {
		return ""
	}

	// Convert type to string (client will handle translation)
	driveTypeValue := strconv.FormatUint(uint64(windows.GetDriveType(driveUTF16)), 10)

	return driveTypeValue
}

func getUniversalName(driveSpec string) (string, error) {
	mpr := windows.NewLazyDLL("mpr.dll")
	wNetGetUniversalNameWProc := mpr.NewProc("WNetGetUniversalNameW")

	driveUTF16, err := windows.UTF16PtrFromString(driveSpec)
	if err != nil {
		return "", err
	}

	var universalNameBuffer UniversalNameInfo
	var bufferSize = uint32(unsafe.Sizeof(universalNameBuffer))

	returnCode, _, err := wNetGetUniversalNameWProc.Call(
		uintptr(unsafe.Pointer(driveUTF16)),
		uintptr(universalNameInfoLevel),
		uintptr(unsafe.Pointer(&universalNameBuffer)),
		uintptr(unsafe.Pointer(&bufferSize)),
	)

	if returnCode != 0 {
		// Then there was an error
		return "", fmt.Errorf("%v", windows.Errno(returnCode).Error())
	}

	bufferStrings := splitStringBuffer(universalNameBuffer.UniversalName[:])
	// There is some junk returned at the beginning of the structure. The actual UNC path starts after the first null terminator.
	if len(bufferStrings) > 1 {
		return bufferStrings[1], nil
	} else {
		return "", errors.New("No valid data returned")
	}
}

func getDiskStats(driveSpec string) (uint64, uint64, uint64, error) {
	var freeSpace uint64 = 0
	var usedSpace uint64 = 0
	var totalSpace uint64 = 0

	driveUTF16, err := windows.UTF16PtrFromString(driveSpec)
	if err != nil {
		return freeSpace, usedSpace, totalSpace, err
	}

	err = windows.GetDiskFreeSpaceEx(driveUTF16, nil, &totalSpace, &freeSpace)
	if err != nil {
		return freeSpace, usedSpace, totalSpace, err
	}

	usedSpace = totalSpace - freeSpace

	return freeSpace, usedSpace, totalSpace, nil
}

func getDriveInfo(driveSpec string) (string, string) {
	// Get label and filesystem type
	driveUTF16, err := windows.UTF16PtrFromString(driveSpec)
	if err != nil {
		return "", ""
	}

	volumeNameBuffer := make([]uint16, pathLength)
	fileSystemNameBuffer := make([]uint16, pathLength)

	err = windows.GetVolumeInformation(driveUTF16, &volumeNameBuffer[0], pathLength, nil, nil, nil, &fileSystemNameBuffer[0], pathLength)
	if err != nil {
		return "", ""
	}

	return windows.UTF16ToString(volumeNameBuffer), windows.UTF16ToString(fileSystemNameBuffer)
}

func constructDriveVolumeMap() (map[string]string, error) {
	driveVolumeMap := make(map[string]string)

	volumes, err := findAllVolumes()
	if err != nil {
		return driveVolumeMap, err
	}

	for _, volume := range volumes {
		mountPoints, err := findMountPointsForVolume(volume)
		if err != nil {
			return driveVolumeMap, err
		}
		for _, mountPoint := range mountPoints {
			driveVolumeMap[mountPoint] = volume
		}
	}
	return driveVolumeMap, nil
}

func GetMountInformation() ([]*sliverpb.MountInfo, error) {
	mountInfo := make([]*sliverpb.MountInfo, 0)

	logicalDrives, err := getAllDrives()
	if err != nil {
		return mountInfo, err
	}

	driveVolumeMap, err := constructDriveVolumeMap()
	if err != nil {
		return mountInfo, err
	}

	for _, drive := range logicalDrives {
		var mountData sliverpb.MountInfo
		mountData.MountPoint = drive
		mountData.VolumeType = getDriveType(drive)

		// Drive type of 4 is "Remote"
		// As per https://cs.opensource.google/go/x/sys/+/refs/tags/v0.18.0:windows/syscall_windows.go;l=33
		if mountData.VolumeType == "4" {
			// Then this is a network drive, so let's figure out the UNC path
			networkPath, err := getUniversalName(drive)
			if err != nil {
				return mountInfo, err
			}
			mountData.VolumeName = networkPath
		} else {
			mountData.VolumeName = driveVolumeMap[drive]
		}

		mountData.Label, mountData.FileSystem = getDriveInfo(drive)
		free, used, total, err := getDiskStats(drive)
		if err != nil {
			return mountInfo, err
		}
		mountData.UsedSpace = used
		mountData.FreeSpace = free
		mountData.TotalSpace = total
		mountInfo = append(mountInfo, &mountData)
	}

	return mountInfo, nil
}
