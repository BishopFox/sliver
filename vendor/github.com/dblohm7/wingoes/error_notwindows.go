// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

//go:build !windows

package wingoes

// HRESULT is equivalent to the HRESULT type in the Win32 SDK for C/C++.
type HRESULT int32
