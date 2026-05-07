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

// FlexContainerType type
type FlexContainerType string

// IntPtr is a helper function for using *int values
func IntPtr(v int) *int {
	return &v
}

// FlexContainerType constants
const (
	FlexContainerTypeBubble   FlexContainerType = "bubble"
	FlexContainerTypeCarousel FlexContainerType = "carousel"
)

// FlexComponentType type
type FlexComponentType string

// FlexComponentType constants
const (
	FlexComponentTypeBox       FlexComponentType = "box"
	FlexComponentTypeButton    FlexComponentType = "button"
	FlexComponentTypeFiller    FlexComponentType = "filler"
	FlexComponentTypeIcon      FlexComponentType = "icon"
	FlexComponentTypeImage     FlexComponentType = "image"
	FlexComponentTypeSeparator FlexComponentType = "separator"
	FlexComponentTypeSpacer    FlexComponentType = "spacer"
	FlexComponentTypeSpan      FlexComponentType = "span"
	FlexComponentTypeText      FlexComponentType = "text"
)

// FlexBubbleSizeType type
type FlexBubbleSizeType string

// FlexBubbleSizeType constants
const (
	FlexBubbleSizeTypeNano  FlexBubbleSizeType = "nano"
	FlexBubbleSizeTypeMicro FlexBubbleSizeType = "micro"
	FlexBubbleSizeTypeKilo  FlexBubbleSizeType = "kilo"
	FlexBubbleSizeTypeMega  FlexBubbleSizeType = "mega"
	FlexBubbleSizeTypeGiga  FlexBubbleSizeType = "giga"
)

// FlexBubbleDirectionType type
type FlexBubbleDirectionType string

// FlexBubbleDirectionType constants
const (
	FlexBubbleDirectionTypeLTR FlexBubbleDirectionType = "ltr"
	FlexBubbleDirectionTypeRTL FlexBubbleDirectionType = "rtl"
)

// FlexButtonStyleType type
type FlexButtonStyleType string

// FlexButtonStyleType constants
const (
	FlexButtonStyleTypeLink      FlexButtonStyleType = "link"
	FlexButtonStyleTypePrimary   FlexButtonStyleType = "primary"
	FlexButtonStyleTypeSecondary FlexButtonStyleType = "secondary"
)

// FlexButtonHeightType type
type FlexButtonHeightType string

// FlexButtonHeightType constants
const (
	FlexButtonHeightTypeMd FlexButtonHeightType = "md"
	FlexButtonHeightTypeSm FlexButtonHeightType = "sm"
)

// FlexIconAspectRatioType type
type FlexIconAspectRatioType string

// FlexIconAspectRatioType constants
const (
	FlexIconAspectRatioType1to1 FlexIconAspectRatioType = "1:1"
	FlexIconAspectRatioType2to1 FlexIconAspectRatioType = "2:1"
	FlexIconAspectRatioType3to1 FlexIconAspectRatioType = "3:1"
)

// FlexImageSizeType type
type FlexImageSizeType string

// FlexImageSizeType constants
const (
	FlexImageSizeTypeXxs  FlexImageSizeType = "xxs"
	FlexImageSizeTypeXs   FlexImageSizeType = "xs"
	FlexImageSizeTypeSm   FlexImageSizeType = "sm"
	FlexImageSizeTypeMd   FlexImageSizeType = "md"
	FlexImageSizeTypeLg   FlexImageSizeType = "lg"
	FlexImageSizeTypeXl   FlexImageSizeType = "xl"
	FlexImageSizeTypeXxl  FlexImageSizeType = "xxl"
	FlexImageSizeType3xl  FlexImageSizeType = "3xl"
	FlexImageSizeType4xl  FlexImageSizeType = "4xl"
	FlexImageSizeType5xl  FlexImageSizeType = "5xl"
	FlexImageSizeTypeFull FlexImageSizeType = "full"
)

// FlexImageAspectRatioType type
type FlexImageAspectRatioType string

