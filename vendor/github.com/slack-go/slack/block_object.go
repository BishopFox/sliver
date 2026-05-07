package slack

import (
	"encoding/json"
	"errors"
)

// Block Objects are also known as Composition Objects
//
// For more information: https://api.slack.com/reference/messaging/composition-objects

// BlockObject defines an interface that all block object types should
// implement.
// @TODO: Is this interface needed?

// blockObject object types
const (
	MarkdownType  = "mrkdwn"
	PlainTextType = "plain_text"
	// The following objects don't actually have types and their corresponding
	// const values are just for internal use
	motConfirmation = "confirm"
	motOption       = "option"
	motOptionGroup  = "option_group"
)

type MessageObjectType string

type blockObject interface {
	validateType() MessageObjectType
}

type BlockObjects struct {
	TextObjects         []*TextBlockObject
	ConfirmationObjects []*ConfirmationBlockObject
	OptionObjects       []*OptionBlockObject
	OptionGroupObjects  []*OptionGroupBlockObject
}

// UnmarshalJSON implements the Unmarshaller interface for BlockObjects, so that any JSON
// unmarshalling is delegated and proper type determination can be made before unmarshal
func (b *BlockObjects) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}

	for _, r := range raw {
		var obj map[string]interface{}
		err := json.Unmarshal(r, &obj)
		if err != nil {
			return err
		}

		blockObjectType := getBlockObjectType(obj)

		switch blockObjectType {
		case PlainTextType, MarkdownType:
			object, err := unmarshalBlockObject(r, &TextBlockObject{})
			if err != nil {
				return err
			}
			b.TextObjects = append(b.TextObjects, object.(*TextBlockObject))
		case motConfirmation:
			object, err := unmarshalBlockObject(r, &ConfirmationBlockObject{})
			if err != nil {
				return err
			}
			b.ConfirmationObjects = append(b.ConfirmationObjects, object.(*ConfirmationBlockObject))
		case motOption:
			object, err := unmarshalBlockObject(r, &OptionBlockObject{})
			if err != nil {
				return err
			}
			b.OptionObjects = append(b.OptionObjects, object.(*OptionBlockObject))
		case motOptionGroup:
			object, err := unmarshalBlockObject(r, &OptionGroupBlockObject{})
			if err != nil {
				return err
			}
			b.OptionGroupObjects = append(b.OptionGroupObjects, object.(*OptionGroupBlockObject))

		}
	}

	return nil
}

// Ideally would have a better way to identify the block objects for
// type casting at time of unmarshalling, should be adapted if possible
// to accomplish in a more reliable manner.
func getBlockObjectType(obj map[string]interface{}) string {
	if t, ok := obj["type"].(string); ok {
		return t
	}
	if _, ok := obj["confirm"].(string); ok {
		return "confirm"
	}
	if _, ok := obj["options"].(string); ok {
		return "option_group"
	}
	if _, ok := obj["text"].(string); ok {
		if _, ok := obj["value"].(string); ok {
			return "option"
		}
	}
	return ""
}

func unmarshalBlockObject(r json.RawMessage, object blockObject) (blockObject, error) {
	err := json.Unmarshal(r, object)
	if err != nil {
		return nil, err
	}
	return object, nil
}

// TextBlockObject defines a text element object to be used with blocks
//
// More Information: https://api.slack.com/reference/messaging/composition-objects#text
type TextBlockObject struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	Emoji    *bool  `json:"emoji,omitempty"`
	Verbatim bool   `json:"verbatim,omitempty"`
}

// validateType enforces block objects for element and block parameters
func (s TextBlockObject) validateType() MessageObjectType {
	return MessageObjectType(s.Type)
}

// validateType enforces block objects for element and block parameters
func (s TextBlockObject) MixedElementType() MixedElementType {
	return MixedElementText
}

// Validate checks if TextBlockObject has valid values
func (s TextBlockObject) Validate() error {
	if s.Type != "plain_text" && s.Type != "mrkdwn" {
		return errors.New("type must be either of plain_text or mrkdwn")
	}

	if s.Type == "mrkdwn" && s.Emoji != nil {
		return errors.New("emoji cannot be set for mrkdwn type")
	}

	// https://api.slack.com/reference/block-kit/composition-objects#text__fields
	if len(s.Text) == 0 {
		return errors.New("text must have a minimum length of 1")
	}

	// https://api.slack.com/reference/block-kit/composition-objects#text__fields
	if len(s.Text) > 3000 {
		return errors.New("text cannot be longer than 3000 characters")
	}

	return nil
}

