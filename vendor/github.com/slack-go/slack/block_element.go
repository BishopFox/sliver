package slack

// https://api.slack.com/reference/messaging/block-elements

const (
	METCheckboxGroups MessageElementType = "checkboxes"
	METImage          MessageElementType = "image"
	METButton         MessageElementType = "button"
	METOverflow       MessageElementType = "overflow"
	METDatepicker     MessageElementType = "datepicker"
	METTimepicker     MessageElementType = "timepicker"
	METDatetimepicker MessageElementType = "datetimepicker"
	METPlainTextInput MessageElementType = "plain_text_input"
	METRadioButtons   MessageElementType = "radio_buttons"
	METRichTextInput  MessageElementType = "rich_text_input"
	METEmailTextInput MessageElementType = "email_text_input"
	METURLTextInput   MessageElementType = "url_text_input"
	METNumber         MessageElementType = "number_input"
	METFileInput      MessageElementType = "file_input"

	MixedElementImage MixedElementType = "mixed_image"
	MixedElementText  MixedElementType = "mixed_text"

	OptTypeStatic        string = "static_select"
	OptTypeExternal      string = "external_select"
	OptTypeUser          string = "users_select"
	OptTypeConversations string = "conversations_select"
	OptTypeChannels      string = "channels_select"

	MultiOptTypeStatic        string = "multi_static_select"
	MultiOptTypeExternal      string = "multi_external_select"
	MultiOptTypeUser          string = "multi_users_select"
	MultiOptTypeConversations string = "multi_conversations_select"
	MultiOptTypeChannels      string = "multi_channels_select"
)

type MessageElementType string
type MixedElementType string

// BlockElement defines an interface that all block element types should implement.
type BlockElement interface {
	ElementType() MessageElementType
}

type MixedElement interface {
	MixedElementType() MixedElementType
}

type Accessory struct {
	ImageElement               *ImageBlockElement
	ButtonElement              *ButtonBlockElement
	OverflowElement            *OverflowBlockElement
	DatePickerElement          *DatePickerBlockElement
	TimePickerElement          *TimePickerBlockElement
	PlainTextInputElement      *PlainTextInputBlockElement
	RichTextInputElement       *RichTextInputBlockElement
	RadioButtonsElement        *RadioButtonsBlockElement
	SelectElement              *SelectBlockElement
	MultiSelectElement         *MultiSelectBlockElement
	CheckboxGroupsBlockElement *CheckboxGroupsBlockElement
	UnknownElement             *UnknownBlockElement
}

// NewAccessory returns a new Accessory for a given block element
func NewAccessory(element BlockElement) *Accessory {
	switch element.(type) {
	case *ImageBlockElement:
		return &Accessory{ImageElement: element.(*ImageBlockElement)}
	case *ButtonBlockElement:
		return &Accessory{ButtonElement: element.(*ButtonBlockElement)}
	case *OverflowBlockElement:
		return &Accessory{OverflowElement: element.(*OverflowBlockElement)}
	case *DatePickerBlockElement:
		return &Accessory{DatePickerElement: element.(*DatePickerBlockElement)}
	case *TimePickerBlockElement:
		return &Accessory{TimePickerElement: element.(*TimePickerBlockElement)}
	case *PlainTextInputBlockElement:
		return &Accessory{PlainTextInputElement: element.(*PlainTextInputBlockElement)}
	case *RichTextInputBlockElement:
		return &Accessory{RichTextInputElement: element.(*RichTextInputBlockElement)}
	case *RadioButtonsBlockElement:
		return &Accessory{RadioButtonsElement: element.(*RadioButtonsBlockElement)}
	case *SelectBlockElement:
		return &Accessory{SelectElement: element.(*SelectBlockElement)}
	case *MultiSelectBlockElement:
		return &Accessory{MultiSelectElement: element.(*MultiSelectBlockElement)}
	case *CheckboxGroupsBlockElement:
		return &Accessory{CheckboxGroupsBlockElement: element.(*CheckboxGroupsBlockElement)}
	default:
		return &Accessory{UnknownElement: element.(*UnknownBlockElement)}
	}
}