// FlexImageAspectRatioType constants
const (
	FlexImageAspectRatioType1to1    FlexImageAspectRatioType = "1:1"
	FlexImageAspectRatioType1_51to1 FlexImageAspectRatioType = "1.51:1"
	FlexImageAspectRatioType1_91to1 FlexImageAspectRatioType = "1.91:1"
	FlexImageAspectRatioType4to3    FlexImageAspectRatioType = "4:3"
	FlexImageAspectRatioType16to9   FlexImageAspectRatioType = "16:9"
	FlexImageAspectRatioType20to13  FlexImageAspectRatioType = "20:13"
	FlexImageAspectRatioType2to1    FlexImageAspectRatioType = "2:1"
	FlexImageAspectRatioType3to1    FlexImageAspectRatioType = "3:1"
	FlexImageAspectRatioType3to4    FlexImageAspectRatioType = "3:4"
	FlexImageAspectRatioType9to16   FlexImageAspectRatioType = "9:16"
	FlexImageAspectRatioType1to2    FlexImageAspectRatioType = "1:2"
	FlexImageAspectRatioType1to3    FlexImageAspectRatioType = "1:3"
)

// FlexImageAspectModeType type
type FlexImageAspectModeType string

// FlexImageAspectModeType constants
const (
	FlexImageAspectModeTypeCover FlexImageAspectModeType = "cover"
	FlexImageAspectModeTypeFit   FlexImageAspectModeType = "fit"
)

// FlexBoxLayoutType type
type FlexBoxLayoutType string

// FlexBoxLayoutType constants
const (
	FlexBoxLayoutTypeHorizontal FlexBoxLayoutType = "horizontal"
	FlexBoxLayoutTypeVertical   FlexBoxLayoutType = "vertical"
	FlexBoxLayoutTypeBaseline   FlexBoxLayoutType = "baseline"
)

// FlexComponentSpacingType type
type FlexComponentSpacingType string

// FlexComponentSpacingType constants
const (
	FlexComponentSpacingTypeNone FlexComponentSpacingType = "none"
	FlexComponentSpacingTypeXs   FlexComponentSpacingType = "xs"
	FlexComponentSpacingTypeSm   FlexComponentSpacingType = "sm"
	FlexComponentSpacingTypeMd   FlexComponentSpacingType = "md"
	FlexComponentSpacingTypeLg   FlexComponentSpacingType = "lg"
	FlexComponentSpacingTypeXl   FlexComponentSpacingType = "xl"
	FlexComponentSpacingTypeXxl  FlexComponentSpacingType = "xxl"
)

// FlexComponentMarginType type
type FlexComponentMarginType string

// FlexComponentMarginType constants
const (
	FlexComponentMarginTypeNone FlexComponentMarginType = "none"
	FlexComponentMarginTypeXs   FlexComponentMarginType = "xs"
	FlexComponentMarginTypeSm   FlexComponentMarginType = "sm"
	FlexComponentMarginTypeMd   FlexComponentMarginType = "md"
	FlexComponentMarginTypeLg   FlexComponentMarginType = "lg"
	FlexComponentMarginTypeXl   FlexComponentMarginType = "xl"
	FlexComponentMarginTypeXxl  FlexComponentMarginType = "xxl"
)

// FlexComponentGravityType type
type FlexComponentGravityType string

// FlexComponentGravityType constants
const (
	FlexComponentGravityTypeTop    FlexComponentGravityType = "top"
	FlexComponentGravityTypeBottom FlexComponentGravityType = "bottom"
	FlexComponentGravityTypeCenter FlexComponentGravityType = "center"
)

// FlexComponentAlignType type
type FlexComponentAlignType string

// FlexComponentAlignType constants
const (
	FlexComponentAlignTypeStart  FlexComponentAlignType = "start"
	FlexComponentAlignTypeEnd    FlexComponentAlignType = "end"
	FlexComponentAlignTypeCenter FlexComponentAlignType = "center"
)

// FlexComponentCornerRadiusType type
type FlexComponentCornerRadiusType string

// FlexComponentCornerRadius constants
const (
	FlexComponentCornerRadiusTypeNone FlexComponentCornerRadiusType = "none"
	FlexComponentCornerRadiusTypeXs   FlexComponentCornerRadiusType = "xs"
	FlexComponentCornerRadiusTypeSm   FlexComponentCornerRadiusType = "sm"
	FlexComponentCornerRadiusTypeMd   FlexComponentCornerRadiusType = "md"
	FlexComponentCornerRadiusTypeLg   FlexComponentCornerRadiusType = "lg"
	FlexComponentCornerRadiusTypeXl   FlexComponentCornerRadiusType = "xl"
	FlexComponentCornerRadiusTypeXxl  FlexComponentCornerRadiusType = "xxl"
)

