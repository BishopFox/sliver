package twilio

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/ttacon/libphonenumber"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// DefaultRegion is the region used to parse phone numbers without a leading
// international prefix.
var DefaultRegion = "US"

type PhoneNumber string

var ErrEmptyNumber = errors.New("twilio: The provided phone number was empty")

// NewPhoneNumber parses the given value as a phone number or returns an error
// if it cannot be parsed as one. If a phone number does not begin with a
// plus sign, we assume it's a national number in the region specified by
// DefaultRegion. Numbers are stored in E.164 format.
func NewPhoneNumber(pn string) (PhoneNumber, error) {
	if len(pn) == 0 {
		return "", ErrEmptyNumber
	}
	num, err := libphonenumber.Parse(pn, DefaultRegion)
	// Add some better error messages - the ones in libphonenumber are generic
	switch {
	case err == libphonenumber.ErrNotANumber:
		return "", fmt.Errorf("twilio: Invalid phone number: %s", pn)
	case err == libphonenumber.ErrInvalidCountryCode:
		return "", fmt.Errorf("twilio: Invalid country code for number: %s", pn)
	case err != nil:
		return "", err
	}
	return PhoneNumber(libphonenumber.Format(num, libphonenumber.E164)), nil
}

// Friendly returns a friendly international representation of the phone
// number, for example, "+14105554092" is returned as "+1 410-555-4092". If the
// phone number is not in E.164 format, we try to parse it as a US number. If
// we cannot parse it as a US number, it is returned as is.
func (pn PhoneNumber) Friendly() string {
	num, err := libphonenumber.Parse(string(pn), "US")
	if err != nil {
		return string(pn)
	}
	return libphonenumber.Format(num, libphonenumber.INTERNATIONAL)
}

// Local returns a friendly national representation of the phone number, for
// example, "+14105554092" is returned as "(410) 555-4092". If the phone number
// is not in E.164 format, we try to parse it as a US number. If we cannot
// parse it as a US number, it is returned as is.
func (pn PhoneNumber) Local() string {
	num, err := libphonenumber.Parse(string(pn), "US")
	if err != nil {
		return string(pn)
	}
	return libphonenumber.Format(num, libphonenumber.NATIONAL)
}

// A uintStr is sent back from Twilio as a str, but should be parsed as a uint.
type uintStr uint

type Segments uintStr
type NumMedia uintStr

func (seg *uintStr) UnmarshalJSON(b []byte) error {
	s := new(string)
	if err := json.Unmarshal(b, s); err != nil {
		return err
	}
	u, err := strconv.ParseUint(*s, 10, 64)
	if err != nil {
		return err
	}
	*seg = uintStr(u)
	return nil
}

func (seg *Segments) UnmarshalJSON(b []byte) (err error) {
	u := new(uintStr)
	if err = json.Unmarshal(b, u); err != nil {
		return
	}
	*seg = Segments(*u)
	return
}

func (seg Segments) MarshalJSON() ([]byte, error) {
	s := strconv.AppendUint(nil, uint64(seg), 10)
	return json.Marshal(string(s))
}

func (n *NumMedia) UnmarshalJSON(b []byte) (err error) {
	u := new(uintStr)
	if err = json.Unmarshal(b, u); err != nil {
		return
	}
	*n = NumMedia(*u)
	return
}

func (n NumMedia) MarshalJSON() ([]byte, error) {
	s := strconv.AppendUint(nil, uint64(n), 10)
	return json.Marshal(string(s))
}

// TwilioTime can parse a timestamp returned in the Twilio API and turn it into
// a valid Go Time struct.
type TwilioTime struct {
	Time  time.Time
	Valid bool
}

// NewTwilioTime returns a TwilioTime instance. val should be formatted using
// the TimeLayout.
func NewTwilioTime(val string) *TwilioTime {
	t, err := time.Parse(TimeLayout, val)
	if err == nil {
		return &TwilioTime{Time: t, Valid: true}
	} else {
		return &TwilioTime{}
	}
}

// Epoch is a time that predates the formation of the company (January 1,
// 2005). Use this for start filters when you don't want to filter old results.
var Epoch = time.Date(2005, 1, 1, 0, 0, 0, 0, time.UTC)

// HeatDeath is a sentinel time that should outdate the extinction of the
// company. Use this with GetXInRange calls when you don't want to specify an
// end date. Feel free to adjust this number in the year 5960 or so.
var HeatDeath = time.Date(6000, 1, 1, 0, 0, 0, 0, time.UTC)

// The reference time, as it appears in the Twilio API.
const TimeLayout = "Mon, 2 Jan 2006 15:04:05 -0700"

// Format expected by Twilio for searching date ranges. Monitor and other API's
// offer better date search filters
const APISearchLayout = "2006-01-02"

func (t *TwilioTime) UnmarshalJSON(b []byte) error {
	s := new(string)
	if err := json.Unmarshal(b, s); err != nil {
		return err
	}
	if s == nil || *s == "null" || *s == "" {
		t.Valid = false
		return nil
	}
	tim, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		tim, err = time.Parse(TimeLayout, *s)
		if err != nil {
			return err
		}
	}
	*t = TwilioTime{Time: tim, Valid: true}
	return nil
}

func (t TwilioTime) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return []byte("null"), nil
	}
	b, err := json.Marshal(t.Time)
	if err != nil {
		return []byte{}, err
	}
	return b, nil
}

