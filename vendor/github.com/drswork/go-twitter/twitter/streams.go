package twitter

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/dghubble/sling"
)

const (
	userAgent    = "go-twitter v0.1"
	publicStream = "https://stream.twitter.com/1.1/"
	userStream   = "https://userstream.twitter.com/1.1/"
	siteStream   = "https://sitestream.twitter.com/1.1/"
)

// StreamService provides methods for accessing the Twitter Streaming API.
type StreamService struct {
	client *http.Client
	public *sling.Sling
	user   *sling.Sling
	site   *sling.Sling
}

// newStreamService returns a new StreamService.
func newStreamService(client *http.Client, sling *sling.Sling) *StreamService {
	sling.Set("User-Agent", userAgent)
	return &StreamService{
		client: client,
		public: sling.New().Base(publicStream).Path("statuses/"),
		user:   sling.New().Base(userStream),
		site:   sling.New().Base(siteStream),
	}
}

// StreamFilterParams are parameters for StreamService.Filter.
type StreamFilterParams struct {
	FilterLevel   string   `url:"filter_level,omitempty"`
	Follow        []string `url:"follow,omitempty,comma"`
	Language      []string `url:"language,omitempty,comma"`
	Locations     []string `url:"locations,omitempty,comma"`
	StallWarnings *bool    `url:"stall_warnings,omitempty"`
	Track         []string `url:"track,omitempty,comma"`
}

// Filter returns messages that match one or more filter predicates.
// https://dev.twitter.com/streaming/reference/post/statuses/filter
func (srv *StreamService) Filter(params *StreamFilterParams) (*Stream, error) {
	req, err := srv.public.New().Post("filter.json").QueryStruct(params).Request()
	if err != nil {
		return nil, err
	}
	return newStream(srv.client, req), nil
}

// StreamSampleParams are the parameters for StreamService.Sample.
type StreamSampleParams struct {
	StallWarnings *bool    `url:"stall_warnings,omitempty"`
	Language      []string `url:"language,omitempty,comma"`
}

// Sample returns a small sample of public stream messages.
// https://dev.twitter.com/streaming/reference/get/statuses/sample
func (srv *StreamService) Sample(params *StreamSampleParams) (*Stream, error) {
	req, err := srv.public.New().Get("sample.json").QueryStruct(params).Request()
	if err != nil {
		return nil, err
	}
	return newStream(srv.client, req), nil
}

// StreamUserParams are the parameters for StreamService.User.
type StreamUserParams struct {
	FilterLevel   string   `url:"filter_level,omitempty"`
	Language      []string `url:"language,omitempty,comma"`
	Locations     []string `url:"locations,omitempty,comma"`
	Replies       string   `url:"replies,omitempty"`
	StallWarnings *bool    `url:"stall_warnings,omitempty"`
	Track         []string `url:"track,omitempty,comma"`
	With          string   `url:"with,omitempty"`
}

// User returns a stream of messages specific to the authenticated User.
// https://dev.twitter.com/streaming/reference/get/user
func (srv *StreamService) User(params *StreamUserParams) (*Stream, error) {
	req, err := srv.user.New().Get("user.json").QueryStruct(params).Request()
	if err != nil {
		return nil, err
	}
	return newStream(srv.client, req), nil
}

// StreamSiteParams are the parameters for StreamService.Site.
type StreamSiteParams struct {
	FilterLevel   string   `url:"filter_level,omitempty"`
	Follow        []string `url:"follow,omitempty,comma"`
	Language      []string `url:"language,omitempty,comma"`
	Replies       string   `url:"replies,omitempty"`
	StallWarnings *bool    `url:"stall_warnings,omitempty"`
	With          string   `url:"with,omitempty"`
}

// Site returns messages for a set of users.
// Requires special permission to access.
// https://dev.twitter.com/streaming/reference/get/site
func (srv *StreamService) Site(params *StreamSiteParams) (*Stream, error) {
	req, err := srv.site.New().Get("site.json").QueryStruct(params).Request()
	if err != nil {
		return nil, err
	}
	return newStream(srv.client, req), nil
}

// StreamFirehoseParams are the parameters for StreamService.Firehose.
type StreamFirehoseParams struct {
	Count         int      `url:"count,omitempty"`
	FilterLevel   string   `url:"filter_level,omitempty"`
	Language      []string `url:"language,omitempty,comma"`
	StallWarnings *bool    `url:"stall_warnings,omitempty"`
}

// Firehose returns all public messages and statuses.
// Requires special permission to access.
// https://dev.twitter.com/streaming/reference/get/statuses/firehose
func (srv *StreamService) Firehose(params *StreamFirehoseParams) (*Stream, error) {
	req, err := srv.public.New().Get("firehose.json").QueryStruct(params).Request()
	if err != nil {
		return nil, err
	}
	return newStream(srv.client, req), nil
}

// Stream maintains a connection to the Twitter Streaming API, receives
// messages from the streaming response, and sends them on the Messages
// channel from a goroutine. The stream goroutine stops itself if an EOF is
// reached or retry errors occur, also closing the Messages channel.
//
// The client must Stop() the stream when finished receiving, which will
// wait until the stream is properly stopped.
type Stream struct {
	client   *http.Client
	Messages chan interface{}
	done     chan struct{}
	group    *sync.WaitGroup
	body     io.Closer
}

