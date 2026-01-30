package twilio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	types "github.com/kevinburke/go-types"
)

const callsPathPart = "Calls"

type CallService struct {
	client *Client
}

type Call struct {
	Sid       string         `json:"sid"`
	From      PhoneNumber    `json:"from"`
	To        PhoneNumber    `json:"to"`
	Status    Status         `json:"status"`
	StartTime TwilioTime     `json:"start_time"`
	EndTime   TwilioTime     `json:"end_time"`
	Duration  TwilioDuration `json:"duration"`
	// The wait time in milliseconds before the call is placed.
	QueueTime      TwilioDurationMS `json:"queue_time"`
	AccountSid     string           `json:"account_sid"`
	Annotation     json.RawMessage  `json:"annotation"`
	AnsweredBy     NullAnsweredBy   `json:"answered_by"`
	CallerName     types.NullString `json:"caller_name"`
	DateCreated    TwilioTime       `json:"date_created"`
	DateUpdated    TwilioTime       `json:"date_updated"`
	Direction      Direction        `json:"direction"`
	ForwardedFrom  PhoneNumber      `json:"forwarded_from"`
	GroupSid       string           `json:"group_sid"`
	ParentCallSid  string           `json:"parent_call_sid"`
	PhoneNumberSid string           `json:"phone_number_sid"`
	Price          string           `json:"price"`
	PriceUnit      string           `json:"price_unit"`
	APIVersion     string           `json:"api_version"`
	URI            string           `json:"uri"`
}

// Ended returns true if the Call has reached a terminal state, and false
// otherwise, or if the state can't be determined.
func (c *Call) Ended() bool {
	// https://www.twilio.com/docs/api/rest/call#call-status-values
	switch c.Status {
	case StatusCompleted, StatusCanceled, StatusFailed, StatusBusy, StatusNoAnswer:
		return true
	default:
		return false
	}
}

// EndedUnsuccessfully returns true if the Call has reached a terminal state
// and that state isn't "completed".
func (c *Call) EndedUnsuccessfully() bool {
	// https://www.twilio.com/docs/api/rest/call#call-status-values
	switch c.Status {
	case StatusCanceled, StatusFailed, StatusBusy, StatusNoAnswer:
		return true
	default:
		return false
	}
}

// FriendlyPrice flips the sign of the Price (which is usually reported from
// the API as a negative number) and adds an appropriate currency symbol in
// front of it. For example, a PriceUnit of "USD" and a Price of "-1.25" is
// reported as "$1.25".
func (c *Call) FriendlyPrice() string {
	if c == nil {
		return ""
	}
	return price(c.PriceUnit, c.Price)
}

// A CallPage contains a Page of calls.
type CallPage struct {
	Page
	Calls []*Call `json:"calls"`
}

func (c *CallService) Get(ctx context.Context, sid string) (*Call, error) {
	call := new(Call)
	err := c.client.GetResource(ctx, callsPathPart, sid, call)
	return call, err
}

// Update the call with the given data. Valid parameters may be found here:
// https://www.twilio.com/docs/api/rest/change-call-state#post-parameters
func (c *CallService) Update(ctx context.Context, sid string, data url.Values) (*Call, error) {
	call := new(Call)
	err := c.client.UpdateResource(ctx, callsPathPart, sid, data, call)
	return call, err
}

// Cancel an in-progress Call with the given sid. Cancel will not affect
// in-progress Calls, only those in queued or ringing.
func (c *CallService) Cancel(sid string) (*Call, error) {
	data := url.Values{}
	data.Set("Status", string(StatusCanceled))
	return c.Update(context.Background(), sid, data)
}

// Hang up an in-progress call.
func (c *CallService) Hangup(sid string) (*Call, error) {
	data := url.Values{}
	data.Set("Status", string(StatusCompleted))
	return c.Update(context.Background(), sid, data)
}

// Redirect the given call to the given URL.
func (c *CallService) Redirect(sid string, u *url.URL) (*Call, error) {
	data := url.Values{}
	data.Set("Url", u.String())
	return c.Update(context.Background(), sid, data)
}

// Initiate a new Call.
func (c *CallService) Create(ctx context.Context, data url.Values) (*Call, error) {
	call := new(Call)
	err := c.client.CreateResource(ctx, callsPathPart, data, call)
	return call, err
}

// MakeCall starts a new Call from the given phone number to the given phone
// number, dialing the url when the call connects. MakeCall is a wrapper around
// Create; if you need more configuration, call that function directly.
func (c *CallService) MakeCall(from string, to string, u *url.URL) (*Call, error) {
	data := url.Values{}
	data.Set("From", from)
	data.Set("To", to)
	data.Set("Url", u.String())
	return c.Create(context.Background(), data)
}

func (c *CallService) GetPage(ctx context.Context, data url.Values) (*CallPage, error) {
	iter := c.GetPageIterator(data)
	return iter.Next(ctx)
}

