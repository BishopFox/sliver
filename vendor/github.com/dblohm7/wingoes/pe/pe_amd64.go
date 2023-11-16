// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

//go:build windows

package pe

import (
	dpe "debug/pe"
)

type optionalHeader dpe.OptionalHeader64
type ptrOffset int64

const (
	expectedMachine     = dpe.IMAGE_FILE_MACHINE_AMD64
	optionalHeaderMagic = 0x020B
)
