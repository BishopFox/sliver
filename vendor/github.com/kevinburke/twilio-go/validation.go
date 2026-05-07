package twilio

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"sort"
)

// ValidateIncomingRequest returns an error if the incoming req could not be
// validated as coming from Twilio.
//
// This process is frequently error prone, especially if you are running behind
// a proxy, or Twilio is making requests with a port in the URL.
// See https://www.twilio.com/docs/security#validating-requests for more information
func ValidateIncomingRequest(host string, authToken string, req *http.Request) (err error) {
	err = req.ParseForm()
	if err != nil {
		return
	}
	err = validateIncomingRequest(host, authToken, req.URL.String(), req.PostForm, req.Header.Get("X-Twilio-Signature"))
	if err != nil {
		return
	}

	return
}

func validateIncomingRequest(host string, authToken string, URL string, postForm url.Values, xTwilioSignature string) (err error) {
	expectedTwilioSignature := GetExpectedTwilioSignature(host, authToken, URL, postForm)
	if xTwilioSignature != expectedTwilioSignature {
		err = errors.New("twilio: received X-Twilio-Signature value that does not match expected value")
		return
	}

	return
}

func GetExpectedTwilioSignature(host string, authToken string, URL string, postForm url.Values) (expectedTwilioSignature string) {
	// Take the full URL of the request URL you specify for your
	// phone number or app, from the protocol (https...) through
	// the end of the query string (everything after the ?).
	str := host + URL

	// If the request is a POST, sort all of the POST parameters
	// alphabetically (using Unix-style case-sensitive sorting order).
	keys := make([]string, 0, len(postForm))
	for key := range postForm {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Iterate through the sorted list of POST parameters, and append
	// the variable name and value (with no delimiters) to the end
	// of the URL string.
	for _, key := range keys {
		str += key + postForm[key][0]
	}

	// Sign the resulting string with HMAC-SHA1 using your AuthToken
	// as the key (remember, your AuthToken's case matters!).
	mac := hmac.New(sha1.New, []byte(authToken))
	mac.Write([]byte(str))
	expectedMac := mac.Sum(nil)

	// Base64 encode the resulting hash value.
	expectedTwilioSignature = base64.StdEncoding.EncodeToString(expectedMac)

	// Compare your hash to ours, submitted in the X-Twilio-Signature header.
	// If they match, then you're good to go.
	return expectedTwilioSignature
}
