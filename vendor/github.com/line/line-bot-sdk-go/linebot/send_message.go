// Copyright 2016 LINE Corporation
//
// LINE Corporation licenses this file to you under the Apache License,
// version 2.0 (the "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at:
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package linebot

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
)

// PushMessage method
func (client *Client) PushMessage(to string, messages ...SendingMessage) *PushMessageCall {
	return &PushMessageCall{
		c:                    client,
		to:                   to,
		messages:             messages,
		notificationDisabled: false,
	}
}

// PushMessageCall type
type PushMessageCall struct {
	c   *Client
	ctx context.Context

	to                   string
	messages             []SendingMessage
	notificationDisabled bool
}

// WithContext method
func (call *PushMessageCall) WithContext(ctx context.Context) *PushMessageCall {
	call.ctx = ctx
	return call
}

// WithNotificationDisabled method will disable push notification
func (call *PushMessageCall) WithNotificationDisabled() *PushMessageCall {
	call.notificationDisabled = true
	return call
}

// WithRetryKey method will set retry key string (UUID) on PushMessage.
func (call *PushMessageCall) WithRetryKey(retryKey string) *PushMessageCall {
	call.c.setRetryKey(retryKey)
	return call
}

func (call *PushMessageCall) encodeJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	return enc.Encode(&struct {
		To                   string           `json:"to"`
		Messages             []SendingMessage `json:"messages"`
		NotificationDisabled bool             `json:"notificationDisabled,omitempty"`
	}{
		To:                   call.to,
		Messages:             call.messages,
		NotificationDisabled: call.notificationDisabled,
	})
}

// Do method
func (call *PushMessageCall) Do() (*BasicResponse, error) {
	var buf bytes.Buffer
	if err := call.encodeJSON(&buf); err != nil {
		return nil, err
	}
	res, err := call.c.post(call.ctx, APIEndpointPushMessage, &buf)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToBasicResponse(res)
}

// ReplyMessage method
func (client *Client) ReplyMessage(replyToken string, messages ...SendingMessage) *ReplyMessageCall {
	return &ReplyMessageCall{
		c:                    client,
		replyToken:           replyToken,
		messages:             messages,
		notificationDisabled: false,
	}
}

// ReplyMessageCall type
type ReplyMessageCall struct {
	c   *Client
	ctx context.Context

	replyToken           string
	messages             []SendingMessage
	notificationDisabled bool
}

// WithContext method
func (call *ReplyMessageCall) WithContext(ctx context.Context) *ReplyMessageCall {
	call.ctx = ctx
	return call
}

// WithNotificationDisabled method will disable push notification
func (call *ReplyMessageCall) WithNotificationDisabled() *ReplyMessageCall {
	call.notificationDisabled = true
	return call
}

func (call *ReplyMessageCall) encodeJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	return enc.Encode(&struct {
		ReplyToken           string           `json:"replyToken"`
		Messages             []SendingMessage `json:"messages"`
		NotificationDisabled bool             `json:"notificationDisabled,omitempty"`
	}{
		ReplyToken:           call.replyToken,
		Messages:             call.messages,
		NotificationDisabled: call.notificationDisabled,
	})
}

// Do method
func (call *ReplyMessageCall) Do() (*BasicResponse, error) {
	var buf bytes.Buffer
	if err := call.encodeJSON(&buf); err != nil {
		return nil, err
	}
	res, err := call.c.post(call.ctx, APIEndpointReplyMessage, &buf)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToBasicResponse(res)
}

// Multicast method
func (client *Client) Multicast(to []string, messages ...SendingMessage) *MulticastCall {
	return &MulticastCall{
		c:                    client,
		to:                   to,
		messages:             messages,
		notificationDisabled: false,
	}
}

// MulticastCall type
type MulticastCall struct {
	c   *Client
	ctx context.Context

	to                   []string
	messages             []SendingMessage
	notificationDisabled bool
}

// WithContext method
func (call *MulticastCall) WithContext(ctx context.Context) *MulticastCall {
	call.ctx = ctx
	return call
}

// WithNotificationDisabled method will disable push notification
func (call *MulticastCall) WithNotificationDisabled() *MulticastCall {
	call.notificationDisabled = true
	return call
}

// WithRetryKey method will set retry key string (UUID) on Multicast.
func (call *MulticastCall) WithRetryKey(retryKey string) *MulticastCall {
	call.c.setRetryKey(retryKey)
	return call
}

func (call *MulticastCall) encodeJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	return enc.Encode(&struct {
		To                   []string         `json:"to"`
		Messages             []SendingMessage `json:"messages"`
		NotificationDisabled bool             `json:"notificationDisabled,omitempty"`
	}{
		To:                   call.to,
		Messages:             call.messages,
		NotificationDisabled: call.notificationDisabled,
	})
}