// BlockElements is a convenience struct defined to allow dynamic unmarshalling of
// the "elements" value in Slack's JSON response, which varies depending on BlockElement type
type BlockElements struct {
	ElementSet []BlockElement `json:"elements,omitempty"`
}

// UnknownBlockElement any block element that this library does not directly support.
// See the "Rich Elements" section at the following URL:
// https://api.slack.com/changelog/2019-09-what-they-see-is-what-you-get-and-more-and-less
// New block element types may be introduced by Slack at any time; this is a catch-all for any such block elements.
type UnknownBlockElement struct {
	Type     MessageElementType `json:"type"`
	Elements BlockElements
}

// ElementType returns the type of the Element
func (s UnknownBlockElement) ElementType() MessageElementType {
	return s.Type
}

// ImageBlockElement An element to insert an image - this element can be used
// in section and context blocks only. If you want a block with only an image
// in it, you're looking for the image block.
//
// More Information: https://api.slack.com/reference/messaging/block-elements#image
type ImageBlockElement struct {
	Type      MessageElementType `json:"type"`
	ImageURL  string             `json:"image_url"`
	AltText   string             `json:"alt_text"`
	SlackFile *SlackFileObject   `json:"slack_file,omitempty"`
}

// ElementType returns the type of the Element
func (s ImageBlockElement) ElementType() MessageElementType {
	return s.Type
}

func (s ImageBlockElement) MixedElementType() MixedElementType {
	return MixedElementImage
}

// NewImageBlockElement returns a new instance of an image block element
func NewImageBlockElement(imageURL, altText string) *ImageBlockElement {
	return &ImageBlockElement{
		Type:     METImage,
		ImageURL: imageURL,
		AltText:  altText,
	}
}

// NewImageBlockElementSlackFile returns a new instance of an image block element
// TODO: BREAKING CHANGE - This should be combined with the function above
func NewImageBlockElementSlackFile(slackFile *SlackFileObject, altText string) *ImageBlockElement {
	return &ImageBlockElement{
		Type:      METImage,
		SlackFile: slackFile,
		AltText:   altText,
	}
}

// Style is a style of Button element
// https://api.slack.com/reference/block-kit/block-elements#button__fields
type Style string

const (
	StyleDefault Style = ""
	StylePrimary Style = "primary"
	StyleDanger  Style = "danger"
)

// ButtonBlockElement defines an interactive element that inserts a button. The
// button can be a trigger for anything from opening a simple link to starting
// a complex workflow.
//
// More Information: https://api.slack.com/reference/messaging/block-elements#button
type ButtonBlockElement struct {
	Type     MessageElementType       `json:"type,omitempty"`
	Text     *TextBlockObject         `json:"text"`
	ActionID string                   `json:"action_id,omitempty"`
	URL      string                   `json:"url,omitempty"`
	Value    string                   `json:"value,omitempty"`
	Confirm  *ConfirmationBlockObject `json:"confirm,omitempty"`
	Style    Style                    `json:"style,omitempty"`
}

// ElementType returns the type of the element
func (s ButtonBlockElement) ElementType() MessageElementType {
	return s.Type
}

// WithStyle adds styling to the button object and returns the modified ButtonBlockElement
func (s *ButtonBlockElement) WithStyle(style Style) *ButtonBlockElement {
	s.Style = style
	return s
}

// WithConfirm adds a confirmation dialogue to the button object and returns the modified ButtonBlockElement
func (s *ButtonBlockElement) WithConfirm(confirm *ConfirmationBlockObject) *ButtonBlockElement {
	s.Confirm = confirm
	return s
}

// WithURL adds a URL for the button to link to and returns the modified ButtonBlockElement
func (s *ButtonBlockElement) WithURL(url string) *ButtonBlockElement {
	s.URL = url
	return s
}

// NewButtonBlockElement returns an instance of a new button element to be used within a block
func NewButtonBlockElement(actionID, value string, text *TextBlockObject) *ButtonBlockElement {
	return &ButtonBlockElement{
		Type:     METButton,
		ActionID: actionID,
		Text:     text,
		Value:    value,
	}
}

