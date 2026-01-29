package pushover

import (
	"net/http"
	"strconv"
	"time"
)

// Limit represents the limitation of the application. This information is
// fetched when posting a new message.
//	Headers example:
//		X-Limit-App-Limit: 7500
// 		X-Limit-App-Remaining: 7496
// 		X-Limit-App-Reset: 1393653600
type Limit struct {
	// Total number of messages you can send during a month.
	Total int
	// Remaining number of messages you can send until the next reset.
	Remaining int
	// NextReset is the time when all the app counters will be reseted.
	NextReset time.Time
}

func newLimit(headers http.Header) (*Limit, error) {
	headersStrings := []string{
		"X-Limit-App-Limit",
		"X-Limit-App-Remaining",
		"X-Limit-App-Reset",
	}
	headersValues := map[string]int{}

	for _, header := range headersStrings {
		// Check if the header is present
		h, ok := headers[header]
		if !ok {
			return nil, ErrInvalidHeaders
		}

		// The header must have only one element
		if len(h) != 1 {
			return nil, ErrInvalidHeaders
		}

		i, err := strconv.Atoi(h[0])
		if err != nil {
			return nil, err
		}

		headersValues[header] = i
	}

	return &Limit{
		Total:     headersValues["X-Limit-App-Limit"],
		Remaining: headersValues["X-Limit-App-Remaining"],
		NextReset: time.Unix(int64(headersValues["X-Limit-App-Reset"]), 0),
	}, nil
}
