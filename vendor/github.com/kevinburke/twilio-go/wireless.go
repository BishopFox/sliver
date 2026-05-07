package twilio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	types "github.com/kevinburke/go-types"
)

const simPathPart = "Sims"

type SimService struct {
	client *Client
}

// Sim represents a Sim resource.
type Sim struct {
	Sid          string           `json:"sid"`
	UniqueName   string           `json:"unique_name"`
	Status       Status           `json:"status"`
	FriendlyName types.NullString `json:"friendly_name"`
	ICCID        string           `json:"iccid"`

	CommandsCallbackMethod string           `json:"commands_callback_method"`
	CommandsCallbackURL    types.NullString `json:"commands_callback_url"`
	DateCreated            TwilioTime       `json:"date_created"`
	DateUpdated            TwilioTime       `json:"date_updated"`
	RatePlanSid            string           `json:"rate_plan_sid"`
	SMSURL                 types.NullString `json:"sms_url"`
	SMSMethod              types.NullString `json:"sms_method"`
	SMSFallbackMethod      types.NullString `json:"sms_fallback_method"`
	SMSFallbackURL         types.NullString `json:"sms_fallback_url"`
	VoiceURL               types.NullString `json:"voice_url"`
	VoiceMethod            types.NullString `json:"voice_method"`
	VoiceFallbackMethod    types.NullString `json:"voice_fallback_method"`
	VoiceFallbackURL       types.NullString `json:"voice_fallback_url"`

	URL        string            `json:"url"`
	AccountSid string            `json:"account_sid"`
	Links      map[string]string `json:"links"`
}

type SimUsageRecord struct {
	AccountSid string        `json:"account_sid"`
	Commands   CommandsUsage `json:"commands"`
	Data       AllDataUsage  `json:"data"`
	Period     UsagePeriod   `json:"period"`
	SimSid     string        `json:"sim_sid"`
}

type CommandsUsage struct {
	CommandUsage
	Home                 *CommandUsage   `json:"home"`
	InternationalRoaming []*CommandUsage `json:"international_roaming"`
	NationalRoaming      *CommandUsage   `json:"national_roaming"`
}

type CommandUsage struct {
	FromSim uint64 `json:"from_sim"`
	ToSim   uint64 `json:"to_sim"`
	Total   uint64 `json:"total"`
}

type AllDataUsage struct {
	// TODO: ugh, naming
	DataUsage
	Home                 *DataUsage   `json:"home"`
	InternationalRoaming []*DataUsage `json:"international_roaming"`
	NationalRoaming      *DataUsage   `json:"national_roaming"`
}

type DataUsage struct {
	Download types.Bits `json:"download"`
	Total    types.Bits `json:"total"`
	Upload   types.Bits `json:"upload"`
	Units    string     `json:"units"`
}

// for parsing from Twilio
type jsonDataUsage struct {
	Download int64  `json:"download"`
	Total    int64  `json:"total"`
	Upload   int64  `json:"upload"`
	Units    string `json:"units"`
}

type jsonAllDataUsage struct {
	Home                 *DataUsage   `json:"home"`
	InternationalRoaming []*DataUsage `json:"international_roaming"`
	NationalRoaming      *DataUsage   `json:"national_roaming"`
	Download             int64        `json:"download"`
	Total                int64        `json:"total"`
	Upload               int64        `json:"upload"`
	Units                string       `json:"units"`
}

func (d *AllDataUsage) UnmarshalJSON(data []byte) error {
	mp := new(jsonAllDataUsage)
	if err := json.Unmarshal(data, mp); err != nil {
		return err
	}
	d.Home = mp.Home
	d.InternationalRoaming = mp.InternationalRoaming
	d.NationalRoaming = mp.NationalRoaming
	if mp.Units != "bytes" {
		return fmt.Errorf("twilio: unknown units parameter %q", mp.Units)
	}
	d.Units = "bytes"
	// multiply bytes by 8 to get bits
	d.Download = types.Bits(mp.Download) * types.Byte
	d.Upload = types.Bits(mp.Upload) * types.Byte
	d.Total = types.Bits(mp.Total) * types.Byte
	return nil
}

