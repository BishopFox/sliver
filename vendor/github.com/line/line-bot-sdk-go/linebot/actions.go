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

import "encoding/json"

// ActionType type
type ActionType string

// ActionType constants
const (
	ActionTypeURI            ActionType = "uri"
	ActionTypeMessage        ActionType = "message"
	ActionTypePostback       ActionType = "postback"
	ActionTypeDatetimePicker ActionType = "datetimepicker"
	ActionTypeCamera         ActionType = "camera"
	ActionTypeCameraRoll     ActionType = "cameraRoll"
	ActionTypeLocation       ActionType = "location"
)

// Action interface
type Action interface {
	json.Marshaler
}

// TemplateAction interface
type TemplateAction interface {
	Action
	TemplateAction()
}

// QuickReplyAction type
type QuickReplyAction interface {
	Action
	QuickReplyAction()
}

// URIAction type
type URIAction struct {
	Label  string
	URI    string
	AltURI *URIActionAltURI
}

// URIActionAltURI type
type URIActionAltURI struct {
	Desktop string `json:"desktop"`
}

// MarshalJSON method of URIAction
func (a *URIAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type   ActionType       `json:"type"`
		Label  string           `json:"label,omitempty"`
		URI    string           `json:"uri"`
		AltURI *URIActionAltURI `json:"altUri,omitempty"`
	}{
		Type:   ActionTypeURI,
		Label:  a.Label,
		URI:    a.URI,
		AltURI: a.AltURI,
	})
}

// MessageAction type
type MessageAction struct {
	Label string
	Text  string
}

// MarshalJSON method of MessageAction
func (a *MessageAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type  ActionType `json:"type"`
		Label string     `json:"label,omitempty"`
		Text  string     `json:"text"`
	}{
		Type:  ActionTypeMessage,
		Label: a.Label,
		Text:  a.Text,
	})
}

// PostbackAction type
type PostbackAction struct {
	Label       string
	Data        string
	Text        string
	DisplayText string
}

// MarshalJSON method of PostbackAction
func (a *PostbackAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type        ActionType `json:"type"`
		Label       string     `json:"label,omitempty"`
		Data        string     `json:"data"`
		Text        string     `json:"text,omitempty"`
		DisplayText string     `json:"displayText,omitempty"`
	}{
		Type:        ActionTypePostback,
		Label:       a.Label,
		Data:        a.Data,
		Text:        a.Text,
		DisplayText: a.DisplayText,
	})
}

// DatetimePickerAction type
type DatetimePickerAction struct {
	Label   string
	Data    string
	Mode    string
	Initial string
	Max     string
	Min     string
}

// MarshalJSON method of DatetimePickerAction
func (a *DatetimePickerAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type    ActionType `json:"type"`
		Label   string     `json:"label,omitempty"`
		Data    string     `json:"data"`
		Mode    string     `json:"mode"`
		Initial string     `json:"initial,omitempty"`
		Max     string     `json:"max,omitempty"`
		Min     string     `json:"min,omitempty"`
	}{
		Type:    ActionTypeDatetimePicker,
		Label:   a.Label,
		Data:    a.Data,
		Mode:    a.Mode,
		Initial: a.Initial,
		Max:     a.Max,
		Min:     a.Min,
	})
}

// CameraAction type
type CameraAction struct {
	Label string
}

// MarshalJSON method of CameraAction
func (a *CameraAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type  ActionType `json:"type"`
		Label string     `json:"label"`
	}{
		Type:  ActionTypeCamera,
		Label: a.Label,
	})
}

// CameraRollAction type
type CameraRollAction struct {
	Label string
}

// MarshalJSON method of CameraRollAction
func (a *CameraRollAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type  ActionType `json:"type"`
		Label string     `json:"label"`
	}{
		Type:  ActionTypeCameraRoll,
		Label: a.Label,
	})
}

// LocationAction type
type LocationAction struct {
	Label string
}

// MarshalJSON method of LocationAction
func (a *LocationAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type  ActionType `json:"type"`
		Label string     `json:"label"`
	}{
		Type:  ActionTypeLocation,
		Label: a.Label,
	})
}

// TemplateAction implements TemplateAction interface
func (*URIAction) TemplateAction() {}

// TemplateAction implements TemplateAction interface
func (*MessageAction) TemplateAction() {}

// TemplateAction implements TemplateAction interface
func (*PostbackAction) TemplateAction() {}

// TemplateAction implements TemplateAction interface
func (*DatetimePickerAction) TemplateAction() {}

// QuickReplyAction implements QuickReplyAction interface
func (*MessageAction) QuickReplyAction() {}

// QuickReplyAction implements QuickReplyAction interface
func (*PostbackAction) QuickReplyAction() {}

// QuickReplyAction implements QuickReplyAction interface
func (*DatetimePickerAction) QuickReplyAction() {}

// QuickReplyAction implements QuickReplyAction interface
func (*CameraAction) QuickReplyAction() {}

// QuickReplyAction implements QuickReplyAction interface
func (*CameraRollAction) QuickReplyAction() {}

// QuickReplyAction implements QuickReplyAction interface
func (*LocationAction) QuickReplyAction() {}

// NewURIAction function
func NewURIAction(label, uri string) *URIAction {
	return &URIAction{
		Label: label,
		URI:   uri,
	}
}

// NewMessageAction function
func NewMessageAction(label, text string) *MessageAction {
	return &MessageAction{
		Label: label,
		Text:  text,
	}
}

// NewPostbackAction function
func NewPostbackAction(label, data, text, displayText string) *PostbackAction {
	return &PostbackAction{
		Label:       label,
		Data:        data,
		Text:        text,
		DisplayText: displayText,
	}
}

// NewDatetimePickerAction function
func NewDatetimePickerAction(label, data, mode, initial, max, min string) *DatetimePickerAction {
	return &DatetimePickerAction{
		Label:   label,
		Data:    data,
		Mode:    mode,
		Initial: initial,
		Max:     max,
		Min:     min,
	}
}

// NewCameraAction function
func NewCameraAction(label string) *CameraAction {
	return &CameraAction{
		Label: label,
	}
}

// NewCameraRollAction function
func NewCameraRollAction(label string) *CameraRollAction {
	return &CameraRollAction{
		Label: label,
	}
}

// NewLocationAction function
func NewLocationAction(label string) *LocationAction {
	return &LocationAction{
		Label: label,
	}
}
