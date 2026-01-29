package mailgun

import (
	"context"

	"github.com/mailgun/errors"
	"github.com/mailgun/mailgun-go/v5/mtypes"
)

type MetricsOptions = mtypes.MetricsRequest

// ListMetrics returns domain/account metrics.
//
// To filter by domain:
//
//	opts.Filter.BoolGroupAnd = []mailgun.MetricsFilterPredicate{{
//		Attribute:     "domain",
//		Comparator:    "=",
//		LabeledValues: []mailgun.MetricsLabeledValue{{Label: "example.com", Value: "example.com"}},
//	}}
//
// https://documentation.mailgun.com/docs/mailgun/api-reference/openapi-final/tag/Metrics/
func (mg *Client) ListMetrics(opts MetricsOptions) (*MetricsIterator, error) {
	if opts.Pagination.Limit == 0 {
		opts.Pagination.Limit = 10
	}

	req := newHTTPRequest(generateApiUrl(mg, 1, metricsEndpoint))
	req.setClient(mg.HTTPClient())
	req.setBasicAuth(basicAuthUser, mg.APIKey())

	return &MetricsIterator{
		opts: opts,
		req:  req,
	}, nil
}

type MetricsIterator struct {
	opts MetricsOptions
	req  *httpRequest
	err  error
}

func (iter *MetricsIterator) Err() error {
	return iter.err
}

// Next retrieves the next page of items from the api. Returns false when there are
// no more pages to retrieve or if there was an error.
// Use `.Err()` to retrieve the error
func (iter *MetricsIterator) Next(ctx context.Context, resp *mtypes.MetricsResponse) (more bool) {
	if iter.err != nil {
		return false
	}

	iter.err = iter.fetch(ctx, resp)
	if iter.err != nil {
		return false
	}

	iter.opts.Pagination.Skip += iter.opts.Pagination.Limit

	return len(resp.Items) == iter.opts.Pagination.Limit
}

func (iter *MetricsIterator) fetch(ctx context.Context, resp *mtypes.MetricsResponse) error {
	if resp == nil {
		return errors.New("resp cannot be nil")
	}

	payload := newJSONEncodedPayload(iter.opts)

	httpResp, err := makePostRequest(ctx, iter.req, payload)
	if err != nil {
		return err
	}

	// preallocate
	resp.Items = make([]mtypes.MetricsItem, 0, iter.opts.Pagination.Limit)

	err = httpResp.parseFromJSON(resp)
	if err != nil {
		return errors.Wrap(err, "decoding response")
	}

	return nil
}
