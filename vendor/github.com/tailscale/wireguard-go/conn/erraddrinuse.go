/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2017-2023 WireGuard LLC. All Rights Reserved.
 */

//go:build !plan9

package conn

import "syscall"

func init() {
	errEADDRINUSE = syscall.EADDRINUSE
}