// FlexIconSizeType type
type FlexIconSizeType string

// FlexIconSizeType constants
const (
	FlexIconSizeTypeXxs FlexIconSizeType = "xxs"
	FlexIconSizeTypeXs  FlexIconSizeType = "xs"
	FlexIconSizeTypeSm  FlexIconSizeType = "sm"
	FlexIconSizeTypeMd  FlexIconSizeType = "md"
	FlexIconSizeTypeLg  FlexIconSizeType = "lg"
	FlexIconSizeTypeXl  FlexIconSizeType = "xl"
	FlexIconSizeTypeXxl FlexIconSizeType = "xxl"
	FlexIconSizeType3xl FlexIconSizeType = "3xl"
	FlexIconSizeType4xl FlexIconSizeType = "4xl"
	FlexIconSizeType5xl FlexIconSizeType = "5xl"
)

// FlexSpacerSizeType type
type FlexSpacerSizeType string

// FlexSpacerSizeType constants
const (
	FlexSpacerSizeTypeXs  FlexSpacerSizeType = "xs"
	FlexSpacerSizeTypeSm  FlexSpacerSizeType = "sm"
	FlexSpacerSizeTypeMd  FlexSpacerSizeType = "md"
	FlexSpacerSizeTypeLg  FlexSpacerSizeType = "lg"
	FlexSpacerSizeTypeXl  FlexSpacerSizeType = "xl"
	FlexSpacerSizeTypeXxl FlexSpacerSizeType = "xxl"
)

// FlexTextWeightType type
type FlexTextWeightType string

// FlexTextWeightType constants
const (
	FlexTextWeightTypeRegular FlexTextWeightType = "regular"
	FlexTextWeightTypeBold    FlexTextWeightType = "bold"
)

// FlexTextSizeType type
type FlexTextSizeType string

// FlexTextSizeType constants
const (
	FlexTextSizeTypeXxs FlexTextSizeType = "xxs"
	FlexTextSizeTypeXs  FlexTextSizeType = "xs"
	FlexTextSizeTypeSm  FlexTextSizeType = "sm"
	FlexTextSizeTypeMd  FlexTextSizeType = "md"
	FlexTextSizeTypeLg  FlexTextSizeType = "lg"
	FlexTextSizeTypeXl  FlexTextSizeType = "xl"
	FlexTextSizeTypeXxl FlexTextSizeType = "xxl"
	FlexTextSizeType3xl FlexTextSizeType = "3xl"
	FlexTextSizeType4xl FlexTextSizeType = "4xl"
	FlexTextSizeType5xl FlexTextSizeType = "5xl"
)

// FlexTextStyleType type
type FlexTextStyleType string

// FlexTextStyleType constants
const (
	FlexTextStyleTypeNormal FlexTextStyleType = "normal"
	FlexTextStyleTypeItalic FlexTextStyleType = "italic"
)

// FlexTextDecorationType type
type FlexTextDecorationType string

// FlexTextDecorationType constants
const (
	FlexTextDecorationTypeNone        FlexTextDecorationType = "none"
	FlexTextDecorationTypeUnderline   FlexTextDecorationType = "underline"
	FlexTextDecorationTypeLineThrough FlexTextDecorationType = "line-through"
)

// FlexContainer interface
type FlexContainer interface {
	FlexContainer()
}

// BubbleContainer type
type BubbleContainer struct {
	Type      FlexContainerType
	Size      FlexBubbleSizeType
	Direction FlexBubbleDirectionType
	Header    *BoxComponent
	Hero      *ImageComponent
	Body      *BoxComponent
	Footer    *BoxComponent
	Styles    *BubbleStyle
}

