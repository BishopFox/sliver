// Copyright (c) 2024 Tailscale Inc & AUTHORS. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows && !386

package com

func getTooBigSlice() []byte {
	return make([]byte, maxStreamRWLen+1)
}