// newStream creates a Stream and starts a goroutine to retry connecting and
// receive from a stream response. The goroutine may stop due to retry errors
// or be stopped by calling Stop() on the stream.
func newStream(client *http.Client, req *http.Request) *Stream {
	s := &Stream{
		client:   client,
		Messages: make(chan interface{}),
		done:     make(chan struct{}),
		group:    &sync.WaitGroup{},
	}
	s.group.Add(1)
	go s.retry(req, newExponentialBackOff(), newAggressiveExponentialBackOff())
	return s
}

// Stop signals retry and receiver to stop, closes the Messages channel, and
// blocks until done.
func (s *Stream) Stop() {
	close(s.done)
	// Scanner does not have a Stop() or take a done channel, so for low volume
	// streams Scan() blocks until the next keep-alive. Close the resp.Body to
	// escape and stop the stream in a timely fashion.
	if s.body != nil {
		s.body.Close()
	}
	// block until the retry goroutine stops
	s.group.Wait()
}

// retry retries making the given http.Request and receiving the response
// according to the Twitter backoff policies. Callers should invoke in a
// goroutine since backoffs sleep between retries.
// https://dev.twitter.com/streaming/overview/connecting
func (s *Stream) retry(req *http.Request, expBackOff backoff.BackOff, aggExpBackOff backoff.BackOff) {
	// close Messages channel and decrement the wait group counter
	defer close(s.Messages)
	defer s.group.Done()

	var wait time.Duration
	for !stopped(s.done) {
		resp, err := s.client.Do(req)
		if err != nil {
			// stop retrying for HTTP protocol errors
			s.Messages <- err
			return
		}
		// when err is nil, resp contains a non-nil Body which must be closed
		defer resp.Body.Close()
		s.body = resp.Body
		switch resp.StatusCode {
		case 200:
			// receive stream response Body, handles closing
			s.receive(resp.Body)
			expBackOff.Reset()
			aggExpBackOff.Reset()
		case 503:
			// exponential backoff
			wait = expBackOff.NextBackOff()
		case 420, 429:
			// aggressive exponential backoff
			wait = aggExpBackOff.NextBackOff()
		default:
			// stop retrying for other response codes
			resp.Body.Close()
			return
		}
		// close response before each retry
		resp.Body.Close()
		if wait == backoff.Stop {
			return
		}
		sleepOrDone(wait, s.done)
	}
}

// receive scans a stream response body, JSON decodes tokens to messages, and
// sends messages to the Messages channel. Receiving continues until an EOF,
// scan error, or the done channel is closed.
func (s *Stream) receive(body io.Reader) {
	reader := newStreamResponseBodyReader(body)
	for !stopped(s.done) {
		data, err := reader.readNext()
		if err != nil {
			return
		}
		if len(data) == 0 {
			// empty keep-alive
			continue
		}
		select {
		// send messages, data, or errors
		case s.Messages <- getMessage(data):
			continue
		// allow client to Stop(), even if not receiving
		case <-s.done:
			return
		}
	}
}

// getMessage unmarshals the token and returns a message struct, if the type
// can be determined. Otherwise, returns the token unmarshalled into a data
// map[string]interface{} or the unmarshal error.
func getMessage(token []byte) interface{} {
	var data map[string]interface{}
	// unmarshal JSON encoded token into a map for
	err := json.Unmarshal(token, &data)
	if err != nil {
		return err
	}
	return decodeMessage(token, data)
}

// decodeMessage determines the message type from known data keys, allocates
// at most one message struct, and JSON decodes the token into the message.
// Returns the message struct or the data map if the message type could not be
// determined.
func decodeMessage(token []byte, data map[string]interface{}) interface{} {
	if hasPath(data, "retweet_count") {
		tweet := new(Tweet)
		json.Unmarshal(token, tweet)
		return tweet
	} else if hasPath(data, "direct_message") {
		notice := new(directMessageNotice)
		json.Unmarshal(token, notice)
		return notice.DirectMessage
	} else if hasPath(data, "delete") {
		notice := new(statusDeletionNotice)
		json.Unmarshal(token, notice)
		return notice.Delete.StatusDeletion
	} else if hasPath(data, "scrub_geo") {
		notice := new(locationDeletionNotice)
		json.Unmarshal(token, notice)
		return notice.ScrubGeo
	} else if hasPath(data, "limit") {
		notice := new(streamLimitNotice)
		json.Unmarshal(token, notice)
		return notice.Limit
	} else if hasPath(data, "status_withheld") {
		notice := new(statusWithheldNotice)
		json.Unmarshal(token, notice)
		return notice.StatusWithheld
	} else if hasPath(data, "user_withheld") {
		notice := new(userWithheldNotice)
		json.Unmarshal(token, notice)
		return notice.UserWithheld
	} else if hasPath(data, "disconnect") {
		notice := new(streamDisconnectNotice)
		json.Unmarshal(token, notice)
		return notice.StreamDisconnect
	} else if hasPath(data, "warning") {
		notice := new(stallWarningNotice)
		json.Unmarshal(token, notice)
		return notice.StallWarning
	} else if hasPath(data, "friends") {
		friendsList := new(FriendsList)
		json.Unmarshal(token, friendsList)
		return friendsList
	} else if hasPath(data, "event") {
		event := new(Event)
		json.Unmarshal(token, event)
		return event
	}
	// message type unknown, return the data map[string]interface{}
	return data
}

// hasPath returns true if the map contains the given key, false otherwise.
func hasPath(data map[string]interface{}, key string) bool {
	_, ok := data[key]
	return ok
}
