// Copyright 2018 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package messaging contains functions for sending messages and managing
// device subscriptions with Firebase Cloud Messaging (FCM).
package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"firebase.google.com/go/v4/internal"
	"google.golang.org/api/transport"
)

const (
	defaultMessagingEndpoint = "https://fcm.googleapis.com/v1"
	defaultBatchEndpoint     = "https://fcm.googleapis.com/batch"

	firebaseClientHeader   = "X-Firebase-Client"
	apiFormatVersionHeader = "X-GOOG-API-FORMAT-VERSION"
	apiFormatVersion       = "2"

	apnsAuthError       = "APNS_AUTH_ERROR"
	internalError       = "INTERNAL"
	thirdPartyAuthError = "THIRD_PARTY_AUTH_ERROR"
	invalidArgument     = "INVALID_ARGUMENT"
	quotaExceeded       = "QUOTA_EXCEEDED"
	senderIDMismatch    = "SENDER_ID_MISMATCH"
	unregistered        = "UNREGISTERED"
	unavailable         = "UNAVAILABLE"

	rfc3339Zulu = "2006-01-02T15:04:05.000000000Z"
)

var (
	topicNamePattern = regexp.MustCompile("^(/topics/)?(private/)?[a-zA-Z0-9-_.~%]+$")
)

// Message to be sent via Firebase Cloud Messaging.
//
// Message contains payload data, recipient information and platform-specific configuration
// options. A Message must specify exactly one of Token, Topic or Condition fields. Apart from
// that a Message may specify any combination of Data, Notification, Android, Webpush and APNS
// fields. See https://firebase.google.com/docs/reference/fcm/rest/v1/projects.messages for more
// details on how the backend FCM servers handle different message parameters.
type Message struct {
	Data         map[string]string `json:"data,omitempty"`
	Notification *Notification     `json:"notification,omitempty"`
	Android      *AndroidConfig    `json:"android,omitempty"`
	Webpush      *WebpushConfig    `json:"webpush,omitempty"`
	APNS         *APNSConfig       `json:"apns,omitempty"`
	FCMOptions   *FCMOptions       `json:"fcm_options,omitempty"`
	Token        string            `json:"token,omitempty"`
	Topic        string            `json:"-"`
	Condition    string            `json:"condition,omitempty"`
}

// MarshalJSON marshals a Message into JSON (for internal use only).
func (m *Message) MarshalJSON() ([]byte, error) {
	// Create a new type to prevent infinite recursion. We use this technique whenever it is needed
	// to customize how a subset of the fields in a struct should be serialized.
	type messageInternal Message
	temp := &struct {
		BareTopic string `json:"topic,omitempty"`
		*messageInternal
	}{
		BareTopic:       strings.TrimPrefix(m.Topic, "/topics/"),
		messageInternal: (*messageInternal)(m),
	}
	return json.Marshal(temp)
}

