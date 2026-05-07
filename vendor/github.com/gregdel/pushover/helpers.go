package pushover

import (
	"encoding/json"
	"fmt"
	"time"
)

// Helper to unmarshal a timestamp as string to a time.Time.
type timestamp struct{ *time.Time }

func (t *timestamp) UnmarshalJSON(data []byte) error {
	var i int64
	if err := json.Unmarshal(data, &i); err != nil {
		return err
	}

	if i > 0 {
		unixTime := time.Unix(i, 0)
		*t = timestamp{&unixTime}
	}

	return nil
}

// Helper to unmarshal a int as a boolean.
type intBool bool

func (i *intBool) UnmarshalJSON(data []byte) error {
	var v int64
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	switch v {
	case 0:
		*i = false
	case 1:
		*i = true
	default:
		return fmt.Errorf("failed to unmarshal int to bool")
	}

	return nil
}