// OptionsResponse defines the response used for select block typahead.
//
// More Information: https://api.slack.com/reference/block-kit/block-elements#external_multi_select
type OptionsResponse struct {
	Options []*OptionBlockObject `json:"options,omitempty"`
}

// OptionGroupsResponse defines the response used for select block typahead.
//
// More Information: https://api.slack.com/reference/block-kit/block-elements#external_multi_select
type OptionGroupsResponse struct {
	OptionGroups []*OptionGroupBlockObject `json:"option_groups,omitempty"`
}

// SelectBlockElement defines the simplest form of select menu, with a static list
// of options passed in when defining the element.
//
// More Information: https://api.slack.com/reference/messaging/block-elements#select
type SelectBlockElement struct {
	Type                         string                    `json:"type,omitempty"`
	Placeholder                  *TextBlockObject          `json:"placeholder,omitempty"`
	ActionID                     string                    `json:"action_id,omitempty"`
	Options                      []*OptionBlockObject      `json:"options,omitempty"`
	OptionGroups                 []*OptionGroupBlockObject `json:"option_groups,omitempty"`
	InitialOption                *OptionBlockObject        `json:"initial_option,omitempty"`
	InitialUser                  string                    `json:"initial_user,omitempty"`
	InitialConversation          string                    `json:"initial_conversation,omitempty"`
	InitialChannel               string                    `json:"initial_channel,omitempty"`
	DefaultToCurrentConversation bool                      `json:"default_to_current_conversation,omitempty"`
	ResponseURLEnabled           bool                      `json:"response_url_enabled,omitempty"`
	Filter                       *SelectBlockElementFilter `json:"filter,omitempty"`
	MinQueryLength               *int                      `json:"min_query_length,omitempty"`
	Confirm                      *ConfirmationBlockObject  `json:"confirm,omitempty"`
}

// SelectBlockElementFilter allows to filter select element conversation options by type.
//
// More Information: https://api.slack.com/reference/block-kit/composition-objects#filter_conversations
type SelectBlockElementFilter struct {
	Include                       []string `json:"include,omitempty"`
	ExcludeExternalSharedChannels bool     `json:"exclude_external_shared_channels,omitempty"`
	ExcludeBotUsers               bool     `json:"exclude_bot_users,omitempty"`
}

// ElementType returns the type of the Element
func (s SelectBlockElement) ElementType() MessageElementType {
	return MessageElementType(s.Type)
}

// NewOptionsSelectBlockElement returns a new instance of SelectBlockElement for use with
// the Options object only.
func NewOptionsSelectBlockElement(optType string, placeholder *TextBlockObject, actionID string, options ...*OptionBlockObject) *SelectBlockElement {
	return &SelectBlockElement{
		Type:        optType,
		Placeholder: placeholder,
		ActionID:    actionID,
		Options:     options,
	}
}

// WithInitialOption sets the initial option for the select element
func (s *SelectBlockElement) WithInitialOption(option *OptionBlockObject) *SelectBlockElement {
	s.InitialOption = option
	return s
}

// WithInitialUser sets the initial user for the select element
func (s *SelectBlockElement) WithInitialUser(user string) *SelectBlockElement {
	s.InitialUser = user
	return s
}

// WithInitialConversation sets the initial conversation for the select element
func (s *SelectBlockElement) WithInitialConversation(conversation string) *SelectBlockElement {
	s.InitialConversation = conversation
	return s
}

// WithInitialChannel sets the initial channel for the select element
func (s *SelectBlockElement) WithInitialChannel(channel string) *SelectBlockElement {
	s.InitialChannel = channel
	return s
}

// WithConfirm adds a confirmation dialogue to the select element
func (s *SelectBlockElement) WithConfirm(confirm *ConfirmationBlockObject) *SelectBlockElement {
	s.Confirm = confirm
	return s
}

