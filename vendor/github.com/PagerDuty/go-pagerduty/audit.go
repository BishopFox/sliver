package pagerduty

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-querystring/query"
)

const auditBaseURL = "/audit/records"

// ListAuditRecordsOptions is the data structure used when calling the
// ListAuditRecords API endpoint.
type ListAuditRecordsOptions struct {
	Actions              []string `url:"actions,omitempty,brackets"`
	ActorID              string   `url:"actor_id,omitempty"`
	ActorType            string   `url:"actor_type,omitempty"`
	Cursor               string   `url:"cursor,omitempty"`
	Limit                uint     `url:"limit,omitempty"`
	MethodTruncatedToken string   `url:"method_truncated_token,omitempty"`
	MethodType           string   `url:"method_type,omitempty"`
	RootResourcesTypes   []string `url:"root_resources_types,omitempty,brackets"`
	Since                string   `url:"since,omitempty"`
	Until                string   `url:"until,omitempty"`
}

// ListAuditRecordsResponse is the response data received when calling the
// ListAuditRecords API endpoint.
type ListAuditRecordsResponse struct {
	Records []AuditRecord `json:"records,omitempty"`
	// ResponseMetadata is not a required field in the pagerduty API response,
	// using a pointer allows us to not marshall an empty ResponseMetaData struct
	// into a JSON.
	ResponseMetaData *ResponseMetadata `json:"response_metadata,omitempty"`
	Limit            uint              `json:"limit,omitempty"`
	// NextCursor is an  opaque string that will deliver the next set of results
	// when provided as the cursor parameter in a subsequent request.
	// A null value for this field indicates that there are no additional results.
	// We use a pointer here to marshall the string value into null
	// when NextCursor is an empty string.
	NextCursor *string `json:"next_cursor"`
}

// AuditRecord is a audit trail record that matches the query criteria.
type AuditRecord struct {
	ID               string           `json:"id,omitempty"`
	Self             string           `json:"self,omitempty"`
	ExecutionTime    string           `json:"execution_time,omitempty"`
	ExecutionContext ExecutionContext `json:"execution_context,omitempty"`
	Actors           []APIObject      `json:"actors,omitempty"`
	Method           Method           `json:"method,omitempty"`
	RootResource     APIObject        `json:"root_resource,omitempty"`
	Action           string           `json:"action,omitempty"`
	Details          Details          `json:"details,omitempty"`
}

// ResponseMetadata contains information about the response.
type ResponseMetadata struct {
	Messages []string `json:"messages,omitempty"`
}

// ExecutionContext contains information about the action execution context.
type ExecutionContext struct {
	RequestID     string `json:"request_id,omitempty"`
	RemoteAddress string `json:"remote_address,omitempty"`
}

// Method contains information on the method used to perform the action.
type Method struct {
	Description    string `json:"description,omitempty"`
	TruncatedToken string `json:"truncated_token,omitempty"`
	Type           string `json:"type,omitempty"`
}

// Details contain additional information about the action or the resource
// that has been audited.
type Details struct {
	Resource APIObject `json:"resource,omitempty"`
	// A set of fields that have been affected.
	// The fields that have not been affected MAY be returned.
	Fields []Field `json:"fields,omitempty"`
	// A set of references that have been affected.
	References []Reference `json:"references,omitempty"`
}

// Field contains information about the resource field that have been affected.
type Field struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Value       string `json:"value,omitempty"`
	BeforeValue string `json:"before_value,omitempty"`
}

// Reference contains information about the reference that have been affected.
type Reference struct {
	Name        string      `json:"name,omitempty"`
	Description string      `json:"description,omitempty"`
	Added       []APIObject `json:"added,omitempty"`
	Removed     []APIObject `json:"removed,omitempty"`
}

// ListAuditRecords lists audit trial records matching provided query params
// or default criteria.
func (c *Client) ListAuditRecords(ctx context.Context, o ListAuditRecordsOptions) (ListAuditRecordsResponse, error) {
	v, err := query.Values(o)
	if err != nil {
		return ListAuditRecordsResponse{}, err
	}

	u := fmt.Sprintf("%s?%s", auditBaseURL, v.Encode())
	resp, err := c.get(ctx, u, nil)
	if err != nil {
		return ListAuditRecordsResponse{}, err
	}

	var result ListAuditRecordsResponse
	if err = c.decodeJSON(resp, &result); err != nil {
		return ListAuditRecordsResponse{}, err
	}

	return result, nil
}

// ListAuditRecordsPaginated lists audit trial records matching provided query
// params or default criteria, processing paginated responses. The include
// function decides whether or not to include a specific AuditRecord in
// the final result. If the include function is nil, all audit records from
// the API are included by default.
func (c *Client) ListAuditRecordsPaginated(ctx context.Context, o ListAuditRecordsOptions, include func(AuditRecord) bool) ([]AuditRecord, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	if include == nil {
		include = func(AuditRecord) bool { return true }
	}

	var records []AuditRecord

	responseHandler := func(response *http.Response) (cursor, error) {
		var result ListAuditRecordsResponse
		if err := c.decodeJSON(response, &result); err != nil {
			return cursor{}, err
		}

		for _, r := range result.Records {
			if include(r) {
				records = append(records, r)
			}
		}

		return cursor{
			Limit:      result.Limit,
			NextCursor: *result.NextCursor,
		}, nil
	}

	u := fmt.Sprintf("%s?%s", auditBaseURL, v.Encode())
	if err := c.cursorGet(ctx, u, responseHandler); err != nil {
		return nil, err
	}

	return records, nil
}
