![badge](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/glebarez/fb4d23f63d866b3e1e58b26d2f5ed01f/raw/badge-gorm-tests.json)
![badge](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/glebarez/fb4d23f63d866b3e1e58b26d2f5ed01f/raw/badge-sqlite-version.json)
<br>[![Hits](https://hits.seeyoufarm.com/api/count/incr/badge.svg?url=https%3A%2F%2Fgithub.com%2Fglebarez%2Fsqlite&count_bg=%2379C83D&title_bg=%23555555&icon=baidu.svg&icon_color=%23E7E7E7&title=hits&edge_flat=false)](https://hits.seeyoufarm.com)
# Pure-Go SQLite driver for GORM
Pure-go (without cgo) implementation of SQLite driver for [GORM](https://gorm.io/)<br><br>
This driver has SQLite embedded, you don't need to install one separately.

# Usage

```go
import (
  "github.com/glebarez/sqlite"
  "gorm.io/gorm"
)

db, err := gorm.Open(sqlite.Open("sqlite.db"), &gorm.Config{})
```

### In-memory DB example
```go
db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
```

### Foreign-key constraint activation
Foreign-key constraint is disabled by default in SQLite. To activate it, use connection URL parameter:
```go
db, err := gorm.Open(sqlite.Open(":memory:?_pragma=foreign_keys(1)"), &gorm.Config{})
```
More info: [https://www.sqlite.org/foreignkeys.html](https://www.sqlite.org/foreignkeys.html)

# FAQ
## How is this better than standard GORM SQLite driver?
The [standard GORM driver for SQLite](https://github.com/go-gorm/sqlite) has one major drawback: it is based on a [Go-bindings of SQLite C-source](https://github.com/mattn/go-sqlite3) (this is called [cgo](https://go.dev/blog/cgo)). This fact imposes following restrictions on Go developers:
- to build and run your code, you will need a C compiler installed on a machine
- SQLite has many features that need to be enabled at compile time (e.g. [json support](https://www.sqlite.org/json1.html)). If you plan to use those, you will have to include proper build tags for every ```go``` command to work properly (```go run```, ```go test```, etc.).
- Because of C-compiler requirement, you can't build your Go code inside tiny stripped containers like (golang-alpine)
- Building on GCP is not possible because Google Cloud Platform does not allow gcc to be executed.

**Instead**, this driver is based on pure-Go implementation of SQLite (https://gitlab.com/cznic/sqlite), which is basically an original SQLite C-source AST, translated into Go! So, you may be sure you're using the original SQLite implementation under the hood.

## Is this tested good ?
Yes, The CI pipeline of this driver employs [whole test base](https://github.com/go-gorm/gorm/tree/master/tests) of GORM, which includes more than **12k** tests (see badge on the page-top). Testing is run against latest major releases of Go:
- 1.18
- 1.19

In following environments:
- Linux
- Windows
- MacOS

## Is it fast?
Well, it's slower than CGo implementation, but not terribly. See the [bechmark of underlying pure-Go driver vs CGo implementation](https://github.com/glebarez/go-sqlite/tree/master/benchmark).

## Included features
-  JSON1 (https://www.sqlite.org/json1.html)
-  Math functions (https://www.sqlite.org/lang_mathfunc.html)
