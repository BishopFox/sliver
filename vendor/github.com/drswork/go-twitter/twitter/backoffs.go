package twitter

import (
	"time"

	"github.com/cenkalti/backoff/v4"
)

func newExponentialBackOff() *backoff.ExponentialBackOff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 5 * time.Second
	b.Multiplier = 2.0
	b.MaxInterval = 320 * time.Second
	b.Reset()
	return b
}

func newAggressiveExponentialBackOff() *backoff.ExponentialBackOff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 1 * time.Minute
	b.Multiplier = 2.0
	b.MaxInterval = 16 * time.Minute
	b.Reset()
	return b
}
