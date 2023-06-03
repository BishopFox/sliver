//go:build darwin

package hostuuid

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

/*
struct uuid_t {
	unsigned long time_low;
	unsigned short time_mid;
	unsigned short time_hi_and_version;
	unsigned char clock_seq_hi_and_reserved;
	unsigned char clock_seq_low;
	unsigned char node[6];
};

typedef unsigned long time_t;

struct timespec {
	time_t tv_sec;
	long tv_nsec;
};
*/

import (
	"fmt"
	"syscall"
	"unsafe"
)

type uuid_t struct {
	time_low                  uint32
	time_mid                  uint16
	time_hi_and_version       uint16
	clock_seq_hi_and_reserved uint8
	clock_seq_low             uint8
	node                      [6]uint8
}

type timespec struct {
	tv_sec  uint32
	tv_nsec int32
}

// Darwin syscall:
// int gethostuuid(unsigned char *uuid_buf, const struct timespec *timeoutp);
const gethostuuid = 142

func GetUUID() string {
	uuid := uuid_t{}
	timespec := timespec{tv_sec: 5, tv_nsec: 0}
	syscall.Syscall(gethostuuid,
		uintptr(unsafe.Pointer(&uuid)),
		uintptr(unsafe.Pointer(&timespec)),
		uintptr(0),
	)
	return fmt.Sprintf(
		"%02x%02x%02x%02x-"+
			"%02x%02x-"+
			"%02x%02x-"+
			"%02x%02x-"+
			"%02x%02x%02x%02x%02x%02x",
		byte(uuid.time_low), byte(uuid.time_low>>8), byte(uuid.time_low>>16), byte(uuid.time_low>>24),
		byte(uuid.time_mid), byte(uuid.time_mid>>8),
		byte(uuid.time_hi_and_version), byte(uuid.time_hi_and_version>>8),
		byte(uuid.clock_seq_hi_and_reserved), byte(uuid.clock_seq_low),
		byte(uuid.node[0]), byte(uuid.node[1]),
		byte(uuid.node[2]), byte(uuid.node[3]),
		byte(uuid.node[4]), byte(uuid.node[5]),
	)
}