// NewOptionsGroupSelectBlockElement returns a new instance of SelectBlockElement for use with
// the Options object only.
func NewOptionsGroupSelectBlockElement(
	optType string,
	placeholder *TextBlockObject,
	actionID string,
	optGroups ...*OptionGroupBlockObject,
) *SelectBlockElement {
	return &SelectBlockElement{
		Type:         optType,
		Placeholder:  placeholder,
		ActionID:     actionID,
		OptionGroups: optGroups,
	}
}

// MultiSelectBlockElement defines a multiselect menu, with a static list
// of options passed in when defining the element.
//
// More Information: https://api.slack.com/reference/messaging/block-elements#multi_select
type MultiSelectBlockElement struct {
	Type                 string                    `json:"type,omitempty"`
	Placeholder          *TextBlockObject          `json:"placeholder,omitempty"`
	ActionID             string                    `json:"action_id,omitempty"`
	Options              []*OptionBlockObject      `json:"options,omitempty"`
	OptionGroups         []*OptionGroupBlockObject `json:"option_groups,omitempty"`
	InitialOptions       []*OptionBlockObject      `json:"initial_options,omitempty"`
	InitialUsers         []string                  `json:"initial_users,omitempty"`
	InitialConversations []string                  `json:"initial_conversations,omitempty"`
	InitialChannels      []string                  `json:"initial_channels,omitempty"`
	Filter               *SelectBlockElementFilter `json:"filter,omitempty"`
	MinQueryLength       *int                      `json:"min_query_length,omitempty"`
	MaxSelectedItems     *int                      `json:"max_selected_items,omitempty"`
	Confirm              *ConfirmationBlockObject  `json:"confirm,omitempty"`
}

// ElementType returns the type of the Element
func (s MultiSelectBlockElement) ElementType() MessageElementType {
	return MessageElementType(s.Type)
}

// NewOptionsMultiSelectBlockElement returns a new instance of SelectBlockElement for use with
// the Options object only.
func NewOptionsMultiSelectBlockElement(optType string, placeholder *TextBlockObject, actionID string, options ...*OptionBlockObject) *MultiSelectBlockElement {
	return &MultiSelectBlockElement{
		Type:        optType,
		Placeholder: placeholder,
		ActionID:    actionID,
		Options:     options,
	}
}

// WithInitialOptions sets the initial options for the multi-select element
func (s *MultiSelectBlockElement) WithInitialOptions(options ...*OptionBlockObject) *MultiSelectBlockElement {
	s.InitialOptions = options
	return s
}

// WithInitialUsers sets the initial users for the multi-select element
func (s *MultiSelectBlockElement) WithInitialUsers(users ...string) *MultiSelectBlockElement {
	s.InitialUsers = users
	return s
}

// WithInitialConversations sets the initial conversations for the multi-select element
func (s *MultiSelectBlockElement) WithInitialConversations(conversations ...string) *MultiSelectBlockElement {
	s.InitialConversations = conversations
	return s
}

// WithInitialChannels sets the initial channels for the multi-select element
func (s *MultiSelectBlockElement) WithInitialChannels(channels ...string) *MultiSelectBlockElement {
	s.InitialChannels = channels
	return s
}

// WithConfirm adds a confirmation dialogue to the multi-select element
func (s *MultiSelectBlockElement) WithConfirm(confirm *ConfirmationBlockObject) *MultiSelectBlockElement {
	s.Confirm = confirm
	return s
}

// WithMaxSelectedItems sets the maximum number of items that can be selected
func (s *MultiSelectBlockElement) WithMaxSelectedItems(maxSelectedItems int) *MultiSelectBlockElement {
	s.MaxSelectedItems = &maxSelectedItems
	return s
}

// WithMinQueryLength sets the minimum query length for the multi-select element
func (s *MultiSelectBlockElement) WithMinQueryLength(minQueryLength int) *MultiSelectBlockElement {
	s.MinQueryLength = &minQueryLength
	return s
}

