package postgres

import (
	"encoding/json"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

var errCodes = map[string]string{
	"uniqueConstraint": "23505",
}

type ErrMessage struct {
	Code     string `json:"Code"`
	Severity string `json:"Severity"`
	Message  string `json:"Message"`
}

// Translate it will translate the error to native gorm errors.
// Since currently gorm supporting both pgx and pg drivers, only checking for pgx PgError types is not enough for translating errors, so we have additional error json marshal fallback.
func (dialector Dialector) Translate(err error) error {
	if pgErr, ok := err.(*pgconn.PgError); ok {
		if pgErr.Code == errCodes["uniqueConstraint"] {
			return gorm.ErrDuplicatedKey
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

	if errMsg.Code == errCodes["uniqueConstraint"] {
		return gorm.ErrDuplicatedKey
	}
	return err
}
