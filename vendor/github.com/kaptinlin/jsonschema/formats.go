package jsonschema

import (
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kaptinlin/jsonpointer"
)

// Formats is a registry of functions, which know how to validate
// a specific format.
//
// New Formats can be registered by adding to this map. Key is format name,
// value is function that knows how to validate that format.
var Formats = map[string]func(any) bool{
	"date-time":             IsDateTime,
	"date":                  IsDate,
	"time":                  IsTime,
	"duration":              IsDuration,
	"period":                IsPeriod,
	"hostname":              IsHostname,
	"email":                 IsEmail,
	"ip-address":            IsIPV4,
	"ipv4":                  IsIPV4,
	"ipv6":                  IsIPV6,
	"uri":                   IsURI,
	"iri":                   IsURI,
	"uri-reference":         IsURIReference,
	"uriref":                IsURIReference,
	"iri-reference":         IsURIReference,
	"uri-template":          IsURITemplate,
	"json-pointer":          IsJSONPointer,
	"relative-json-pointer": IsRelativeJSONPointer,
	"uuid":                  IsUUID,
	"regex":                 IsRegex,
	"unknown":               func(any) bool { return true },
}

// IsDateTime tells whether given string is a valid date representation
// as defined by RFC 3339, section 5.6.
//
// see https://datatracker.ietf.org/doc/html/rfc3339#section-5.6, for details
func IsDateTime(v any) bool {
	s, ok := v.(string)
	if !ok {
		return true
	}
	if len(s) < 20 { // yyyy-mm-ddThh:mm:ssZ
		return false
	}
	if s[10] != 'T' && s[10] != 't' {
		return false
	}
	return IsDate(s[:10]) && IsTime(s[11:])
}

