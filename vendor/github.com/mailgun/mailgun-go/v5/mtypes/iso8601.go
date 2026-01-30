package mtypes

import (
	"strconv"
	"time"

	"github.com/mailgun/errors"
)

// ISO8601Time Mailgun uses ISO8601 format without timezone for timestamps in the API key endpoints ('2006-01-02T15:04:05'),
// but by default Go's JSON package uses another format when decoding/encoding timestamps.
type ISO8601Time struct {
	time.Time
}

const (
	ISO8601Format = "2006-01-02T15:04:05"
)

func NewISO8601Time(str string) (ISO8601Time, error) {
	t, err := time.Parse(ISO8601Format, str)
	if err != nil {
		return ISO8601Time{}, err
	}
	return ISO8601Time{t}, nil
}

func (t ISO8601Time) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(t.Format(ISO8601Format))), nil
}

func (t *ISO8601Time) UnmarshalJSON(s []byte) error {
	q, err := strconv.Unquote(string(s))
	if err != nil {
		return err
	}

	if q == "" {
		return nil
	}

	var err1 error
	*t, err1 = NewISO8601Time(q)
	if err1 != nil {
		return errors.Errorf("%s", err1)
	}

	return nil
}

func (t ISO8601Time) String() string {
	return t.Format(ISO8601Format)
}
