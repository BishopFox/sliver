package plivo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"

	"github.com/google/go-querystring/query"
)

const sdkVersion = "7.59.2"

const lookupBaseUrl = "lookup.plivo.com"

type ClientOptions struct {
	HttpClient *http.Client
}

type BaseClient struct {
	httpClient *http.Client

	AuthId    string
	AuthToken string

	BaseUrl   *url.URL
	userAgent string

	RequestInterceptor  func(request *http.Request)
	ResponseInterceptor func(response *http.Response)
}

func (client *BaseClient) NewRequest(method string, params interface{}, baseRequestString string, formatString string,
	formatParams ...interface{}) (request *http.Request, err error) {

	if client == nil || client.httpClient == nil {
		err = errors.New("client and httpClient cannot be nil")
		return
	}

	isCallInsightsRequest := false
	var requestPath string

	for i, param := range formatParams {
		if !isCallInsightsRequest {
			isCallInsightsRequest, requestPath = checkAndFetchCallInsightsRequestDetails(param)
		}
		if param == nil || param == "" {
			err = fmt.Errorf("Request path parameter #%d is nil/empty but should not be so.", i)
			return
		}
	}

	requestUrl := *client.BaseUrl
	requestUrl.Path = fmt.Sprintf(baseRequestString, fmt.Sprintf(formatString, formatParams...))

	if isCallInsightsRequest {
		requestUrl.Host = CallInsightsBaseURL
		requestUrl.Path = requestPath
	}

	var buffer = new(bytes.Buffer)
	if method == "GET" {
		var values url.Values
		if values, err = query.Values(params); err != nil {
			return
		}

		requestUrl.RawQuery = values.Encode()
	} else {
		if reflect.ValueOf(params).Kind().String() != "map" && params != nil {
			if err = json.NewEncoder(buffer).Encode(params); err != nil {
				return
			}
		} else if reflect.ValueOf(params).Kind().String() == "map" && !reflect.ValueOf(params).IsNil() {
			if err = json.NewEncoder(buffer).Encode(params); err != nil {
				return
			}
		}

	}

	request, err = http.NewRequest(method, requestUrl.String(), buffer)

	request.Header.Add("User-Agent", client.userAgent)
	request.Header.Add("Content-Type", "application/json")

	request.SetBasicAuth(client.AuthId, client.AuthToken)

	return
}

func (client *BaseClient) ExecuteRequest(request *http.Request, body interface{}, extra ...map[string]interface{}) (err error) {
	isVoiceRequest := false
	if extra != nil {
		if _, ok := extra[0]["is_voice_request"]; ok {
			isVoiceRequest = true
			request.URL.Host = voiceBaseUrlString
			request.Host = voiceBaseUrlString
			request.URL.Scheme = HttpsScheme
			if extra[0]["retry"] == 1 {
				request.URL.Host = voiceBaseUrlStringFallback1
				request.Host = voiceBaseUrlStringFallback2
				request.URL.Scheme = HttpsScheme
			} else if extra[0]["retry"] == 2 {
				request.URL.Host = voiceBaseUrlStringFallback2
				request.Host = voiceBaseUrlStringFallback2
				request.URL.Scheme = HttpsScheme
			}
		}

		if _, ok := extra[0]["is_lookup_request"]; ok {
			if request.URL.Host == "api.plivo.com" { // hack for unit tests
				request.URL.Host = lookupBaseUrl
				request.Host = lookupBaseUrl
				request.URL.Scheme = HttpsScheme
			}
		}
	}
	bodyCopy, _ := ioutil.ReadAll(request.Body)
	request.Body = ioutil.NopCloser(bytes.NewReader(bodyCopy))
	response, err := client.httpClient.Do(request)

	if err != nil {
		return
	}

	data, err := ioutil.ReadAll(response.Body)
	if err == nil && data != nil && len(data) > 0 {
		if isVoiceRequest && response.StatusCode >= 500 {
			if extra[0]["retry"] == 2 {
				err = errors.New(string(data))
				return
			}
			extra[0]["retry"] = extra[0]["retry"].(int) + 1

			newRequest, _ := http.NewRequest(request.Method, request.URL.String(), bytes.NewReader(bodyCopy))
			newRequest.Header.Add("User-Agent", client.userAgent)
			newRequest.Header.Add("Content-Type", "application/json")
			newRequest.SetBasicAuth(client.AuthId, client.AuthToken)

			_ = client.ExecuteRequest(newRequest, body, extra...)
		} else if response.StatusCode >= 200 && response.StatusCode < 300 {
			if body != nil {
				err = json.Unmarshal(data, body)
			}
		} else if string(data) == "{}" && response.StatusCode == 404 {
			err = errors.New("Resource not found exception \n" + response.Status)
		} else {
			err = errors.New(string(data))
		}

	}

	return
}