// IsDate tells whether given string is a valid full-date production
// as defined by RFC 3339, section 5.6.
//
// see https://datatracker.ietf.org/doc/html/rfc3339#section-5.6, for details
func IsDate(v any) bool {
	s, ok := v.(string)
	if !ok {
		return true
	}
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// IsTime tells whether given string is a valid full-time production
// as defined by RFC 3339, section 5.6.
//
// see https://datatracker.ietf.org/doc/html/rfc3339#section-5.6, for details
func IsTime(v any) bool {
	str, ok := v.(string)
	if !ok {
		return true
	}

	// golang time package does not support leap seconds.
	// so we are parsing it manually here.

	// hh:mm:ss
	// 01234567
	if len(str) < 9 || str[2] != ':' || str[5] != ':' {
		return false
	}
	isInRange := func(str string, minVal, maxVal int) (int, bool) {
		n, err := strconv.Atoi(str)
		if err != nil {
			return 0, false
		}
		if n < minVal || n > maxVal {
			return 0, false
		}
		return n, true
	}
	var h, m, s int
	if h, ok = isInRange(str[0:2], 0, 23); !ok {
		return false
	}
	if m, ok = isInRange(str[3:5], 0, 59); !ok {
		return false
	}
	if s, ok = isInRange(str[6:8], 0, 60); !ok {
		return false
	}
	str = str[8:]

	// parse secfrac if present
	if str[0] == '.' {
		// dot following more than one digit
		str = str[1:]
		var numDigits int
		for str != "" {
			if str[0] < '0' || str[0] > '9' {
				break
			}
			numDigits++
			str = str[1:]
		}
		if numDigits == 0 {
			return false
		}
	}

	if len(str) == 0 {
		return false
	}

	if str[0] == 'z' || str[0] == 'Z' {
		if len(str) != 1 {
			return false
		}
	} else {
		// time-numoffset
		// +hh:mm
		// 012345
		if len(str) != 6 || str[3] != ':' {
			return false
		}

		var sign int
		switch str[0] {
		case '+':
			sign = -1
		case '-':
			sign = +1
		default:
			return false
		}

		var zh, zm int
		ok := false
		if zh, ok = isInRange(str[1:3], 0, 23); !ok {
			return false
		}
		if zm, ok = isInRange(str[4:6], 0, 59); !ok {
			return false
		}

		// apply timezone offset
		hm := (h*60 + m) + sign*(zh*60+zm)
		if hm < 0 {
			hm += 24 * 60
		}
		h, m = hm/60, hm%60
	}

	// check leapsecond
	if s == 60 { // leap second
		if h != 23 || m != 59 {
			return false
		}
	}

	return true
}

// IsDuration tells whether given string is a valid duration format
// from the ISO 8601 ABNF as given in Appendix A of RFC 3339.
//
// see https://datatracker.ietf.org/doc/html/rfc3339#appendix-A, for details
func IsDuration(v any) bool {
	s, ok := v.(string)
	if !ok {
		return true
	}
	if len(s) == 0 || s[0] != 'P' {
		return false
	}
	s = s[1:]
	parseUnits := func() (units string, ok bool) {
		for len(s) > 0 && s[0] != 'T' {
			digits := false
			for len(s) != 0 {
				if s[0] < '0' || s[0] > '9' {
					break
				}
				digits = true
				s = s[1:]
			}
			if !digits || len(s) == 0 {
				return units, false
			}
			units += s[:1]
			s = s[1:]
		}
		return units, true
	}
	units, ok := parseUnits()
	if !ok {
		return false
	}
	if units == "W" {
		return len(s) == 0 // P_W
	}
	if len(units) > 0 {
		if !strings.Contains("YMD", units) { //nolint:gocritic
			return false
		}
		if len(s) == 0 {
			return true // "P" dur-date
		}
	}
	if len(s) == 0 || s[0] != 'T' {
		return false
	}
	s = s[1:]
	units, ok = parseUnits()
	return ok && len(s) == 0 && len(units) > 0 && strings.Contains("HMS", units) //nolint:gocritic
}

// IsPeriod tells whether given string is a valid period format
// from the ISO 8601 ABNF as given in Appendix A of RFC 3339.
//
// see https://datatracker.ietf.org/doc/html/rfc3339#appendix-A, for details
func IsPeriod(v any) bool {
	s, ok := v.(string)
	if !ok {
		return true
	}
	slash := strings.IndexByte(s, '/')
	if slash == -1 {
		return false
	}
	start, end := s[:slash], s[slash+1:]
	if IsDateTime(start) {
		return IsDateTime(end) || IsDuration(end)
	}
	return IsDuration(start) && IsDateTime(end)
}

// IsHostname tells whether given string is a valid representation
// for an Internet host name, as defined by RFC 1034 section 3.1 and
// RFC 1123 section 2.1.
//
// See https://en.wikipedia.org/wiki/Hostname#Restrictions_on_valid_host_names, for details.
func IsHostname(v any) bool {
	s, ok := v.(string)
	if !ok {
		return true
	}
	// entire hostname (including the delimiting dots but not a trailing dot) has a maximum of 253 ASCII characters
	s = strings.TrimSuffix(s, ".")
	if len(s) > 253 {
		return false
	}

	// Hostnames are composed of series of labels concatenated with dots, as are all domain names
	for _, label := range strings.Split(s, ".") {
		// Each label must be from 1 to 63 characters long
		if labelLen := len(label); labelLen < 1 || labelLen > 63 {
			return false
		}

		// labels must not start with a hyphen
		// RFC 1123 section 2.1: restriction on the first character
		// is relaxed to allow either a letter or a digit
		if first := s[0]; first == '-' {
			return false
		}

		// must not end with a hyphen
		if label[len(label)-1] == '-' {
			return false
		}

		// labels may contain only the ASCII letters 'a' through 'z' (in a case-insensitive manner),
		// the digits '0' through '9', and the hyphen ('-')
		for _, c := range label {
			if valid := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || (c == '-'); !valid {
				return false
			}
		}
	}

	return true
}

// IsEmail tells whether given string is a valid Internet email address
// as defined by RFC 5322, section 3.4.1.
//
// See https://en.wikipedia.org/wiki/Email_address, for details.
func IsEmail(v any) bool {
	s, ok := v.(string)
	if !ok {
		return true
	}
	// entire email address to be no more than 254 characters long
	if len(s) > 254 {
		return false
	}

	// email address is generally recognized as having two parts joined with an at-sign
	at := strings.LastIndexByte(s, '@')
	if at == -1 {
		return false
	}
	local := s[0:at]
	domain := s[at+1:]

	// local part may be up to 64 characters long
	if len(local) > 64 {
		return false
	}

	// domain if enclosed in brackets, must match an IP address
	if len(domain) >= 2 && domain[0] == '[' && domain[len(domain)-1] == ']' {
		ip := domain[1 : len(domain)-1]
		if strings.HasPrefix(ip, "IPv6:") {
			return IsIPV6(strings.TrimPrefix(ip, "IPv6:"))
		}
		return IsIPV4(ip)
	}

	// domain must match the requirements for a hostname
	if !IsHostname(domain) {
		return false
	}

	_, err := mail.ParseAddress(s)
	return err == nil
}

// IsIPV4 tells whether given string is a valid representation of an IPv4 address
// according to the "dotted-quad" ABNF syntax as defined in RFC 2673, section 3.2.
func IsIPV4(v any) bool {
	s, ok := v.(string)
	if !ok {
		return true
	}
	groups := strings.Split(s, ".")
	if len(groups) != 4 {
		return false
	}
	for _, group := range groups {
		n, err := strconv.Atoi(group)
		if err != nil {
			return false
		}
		if n < 0 || n > 255 {
			return false
		}
		if n != 0 && group[0] == '0' {
			return false // leading zeroes should be rejected, as they are treated as octals
		}
	}
	return true
}

// IsIPV6 tells whether given string is a valid representation of an IPv6 address
// as defined in RFC 2373, section 2.2.
func IsIPV6(v any) bool {
	s, ok := v.(string)
	if !ok {
		return true
	}
	if !strings.Contains(s, ":") {
		return false
	}
	return net.ParseIP(s) != nil
}

// IsURI tells whether given string is valid URI, according to RFC 3986.
func IsURI(v any) bool {
	s, ok := v.(string)
	if !ok {
		return true
	}
	u, err := urlParse(s)
	return err == nil && u.IsAbs()
}

func urlParse(s string) (*url.URL, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}

	// if hostname is ipv6, validate it
	hostname := u.Hostname()
	if strings.IndexByte(hostname, ':') != -1 {
		if strings.IndexByte(u.Host, '[') == -1 || strings.IndexByte(u.Host, ']') == -1 {
			return nil, ErrIPv6AddressFormat
		}
		if !IsIPV6(hostname) {
			return nil, ErrInvalidIPv6
		}
	}
	return u, nil
}

