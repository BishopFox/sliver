// Copyright 2017 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sqlite is a sql/database driver using a CGo-free port of the C
// SQLite3 library.
//
// SQLite is an in-process implementation of a self-contained, serverless,
// zero-configuration, transactional SQL database engine.
//
// # Thanks
//
// This project is sponsored by Schleibinger Geräte Teubert u. Greim GmbH by
// allowing one of the maintainers to work on it also in office hours.
//
// # Supported platforms and architectures
//
// These combinations of GOOS and GOARCH are currently supported
//
//	OS      Arch    SQLite version
//	------------------------------
//	darwin	amd64   3.41.2
//	darwin	arm64   3.41.2
//	freebsd	amd64   3.41.2
//	freebsd	arm64   3.41.2
//	linux	386     3.41.2
//	linux	amd64   3.41.2
//	linux	arm     3.41.2
//	linux	arm64   3.41.2
//	linux	ppc64le 3.41.2
//	linux	riscv64 3.41.2
//	linux	s390x   3.41.2
//	windows	amd64   3.41.2
//	windows	arm64   3.41.2
//
// # Builders
//
// Builder results available at:
//
// https://modern-c.appspot.com/-/builder/?importpath=modernc.org%2fsqlite
//
// # Speedtest1
//
// Numbers for the pure Go version were produced by
//
//	~/src/modernc.org/sqlite/speedtest1$ go build && ./speedtest1
//
// Numbers for the pure C version were produced by
//
//	~/src/modernc.org/sqlite/testdata/sqlite-src-3410200/test$ gcc speedtest1.c ../../sqlite-amalgamation-3410200/sqlite3.c -lpthread -ldl && ./a.out
//
// The results are from Go version 1.20.4 and GCC version 10.2.1 on a
// Linux/amd64 machine, CPU: AMD Ryzen 9 3900X 12-Core Processor × 24, 128GB
// RAM. Shown are the best of 3 runs.
//
//	Go											C
//
//	-- Speedtest1 for SQLite 3.41.2 2023-03-22 11:56:21 0d1fc92f94cb6b76bffe3ec34d69	-- Speedtest1 for SQLite 3.41.2 2023-03-22 11:56:21 0d1fc92f94cb6b76bffe3ec34d69
//	 100 - 50000 INSERTs into table with no index......................    0.071s            100 - 50000 INSERTs into table with no index......................    0.077s
//	 110 - 50000 ordered INSERTS with one index/PK.....................    0.114s            110 - 50000 ordered INSERTS with one index/PK.....................    0.082s
//	 120 - 50000 unordered INSERTS with one index/PK...................    0.137s            120 - 50000 unordered INSERTS with one index/PK...................    0.099s
//	 130 - 25 SELECTS, numeric BETWEEN, unindexed......................    0.083s            130 - 25 SELECTS, numeric BETWEEN, unindexed......................    0.091s
//	 140 - 10 SELECTS, LIKE, unindexed.................................    0.210s            140 - 10 SELECTS, LIKE, unindexed.................................    0.120s
//	 142 - 10 SELECTS w/ORDER BY, unindexed............................    0.276s            142 - 10 SELECTS w/ORDER BY, unindexed............................    0.182s
//	 145 - 10 SELECTS w/ORDER BY and LIMIT, unindexed..................    0.183s            145 - 10 SELECTS w/ORDER BY and LIMIT, unindexed..................    0.099s
//	 150 - CREATE INDEX five times.....................................    0.172s            150 - CREATE INDEX five times.....................................    0.127s
//	 160 - 10000 SELECTS, numeric BETWEEN, indexed.....................    0.080s            160 - 10000 SELECTS, numeric BETWEEN, indexed.....................    0.078s
//	 161 - 10000 SELECTS, numeric BETWEEN, PK..........................    0.080s            161 - 10000 SELECTS, numeric BETWEEN, PK..........................    0.078s
//	 170 - 10000 SELECTS, text BETWEEN, indexed........................    0.187s            170 - 10000 SELECTS, text BETWEEN, indexed........................    0.169s
//	 180 - 50000 INSERTS with three indexes............................    0.196s            180 - 50000 INSERTS with three indexes............................    0.154s
//	 190 - DELETE and REFILL one table.................................    0.200s            190 - DELETE and REFILL one table.................................    0.155s
//	 200 - VACUUM......................................................    0.180s            200 - VACUUM......................................................    0.142s
//	 210 - ALTER TABLE ADD COLUMN, and query...........................    0.004s            210 - ALTER TABLE ADD COLUMN, and query...........................    0.005s
//	 230 - 10000 UPDATES, numeric BETWEEN, indexed.....................    0.093s            230 - 10000 UPDATES, numeric BETWEEN, indexed.....................    0.080s
//	 240 - 50000 UPDATES of individual rows............................    0.153s            240 - 50000 UPDATES of individual rows............................    0.137s
//	 250 - One big UPDATE of the whole 50000-row table.................    0.024s            250 - One big UPDATE of the whole 50000-row table.................    0.019s
//	 260 - Query added column after filling............................    0.004s            260 - Query added column after filling............................    0.005s
//	 270 - 10000 DELETEs, numeric BETWEEN, indexed.....................    0.278s            270 - 10000 DELETEs, numeric BETWEEN, indexed.....................    0.263s
//	 280 - 50000 DELETEs of individual rows............................    0.188s            280 - 50000 DELETEs of individual rows............................    0.180s
//	 290 - Refill two 50000-row tables using REPLACE...................    0.411s            290 - Refill two 50000-row tables using REPLACE...................    0.359s
//	 300 - Refill a 50000-row table using (b&1)==(a&1).................    0.175s            300 - Refill a 50000-row table using (b&1)==(a&1).................    0.151s
//	 310 - 10000 four-ways joins.......................................    0.427s            310 - 10000 four-ways joins.......................................    0.365s
//	 320 - subquery in result set......................................    0.440s            320 - subquery in result set......................................    0.521s
//	 400 - 70000 REPLACE ops on an IPK.................................    0.125s            400 - 70000 REPLACE ops on an IPK.................................    0.106s
//	 410 - 70000 SELECTS on an IPK.....................................    0.081s            410 - 70000 SELECTS on an IPK.....................................    0.078s
//	 500 - 70000 REPLACE on TEXT PK....................................    0.174s            500 - 70000 REPLACE on TEXT PK....................................    0.116s
//	 510 - 70000 SELECTS on a TEXT PK..................................    0.153s            510 - 70000 SELECTS on a TEXT PK..................................    0.117s
//	 520 - 70000 SELECT DISTINCT.......................................    0.083s            520 - 70000 SELECT DISTINCT.......................................    0.067s
//	 980 - PRAGMA integrity_check......................................    0.436s            980 - PRAGMA integrity_check......................................    0.377s
//	 990 - ANALYZE.....................................................    0.107s            990 - ANALYZE.....................................................    0.038s
//	       TOTAL.......................................................    5.525s                  TOTAL.......................................................    4.637s
//
// This particular test executes 16.1% faster in the C version.
//
// # Changelog
//
// 2023-08-03 v1.25.0: enable SQLITE_ENABLE_DBSTAT_VTAB.
//
// 2023-07-11 v1.24.0:
//
// Add (*conn).{Serialize,Deserialize,NewBackup,NewRestore} methods, add Backup type.
//
// 2023-06-01 v1.23.0:
//
// Allow registering aggregate functions.
//
// 2023-04-22 v1.22.0:
//
// Support linux/s390x.
//
// 2023-02-23 v1.21.0:
//
// Upgrade to SQLite 3.41.0, release notes at https://sqlite.org/releaselog/3_41_0.html.
//
// 2022-11-28 v1.20.0
//
// Support linux/ppc64le.
//
// 2022-09-16 v1.19.0:
//
// Support frebsd/arm64.
//
// 2022-07-26 v1.18.0:
//
// Adds support for Go fs.FS based SQLite virtual filesystems, see function New
// in modernc.org/sqlite/vfs and/or TestVFS in all_test.go
//
// 2022-04-24 v1.17.0:
//
// Support windows/arm64.
//
// 2022-04-04 v1.16.0:
//
// Support scalar application defined functions written in Go.
//
//	https://www.sqlite.org/appfunc.html
//
// 2022-03-13 v1.15.0:
//
// Support linux/riscv64.
//
// 2021-11-13 v1.14.0:
//
// Support windows/amd64. This target had previously only experimental status
// because of a now resolved memory leak.
//
// 2021-09-07 v1.13.0:
//
// Support freebsd/amd64.
//
// 2021-06-23 v1.11.0:
//
// Upgrade to use sqlite 3.36.0, release notes at https://www.sqlite.org/releaselog/3_36_0.html.
//
// 2021-05-06 v1.10.6:
//
// Fixes a memory corruption issue
// (https://gitlab.com/cznic/sqlite/-/issues/53).  Versions since v1.8.6 were
// affected and should be updated to v1.10.6.
//
// 2021-03-14 v1.10.0:
//
// Update to use sqlite 3.35.0, release notes at https://www.sqlite.org/releaselog/3_35_0.html.
//
// 2021-03-11 v1.9.0:
//
// Support darwin/arm64.
//
// 2021-01-08 v1.8.0:
//
// Support darwin/amd64.
//
// 2020-09-13 v1.7.0:
//
// Support linux/arm and linux/arm64.
//
// 2020-09-08 v1.6.0:
//
// Support linux/386.
//
// 2020-09-03 v1.5.0:
//
// This project is now completely CGo-free, including the Tcl tests.
//
// 2020-08-26 v1.4.0:
//
// First stable release for linux/amd64.  The database/sql driver and its tests
// are CGo free.  Tests of the translated sqlite3.c library still require CGo.
//
//	$ make full
//
//	...
//
//	SQLite 2020-08-14 13:23:32 fca8dc8b578f215a969cd899336378966156154710873e68b3d9ac5881b0ff3f
//	0 errors out of 928271 tests on 3900x Linux 64-bit little-endian
//	WARNING: Multi-threaded tests skipped: Linked against a non-threadsafe Tcl build
//	All memory allocations freed - no leaks
//	Maximum memory usage: 9156360 bytes
//	Current memory usage: 0 bytes
//	Number of malloc()  : -1 calls
//	--- PASS: TestTclTest (1785.04s)
//	PASS
//	ok  	modernc.org/sqlite	1785.041s
//	$
//
// 2020-07-26 v1.4.0-beta1:
//
// The project has reached beta status while supporting linux/amd64 only at the
// moment. The 'extraquick' Tcl testsuite reports
//
//	630 errors out of 200177 tests on  Linux 64-bit little-endian
//
// and some memory leaks
//
//	Unfreed memory: 698816 bytes in 322 allocations
//
// 2019-12-28 v1.2.0-alpha.3: Third alpha fixes issue #19.
//
// It also bumps the minor version as the repository was wrongly already tagged
// with v1.1.0 before.  Even though the tag was deleted there are proxies that
// cached that tag. Thanks /u/garaktailor for detecting the problem and
// suggesting this solution.
//
// 2019-12-26 v1.1.0-alpha.2: Second alpha release adds support for accessing a
// database concurrently by multiple goroutines and/or processes. v1.1.0 is now
// considered feature-complete. Next planed release should be a beta with a
// proper test suite.
//
// 2019-12-18 v1.1.0-alpha.1: First alpha release using the new cc/v3, gocc,
// qbe toolchain. Some primitive tests pass on linux_{amd64,386}. Not yet safe
// for concurrent access by multiple goroutines. Next alpha release is planed
// to arrive before the end of this year.
//
// 2017-06-10 Windows/Intel no more uses the VM (thanks Steffen Butzer).
//
// 2017-06-05 Linux/Intel no more uses the VM (cznic/virtual).
//
// # Connecting to a database
//
// To access a Sqlite database do something like
//
//	import (
//		"database/sql"
//
//		_ "modernc.org/sqlite"
//	)
//
//	...
//
//
//	db, err := sql.Open("sqlite", dsnURI)
//
//	...
//
// # Debug and development versions
//
// A comma separated list of options can be passed to `go generate` via the
// environment variable GO_GENERATE. Some useful options include for example:
//
//	-DSQLITE_DEBUG
//	-DSQLITE_MEM_DEBUG
//	-ccgo-verify-structs
//
// To create a debug/development version, issue for example:
//
//	$ GO_GENERATE=-DSQLITE_DEBUG,-DSQLITE_MEM_DEBUG go generate
//
// Note: To run `go generate` you need to have modernc.org/ccgo/v3 installed.
//
// # Hacking
//
// This is an example of how to use the debug logs in modernc.org/libc when hunting a bug.
//
//	0:jnml@e5-1650:~/src/modernc.org/sqlite$ git status
//	On branch master
//	Your branch is up to date with 'origin/master'.
//
//	nothing to commit, working tree clean
//	0:jnml@e5-1650:~/src/modernc.org/sqlite$ git log -1
//	commit df33b8d15107f3cc777799c0fe105f74ef499e62 (HEAD -> master, tag: v1.21.1, origin/master, origin/HEAD, wips, ok)
//	Author: Jan Mercl <0xjnml@gmail.com>
//	Date:   Mon Mar 27 16:18:28 2023 +0200
//
//	    upgrade to SQLite 3.41.2
//	0:jnml@e5-1650:~/src/modernc.org/sqlite$ rm -f /tmp/libc.log ; go test -v -tags=libc.dmesg -run TestScalar ; ls -l /tmp/libc.log
//	test binary compiled for linux/amd64
//	=== RUN   TestScalar
//	--- PASS: TestScalar (0.09s)
//	PASS
//	ok  modernc.org/sqlite 0.128s
//	-rw-r--r-- 1 jnml jnml 76 Apr  6 11:22 /tmp/libc.log
//	0:jnml@e5-1650:~/src/modernc.org/sqlite$ cat /tmp/libc.log
//	[10723 sqlite.test] 2023-04-06 11:22:48.288066057 +0200 CEST m=+0.000707150
//	0:jnml@e5-1650:~/src/modernc.org/sqlite$
//
// The /tmp/libc.log file is created as requested. No useful messages there because none are enabled in libc. Let's try to enable Xwrite as an example.
//
//	0:jnml@e5-1650:~/src/modernc.org/libc$ git status
//	On branch master
//	Your branch is up to date with 'origin/master'.
//
//	Changes not staged for commit:
//	  (use "git add <file>..." to update what will be committed)
//	  (use "git restore <file>..." to discard changes in working directory)
//	modified:   libc_linux.go
//
//	no changes added to commit (use "git add" and/or "git commit -a")
//	0:jnml@e5-1650:~/src/modernc.org/libc$ git log -1
//	commit 1e22c18cf2de8aa86d5b19b165f354f99c70479c (HEAD -> master, tag: v1.22.3, origin/master, origin/HEAD)
//	Author: Jan Mercl <0xjnml@gmail.com>
//	Date:   Wed Feb 22 20:27:45 2023 +0100
//
//	    support sqlite 3.41 on linux targets
//	0:jnml@e5-1650:~/src/modernc.org/libc$ git diff
//	diff --git a/libc_linux.go b/libc_linux.go
//	index 1c2f482..ac1f08d 100644
//	--- a/libc_linux.go
//	+++ b/libc_linux.go
//	@@ -332,19 +332,19 @@ func Xwrite(t *TLS, fd int32, buf uintptr, count types.Size_t) types.Ssize_t {
//	                var n uintptr
//	                switch n, _, err = unix.Syscall(unix.SYS_WRITE, uintptr(fd), buf, uintptr(count)); err {
//	                case 0:
//	-                       // if dmesgs {
//	-                       //      // dmesg("%v: %d %#x: %#x\n%s", origin(1), fd, count, n, hex.Dump(GoBytes(buf, int(n))))
//	-                       //      dmesg("%v: %d %#x: %#x", origin(1), fd, count, n)
//	-                       // }
//	+                       if dmesgs {
//	+                               // dmesg("%v: %d %#x: %#x\n%s", origin(1), fd, count, n, hex.Dump(GoBytes(buf, int(n))))
//	+                               dmesg("%v: %d %#x: %#x", origin(1), fd, count, n)
//	+                       }
//	                        return types.Ssize_t(n)
//	                case errno.EAGAIN:
//	                        // nop
//	                }
//	        }
//
//	-       // if dmesgs {
//	-       //      dmesg("%v: fd %v, count %#x: %v", origin(1), fd, count, err)
//	-       // }
//	+       if dmesgs {
//	+               dmesg("%v: fd %v, count %#x: %v", origin(1), fd, count, err)
//	+       }
//	        t.setErrno(err)
//	        return -1
//	 }
//	0:jnml@e5-1650:~/src/modernc.org/libc$
//
// We need to tell the Go build system to use our local, patched/debug libc:
//
//	0:jnml@e5-1650:~/src/modernc.org/sqlite$ go work use $(go env GOPATH)/src/modernc.org/libc
//	0:jnml@e5-1650:~/src/modernc.org/sqlite$ go work use .
//
// And run the test again:
//
//	0:jnml@e5-1650:~/src/modernc.org/sqlite$ rm -f /tmp/libc.log ; go test -v -tags=libc.dmesg -run TestScalar ; ls -l /tmp/libc.log
//	test binary compiled for linux/amd64
//	=== RUN   TestScalar
//	--- PASS: TestScalar (0.26s)
//	PASS
//	ok   modernc.org/sqlite 0.285s
//	-rw-r--r-- 1 jnml jnml 918 Apr  6 11:29 /tmp/libc.log
//	0:jnml@e5-1650:~/src/modernc.org/sqlite$ cat /tmp/libc.log
//	[11910 sqlite.test] 2023-04-06 11:29:13.143589542 +0200 CEST m=+0.000689270
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 8 0x200: 0x200
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 8 0xc: 0xc
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 7 0x1000: 0x1000
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 7 0x1000: 0x1000
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 8 0x200: 0x200
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 8 0x4: 0x4
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 8 0x1000: 0x1000
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 8 0x4: 0x4
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 8 0x4: 0x4
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 8 0x1000: 0x1000
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 8 0x4: 0x4
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 8 0xc: 0xc
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 7 0x1000: 0x1000
//	[11910 sqlite.test] libc_linux.go:337:Xwrite: 7 0x1000: 0x1000
//	0:jnml@e5-1650:~/src/modernc.org/sqlite$
//
// # Sqlite documentation
//
// See https://sqlite.org/docs.html
package sqlite // import "modernc.org/sqlite"