// MarshalJSON method of BubbleContainer
func (c *BubbleContainer) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type      FlexContainerType       `json:"type"`
		Size      FlexBubbleSizeType      `json:"size,omitempty"`
		Direction FlexBubbleDirectionType `json:"direction,omitempty"`
		Header    *BoxComponent           `json:"header,omitempty"`
		Hero      *ImageComponent         `json:"hero,omitempty"`
		Body      *BoxComponent           `json:"body,omitempty"`
		Footer    *BoxComponent           `json:"footer,omitempty"`
		Styles    *BubbleStyle            `json:"styles,omitempty"`
	}{
		Type:      FlexContainerTypeBubble,
		Size:      c.Size,
		Direction: c.Direction,
		Header:    c.Header,
		Hero:      c.Hero,
		Body:      c.Body,
		Footer:    c.Footer,
		Styles:    c.Styles,
	})
}

// CarouselContainer type
type CarouselContainer struct {
	Type     FlexContainerType
	Contents []*BubbleContainer
}

// MarshalJSON method of CarouselContainer
func (c *CarouselContainer) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type     FlexContainerType  `json:"type"`
		Contents []*BubbleContainer `json:"contents"`
	}{
		Type:     FlexContainerTypeCarousel,
		Contents: c.Contents,
	})
}

// FlexContainer implements FlexContainer interface
func (*BubbleContainer) FlexContainer() {}

// FlexContainer implements FlexContainer interface
func (*CarouselContainer) FlexContainer() {}

// BubbleStyle type
type BubbleStyle struct {
	Header *BlockStyle `json:"header,omitempty"`
	Hero   *BlockStyle `json:"hero,omitempty"`
	Body   *BlockStyle `json:"body,omitempty"`
	Footer *BlockStyle `json:"footer,omitempty"`
}

// BlockStyle type
type BlockStyle struct {
	BackgroundColor string `json:"backgroundColor,omitempty"`
	Separator       bool   `json:"separator,omitempty"`
	SeparatorColor  string `json:"separatorColor,omitempty"`
}

// FlexComponent interface
type FlexComponent interface {
	FlexComponent()
}

// BoxComponent type
type BoxComponent struct {
	Type            FlexComponentType
	Layout          FlexBoxLayoutType
	Contents        []FlexComponent
	Flex            *int
	Spacing         FlexComponentSpacingType
	Margin          FlexComponentMarginType
	Width           string
	Height          string
	CornerRadius    FlexComponentCornerRadiusType
	BackgroundColor string
	BorderColor     string
	Action          TemplateAction
}

// MarshalJSON method of BoxComponent
func (c *BoxComponent) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type            FlexComponentType             `json:"type"`
		Layout          FlexBoxLayoutType             `json:"layout"`
		Contents        []FlexComponent               `json:"contents"`
		Flex            *int                          `json:"flex,omitempty"`
		Spacing         FlexComponentSpacingType      `json:"spacing,omitempty"`
		Margin          FlexComponentMarginType       `json:"margin,omitempty"`
		Width           string                        `json:"width,omitempty"`
		Height          string                        `json:"height,omitempty"`
		CornerRadius    FlexComponentCornerRadiusType `json:"cornerRadius,omitempty"`
		BackgroundColor string                        `json:"backgroundColor,omitempty"`
		BorderColor     string                        `json:"borderColor,omitempty"`
		Action          TemplateAction                `json:"action,omitempty"`
	}{
		Type:            FlexComponentTypeBox,
		Layout:          c.Layout,
		Contents:        c.Contents,
		Flex:            c.Flex,
		Spacing:         c.Spacing,
		Margin:          c.Margin,
		Width:           c.Width,
		Height:          c.Height,
		CornerRadius:    c.CornerRadius,
		BackgroundColor: c.BackgroundColor,
		BorderColor:     c.BorderColor,
		Action:          c.Action,
	})
}

// ButtonComponent type
type ButtonComponent struct {
	Type    FlexComponentType
	Action  TemplateAction
	Flex    *int
	Margin  FlexComponentMarginType
	Height  FlexButtonHeightType
	Style   FlexButtonStyleType
	Color   string
	Gravity FlexComponentGravityType
}

