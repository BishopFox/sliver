package twitter

import (
	"bufio"
	"bytes"
	"io"
	"time"
)

// stopped returns true if the done channel receives, false otherwise.
func stopped(done <-chan struct{}) bool {
	select {
	case <-done:
		return true
	default:
		return false
	}
}

// sleepOrDone pauses the current goroutine until the done channel receives
// or until at least the duration d has elapsed, whichever comes first. This
// is similar to time.Sleep(d), except it can be interrupted.
func sleepOrDone(d time.Duration, done <-chan struct{}) {
	sleep := time.NewTimer(d)
	defer sleep.Stop()
	select {
	case <-sleep.C:
		return
	case <-done:
		return
	}
}

// streamResponseBodyReader is a buffered reader for Twitter stream response
// body. It can scan the arbitrary length of response body unlike bufio.Scanner.
type streamResponseBodyReader struct {
	reader *bufio.Reader
	buf    bytes.Buffer
}

// newStreamResponseBodyReader returns an instance of streamResponseBodyReader
// for the given Twitter stream response body.
func newStreamResponseBodyReader(body io.Reader) *streamResponseBodyReader {
	return &streamResponseBodyReader{reader: bufio.NewReader(body)}
}

// readNext reads Twitter stream response body and returns the next stream
// content if exists. Returns io.EOF error if we reached the end of the stream
// and there's no more message to read.
func (r *streamResponseBodyReader) readNext() ([]byte, error) {
	// Discard all the bytes from buf and continue to use the allocated memory
	// space for reading the next message.
	r.buf.Truncate(0)
	for {
		// Twitter stream messages are separated with "\r\n", and a valid
		// message may sometimes contain '\n' in the middle.
		// bufio.Reader.Read() can accept one byte delimiter only, so we need to
		// first break out each line on '\n' and then check whether the line ends
		// with "\r\n" to find message boundaries.
		// https://dev.twitter.com/streaming/overview/processing
		line, err := r.reader.ReadBytes('\n')
		// Non-EOF error should be propagated to callers immediately.
		if err != nil && err != io.EOF {
			return nil, err
		}
		// EOF error means that we reached the end of the stream body before finding
		// delimiter '\n'. If "line" is empty, it means the reader didn't read any
		// data from the stream before reaching EOF and there's nothing to append to
		// buf.
		if err == io.EOF && len(line) == 0 {
			// if buf has no data, propagate io.EOF to callers and let them know that
			// we've finished processing the stream.
			if r.buf.Len() == 0 {
				return nil, err
			}
			// Otherwise, we still have a remaining stream message to return.
			break
		}
		// If the line ends with "\r\n", it's the end of one stream message data.
		if bytes.HasSuffix(line, []byte("\r\n")) {
			// reader.ReadBytes() returns a slice including the delimiter itself, so
			// we need to trim '\n' as well as '\r' from the end of the slice.
			r.buf.Write(bytes.TrimRight(line, "\r\n"))
			break
		}
		// Otherwise, the line is not the end of a stream message, so we append
		// the line to buf and continue to scan lines.
		r.buf.Write(line)
	}

	// Get the stream message bytes from buf. Not that Bytes() won't mark the
	// returned data as "read", and we need to explicitly call Truncate(0) to
	// discard from buf before writing the next stream message to buf.
	return r.buf.Bytes(), nil
}