// IsURIReference tells whether given string is a valid URI Reference
// (either a URI or a relative-reference), according to RFC 3986.
func IsURIReference(v any) bool {
	s, ok := v.(string)
	if !ok {
		return true
	}
	_, err := urlParse(s)
	return err == nil && !strings.Contains(s, `\`)
}

// IsURITemplate tells whether given string is a valid URI Template
// according to RFC6570.
//
// Current implementation does minimal validation.
func IsURITemplate(v any) bool {
	s, ok := v.(string)
	if !ok {
		return true
	}
	u, err := urlParse(s)
	if err != nil {
		return false
	}
	for _, item := range strings.Split(u.RawPath, "/") {
		depth := 0
		for _, ch := range item {
			switch ch {
			case '{':
				depth++
				if depth != 1 {
					return false
				}
			case '}':
				depth--
				if depth != 0 {
					return false
				}
			}
		}
		if depth != 0 {
			return false
		}
	}
	return true
}

// IsJSONPointer tells whether given string is a valid JSON Pointer.
//
// Note: It returns false for JSON Pointer URI fragments.
func IsJSONPointer(v any) bool {
	s, ok := v.(string)
	if !ok {
		return true
	}
	// Empty string is a valid JSON Pointer (points to the whole document)
	if s == "" {
		return true
	}
	// Use jsonpointer library for validation
	return jsonpointer.Validate(s) == nil
}

// IsRelativeJSONPointer tells whether given string is a valid Relative JSON Pointer.
//
// see https://tools.ietf.org/html/draft-handrews-relative-json-pointer-01#section-3
func IsRelativeJSONPointer(v any) bool {
	s, ok := v.(string)
	if !ok {
		return true
	}
	if s == "" {
		return false
	}
	switch {
	case s[0] == '0':
		s = s[1:]
	case s[0] >= '0' && s[0] <= '9':
		for s != "" && s[0] >= '0' && s[0] <= '9' {
			s = s[1:]
		}
	default:
		return false
	}
	return s == "#" || IsJSONPointer(s)
}

// IsUUID tells whether given string is a valid uuid format
// as specified in RFC4122.
//
// see https://datatracker.ietf.org/doc/html/rfc4122#page-4, for details
func IsUUID(v any) bool {
	s, ok := v.(string)
	if !ok {
		return true
	}
	parseHex := func(n int) bool {
		for n > 0 {
			if len(s) == 0 {
				return false
			}
			hex := (s[0] >= '0' && s[0] <= '9') || (s[0] >= 'a' && s[0] <= 'f') || (s[0] >= 'A' && s[0] <= 'F')
			if !hex {
				return false
			}
			s = s[1:]
			n--
		}
		return true
	}
	groups := []int{8, 4, 4, 4, 12}
	for i, numDigits := range groups {
		if !parseHex(numDigits) {
			return false
		}
		if i == len(groups)-1 {
			break
		}
		if len(s) == 0 || s[0] != '-' {
			return false
		}
		s = s[1:]
	}
	return len(s) == 0
}

// IsRegex tells whether given string is a valid regex pattern
func IsRegex(v any) bool {
	pattern, ok := v.(string)
	if !ok {
		return true
	}

	// Attempt to compile the string as a regex pattern.
	_, err := regexp.Compile(pattern)

	// If there is no error, the pattern is a valid regex.
	return err == nil
}
