package driver

import (
	"database/sql/driver"
	"time"
)

// Convert a string in [time.RFC3339Nano] format into a [time.Time]
// if it roundtrips back to the same string.
// This way times can be persisted to, and recovered from, the database,
// but if a string is needed, [database/sql] will recover the same string.
func stringOrTime(text string) driver.Value {
	// Weed out (some) values that can't possibly be
	// [time.RFC3339Nano] timestamps.
	if len(text) < len("2006-01-02T15:04:05Z") {
		return text
	}
	if len(text) > len(time.RFC3339Nano) {
		return text
	}
	if text[4] != '-' || text[10] != 'T' || text[16] != ':' {
		return text
	}

	// Slow path.
	date, err := time.Parse(time.RFC3339Nano, text)
	if err == nil && date.Format(time.RFC3339Nano) == text {
		return date
	}
	return text
}
