package postgres

import (
	"encoding/json"

	"gorm.io/gorm"

	"github.com/jackc/pgx/v5/pgconn"
)

var errCodes = map[string]error{
	"23505": gorm.ErrDuplicatedKey,
	"23503": gorm.ErrForeignKeyViolated,
	"42703": gorm.ErrInvalidField,
}

type ErrMessage struct {
	Code     string
	Severity string
	Message  string
}

// Translate it will translate the error to native gorm errors.
// Since currently gorm supporting both pgx and pg drivers, only checking for pgx PgError types is not enough for translating errors, so we have additional error json marshal fallback.
func (dialector Dialector) Translate(err error) error {
	if pgErr, ok := err.(*pgconn.PgError); ok {
		if translatedErr, found := errCodes[pgErr.Code]; found {
			return translatedErr
		}
		return err
	}

	parsedErr, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		return err
	}

	var errMsg ErrMessage
	unmarshalErr := json.Unmarshal(parsedErr, &errMsg)
	if unmarshalErr != nil {
		return err
	}

	if translatedErr, found := errCodes[errMsg.Code]; found {
		return translatedErr
	}
	return err
}
