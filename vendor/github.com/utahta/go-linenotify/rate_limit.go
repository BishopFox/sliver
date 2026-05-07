package linenotify

import (
	"net/http"
	"strconv"
	"time"
)

// RateLimit
type RateLimit struct {
	Limit          int
	Remaining      int
	ImageLimit     int
	ImageRemaining int
	Reset          time.Time
}

// Parse parses rate limit from response header
func (r *RateLimit) Parse(header http.Header) {
	if v, err := strconv.Atoi(header.Get("X-RateLimit-Limit")); err == nil {
		r.Limit = v
	}

	if v, err := strconv.Atoi(header.Get("X-RateLimit-Remaining")); err == nil {
		r.Remaining = v
	}

	if v, err := strconv.Atoi(header.Get("X-RateLimit-ImageLimit")); err == nil {
		r.ImageLimit = v
	}

	if v, err := strconv.Atoi(header.Get("X-RateLimit-ImageRemaining")); err == nil {
		r.ImageRemaining = v
	}

	if v, err := strconv.ParseInt(header.Get("X-RateLimit-Reset"), 10, 64); err == nil {
		r.Reset = time.Unix(v, 0)
	}
}
