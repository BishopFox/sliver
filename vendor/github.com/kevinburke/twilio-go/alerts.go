package twilio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	types "github.com/kevinburke/go-types"
)

const alertPathPart = "Alerts"

type AlertService struct {
	client *Client
}

// Alert represents a single Twilio Alert.
type Alert struct {
	Sid        string `json:"sid"`
	AccountSid string `json:"account_sid"`
	// For Calls, AlertText is a series of key=value pairs separated by
	// ampersands
	AlertText        string          `json:"alert_text"`
	APIVersion       string          `json:"api_version"`
	DateCreated      TwilioTime      `json:"date_created"`
	DateGenerated    TwilioTime      `json:"date_generated"`
	DateUpdated      TwilioTime      `json:"date_updated"`
	ErrorCode        Code            `json:"error_code"`
	LogLevel         LogLevel        `json:"log_level"`
	MoreInfo         string          `json:"more_info"`
	RequestMethod    string          `json:"request_method"`
	RequestURL       string          `json:"request_url"`
	RequestVariables Values          `json:"request_variables"`
	ResponseBody     string          `json:"response_body"`
	ResponseHeaders  Values          `json:"response_headers"`
	ResourceSid      string          `json:"resource_sid"`
	ServiceSid       json.RawMessage `json:"service_sid"`
	URL              string          `json:"url"`
}

// AlertPage represents a page of Alerts.
type AlertPage struct {
	Meta   Meta     `json:"meta"`
	Alerts []*Alert `json:"alerts"`
}

// Get finds a single Alert resource by its sid, or returns an error.
func (a *AlertService) Get(ctx context.Context, sid string) (*Alert, error) {
	alert := new(Alert)
	err := a.client.GetResource(ctx, alertPathPart, sid, alert)
	return alert, err
}

// GetPage returns a single Page of resources, filtered by data.
//
// See https://www.twilio.com/docs/api/monitor/alerts#list-get-filters.
func (a *AlertService) GetPage(ctx context.Context, data url.Values) (*AlertPage, error) {
	return a.GetPageIterator(data).Next(ctx)
}

// AlertPageIterator lets you retrieve consecutive pages of resources.
type AlertPageIterator interface {
	// Next returns the next page of resources. If there are no more resources,
	// NoMoreResults is returned.
	Next(context.Context) (*AlertPage, error)
}

type alertPageIterator struct {
	p *PageIterator
}

// GetAlertsInRange gets an Iterator containing conferences in the range
// [start, end), optionally further filtered by data. GetAlertsInRange
// panics if start is not before end. Any date filters provided in data will
// be ignored. If you have an end, but don't want to specify a start, use
// twilio.Epoch for start. If you have a start, but don't want to specify an
// end, use twilio.HeatDeath for end.
//
// Assumes that Twilio returns resources in chronological order, latest
// first. If this assumption is incorrect, your results will not be correct.
//
// Returned AlertPages will have at most PageSize results, but may have fewer,
// based on filtering.
func (a *AlertService) GetAlertsInRange(start time.Time, end time.Time, data url.Values) AlertPageIterator {
	if start.After(end) {
		panic("start date is after end date")
	}
	d := url.Values{}
	for k, v := range data {
		d[k] = v
	}
	d.Del("Page") // just in case
	if start != Epoch {
		startFormat := start.UTC().Format(time.RFC3339)
		d.Set("StartDate", startFormat)
	}
	if end != HeatDeath {
		// If you specify "StartTime<=YYYY-MM-DD", the *latest* result returned
		// will be midnight (the earliest possible second) on DD. We want all
		// of the results for DD so we need to specify DD+1 in the API.
		//
		// TODO validate midnight-instant math more closely, since I don't think
		// Twilio returns the correct results for that instant.
		endFormat := end.UTC().Format(time.RFC3339)
		d.Set("EndDate", endFormat)
	}
	iter := NewPageIterator(a.client, d, alertPathPart)
	return &alertDateIterator{
		start: start,
		end:   end,
		p:     iter,
	}
}

// GetNextAlertsInRange retrieves the page at the nextPageURI and continues
// retrieving pages until any results are found in the range given by start or
// end, or we determine there are no more records to be found in that range.
//
// If AlertPage is non-nil, it will have at least one result.
func (a *AlertService) GetNextAlertsInRange(start time.Time, end time.Time, nextPageURI string) AlertPageIterator {
	if nextPageURI == "" {
		panic("nextpageuri is empty")
	}
	iter := NewNextPageIterator(a.client, callsPathPart)
	iter.SetNextPageURI(types.NullString{Valid: true, String: nextPageURI})
	return &alertDateIterator{
		start: start,
		end:   end,
		p:     iter,
	}
}

type alertDateIterator struct {
	p     *PageIterator
	start time.Time
	end   time.Time
}