var symbols = map[string]string{
	"USD": "$",
	"GBP": "£",
	"JPY": "¥",
	"MXN": "$",
	"CHF": "CHF",
	"CAD": "$",
	"CNY": "¥",
	"SGD": "$",
	"EUR": "€",
}

// Price flips the sign of the amount and prints it with a currency symbol for
// the given unit.
func price(unit string, amount string) string {
	if len(amount) == 0 {
		return amount
	}
	if amount[0] == '-' {
		amount = amount[1:]
	} else {
		amount = "-" + amount
	}
	for strings.Contains(amount, ".") && strings.HasSuffix(amount, "0") {
		amount = amount[:len(amount)-1]
	}
	amount = strings.TrimSuffix(amount, ".")
	unit = strings.ToUpper(unit)
	if sym, ok := symbols[unit]; ok {
		return sym + amount
	} else {
		if unit == "" {
			return amount
		}
		return unit + " " + amount
	}
}

// TwilioDuration represents a duration in the Twilio API that Twilio returns to
// us as a number of seconds.
type TwilioDuration time.Duration

func (td *TwilioDuration) UnmarshalJSON(b []byte) error {
	s := new(string)
	if err := json.Unmarshal(b, s); err != nil {
		return err
	}
	if *s == "null" || *s == "" {
		*td = 0
		return nil
	}
	i, err := strconv.ParseInt(*s, 10, 64)
	if err != nil {
		return err
	}
	*td = TwilioDuration(i) * TwilioDuration(time.Second)
	return nil
}

func (td TwilioDuration) MarshalJSON() ([]byte, error) {
	if td == 0 {
		return []byte("null"), nil
	}
	s := strconv.AppendInt(nil, int64(td/TwilioDuration(time.Second)), 10)
	return json.Marshal(string(s))
}

func (td TwilioDuration) String() string {
	return time.Duration(td).String()
}

// TwilioDurationMS represents a duration in the Twilio API that Twilio returns
// to us as a number of milliseconds.
type TwilioDurationMS time.Duration

func (tdm *TwilioDurationMS) UnmarshalJSON(b []byte) error {
	s := new(string)
	if err := json.Unmarshal(b, s); err != nil {
		return err
	}
	if *s == "null" || *s == "" {
		*tdm = 0
		return nil
	}
	i, err := strconv.ParseInt(*s, 10, 64)
	if err != nil {
		return err
	}
	*tdm = TwilioDurationMS(i) * TwilioDurationMS(time.Millisecond)
	return nil
}

func (t TwilioDurationMS) String() string {
	return time.Duration(t).String()
}

type AnsweredBy string

const AnsweredByHuman = AnsweredBy("human")
const AnsweredByMachine = AnsweredBy("machine")

type NullAnsweredBy struct {
	Valid      bool
	AnsweredBy AnsweredBy
}

// The status of a resource ("accepted", "queued", etc).
// For more information, see
//
// https://www.twilio.com/docs/api/rest/message
// https://www.twilio.com/docs/api/fax/rest/faxes#fax-status-values
type Status string

func (s Status) Friendly() string {
	switch s {
	case StatusInProgress:
		return "In Progress"
	case StatusNoAnswer:
		return "No Answer"
	default:
		return cases.Title(language.AmericanEnglish).String(string(s))
	}
}

// Values has the methods of url.Values, but can decode JSON from the
// response_headers field of an Alert.
type Values struct {
	url.Values
}

func (h *Values) UnmarshalJSON(b []byte) error {
	s := new(string)
	if err := json.Unmarshal(b, s); err != nil {
		return err
	}
	vals, err := url.ParseQuery(*s)
	if err != nil {
		return err
	}
	*h = Values{url.Values{}}
	for k, arr := range vals {
		for _, val := range arr {
			h.Add(k, val)
		}
	}
	return nil
}

const StatusAccepted = Status("accepted")
const StatusSent = Status("sent")
const StatusUndelivered = Status("undelivered")

// Call statuses

const StatusCompleted = Status("completed")
const StatusInProgress = Status("in-progress")
const StatusRinging = Status("ringing")

// Fax statuses

const StatusProcessing = Status("processing")

// WhatsApp statuses

const StatusRead = Status("read")

// Shared statuses

const StatusActive = Status("active")
const StatusBusy = Status("busy")
const StatusCanceled = Status("canceled")
const StatusClosed = Status("closed")
const StatusDelivered = Status("delivered")
const StatusFailed = Status("failed")
const StatusNoAnswer = Status("no-answer")
const StatusQueued = Status("queued")
const StatusReceiving = Status("receiving")
const StatusReceived = Status("received")
const StatusSending = Status("sending")
const StatusSuspended = Status("suspended")

// A log level returned for an Alert.
type LogLevel string

const LogLevelError = LogLevel("error")
const LogLevelWarning = LogLevel("warning")
const LogLevelNotice = LogLevel("notice")
const LogLevelDebug = LogLevel("debug")

func (l LogLevel) Friendly() string {
	return capitalize(string(l))
}

// capitalize the first letter in s
func capitalize(s string) string {
	r, l := utf8.DecodeRuneInString(s)
	b := make([]byte, l)
	utf8.EncodeRune(b, unicode.ToTitle(r))
	return strings.Join([]string{string(b), s[l:]}, "")
}

// types of video room
const RoomType = "group"
const RoomTypePeerToPeer = "peer-to-peer"
