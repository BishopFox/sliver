package mtypes

import (
	"strconv"
	"time"

	"github.com/mailgun/errors"
)

// RFC2822Time Mailgun uses RFC2822 format for timestamps in most endpoints ('Thu, 13 Oct 2011 18:02:00 +0000'), but
// by default Go's JSON package uses another format when decoding/encoding timestamps.
// https://documentation.mailgun.com/docs/mailgun/user-manual/get-started/#date-format
// TODO(v6): make a struct and embed time.Time to inherit all its methods.
type RFC2822Time time.Time

func NewRFC2822Time(str string) (RFC2822Time, error) {
	t, err := time.Parse(time.RFC1123, str)
	if err != nil {
		return RFC2822Time{}, err
	}
	return RFC2822Time(t), nil
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

	var err1 error
	*(*time.Time)(t), err1 = time.Parse(time.RFC1123, q)
	if err1 != nil {
		var err2 error
		*(*time.Time)(t), err2 = time.Parse(time.RFC1123Z, q)
		if err2 != nil {
			// TODO(go1.20): use errors.Join:
			return errors.Errorf("%s; %s", err1, err2)
		}
	}

	return nil
}

func (t RFC2822Time) String() string {
	return time.Time(t).Format(time.RFC1123)
}
