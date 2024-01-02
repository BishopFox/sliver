// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

//go:build darwin || freebsd || linux

package purego

// Dlerror represents an error value returned from Dlopen, Dlsym, or Dlclose.
type Dlerror struct {
	s string
}

func (e Dlerror) Error() string {
	return e.s
}
