DB
===

[GORM-based](https://gorm.io/) database interactions.

#### Contents:

 * `gosqlite/` - A fork of the GORM sqlite driver that uses a pure go sqlite3 implementation. This can be compiled using the Go build tag `gosqlite`
 * `models/` - The database/GORM models 
 * `db.go` - Primary abstraction for client and db sessions
 * `helpers.go` - Helper functions for querying the GORM models
 * `logger.go` - Database logger
 * `sql_cgo.go` - The CGO sqlite client
 * `sql_go.go` - The pure Go sqlite client
 * `sql.go` - Database setup and configuration
