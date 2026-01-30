package mailgun

import (
	"context"
	"net/http"
)

// The UserAgent identifies the client to the server, for logging purposes.
// In the event of problems requiring a human administrator's assistance,
// this user agent allows them to identify the client from human-generated activity.
const UserAgent = "mailgun-go/" + Version

// notGood searches a list of response codes (the haystack) for a matching entry (the needle).
// If found, the response code is considered good, and thus false is returned.
// Otherwise true is returned.
func notGood(needle int, haystack []int) bool {
	for _, i := range haystack {
		if needle == i {
			return false
		}
	}
	return true
}

// expected denotes the expected list of known-good HTTP response codes possible from the Mailgun API.
var expected = []int{200, 202, 204}

// doRequest performs a generic request, checking for a positive outcome.
func doRequest(ctx context.Context, r *httpRequest, method string, p payload) (*httpResponse, error) {
	r.addHeader("User-Agent", UserAgent)
	rsp, err := r.do(ctx, method, p)
	if (err == nil) && notGood(rsp.Code, expected) {
		return rsp, newError(method, r.URL, expected, rsp)
	}
	return rsp, err
}

// getResponseFromJSON shim performs a GET request, checking for a positive outcome.
func getResponseFromJSON(ctx context.Context, r *httpRequest, v any) error {
	r.addHeader("User-Agent", UserAgent)
	response, err := r.makeGetRequest(ctx)
	if err != nil {
		return err
	}
	if notGood(response.Code, expected) {
		return newError(http.MethodGet, r.URL, expected, response)
	}
	return response.parseFromJSON(v)
}

// postResponseFromJSON shim performs a POST request, checking for a positive outcome.
func postResponseFromJSON(ctx context.Context, r *httpRequest, p payload, v any) error {
	r.addHeader("User-Agent", UserAgent)
	response, err := r.makePostRequest(ctx, p)
	if err != nil {
		return err
	}
	if notGood(response.Code, expected) {
		return newError(http.MethodPost, r.URL, expected, response)
	}
	return response.parseFromJSON(v)
}

// putResponseFromJSON shim performs a PUT request, checking for a positive outcome.
func putResponseFromJSON(ctx context.Context, r *httpRequest, p payload, v any) error {
	r.addHeader("User-Agent", UserAgent)
	response, err := r.makePutRequest(ctx, p)
	if err != nil {
		return err
	}
	if notGood(response.Code, expected) {
		return newError(http.MethodPut, r.URL, expected, response)
	}
	return response.parseFromJSON(v)
}

// makeGetRequest shim performs a GET request, checking for a positive outcome.
func makeGetRequest(ctx context.Context, r *httpRequest) (*httpResponse, error) {
	r.addHeader("User-Agent", UserAgent)
	rsp, err := r.makeGetRequest(ctx)
	if (err == nil) && notGood(rsp.Code, expected) {
		return rsp, newError(http.MethodGet, r.URL, expected, rsp)
	}
	return rsp, err
}

// makePostRequest shim performs a POST request, checking for a positive outcome.
func makePostRequest(ctx context.Context, r *httpRequest, p payload) (*httpResponse, error) {
	r.addHeader("User-Agent", UserAgent)
	rsp, err := r.makePostRequest(ctx, p)
	if (err == nil) && notGood(rsp.Code, expected) {
		return rsp, newError(http.MethodPost, r.URL, expected, rsp)
	}
	return rsp, err
}

// makePutRequest shim performs a PUT request, checking for a positive outcome.
func makePutRequest(ctx context.Context, r *httpRequest, p payload) (*httpResponse, error) {
	r.addHeader("User-Agent", UserAgent)
	rsp, err := r.makePutRequest(ctx, p)
	if (err == nil) && notGood(rsp.Code, expected) {
		return rsp, newError(http.MethodPut, r.URL, expected, rsp)
	}
	return rsp, err
}

// makeDeleteRequest shim performs a DELETE request, checking for a positive outcome.
func makeDeleteRequest(ctx context.Context, r *httpRequest) (*httpResponse, error) {
	r.addHeader("User-Agent", UserAgent)
	rsp, err := r.makeDeleteRequest(ctx)
	if (err == nil) && notGood(rsp.Code, expected) {
		return rsp, newError(http.MethodDelete, r.URL, expected, rsp)
	}
	return rsp, err
}
