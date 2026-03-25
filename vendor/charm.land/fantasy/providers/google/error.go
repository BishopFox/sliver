package google

import (
	"cmp"
	"errors"

	"charm.land/fantasy"
	"google.golang.org/genai"
)

func toProviderErr(err error) error {
	var apiErr genai.APIError
	if !errors.As(err, &apiErr) {
		return err
	}
	return &fantasy.ProviderError{
		Message:      apiErr.Message,
		Title:        cmp.Or(fantasy.ErrorTitleForStatusCode(apiErr.Code), "provider request failed"),
		Cause:        err,
		StatusCode:   apiErr.Code,
		ResponseBody: []byte(apiErr.Message),
	}
}