// MarshalJSON method of ButtonComponent
func (c *ButtonComponent) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type    FlexComponentType        `json:"type"`
		Action  TemplateAction           `json:"action"`
		Flex    *int                     `json:"flex,omitempty"`
		Margin  FlexComponentMarginType  `json:"margin,omitempty"`
		Height  FlexButtonHeightType     `json:"height,omitempty"`
		Style   FlexButtonStyleType      `json:"style,omitempty"`
		Color   string                   `json:"color,omitempty"`
		Gravity FlexComponentGravityType `json:"gravity,omitempty"`
	}{
		Type:    FlexComponentTypeButton,
		Action:  c.Action,
		Flex:    c.Flex,
		Margin:  c.Margin,
		Height:  c.Height,
		Style:   c.Style,
		Color:   c.Color,
		Gravity: c.Gravity,
	})
}

// FillerComponent type
type FillerComponent struct {
	Type FlexComponentType
	Flex *int
}

// MarshalJSON method of FillerComponent
func (c *FillerComponent) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type FlexComponentType `json:"type"`
		Flex *int              `json:"flex,omitempty"`
	}{
		Type: FlexComponentTypeFiller,
		Flex: c.Flex,
	})
}

// IconComponent type
type IconComponent struct {
	Type        FlexComponentType
	URL         string
	Margin      FlexComponentMarginType
	Size        FlexIconSizeType
	AspectRatio FlexIconAspectRatioType
}

// MarshalJSON method of IconComponent
func (c *IconComponent) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type        FlexComponentType       `json:"type"`
		URL         string                  `json:"url"`
		Margin      FlexComponentMarginType `json:"margin,omitempty"`
		Size        FlexIconSizeType        `json:"size,omitempty"`
		AspectRatio FlexIconAspectRatioType `json:"aspectRatio,omitempty"`
	}{
		Type:        FlexComponentTypeIcon,
		URL:         c.URL,
		Margin:      c.Margin,
		Size:        c.Size,
		AspectRatio: c.AspectRatio,
	})
}

// ImageComponent type
type ImageComponent struct {
	Type            FlexComponentType
	URL             string
	Flex            *int
	Margin          FlexComponentMarginType
	Align           FlexComponentAlignType
	Gravity         FlexComponentGravityType
	Size            FlexImageSizeType
	AspectRatio     FlexImageAspectRatioType
	AspectMode      FlexImageAspectModeType
	BackgroundColor string
	Action          TemplateAction
}

// MarshalJSON method of ImageComponent
func (c *ImageComponent) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type            FlexComponentType        `json:"type"`
		URL             string                   `json:"url"`
		Flex            *int                     `json:"flex,omitempty"`
		Margin          FlexComponentMarginType  `json:"margin,omitempty"`
		Align           FlexComponentAlignType   `json:"align,omitempty"`
		Gravity         FlexComponentGravityType `json:"gravity,omitempty"`
		Size            FlexImageSizeType        `json:"size,omitempty"`
		AspectRatio     FlexImageAspectRatioType `json:"aspectRatio,omitempty"`
		AspectMode      FlexImageAspectModeType  `json:"aspectMode,omitempty"`
		BackgroundColor string                   `json:"backgroundColor,omitempty"`
		Action          TemplateAction           `json:"action,omitempty"`
	}{
		Type:            FlexComponentTypeImage,
		URL:             c.URL,
		Flex:            c.Flex,
		Margin:          c.Margin,
		Align:           c.Align,
		Gravity:         c.Gravity,
		Size:            c.Size,
		AspectRatio:     c.AspectRatio,
		AspectMode:      c.AspectMode,
		BackgroundColor: c.BackgroundColor,
		Action:          c.Action,
	})
}

// SeparatorComponent type
type SeparatorComponent struct {
	Type   FlexComponentType
	Margin FlexComponentMarginType
	Color  string
}

// MarshalJSON method of SeparatorComponent
func (c *SeparatorComponent) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type   FlexComponentType       `json:"type"`
		Margin FlexComponentMarginType `json:"margin,omitempty"`
		Color  string                  `json:"color,omitempty"`
	}{
		Type:   FlexComponentTypeSeparator,
		Margin: c.Margin,
		Color:  c.Color,
	})
}

// SpacerComponent type
type SpacerComponent struct {
	Type FlexComponentType
	Size FlexSpacerSizeType
}

