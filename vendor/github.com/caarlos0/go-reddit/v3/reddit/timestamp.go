package reddit

import (
	"strconv"
	"time"
)

// Timestamp represents a time that can be unmarshalled from a JSON string
// formatted as either an RFC3339 or Unix timestamp.
type Timestamp struct {
	time.Time
}

// MarshalJSON implements the json.Marshaler interface.
func (t *Timestamp) MarshalJSON() ([]byte, error) {
	if t == nil || t.Time.IsZero() {
		return []byte(`false`), nil
	}

	parsed := t.Time.Format(time.RFC3339)
	return []byte(`"` + parsed + `"`), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// Time is expected in RFC3339 or Unix format.
func (t *Timestamp) UnmarshalJSON(data []byte) (err error) {
	str := string(data)

	// "edited" for posts and comments is either false, or a timestamp.
	if str == "false" {
		return
	}

	f, err := strconv.ParseFloat(str, 64)
	if err == nil {
		t.Time = time.Unix(int64(f), 0).UTC()
	} else {
		t.Time, err = time.Parse(`"`+time.RFC3339+`"`, str)
	}

	return
}

// Equal reports whether t and u are equal based on time.Equal
func (t Timestamp) Equal(u Timestamp) bool {
	return t.Time.Equal(u.Time)
}
