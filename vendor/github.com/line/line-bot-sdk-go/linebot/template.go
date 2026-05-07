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

// TemplateType type
type TemplateType string

// TemplateType constants
const (
	TemplateTypeButtons       TemplateType = "buttons"
	TemplateTypeConfirm       TemplateType = "confirm"
	TemplateTypeCarousel      TemplateType = "carousel"
	TemplateTypeImageCarousel TemplateType = "image_carousel"
)

// ImageAspectRatioType type
type ImageAspectRatioType string

// ImageAspectRatioType constants
const (
	ImageAspectRatioTypeRectangle ImageAspectRatioType = "rectangle"
	ImageAspectRatioTypeSquare    ImageAspectRatioType = "square"
)

// ImageSizeType type
type ImageSizeType string

// ImageSizeType constants
const (
	ImageSizeTypeCover   ImageSizeType = "cover"
	ImageSizeTypeContain ImageSizeType = "contain"
)

// Template interface
type Template interface {
	json.Marshaler
	Template()
}

// ButtonsTemplate type
type ButtonsTemplate struct {
	ThumbnailImageURL    string
	ImageAspectRatio     ImageAspectRatioType
	ImageSize            ImageSizeType
	ImageBackgroundColor string
	Title                string
	Text                 string
	Actions              []TemplateAction
	DefaultAction        TemplateAction
}

// MarshalJSON method of ButtonsTemplate
func (t *ButtonsTemplate) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type                 TemplateType         `json:"type"`
		ThumbnailImageURL    string               `json:"thumbnailImageUrl,omitempty"`
		ImageAspectRatio     ImageAspectRatioType `json:"imageAspectRatio,omitempty"`
		ImageSize            ImageSizeType        `json:"imageSize,omitempty"`
		ImageBackgroundColor string               `json:"imageBackgroundColor,omitempty"`
		Title                string               `json:"title,omitempty"`
		Text                 string               `json:"text"`
		Actions              []TemplateAction     `json:"actions"`
		DefaultAction        TemplateAction       `json:"defaultAction,omitempty"`
	}{
		Type:                 TemplateTypeButtons,
		ThumbnailImageURL:    t.ThumbnailImageURL,
		ImageAspectRatio:     t.ImageAspectRatio,
		ImageSize:            t.ImageSize,
		ImageBackgroundColor: t.ImageBackgroundColor,
		Title:                t.Title,
		Text:                 t.Text,
		Actions:              t.Actions,
		DefaultAction:        t.DefaultAction,
	})
}

// WithImageOptions method, ButtonsTemplate can set imageAspectRatio, imageSize and imageBackgroundColor
func (t *ButtonsTemplate) WithImageOptions(imageAspectRatio ImageAspectRatioType, imageSize ImageSizeType, imageBackgroundColor string) *ButtonsTemplate {
	t.ImageAspectRatio = imageAspectRatio
	t.ImageSize = imageSize
	t.ImageBackgroundColor = imageBackgroundColor
	return t
}

// WithDefaultAction method, ButtonsTemplate can set defaultAction
func (t *ButtonsTemplate) WithDefaultAction(defaultAction TemplateAction) *ButtonsTemplate {
	t.DefaultAction = defaultAction
	return t
}

// ConfirmTemplate type
type ConfirmTemplate struct {
	Text    string
	Actions []TemplateAction
}

// MarshalJSON method of ConfirmTemplate
func (t *ConfirmTemplate) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type    TemplateType     `json:"type"`
		Text    string           `json:"text"`
		Actions []TemplateAction `json:"actions"`
	}{
		Type:    TemplateTypeConfirm,
		Text:    t.Text,
		Actions: t.Actions,
	})
}

// CarouselTemplate type
type CarouselTemplate struct {
	Columns          []*CarouselColumn
	ImageAspectRatio ImageAspectRatioType
	ImageSize        ImageSizeType
}

// CarouselColumn type
type CarouselColumn struct {
	ThumbnailImageURL    string           `json:"thumbnailImageUrl,omitempty"`
	ImageBackgroundColor string           `json:"imageBackgroundColor,omitempty"`
	Title                string           `json:"title,omitempty"`
	Text                 string           `json:"text"`
	Actions              []TemplateAction `json:"actions"`
	DefaultAction        TemplateAction   `json:"defaultAction,omitempty"`
}

