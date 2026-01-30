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
	"encoding/json"
)

// ImagemapActionType type
type ImagemapActionType string

// ImagemapActionType constants
const (
	ImagemapActionTypeURI     ImagemapActionType = "uri"
	ImagemapActionTypeMessage ImagemapActionType = "message"
)

// ImagemapBaseSize type
type ImagemapBaseSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// ImagemapArea type
type ImagemapArea struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// ImagemapVideo type
type ImagemapVideo struct {
	OriginalContentURL string                     `json:"originalContentUrl"`
	PreviewImageURL    string                     `json:"previewImageUrl"`
	Area               ImagemapArea               `json:"area"`
	ExternalLink       *ImagemapVideoExternalLink `json:"externalLink,omitempty"`
}

// ImagemapVideoExternalLink type
type ImagemapVideoExternalLink struct {
	LinkURI string `json:"linkUri"`
	Label   string `json:"label"`
}

// ImagemapAction type
type ImagemapAction interface {
	json.Marshaler
	ImagemapAction()
}

// URIImagemapAction type
type URIImagemapAction struct {
	Label   string
	LinkURL string
	Area    ImagemapArea
}

// MarshalJSON method of URIImagemapAction
func (a *URIImagemapAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type    ImagemapActionType `json:"type"`
		Label   string             `json:"label,omitempty"`
		LinkURL string             `json:"linkUri"`
		Area    ImagemapArea       `json:"area"`
	}{
		Type:    ImagemapActionTypeURI,
		Label:   a.Label,
		LinkURL: a.LinkURL,
		Area:    a.Area,
	})
}

// MessageImagemapAction type
type MessageImagemapAction struct {
	Label string
	Text  string
	Area  ImagemapArea
}

// MarshalJSON method of MessageImagemapAction
func (a *MessageImagemapAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type  ImagemapActionType `json:"type"`
		Label string             `json:"label,omitempty"`
		Text  string             `json:"text"`
		Area  ImagemapArea       `json:"area"`
	}{
		Type:  ImagemapActionTypeMessage,
		Label: a.Label,
		Text:  a.Text,
		Area:  a.Area,
	})
}

// ImagemapAction implements ImagemapAction interface
func (a *URIImagemapAction) ImagemapAction() {}

// ImagemapAction implements ImagemapAction interface
func (a *MessageImagemapAction) ImagemapAction() {}

// NewURIImagemapAction function
func NewURIImagemapAction(label, linkURL string, area ImagemapArea) *URIImagemapAction {
	return &URIImagemapAction{
		Label:   label,
		LinkURL: linkURL,
		Area:    area,
	}
}

// NewMessageImagemapAction function
func NewMessageImagemapAction(label, text string, area ImagemapArea) *MessageImagemapAction {
	return &MessageImagemapAction{
		Label: label,
		Text:  text,
		Area:  area,
	}
}
