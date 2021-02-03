// +build darwin

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
import "C"

import (
	"fmt"
	"golang.org/x/sys/unix"
	"unsafe"
)

func gethostuuid() string {
	uuid := C.struct_uuid_t{}
	timespec := C.struct_timespec{tv_sec: 5, tv_nsec: 0}
	unix.Syscall(142,
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
		byte(uuid.time_low>>32), byte(uuid.time_low>>40),
		byte(uuid.time_low>>48), byte(uuid.time_low>>56),
		byte(uuid.time_mid), byte(uuid.time_mid>>8),
		byte(uuid.time_hi_and_version), byte(uuid.time_hi_and_version>>8),
		byte(uuid.clock_seq_hi_and_reserved), byte(uuid.clock_seq_low),
		byte(uuid.node[0]), byte(uuid.node[1]),
	)
}
