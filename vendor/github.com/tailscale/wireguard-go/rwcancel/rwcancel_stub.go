//go:build windows || wasm || plan9

// SPDX-License-Identifier: MIT

package rwcancel

type RWCancel struct{}

func (*RWCancel) Cancel() {}
