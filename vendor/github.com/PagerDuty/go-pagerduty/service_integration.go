package pagerduty

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/go-querystring/query"
)

// IntegrationEmailFilterMode is a type to respresent the different filter modes
// for a Generic Email Integration. This defines how the email filter rules
// (IntegrationEmailFilterRuleMode) are used when emails are ingested.
type IntegrationEmailFilterMode uint8

const (
	// EmailFilterModeInvalid only exists to make it harder to use values of
	// this type incorrectly. Please instead use one of EmailFilterModeAll,
	// EmailFilterModeOr, EmailFilterModeAnd
	//
	// This value should not get marshaled to JSON by the encoding/json package.
	EmailFilterModeInvalid IntegrationEmailFilterMode = iota

	// EmailFilterModeAll means that all incoming email will be be accepted, and
	// no email rules will be considered.
	EmailFilterModeAll

	// EmailFilterModeOr instructs the email filtering system to accept the
	// email if one or more rules match the message.
	EmailFilterModeOr

	// EmailFilterModeAnd instructs the email filtering system to accept the
	// email only if all of the rules match the message.
	EmailFilterModeAnd
)

// string values for each IntegrationEmailFilterMode value
const (
	efmAll = "all-email"       // EmailFilterModeAll
	efmOr  = "or-rules-email"  // EmailFilterModeOr
	efmAnd = "and-rules-email" // EmailFilterModeAnd
)

func (i IntegrationEmailFilterMode) String() string {
	switch i {
	case EmailFilterModeAll:
		return efmAll

	case EmailFilterModeOr:
		return efmOr

	case EmailFilterModeAnd:
		return efmAnd

	default:
		return "invalid"
	}
}

// compile time encoding/json interface satisfaction assertions
var (
	_ json.Marshaler   = IntegrationEmailFilterMode(0)
	_ json.Unmarshaler = (*IntegrationEmailFilterMode)(nil)
)

// MarshalJSON satisfies json.Marshaler
func (i IntegrationEmailFilterMode) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", i.String())), nil
}

// UnmarshalJSON satisfies json.Unmarshaler
func (i *IntegrationEmailFilterMode) UnmarshalJSON(b []byte) error {
	if b[0] != '"' {
		if bytes.Equal(b, []byte(`null`)) {
			return errors.New("value cannot be null")
		}

		// just return json.Unmarshal error
		var s string

		err := json.Unmarshal(b, &s)
		if err == nil {
			panic("this should not be possible...")
		}

		return err
	}

	v := string(b[1 : len(b)-1])

	switch v {
	case efmAll:
		*i = EmailFilterModeAll

	case efmOr:
		*i = EmailFilterModeOr

	case efmAnd:
		*i = EmailFilterModeAnd

	default:
		return fmt.Errorf("unknown value %q", v)
	}

	return nil
}

// IntegrationEmailFilterRuleMode is a type to represent the different matching
// modes of Generic Email Integration Filer Rules without consumers of this
// package needing to be intimately familiar with the specifics of the REST API.
type IntegrationEmailFilterRuleMode uint8

const (
	// EmailFilterRuleModeInvalid only exists to make it harder to use values of this
	// type incorrectly. Please instead use one of EmailFilterRuleModeAlways,
	// EmailFilterRuleModeMatch, or EmailFilterRuleModeNoMatch.
	//
	// This value should not get marshaled to JSON by the encoding/json package.
	EmailFilterRuleModeInvalid IntegrationEmailFilterRuleMode = iota

	// EmailFilterRuleModeAlways means that the specific value can be anything. Any
	// associated regular expression will be ignored.
	EmailFilterRuleModeAlways

	// EmailFilterRuleModeMatch means that the associated regular expression must
	// match the associated value.
	EmailFilterRuleModeMatch

	// EmailFilterRuleModeNoMatch means that the associated regular expression must NOT
	// match the associated value.
	EmailFilterRuleModeNoMatch
)