// GetCallsInRange gets an Iterator containing calls in the range [start, end),
// optionally further filtered by data. GetCallsInRange panics if start is not
// before end. Any date filters provided in data will be ignored. If you have
// an end, but don't want to specify a start, use twilio.Epoch for start. If
// you have a start, but don't want to specify an end, use twilio.HeatDeath for
// end.
//
// Assumes that Twilio returns resources in chronological order, latest
// first. If this assumption is incorrect, your results will not be correct.
//
// Returned CallPages will have at most PageSize results, but may have fewer,
// based on filtering.
func (c *CallService) GetCallsInRange(start time.Time, end time.Time, data url.Values) CallPageIterator {
	if start.After(end) {
		panic("start date is after end date")
	}
	d := url.Values{}
	for k, v := range data {
		d[k] = v
	}
	d.Del("StartTime")
	d.Del("Page") // just in case
	if start != Epoch {
		startFormat := start.UTC().Format(APISearchLayout)
		d.Set("StartTime>", startFormat)
	}
	if end != HeatDeath {
		// If you specify "StartTime<=YYYY-MM-DD", the *latest* result returned
		// will be midnight (the earliest possible second) on DD. We want all
		// of the results for DD so we need to specify DD+1 in the API.
		//
		// TODO validate midnight-instant math more closely, since I don't think
		// Twilio returns the correct results for that instant.
		endFormat := end.UTC().Add(24 * time.Hour).Format(APISearchLayout)
		d.Set("StartTime<", endFormat)
	}
	iter := NewPageIterator(c.client, d, callsPathPart)
	return &callDateIterator{
		start: start,
		end:   end,
		p:     iter,
	}
}

// GetNextCallsInRange retrieves the page at the nextPageURI and continues
// retrieving pages until any results are found in the range given by start or
// end, or we determine there are no more records to be found in that range.
//
// If CallPage is non-nil, it will have at least one result.
func (c *CallService) GetNextCallsInRange(start time.Time, end time.Time, nextPageURI string) CallPageIterator {
	if nextPageURI == "" {
		panic("nextpageuri is empty")
	}
	iter := NewNextPageIterator(c.client, callsPathPart)
	iter.SetNextPageURI(types.NullString{Valid: true, String: nextPageURI})
	return &callDateIterator{
		start: start,
		end:   end,
		p:     iter,
	}
}

type callDateIterator struct {
	p     *PageIterator
	start time.Time
	end   time.Time
}

// Next returns the next page of resources. We may need to fetch multiple
// pages from the Twilio API before we find one in the right date range, so
// latency may be higher than usual. If page is non-nil, it contains at least
// one result.
func (c *callDateIterator) Next(ctx context.Context) (*CallPage, error) {
	var page *CallPage
	for {
		// just wipe it clean every time to avoid remnants hanging around
		page = new(CallPage)
		if err := c.p.Next(ctx, page); err != nil {
			return nil, err
		}
		if len(page.Calls) == 0 {
			return nil, NoMoreResults
		}
		times := make([]time.Time, len(page.Calls))
		for i, call := range page.Calls {
			if !call.DateCreated.Valid {
				// we really should not ever hit this case but if we can't parse
				// a date, better to give you back an error than to give you back
				// a list of calls that may or may not be in the time range
				return nil, fmt.Errorf("twilio: couldn't verify the date of call: %#v", call)
			}
			// not ideal but the start time field is not guaranteed to be
			// populated.
			if call.StartTime.Valid {
				times[i] = call.StartTime.Time
			} else {
				times[i] = call.DateCreated.Time
			}
		}
		if containsResultsInRange(c.start, c.end, times) {
			indexesToDelete := indexesOutsideRange(c.start, c.end, times)
			// iterate in descending order so we don't delete the wrong index
			for i := len(indexesToDelete) - 1; i >= 0; i-- {
				index := indexesToDelete[i]
				page.Calls = append(page.Calls[:index], page.Calls[index+1:]...)
			}
			c.p.SetNextPageURI(page.NextPageURI)
			return page, nil
		}
		if shouldContinuePaging(c.start, times) {
			c.p.SetNextPageURI(page.NextPageURI)
			continue
		} else {
			// should not continue paging and no results in range, stop
			return nil, NoMoreResults
		}
	}
}

// CallPageIterator lets you retrieve consecutive pages of resources.
type CallPageIterator interface {
	// Next returns the next page of resources. If there are no more resources,
	// NoMoreResults is returned.
	Next(context.Context) (*CallPage, error)
}

type callPageIterator struct {
	p *PageIterator
}

// GetPageIterator returns an iterator which can be used to retrieve pages.
func (c *CallService) GetPageIterator(data url.Values) CallPageIterator {
	iter := NewPageIterator(c.client, data, callsPathPart)
	return &callPageIterator{
		p: iter,
	}
}

// Next returns the next page of resources. If there are no more resources,
// NoMoreResults is returned.
func (c *callPageIterator) Next(ctx context.Context) (*CallPage, error) {
	cp := new(CallPage)
	err := c.p.Next(ctx, cp)
	if err != nil {
		return nil, err
	}
	c.p.SetNextPageURI(cp.NextPageURI)
	return cp, nil
}

// GetRecordings returns an array of recordings for this Call. Note there may
// be more than one Page of results.
func (c *CallService) GetRecordings(ctx context.Context, callSid string, data url.Values) (*RecordingPage, error) {
	if data == nil {
		data = url.Values{}
	}
	// Cheat - hit the Recordings list view with a filter instead of
	// GET /calls/CA123/Recordings. The former is probably more reliable
	data.Set("CallSid", callSid)
	return c.client.Recordings.GetPage(ctx, data)
}

// GetRecordings returns an iterator of recording pages for this Call.
// Note there may be more than one Page of results.
func (c *CallService) GetRecordingsIterator(callSid string, data url.Values) *RecordingPageIterator {
	if data == nil {
		data = url.Values{}
	}
	// Cheat - hit the Recordings list view with a filter instead of
	// GET /calls/CA123/Recordings. The former is probably more reliable
	data.Set("CallSid", callSid)
	return c.client.Recordings.GetPageIterator(data)
}
