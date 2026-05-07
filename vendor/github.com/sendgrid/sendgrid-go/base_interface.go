package sendgrid

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/sendgrid/rest"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// Version is this client library's current version
const (
	Version        = "3.16.1"
	rateLimitRetry = 5
	rateLimitSleep = 1100
)

type options struct {
	Auth     string
	Endpoint string
	Host     string
	Subuser  string
}

// Client is the Twilio SendGrid Go client
type Client struct {
	rest.Request
}

func (o *options) baseURL() string {
	return o.Host + o.Endpoint
}

// requestNew create Request
// @return [Request] a default request object
func requestNew(options options) rest.Request {
	requestHeaders := map[string]string{
		"Authorization": options.Auth,
		"User-Agent":    "sendgrid/" + Version + ";go",
		"Accept":        "application/json",
	}

	if len(options.Subuser) != 0 {
		requestHeaders["On-Behalf-Of"] = options.Subuser
	}

	return rest.Request{
		BaseURL: options.baseURL(),
		Headers: requestHeaders,
	}
}

// Send sends an email through Twilio SendGrid
func (cl *Client) Send(email *mail.SGMailV3) (*rest.Response, error) {
	return cl.SendWithContext(context.Background(), email)
}

// SendWithContext sends an email through Twilio SendGrid with context.Context.
func (cl *Client) SendWithContext(ctx context.Context, email *mail.SGMailV3) (*rest.Response, error) {
	cl.Body = mail.GetRequestBody(email)
	// when Content-Encoding header is set to "gzip"
	// mail body is compressed using gzip according to
	// https://docs.sendgrid.com/api-reference/mail-send/mail-send#mail-body-compression
	if cl.Headers["Content-Encoding"] == "gzip" {
		var gzipped bytes.Buffer
		gz := gzip.NewWriter(&gzipped)
		if _, err := gz.Write(cl.Body); err != nil {
			return nil, err
		}
		if err := gz.Flush(); err != nil {
			return nil, err
		}
		if err := gz.Close(); err != nil {
			return nil, err
		}

		cl.Body = gzipped.Bytes()
	}
	return MakeRequestWithContext(ctx, cl.Request)
}

// DefaultClient is used if no custom HTTP client is defined
var DefaultClient = rest.DefaultClient

// API sets up the request to the Twilio SendGrid API, this is main interface.
// Please use the MakeRequest or MakeRequestAsync functions instead.
// (deprecated)
func API(request rest.Request) (*rest.Response, error) {
	return MakeRequest(request)
}

// MakeRequest attempts a Twilio SendGrid request synchronously.
func MakeRequest(request rest.Request) (*rest.Response, error) {
	return MakeRequestWithContext(context.Background(), request)
}

// MakeRequestWithContext attempts a Twilio SendGrid request synchronously with context.Context.
func MakeRequestWithContext(ctx context.Context, request rest.Request) (*rest.Response, error) {
	return DefaultClient.SendWithContext(ctx, request)
}

// MakeRequestRetry a synchronous request, but retry in the event of a rate
// limited response.
func MakeRequestRetry(request rest.Request) (*rest.Response, error) {
	return MakeRequestRetryWithContext(context.Background(), request)
}

// MakeRequestRetryWithContext a synchronous request with context.Context, but retry in the event of a rate
// limited response.
func MakeRequestRetryWithContext(ctx context.Context, request rest.Request) (*rest.Response, error) {
	retry := 0
	var response *rest.Response
	var err error

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		response, err = MakeRequestWithContext(ctx, request)
		if err != nil {
			return nil, err
		}

		if response.StatusCode != http.StatusTooManyRequests {
			return response, nil
		}

		if retry > rateLimitRetry {
			return nil, errors.New("rate limit retry exceeded")
		}
		retry++

		resetTime := time.Now().Add(rateLimitSleep * time.Millisecond)

		reset, ok := response.Headers["X-RateLimit-Reset"]
		if ok && len(reset) > 0 {
			t, err := strconv.Atoi(reset[0])
			if err == nil {
				resetTime = time.Unix(int64(t), 0)
			}
		}
		time.Sleep(resetTime.Sub(time.Now()))
	}
}

// MakeRequestAsync attempts a request asynchronously in a new go
// routine. This function returns two channels: responses
// and errors. This function will retry in the case of a
// rate limit.
func MakeRequestAsync(request rest.Request) (chan *rest.Response, chan error) {
	return MakeRequestAsyncWithContext(context.Background(), request)
}

// MakeRequestAsyncWithContext attempts a request asynchronously in a new go
// routine with context.Context. This function returns two channels: responses
// and errors. This function will retry in the case of a
// rate limit.
func MakeRequestAsyncWithContext(ctx context.Context, request rest.Request) (chan *rest.Response, chan error) {
	r := make(chan *rest.Response)
	e := make(chan error)

	go func() {
		response, err := MakeRequestRetryWithContext(ctx, request)
		if err != nil {
			e <- err
		}
		if response != nil {
			r <- response
		}
	}()

	return r, e
}
