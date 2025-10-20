// Copyright 2024 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux && (amd64 || loong64)

package sqlite // import "modernc.org/sqlite"

import (
	"unsafe"

	"modernc.org/libc"
	sqlite3 "modernc.org/sqlite/lib"
)

// C documentation
//
//	int sqlite3_bind_blob(sqlite3_stmt*, int, const void*, int n, void(*)(void*));
func (c *conn) bindBlob(pstmt uintptr, idx1 int, value []byte) (uintptr, error) {
	if value == nil {
		if rc := sqlite3.Xsqlite3_bind_null(c.tls, pstmt, int32(idx1)); rc != sqlite3.SQLITE_OK {
			return 0, c.errstr(rc)
		}
		return 0, nil
	}

	p, err := c.malloc(len(value))
	if err != nil {
		return 0, err
	}
	if len(value) != 0 {
		copy((*libc.RawMem)(unsafe.Pointer(p))[:len(value):len(value)], value)
	}
	if rc := sqlite3.Xsqlite3_bind_blob(c.tls, pstmt, int32(idx1), p, int32(len(value)), 0); rc != sqlite3.SQLITE_OK {
		c.free(p)
		return 0, c.errstr(rc)
	}

	return p, nil
}
