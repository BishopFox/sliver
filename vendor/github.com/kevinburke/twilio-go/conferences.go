package twilio

import (
	"context"
	"fmt"
	"net/url"
	"time"

	types "github.com/kevinburke/go-types"
)

const conferencePathPart = "Conferences"

type ConferenceService struct {
	client *Client
}

type Conference struct {
	Sid string `json:"sid"`
	// Call status, StatusInProgress or StatusCompleted
	Status       Status `json:"status"`
	FriendlyName string `json:"friendly_name"`
	// The conference region, probably "us1"
	Region                  string            `json:"region"`
	DateCreated             TwilioTime        `json:"date_created"`
	AccountSid              string            `json:"account_sid"`
	APIVersion              string            `json:"api_version"`
	DateUpdated             TwilioTime        `json:"date_updated"`
	URI                     string            `json:"uri"`
	SubresourceURIs         map[string]string `json:"subresource_uris"`
	CallSidEndingConference types.NullString  `json:"call_sid_ending_conference"`
}

type ConferencePage struct {
	Page
	Conferences []*Conference
}

func (c *ConferenceService) Get(ctx context.Context, sid string) (*Conference, error) {
	conference := new(Conference)
	err := c.client.GetResource(ctx, conferencePathPart, sid, conference)
	return conference, err
}

func (c *ConferenceService) GetPage(ctx context.Context, data url.Values) (*ConferencePage, error) {
	return c.GetPageIterator(data).Next(ctx)
}

// GetConferencesInRange gets an Iterator containing conferences in the range
// [start, end), optionally further filtered by data. GetConferencesInRange
// panics if start is not before end. Any date filters provided in data will
// be ignored. If you have an end, but don't want to specify a start, use
// twilio.Epoch for start. If you have a start, but don't want to specify an
// end, use twilio.HeatDeath for end.
//
// Assumes that Twilio returns resources in chronological order, latest
// first. If this assumption is incorrect, your results will not be correct.
//
// Returned ConferencePages will have at most PageSize results, but may have fewer,
// based on filtering.
func (c *ConferenceService) GetConferencesInRange(start time.Time, end time.Time, data url.Values) ConferencePageIterator {
	if start.After(end) {
		panic("start date is after end date")
	}
	d := url.Values{}
	for k, v := range data {
		d[k] = v
	}
	d.Del("DateCreated")
	d.Del("Page") // just in case
	if start != Epoch {
		startFormat := start.UTC().Format(APISearchLayout)
		d.Set("DateCreated>", startFormat)
	}
	if end != HeatDeath {
		// If you specify "StartTime<=YYYY-MM-DD", the *latest* result returned
		// will be midnight (the earliest possible second) on DD. We want all
		// of the results for DD so we need to specify DD+1 in the API.
		//
		// TODO validate midnight-instant math more closely, since I don't think
		// Twilio returns the correct results for that instant.
		endFormat := end.UTC().Add(24 * time.Hour).Format(APISearchLayout)
		d.Set("DateCreated<", endFormat)
	}
	iter := NewPageIterator(c.client, d, conferencePathPart)
	return &conferenceDateIterator{
		start: start,
		end:   end,
		p:     iter,
	}
}

// GetNextConferencesInRange retrieves the page at the nextPageURI and continues
// retrieving pages until any results are found in the range given by start or
// end, or we determine there are no more records to be found in that range.
//
// If ConferencePage is non-nil, it will have at least one result.
func (c *ConferenceService) GetNextConferencesInRange(start time.Time, end time.Time, nextPageURI string) ConferencePageIterator {
	if nextPageURI == "" {
		panic("nextpageuri is empty")
	}
	iter := NewNextPageIterator(c.client, conferencePathPart)
	iter.SetNextPageURI(types.NullString{Valid: true, String: nextPageURI})
	return &conferenceDateIterator{
		start: start,
		end:   end,
		p:     iter,
	}
}

type conferenceDateIterator struct {
	p     *PageIterator
	start time.Time
	end   time.Time
}

// Next returns the next page of resources. We may need to fetch multiple
// pages from the Twilio API before we find one in the right date range, so
// latency may be higher than usual. If page is non-nil, it contains at least
// one result.
func (c *conferenceDateIterator) Next(ctx context.Context) (*ConferencePage, error) {
	var page *ConferencePage
	for {
		// just wipe it clean every time to avoid remnants hanging around
		page = new(ConferencePage)
		if err := c.p.Next(ctx, page); err != nil {
			return nil, err
		}
		if len(page.Conferences) == 0 {
			return nil, NoMoreResults
		}
		times := make([]time.Time, len(page.Conferences))
		for i, conference := range page.Conferences {
			if !conference.DateCreated.Valid {
				// we really should not ever hit this case but if we can't parse
				// a date, better to give you back an error than to give you back
				// a list of conferences that may or may not be in the time range
				return nil, fmt.Errorf("twilio: couldn't verify the date of conference: %#v", conference)
			}
			times[i] = conference.DateCreated.Time
		}
		if containsResultsInRange(c.start, c.end, times) {
			indexesToDelete := indexesOutsideRange(c.start, c.end, times)
			// reverse order so we don't delete the wrong index
			for i := len(indexesToDelete) - 1; i >= 0; i-- {
				index := indexesToDelete[i]
				page.Conferences = append(page.Conferences[:index], page.Conferences[index+1:]...)
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

type ConferencePageIterator interface {
	// Next returns the next page of resources. If there are no more resources,
	// NoMoreResults is returned.
	Next(context.Context) (*ConferencePage, error)
}

// ConferencePageIterator lets you retrieve consecutive ConferencePages.
type conferencePageIterator struct {
	p *PageIterator
}

// GetPageIterator returns a ConferencePageIterator with the given page
// filters. Call iterator.Next() to get the first page of resources (and again
// to retrieve subsequent pages).
func (c *ConferenceService) GetPageIterator(data url.Values) ConferencePageIterator {
	return &conferencePageIterator{
		p: NewPageIterator(c.client, data, conferencePathPart),
	}
}

// Next returns the next page of resources. If there are no more resources,
// NoMoreResults is returned.
func (c *conferencePageIterator) Next(ctx context.Context) (*ConferencePage, error) {
	cp := new(ConferencePage)
	err := c.p.Next(ctx, cp)
	if err != nil {
		return nil, err
	}
	c.p.SetNextPageURI(cp.NextPageURI)
	return cp, nil
}
