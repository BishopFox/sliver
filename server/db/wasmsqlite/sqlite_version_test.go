package wasmsqlite

import (
	"database/sql"
	"log"
	"testing"
)

func TestSQLiteVersion(t *testing.T) {
	var version string

	db, err := sql.Open(DriverName, ":memory:")
	if err != nil {
		log.Fatal(err)
	}

	row := db.QueryRow("select sqlite_version()")
	if row.Scan(&version) != nil {
		log.Fatal(err)
	}

	t.Log(version)
}
