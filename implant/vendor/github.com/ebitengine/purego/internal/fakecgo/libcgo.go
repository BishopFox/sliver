// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

//go:build darwin || linux

package fakecgo

type size_t uintptr
type sigset_t [128]byte
type pthread_attr_t [64]byte
type pthread_t int

// for pthread_sigmask:

type sighow int32

const (
	SIG_BLOCK   sighow = 0
	SIG_UNBLOCK sighow = 1
	SIG_SETMASK sighow = 2
)

type G struct {
	stacklo uintptr
	stackhi uintptr
}

type ThreadStart struct {
	g   *G
	tls *uintptr
	fn  uintptr
}
