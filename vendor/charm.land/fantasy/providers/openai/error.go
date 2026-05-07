package openai

import (
	"cmp"
	"errors"
	"io"
	"net/http"
	"strings"

	"charm.land/fantasy"
	"github.com/openai/openai-go/v2"
)

func toProviderErr(err error) error {
	var apiErr *openai.Error
	if errors.As(err, &apiErr) {
		return &fantasy.ProviderError{
			Title:           cmp.Or(fantasy.ErrorTitleForStatusCode(apiErr.StatusCode), "provider request failed"),
			Message:         toProviderErrMessage(apiErr),
			Cause:           apiErr,
			URL:             apiErr.Request.URL.String(),
			StatusCode:      apiErr.StatusCode,
			RequestBody:     apiErr.DumpRequest(true),
			ResponseHeaders: toHeaderMap(apiErr.Response.Header),
			ResponseBody:    apiErr.DumpResponse(true),
		}
	}
	return err
}

func toProviderErrMessage(apiErr *openai.Error) string {
	if apiErr.Message != "" {
		return apiErr.Message
	}

	// For some OpenAI-compatible providers, the SDK is not always able to parse
	// the error message correctly.
	// Fallback to returning the raw response body in such cases.
	data, _ := io.ReadAll(apiErr.Response.Body)
	return string(data)
}

func toHeaderMap(in http.Header) (out map[string]string) {
	out = make(map[string]string, len(in))
	for k, v := range in {
		if l := len(v); l > 0 {
			out[k] = v[l-1]
			in[strings.ToLower(k)] = v
		}
	}
	return out
}
