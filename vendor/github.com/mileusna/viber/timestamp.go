package viber

import (
	"fmt"
	"strconv"
	"time"
)

// Timestamp struct for easily MarshalJSON/UnmarshalJSON unix timestamp to time.Time
type Timestamp struct {
	time.Time
}

// MarshalJSON converts golang time to unix timestam number
func (t *Timestamp) MarshalJSON() ([]byte, error) {
	ts := t.Time.Unix()
	stamp := fmt.Sprint(ts)
	return []byte(stamp), nil
}

// UnmarshalJSON converts unix timestamp to golang time
func (t *Timestamp) UnmarshalJSON(b []byte) error {
	ts, err := strconv.Atoi(string(b))
	if err != nil {
		return err
	}
	t.Time = time.Unix(int64(ts/1000), 0)
	return nil
}
