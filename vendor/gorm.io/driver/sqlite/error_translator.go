package sqlite

import (
	"github.com/mattn/go-sqlite3"

	"gorm.io/gorm"
)

var errCodes = map[string]sqlite3.ErrNoExtended{
	"uniqueConstraint": 2067,
}

func (dialector Dialector) Translate(err error) error {
	if sqliteErr, ok := err.(*sqlite3.Error); ok {
		if sqliteErr.ExtendedCode == errCodes["uniqueConstraint"] {
			return gorm.ErrDuplicatedKey
		}
	}

	return err
}
