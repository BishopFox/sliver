package twilio

import (
	"context"
	"fmt"
	"net/url"
	"time"

	types "github.com/kevinburke/go-types"
	"golang.org/x/sync/errgroup"
)

const messagesPathPart = "Messages"

type MessageService struct {
	client *Client
}

// The direction of the message.
type Direction string

// Friendly prints out a friendly version of the Direction, following the
// example shown in the Twilio Dashboard.
func (d Direction) Friendly() string {
	switch d {
	case DirectionOutboundReply:
		return "Reply"
	case DirectionOutboundCall:
		return "Outgoing (from call)"
	case DirectionOutboundAPI:
		return "Outgoing (from API)"
	case DirectionInbound:
		return "Incoming"
	case DirectionOutboundDial:
		return "Outgoing (via Dial)"
	case DirectionTrunkingTerminating:
		return "Trunking (terminating)"
	case DirectionTrunkingOriginating:
		return "Trunking (originating)"
	default:
		return string(d)
	}
}

const DirectionOutboundReply = Direction("outbound-reply")
const DirectionInbound = Direction("inbound")
const DirectionOutboundCall = Direction("outbound-call")
const DirectionOutboundAPI = Direction("outbound-api")
const DirectionOutboundDial = Direction("outbound-dial")
const DirectionTrunkingTerminating = Direction("trunking-terminating")
const DirectionTrunkingOriginating = Direction("trunking-originating")

type Message struct {
	Sid                 string            `json:"sid"`
	Body                string            `json:"body"`
	From                PhoneNumber       `json:"from"`
	To                  PhoneNumber       `json:"to"`
	Price               string            `json:"price"`
	Status              Status            `json:"status"`
	AccountSid          string            `json:"account_sid"`
	MessagingServiceSid types.NullString  `json:"messaging_service_sid"`
	DateCreated         TwilioTime        `json:"date_created"`
	DateUpdated         TwilioTime        `json:"date_updated"`
	DateSent            TwilioTime        `json:"date_sent"`
	NumSegments         Segments          `json:"num_segments"`
	NumMedia            NumMedia          `json:"num_media"`
	PriceUnit           string            `json:"price_unit"`
	Direction           Direction         `json:"direction"`
	SubresourceURIs     map[string]string `json:"subresource_uris"`
	URI                 string            `json:"uri"`
	APIVersion          string            `json:"api_version"`
	ErrorCode           Code              `json:"error_code"`
	ErrorMessage        string            `json:"error_message"`
}

// FriendlyPrice flips the sign of the Price (which is usually reported from
// the API as a negative number) and adds an appropriate currency symbol in
// front of it. For example, a PriceUnit of "USD" and a Price of "-1.25" is
// reported as "$1.25".
func (m *Message) FriendlyPrice() string {
	return price(m.PriceUnit, m.Price)
}

// A MessagePage contains a Page of messages.
type MessagePage struct {
	Page
	Messages []*Message `json:"messages"`
}

// Create a message with the given url.Values. For more information on valid
// values, see https://www.twilio.com/docs/api/rest/sending-messages or use the
// SendMessage helper.
func (m *MessageService) Create(ctx context.Context, data url.Values) (*Message, error) {
	msg := new(Message)
	err := m.client.CreateResource(ctx, messagesPathPart, data, msg)
	return msg, err
}

// SendMessage sends an outbound Message with the given body or mediaURLs.
func (m *MessageService) SendMessage(from string, to string, body string, mediaURLs []*url.URL) (*Message, error) {
	v := url.Values{}
	v.Set("Body", body)
	v.Set("From", from)
	v.Set("To", to)
	for _, mediaURL := range mediaURLs {
		v.Add("MediaUrl", mediaURL.String())
	}
	return m.Create(context.Background(), v)
}

// MessagePageIterator lets you retrieve consecutive pages of resources.
type MessagePageIterator interface {
	// Next returns the next page of resources. If there are no more resources,
	// NoMoreResults is returned.
	Next(context.Context) (*MessagePage, error)
}

type messagePageIterator struct {
	p *PageIterator
}

// Next returns the next page of resources. If there are no more resources,
// NoMoreResults is returned.
func (m *messagePageIterator) Next(ctx context.Context) (*MessagePage, error) {
	mp := new(MessagePage)
	err := m.p.Next(ctx, mp)
	if err != nil {
		return nil, err
	}
	m.p.SetNextPageURI(mp.NextPageURI)
	return mp, nil
}

// GetPageIterator returns an iterator which can be used to retrieve pages.
func (m *MessageService) GetPageIterator(data url.Values) MessagePageIterator {
	iter := NewPageIterator(m.client, data, messagesPathPart)
	return &messagePageIterator{
		p: iter,
	}
}

func (m *MessageService) Get(ctx context.Context, sid string) (*Message, error) {
	msg := new(Message)
	err := m.client.GetResource(ctx, messagesPathPart, sid, msg)
	return msg, err
}

// GetPage returns a single page of resources. To retrieve multiple pages, use
// GetPageIterator.
func (m *MessageService) GetPage(ctx context.Context, data url.Values) (*MessagePage, error) {
	iter := m.GetPageIterator(data)
	return iter.Next(ctx)
}

// Delete the Message with the given sid. If the Message has already been
// deleted, or does not exist, Delete returns nil. If another error or a
// timeout occurs, the error is returned.
func (m *MessageService) Delete(ctx context.Context, sid string) error {
	return m.client.DeleteResource(ctx, messagesPathPart, sid)
}