// NewOptionsGroupMultiSelectBlockElement returns a new instance of MultiSelectBlockElement for use with
// the Options object only.
func NewOptionsGroupMultiSelectBlockElement(
	optType string,
	placeholder *TextBlockObject,
	actionID string,
	optGroups ...*OptionGroupBlockObject,
) *MultiSelectBlockElement {
	return &MultiSelectBlockElement{
		Type:         optType,
		Placeholder:  placeholder,
		ActionID:     actionID,
		OptionGroups: optGroups,
	}
}

// OverflowBlockElement defines the fields needed to use an overflow element.
// And Overflow Element is like a cross between a button and a select menu -
// when a user clicks on this overflow button, they will be presented with a
// list of options to choose from.
//
// More Information: https://api.slack.com/reference/messaging/block-elements#overflow
type OverflowBlockElement struct {
	Type     MessageElementType       `json:"type"`
	ActionID string                   `json:"action_id,omitempty"`
	Options  []*OptionBlockObject     `json:"options"`
	Confirm  *ConfirmationBlockObject `json:"confirm,omitempty"`
}

// ElementType returns the type of the Element
func (s OverflowBlockElement) ElementType() MessageElementType {
	return s.Type
}

// NewOverflowBlockElement returns an instance of a new Overflow Block Element
func NewOverflowBlockElement(actionID string, options ...*OptionBlockObject) *OverflowBlockElement {
	return &OverflowBlockElement{
		Type:     METOverflow,
		ActionID: actionID,
		Options:  options,
	}
}

// WithConfirm adds a confirmation dialogue to the overflow element
func (s *OverflowBlockElement) WithConfirm(confirm *ConfirmationBlockObject) *OverflowBlockElement {
	s.Confirm = confirm
	return s
}

// DatePickerBlockElement defines an element which lets users easily select a
// date from a calendar style UI. Date picker elements can be used inside of
// section and actions blocks.
//
// More Information: https://api.slack.com/reference/messaging/block-elements#datepicker
type DatePickerBlockElement struct {
	Type        MessageElementType       `json:"type"`
	ActionID    string                   `json:"action_id,omitempty"`
	Placeholder *TextBlockObject         `json:"placeholder,omitempty"`
	InitialDate string                   `json:"initial_date,omitempty"`
	Confirm     *ConfirmationBlockObject `json:"confirm,omitempty"`
}

// ElementType returns the type of the Element
func (s DatePickerBlockElement) ElementType() MessageElementType {
	return s.Type
}

// NewDatePickerBlockElement returns an instance of a date picker element
func NewDatePickerBlockElement(actionID string) *DatePickerBlockElement {
	return &DatePickerBlockElement{
		Type:     METDatepicker,
		ActionID: actionID,
	}
}

// TimePickerBlockElement defines an element which lets users easily select a
// time from nice UI. Time picker elements can be used inside of
// section and actions blocks.
//
// More Information: https://api.slack.com/reference/messaging/block-elements#timepicker
type TimePickerBlockElement struct {
	Type        MessageElementType       `json:"type"`
	ActionID    string                   `json:"action_id,omitempty"`
	Placeholder *TextBlockObject         `json:"placeholder,omitempty"`
	InitialTime string                   `json:"initial_time,omitempty"`
	Confirm     *ConfirmationBlockObject `json:"confirm,omitempty"`
	Timezone    string                   `json:"timezone,omitempty"`
}

// ElementType returns the type of the Element
func (s TimePickerBlockElement) ElementType() MessageElementType {
	return s.Type
}

// NewTimePickerBlockElement returns an instance of a date picker element
func NewTimePickerBlockElement(actionID string) *TimePickerBlockElement {
	return &TimePickerBlockElement{
		Type:     METTimepicker,
		ActionID: actionID,
	}
}

// DateTimePickerBlockElement defines an element that allows the selection of both
// a date and a time of day formatted as a UNIX timestamp.
// More Information: https://api.slack.com/reference/messaging/block-elements#datetimepicker
type DateTimePickerBlockElement struct {
	Type            MessageElementType       `json:"type"`
	ActionID        string                   `json:"action_id,omitempty"`
	InitialDateTime int64                    `json:"initial_date_time,omitempty"`
	Confirm         *ConfirmationBlockObject `json:"confirm,omitempty"`
	FocusOnLoad     bool                     `json:"focus_on_load,omitempty"`
}

