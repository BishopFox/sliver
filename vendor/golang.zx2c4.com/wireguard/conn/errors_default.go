//go:build !linux

/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2017-2025 WireGuard LLC. All Rights Reserved.
 */

package conn

func errShouldDisableUDPGSO(_ error) bool {
	return false
}
