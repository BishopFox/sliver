package types

import (
	"encoding/json"
	"time"
)

const DateFormat = "2006-01-02"

type Date struct {
	time.Time
}

func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Format(DateFormat))
}

func (d *Date) UnmarshalJSON(data []byte) error {
	var dateStr string
	err := json.Unmarshal(data, &dateStr)
	if err != nil {
		return err
	}
	parsed, err := time.Parse(DateFormat, dateStr)
	if err != nil {
		return err
	}
	d.Time = parsed
	return nil
}

func (d Date) String() string {
	return d.Format(DateFormat)
}

func (d *Date) UnmarshalText(data []byte) error {
	parsed, err := time.Parse(DateFormat, string(data))
	if err != nil {
		return err
	}
	d.Time = parsed
	return nil
}

// Bind implements the runtime.Binder interface so that Date is treated as a
// scalar value when binding query parameters rather than being decomposed as
// a struct with key-value pairs.
func (d *Date) Bind(src string) error {
	if src == "" {
		return nil
	}
	parsed, err := time.Parse(DateFormat, src)
	if err != nil {
		return err
	}
	d.Time = parsed
	return nil
}
