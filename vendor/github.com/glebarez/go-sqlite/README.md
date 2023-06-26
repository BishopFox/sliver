[![Tests](https://github.com/glebarez/go-sqlite/actions/workflows/tests.yml/badge.svg)](https://github.com/glebarez/go-sqlite/actions/workflows/tests.yml)
![badge](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/glebarez/0fd7561eb29baf31d5362ffee1ae1702/raw/badge-sqlite-version-with-date.json)

# go-sqlite
This is a pure-Go SQLite driver for Golang's native [database/sql](https://pkg.go.dev/database/sql) package.
The driver has [Go-based implementation of SQLite](https://gitlab.com/cznic/sqlite) embedded in itself (so, you don't need to install SQLite separately)

# Usage

## Example

```go
package main

import (
	"database/sql"
	"log"

	_ "github.com/glebarez/go-sqlite"
)

func main() {
	// connect
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}

	// get SQLite version
	_ := db.QueryRow("select sqlite_version()")
}
```

## Connection string examples
- in-memory SQLite: ```":memory:"```
- on-disk SQLite: ```"path/to/some.db"```
- Foreign-key constraint activation: ```":memory:?_pragma=foreign_keys(1)"```

## Settings PRAGMAs in connection string
Any SQLIte pragma can be preset for a Database connection using ```_pragma``` query parameter. Examples:
- [journal mode](https://www.sqlite.org/pragma.html#pragma_journal_mode): ```path/to/some.db?_pragma=journal_mode(WAL)```
- [busy timeout](https://www.sqlite.org/pragma.html#pragma_busy_timeout): ```:memory:?_pragma=busy_timeout(5000)```

Multiple PRAGMAs can be specified, e.g.:<br>
```path/to/some.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)```
