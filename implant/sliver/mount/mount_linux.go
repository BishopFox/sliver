//go:build linux

package mount

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

import (
	"bufio"
	"os"
	"strings"
	"syscall"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

func GetMountInformation() ([]*sliverpb.MountInfo, error) {
	mountInfo := make([]*sliverpb.MountInfo, 0)

	file, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return mountInfo, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		// Extract fields according to the /proc/self/mountinfo format
		// https://man7.org/linux/man-pages/man5/proc.5.html
		mountRoot := fields[3]
		mountPoint := fields[4]
		mountOptions := fields[5]
		mountType := fields[len(fields)-3]
		mountSource := fields[len(fields)-2]

		// Get mount information using statfs
		var stat syscall.Statfs_t
		err := syscall.Statfs(mountPoint, &stat)
		if err != nil {
			continue
		}

		var mountData sliverpb.MountInfo

		mountData.Label = mountRoot
		mountData.MountPoint = mountPoint
		mountData.VolumeType = mountType
		mountData.VolumeName = mountSource
		mountData.MountOptions = mountOptions
		mountData.TotalSpace = stat.Blocks * uint64(stat.Bsize)
		mountInfo = append(mountInfo, &mountData)

	}

	return mountInfo, nil
}
