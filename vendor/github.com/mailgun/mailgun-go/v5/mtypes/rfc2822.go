package mtypes

import (
	"errors"
	"strconv"
	"time"
)

// RFC2822Time Mailgun uses RFC2822 format for timestamps in most endpoints ('Thu, 13 Oct 2011 18:02:00 +0000'), but
// by default Go's JSON package uses another format when decoding/encoding timestamps.
// https://documentation.mailgun.com/docs/mailgun/user-manual/get-started/#date-format
// TODO(v6): make a struct and embed time.Time to inherit all its methods.
type RFC2822Time time.Time

var rfc2822TimeFormats = []string{
	time.RFC1123,
	time.RFC1123Z,
	time.RFC3339, // Just in case. See https://github.com/mailgun/mailgun-go/issues/506
}

func NewRFC2822Time(str string) (RFC2822Time, error) {
	errs := make([]error, 0, len(rfc2822TimeFormats))
	for _, format := range rfc2822TimeFormats {
		t, err := time.Parse(format, str)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		return RFC2822Time(t), nil
	}

	return RFC2822Time{}, errors.Join(errs...)
}

func (t RFC2822Time) Unix() int64 {
	return time.Time(t).Unix()
}

func (t RFC2822Time) IsZero() bool {
	return time.Time(t).IsZero()
}

func (t RFC2822Time) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(time.Time(t).Format(time.RFC1123Z))), nil
}

func (t *RFC2822Time) UnmarshalJSON(s []byte) error {
	q, err := strconv.Unquote(string(s))
	if err != nil {
		return err
	}

	parsed, err := NewRFC2822Time(q)
	if err != nil {
		return err
	}

	*t = parsed

	return nil
}

func (t RFC2822Time) String() string {
	return time.Time(t).Format(time.RFC1123)
}