// GetMessagesInRange gets an Iterator containing calls in the range [start,
// end), optionally further filtered by data. GetMessagesInRange panics if
// start is not before end. Any date filters provided in data will be ignored.
// If you have an end, but don't want to specify a start, use twilio.Epoch for
// start. If you have a start, but don't want to specify an end, use
// twilio.HeatDeath for end.
//
// Assumes that Twilio returns resources in chronological order, latest
// first. If this assumption is incorrect, your results will not be correct.
//
// Returned MessagePages will have at most PageSize results, but may have
// fewer, based on filtering.
func (c *MessageService) GetMessagesInRange(start time.Time, end time.Time, data url.Values) MessagePageIterator {
	if start.After(end) {
		panic("start date is after end date")
	}
	d := url.Values{}
	for k, v := range data {
		d[k] = v
	}
	d.Del("DateSent")
	d.Del("Page") // just in case
	// Omit these parameters if they are the sentinel values, since I think
	// that API paging will be faster.
	if start != Epoch {
		startFormat := start.UTC().Format(APISearchLayout)
		d.Set("DateSent>", startFormat)
	}
	if end != HeatDeath {
		// If you specify "DateSent<=YYYY-MM-DD", the *latest* result returned
		// will be midnight (the earliest possible second) on DD. We want all of
		// the results for DD so we need to specify DD+1 in the API.
		//
		// TODO validate midnight-instant math more closely, since I don't think
		// Twilio returns the correct results for that instant.
		endFormat := end.UTC().Add(24 * time.Hour).Format(APISearchLayout)
		d.Set("DateSent<", endFormat)
	}
	iter := NewPageIterator(c.client, d, messagesPathPart)
	return &messageDateIterator{
		start: start,
		end:   end,
		p:     iter,
	}
}

// GetNextMessagesInRange retrieves the page at the nextPageURI and continues
// retrieving pages until any results are found in the range given by start or
// end, or we determine there are no more records to be found in that range.
//
// If MessagePage is non-nil, it will have at least one result.
func (c *MessageService) GetNextMessagesInRange(start time.Time, end time.Time, nextPageURI string) MessagePageIterator {
	if nextPageURI == "" {
		panic("nextpageuri is empty")
	}
	iter := NewNextPageIterator(c.client, messagesPathPart)
	iter.SetNextPageURI(types.NullString{Valid: true, String: nextPageURI})
	return &messageDateIterator{
		start: start,
		end:   end,
		p:     iter,
	}
}

type messageDateIterator struct {
	p     *PageIterator
	start time.Time
	end   time.Time
}

// Next returns the next page of resources. We may need to fetch multiple
// pages from the Twilio API before we find one in the right date range, so
// latency may be higher than usual.
func (c *messageDateIterator) Next(ctx context.Context) (*MessagePage, error) {
	var page *MessagePage
	for {
		// just wipe it clean every time to avoid remnants hanging around
		page = new(MessagePage)
		if err := c.p.Next(ctx, page); err != nil {
			return nil, err
		}
		if len(page.Messages) == 0 {
			return nil, NoMoreResults
		}
		times := make([]time.Time, len(page.Messages))
		for i, message := range page.Messages {
			if !message.DateCreated.Valid {
				// we really should not ever hit this case but if we can't parse
				// a date, better to give you back an error than to give you back
				// a list of messages that may or may not be in the time range
				return nil, fmt.Errorf("twilio: couldn't verify the date of message: %#v", message)
			}
			// this isn't ideal, but DateSent is used as the sort field if
			// present, and it is not populated for all records, so we need
			// a fallback.
			if message.DateSent.Valid {
				times[i] = message.DateSent.Time
			} else {
				times[i] = message.DateCreated.Time
			}
		}
		if containsResultsInRange(c.start, c.end, times) {
			indexesToDelete := indexesOutsideRange(c.start, c.end, times)
			// reverse order so we don't delete the wrong index
			for i := len(indexesToDelete) - 1; i >= 0; i-- {
				index := indexesToDelete[i]
				page.Messages = append(page.Messages[:index], page.Messages[index+1:]...)
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

// GetMediaURLs gets the URLs of any media for this message. This uses threads
// to retrieve all URLs simultaneously; if retrieving any URL fails, we return
// an error for the entire request.
//
// The data can be used to filter the list of returned Media as described here:
// https://www.twilio.com/docs/api/rest/media#list-get-filters
//
// As of October 2016, only 10 MediaURLs are permitted per message. No attempt
// is made to page through media resources; omit the PageSize parameter in
// data, or set it to a value greater than 10, to retrieve all resources.
func (m *MessageService) GetMediaURLs(ctx context.Context, sid string, data url.Values) ([]*url.URL, error) {
	page, err := m.client.Media.GetPage(ctx, sid, data)
	if err != nil {
		return nil, err
	}
	if len(page.MediaList) == 0 {
		urls := make([]*url.URL, 0)
		return urls, nil
	}
	urls := make([]*url.URL, len(page.MediaList))
	g, errctx := errgroup.WithContext(ctx)
	for i, media := range page.MediaList {
		i := i
		mediaSid := media.Sid
		g.Go(func() error {
			url, err := m.client.Media.GetURL(errctx, sid, mediaSid)
			if err != nil {
				return err
			}
			urls[i] = url
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return urls, nil
}
