package mysql

import (
	"github.com/go-sql-driver/mysql"

	"gorm.io/gorm"
)

// The error codes to map mysql errors to gorm errors, here is the mysql error codes reference https://dev.mysql.com/doc/mysql-errors/8.0/en/server-error-reference.html.
var errCodes = map[uint16]error{
	1062: gorm.ErrDuplicatedKey,
	1451: gorm.ErrForeignKeyViolated,
	1452: gorm.ErrForeignKeyViolated,
}

func (dialector Dialector) Translate(err error) error {
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		if translatedErr, found := errCodes[mysqlErr.Number]; found {
			return translatedErr
		}
		return mysqlErr
	}

	return err
}