// string values for each IntegrationEmailFilterRuleMode value
const (
	efrmAlways  = "always"   // EmailFilterRuleModeAlways
	efrmMatch   = "match"    // EmailFilterRuleModeMatch
	efrmNoMatch = "no-match" // EmailFilterRuleModeNoMatch
)

func (i IntegrationEmailFilterRuleMode) String() string {
	switch i {
	case EmailFilterRuleModeMatch:
		return efrmMatch

	case EmailFilterRuleModeNoMatch:
		return efrmNoMatch

	case EmailFilterRuleModeAlways:
		return efrmAlways

	default:
		return "invalid"
	}
}

// compile time encoding/json interface satisfaction assertions
var (
	_ json.Marshaler   = IntegrationEmailFilterRuleMode(0)
	_ json.Unmarshaler = (*IntegrationEmailFilterRuleMode)(nil)
)

// MarshalJSON satisfies json.Marshaler
func (i IntegrationEmailFilterRuleMode) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", i.String())), nil
}

// UnmarshalJSON satisfies json.Unmarshaler
func (i *IntegrationEmailFilterRuleMode) UnmarshalJSON(b []byte) error {
	if b[0] != '"' {
		if bytes.Equal(b, []byte(`null`)) {
			return errors.New("value cannot be null")
		}

		// just return json.Unmarshal error
		var s string

		err := json.Unmarshal(b, &s)
		if err == nil {
			panic("this should not be possible...")
		}

		return err
	}

	v := string(b[1 : len(b)-1])

	switch v {
	case efrmMatch:
		*i = EmailFilterRuleModeMatch

	case efrmNoMatch:
		*i = EmailFilterRuleModeNoMatch

	case efrmAlways:
		*i = EmailFilterRuleModeAlways

	default:
		return fmt.Errorf("unknown value %q", v)
	}

	return nil
}

// IntegrationEmailFilterRule represents a single email filter rule for an
// integration of type generic_email_inbound_integration. Information about how
// to configure email rules can be found here:
// https://support.pagerduty.com/docs/email-management-filters-and-rules.
type IntegrationEmailFilterRule struct {
	// SubjectMode and SubjectRegex control the behaviors of how this filter
	// matches the subject of an inbound email.
	SubjectMode  IntegrationEmailFilterRuleMode `json:"subject_mode,omitempty"`
	SubjectRegex *string                        `json:"subject_regex,omitempty"`

	// BodyMode and BodyRegex control the behaviors of how this filter matches
	// the body of an inbound email.
	BodyMode  IntegrationEmailFilterRuleMode `json:"body_mode,omitempty"`
	BodyRegex *string                        `json:"body_regex,omitempty"`

	FromEmailMode  IntegrationEmailFilterRuleMode `json:"from_email_mode,omitempty"`
	FromEmailRegex *string                        `json:"from_email_regex,omitempty"`
}

// UnmarshalJSON satisfies json.Unmarshaler.
func (i *IntegrationEmailFilterRule) UnmarshalJSON(b []byte) error {
	// the purpose of this function is to ensure that when unmarshaling, the
	// different *string values are never nil pointers.
	//
	// this is not a communicated feature of the API, so if it chnages
	// it's not a breaking change -- doesn't mean we can't try.
	var ief integrationEmailFilterRule
	if err := json.Unmarshal(b, &ief); err != nil {
		return err
	}

	i.BodyMode = ief.BodyMode
	i.SubjectMode = ief.SubjectMode
	i.FromEmailMode = ief.FromEmailMode

	// if the *string is nil, set it to a *string with value ""
	if ief.SubjectRegex == nil {
		i.SubjectRegex = new(string)
	} else {
		i.SubjectRegex = ief.SubjectRegex
	}

	if ief.BodyRegex == nil {
		i.BodyRegex = new(string)
	} else {
		i.BodyRegex = ief.BodyRegex
	}

	if ief.FromEmailRegex == nil {
		i.FromEmailRegex = new(string)
	} else {
		i.FromEmailRegex = ief.FromEmailRegex
	}

	return nil
}