// UnmarshalJSON unmarshals a JSON string into a Message (for internal use only).
func (m *Message) UnmarshalJSON(b []byte) error {
	type messageInternal Message
	s := struct {
		BareTopic string `json:"topic,omitempty"`
		*messageInternal
	}{
		messageInternal: (*messageInternal)(m),
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	m.Topic = s.BareTopic
	return nil
}

// Notification is the basic notification template to use across all platforms.
type Notification struct {
	Title    string `json:"title,omitempty"`
	Body     string `json:"body,omitempty"`
	ImageURL string `json:"image,omitempty"`
}

// AndroidConfig contains messaging options specific to the Android platform.
type AndroidConfig struct {
	CollapseKey           string               `json:"collapse_key,omitempty"`
	Priority              string               `json:"priority,omitempty"` // one of "normal" or "high"
	TTL                   *time.Duration       `json:"-"`
	RestrictedPackageName string               `json:"restricted_package_name,omitempty"`
	Data                  map[string]string    `json:"data,omitempty"` // if specified, overrides the Data field on Message type
	Notification          *AndroidNotification `json:"notification,omitempty"`
	FCMOptions            *AndroidFCMOptions   `json:"fcm_options,omitempty"`
	DirectBootOK          bool                 `json:"direct_boot_ok,omitempty"`
}

// MarshalJSON marshals an AndroidConfig into JSON (for internal use only).
func (a *AndroidConfig) MarshalJSON() ([]byte, error) {
	var ttl string
	if a.TTL != nil {
		ttl = durationToString(*a.TTL)
	}

	type androidInternal AndroidConfig
	temp := &struct {
		TTL string `json:"ttl,omitempty"`
		*androidInternal
	}{
		TTL:             ttl,
		androidInternal: (*androidInternal)(a),
	}
	return json.Marshal(temp)
}

// UnmarshalJSON unmarshals a JSON string into an AndroidConfig (for internal use only).
func (a *AndroidConfig) UnmarshalJSON(b []byte) error {
	type androidInternal AndroidConfig
	temp := struct {
		TTL string `json:"ttl,omitempty"`
		*androidInternal
	}{
		androidInternal: (*androidInternal)(a),
	}
	if err := json.Unmarshal(b, &temp); err != nil {
		return err
	}
	if temp.TTL != "" {
		ttl, err := stringToDuration(temp.TTL)
		if err != nil {
			return err
		}
		a.TTL = &ttl
	}
	return nil
}

// AndroidNotification is a notification to send to Android devices.
type AndroidNotification struct {
	Title                 string                        `json:"title,omitempty"` // if specified, overrides the Title field of the Notification type
	Body                  string                        `json:"body,omitempty"`  // if specified, overrides the Body field of the Notification type
	Icon                  string                        `json:"icon,omitempty"`
	Color                 string                        `json:"color,omitempty"` // notification color in #RRGGBB format
	Sound                 string                        `json:"sound,omitempty"`
	Tag                   string                        `json:"tag,omitempty"`
	ClickAction           string                        `json:"click_action,omitempty"`
	BodyLocKey            string                        `json:"body_loc_key,omitempty"`
	BodyLocArgs           []string                      `json:"body_loc_args,omitempty"`
	TitleLocKey           string                        `json:"title_loc_key,omitempty"`
	TitleLocArgs          []string                      `json:"title_loc_args,omitempty"`
	ChannelID             string                        `json:"channel_id,omitempty"`
	ImageURL              string                        `json:"image,omitempty"`
	Ticker                string                        `json:"ticker,omitempty"`
	Sticky                bool                          `json:"sticky,omitempty"`
	EventTimestamp        *time.Time                    `json:"-"`
	LocalOnly             bool                          `json:"local_only,omitempty"`
	Priority              AndroidNotificationPriority   `json:"-"`
	VibrateTimingMillis   []int64                       `json:"-"`
	DefaultVibrateTimings bool                          `json:"default_vibrate_timings,omitempty"`
	DefaultSound          bool                          `json:"default_sound,omitempty"`
	LightSettings         *LightSettings                `json:"light_settings,omitempty"`
	DefaultLightSettings  bool                          `json:"default_light_settings,omitempty"`
	Visibility            AndroidNotificationVisibility `json:"-"`
	NotificationCount     *int                          `json:"notification_count,omitempty"`
	Proxy                 AndroidNotificationProxy      `json:"-"`
}

// MarshalJSON marshals an AndroidNotification into JSON (for internal use only).
func (a *AndroidNotification) MarshalJSON() ([]byte, error) {
	var priority string
	if a.Priority != priorityUnspecified {
		priorities := map[AndroidNotificationPriority]string{
			PriorityMin:     "PRIORITY_MIN",
			PriorityLow:     "PRIORITY_LOW",
			PriorityDefault: "PRIORITY_DEFAULT",
			PriorityHigh:    "PRIORITY_HIGH",
			PriorityMax:     "PRIORITY_MAX",
		}
		priority, _ = priorities[a.Priority]
	}

	var visibility string
	if a.Visibility != visibilityUnspecified {
		visibilities := map[AndroidNotificationVisibility]string{
			VisibilityPrivate: "PRIVATE",
			VisibilityPublic:  "PUBLIC",
			VisibilitySecret:  "SECRET",
		}
		visibility, _ = visibilities[a.Visibility]
	}

	var proxy string
	if a.Proxy != proxyUnspecified {
		proxies := map[AndroidNotificationProxy]string{
			ProxyAllow:             "ALLOW",
			ProxyDeny:              "DENY",
			ProxyIfPriorityLowered: "IF_PRIORITY_LOWERED",
		}
		proxy, _ = proxies[a.Proxy]
	}

	var timestamp string
	if a.EventTimestamp != nil {
		timestamp = a.EventTimestamp.UTC().Format(rfc3339Zulu)
	}

	var vibTimings []string
	for _, t := range a.VibrateTimingMillis {
		vibTimings = append(vibTimings, durationToString(time.Duration(t)*time.Millisecond))
	}

	type androidInternal AndroidNotification
	temp := &struct {
		EventTimestamp string   `json:"event_time,omitempty"`
		Priority       string   `json:"notification_priority,omitempty"`
		Visibility     string   `json:"visibility,omitempty"`
		Proxy          string   `json:"proxy,omitempty"`
		VibrateTimings []string `json:"vibrate_timings,omitempty"`
		*androidInternal
	}{
		EventTimestamp:  timestamp,
		Priority:        priority,
		Visibility:      visibility,
		Proxy:           proxy,
		VibrateTimings:  vibTimings,
		androidInternal: (*androidInternal)(a),
	}
	return json.Marshal(temp)
}

// UnmarshalJSON unmarshals a JSON string into an AndroidNotification (for internal use only).
func (a *AndroidNotification) UnmarshalJSON(b []byte) error {
	type androidInternal AndroidNotification
	temp := struct {
		EventTimestamp string   `json:"event_time,omitempty"`
		Priority       string   `json:"notification_priority,omitempty"`
		Visibility     string   `json:"visibility,omitempty"`
		Proxy          string   `json:"proxy,omitempty"`
		VibrateTimings []string `json:"vibrate_timings,omitempty"`
		*androidInternal
	}{
		androidInternal: (*androidInternal)(a),
	}
	if err := json.Unmarshal(b, &temp); err != nil {
		return err
	}

	if temp.Priority != "" {
		priorities := map[string]AndroidNotificationPriority{
			"PRIORITY_MIN":     PriorityMin,
			"PRIORITY_LOW":     PriorityLow,
			"PRIORITY_DEFAULT": PriorityDefault,
			"PRIORITY_HIGH":    PriorityHigh,
			"PRIORITY_MAX":     PriorityMax,
		}
		if prio, ok := priorities[temp.Priority]; ok {
			a.Priority = prio
		} else {
			return fmt.Errorf("unknown priority value: %q", temp.Priority)
		}
	}

	if temp.Visibility != "" {
		visibilities := map[string]AndroidNotificationVisibility{
			"PRIVATE": VisibilityPrivate,
			"PUBLIC":  VisibilityPublic,
			"SECRET":  VisibilitySecret,
		}
		if vis, ok := visibilities[temp.Visibility]; ok {
			a.Visibility = vis
		} else {
			return fmt.Errorf("unknown visibility value: %q", temp.Visibility)
		}
	}

	if temp.Proxy != "" {
		proxies := map[string]AndroidNotificationProxy{
			"ALLOW":               ProxyAllow,
			"DENY":                ProxyDeny,
			"IF_PRIORITY_LOWERED": ProxyIfPriorityLowered,
		}
		if prox, ok := proxies[temp.Proxy]; ok {
			a.Proxy = prox
		} else {
			return fmt.Errorf("unknown proxy value: %q", temp.Proxy)
		}
	}

	if temp.EventTimestamp != "" {
		ts, err := time.Parse(rfc3339Zulu, temp.EventTimestamp)
		if err != nil {
			return err
		}

		a.EventTimestamp = &ts
	}

	var vibTimings []int64
	for _, t := range temp.VibrateTimings {
		vibTime, err := stringToDuration(t)
		if err != nil {
			return err
		}

		millis := int64(vibTime / time.Millisecond)
		vibTimings = append(vibTimings, millis)
	}
	a.VibrateTimingMillis = vibTimings
	return nil
}

// AndroidNotificationPriority represents the priority levels of a notification.
type AndroidNotificationPriority int

const (
	priorityUnspecified AndroidNotificationPriority = iota

	// PriorityMin is the lowest notification priority. Notifications with this priority might not
	// be shown to the user except under special circumstances, such as detailed notification logs.
	PriorityMin

	// PriorityLow is a lower notification priority. The UI may choose to show the notifications
	// smaller, or at a different position in the list, compared with notifications with PriorityDefault.
	PriorityLow

	// PriorityDefault is the default notification priority. If the application does not prioritize
	// its own notifications, use this value for all notifications.
	PriorityDefault

	// PriorityHigh is a higher notification priority. Use this for more important
	// notifications or alerts. The UI may choose to show these notifications larger, or at a
	// different position in the notification lists, compared with notifications with PriorityDefault.
	PriorityHigh

	// PriorityMax is the highest notification priority. Use this for the application's most
	// important items that require the user's prompt attention or input.
	PriorityMax
)

// AndroidNotificationVisibility represents the different visibility levels of a notification.
type AndroidNotificationVisibility int

const (
	visibilityUnspecified AndroidNotificationVisibility = iota

	// VisibilityPrivate shows this notification on all lockscreens, but conceal sensitive or
	// private information on secure lockscreens.
	VisibilityPrivate

	// VisibilityPublic shows this notification in its entirety on all lockscreens.
	VisibilityPublic

	// VisibilitySecret does not reveal any part of this notification on a secure lockscreen.
	VisibilitySecret
)

// AndroidNotificationProxy to control when a notification may be proxied.
type AndroidNotificationProxy int

const (
	proxyUnspecified AndroidNotificationProxy = iota

	// ProxyAllow tries to proxy this notification.
	ProxyAllow

	// ProxyDeny does not proxy this notification.
	ProxyDeny

	// ProxyIfPriorityLowered only tries to proxy this notification if its AndroidConfig's Priority was
	// lowered from high to normal on the device.
	ProxyIfPriorityLowered
)

// LightSettings to control notification LED.
type LightSettings struct {
	Color                  string
	LightOnDurationMillis  int64
	LightOffDurationMillis int64
}

// MarshalJSON marshals an LightSettings into JSON (for internal use only).
func (l *LightSettings) MarshalJSON() ([]byte, error) {
	clr, err := newColor(l.Color)
	if err != nil {
		return nil, err
	}

	temp := struct {
		Color            *color `json:"color"`
		LightOnDuration  string `json:"light_on_duration"`
		LightOffDuration string `json:"light_off_duration"`
	}{
		Color:            clr,
		LightOnDuration:  durationToString(time.Duration(l.LightOnDurationMillis) * time.Millisecond),
		LightOffDuration: durationToString(time.Duration(l.LightOffDurationMillis) * time.Millisecond),
	}
	return json.Marshal(temp)
}

// UnmarshalJSON unmarshals a JSON string into an LightSettings (for internal use only).
func (l *LightSettings) UnmarshalJSON(b []byte) error {
	temp := struct {
		Color            *color `json:"color"`
		LightOnDuration  string `json:"light_on_duration"`
		LightOffDuration string `json:"light_off_duration"`
	}{}
	if err := json.Unmarshal(b, &temp); err != nil {
		return err
	}

	on, err := stringToDuration(temp.LightOnDuration)
	if err != nil {
		return err
	}

	off, err := stringToDuration(temp.LightOffDuration)
	if err != nil {
		return err
	}

	l.Color = temp.Color.toString()
	l.LightOnDurationMillis = int64(on / time.Millisecond)
	l.LightOffDurationMillis = int64(off / time.Millisecond)
	return nil
}

func durationToString(ms time.Duration) string {
	seconds := int64(ms / time.Second)
	nanos := int64((ms - time.Duration(seconds)*time.Second) / time.Nanosecond)
	if nanos > 0 {
		return fmt.Sprintf("%d.%09ds", seconds, nanos)
	}
	return fmt.Sprintf("%ds", seconds)
}

func stringToDuration(s string) (time.Duration, error) {
	segments := strings.Split(strings.TrimSuffix(s, "s"), ".")
	if len(segments) != 1 && len(segments) != 2 {
		return 0, fmt.Errorf("incorrect number of segments in ttl: %q", s)
	}

	seconds, err := strconv.ParseInt(segments[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse %s: %v", s, err)
	}

	ttl := time.Duration(seconds) * time.Second
	if len(segments) == 2 {
		nanos, err := strconv.ParseInt(strings.TrimLeft(segments[1], "0"), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse %s: %v", s, err)
		}
		ttl += time.Duration(nanos) * time.Nanosecond
	}

	return ttl, nil
}

type color struct {
	Red   float64 `json:"red"`
	Green float64 `json:"green"`
	Blue  float64 `json:"blue"`
	Alpha float64 `json:"alpha"`
}

func newColor(clr string) (*color, error) {
	red, err := strconv.ParseInt(clr[1:3], 16, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %v", clr, err)
	}

	green, err := strconv.ParseInt(clr[3:5], 16, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %v", clr, err)
	}

	blue, err := strconv.ParseInt(clr[5:7], 16, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %v", clr, err)
	}

	alpha := int64(255)
	if len(clr) == 9 {
		alpha, err = strconv.ParseInt(clr[7:9], 16, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %v", clr, err)
		}
	}

	return &color{
		Red:   float64(red) / 255.0,
		Green: float64(green) / 255.0,
		Blue:  float64(blue) / 255.0,
		Alpha: float64(alpha) / 255.0,
	}, nil
}

func (c *color) toString() string {
	red := int(c.Red * 255.0)
	green := int(c.Green * 255.0)
	blue := int(c.Blue * 255.0)
	alpha := int(c.Alpha * 255.0)
	if alpha == 255 {
		return fmt.Sprintf("#%X%X%X", red, green, blue)
	}
	return fmt.Sprintf("#%X%X%X%X", red, green, blue, alpha)
}

// AndroidFCMOptions contains additional options for features provided by the FCM Android SDK.
type AndroidFCMOptions struct {
	AnalyticsLabel string `json:"analytics_label,omitempty"`
}

// WebpushConfig contains messaging options specific to the WebPush protocol.
//
// See https://tools.ietf.org/html/rfc8030#section-5 for additional details, and supported
// headers.
type WebpushConfig struct {
	Headers      map[string]string    `json:"headers,omitempty"`
	Data         map[string]string    `json:"data,omitempty"`
	Notification *WebpushNotification `json:"notification,omitempty"`
	FCMOptions   *WebpushFCMOptions   `json:"fcm_options,omitempty"`
}

// WebpushNotificationAction represents an action that can be performed upon receiving a WebPush notification.
type WebpushNotificationAction struct {
	Action string `json:"action,omitempty"`
	Title  string `json:"title,omitempty"`
	Icon   string `json:"icon,omitempty"`
}

// WebpushNotification is a notification to send via WebPush protocol.
//
// See https://developer.mozilla.org/en-US/docs/Web/API/notification/Notification for additional
// details.
type WebpushNotification struct {
	Actions            []*WebpushNotificationAction `json:"actions,omitempty"`
	Title              string                       `json:"title,omitempty"` // if specified, overrides the Title field of the Notification type
	Body               string                       `json:"body,omitempty"`  // if specified, overrides the Body field of the Notification type
	Icon               string                       `json:"icon,omitempty"`
	Badge              string                       `json:"badge,omitempty"`
	Direction          string                       `json:"dir,omitempty"` // one of 'ltr' or 'rtl'
	Data               interface{}                  `json:"data,omitempty"`
	Image              string                       `json:"image,omitempty"`
	Language           string                       `json:"lang,omitempty"`
	Renotify           bool                         `json:"renotify,omitempty"`
	RequireInteraction bool                         `json:"requireInteraction,omitempty"`
	Silent             bool                         `json:"silent,omitempty"`
	Tag                string                       `json:"tag,omitempty"`
	TimestampMillis    *int64                       `json:"timestamp,omitempty"`
	Vibrate            []int                        `json:"vibrate,omitempty"`
	CustomData         map[string]interface{}
}

// standardFields creates a map containing all the fields except the custom data.
//
// We implement a standardFields function whenever we want to add custom and arbitrary
// fields to an object during its serialization. This helper function also comes in
// handy during validation of the message (to detect duplicate specifications of
// fields), and also during deserialization.
func (n *WebpushNotification) standardFields() map[string]interface{} {
	m := make(map[string]interface{})
	addNonEmpty := func(key, value string) {
		if value != "" {
			m[key] = value
		}
	}
	addTrue := func(key string, value bool) {
		if value {
			m[key] = value
		}
	}
	if len(n.Actions) > 0 {
		m["actions"] = n.Actions
	}
	addNonEmpty("title", n.Title)
	addNonEmpty("body", n.Body)
	addNonEmpty("icon", n.Icon)
	addNonEmpty("badge", n.Badge)
	addNonEmpty("dir", n.Direction)
	addNonEmpty("image", n.Image)
	addNonEmpty("lang", n.Language)
	addTrue("renotify", n.Renotify)
	addTrue("requireInteraction", n.RequireInteraction)
	addTrue("silent", n.Silent)
	addNonEmpty("tag", n.Tag)
	if n.Data != nil {
		m["data"] = n.Data
	}
	if n.TimestampMillis != nil {
		m["timestamp"] = *n.TimestampMillis
	}
	if len(n.Vibrate) > 0 {
		m["vibrate"] = n.Vibrate
	}
	return m
}

// MarshalJSON marshals a WebpushNotification into JSON (for internal use only).
func (n *WebpushNotification) MarshalJSON() ([]byte, error) {
	m := n.standardFields()
	for k, v := range n.CustomData {
		m[k] = v
	}
	return json.Marshal(m)
}

// UnmarshalJSON unmarshals a JSON string into a WebpushNotification (for internal use only).
func (n *WebpushNotification) UnmarshalJSON(b []byte) error {
	type webpushNotificationInternal WebpushNotification
	var temp = (*webpushNotificationInternal)(n)
	if err := json.Unmarshal(b, temp); err != nil {
		return err
	}
	allFields := make(map[string]interface{})
	if err := json.Unmarshal(b, &allFields); err != nil {
		return err
	}
	for k := range n.standardFields() {
		delete(allFields, k)
	}
	if len(allFields) > 0 {
		n.CustomData = allFields
	}
	return nil
}

// WebpushFCMOptions contains additional options for features provided by the FCM web SDK.
type WebpushFCMOptions struct {
	Link string `json:"link,omitempty"`
}

// APNSConfig contains messaging options specific to the Apple Push Notification Service (APNS).
//
// See https://developer.apple.com/library/content/documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/CommunicatingwithAPNs.html
// for more details on supported headers and payload keys.
type APNSConfig struct {
	Headers           map[string]string `json:"headers,omitempty"`
	Payload           *APNSPayload      `json:"payload,omitempty"`
	FCMOptions        *APNSFCMOptions   `json:"fcm_options,omitempty"`
	LiveActivityToken string            `json:"live_activity_token,omitempty"`
}

// APNSPayload is the payload that can be included in an APNS message.
//
// The payload mainly consists of the aps dictionary. Additionally it may contain arbitrary
// key-values pairs as custom data fields.
//
// See https://developer.apple.com/library/content/documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/PayloadKeyReference.html
// for a full list of supported payload fields.
type APNSPayload struct {
	Aps        *Aps                   `json:"aps,omitempty"`
	CustomData map[string]interface{} `json:"-"`
}

// standardFields creates a map containing all the fields except the custom data.
func (p *APNSPayload) standardFields() map[string]interface{} {
	return map[string]interface{}{"aps": p.Aps}
}

// MarshalJSON marshals an APNSPayload into JSON (for internal use only).
func (p *APNSPayload) MarshalJSON() ([]byte, error) {
	m := p.standardFields()
	for k, v := range p.CustomData {
		m[k] = v
	}
	return json.Marshal(m)
}

// UnmarshalJSON unmarshals a JSON string into an APNSPayload (for internal use only).
func (p *APNSPayload) UnmarshalJSON(b []byte) error {
	type apnsPayloadInternal APNSPayload
	var temp = (*apnsPayloadInternal)(p)
	if err := json.Unmarshal(b, temp); err != nil {
		return err
	}
	allFields := make(map[string]interface{})
	if err := json.Unmarshal(b, &allFields); err != nil {
		return err
	}
	for k := range p.standardFields() {
		delete(allFields, k)
	}
	if len(allFields) > 0 {
		p.CustomData = allFields
	}
	return nil
}

// Aps represents the aps dictionary that may be included in an APNSPayload.
//
// Alert may be specified as a string (via the AlertString field), or as a struct (via the Alert
// field).
type Aps struct {
	AlertString      string                 `json:"-"`
	Alert            *ApsAlert              `json:"-"`
	Badge            *int                   `json:"badge,omitempty"`
	Sound            string                 `json:"-"`
	CriticalSound    *CriticalSound         `json:"-"`
	ContentAvailable bool                   `json:"-"`
	MutableContent   bool                   `json:"-"`
	Category         string                 `json:"category,omitempty"`
	ThreadID         string                 `json:"thread-id,omitempty"`
	CustomData       map[string]interface{} `json:"-"`
}

// standardFields creates a map containing all the fields except the custom data.
func (a *Aps) standardFields() map[string]interface{} {
	m := make(map[string]interface{})
	if a.Alert != nil {
		m["alert"] = a.Alert
	} else if a.AlertString != "" {
		m["alert"] = a.AlertString
	}
	if a.ContentAvailable {
		m["content-available"] = 1
	}
	if a.MutableContent {
		m["mutable-content"] = 1
	}
	if a.Badge != nil {
		m["badge"] = *a.Badge
	}
	if a.CriticalSound != nil {
		m["sound"] = a.CriticalSound
	} else if a.Sound != "" {
		m["sound"] = a.Sound
	}
	if a.Category != "" {
		m["category"] = a.Category
	}
	if a.ThreadID != "" {
		m["thread-id"] = a.ThreadID
	}
	return m
}

// MarshalJSON marshals an Aps into JSON (for internal use only).
func (a *Aps) MarshalJSON() ([]byte, error) {
	m := a.standardFields()
	for k, v := range a.CustomData {
		m[k] = v
	}
	return json.Marshal(m)
}

// UnmarshalJSON unmarshals a JSON string into an Aps (for internal use only).
func (a *Aps) UnmarshalJSON(b []byte) error {
	type apsInternal Aps
	temp := struct {
		AlertObject         *json.RawMessage `json:"alert,omitempty"`
		SoundObject         *json.RawMessage `json:"sound,omitempty"`
		ContentAvailableInt int              `json:"content-available,omitempty"`
		MutableContentInt   int              `json:"mutable-content,omitempty"`
		*apsInternal
	}{
		apsInternal: (*apsInternal)(a),
	}
	if err := json.Unmarshal(b, &temp); err != nil {
		return err
	}
	a.ContentAvailable = (temp.ContentAvailableInt == 1)
	a.MutableContent = (temp.MutableContentInt == 1)
	if temp.AlertObject != nil {
		if err := json.Unmarshal(*temp.AlertObject, &a.Alert); err != nil {
			a.Alert = nil
			if err := json.Unmarshal(*temp.AlertObject, &a.AlertString); err != nil {
				return fmt.Errorf("failed to unmarshal alert as a struct or a string: %v", err)
			}
		}
	}
	if temp.SoundObject != nil {
		if err := json.Unmarshal(*temp.SoundObject, &a.CriticalSound); err != nil {
			a.CriticalSound = nil
			if err := json.Unmarshal(*temp.SoundObject, &a.Sound); err != nil {
				return fmt.Errorf("failed to unmarshal sound as a struct or a string")
			}
		}
	}

	allFields := make(map[string]interface{})
	if err := json.Unmarshal(b, &allFields); err != nil {
		return err
	}
	for k := range a.standardFields() {
		delete(allFields, k)
	}
	if len(allFields) > 0 {
		a.CustomData = allFields
	}
	return nil
}

// CriticalSound is the sound payload that can be included in an Aps.
type CriticalSound struct {
	Critical bool    `json:"-"`
	Name     string  `json:"name,omitempty"`
	Volume   float64 `json:"volume,omitempty"`
}

// MarshalJSON marshals a CriticalSound into JSON (for internal use only).
func (cs *CriticalSound) MarshalJSON() ([]byte, error) {
	type criticalSoundInternal CriticalSound
	temp := struct {
		CriticalInt int `json:"critical,omitempty"`
		*criticalSoundInternal
	}{
		criticalSoundInternal: (*criticalSoundInternal)(cs),
	}
	if cs.Critical {
		temp.CriticalInt = 1
	}
	return json.Marshal(temp)
}

// UnmarshalJSON unmarshals a JSON string into a CriticalSound (for internal use only).
func (cs *CriticalSound) UnmarshalJSON(b []byte) error {
	type criticalSoundInternal CriticalSound
	temp := struct {
		CriticalInt int `json:"critical,omitempty"`
		*criticalSoundInternal
	}{
		criticalSoundInternal: (*criticalSoundInternal)(cs),
	}
	if err := json.Unmarshal(b, &temp); err != nil {
		return err
	}
	cs.Critical = (temp.CriticalInt == 1)
	return nil
}

// ApsAlert is the alert payload that can be included in an Aps.
//
// See https://developer.apple.com/library/content/documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/PayloadKeyReference.html
// for supported fields.
type ApsAlert struct {
	Title           string   `json:"title,omitempty"` // if specified, overrides the Title field of the Notification type
	SubTitle        string   `json:"subtitle,omitempty"`
	Body            string   `json:"body,omitempty"` // if specified, overrides the Body field of the Notification type
	LocKey          string   `json:"loc-key,omitempty"`
	LocArgs         []string `json:"loc-args,omitempty"`
	TitleLocKey     string   `json:"title-loc-key,omitempty"`
	TitleLocArgs    []string `json:"title-loc-args,omitempty"`
	SubTitleLocKey  string   `json:"subtitle-loc-key,omitempty"`
	SubTitleLocArgs []string `json:"subtitle-loc-args,omitempty"`
	ActionLocKey    string   `json:"action-loc-key,omitempty"`
	LaunchImage     string   `json:"launch-image,omitempty"`
}

// APNSFCMOptions contains additional options for features provided by the FCM Aps SDK.
type APNSFCMOptions struct {
	AnalyticsLabel string `json:"analytics_label,omitempty"`
	ImageURL       string `json:"image,omitempty"`
}

// FCMOptions contains additional options to use across all platforms.
type FCMOptions struct {
	AnalyticsLabel string `json:"analytics_label,omitempty"`
}

// ErrorInfo is a topic management error.
type ErrorInfo struct {
	Index  int
	Reason string
}

// Client is the interface for the Firebase Cloud Messaging (FCM) service.
type Client struct {
	*fcmClient
	*iidClient
}

// NewClient creates a new instance of the Firebase Cloud Messaging Client.
//
// This function can only be invoked from within the SDK. Client applications should access the
// the messaging service through firebase.App.
func NewClient(ctx context.Context, c *internal.MessagingConfig) (*Client, error) {
	if c.ProjectID == "" {
		return nil, errors.New("project ID is required to access Firebase Cloud Messaging client")
	}

	hc, messagingEndpoint, err := transport.NewHTTPClient(ctx, c.Opts...)
	if err != nil {
		return nil, err
	}

	batchEndpoint := messagingEndpoint

	if messagingEndpoint == "" {
		messagingEndpoint = defaultMessagingEndpoint
		batchEndpoint = defaultBatchEndpoint
	}

	return &Client{
		fcmClient: newFCMClient(hc, c, messagingEndpoint, batchEndpoint),
		iidClient: newIIDClient(hc, c),
	}, nil
}

type fcmClient struct {
	fcmEndpoint   string
	batchEndpoint string
	project       string
	version       string
	httpClient    *internal.HTTPClient
}

func newFCMClient(hc *http.Client, conf *internal.MessagingConfig, messagingEndpoint string, batchEndpoint string) *fcmClient {
	client := internal.WithDefaultRetryConfig(hc)
	client.CreateErrFn = handleFCMError

	version := fmt.Sprintf("fire-admin-go/%s", conf.Version)
	client.Opts = []internal.HTTPOption{
		internal.WithHeader(apiFormatVersionHeader, apiFormatVersion),
		internal.WithHeader(firebaseClientHeader, version),
		internal.WithHeader("x-goog-api-client", internal.GetMetricsHeader(conf.Version)),
	}

	return &fcmClient{
		fcmEndpoint:   messagingEndpoint,
		batchEndpoint: batchEndpoint,
		project:       conf.ProjectID,
		version:       version,
		httpClient:    client,
	}
}

// Send sends a Message to Firebase Cloud Messaging.
//
// The Message must specify exactly one of Token, Topic and Condition fields. FCM will
// customize the message for each target platform based on the arguments specified in the
// Message.
func (c *fcmClient) Send(ctx context.Context, message *Message) (string, error) {
	payload := &fcmRequest{
		Message: message,
	}
	return c.makeSendRequest(ctx, payload)
}

// SendDryRun sends a Message to Firebase Cloud Messaging in the dry run (validation only) mode.
//
// This function does not actually deliver the message to target devices. Instead, it performs all
// the SDK-level and backend validations on the message, and emulates the send operation.
func (c *fcmClient) SendDryRun(ctx context.Context, message *Message) (string, error) {
	payload := &fcmRequest{
		ValidateOnly: true,
		Message:      message,
	}
	return c.makeSendRequest(ctx, payload)
}

func (c *fcmClient) makeSendRequest(ctx context.Context, req *fcmRequest) (string, error) {
	if err := validateMessage(req.Message); err != nil {
		return "", err
	}

	request := &internal.Request{
		Method: http.MethodPost,
		URL:    fmt.Sprintf("%s/projects/%s/messages:send", c.fcmEndpoint, c.project),
		Body:   internal.NewJSONEntity(req),
	}

	var result fcmResponse
	_, err := c.httpClient.DoAndUnmarshal(ctx, request, &result)
	return result.Name, err
}

// IsInternal checks if the given error was due to an internal server error.
func IsInternal(err error) bool {
	return hasMessagingErrorCode(err, internalError)
}

// IsInvalidAPNSCredentials checks if the given error was due to invalid APNS certificate or auth
// key.
//
// Deprecated: Use IsThirdPartyAuthError().
func IsInvalidAPNSCredentials(err error) bool {
	return IsThirdPartyAuthError(err)
}

// IsThirdPartyAuthError checks if the given error was due to invalid APNS certificate or auth
// key.
func IsThirdPartyAuthError(err error) bool {
	return hasMessagingErrorCode(err, thirdPartyAuthError) || hasMessagingErrorCode(err, apnsAuthError)
}

// IsInvalidArgument checks if the given error was due to an invalid argument in the request.
func IsInvalidArgument(err error) bool {
	return hasMessagingErrorCode(err, invalidArgument)
}

// IsMessageRateExceeded checks if the given error was due to the client exceeding a quota.
//
// Deprecated: Use IsQuotaExceeded().
func IsMessageRateExceeded(err error) bool {
	return IsQuotaExceeded(err)
}

// IsQuotaExceeded checks if the given error was due to the client exceeding a quota.
func IsQuotaExceeded(err error) bool {
	return hasMessagingErrorCode(err, quotaExceeded)
}

// IsMismatchedCredential checks if the given error was due to an invalid credential or permission
// error.
//
// Deprecated: Use IsSenderIDMismatch().
func IsMismatchedCredential(err error) bool {
	return IsSenderIDMismatch(err)
}

// IsSenderIDMismatch checks if the given error was due to an invalid credential or permission
// error.
func IsSenderIDMismatch(err error) bool {
	return hasMessagingErrorCode(err, senderIDMismatch)
}

// IsRegistrationTokenNotRegistered checks if the given error was due to a registration token that
// became invalid.
//
// Deprecated: Use IsUnregistered().
func IsRegistrationTokenNotRegistered(err error) bool {
	return IsUnregistered(err)
}

// IsUnregistered checks if the given error was due to a registration token that
// became invalid.
func IsUnregistered(err error) bool {
	return hasMessagingErrorCode(err, unregistered)
}

// IsServerUnavailable checks if the given error was due to the backend server being temporarily
// unavailable.
//
// Deprecated: Use IsUnavailable().
func IsServerUnavailable(err error) bool {
	return IsUnavailable(err)
}

// IsUnavailable checks if the given error was due to the backend server being temporarily
// unavailable.
func IsUnavailable(err error) bool {
	return hasMessagingErrorCode(err, unavailable)
}

// IsTooManyTopics checks if the given error was due to the client exceeding the allowed number
// of topics.
//
// Deprecated: Always returns false.
func IsTooManyTopics(err error) bool {
	return false
}

// IsUnknown checks if the given error was due to unknown error returned by the backend server.
//
// Deprecated: Always returns false.
func IsUnknown(err error) bool {
	return false
}

type fcmRequest struct {
	ValidateOnly bool     `json:"validate_only,omitempty"`
	Message      *Message `json:"message,omitempty"`
}

type fcmResponse struct {
	Name string `json:"name"`
}

type fcmErrorResponse struct {
	Error struct {
		Details []struct {
			Type      string `json:"@type"`
			ErrorCode string `json:"errorCode"`
		}
	} `json:"error"`
}

func handleFCMError(resp *internal.Response) error {
	base := internal.NewFirebaseErrorOnePlatform(resp)
	var fe fcmErrorResponse
	json.Unmarshal(resp.Body, &fe) // ignore any json parse errors at this level
	for _, d := range fe.Error.Details {
		if d.Type == "type.googleapis.com/google.firebase.fcm.v1.FcmError" {
			base.Ext["messagingErrorCode"] = d.ErrorCode
			break
		}
	}

	return base
}

func hasMessagingErrorCode(err error, code string) bool {
	fe, ok := err.(*internal.FirebaseError)
	if !ok {
		return false
	}

	got, ok := fe.Ext["messagingErrorCode"]
	return ok && got == code
}
