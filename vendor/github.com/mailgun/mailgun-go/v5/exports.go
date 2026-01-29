package mailgun

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/mailgun/mailgun-go/v5/mtypes"
)

// CreateExport creates an export based on the URL given
func (mg *Client) CreateExport(ctx context.Context, url string) error {
	r := newHTTPRequest(generateApiUrl(mg, 3, exportsEndpoint))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	payload := newUrlEncodedPayload()
	payload.addValue("url", url)
	_, err := makePostRequest(ctx, r, payload)
	return err
}

// ListExports lists all exports created within the past 24 hours
func (mg *Client) ListExports(ctx context.Context, url string) ([]mtypes.Export, error) {
	r := newHTTPRequest(generateApiUrl(mg, 3, exportsEndpoint))
	r.setClient(mg.HTTPClient())
	if url != "" {
		r.addParameter("url", url)
	}
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	var resp mtypes.ExportList
	if err := getResponseFromJSON(ctx, r, &resp); err != nil {
		return nil, err
	}

	var result []mtypes.Export
	for _, item := range resp.Items {
		result = append(result, mtypes.Export(item))
	}
	return result, nil
}

// GetExport gets an export by id
func (mg *Client) GetExport(ctx context.Context, id string) (mtypes.Export, error) {
	r := newHTTPRequest(generateApiUrl(mg, 3, exportsEndpoint) + "/" + id)
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	var resp mtypes.Export
	err := getResponseFromJSON(ctx, r, &resp)
	return resp, err
}

// Download an export by ID. This will respond with a '302 Moved'
// with the Location header of temporary S3 URL if it is available.
func (mg *Client) GetExportLink(ctx context.Context, id string) (string, error) {
	r := newHTTPRequest(generateApiUrl(mg, 3, exportsEndpoint) + "/" + id + "/download_url")
	c := mg.HTTPClient()

	// Ensure the client doesn't attempt to retry
	c.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return errors.New("redirect")
	}

	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	r.addHeader("User-Agent", UserAgent)

	req, err := r.NewRequest(ctx, http.MethodGet, nil)
	if err != nil {
		return "", err
	}

	if Debug {
		fmt.Println(curlString(req, nil))
	}

	resp, err := r.Client.Do(req)
	if err != nil {
		if resp != nil { // TODO(vtopc): not nil err and resp at the same time, is that possible at all?
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusFound {
				url, err := resp.Location()
				if err != nil {
					return "", fmt.Errorf("while parsing 302 redirect url: %s", err)
				}

				return url.String(), nil
			}
		}

		return "", err
	}

	defer resp.Body.Close()

	return "", fmt.Errorf("expected a 302 response, API returned a '%d' instead", resp.StatusCode)
}
