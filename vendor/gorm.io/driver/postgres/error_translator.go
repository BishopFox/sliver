package postgres

import (
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

var errCodes = map[string]string{
	"uniqueConstraint": "23505",
}

func (dialector Dialector) Translate(err error) error {
	if pgErr, ok := err.(*pgconn.PgError); ok {
		if pgErr.Code == errCodes["uniqueConstraint"] {
			return gorm.ErrDuplicatedKey
		}
	}

	return err
}