// Next returns the next page of resources. We may need to fetch multiple
// pages from the Twilio API before we find one in the right date range, so
// latency may be higher than usual. If page is non-nil, it contains at least
// one result.
func (a *alertDateIterator) Next(ctx context.Context) (*AlertPage, error) {
	var page *AlertPage
	for {
		// just wipe it clean every time to avoid remnants hanging around
		page = new(AlertPage)
		if err := a.p.Next(ctx, page); err != nil {
			return nil, err
		}
		if len(page.Alerts) == 0 {
			return nil, NoMoreResults
		}
		times := make([]time.Time, len(page.Alerts))
		for i, alert := range page.Alerts {
			if !alert.DateCreated.Valid {
				// we really should not ever hit this case but if we can't parse
				// a date, better to give you back an error than to give you back
				// a list of alerts that may or may not be in the time range
				return nil, fmt.Errorf("twilio: couldn't verify the date of alert: %#v", alert)
			}
			times[i] = alert.DateCreated.Time
		}
		if containsResultsInRange(a.start, a.end, times) {
			indexesToDelete := indexesOutsideRange(a.start, a.end, times)
			// reverse order so we don't delete the wrong index
			for i := len(indexesToDelete) - 1; i >= 0; i-- {
				index := indexesToDelete[i]
				page.Alerts = append(page.Alerts[:index], page.Alerts[index+1:]...)
			}
			a.p.SetNextPageURI(page.Meta.NextPageURL)
			return page, nil
		}
		if shouldContinuePaging(a.start, times) {
			a.p.SetNextPageURI(page.Meta.NextPageURL)
			continue
		} else {
			// should not continue paging and no results in range, stop
			return nil, NoMoreResults
		}
	}
}

// GetPageIterator returns a AlertPageIterator with the given page
// filters. Call iterator.Next() to get the first page of resources (and again
// to retrieve subsequent pages).
func (a *AlertService) GetPageIterator(data url.Values) AlertPageIterator {
	iter := NewPageIterator(a.client, data, alertPathPart)
	return &alertPageIterator{
		p: iter,
	}
}

// Next returns the next page of resources. If there are no more resources,
// NoMoreResults is returned.
func (a *alertPageIterator) Next(ctx context.Context) (*AlertPage, error) {
	ap := new(AlertPage)
	err := a.p.Next(ctx, ap)
	if err != nil {
		return nil, err
	}
	a.p.SetNextPageURI(ap.Meta.NextPageURL)
	return ap, nil
}

func (a *Alert) description() string {
	vals, err := url.ParseQuery(a.AlertText)
	if err == nil && a.ErrorCode != 0 {
		switch a.ErrorCode {
		case CodeHTTPRetrievalFailure:
			s := "HTTP retrieval failure"
			if resp := vals.Get("httpResponse"); resp != "" {
				s = fmt.Sprintf("%s: status code %s when fetching TwiML", s, resp)
			}
			return s
		case CodeReplyLimitExceeded:
			msg := vals.Get("Msg")
			if msg == "" {
				break
			}
			if idx := strings.Index(msg, "over"); idx >= 0 {
				return msg[:idx]
			}
			return msg
		case CodeDocumentParseFailure:
			// There's a more detailed error message here but it doesn't really
			// make sense in a sentence context: "Error on line 18 of document:
			// Content is not allowed in trailing section."
			return "Document parse failure"
		case CodeSayInvalidText:
			return "The text of the Say verb was empty or un-parsable"
		case CodeForbiddenPhoneNumber, CodeNoInternationalAuthorization:
			if vals.Get("Msg") != "" && vals.Get("phonenumber") != "" {
				return strings.TrimSpace(vals.Get("Msg")) + " " + vals.Get("phonenumber")
			}
		default:
			if msg := vals.Get("Msg"); msg != "" {
				return msg
			}
			if a.MoreInfo != "" {
				return fmt.Sprintf("Error %d: %s", a.ErrorCode, a.MoreInfo)
			}
			return fmt.Sprintf("Error %d", a.ErrorCode)
		}
	}
	if a.MoreInfo != "" {
		return "Unknown failure: " + a.MoreInfo
	}
	return "Unknown failure"
}

// Description tries as hard as possible to give you a one sentence description
// of this Alert, based on its contents. Description does not include a
// trailing period.
func (a *Alert) Description() string {
	return capitalize(strings.TrimSpace(strings.TrimSuffix(a.description(), ".")))
}

// StatusCode attempts to return a HTTP status code for this Alert. Returns
// 0 if the status code cannot be found.
func (a *Alert) StatusCode() int {
	vals, err := url.ParseQuery(a.AlertText)
	if err != nil {
		return 0
	}
	if code := vals.Get("httpResponse"); code != "" {
		i, err := strconv.ParseInt(code, 10, 64)
		if err == nil && i > 99 && i < 600 {
			return int(i)
		}
	}
	return 0
}
