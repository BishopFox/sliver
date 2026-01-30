package types

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// Implementation taken from
// https://github.com/lib/pq/blob/master/encode.go#L518-L538.
//
// It would be great to just import it from there, but you get duplicate driver
// exceptions

// A NullTime is a Time that may be null. It can be encoded or decoded from
// JSON or the database.
type NullTime struct {
	Valid bool
	Time  time.Time
}

func (nt *NullTime) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		nt.Valid = false
		return nil
	}
	var t time.Time
	err := json.Unmarshal(b, &t)
	if err != nil {
		return err
	}
	nt.Valid = true
	nt.Time = t
	return nil
}

func (nt NullTime) MarshalJSON() ([]byte, error) {
	if !nt.Valid {
		return []byte("null"), nil
	}
	b, err := json.Marshal(nt.Time)
	if err != nil {
		return []byte{}, err
	}
	return b, nil
}

// Scan implements the Scanner interface.
func (nt *NullTime) Scan(value interface{}) error {
	nt.Time, nt.Valid = value.(time.Time)
	return nil
}

// Value implements the driver Valuer interface.
func (nt NullTime) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}
	return nt.Time, nil
}