// ElementType returns the type of the Element
func (s DateTimePickerBlockElement) ElementType() MessageElementType {
	return s.Type
}

// NewDatePickerBlockElement returns an instance of a datetime picker element
func NewDateTimePickerBlockElement(actionID string) *DateTimePickerBlockElement {
	return &DateTimePickerBlockElement{
		Type:     METDatetimepicker,
		ActionID: actionID,
	}
}

// EmailTextInputBlockElement creates a field where a user can enter email
// data.
// email-text-input elements are currently only available in modals.
//
// More Information: https://api.slack.com/reference/block-kit/block-elements#email
type EmailTextInputBlockElement struct {
	Type                 MessageElementType    `json:"type"`
	ActionID             string                `json:"action_id,omitempty"`
	Placeholder          *TextBlockObject      `json:"placeholder,omitempty"`
	InitialValue         string                `json:"initial_value,omitempty"`
	DispatchActionConfig *DispatchActionConfig `json:"dispatch_action_config,omitempty"`
	FocusOnLoad          bool                  `json:"focus_on_load,omitempty"`
}

// ElementType returns the type of the Element
func (s EmailTextInputBlockElement) ElementType() MessageElementType {
	return s.Type
}

// NewEmailTextInputBlockElement returns an instance of a plain-text input
// element
func NewEmailTextInputBlockElement(placeholder *TextBlockObject, actionID string) *EmailTextInputBlockElement {
	return &EmailTextInputBlockElement{
		Type:        METEmailTextInput,
		ActionID:    actionID,
		Placeholder: placeholder,
	}
}

// URLTextInputBlockElement creates a field where a user can enter url data.
//
// url-text-input elements are currently only available in modals.
//
// More Information: https://api.slack.com/reference/block-kit/block-elements#url
type URLTextInputBlockElement struct {
	Type                 MessageElementType    `json:"type"`
	ActionID             string                `json:"action_id,omitempty"`
	Placeholder          *TextBlockObject      `json:"placeholder,omitempty"`
	InitialValue         string                `json:"initial_value,omitempty"`
	DispatchActionConfig *DispatchActionConfig `json:"dispatch_action_config,omitempty"`
	FocusOnLoad          bool                  `json:"focus_on_load,omitempty"`
}

// ElementType returns the type of the Element
func (s URLTextInputBlockElement) ElementType() MessageElementType {
	return s.Type
}

// NewURLTextInputBlockElement returns an instance of a plain-text input
// element
func NewURLTextInputBlockElement(placeholder *TextBlockObject, actionID string) *URLTextInputBlockElement {
	return &URLTextInputBlockElement{
		Type:        METURLTextInput,
		ActionID:    actionID,
		Placeholder: placeholder,
	}
}

// PlainTextInputBlockElement creates a field where a user can enter freeform
// data.
// Plain-text input elements are currently only available in modals.
//
// More Information: https://api.slack.com/reference/block-kit/block-elements#input
type PlainTextInputBlockElement struct {
	Type                 MessageElementType    `json:"type"`
	ActionID             string                `json:"action_id,omitempty"`
	Placeholder          *TextBlockObject      `json:"placeholder,omitempty"`
	InitialValue         string                `json:"initial_value,omitempty"`
	Multiline            bool                  `json:"multiline,omitempty"`
	MinLength            int                   `json:"min_length,omitempty"`
	MaxLength            int                   `json:"max_length,omitempty"`
	DispatchActionConfig *DispatchActionConfig `json:"dispatch_action_config,omitempty"`
}

type DispatchActionConfig struct {
	TriggerActionsOn []string `json:"trigger_actions_on,omitempty"`
}

// ElementType returns the type of the Element
func (s PlainTextInputBlockElement) ElementType() MessageElementType {
	return s.Type
}

