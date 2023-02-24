// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

//go:build darwin || linux

package fakecgo

import (
	"syscall"
	"unsafe"
)

//go:nosplit
func x_cgo_notify_runtime_init_done() {
	// we don't support being called as a library
}

// _cgo_try_pthread_create retries pthread_create if it fails with
// EAGAIN.
//
//go:nosplit
//go:norace
func _cgo_try_pthread_create(thread *pthread_t, attr *pthread_attr_t, pfn unsafe.Pointer, arg *ThreadStart) int {
	var ts syscall.Timespec
	// tries needs to be the same type as syscall.Timespec.Nsec
	// but the fields are int32 on 32bit and int64 on 64bit.
	// tries is assigned to syscall.Timespec.Nsec in order to match its type.
	var tries = ts.Nsec
	var err int

	for tries = 0; tries < 20; tries++ {
		err = int(pthread_create(thread, attr, pfn, unsafe.Pointer(arg)))
		if err == 0 {
			pthread_detach(*thread)
			return 0
		}
		if err != int(syscall.EAGAIN) {
			return err
		}
		ts.Sec = 0
		ts.Nsec = (tries + 1) * 1000 * 1000 // Milliseconds.
		nanosleep(&ts, nil)
	}
	return int(syscall.EAGAIN)
}