// MarshalJSON method of SpacerComponent
func (c *SpacerComponent) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type FlexComponentType  `json:"type"`
		Size FlexSpacerSizeType `json:"size,omitempty"`
	}{
		Type: FlexComponentTypeSpacer,
		Size: c.Size,
	})
}

// SpanComponent type
type SpanComponent struct {
	Type       FlexComponentType
	Text       string
	Size       FlexTextSizeType
	Weight     FlexTextWeightType
	Color      string
	Style      FlexTextStyleType
	Decoration FlexTextDecorationType
}

// MarshalJSON method of SpanComponent
func (c *SpanComponent) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type       FlexComponentType      `json:"type"`
		Text       string                 `json:"text,omitempty"`
		Size       FlexTextSizeType       `json:"size,omitempty"`
		Weight     FlexTextWeightType     `json:"weight,omitempty"`
		Color      string                 `json:"color,omitempty"`
		Style      FlexTextStyleType      `json:"style,omitempty"`
		Decoration FlexTextDecorationType `json:"decoration,omitempty"`
	}{
		Type:       FlexComponentTypeSpan,
		Text:       c.Text,
		Size:       c.Size,
		Weight:     c.Weight,
		Color:      c.Color,
		Style:      c.Style,
		Decoration: c.Decoration,
	})
}

// TextComponent type
type TextComponent struct {
	Type       FlexComponentType
	Text       string
	Contents   []*SpanComponent
	Flex       *int
	Margin     FlexComponentMarginType
	Size       FlexTextSizeType
	Align      FlexComponentAlignType
	Gravity    FlexComponentGravityType
	Wrap       bool
	Weight     FlexTextWeightType
	Color      string
	Action     TemplateAction
	Style      FlexTextStyleType
	Decoration FlexTextDecorationType
	MaxLines   *int
}

// MarshalJSON method of TextComponent
func (c *TextComponent) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type       FlexComponentType        `json:"type"`
		Text       string                   `json:"text,omitempty"`
		Contents   []*SpanComponent         `json:"contents,omitempty"`
		Flex       *int                     `json:"flex,omitempty"`
		Margin     FlexComponentMarginType  `json:"margin,omitempty"`
		Size       FlexTextSizeType         `json:"size,omitempty"`
		Align      FlexComponentAlignType   `json:"align,omitempty"`
		Gravity    FlexComponentGravityType `json:"gravity,omitempty"`
		Wrap       bool                     `json:"wrap,omitempty"`
		Weight     FlexTextWeightType       `json:"weight,omitempty"`
		Color      string                   `json:"color,omitempty"`
		Action     TemplateAction           `json:"action,omitempty"`
		Style      FlexTextStyleType        `json:"style,omitempty"`
		Decoration FlexTextDecorationType   `json:"decoration,omitempty"`
		MaxLines   *int                     `json:"maxLines,omitempty"`
	}{
		Type:       FlexComponentTypeText,
		Text:       c.Text,
		Contents:   c.Contents,
		Flex:       c.Flex,
		Margin:     c.Margin,
		Size:       c.Size,
		Align:      c.Align,
		Gravity:    c.Gravity,
		Wrap:       c.Wrap,
		Weight:     c.Weight,
		Color:      c.Color,
		Action:     c.Action,
		Style:      c.Style,
		Decoration: c.Decoration,
		MaxLines:   c.MaxLines,
	})
}

// FlexComponent implements FlexComponent interface
func (*BoxComponent) FlexComponent() {}

// FlexComponent implements FlexComponent interface
func (*ButtonComponent) FlexComponent() {}

// FlexComponent implements FlexComponent interface
func (*FillerComponent) FlexComponent() {}

// FlexComponent implements FlexComponent interface
func (*IconComponent) FlexComponent() {}

// FlexComponent implements FlexComponent interface
func (*ImageComponent) FlexComponent() {}

// FlexComponent implements FlexComponent interface
func (*SeparatorComponent) FlexComponent() {}

// FlexComponent implements FlexComponent interface
func (*SpacerComponent) FlexComponent() {}

// FlexComponent implements FlexComponent interface
func (*SpanComponent) FlexComponent() {}

// FlexComponent implements FlexComponent interface
func (*TextComponent) FlexComponent() {}
