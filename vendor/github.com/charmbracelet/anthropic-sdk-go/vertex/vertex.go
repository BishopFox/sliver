package vertex

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/transport"
	thttp "google.golang.org/api/transport/http"

	"github.com/charmbracelet/anthropic-sdk-go/internal/requestconfig"
	sdkoption "github.com/charmbracelet/anthropic-sdk-go/option"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const DefaultVersion = "vertex-2023-10-16"

// WithGoogleAuth returns a request option which loads the [Application Default Credentials] for Google Vertex AI and registers
// middleware that intercepts requests to the Messages API.
//
// If you already have a [*google.Credentials], it is recommended that you instead call [WithCredentials] directly.
//
// [Application Default Credentials]: https://cloud.google.com/docs/authentication/application-default-credentials
func WithGoogleAuth(ctx context.Context, region string, projectID string, scopes ...string) sdkoption.RequestOption {
	if region == "" {
		panic("region must be provided")
	}
	creds, err := google.FindDefaultCredentials(ctx, scopes...)
	if err != nil {
		panic(fmt.Errorf("failed to find default credentials: %v", err))
	}
	return WithCredentials(ctx, region, projectID, creds)
}

// WithCredentials returns a request option which uses the provided credentials for Google Vertex AI and registers middleware that
// intercepts request to the Messages API.
func WithCredentials(ctx context.Context, region string, projectID string, creds *google.Credentials) sdkoption.RequestOption {
	middleware := vertexMiddleware(region, projectID)

	var baseURL string
	if region == "global" {
		baseURL = "https://aiplatform.googleapis.com/"
	} else {
		baseURL = fmt.Sprintf("https://%s-aiplatform.googleapis.com/", region)
	}

	return requestconfig.RequestOptionFunc(func(rc *requestconfig.RequestConfig) error {
		getClient := func() (*http.Client, error) {
			if rc.HTTPClient == nil || rc.HTTPClient.Transport == nil {
				c, _, err := transport.NewHTTPClient(ctx, option.WithTokenSource(creds.TokenSource))
				return c, err
			}
			transport, err := thttp.NewTransport(
				ctx,
				rc.HTTPClient.Transport,
				option.WithTokenSource(creds.TokenSource),
			)
			return &http.Client{Transport: transport}, err
		}

		client, err := getClient()
		if err != nil {
			return fmt.Errorf("failed to create http client: %v", err)
		}

		return rc.Apply(
			sdkoption.WithBaseURL(baseURL),
			sdkoption.WithMiddleware(middleware),
			sdkoption.WithHTTPClient(client),
		)
	})
}

func vertexMiddleware(region, projectID string) sdkoption.Middleware {
	return func(r *http.Request, next sdkoption.MiddlewareNext) (*http.Response, error) {
		if r.Body != nil {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, err
			}
			if err := r.Body.Close(); err != nil {
				return nil, err
			}

			if !gjson.GetBytes(body, "anthropic_version").Exists() {
				body, _ = sjson.SetBytes(body, "anthropic_version", DefaultVersion)
			}

			switch {
			case r.URL.Path == "/v1/messages" && r.Method == http.MethodPost:
				if projectID == "" {
					return nil, fmt.Errorf("no projectId was given and it could not be resolved from credentials")
				}

				model := gjson.GetBytes(body, "model").String()
				stream := gjson.GetBytes(body, "stream").Bool()

				body, _ = sjson.DeleteBytes(body, "model")

				specifier := "rawPredict"
				if stream {
					specifier = "streamRawPredict"
				}

				r.URL.Path = fmt.Sprintf("/v1/projects/%s/locations/%s/publishers/anthropic/models/%s:%s", projectID, region, model, specifier)

			case r.URL.Path == "/v1/messages/count_tokens" && r.Method == http.MethodPost:
				if projectID == "" {
					return nil, fmt.Errorf("no projectId was given and it could not be resolved from credentials")
				}

				r.URL.Path = fmt.Sprintf("/v1/projects/%s/locations/%s/publishers/anthropic/models/count-tokens:rawPredict", projectID, region)

			default:
				return nil, fmt.Errorf("vertex middleware does not support %s %s", r.Method, r.URL.Path)
			}

			reader := bytes.NewReader(body)
			r.Body = io.NopCloser(reader)
			r.GetBody = func() (io.ReadCloser, error) {
				_, err := reader.Seek(0, 0)
				return io.NopCloser(reader), err
			}
			r.ContentLength = int64(len(body))
		}

		return next(r)
	}
}