type integrationEmailFilterRule struct {
	SubjectMode    IntegrationEmailFilterRuleMode `json:"subject_mode"`
	SubjectRegex   *string                        `json:"subject_regex,omitempty"`
	BodyMode       IntegrationEmailFilterRuleMode `json:"body_mode"`
	BodyRegex      *string                        `json:"body_regex,omitempty"`
	FromEmailMode  IntegrationEmailFilterRuleMode `json:"from_email_mode"`
	FromEmailRegex *string                        `json:"from_email_regex,omitempty"`
}

// Integration is an endpoint (like Nagios, email, or an API call) that
// generates events, which are normalized and de-duplicated by PagerDuty to
// create incidents.
type Integration struct {
	APIObject
	Name             string                       `json:"name,omitempty"`
	Service          *APIObject                   `json:"service,omitempty"`
	CreatedAt        string                       `json:"created_at,omitempty"`
	Vendor           *APIObject                   `json:"vendor,omitempty"`
	IntegrationKey   string                       `json:"integration_key,omitempty"`
	IntegrationEmail string                       `json:"integration_email,omitempty"`
	EmailFilterMode  IntegrationEmailFilterMode   `json:"email_filter_mode,omitempty"`
	EmailFilters     []IntegrationEmailFilterRule `json:"email_filters,omitempty"`
}

// CreateIntegration creates a new integration belonging to a service.
//
// Deprecated: Use CreateIntegrationWithContext instead.
func (c *Client) CreateIntegration(id string, i Integration) (*Integration, error) {
	return c.CreateIntegrationWithContext(context.Background(), id, i)
}

// CreateIntegrationWithContext creates a new integration belonging to a service.
func (c *Client) CreateIntegrationWithContext(ctx context.Context, id string, i Integration) (*Integration, error) {
	d := map[string]Integration{
		"integration": i,
	}

	resp, err := c.post(ctx, "/services/"+id+"/integrations", d, nil)
	return getIntegrationFromResponse(c, resp, err)
}

// GetIntegrationOptions is the data structure used when calling the GetIntegration API endpoint.
type GetIntegrationOptions struct {
	Includes []string `url:"include,omitempty,brackets"`
}

// GetIntegration gets details about an integration belonging to a service.
//
// Deprecated: Use GetIntegrationWithContext instead.
func (c *Client) GetIntegration(serviceID, integrationID string, o GetIntegrationOptions) (*Integration, error) {
	return c.GetIntegrationWithContext(context.Background(), serviceID, integrationID, o)
}

// GetIntegrationWithContext gets details about an integration belonging to a service.
func (c *Client) GetIntegrationWithContext(ctx context.Context, serviceID, integrationID string, o GetIntegrationOptions) (*Integration, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, "/services/"+serviceID+"/integrations/"+integrationID+"?"+v.Encode(), nil)
	return getIntegrationFromResponse(c, resp, err)
}

// UpdateIntegration updates an integration belonging to a service.
//
// Deprecated: Use UpdateIntegrationWithContext instead.
func (c *Client) UpdateIntegration(serviceID string, i Integration) (*Integration, error) {
	return c.UpdateIntegrationWithContext(context.Background(), serviceID, i)
}

// UpdateIntegrationWithContext updates an integration belonging to a service.
func (c *Client) UpdateIntegrationWithContext(ctx context.Context, serviceID string, i Integration) (*Integration, error) {
	resp, err := c.put(ctx, "/services/"+serviceID+"/integrations/"+i.ID, i, nil)
	return getIntegrationFromResponse(c, resp, err)
}

// DeleteIntegration deletes an existing integration.
//
// Deprecated: Use DeleteIntegrationWithContext instead.
func (c *Client) DeleteIntegration(serviceID string, integrationID string) error {
	return c.DeleteIntegrationWithContext(context.Background(), serviceID, integrationID)
}

// DeleteIntegrationWithContext deletes an existing integration.
func (c *Client) DeleteIntegrationWithContext(ctx context.Context, serviceID string, integrationID string) error {
	_, err := c.delete(ctx, "/services/"+serviceID+"/integrations/"+integrationID)
	return err
}