// MarshalJSON method of CarouselTemplate
func (t *CarouselTemplate) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type             TemplateType         `json:"type"`
		Columns          []*CarouselColumn    `json:"columns"`
		ImageAspectRatio ImageAspectRatioType `json:"imageAspectRatio,omitempty"`
		ImageSize        ImageSizeType        `json:"imageSize,omitempty"`
	}{
		Type:             TemplateTypeCarousel,
		Columns:          t.Columns,
		ImageAspectRatio: t.ImageAspectRatio,
		ImageSize:        t.ImageSize,
	})
}

// WithImageOptions method, CarouselTemplate can set imageAspectRatio and imageSize
func (t *CarouselTemplate) WithImageOptions(imageAspectRatio ImageAspectRatioType, imageSize ImageSizeType) *CarouselTemplate {
	t.ImageAspectRatio = imageAspectRatio
	t.ImageSize = imageSize
	return t
}

// WithImageOptions method, CarouselColumn can set imageBackgroundColor
func (t *CarouselColumn) WithImageOptions(imageBackgroundColor string) *CarouselColumn {
	t.ImageBackgroundColor = imageBackgroundColor
	return t
}

// WithDefaultAction method, CarouselColumn can set defaultAction
func (t *CarouselColumn) WithDefaultAction(defaultAction TemplateAction) *CarouselColumn {
	t.DefaultAction = defaultAction
	return t
}

// ImageCarouselTemplate type
type ImageCarouselTemplate struct {
	Columns []*ImageCarouselColumn
}

// ImageCarouselColumn type
type ImageCarouselColumn struct {
	ImageURL string         `json:"imageUrl"`
	Action   TemplateAction `json:"action"`
}

// MarshalJSON method of ImageCarouselTemplate
func (t *ImageCarouselTemplate) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type    TemplateType           `json:"type"`
		Columns []*ImageCarouselColumn `json:"columns"`
	}{
		Type:    TemplateTypeImageCarousel,
		Columns: t.Columns,
	})
}

// Template implements Template interface
func (*ConfirmTemplate) Template() {}

// Template implements Template interface
func (*ButtonsTemplate) Template() {}

// Template implements Template interface
func (*CarouselTemplate) Template() {}

// Template implements Template interface
func (*ImageCarouselTemplate) Template() {}

// NewConfirmTemplate function
func NewConfirmTemplate(text string, left, right TemplateAction) *ConfirmTemplate {
	return &ConfirmTemplate{
		Text:    text,
		Actions: []TemplateAction{left, right},
	}
}

// NewButtonsTemplate function
// `thumbnailImageURL` and `title` are optional. they can be empty.
func NewButtonsTemplate(thumbnailImageURL, title, text string, actions ...TemplateAction) *ButtonsTemplate {
	return &ButtonsTemplate{
		ThumbnailImageURL: thumbnailImageURL,
		Title:             title,
		Text:              text,
		Actions:           actions,
	}
}

// NewCarouselTemplate function
func NewCarouselTemplate(columns ...*CarouselColumn) *CarouselTemplate {
	return &CarouselTemplate{
		Columns: columns,
	}
}

// NewCarouselColumn function
// `thumbnailImageURL` and `title` are optional. they can be empty.
func NewCarouselColumn(thumbnailImageURL, title, text string, actions ...TemplateAction) *CarouselColumn {
	return &CarouselColumn{
		ThumbnailImageURL: thumbnailImageURL,
		Title:             title,
		Text:              text,
		Actions:           actions,
	}
}

// NewImageCarouselTemplate function
func NewImageCarouselTemplate(columns ...*ImageCarouselColumn) *ImageCarouselTemplate {
	return &ImageCarouselTemplate{
		Columns: columns,
	}
}

// NewImageCarouselColumn function
func NewImageCarouselColumn(imageURL string, action TemplateAction) *ImageCarouselColumn {
	return &ImageCarouselColumn{
		ImageURL: imageURL,
		Action:   action,
	}
}
