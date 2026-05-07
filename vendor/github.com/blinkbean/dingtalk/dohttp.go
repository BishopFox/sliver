package dingtalk

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

var (
	myHTTPClient *http.Client
)

const (
	defaultDialTimeout = 2 * time.Second
	defaultKeepAlive   = 2 * time.Second
)

func init() {
	myHTTPClient = initDefaultHTTPClient()
}

// initDefaultHTTPClient for connection re-use
func initDefaultHTTPClient() *http.Client {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   defaultDialTimeout,
				KeepAlive: defaultKeepAlive,
			}).DialContext,
		},
		Timeout: defaultDialTimeout,
	}
	return client
}

func doRequest(ctx context.Context, callMethod string, endPoint string, header map[string]string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(callMethod, endPoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	if header != nil && len(header) > 0 {
		for k, v := range header {
			req.Header.Set(k, v)
		}
	}
	req = req.WithContext(ctx)
	// use myHttpClient to send request
	response, err := myHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("response is nil, please check it")
	}

	return response, nil
}