// NewTextBlockObject returns an instance of a new Text Block Object
//
// If you want to create a mrkdwn object, you should set the emoji parameter to false. The
// reason is that Slack doesn't accept emoji in mrkdwn.
func NewTextBlockObject(elementType, text string, emoji bool, verbatim bool) *TextBlockObject {
	// If we're trying to build a mrkdwn object, we can't send emoji at all. I think the
	// right approach here is to be a bit clever, and not break the function interface.
	//
	// So, here's the plan:
	// 1. If the type is mrkdwn, set emoji to nil, regardless of what the user passed in
	// 2. Else, set emoji to the value passed in
	var emojiPtr *bool

	if elementType == "mrkdwn" {
		emojiPtr = nil
	} else {
		emojiPtr = &emoji
	}

	return &TextBlockObject{
		Type:     elementType,
		Text:     text,
		Emoji:    emojiPtr,
		Verbatim: verbatim,
	}
}

// BlockType returns the type of the block
func (t TextBlockObject) BlockType() MessageBlockType {
	if t.Type == "mrkdwn" {
		return MarkdownType
	}
	return PlainTextType
}

// ConfirmationBlockObject defines a dialog that provides a confirmation step to
// any interactive element. This dialog will ask the user to confirm their action by
// offering a confirm and deny buttons.
//
// More Information: https://api.slack.com/reference/messaging/composition-objects#confirm
type ConfirmationBlockObject struct {
	Title   *TextBlockObject `json:"title"`
	Text    *TextBlockObject `json:"text"`
	Confirm *TextBlockObject `json:"confirm"`
	Deny    *TextBlockObject `json:"deny,omitempty"`
	Style   Style            `json:"style,omitempty"`
}

// validateType enforces block objects for element and block parameters
func (s ConfirmationBlockObject) validateType() MessageObjectType {
	return motConfirmation
}

// WithStyle add styling to confirmation object
func (s *ConfirmationBlockObject) WithStyle(style Style) *ConfirmationBlockObject {
	s.Style = style
	return s
}

// NewConfirmationBlockObject returns an instance of a new Confirmation Block Object
func NewConfirmationBlockObject(title, text, confirm, deny *TextBlockObject) *ConfirmationBlockObject {
	return &ConfirmationBlockObject{
		Title:   title,
		Text:    text,
		Confirm: confirm,
		Deny:    deny,
	}
}

// OptionBlockObject represents a single selectable item in a select menu
//
// More Information: https://api.slack.com/reference/messaging/composition-objects#option
type OptionBlockObject struct {
	Text        *TextBlockObject `json:"text"`
	Value       string           `json:"value"`
	Description *TextBlockObject `json:"description,omitempty"`
	URL         string           `json:"url,omitempty"`
}

// NewOptionBlockObject returns an instance of a new Option Block Element
func NewOptionBlockObject(value string, text, description *TextBlockObject) *OptionBlockObject {
	return &OptionBlockObject{
		Text:        text,
		Value:       value,
		Description: description,
	}
}

// validateType enforces block objects for element and block parameters
func (s OptionBlockObject) validateType() MessageObjectType {
	return motOption
}

// OptionGroupBlockObject Provides a way to group options in a select menu.
//
// More Information: https://api.slack.com/reference/messaging/composition-objects#option-group
type OptionGroupBlockObject struct {
	Label   *TextBlockObject     `json:"label,omitempty"`
	Options []*OptionBlockObject `json:"options"`
}

// validateType enforces block objects for element and block parameters
func (s OptionGroupBlockObject) validateType() MessageObjectType {
	return motOptionGroup
}

// NewOptionGroupBlockElement returns an instance of a new option group block element
func NewOptionGroupBlockElement(label *TextBlockObject, options ...*OptionBlockObject) *OptionGroupBlockObject {
	return &OptionGroupBlockObject{
		Label:   label,
		Options: options,
	}
}
