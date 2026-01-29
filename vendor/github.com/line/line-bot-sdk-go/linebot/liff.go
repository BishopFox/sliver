// Copyright 2018 LINE Corporation
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
	"fmt"
	"io"
)

// LIFFViewType type
type LIFFViewType string

// LIFFViewType constants
const (
	LIFFViewTypeCompact LIFFViewType = "compact"
	LIFFViewTypeTall    LIFFViewType = "tall"
	LIFFViewTypeFull    LIFFViewType = "full"
)

// LIFFApp type
type LIFFApp struct {
	LIFFID string `json:"liffId"`
	View   View   `json:"view"`
}

// ViewRequest type
type ViewRequest struct {
	View View `json:"view"`
}

// View type
type View struct {
	Type LIFFViewType `json:"type"`
	URL  string       `json:"url"`
}

// GetLIFF method
func (client *Client) GetLIFF() *GetLIFFAllCall {
	return &GetLIFFAllCall{
		c: client,
	}
}

// GetLIFFAllCall type
type GetLIFFAllCall struct {
	c   *Client
	ctx context.Context
}

// WithContext method
func (call *GetLIFFAllCall) WithContext(ctx context.Context) *GetLIFFAllCall {
	call.ctx = ctx
	return call
}

// Do method
func (call *GetLIFFAllCall) Do() (*LIFFAppsResponse, error) {
	res, err := call.c.get(call.ctx, call.c.endpointBase, APIEndpointGetAllLIFFApps, nil)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToLIFFResponse(res)
}

// AddLIFF method
func (client *Client) AddLIFF(view View) *AddLIFFCall {
	return &AddLIFFCall{
		c:    client,
		view: view,
	}
}

// AddLIFFCall type
type AddLIFFCall struct {
	c   *Client
	ctx context.Context

	view View
}

// WithContext method
func (call *AddLIFFCall) WithContext(ctx context.Context) *AddLIFFCall {
	call.ctx = ctx
	return call
}

func (call *AddLIFFCall) encodeJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	return enc.Encode(&struct {
		View View `json:"view"`
	}{
		View: call.view,
	})
}

// Do method
func (call *AddLIFFCall) Do() (*LIFFIDResponse, error) {
	var buf bytes.Buffer
	if err := call.encodeJSON(&buf); err != nil {
		return nil, err
	}
	res, err := call.c.post(call.ctx, APIEndpointAddLIFFApp, &buf)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToLIFFIDResponse(res)
}

// UpdateLIFF method
func (client *Client) UpdateLIFF(liffID string, view View) *UpdateLIFFCall {
	return &UpdateLIFFCall{
		c:      client,
		liffID: liffID,
		view:   view,
	}
}

// UpdateLIFFCall type
type UpdateLIFFCall struct {
	c   *Client
	ctx context.Context

	liffID string
	view   View
}

// WithContext method
func (call *UpdateLIFFCall) WithContext(ctx context.Context) *UpdateLIFFCall {
	call.ctx = ctx
	return call
}

func (call *UpdateLIFFCall) encodeJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	return enc.Encode(&struct {
		Type LIFFViewType `json:"type"`
		URL  string       `json:"url"`
	}{
		Type: call.view.Type,
		URL:  call.view.URL,
	})
}

// Do method
func (call *UpdateLIFFCall) Do() (*BasicResponse, error) {
	var buf bytes.Buffer
	if err := call.encodeJSON(&buf); err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf(APIEndpointUpdateLIFFApp, call.liffID)
	res, err := call.c.put(call.ctx, endpoint, &buf)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToBasicResponse(res)
}

// DeleteLIFF method
func (client *Client) DeleteLIFF(liffID string) *DeleteLIFFCall {
	return &DeleteLIFFCall{
		c:      client,
		liffID: liffID,
	}
}

// DeleteLIFFCall type
type DeleteLIFFCall struct {
	c   *Client
	ctx context.Context

	liffID string
}

// WithContext method
func (call *DeleteLIFFCall) WithContext(ctx context.Context) *DeleteLIFFCall {
	call.ctx = ctx
	return call
}

// Do method
func (call *DeleteLIFFCall) Do() (*BasicResponse, error) {
	endpoint := fmt.Sprintf(APIEndpointDeleteLIFFApp, call.liffID)
	res, err := call.c.delete(call.ctx, endpoint)
	if err != nil {
		return nil, err
	}
	defer closeResponse(res)
	return decodeToBasicResponse(res)
}
