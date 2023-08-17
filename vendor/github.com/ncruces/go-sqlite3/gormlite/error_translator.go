package gormlite

import (
	"errors"

	"github.com/ncruces/go-sqlite3"
	"gorm.io/gorm"
)

func (dialector Dialector) Translate(err error) error {
	switch {
	case
		errors.Is(err, sqlite3.CONSTRAINT_UNIQUE),
		errors.Is(err, sqlite3.CONSTRAINT_PRIMARYKEY):
		return gorm.ErrDuplicatedKey
	case
		errors.Is(err, sqlite3.CONSTRAINT_FOREIGNKEY):
		return gorm.ErrForeignKeyViolated
	}
	return err
}
