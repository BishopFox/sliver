//go:build darwin

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
	"strings"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"golang.org/x/sys/unix"
)

func getString(input []byte) string {
	ver := string(input)
	if i := strings.Index(ver, "\x00"); i != -1 {
		ver = ver[:i]
	}
	return ver
}

func formatMountFlags(flags uint32) string {
	opts := make([]string, 0, 12)

	if flags&unix.MNT_RDONLY != 0 {
		opts = append(opts, "ro")
	} else {
		opts = append(opts, "rw")
	}
	if flags&unix.MNT_NOSUID != 0 {
		opts = append(opts, "nosuid")
	}
	if flags&unix.MNT_NODEV != 0 {
		opts = append(opts, "nodev")
	}
	if flags&unix.MNT_NOEXEC != 0 {
		opts = append(opts, "noexec")
	}
	if flags&unix.MNT_SYNCHRONOUS != 0 {
		opts = append(opts, "sync")
	}
	if flags&unix.MNT_ASYNC != 0 {
		opts = append(opts, "async")
	}
	if flags&unix.MNT_NOATIME != 0 {
		opts = append(opts, "noatime")
	}
	if flags&unix.MNT_JOURNALED != 0 {
		opts = append(opts, "journaled")
	}
	if flags&unix.MNT_QUOTA != 0 {
		opts = append(opts, "quota")
	}
	if flags&unix.MNT_UNION != 0 {
		opts = append(opts, "union")
	}
	if flags&unix.MNT_AUTOMOUNTED != 0 {
		opts = append(opts, "automounted")
	}
	if flags&unix.MNT_REMOVABLE != 0 {
		opts = append(opts, "removable")
	}
	if flags&unix.MNT_DONTBROWSE != 0 {
		opts = append(opts, "nobrowse")
	}
	if flags&unix.MNT_IGNORE_OWNERSHIP != 0 {
		opts = append(opts, "ignore-ownership")
	}
	if flags&unix.MNT_LOCAL != 0 {
		opts = append(opts, "local")
	}
	if flags&unix.MNT_ROOTFS != 0 {
		opts = append(opts, "rootfs")
	}
	if flags&unix.MNT_SNAPSHOT != 0 {
		opts = append(opts, "snapshot")
	}
	if flags&unix.MNT_CPROTECT != 0 {
		opts = append(opts, "cprotect")
	}
	if flags&unix.MNT_NOUSERXATTR != 0 {
		opts = append(opts, "nouserxattr")
	}
	if flags&unix.MNT_QUARANTINE != 0 {
		opts = append(opts, "quarantine")
	}
	if flags&unix.MNT_UNKNOWNPERMISSIONS != 0 {
		opts = append(opts, "unknown-permissions")
	}
	if flags&unix.MNT_DEFWRITE != 0 {
		opts = append(opts, "defwrite")
	}
	if flags&unix.MNT_DOVOLFS != 0 {
		opts = append(opts, "volfs")
	}
	if flags&unix.MNT_NOBLOCK != 0 {
		opts = append(opts, "noblock")
	}
	if flags&unix.MNT_RELOAD != 0 {
		opts = append(opts, "reload")
	}
	if flags&unix.MNT_UPDATE != 0 {
		opts = append(opts, "update")
	}
	if flags&unix.MNT_EXPORTED != 0 {
		opts = append(opts, "exported")
	}
	if flags&unix.MNT_MULTILABEL != 0 {
		opts = append(opts, "multilabel")
	}
	if flags&unix.MNT_STRICTATIME != 0 {
		opts = append(opts, "strictatime")
	}
	if flags&unix.MNT_FORCE != 0 {
		opts = append(opts, "force")
	}

	return strings.Join(opts, ",")
}

func GetMountInformation() ([]*sliverpb.MountInfo, error) {
	mountInfo := make([]*sliverpb.MountInfo, 0)

	count, err := unix.Getfsstat(nil, unix.MNT_NOWAIT)
	if err != nil || count == 0 {
		return mountInfo, err
	}

	stats := make([]unix.Statfs_t, count)
	count, err = unix.Getfsstat(stats, unix.MNT_NOWAIT)
	if err != nil {
		return mountInfo, err
	}

	stats = stats[:count]
	for _, stat := range stats {
		var mountData sliverpb.MountInfo
		mountData.VolumeName = getString(stat.Mntfromname[:])
		mountData.MountPoint = getString(stat.Mntonname[:])
		mountData.Label = "/"
		mountData.VolumeType = getString(stat.Fstypename[:])
		mountData.MountOptions = formatMountFlags(stat.Flags)
		mountData.TotalSpace = stat.Blocks * uint64(stat.Bsize)
		mountData.FreeSpace = stat.Bavail * uint64(stat.Bsize)
		mountData.UsedSpace = (stat.Blocks - stat.Bfree) * uint64(stat.Bsize)
		mountInfo = append(mountInfo, &mountData)
	}

	return mountInfo, nil
}
