package wasmsqlite

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/ncruces/go-sqlite3"
	"gorm.io/gorm"
	"modernc.org/sqlite"
)

func TestDialector(t *testing.T) {
	// This is the DSN of the in-memory SQLite database for these tests.
	const InMemoryDSN = "file:testdatabase?mode=memory&cache=shared"
	// This is the custom SQLite driver name.
	const CustomDriverName = "my_custom_driver"

	// Register the custom SQlite3 driver.
	// It will have one custom function called "my_custom_function".

	sql.Register(CustomDriverName,
		&sqlite.Driver{},
	)

	rows := []struct {
		description  string
		dialector    *Dialector
		openSuccess  bool
		query        string
		querySuccess bool
	}{
		{
			description: "Default driver",
			dialector: &Dialector{
				DSN: InMemoryDSN,
			},
			openSuccess:  true,
			query:        "SELECT 1",
			querySuccess: true,
		},
		{
			description: "Explicit default driver",
			dialector: &Dialector{
				DriverName: DriverName,
				DSN:        InMemoryDSN,
			},
			openSuccess:  true,
			query:        "SELECT 1",
			querySuccess: true,
		},
		{
			description: "Bad driver",
			dialector: &Dialector{
				DriverName: "not-a-real-driver",
				DSN:        InMemoryDSN,
			},
			openSuccess: false,
		},
		// {
		// 	description: "Explicit default driver, custom function",
		// 	dialector: &Dialector{
		// 		DriverName: DriverName,
		// 		DSN:        InMemoryDSN,
		// 	},
		// 	openSuccess:  true,
		// 	query:        "SELECT my_custom_function()",
		// 	querySuccess: false,
		// },
		{
			description: "Custom driver",
			dialector: &Dialector{
				DriverName: CustomDriverName,
				DSN:        InMemoryDSN,
			},
			openSuccess:  true,
			query:        "SELECT 1",
			querySuccess: true,
		},
		// {
		// 	description: "Custom driver, custom function",
		// 	dialector: &Dialector{
		// 		DriverName: CustomDriverName,
		// 		DSN:        InMemoryDSN,
		// 	},
		// 	openSuccess:  true,
		// 	query:        "SELECT my_custom_function()",
		// 	querySuccess: true,
		// },
	}
	for rowIndex, row := range rows {
		t.Run(fmt.Sprintf("%d/%s", rowIndex, row.description), func(t *testing.T) {
			db, err := gorm.Open(row.dialector, &gorm.Config{})
			if !row.openSuccess {
				if err == nil {
					t.Errorf("Expected Open to fail.")
				}
				return
			}

			if err != nil {
				t.Errorf("Expected Open to succeed; got error: %v", err)
			}
			if db == nil {
				t.Errorf("Expected db to be non-nil.")
			}
			if row.query != "" {
				err = db.Exec(row.query).Error
				if !row.querySuccess {
					if err == nil {
						t.Errorf("Expected query to fail.")
					}
					return
				}

				if err != nil {
					t.Errorf("Expected query to succeed; got error: %v", err)
				}
			}
		})
	}
}
