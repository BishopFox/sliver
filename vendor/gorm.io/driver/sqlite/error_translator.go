package sqlite

import (
	"encoding/json"

	"gorm.io/gorm"
)

var errCodes = map[string]int{
	"uniqueConstraint": 2067,
}

type ErrMessage struct {
	Code         int `json:"Code"`
	ExtendedCode int `json:"ExtendedCode"`
	SystemErrno  int `json:"SystemErrno"`
}

// Translate it will translate the error to native gorm errors.
// We are not using go-sqlite3 error type intentionally here because it will need the CGO_ENABLED=1 and cross-C-compiler.
func (dialector Dialector) Translate(err error) error {
	parsedErr, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		return err
	}

	var errMsg ErrMessage
	unmarshalErr := json.Unmarshal(parsedErr, &errMsg)
	if unmarshalErr != nil {
		return err
	}

	if errMsg.ExtendedCode == errCodes["uniqueConstraint"] {
		return gorm.ErrDuplicatedKey
	}
	return err
}
