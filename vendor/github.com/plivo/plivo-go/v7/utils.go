package plivo

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strings"

	validator "gopkg.in/go-playground/validator.v9"
)

func Numbers(numbers ...string) string {
	return strings.Join(numbers, "<")
}

func headersWithSep(headers map[string]string, keyValSep, itemSep string, escape bool) string {
	v := url.Values{}
	for key, value := range headers {
		v.Set(key, value)
	}
	var buf bytes.Buffer
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		vs := v[k]
		var prefix string
		if escape {
			prefix = url.QueryEscape(k) + keyValSep
		} else {
			prefix = k + keyValSep
		}

		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteString(itemSep)
			}
			buf.WriteString(prefix)
			if escape {
				buf.WriteString(url.QueryEscape(v))
			} else {
				buf.WriteString(v)
			}
		}
	}
	return buf.String()
}

// Some code from encode.go from the Go Standard Library
func Headers(headers map[string]string) string {
	return headersWithSep(headers, "=", ",", true)
}

// The old signature validation is deprecated. Will be marked deprecated in the next release.

func ComputeSignature(authToken, uri string, params map[string]string) string {
	originalString := fmt.Sprintf("%s%s", uri, headersWithSep(params, "", "", false))
	mac := hmac.New(sha1.New, []byte(authToken))
	mac.Write([]byte(originalString))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func ValidateSignature(authToken, uri string, params map[string]string, signature string) bool {
	return ComputeSignature(authToken, uri, params) == signature
}

// Adding V2 signature validation

func ComputeSignatureV2(authToken, uri string, nonce string) string {
	parsedUrl, err := url.Parse(uri)
	if err != nil {
		panic(err)
	}

	var originalString string = parsedUrl.Scheme + "://" + parsedUrl.Host + parsedUrl.Path + nonce
	mac := hmac.New(sha256.New, []byte(authToken))
	mac.Write([]byte(originalString))
	var messageMAC string = base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return messageMAC
}

func ValidateSignatureV2(uri string, nonce string, signature string, authToken string) bool {
	return ComputeSignatureV2(authToken, uri, nonce) == signature
}

func GenerateUrl(uri string, params map[string]string, method string) string {
	parsedUrl, err := url.Parse(uri)
	if err != nil {
		panic(err)
	}
	uri = parsedUrl.Scheme + "://" + parsedUrl.Host + parsedUrl.Path
	if len(params) > 0 || len(parsedUrl.RawQuery) > 0 {
		uri += "?"
	}
	if len(parsedUrl.RawQuery) > 0 {
		if method == "GET" {
			queryParamMap := getMapFromQueryString(parsedUrl.Query())
			for k, v := range params {
				queryParamMap[k] = v
			}
			uri += GetSortedQueryParamString(queryParamMap, true)
		} else {
			uri += GetSortedQueryParamString(getMapFromQueryString(parsedUrl.Query()), true) + "." + GetSortedQueryParamString(params, false)
			uri = strings.TrimRight(uri, ".")
		}
	} else {
		if method == "GET" {
			uri += GetSortedQueryParamString(params, true)
		} else {
			uri += GetSortedQueryParamString(params, false)
		}
	}
	return uri
}

func getMapFromQueryString(query url.Values) map[string]string {
	/*
		Example: input  "a=b&c=d&z=x"
		output (string): {"z":x", "c": "d", "a": "b"}
	*/
	mp := make(map[string]string)
	if len(query) == 0 {
		return mp
	}
	for key, val := range query {
		mp[key] = val[0]
	}
	return mp
}

func GetSortedQueryParamString(params map[string]string, queryParams bool) string {
	url := ""
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if queryParams {
		for _, key := range keys {
			url += key + "=" + params[key] + "&"
		}
		url = strings.TrimRight(url, "&")
	} else {
		for _, key := range keys {
			url += key + params[key]
		}
	}
	return url
}

func ComputeSignatureV3(authToken, uri, method string, nonce string, params map[string]string) string {
	var newUrl = GenerateUrl(uri, params, method) + "." + nonce
	mac := hmac.New(sha256.New, []byte(authToken))
	mac.Write([]byte(newUrl))
	var messageMAC = base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return messageMAC
}

func ValidateSignatureV3(uri, nonce, method, signature, authToken string, params ...map[string]string) bool {
	parameters := map[string]string{}
	if len(params) != 0 {
		parameters = params[0]
	}
	multipleSignatures := strings.Split(signature, ",")
	return Find(ComputeSignatureV3(authToken, uri, method, nonce, parameters), multipleSignatures)
}
func CreateWhatsappInteractive(interactiveData string) (interactive Interactive, err error) {
	err = json.Unmarshal([]byte(interactiveData), &interactive)
	if err != nil {
		return
	}
	validate := validator.New()
	err = validate.Struct(interactive)
	return
}

func CreateWhatsappLocation(locationData string) (location Location, err error) {
	err = json.Unmarshal([]byte(locationData), &location)
	if err != nil {
		return
	}
	validate := validator.New()
	err = validate.Struct(location)
	return
}

func CreateWhatsappTemplate(templateData string) (template Template, err error) {
	err = json.Unmarshal([]byte(templateData), &template)
	if err != nil {
		return
	}
	err = validateWhatsappTemplate(template)
	return
}

func validateWhatsappTemplate(template Template) (err error) {
	validate := validator.New()
	err = validate.Struct(template)
	if err != nil {
		return
	}
	if template.Components != nil {
		for _, component := range template.Components {
			err = validate.Struct(component)
			if err != nil {
				return
			}
			if component.Parameters != nil {
				for _, parameter := range component.Parameters {
					err = validate.Struct(parameter)
					if err != nil {
						return
					}
					if parameter.Currency != nil {
						err = validate.Struct(parameter.Currency)
						if err != nil {
							return
						}
					}
					if parameter.DateTime != nil {
						err = validate.Struct(parameter.DateTime)
						if err != nil {
							return
						}
					}
				}
			}

		}
	}
	return
}

func Find(val string, slice []string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func checkAndFetchCallInsightsRequestDetails(param interface{}) (isCallInsightsRequest bool, requestPath string) {
	isCallInsightsRequest = false
	if reflect.TypeOf(param).Kind() == reflect.Map {
		if reflect.TypeOf(param).Key().Kind() == reflect.String {
			if _, ok := param.(map[string]interface{})[CallInsightsParams]; ok {
				isCallInsightsRequest = true
				requestPath = param.(map[string]interface{})[CallInsightsParams].(map[string]interface{})[CallInsightsRequestPath].(string)
			}
		}
	}
	return
}

func isVoiceRequest() (extraData map[string]interface{}) {
	extraData = make(map[string]interface{})
	extraData["is_voice_request"] = true
	extraData["retry"] = 0
	return extraData
}
