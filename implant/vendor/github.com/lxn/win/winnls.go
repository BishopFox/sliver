// Copyright 2010 The win Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package win

const (
	//  Code Page Default Values.
	//  Please Use Unicode, either UTF-16 (as in WCHAR) or UTF-8 (code page CP_ACP)
	CP_ACP        = 0  // default to ANSI code page
	CP_OEMCP      = 1  // default to OEM  code page
	CP_MACCP      = 2  // default to MAC  code page
	CP_THREAD_ACP = 3  // current thread's ANSI code page
	CP_SYMBOL     = 42 // SYMBOL translations

	CP_UTF7 = 65000 // UTF-7 translation
	CP_UTF8 = 65001 // UTF-8 translation
)
