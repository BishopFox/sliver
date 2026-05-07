package plivo

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"time"
)

const phloUrlString = "https://phlorunner.plivo.com"
const phloBaseRequestString = "v1/%s"

type PhloClient struct {
	BaseClient

	Phlos *Phlos
}

/*
To set a proxy for all requests, configure the Transport for the HttpClient passed in:

	&http.Client{
 		Transport: &http.Transport{
 			Proxy: http.ProxyURL("http//your.proxy.here"),
 		},
 	}

Similarly, to configure the timeout, set it on the HttpClient passed in:

	&http.Client{
 		Timeout: time.Minute,
 	}
*/
func NewPhloClient(authId, authToken string, options *ClientOptions) (client *PhloClient, err error) {

	client = &PhloClient{}

	if len(authId) == 0 {
		authId = os.Getenv("PLIVO_AUTH_ID")
	}
	if len(authToken) == 0 {
		authToken = os.Getenv("PLIVO_AUTH_TOKEN")
	}
	client.AuthId = authId
	client.AuthToken = authToken
	client.userAgent = fmt.Sprintf("%s/%s (Go: %s)", "plivo-go", sdkVersion, runtime.Version())

	baseUrl, err := url.Parse(phloUrlString) // Todo: handle error case?

	client.BaseUrl = baseUrl
	client.httpClient = &http.Client{
		Timeout: time.Minute,
	}

	if options.HttpClient != nil {
		client.httpClient = options.HttpClient
	}

	client.Phlos = NewPhlos(client)

	return
}

func (client *PhloClient) NewRequest(method string, params interface{}, formatString string,
	formatParams ...interface{}) (*http.Request, error) {

	return client.BaseClient.NewRequest(method, params, phloBaseRequestString, formatString, formatParams...)
}
