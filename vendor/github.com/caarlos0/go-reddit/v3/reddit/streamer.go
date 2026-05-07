package reddit

import "time"

const defaultStreamInterval = time.Second * 5

type streamConfig struct {
	Interval       time.Duration
	DiscardInitial bool
	MaxRequests    int
}

// StreamOpt is a configuration option to configure a stream.
type StreamOpt func(*streamConfig)

// StreamInterval sets the frequency at which data will be fetched for the stream.
// If the duration is 0 or less, it will not be set and the default will be used.
func StreamInterval(v time.Duration) StreamOpt {
	return func(c *streamConfig) {
		if v > 0 {
			c.Interval = v
		}
	}
}

// StreamDiscardInitial will discard data from the first fetch for the stream.
func StreamDiscardInitial(c *streamConfig) {
	c.DiscardInitial = true
}

// StreamMaxRequests sets a limit on the number of times data is fetched for a stream.
// If less than or equal to 0, it is assumed to be infinite.
func StreamMaxRequests(v int) StreamOpt {
	return func(c *streamConfig) {
		if v > 0 {
			c.MaxRequests = v
		}
	}
}

// Streamer streams data to the client.
// type Streamer interface {
// 	Stream() (<-chan *rootListing, <-chan error, func())
// }