func (d *DataUsage) UnmarshalJSON(data []byte) error {
	mp := new(jsonDataUsage)
	if err := json.Unmarshal(data, mp); err != nil {
		return err
	}
	if mp.Units != "bytes" {
		return fmt.Errorf("twilio: unknown units parameter %q", mp.Units)
	}
	d.Units = "bytes"
	// multiply by 8 to get bits
	d.Download = types.Bits(mp.Download) * types.Byte
	d.Upload = types.Bits(mp.Upload) * types.Byte
	d.Total = types.Bits(mp.Total) * types.Byte
	return nil
}

type UsagePeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type SimUsageRecordPage struct {
	Meta         Meta              `json:"meta"`
	UsageRecords []*SimUsageRecord `json:"usage_records"`
}

// SimPage represents a page of Sims.
type SimPage struct {
	Meta Meta   `json:"meta"`
	Sims []*Sim `json:"sims"`
}

// Get finds a single Sim resource by its sid, or returns an error.
func (s *SimService) Get(ctx context.Context, sid string) (*Sim, error) {
	sim := new(Sim)
	err := s.client.GetResource(ctx, simPathPart, sid, sim)
	return sim, err
}

// GetUsageRecords finds a page of UsageRecord resources.
func (s *SimService) GetUsageRecords(ctx context.Context, simSid string, data url.Values) (*SimUsageRecordPage, error) {
	return s.GetUsageRecordsIterator(simSid, data).Next(ctx)
}

func (s *SimService) GetUsageRecordsIterator(simSid string, data url.Values) SimUsageRecordPageIterator {
	// TODO this is messy
	iter := NewPageIterator(s.client, data, simPathPart+"/"+simSid+"/UsageRecords")
	return &simUsageRecordPageIterator{
		p: iter,
	}
}

type SimUsageRecordPageIterator interface {
	// Next returns the next page of resources. If there are no more resources,
	// NoMoreResults is returned.
	Next(context.Context) (*SimUsageRecordPage, error)
}

type simUsageRecordPageIterator struct {
	p *PageIterator
}

func (i *simUsageRecordPageIterator) Next(ctx context.Context) (*SimUsageRecordPage, error) {
	ap := new(SimUsageRecordPage)
	err := i.p.Next(ctx, ap)
	if err != nil {
		return nil, err
	}
	i.p.SetNextPageURI(ap.Meta.NextPageURL)
	return ap, nil
}

// Update the sim with the given data. Valid parameters may be found here:
// https://www.twilio.com/docs/api/wireless/rest-api/sim#instance-post
func (c *SimService) Update(ctx context.Context, sid string, data url.Values) (*Sim, error) {
	sim := new(Sim)
	err := c.client.UpdateResource(ctx, simPathPart, sid, data, sim)
	return sim, err
}

// SimPageIterator lets you retrieve consecutive pages of resources.
type SimPageIterator interface {
	// Next returns the next page of resources. If there are no more resources,
	// NoMoreResults is returned.
	Next(context.Context) (*SimPage, error)
}

type simPageIterator struct {
	p *PageIterator
}

// GetPage returns a single Page of resources, filtered by data.
//
// See https://www.twilio.com/docs/api/wireless/rest-api/sim#list-get.
func (f *SimService) GetPage(ctx context.Context, data url.Values) (*SimPage, error) {
	return f.GetPageIterator(data).Next(ctx)
}

// GetPageIterator returns a SimPageIterator with the given page
// filters. Call iterator.Next() to get the first page of resources (and again
// to retrieve subsequent pages).
func (f *SimService) GetPageIterator(data url.Values) SimPageIterator {
	iter := NewPageIterator(f.client, data, simPathPart)
	return &simPageIterator{
		p: iter,
	}
}

// Next returns the next page of resources. If there are no more resources,
// NoMoreResults is returned.
func (f *simPageIterator) Next(ctx context.Context) (*SimPage, error) {
	ap := new(SimPage)
	err := f.p.Next(ctx, ap)
	if err != nil {
		return nil, err
	}
	f.p.SetNextPageURI(ap.Meta.NextPageURL)
	return ap, nil
}