// NewPlainTextInputBlockElement returns an instance of a plain-text input
// element
func NewPlainTextInputBlockElement(placeholder *TextBlockObject, actionID string) *PlainTextInputBlockElement {
	return &PlainTextInputBlockElement{
		Type:        METPlainTextInput,
		ActionID:    actionID,
		Placeholder: placeholder,
	}
}

// WithInitialValue sets the initial value for the plain-text input element
func (s *PlainTextInputBlockElement) WithInitialValue(initialValue string) *PlainTextInputBlockElement {
	s.InitialValue = initialValue
	return s
}

// WithMinLength sets the minimum length for the plain-text input element
func (s *PlainTextInputBlockElement) WithMinLength(minLength int) *PlainTextInputBlockElement {
	s.MinLength = minLength
	return s
}

// WithMaxLength sets the maximum length for the plain-text input element
func (s *PlainTextInputBlockElement) WithMaxLength(maxLength int) *PlainTextInputBlockElement {
	s.MaxLength = maxLength
	return s
}

// WithMultiline sets the multiline property for the plain-text input element
func (s *PlainTextInputBlockElement) WithMultiline(multiline bool) *PlainTextInputBlockElement {
	s.Multiline = multiline
	return s
}

// WithDispatchActionConfig sets the dispatch action config for the plain-text input element
func (s *PlainTextInputBlockElement) WithDispatchActionConfig(config *DispatchActionConfig) *PlainTextInputBlockElement {
	s.DispatchActionConfig = config
	return s
}

// RichTextInputBlockElement creates a field where allows users to enter formatted text
// in a WYSIWYG composer, offering the same messaging writing experience as in Slack
// More Information: https://api.slack.com/reference/block-kit/block-elements#rich_text_input
type RichTextInputBlockElement struct {
	Type                 MessageElementType    `json:"type"`
	ActionID             string                `json:"action_id,omitempty"`
	Placeholder          *TextBlockObject      `json:"placeholder,omitempty"`
	InitialValue         *RichTextBlock        `json:"initial_value,omitempty"`
	DispatchActionConfig *DispatchActionConfig `json:"dispatch_action_config,omitempty"`
	FocusOnLoad          bool                  `json:"focus_on_load,omitempty"`
}

// ElementType returns the type of the Element
func (s RichTextInputBlockElement) ElementType() MessageElementType {
	return s.Type
}

// NewRichTextInputBlockElement returns an instance of a rich-text input element
func NewRichTextInputBlockElement(placeholder *TextBlockObject, actionID string) *RichTextInputBlockElement {
	return &RichTextInputBlockElement{
		Type:        METRichTextInput,
		ActionID:    actionID,
		Placeholder: placeholder,
	}
}

// CheckboxGroupsBlockElement defines an element which allows users to choose
// one or more items from a list of possible options.
//
// More Information: https://api.slack.com/reference/block-kit/block-elements#checkboxes
type CheckboxGroupsBlockElement struct {
	Type           MessageElementType       `json:"type"`
	ActionID       string                   `json:"action_id,omitempty"`
	Options        []*OptionBlockObject     `json:"options"`
	InitialOptions []*OptionBlockObject     `json:"initial_options,omitempty"`
	Confirm        *ConfirmationBlockObject `json:"confirm,omitempty"`
}

// ElementType returns the type of the Element
func (c CheckboxGroupsBlockElement) ElementType() MessageElementType {
	return c.Type
}

// NewCheckboxGroupsBlockElement returns an instance of a checkbox-group block element
func NewCheckboxGroupsBlockElement(actionID string, options ...*OptionBlockObject) *CheckboxGroupsBlockElement {
	return &CheckboxGroupsBlockElement{
		Type:     METCheckboxGroups,
		ActionID: actionID,
		Options:  options,
	}
}

// RadioButtonsBlockElement defines an element which lets users choose one item
// from a list of possible options.
//
// More Information: https://api.slack.com/reference/block-kit/block-elements#radio
type RadioButtonsBlockElement struct {
	Type          MessageElementType       `json:"type"`
	ActionID      string                   `json:"action_id,omitempty"`
	Options       []*OptionBlockObject     `json:"options"`
	InitialOption *OptionBlockObject       `json:"initial_option,omitempty"`
	Confirm       *ConfirmationBlockObject `json:"confirm,omitempty"`
}

