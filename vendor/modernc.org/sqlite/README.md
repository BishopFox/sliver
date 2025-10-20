# sqlite

Package sqlite is a cgo-free port of SQLite. Although you could see mattn's driver (`github.com/mattn/go-sqlite3`) in go.mod file, we import it for tests only.

SQLite is an in-process implementation of a self-contained, serverless,
zero-configuration, transactional SQL database engine.

## Thanks

This project is sponsored by Schleibinger Geräte Teubert u. Greim GmbH by
allowing one of the maintainers to work on it also in office hours.

## Installation

    $ go get modernc.org/sqlite

## Documentation

[pkg.go.dev/modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)

## Builders

[modern-c.appspot.com/-/builder/?importpath=modernc.org%2fsqlite](https://modern-c.appspot.com/-/builder/?importpath=modernc.org%2fsqlite)

## Speedtest1

Numbers for the pure Go version were produced by

     ~/src/modernc.org/sqlite/speedtest1$ go build && ./speedtest1

Numbers for the pure C version were produced by

     ~/src/modernc.org/sqlite/testdata/sqlite-src-3410200/test$ gcc speedtest1.c ../../sqlite-amalgamation-3410200/sqlite3.c -lpthread -ldl && ./a.out

The results are from Go version 1.20.4 and GCC version 10.2.1 on a
Linux/amd64 machine, CPU: AMD Ryzen 9 3900X 12-Core Processor × 24, 128GB
RAM. Shown are the best of 3 runs.

     Go											C

     -- Speedtest1 for SQLite 3.41.2 2023-03-22 11:56:21 0d1fc92f94cb6b76bffe3ec34d69	-- Speedtest1 for SQLite 3.41.2 2023-03-22 11:56:21 0d1fc92f94cb6b76bffe3ec34d69
      100 - 50000 INSERTs into table with no index......................    0.071s            100 - 50000 INSERTs into table with no index......................    0.077s
      110 - 50000 ordered INSERTS with one index/PK.....................    0.114s            110 - 50000 ordered INSERTS with one index/PK.....................    0.082s
      120 - 50000 unordered INSERTS with one index/PK...................    0.137s            120 - 50000 unordered INSERTS with one index/PK...................    0.099s
      130 - 25 SELECTS, numeric BETWEEN, unindexed......................    0.083s            130 - 25 SELECTS, numeric BETWEEN, unindexed......................    0.091s
      140 - 10 SELECTS, LIKE, unindexed.................................    0.210s            140 - 10 SELECTS, LIKE, unindexed.................................    0.120s
      142 - 10 SELECTS w/ORDER BY, unindexed............................    0.276s            142 - 10 SELECTS w/ORDER BY, unindexed............................    0.182s
      145 - 10 SELECTS w/ORDER BY and LIMIT, unindexed..................    0.183s            145 - 10 SELECTS w/ORDER BY and LIMIT, unindexed..................    0.099s
      150 - CREATE INDEX five times.....................................    0.172s            150 - CREATE INDEX five times.....................................    0.127s
      160 - 10000 SELECTS, numeric BETWEEN, indexed.....................    0.080s            160 - 10000 SELECTS, numeric BETWEEN, indexed.....................    0.078s
      161 - 10000 SELECTS, numeric BETWEEN, PK..........................    0.080s            161 - 10000 SELECTS, numeric BETWEEN, PK..........................    0.078s
      170 - 10000 SELECTS, text BETWEEN, indexed........................    0.187s            170 - 10000 SELECTS, text BETWEEN, indexed........................    0.169s
      180 - 50000 INSERTS with three indexes............................    0.196s            180 - 50000 INSERTS with three indexes............................    0.154s
      190 - DELETE and REFILL one table.................................    0.200s            190 - DELETE and REFILL one table.................................    0.155s
      200 - VACUUM......................................................    0.180s            200 - VACUUM......................................................    0.142s
      210 - ALTER TABLE ADD COLUMN, and query...........................    0.004s            210 - ALTER TABLE ADD COLUMN, and query...........................    0.005s
      230 - 10000 UPDATES, numeric BETWEEN, indexed.....................    0.093s            230 - 10000 UPDATES, numeric BETWEEN, indexed.....................    0.080s
      240 - 50000 UPDATES of individual rows............................    0.153s            240 - 50000 UPDATES of individual rows............................    0.137s
      250 - One big UPDATE of the whole 50000-row table.................    0.024s            250 - One big UPDATE of the whole 50000-row table.................    0.019s
      260 - Query added column after filling............................    0.004s            260 - Query added column after filling............................    0.005s
      270 - 10000 DELETEs, numeric BETWEEN, indexed.....................    0.278s            270 - 10000 DELETEs, numeric BETWEEN, indexed.....................    0.263s
      280 - 50000 DELETEs of individual rows............................    0.188s            280 - 50000 DELETEs of individual rows............................    0.180s
      290 - Refill two 50000-row tables using REPLACE...................    0.411s            290 - Refill two 50000-row tables using REPLACE...................    0.359s
      300 - Refill a 50000-row table using (b&1)==(a&1).................    0.175s            300 - Refill a 50000-row table using (b&1)==(a&1).................    0.151s
      310 - 10000 four-ways joins.......................................    0.427s            310 - 10000 four-ways joins.......................................    0.365s
      320 - subquery in result set......................................    0.440s            320 - subquery in result set......................................    0.521s
      400 - 70000 REPLACE ops on an IPK.................................    0.125s            400 - 70000 REPLACE ops on an IPK.................................    0.106s
      410 - 70000 SELECTS on an IPK.....................................    0.081s            410 - 70000 SELECTS on an IPK.....................................    0.078s
      500 - 70000 REPLACE on TEXT PK....................................    0.174s            500 - 70000 REPLACE on TEXT PK....................................    0.116s
      510 - 70000 SELECTS on a TEXT PK..................................    0.153s            510 - 70000 SELECTS on a TEXT PK..................................    0.117s
      520 - 70000 SELECT DISTINCT.......................................    0.083s            520 - 70000 SELECT DISTINCT.......................................    0.067s
      980 - PRAGMA integrity_check......................................    0.436s            980 - PRAGMA integrity_check......................................    0.377s
      990 - ANALYZE.....................................................    0.107s            990 - ANALYZE.....................................................    0.038s
            TOTAL.......................................................    5.525s                  TOTAL.......................................................    4.637s

This particular test executes 16.1% faster in the C version.

## Troubleshooting

* Q: **How can I write to a database concurrently without getting the `database is locked` error (or `SQLITE_BUSY`)?**
     * A: You can't. The C sqlite implementation does not allow concurrent writes, and this libary does not modify that behaviour. You can, however, use [DB.SetMaxOpenConns(1)](https://pkg.go.dev/database/sql#DB.SetMaxOpenConns) so that only 1 connection is ever used by the `DB`, allowing concurrent access to DB without making the writes concurrent. More information on issues [#65](https://gitlab.com/cznic/sqlite/-/issues/65) and [#106](https://gitlab.com/cznic/sqlite/-/issues/106).