// Do method
func (call *MulticastCall) Do() (*BasicResponse, error) {
	var buf bytes.Buffer
	if err := call.encodeJSON(&buf); err != nil {
		return nil, err
	}
	res, err := call.c.post(call.ctx, APIEndpointMulticast, &buf)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToBasicResponse(res)
}

// BroadcastMessage method
func (client *Client) BroadcastMessage(messages ...SendingMessage) *BroadcastMessageCall {
	return &BroadcastMessageCall{
		c:        client,
		messages: messages,
	}
}

// BroadcastMessageCall type
type BroadcastMessageCall struct {
	c   *Client
	ctx context.Context

	messages []SendingMessage
}

// WithContext method
func (call *BroadcastMessageCall) WithContext(ctx context.Context) *BroadcastMessageCall {
	call.ctx = ctx
	return call
}

// WithRetryKey method will set retry key string (UUID) on BroadcastMessage.
func (call *BroadcastMessageCall) WithRetryKey(retryKey string) *BroadcastMessageCall {
	call.c.setRetryKey(retryKey)
	return call
}

func (call *BroadcastMessageCall) encodeJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	return enc.Encode(&struct {
		Messages []SendingMessage `json:"messages"`
	}{
		Messages: call.messages,
	})
}

// Do method
func (call *BroadcastMessageCall) Do() (*BasicResponse, error) {
	var buf bytes.Buffer
	if err := call.encodeJSON(&buf); err != nil {
		return nil, err
	}
	res, err := call.c.post(call.ctx, APIEndpointBroadcastMessage, &buf)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToBasicResponse(res)
}

// Narrowcast method
func (client *Client) Narrowcast(messages ...SendingMessage) *NarrowcastCall {
	return &NarrowcastCall{
		c:        client,
		messages: messages,
	}
}

// NarrowcastCall type
type NarrowcastCall struct {
	c   *Client
	ctx context.Context

	messages  []SendingMessage
	recipient Recipient
	filter    *Filter
	limit     *NarrowcastMessageLimit
}

// Filter type
type Filter struct {
	Demographic DemographicFilter `json:"demographic"`
}

// NarrowcastMessageLimit type
type NarrowcastMessageLimit struct {
	Max                int  `json:"max"`
	UpToRemainingQuota bool `json:"upToRemainingQuota,omitempty"`
}

// WithContext method
func (call *NarrowcastCall) WithContext(ctx context.Context) *NarrowcastCall {
	call.ctx = ctx
	return call
}

// WithRecipient method will send to specific recipient objects
func (call *NarrowcastCall) WithRecipient(recipient Recipient) *NarrowcastCall {
	call.recipient = recipient
	return call
}

// WithDemographic method will send to specific recipients filter by demographic
func (call *NarrowcastCall) WithDemographic(demographic DemographicFilter) *NarrowcastCall {
	call.filter = &Filter{Demographic: demographic}
	return call
}

// WithLimitMax method will set maximum number of recipients
func (call *NarrowcastCall) WithLimitMax(max int) *NarrowcastCall {
	call.limit = &NarrowcastMessageLimit{Max: max}
	return call
}

// WithLimitMaxUpToRemainingQuota method will set maximum number of recipients but not over remaining quota.
func (call *NarrowcastCall) WithLimitMaxUpToRemainingQuota(max int, upToRemainingQuota bool) *NarrowcastCall {
	call.limit = &NarrowcastMessageLimit{Max: max, UpToRemainingQuota: upToRemainingQuota}
	return call
}

// WithRetryKey method will set retry key string (UUID) on narrowcast.
func (call *NarrowcastCall) WithRetryKey(retryKey string) *NarrowcastCall {
	call.c.setRetryKey(retryKey)
	return call
}

func (call *NarrowcastCall) encodeJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	return enc.Encode(&struct {
		Messages  []SendingMessage        `json:"messages"`
		Recipient Recipient               `json:"recipient,omitempty"`
		Filter    *Filter                 `json:"filter,omitempty"`
		Limit     *NarrowcastMessageLimit `json:"limit,omitempty"`
	}{
		Messages:  call.messages,
		Recipient: call.recipient,
		Filter:    call.filter,
		Limit:     call.limit,
	})
}

// Do method
func (call *NarrowcastCall) Do() (*BasicResponse, error) {
	var buf bytes.Buffer
	if err := call.encodeJSON(&buf); err != nil {
		return nil, err
	}
	res, err := call.c.post(call.ctx, APIEndpointNarrowcast, &buf)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToBasicResponse(res)
}