// ElementType returns the type of the Element
func (s RadioButtonsBlockElement) ElementType() MessageElementType {
	return s.Type
}

// NewRadioButtonsBlockElement returns an instance of a radio buttons element.
func NewRadioButtonsBlockElement(actionID string, options ...*OptionBlockObject) *RadioButtonsBlockElement {
	return &RadioButtonsBlockElement{
		Type:     METRadioButtons,
		ActionID: actionID,
		Options:  options,
	}
}

// NumberInputBlockElement creates a field where a user can enter number
// data.
// Number input elements are currently only available in modals.
//
// More Information: https://api.slack.com/reference/block-kit/block-elements#number
type NumberInputBlockElement struct {
	Type                 MessageElementType    `json:"type"`
	IsDecimalAllowed     bool                  `json:"is_decimal_allowed"`
	ActionID             string                `json:"action_id,omitempty"`
	Placeholder          *TextBlockObject      `json:"placeholder,omitempty"`
	InitialValue         string                `json:"initial_value,omitempty"`
	MinValue             string                `json:"min_value,omitempty"`
	MaxValue             string                `json:"max_value,omitempty"`
	DispatchActionConfig *DispatchActionConfig `json:"dispatch_action_config,omitempty"`
}

// ElementType returns the type of the Element
func (s NumberInputBlockElement) ElementType() MessageElementType {
	return s.Type
}

// NewNumberInputBlockElement returns an instance of a number input element
func NewNumberInputBlockElement(placeholder *TextBlockObject, actionID string, isDecimalAllowed bool) *NumberInputBlockElement {
	return &NumberInputBlockElement{
		Type:             METNumber,
		ActionID:         actionID,
		Placeholder:      placeholder,
		IsDecimalAllowed: isDecimalAllowed,
	}
}

// WithInitialValue sets the initial value for the number input element
func (s *NumberInputBlockElement) WithInitialValue(initialValue string) *NumberInputBlockElement {
	s.InitialValue = initialValue
	return s
}

// WithMinValue sets the minimum value for the number input element
func (s *NumberInputBlockElement) WithMinValue(minValue string) *NumberInputBlockElement {
	s.MinValue = minValue
	return s
}

// WithMaxValue sets the maximum value for the number input element
func (s *NumberInputBlockElement) WithMaxValue(maxValue string) *NumberInputBlockElement {
	s.MaxValue = maxValue
	return s
}

// WithDispatchActionConfig sets the dispatch action config for the number input element
func (s *NumberInputBlockElement) WithDispatchActionConfig(config *DispatchActionConfig) *NumberInputBlockElement {
	s.DispatchActionConfig = config
	return s
}

// FileInputBlockElement creates a field where a user can upload a file.
//
// File input elements are currently only available in modals.
//
// More Information: https://api.slack.com/reference/block-kit/block-elements#file_input
type FileInputBlockElement struct {
	Type      MessageElementType `json:"type"`
	ActionID  string             `json:"action_id,omitempty"`
	FileTypes []string           `json:"filetypes,omitempty"`
	MaxFiles  int                `json:"max_files,omitempty"`
}

// ElementType returns the type of the Element
func (s FileInputBlockElement) ElementType() MessageElementType {
	return s.Type
}

// NewFileInputBlockElement returns an instance of a file input element
func NewFileInputBlockElement(actionID string) *FileInputBlockElement {
	return &FileInputBlockElement{
		Type:     METFileInput,
		ActionID: actionID,
	}
}

// WithFileTypes sets the file types that can be uploaded
func (s *FileInputBlockElement) WithFileTypes(fileTypes ...string) *FileInputBlockElement {
	s.FileTypes = fileTypes
	return s
}

// WithMaxFiles sets the maximum number of files that can be uploaded
func (s *FileInputBlockElement) WithMaxFiles(maxFiles int) *FileInputBlockElement {
	s.MaxFiles = maxFiles
	return s
}
