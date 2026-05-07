// Copyright 2022 Adam Chalkley
//
// https://github.com/atc0005/go-teams-notify
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package adaptivecard

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"

	goteamsnotify "github.com/atc0005/go-teams-notify/v2"
	"github.com/atc0005/go-teams-notify/v2/internal/validator"
)

// General constants.
const (
	// TypeMessage is the type for an Adaptive Card Message.
	TypeMessage string = "message"

	// PixelSizeRegex is a regular expression pattern intended to match
	// specific pixel size (height, width) values such as "50px".
	PixelSizeRegex string = "^[0-9]+px$"

	// PixelSizeExample is an example of a valid pixel size (height, width)
	// value.
	PixelSizeExample string = "50px"
)

// Card & TopLevelCard specific constants.
const (
	// TypeAdaptiveCard is the supported type value for an Adaptive Card.
	TypeAdaptiveCard string = "AdaptiveCard"

	// AdaptiveCardSchema represents the URI of the Adaptive Card schema.
	AdaptiveCardSchema string = "http://adaptivecards.io/schemas/adaptive-card.json"

	// AdaptiveCardMaxVersion represents the highest supported version of the
	// Adaptive Card schema supported in Microsoft Teams messages.
	//
	// https://docs.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-reference#support-for-adaptive-cards
	// https://adaptivecards.io/designer
	//
	// NOTE: Documented as 1.5 (adaptivecards.io/designer), but in practice >
	// 1.4 is rejected for Power Automate workflow connectors.
	//
	// Setting to 1.4 works both for legacy O365 connectors and Workflow
	// connectors.
	AdaptiveCardMaxVersion  float64 = 1.4
	AdaptiveCardMinVersion  float64 = 1.0
	AdaptiveCardVersionTmpl string  = "%0.1f"
)

// Mention constants.
const (
	// TypeMention is the type for a user mention for a Adaptive Card Message.
	TypeMention string = "mention"

	// MentionTextFormatTemplate is the expected format of the Mention.Text
	// field value.
	MentionTextFormatTemplate string = "<at>%s</at>"

	// defaultMentionTextSeparator is the default separator used between the
	// contents of the Mention.Text field and a TextBlock.Text field.
	defaultMentionTextSeparator string = " "
)

// Attachment constants.
//
//   - https://docs.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-reference
//   - https://docs.microsoft.com/en-us/dotnet/api/microsoft.bot.schema.attachmentlayouttypes
//   - https://docs.microsoft.com/en-us/javascript/api/botframework-schema/attachmentlayouttypes
//   - https://github.com/matthidinger/ContosoScubaBot/blob/master/Cards/1-Schools.JSON
const (

	// AttachmentContentType is the supported type value for an attached
	// Adaptive Card for a Microsoft Teams message.
	AttachmentContentType string = "application/vnd.microsoft.card.adaptive"

	AttachmentLayoutList     string = "list"
	AttachmentLayoutCarousel string = "carousel"
)

// TextBlock specific constants.
// https://adaptivecards.io/explorer/TextBlock.html
const (
	// TextBlockStyleDefault indicates that the TextBlock uses the default
	// style which provides no special styling or behavior.
	TextBlockStyleDefault string = "default"

	// TextBlockStyleHeading indicates that the TextBlock is a heading. This
	// will apply the heading styling defaults and mark the text block as a
	// heading for accessibility.
	TextBlockStyleHeading string = "heading"
)

// Column specific constants.
// https://adaptivecards.io/explorer/Column.html
const (
	// TypeColumn is the type for an Adaptive Card Column.
	TypeColumn string = "Column"

	// ColumnWidthAuto indicates that a column's width should be determined
	// automatically based on other columns in the column group.
	ColumnWidthAuto string = "auto"

	// ColumnWidthStretch indicates that a column's width should be stretched
	// to fill the enclosing column group.
	ColumnWidthStretch string = "stretch"
)

// Table specific constants.
//
// https://adaptivecards.io/explorer/Table.html
// https://adaptivecards.io/explorer/TableCell.html
const (

	// NOTE: Table is not a type, it is an Card Element
	// TypeTable     string = "Table"

	TypeTableColumnDefinition string = "TableColumnDefinition"
	TypeTableRow              string = "TableRow"
	TypeTableCell             string = "TableCell"
)

// Text size for TextBlock or TextRun elements.
const (
	SizeSmall      string = "small"
	SizeDefault    string = "default"
	SizeMedium     string = "medium"
	SizeLarge      string = "large"
	SizeExtraLarge string = "extraLarge"
)

// Text weight for TextBlock or TextRun elements.
const (
	WeightBolder  string = "bolder"
	WeightLighter string = "lighter"
	WeightDefault string = "default"
)

// Supported colors for TextBlock or TextRun elements.
const (
	ColorDefault   string = "default"
	ColorDark      string = "dark"
	ColorLight     string = "light"
	ColorAccent    string = "accent"
	ColorGood      string = "good"
	ColorWarning   string = "warning"
	ColorAttention string = "attention"
)

// Image specific constants.
// https://adaptivecards.io/explorer/Image.html
const (
	ImageStyleDefault string = ""
	ImageStylePerson  string = ""
)

// ChoiceInput specific constants.
const (
	ChoiceInputStyleCompact  string = "compact"
	ChoiceInputStyleExpanded string = "expanded"
	ChoiceInputStyleFiltered string = "filtered" // Introduced in version 1.5
)

// TextInput specific constants.
const (
	TextInputStyleText     string = "text"
	TextInputStyleTel      string = "tel"
	TextInputStyleURL      string = "url"
	TextInputStyleEmail    string = "email"
	TextInputStylePassword string = "password" // Introduced in version 1.5
)

// Container specific constants.
const (
	ContainerStyleDefault   string = "default"
	ContainerStyleEmphasis  string = "emphasis"
	ContainerStyleGood      string = "good"
	ContainerStyleAttention string = "attention"
	ContainerStyleWarning   string = "warning"
	ContainerStyleAccent    string = "accent"
)

// Supported spacing values for FactSet, Container and other container element
// types.
const (
	SpacingDefault    string = "default"
	SpacingNone       string = "none"
	SpacingSmall      string = "small"
	SpacingMedium     string = "medium"
	SpacingLarge      string = "large"
	SpacingExtraLarge string = "extraLarge"
	SpacingPadding    string = "padding"
)

// Supported Horizontal alignment values for (supported) container and text
// types.
const (
	HorizontalAlignmentLeft   string = "left"
	HorizontalAlignmentCenter string = "center"
	HorizontalAlignmentRight  string = "right"
)

// Supported Horizontal alignment values for (supported) container types.
const (
	VerticalAlignmentTop    string = "top"
	VerticalAlignmentCenter string = "center"
	VerticalAlignmentBottom string = "bottom"
)

// Supported width values for the msteams property used in in Adaptive Card
// messages sent via Microsoft Teams.
const (
	MSTeamsWidthFull string = "Full"
)

// Supported Actions
const (

	// TeamsActionsDisplayLimit is the observed limit on the number of visible
	// URL "buttons" in a Microsoft Teams message.
	//
	// Unlike the MessageCard format which has a clearly documented limit of 4
	// actions, testing reveals that Desktop / Web displays 6 without the
	// option to expand and see any additional defined actions. Mobile
	// displays 6 with an ellipsis to expand into a list of other Actions.
	//
	// This results in a maximum limit of 6 actions in the Actions array for a
	// Card.
	//
	// A workaround is to create multiple ActionSet elements and limit the
	// number of Actions in each set ot 6.
	//
	// https://docs.microsoft.com/en-us/outlook/actionable-messages/message-card-reference#actions
	TeamsActionsDisplayLimit int = 6

	// TypeActionExecute is an action that gathers input fields, merges with
	// optional data field, and sends an event to the client. Clients process
	// the event by sending an Invoke activity of type adaptiveCard/action to
	// the target Bot. The inputs that are gathered are those on the current
	// card, and in the case of a show card those on any parent cards. See
	// Universal Action Model documentation for more details:
	// https://docs.microsoft.com/en-us/adaptive-cards/authoring-cards/universal-action-model
	//
	// TypeActionExecute was introduced in Adaptive Cards schema version 1.4.
	// TypeActionExecute actions may not render with earlier versions of the
	// Teams client.
	TypeActionExecute string = "Action.Execute"

	// ActionExecuteMinCardVersionRequired is the minimum version of the
	// Adaptive Card schema required to support Action.Execute.
	ActionExecuteMinCardVersionRequired float64 = 1.4

	// TypeActionSubmit is used in Adaptive Cards schema version 1.3 and
	// earlier or as a fallback for TypeActionExecute in schema version 1.4.
	// TypeActionSubmit is not supported in Incoming Webhooks.
	TypeActionSubmit string = "Action.Submit"

	// TypeActionOpenURL (when invoked) shows the given url either by
	// launching it in an external web browser or showing within an embedded
	// web browser.
	TypeActionOpenURL string = "Action.OpenUrl"

	// TypeActionShowCard defines an AdaptiveCard which is shown to the user
	// when the button or link is clicked.
	TypeActionShowCard string = "Action.ShowCard"

	// TypeActionToggleVisibility toggles the visibility of associated card
	// elements.
	TypeActionToggleVisibility string = "Action.ToggleVisibility"
)

// Supported Fallback options.
const (
	TypeFallbackActionExecute          string = TypeActionExecute
	TypeFallbackActionOpenURL          string = TypeActionOpenURL
	TypeFallbackActionShowCard         string = TypeActionShowCard
	TypeFallbackActionSubmit           string = TypeActionSubmit
	TypeFallbackActionToggleVisibility string = TypeActionToggleVisibility

	// TypeFallbackOptionDrop causes this element to be dropped immediately
	// when unknown elements are encountered. The unknown element doesn't
	// bubble up any higher.
	TypeFallbackOptionDrop string = "drop"
)

// Valid types for an Adaptive Card element. Not all types are supported by
// Microsoft Teams.
//
// TODO: Confirm whether all types are supported.
//
//   - https://adaptivecards.io/explorer/AdaptiveCard.html
//   - https://docs.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-reference#support-for-adaptive-cards
//   - https://docs.microsoft.com/en-us/adaptive-cards/authoring-cards/universal-action-model#schema
const (
	TypeElementActionSet      string = "ActionSet"
	TypeElementColumnSet      string = "ColumnSet"
	TypeElementContainer      string = "Container"
	TypeElementFactSet        string = "FactSet"
	TypeElementImage          string = "Image"
	TypeElementImageSet       string = "ImageSet"
	TypeElementInputChoiceSet string = "Input.ChoiceSet"
	TypeElementInputDate      string = "Input.Date"
	TypeElementInputNumber    string = "Input.Number"
	TypeElementInputText      string = "Input.Text"
	TypeElementInputTime      string = "Input.Time"
	TypeElementInputToggle    string = "Input.Toggle"
	TypeElementMedia          string = "Media"         // Introduced in version 1.1 (TODO: Is this supported in Teams message?)
	TypeElementRichTextBlock  string = "RichTextBlock" // Introduced in version 1.2
	TypeElementTable          string = "Table"         // Introduced in version 1.5
	TypeElementTextBlock      string = "TextBlock"
	TypeElementTextRun        string = "TextRun" // Introduced in version 1.2
)

// Known extension types for an Adaptive Card element.
//
//   - https://learn.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-format?tabs=adaptive-md%2Cdesktop%2Cconnector-html#codeblock-in-adaptive-cards
const (
	TypeElementMSTeamsCodeBlock string = "CodeBlock"
)

// Sentinel errors for this package.
var (
	// ErrInvalidType indicates that an invalid type was specified.
	ErrInvalidType = errors.New("invalid type value")

	// ErrInvalidFieldValue indicates that an invalid value was specified.
	ErrInvalidFieldValue = errors.New("invalid field value")

	// ErrMissingValue indicates that an expected value was missing.
	ErrMissingValue = errors.New("missing expected value")

	// ErrValueNotFound indicates that a requested value was not found.
	ErrValueNotFound = errors.New("requested value not found")
)

// Message represents a Microsoft Teams message containing one or more
// Adaptive Cards.
type Message struct {
	// Type is required; must be set to "message".
	Type string `json:"type"`

	// Attachments is a collection of one or more Adaptive Cards.
	//
	// NOTE: Including multiple attachment *without* AttachmentLayout set to
	// "carousel" hides cards after the first. Not sure if this is a bug, or
	// if it's intentional.
	Attachments []Attachment `json:"attachments"`

	// AttachmentLayout controls the layout for Adaptive Cards in the
	// Attachments collection.
	AttachmentLayout string `json:"attachmentLayout,omitempty"`

	// ValidateFunc is an optional user-specified validation function that is
	// responsible for validating a Message. If not specified, default
	// validation is performed.
	ValidateFunc func() error `json:"-"`

	// payload is a prepared Message in JSON format for submission or pretty
	// printing.
	payload *bytes.Buffer `json:"-"`
}

// Attachments is a collection of Adaptive Cards for a Microsoft Teams
// message.
type Attachments []Attachment

// Attachment represents an attached Adaptive Card for a Microsoft Teams
// message.
type Attachment struct {

	// ContentType is required; must be set to
	// "application/vnd.microsoft.card.adaptive".
	ContentType string `json:"contentType"`

	// ContentURL appears to be related to support for tabs. Most examples
	// have this value set to null.
	//
	// TODO: Update this description with confirmed details.
	ContentURL NullString `json:"contentUrl,omitempty"`

	// Content represents the content of an Adaptive Card.
	//
	// TODO: Should this be a pointer?
	Content TopLevelCard `json:"content"`
}

// TopLevelCard represents the outer or top-level Card for a Microsoft Teams
// Message attachment.
type TopLevelCard struct {
	Card
}

// Card represents the content of an Adaptive Card. The TopLevelCard is a
// superset of this one, asserting that the Version field is properly set.
// That type is used exclusively for Message Attachments. This type is used
// directly for the Action.ShowCard Card field.
type Card struct {

	// Type is required; must be set to "AdaptiveCard"
	Type string `json:"type"`

	// Schema represents the URI of the Adaptive Card schema.
	Schema string `json:"$schema"`

	// Version is required for top-level cards (i.e., the outer card in an
	// attachment); the schema version that the content for an Adaptive Card
	// requires.
	//
	// The TopLevelCard type is a superset of the Card type and asserts that
	// this field is properly set, whereas the validation logic for this
	// (Card) type skips that assertion.
	Version string `json:"version"`

	// FallbackText is the text shown when the client doesn't support the
	// version specified (may contain markdown).
	FallbackText string `json:"fallbackText,omitempty"`

	// Body represents the body of an Adaptive Card. The body is made up of
	// building-blocks known as elements. Elements can be composed to create
	// many types of cards. These elements are shown in the primary card
	// region.
	Body []Element `json:"body"`

	// Actions is a collection of actions to show in the card's action bar.
	// The action bar is displayed at the bottom of a Card.
	//
	// NOTE: The max display limit has been observed to be a fixed value for
	// web/desktop app and a matching value as an initial display limit for
	// mobile app with the option to expand remaining actions in a list.
	//
	// This value is recorded in this package as "TeamsActionsDisplayLimit".
	//
	// To work around this limit, create multiple ActionSets each limited to
	// the value of TeamsActionsDisplayLimit.
	Actions []Action `json:"actions,omitempty"`

	// MSTeams is a container for properties specific to Microsoft Teams
	// messages, including formatting properties and user mentions.
	//
	// NOTE: Using pointer in order to omit unused field from JSON output.
	// https://stackoverflow.com/questions/18088294/how-to-not-marshal-an-empty-struct-into-json-with-go
	// MSTeams *MSTeams `json:"msteams,omitempty"`
	//
	// TODO: Revisit this and use a pointer if remote API doesn't like
	// receiving an empty object, though brief testing doesn't show this to be
	// a problem.
	MSTeams MSTeams `json:"msteams,omitempty"`

	// MinHeight specifies the minimum height of the card.
	MinHeight string `json:"minHeight,omitempty"`

	// VerticalContentAlignment defines how the content should be aligned
	// vertically within the container. Only relevant for fixed-height cards,
	// or cards with a minHeight specified. If MinHeight field is specified,
	// this field is required.
	VerticalContentAlignment string `json:"verticalContentAlignment,omitempty"`
}

// Elements is a collection of Element values.
type Elements []Element

// Element is a "building block" for an Adaptive Card. Elements are shown
// within the primary card region (aka, "body"), columns and other container
// types. Not all fields of this Go struct type are supported by all Adaptive
// Card element types.
type Element struct {

	// Type is required and indicates the type of the element used in the body
	// of an Adaptive Card.
	// https://adaptivecards.io/explorer/AdaptiveCard.html
	Type string `json:"type"`

	// ID is a unique identifier associated with this Element.
	ID string `json:"id,omitempty"`

	// Text is required by the TextBlock and TextRun element types. Text is
	// used to display text. A subset of markdown is supported for text used
	// in TextBlock elements, but no formatting is permitted in text used in
	// TextRun elements.
	//
	// https://docs.microsoft.com/en-us/adaptive-cards/authoring-cards/text-features
	// https://adaptivecards.io/explorer/TextBlock.html
	// https://adaptivecards.io/explorer/TextRun.html
	Text string `json:"text,omitempty"`

	// URL is required for the Image element type. URL is the URL to an Image
	// in an ImageSet element type.
	//
	// https://adaptivecards.io/explorer/Image.html
	// https://adaptivecards.io/explorer/ImageSet.html
	URL string `json:"url,omitempty"`

	// Size controls the size of text within a TextBlock element.
	Size string `json:"size,omitempty"`

	// Weight controls the weight of text in TextBlock or TextRun elements.
	Weight string `json:"weight,omitempty"`

	// Color controls the color of TextBlock elements or text used in TextRun
	// elements.
	Color string `json:"color,omitempty"`

	// Spacing controls the amount of spacing between this element and the
	// preceding element.
	Spacing string `json:"spacing,omitempty"`

	// HorizontalAlignment controls the horizontal text alignment.
	HorizontalAlignment string `json:"horizontalAlignment,omitempty"`

	// The style of the element for accessibility purposes. Valid values
	// differ based on the element type. For example, a TextBlock element
	// supports the "heading" style, whereas the Column element supports the
	// "attention" style (TextBlock does not).
	Style string `json:"style,omitempty"`

	// Items is required for most Container element types. Items is a
	// collection of card elements to render inside the Container.
	Items []Element `json:"items,omitempty"`

	// Columns is a collection of Columns used to divide a region. This field
	// is used both by ColumnSet and Table element types. The specific field
	// validation applied is based on the Type field of this Element.
	Columns []Column `json:"columns,omitempty"`

	// Rows defines the rows of the table. This field is used by a Table
	// element type.
	Rows []TableRow `json:"rows,omitempty"`

	// GridStyle defines the style of the grid. This property currently only
	// controls the grid's color. This field is used by a Table element type.
	GridStyle string `json:"gridStyle,omitempty"`

	// FirstRowAsHeaders specifies whether the first row of the table should be
	// treated as a header row, and be announced as such by accessibility
	// software. This field is used by a Table element type.
	//
	// If not specified defaults to true.
	//
	// NOTE: We define this field as a pointer type so that omitting a value
	// for the pointer leaves the field out of the generated JSON payload (due
	// to 'omitempty' behavior of the JSON encoder and results in the
	// "defaults to true" behavior as defined by the schema.
	FirstRowAsHeaders *bool `json:"firstRowAsHeaders,omitempty"`

	// Visible specifies whether this element will be removed from the visual
	// tree.
	//
	// If not specified defaults to true.
	//
	// NOTE: We define this field as a pointer type so that omitting a value
	// for the pointer leaves the field out of the generated JSON payload (due
	// to 'omitempty' behavior of the JSON encoder and results in the
	// "defaults to true" behavior as defined by the schema.
	Visible *bool `json:"isVisible,omitempty"`

	// ShowGridLines specified whether grid lines should be displayed.  This
	// field is used by a Table element type.
	//
	// If not specified defaults to true.
	//
	// NOTE: We define this field as a pointer type so that omitting a value
	// for the pointer leaves the field out of the generated JSON payload (due
	// to 'omitempty' behavior of the JSON encoder and results in the
	// "defaults to true" behavior as defined by the schema.
	ShowGridLines *bool `json:"showGridLines,omitempty"`

	// Actions is required for the ActionSet element type. Actions is a
	// collection of Actions to show for an ActionSet element type.
	//
	// TODO: Should this be a pointer?
	Actions []Action `json:"actions,omitempty"`

	// SelectAction is an Action that will be invoked when the Container
	// element is tapped or selected. Action.ShowCard is not supported.
	//
	// This field is used by supported Container element types (Column,
	// ColumnSet, Container).
	//
	SelectAction *ISelectAction `json:"selectAction,omitempty"`

	// Facts is required for the FactSet element type. Actions is a collection
	// of Fact values that are part of a FactSet element type. Each Fact value
	// is a key/value pair displayed in tabular form.
	//
	// TODO: Should this be a pointer?
	Facts []Fact `json:"facts,omitempty"`

	// Wrap controls whether text is allowed to wrap or is clipped for
	// TextBlock elements.
	Wrap bool `json:"wrap,omitempty"`

	// IsSubtle specifies whether this element should appear slightly toned
	// down.
	IsSubtle bool `json:"isSubtle,omitempty"`

	// Separator, when true, indicates that a separating line shown should be
	// drawn at the top of the element.
	Separator bool `json:"separator,omitempty"`

	// CodeSnippet provides the content for a CodeBlock element, specific to MSTeams.
	CodeSnippet string `json:"codeSnippet,omitempty"`

	// Language specifies the language of a CodeBlock element, specific to MSTeams.
	Language string `json:"language,omitempty"`

	// StartLineNumber specifies the initial line number of CodeBlock element, specific to MSTeams.
	StartLineNumber int `json:"startLineNumber,omitempty"`
}

// Container is an Element type that allows grouping items together.
type Container Element

// FactSet is an Element type that groups and displays a series of facts (i.e.
// name/value pairs) in a tabular form.
type FactSet Element

// Columns is a collection of Column values for a ColumnSet or a Table.
type Columns []Column

// ColumnItems is a collection of card elements that should be rendered inside
// of the column.
type ColumnItems []*Element

// Column is a container used by a ColumnSet or Table element type. Each
// container may contain one or more elements.
//
// https://adaptivecards.io/explorer/Column.html
type Column struct {
	// Type is required; must be set to "Column" when used with ColumnSet type
	// or "TableColumnDefinition" when used as a Table column.
	Type string `json:"type,omitempty"`

	// ID is a unique identifier associated with this Column.
	ID string `json:"id,omitempty"`

	// Width represents the width of a column in the column group OR a column
	// in a table. Valid values consist of fixed strings OR a number
	// representing the relative width.
	//
	// If used in a column group, valid values are "auto", "stretch", a number
	// representing relative width of the column in the column group or a
	// string that specifies a pixel width, like "50px".
	//
	// If used in a table, valid values are a number representing relative
	// width of the column relative to the other columns in the table or a
	// string that specifies a pixel width, like "50px".
	Width interface{} `json:"width,omitempty"`

	// Items are the card elements that should be rendered inside of the
	// column.
	Items []*Element `json:"items,omitempty"`

	// SelectAction is an action that will be invoked when the Column is
	// tapped or selected. Action.ShowCard is not supported.
	SelectAction *ISelectAction `json:"selectAction,omitempty"`

	// HorizontalCellContentAlignment is a property of the Table element type.
	//
	// This field controls how the content of all cells in the column is
	// horizontally aligned by default. When specified, this value overrides
	// the setting at the table level. When not specified, horizontal
	// alignment is defined at the table, row or cell level.
	HorizontalCellContentAlignment string `json:"horizontalCellContentAlignment,omitempty"`

	// VerticalCellContentAlignment is a property of the Table element type.
	//
	// This field controls how the content of all cells in the column is
	// vertically aligned by default. When specified, this value overrides the
	// setting at the table level. When not specified, vertical alignment is
	// defined at the table, row or cell level.
	VerticalCellContentAlignment string `json:"verticalCellContentAlignment,omitempty"`
}

// Facts is a collection of Fact values.
type Facts []Fact

// Fact represents a Fact in a FactSet as a key/value pair.
type Fact struct {
	// Title is required; the title of the fact.
	Title string `json:"title"`

	// Value is required; the value of the fact.
	Value string `json:"value"`
}

// TableColumnDefinition defines the characteristics of a column in a Table
// element such as number of columns or their sizes.
//
// https://adaptivecards.io/explorer/Table.html
type TableColumnDefinition Column

// TableColumnDefinitions is a collection of TableColumnDefinition values.
//
// We use this as a "wrapper" type to convert a Columns collection so that we
// can apply specific validation requirements specific to a Table column.
type TableColumnDefinitions []Column

// TableCell represents a cell within a row of a Table element.
//
// https://adaptivecards.io/explorer/TableCell.html
type TableCell struct {
	// Type is required; must be set to "TableCell".
	Type string `json:"type"`

	// Style is a style hint for a TableCell.
	Style string `json:"style,omitempty"`

	// Bleed determines whether the element should bleed through its parent's
	// padding.
	Bleed bool `json:"bleed,omitempty"`

	// MinHeight specifies the minimum height of the container in pixels
	// (e.g., 80px).
	MinHeight string `json:"minHeight,omitempty"`

	// VerticalContentAlignment defines how the content should be aligned
	// vertically within the container.
	//
	// When not specified, the value of VerticalContentAlignment is inherited
	// from the parent container. If no parent container has
	// VerticalContentAlignment set, it defaults to Top.
	VerticalContentAlignment string `json:"verticalContentAlignment,omitempty"`

	// Items are the card elements that should be rendered inside of the
	// cell.
	Items []*Element `json:"items,omitempty"`
}

// TableCells is a collection of TableCell values.
type TableCells []TableCell

// TableRow is a row within a Table each being a collection of cells. Rows are
// not required, which allows empty Tables to be generated via templating
// without breaking the rendering of the whole card.
//
// https://adaptivecards.io/explorer/Table.html
type TableRow struct {
	// Type is required; must be set to "TableRow".
	Type string `json:"type"`

	// Style defines the style of the entire row.
	Style string `json:"style,omitempty"`

	// HorizontalCellContentAlignment is a property of the Table element type.
	//
	// This field controls how the content of all cells in the row is
	// horizontally aligned by default. When specified, this value overrides
	// both the setting at the table and columns level. When not specified,
	// horizontal alignment is defined at the table, column or cell level.
	HorizontalCellContentAlignment string `json:"horizontalCellContentAlignment,omitempty"`

	// VerticalCellContentAlignment is a property of the Table element type.
	//
	// This field controls how the content of all cells in the column is
	// vertically aligned by default. When specified, this value overrides the
	// setting at the table and column level. When not specified, vertical
	// alignment is defined either at the table, column or cell level.
	VerticalCellContentAlignment string `json:"verticalCellContentAlignment,omitempty"`

	// Cells are the cells in this row. If a row contains more cells than
	// there are columns defined on the Table element, the extra cells are
	// ignored.
	Cells []TableCell `json:"cells"`
}

// TableRows is a collection of TableRow values.
type TableRows []TableRow

// Actions is a collection of Action values.
type Actions []Action

// Action represents an action that a user may take on a card. Actions
// typically get rendered in an "action bar" at the bottom of a card.
//
//   - https://adaptivecards.io/explorer/ActionSet.html
//   - https://adaptivecards.io/explorer/AdaptiveCard.html
//   - https://docs.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-reference
//
// TODO: Extend with additional supported fields.
type Action struct {

	// Type is required; specific values are supported.
	//
	// Action.Submit is not supported for Incoming Webhooks.
	//
	// Action.Execute was added in Adaptive Card schema version 1.4. which
	// Teams MAY not fully support.
	//
	// The supported actions are Action.OpenURL, Action.ShowCard,
	// Action.ToggleVisibility, and Action.Execute (see above).
	//
	// https://docs.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-reference#support-for-adaptive-cards
	// https://docs.microsoft.com/en-us/adaptive-cards/authoring-cards/universal-action-model#schema
	Type string `json:"type"`

	// ID is a unique identifier associated with this Action.
	ID string `json:"id,omitempty"`

	// Title is a label for the button or link that represents this action.
	Title string `json:"title,omitempty"`

	// URL to open; required for the Action.OpenUrl type, optional for other
	// action types.
	URL string `json:"url,omitempty"`

	// Fallback describes what to do when an unknown element is encountered or
	// the requirements of this or any children can't be met.
	Fallback string `json:"fallback,omitempty"`

	// Card property is used by Action.ShowCard type.
	//
	// NOTE: Based on a review of JSON content, it looks like `ActionCard` is
	// really just a `Card` type.
	//
	// refs https://github.com/matthidinger/ContosoScubaBot/blob/master/Cards/SubscriberNotification.JSON
	Card *Card `json:"card,omitempty"`

	// TargetElements is the collection of TargetElement values.
	//
	// It is not recommended to include Input elements with validation due to
	// confusion that can arise from invalid inputs that are not currently
	// visible.
	//
	// https://docs.microsoft.com/en-us/adaptive-cards/authoring-cards/input-validation
	TargetElements []TargetElement `json:"targetElements,omitempty"`
}

// TargetElement represents an entry for Action.ToggleVisibility's
// targetElements property.
//
//   - https://adaptivecards.io/explorer/TargetElement.html
//   - https://adaptivecards.io/explorer/Action.ToggleVisibility.html
type TargetElement struct {
	// ElementID is the ID value of the element to toggle.
	ElementID string `json:"elementId"`

	// Visible provides display or visibility control for a target Element.
	//
	//  - If true, always show target element.
	//  - If false, always hide target element.
	//  - If not supplied, toggle target element's visibility.
	//
	// NOTE: We define this field as a pointer type so that omitting a value
	// for the pointer leaves the field out of the generated JSON payload (due
	// to 'omitempty' behavior of the JSON encoder. If leaving this field out,
	// visibility can be toggled for target Elements.
	Visible *bool `json:"isVisible,omitempty"`
}

/*

General scratch notes for https://github.com/atc0005/go-teams-notify/issues/243
===============================================================================

https://adaptivecards.io/explorer/Action.ToggleVisibility.html
https://adaptivecards.io/explorer/TargetElement.html

While the targetElements array (JSON) supports raw text strings OR
TargetElement values, we will opt to only support TargetElement values.
Otherwise, we end up needing to use more complicated logic.

Instead of trying to support this:

	"targetElements": [
		"textToToggle",
		"imageToToggle",
		"imageToToggle2"
	]

we support this instead:

	"targetElements": [
		{
			"elementId": "textToToggle"
		},
		{
			"elementId": "imageToToggle"
		},
		{
			"elementId": "imageToToggle2"
		}
	]


A Container type has a selectAction field. That slice contains TargetElement
entries.

*/

// ISelectAction represents an Action that will be invoked when a container
// type (e.g., Column, ColumnSet, Container) is tapped or selected.
// Action.ShowCard is not supported.
//
//   - https://adaptivecards.io/explorer/Container.html
//   - https://adaptivecards.io/explorer/ColumnSet.html
//   - https://adaptivecards.io/explorer/Column.html
//
// TODO: Extend with additional supported fields.
type ISelectAction struct {

	// Type is required; specific values are supported.
	//
	// The supported actions are Action.Execute, Action.OpenUrl,
	// Action.ToggleVisibility.
	//
	// See also https://docs.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-reference
	Type string `json:"type"`

	// ID is a unique identifier associated with this ISelectAction.
	ID string `json:"id,omitempty"`

	// Title is a label for the button or link that represents this action.
	Title string `json:"title,omitempty"`

	// URL is required for the Action.OpenUrl type, optional for other action
	// types.
	URL string `json:"url,omitempty"`

	// Fallback describes what to do when an unknown element is encountered or
	// the requirements of this or any children can't be met.
	Fallback string `json:"fallback,omitempty"`

	// TargetElements is the collection of TargetElement values.
	//
	// This field is specific to the Action.ToggleVisibility Action type.
	//
	// It is not recommended to include Input elements with validation due to
	// confusion that can arise from invalid inputs that are not currently
	// visible.
	//
	// https://docs.microsoft.com/en-us/adaptive-cards/authoring-cards/input-validation
	TargetElements []TargetElement `json:"targetElements,omitempty"`
}

// MSTeams represents a container for properties specific to Microsoft Teams
// messages, including formatting properties and user mentions.
type MSTeams struct {

	// Width controls the width of Adaptive Cards within a Microsoft Teams
	// messages.
	// https://docs.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-format#full-width-adaptive-card
	Width string `json:"width,omitempty"`

	// AllowExpand controls whether images can be displayed in stage view
	// selectively.
	//
	// https://docs.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-format#stage-view-for-images-in-adaptive-cards
	AllowExpand bool `json:"allowExpand,omitempty"`

	// Entities is a collection of user mentions.
	// TODO: Should this be a slice of pointers?
	Entities []Mention `json:"entities,omitempty"`
}

// Mentions is a collection of Mention values.
type Mentions []Mention

// Mention represents a mention in the message for a specific user.
type Mention struct {
	// Type is required; must be set to "mention".
	Type string `json:"type"`

	// Text must match a portion of the message text field. If it does not,
	// the mention is ignored.
	//
	// Brief testing indicates that this needs to wrap a name/value in <at>NAME
	// HERE</at> tags.
	Text string `json:"text"`

	// Mentioned represents a user that is mentioned.
	Mentioned Mentioned `json:"mentioned"`
}

// Mentioned represents the user id and name of a user that is mentioned.
type Mentioned struct {
	// ID is the unique identifier for a user that is mentioned. This value
	// can be an object ID (e.g., 5e8b0f4d-2cd4-4e17-9467-b0f6a5c0c4d0) or a
	// UserPrincipalName (e.g., NewUser@contoso.onmicrosoft.com).
	ID string `json:"id"`

	// Name is the DisplayName of the user mentioned.
	Name string `json:"name"`
}

// NewMessage creates a new Message with required fields predefined.
func NewMessage() *Message {
	return &Message{
		Type: TypeMessage,
	}
}

// NewSimpleMessage creates a new simple Message using the specified text and
// optional title. If specified, text wrapping is enabled. An error is
// returned if an empty text string is specified.
func NewSimpleMessage(text string, title string, wrap bool) (*Message, error) {
	if text == "" {
		return nil, fmt.Errorf(
			"required field text is empty: %w",
			ErrMissingValue,
		)
	}

	msg := Message{
		Type: TypeMessage,
	}

	textCard, err := NewTextBlockCard(text, title, wrap)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create TextBlock card: %w",
			err,
		)
	}

	if err := msg.Attach(textCard); err != nil {
		return nil, fmt.Errorf(
			"failed to create simple message: %w",
			err,
		)
	}

	return &msg, nil
}

// NewTextBlockCard creates a new Card using the specified text and optional
// title. If specified, the TextBlock has text wrapping enabled.
func NewTextBlockCard(text string, title string, wrap bool) (Card, error) {
	if text == "" {
		return Card{}, fmt.Errorf(
			"required field text is empty: %w",
			ErrMissingValue,
		)
	}

	textBlock := Element{
		Type: TypeElementTextBlock,
		Wrap: wrap,
		Text: text,
	}

	card := Card{
		Type:    TypeAdaptiveCard,
		Schema:  AdaptiveCardSchema,
		Version: fmt.Sprintf(AdaptiveCardVersionTmpl, AdaptiveCardMaxVersion),
		Body: []Element{
			textBlock,
		},
	}

	if title != "" {
		titleTextBlock := NewTitleTextBlock(title, wrap)
		card.Body = append([]Element{titleTextBlock}, card.Body...)
	}

	return card, nil
}

// NewCard creates and returns an empty Card.
func NewCard() Card {
	return Card{
		Type:    TypeAdaptiveCard,
		Schema:  AdaptiveCardSchema,
		Version: fmt.Sprintf(AdaptiveCardVersionTmpl, AdaptiveCardMaxVersion),
	}
}

// Attach receives and adds one or more Card values to the Attachments
// collection for a Microsoft Teams message.
//
// NOTE: Including multiple cards in the attachments collection *without*
// attachmentLayout set to "carousel" hides cards after the first. Not sure if
// this is a bug, or if it's intentional.
func (m *Message) Attach(cards ...Card) error {
	if len(cards) == 0 {
		return fmt.Errorf(
			"received empty collection of cards: %w",
			ErrMissingValue,
		)
	}

	for _, card := range cards {
		attachment := Attachment{
			ContentType: AttachmentContentType,

			// Explicitly convert Card to TopLevelCard in order to assert that
			// TopLevelCard specific requirements are checked during
			// validation.
			Content: TopLevelCard{card},
		}

		m.Attachments = append(m.Attachments, attachment)
	}

	return nil
}

// Carousel sets the Message Attachment layout to Carousel display mode.
func (m *Message) Carousel() *Message {
	m.AttachmentLayout = AttachmentLayoutCarousel
	return m
}

// PrettyPrint returns a formatted JSON payload of the Message if the
// Prepare() method has been called, or an empty string otherwise.
func (m *Message) PrettyPrint() string {
	if m.payload != nil {
		var prettyJSON bytes.Buffer
		_ = json.Indent(&prettyJSON, m.payload.Bytes(), "", "\t")

		return prettyJSON.String()
	}

	return ""
}

// Prepare handles tasks needed to construct a payload from a Message for
// delivery to an endpoint.
func (m *Message) Prepare() error {
	jsonMessage, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf(
			"error marshalling Message to JSON: %w",
			err,
		)
	}

	switch {
	case m.payload == nil:
		m.payload = &bytes.Buffer{}
	default:
		m.payload.Reset()
	}

	_, err = m.payload.Write(jsonMessage)
	if err != nil {
		return fmt.Errorf(
			"error updating JSON payload for Message: %w",
			err,
		)
	}

	return nil
}

// Payload returns the prepared Message payload. The caller should call
// Prepare() prior to calling this method, results are undefined otherwise.
func (m *Message) Payload() io.Reader {
	return m.payload
}

// Validate performs validation for Message using ValidateFunc if defined,
// otherwise applying default validation.
func (m Message) Validate() error {
	if m.ValidateFunc != nil {
		return m.ValidateFunc()
	}

	v := validator.Validator{}

	v.FieldHasSpecificValue(
		m.Type,
		"type",
		TypeMessage,
		"message",
		ErrInvalidType,
	)

	// We need an attachment (containing one or more Adaptive Cards) in order
	// to generate a valid Message for Microsoft Teams delivery.
	v.NotEmptyCollection("Attachments", m.Type, ErrMissingValue, m.Attachments)

	v.SelfValidate(Attachments(m.Attachments))

	// Optional field, but only specific values permitted if set.
	v.InListIfFieldValNotEmpty(
		m.AttachmentLayout,
		"AttachmentLayout",
		"message",
		supportedAttachmentLayoutValues(),
		ErrInvalidFieldValue,
	)

	return v.Err()
}

// Validate asserts that fields have valid values.
func (a Attachment) Validate() error {
	v := validator.Validator{}

	v.FieldHasSpecificValue(
		a.ContentType,
		"attachment type",
		AttachmentContentType,
		"attachment",
		ErrInvalidType,
	)

	v.SelfValidate(a.Content)

	return v.Err()
}

// Validate asserts that the collection of Attachment values are all valid.
func (a Attachments) Validate() error {
	for _, attachment := range a {
		if err := attachment.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate asserts that fields have valid values.
func (c Card) Validate() error {
	v := validator.Validator{}

	// TODO: Version field validation
	//
	// The Version field is required for top-level cards, optional for Cards
	// nested within an Action.ShowCard. Because we don't have a reliable way
	// to assert that relationship, we skip applying validation for that value
	// for now.

	v.FieldHasSpecificValue(
		c.Type,
		"type",
		TypeAdaptiveCard,
		"card",
		ErrInvalidType,
	)

	// While the schema value should be set it is not strictly required. If it
	// is set, we assert that it is the correct value.
	v.FieldHasSpecificValueIfFieldNotEmpty(
		c.Schema,
		"Schema",
		AdaptiveCardSchema,
		"card",
		ErrInvalidFieldValue,
	)

	// Both are optional fields, unless MinHeight is set in which case
	// VerticalContentAlignment is required.
	v.SuccessfulFuncCall(
		func() error {
			return assertHeightAlignmentFieldsSetWhenRequired(
				c.MinHeight, c.VerticalContentAlignment,
			)
		},
	)

	v.SuccessfulFuncCall(
		func() error {
			return assertCardBodyHasMention(c.Body, c.MSTeams.Entities)
		},
	)

	v.SelfValidate(Elements(c.Body))
	v.SelfValidate(Actions(c.Actions))

	return v.Err()
}

// Validate asserts that fields have valid values.
func (tc TopLevelCard) Validate() error {
	v := validator.Validator{}

	// Validate embedded Card first as those validation requirements apply
	// here also.
	v.SelfValidate(tc.Card)

	// The Version field is required for top-level cards (this one), optional
	// for Cards nested within an Action.ShowCard.
	v.SuccessfulFuncCall(
		func() error { return assertValidVersionFieldValue(tc.Version) },
	)

	return v.Err()
}

// Validate asserts that the collection of Element values are all valid.
func (e Elements) Validate() error {
	for _, element := range e {
		if err := element.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate asserts that fields have valid values.
func (e Element) Validate() error {
	v := validator.Validator{}

	supportedElementTypes := supportedElementTypes()
	supportedSizeValues := supportedSizeValues()
	supportedWeightValues := supportedWeightValues()
	supportedColorValues := supportedColorValues()
	supportedSpacingValues := supportedSpacingValues()
	supportedHorizontalAlignmentValues := supportedHorizontalAlignmentValues()

	// Valid Style field values differ based on type. For example, a Container
	// element supports Container styles whereas a TextBlock supports a
	// different and more limited set of style values. We use a helper
	// function to retrieve valid style values for evaluation.
	supportedStyleValues := supportedStyleValues(e.Type)

	/******************************************************************
		General requirements for all Element types.
	******************************************************************/

	v.InListIfFieldValNotEmpty(e.Type, "Type", "element", supportedElementTypes, ErrInvalidType)
	v.InListIfFieldValNotEmpty(e.Size, "Size", "element", supportedSizeValues, ErrInvalidFieldValue)
	v.InListIfFieldValNotEmpty(e.Weight, "Weight", "element", supportedWeightValues, ErrInvalidFieldValue)
	v.InListIfFieldValNotEmpty(e.Color, "Color", "element", supportedColorValues, ErrInvalidFieldValue)
	v.InListIfFieldValNotEmpty(e.Spacing, "Spacing", "element", supportedSpacingValues, ErrInvalidFieldValue)
	v.InListIfFieldValNotEmpty(e.HorizontalAlignment, "HorizontalAlignment", "element", supportedHorizontalAlignmentValues, ErrInvalidFieldValue)
	v.InListIfFieldValNotEmpty(e.Style, "Style", "element", supportedStyleValues, ErrInvalidFieldValue)

	/******************************************************************
		Requirements for specific Element types.
	******************************************************************/

	switch {
	// The Text field is required by TextBlock and TextRun elements, but an
	// empty string appears to be permitted. Because of this, we avoid
	// asserting that a value is present for the field.
	// case e.Type == TypeElementTextBlock:
	// case e.Type == TypeElementTextRun:

	// Columns collection is used by the ColumnSet type. While not required,
	// the collection should be checked.
	case e.Type == TypeElementColumnSet:
		v.SelfValidate(Columns(e.Columns))

		if e.SelectAction != nil {
			v.SelfValidate(e.SelectAction)
		}

	// Actions collection is required for ActionSet element type.
	// https://adaptivecards.io/explorer/ActionSet.html
	case e.Type == TypeElementActionSet:
		v.NotEmptyCollection("Actions", e.Type, ErrMissingValue, e.Actions)
		v.SelfValidate(Actions(e.Actions))

	// Items collection is required for Container element type.
	// https://adaptivecards.io/explorer/Container.html
	case e.Type == TypeElementContainer:
		v.NotEmptyCollection("Items", e.Type, ErrMissingValue, e.Items)
		v.SelfValidate(Elements(e.Items))

		if e.SelectAction != nil {
			v.SelfValidate(e.SelectAction)
		}

	// URL is required for Image element type.
	// https://adaptivecards.io/explorer/Image.html
	case e.Type == TypeElementImage:
		v.NotEmptyValue(e.URL, "URL", e.Type, ErrMissingValue)

	// Facts collection is required for FactSet element type.
	// https://adaptivecards.io/explorer/FactSet.html
	case e.Type == TypeElementFactSet:
		v.NotEmptyCollection("Facts", e.Type, ErrMissingValue, e.Facts)
		v.SelfValidate(Facts(e.Facts))

	case e.Type == TypeElementTable:
		v.InListIfFieldValNotEmpty(
			e.GridStyle,
			"GridStyle",
			e.Type,
			supportedContainerStyleValues(),
			ErrInvalidFieldValue,
		)

		v.SelfValidate(TableRows(e.Rows))

		v.SelfValidate(TableColumnDefinitions(e.Columns))

	case e.Type == TypeElementMSTeamsCodeBlock:
		v.NotEmptyValue(e.CodeSnippet, "CodeSnippet", e.Type, ErrMissingValue)
		v.NotEmptyValue(e.Language, "Language", e.Type, ErrMissingValue)
	}

	// Return the last recorded validation error, or nil if no validation
	// errors occurred.
	return v.Err()
}

// Validate asserts that the collection of Column values are all valid.
func (c Columns) Validate() error {
	for _, column := range c {
		if err := column.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate asserts that the Items collection field for a column contains
// valid values. Special handling is applied since the collection could
// contain nil values.
func (ci ColumnItems) Validate() error {
	for _, item := range ci {
		if item == nil {
			return fmt.Errorf(
				"card element in Column is nil: %w",
				ErrMissingValue,
			)
		}

		if err := item.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate asserts that the collection of TableColumnDefinition values are
// all valid.
func (tcds TableColumnDefinitions) Validate() error {
	for _, c := range tcds {
		// We convert the Column type to a TableColumnDefinition so that
		// fields specific to that "subtype" have separate validation logic
		// applied vs the Column type used by the ColumnSet container type.
		if err := TableColumnDefinition(c).Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate asserts that fields have valid values.
func (tcd TableColumnDefinition) Validate() error {
	v := validator.Validator{}

	// The schema shows that this is supposed to be set to
	// "TableColumnDefinition", though the example payload I reviewed did not
	// set the Type field. Because of this, we should support either not
	// setting the field at all OR requiring this specific type.
	v.FieldHasSpecificValueIfFieldNotEmpty(
		tcd.Type,
		"type",
		TypeTableColumnDefinition,
		"column",
		ErrInvalidType,
	)

	v.SuccessfulFuncCall(
		func() error { return assertTableColumnDefinitionWidthValidValues(tcd) },
	)

	v.InListIfFieldValNotEmpty(
		tcd.VerticalCellContentAlignment,
		"VerticalCellContentAlignment",
		TypeTableColumnDefinition,
		supportedVerticalContentAlignmentValues(),
		ErrInvalidFieldValue,
	)

	v.InListIfFieldValNotEmpty(
		tcd.HorizontalCellContentAlignment,
		"HorizontalCellContentAlignment",
		TypeTableColumnDefinition,
		supportedHorizontalAlignmentValues(),
		ErrInvalidFieldValue,
	)

	return v.Err()
}

// Validate asserts that the collection of TableRow values are all valid.
func (trs TableRows) Validate() error {
	for _, row := range trs {
		if err := row.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// AddCell adds one or many TableCell values to a TableRow. An error is
// returned if any TableCell value fails validation.
func (tr *TableRow) AddCell(cells ...TableCell) error {
	if len(cells) == 0 {
		return fmt.Errorf("no data provided: %w", ErrMissingValue)
	}

	for _, cell := range cells {
		if err := cell.Validate(); err != nil {
			return err
		}
	}

	tr.Cells = append(tr.Cells, cells...)

	return nil
}

// // TableCells returns a collection of underlying TableCell pointers or an
// // empty collection if no TableCell values are available.
// func (trs *TableRows) TableCells() []*TableCell {
// 	if trs == nil {
// 		return []*TableCell{}
// 	}
//
// 	var numCells int
// 	for _, row := range *trs {
// 		for range row.Cells {
// 			numCells++
// 		}
// 	}
// 	cells := make([]*TableCell, numCells)
// 	for _, row := range *trs {
// 		for i := range row.Cells {
// 			cells = append(cells, &row.Cells[i])
// 		}
// 	}
//
// 	return cells
// }

// Validate asserts that fields have valid values.
func (tr TableRow) Validate() error {
	v := validator.Validator{}

	v.FieldHasSpecificValueIfFieldNotEmpty(
		tr.Type,
		"type",
		TypeTableRow,
		"table row",
		ErrInvalidType,
	)

	v.InListIfFieldValNotEmpty(
		tr.Style,
		"Style",
		TypeTableRow,
		supportedContainerStyleValues(),
		ErrInvalidFieldValue,
	)

	v.InListIfFieldValNotEmpty(
		tr.VerticalCellContentAlignment,
		"VerticalCellContentAlignment",
		TypeTableRow,
		supportedVerticalContentAlignmentValues(),
		ErrInvalidFieldValue,
	)

	v.InListIfFieldValNotEmpty(
		tr.HorizontalCellContentAlignment,
		"HorizontalCellContentAlignment",
		TypeTableRow,
		supportedHorizontalAlignmentValues(),
		ErrInvalidFieldValue,
	)

	// Validate collection by using "wrapper" type.
	v.SelfValidate(TableCells(tr.Cells))

	return v.Err()
}

// Validate asserts that the collection of TableCell values are all valid.
func (tcs TableCells) Validate() error {
	for _, cell := range tcs {
		if err := cell.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// AddElement adds one or many Element value pointers to a TableCell. An error
// is returned if any Element value fails validation.
func (tr *TableCell) AddElement(elements ...*Element) error {
	if len(elements) == 0 {
		return fmt.Errorf("no data provided: %w", ErrMissingValue)
	}

	for _, cell := range elements {
		if cell == nil {
			return fmt.Errorf("no data provided: %w", ErrMissingValue)
		}

		if err := cell.Validate(); err != nil {
			return err
		}
	}

	tr.Items = append(tr.Items, elements...)

	return nil
}

// Validate asserts that fields have valid values.
func (tr TableCell) Validate() error {
	v := validator.Validator{}

	v.FieldHasSpecificValueIfFieldNotEmpty(
		tr.Type,
		"type",
		TypeTableCell,
		"table cell",
		ErrInvalidType,
	)

	v.InListIfFieldValNotEmpty(
		tr.Style,
		"Style",
		TypeTableCell,
		supportedContainerStyleValues(),
		ErrInvalidFieldValue,
	)

	v.SuccessfulFuncCall(
		func() error {
			return assertValidPixelSizeOrEmptyValue(tr.MinHeight)
		},
	)

	v.InListIfFieldValNotEmpty(
		tr.VerticalContentAlignment,
		"VerticalContentAlignment",
		TypeTableCell,
		supportedVerticalContentAlignmentValues(),
		ErrInvalidFieldValue,
	)

	v.NotEmptyCollection(
		"TableCellItems",
		TypeTableCell,
		ErrMissingValue,
		tr.Items,
	)

	v.NoNilValuesInCollection(
		"TableCellItems",
		TypeTableCell,
		ErrMissingValue,
		tr.Items,
	)

	for _, item := range tr.Items {
		v.SelfValidate(item)
	}

	return v.Err()
}

// AddSelectAction adds a given Action or ISelectAction value to the
// associated Column. This action will be invoked when the Column is
// tapped or selected.
//
// An error is returned if the given Action or ISelectAction value fails
// validation or if a value other than an Action or ISelectAction is provided.
func (c *Column) AddSelectAction(action interface{}) error {
	switch v := action.(type) {
	case Action:
		// Perform manual conversion to the supported type.
		selectAction := ISelectAction{
			Type:     v.Type,
			ID:       v.ID,
			Title:    v.Title,
			URL:      v.URL,
			Fallback: v.Fallback,
		}

		// Don't touch the new TargetElements field unless the provided Action
		// has specified values.
		if len(v.TargetElements) > 0 {
			selectAction.TargetElements = append(
				selectAction.TargetElements,
				v.TargetElements...,
			)
		}

		c.SelectAction = &selectAction

	case ISelectAction:
		c.SelectAction = &v

	// unsupported value provided
	default:
		return fmt.Errorf(
			"error: unsupported value provided; "+
				" only Action or ISelectAction values are supported: %w",
			ErrInvalidFieldValue,
		)
	}

	return nil
}

// Validate asserts that fields have valid values.
func (c Column) Validate() error {
	v := validator.Validator{}

	v.FieldHasSpecificValue(
		c.Type,
		"type",
		TypeColumn,
		"column",
		ErrInvalidType,
	)

	v.SuccessfulFuncCall(
		func() error { return assertColumnWidthValidValues(c) },
	)

	// Assert that the collection does not contain nil items.
	v.NoNilValuesInCollection("Items", c.Type, ErrMissingValue, c.Items)

	// Convert []*Element to ColumnItems so that we can use its Validate()
	// method to handle cases where nil values could be present in the
	// collection.
	v.SelfValidate(ColumnItems(c.Items))

	if c.SelectAction != nil {
		v.SelfValidate(c.SelectAction)
	}

	return v.Err()
}

// Validate asserts that the collection of Fact values are all valid.
func (f Facts) Validate() error {
	for _, fact := range f {
		if err := fact.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate asserts that fields have valid values.
func (f Fact) Validate() error {
	v := validator.Validator{}

	v.NotEmptyValue(f.Title, "Title", "Fact", ErrMissingValue)
	v.NotEmptyValue(f.Value, "Value", "Fact", ErrMissingValue)

	return v.Err()
}

// Validate asserts that fields have valid values.
func (m MSTeams) Validate() error {
	v := validator.Validator{}

	// If an optional width value is set, assert that it is a valid value.
	v.InListIfFieldValNotEmpty(
		m.Width,
		"Width",
		"MSTeams",
		supportedMSTeamsWidthValues(),
		ErrInvalidFieldValue,
	)

	v.SelfValidate(Mentions(m.Entities))

	return v.Err()
}

// Validate asserts that fields have valid values.
func (i ISelectAction) Validate() error {
	supportedISelectActionValues := supportedISelectActionValues(AdaptiveCardMaxVersion)
	fallbackValues := supportedActionFallbackValues(AdaptiveCardMaxVersion)

	v := validator.Validator{}

	// Some supportedISelectActionValues are restricted to later Adaptive Card
	// schema versions.
	v.InList(
		i.Type,
		"Type",
		"ISelectAction",
		supportedISelectActionValues,
		ErrInvalidType,
	)

	v.InListIfFieldValNotEmpty(
		i.Fallback,
		"Fallback",
		"ISelectAction",
		supportedISelectActionFallbackValues(AdaptiveCardMaxVersion),
		ErrInvalidFieldValue,
	)

	// See also: Action.Validate() logic.
	switch {
	case i.Type == TypeActionOpenURL:
		v.NotEmptyValue(i.URL, "URL", i.Type, ErrMissingValue)

	case i.Fallback != "":
		v.InList(i.Fallback, "Fallback", "action", fallbackValues, ErrInvalidFieldValue)

	case i.Type == TypeActionToggleVisibility:
		v.NotEmptyCollection("TargetElements", i.Type, ErrMissingValue, i.TargetElements)
	}

	return v.Err()
}

// Validate asserts that the collection of Action values are all valid.
func (a Actions) Validate() error {
	for _, action := range a {
		if err := action.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// AddTargetElement records the IDs from the given Elements in new
// TargetElement values. The specified visibility setting is used for the new
// TargetElement values.
//
//   - If true, always show target Element.
//   - If false, always hide target Element.
//   - If nil, allow toggling target Element's visibility.
//
// If the given visibility setting is nil, then the visibility setting for the
// TargetElement values is omitted. This enables toggling visibility for the
// target Elements (e.g., toggle button behavior).
func (a *Action) AddTargetElement(visible *bool, elements ...Element) error {
	elementIDs := make([]string, 0, len(elements))
	for _, e := range elements {
		if strings.TrimSpace(e.ID) == "" {
			return fmt.Errorf(
				"given Element has empty ID value: %w",
				ErrInvalidFieldValue,
			)
		}

		elementIDs = append(elementIDs, e.ID)
	}

	return a.AddTargetElementID(visible, elementIDs...)
}

// AddVisibleTargetElement records the Element IDs from the given Elements in
// new TargetElement values. All new TargetElement values are explicitly set
// as visible.
func (a *Action) AddVisibleTargetElement(elements ...Element) error {
	visible := true

	return a.AddTargetElement(&visible, elements...)
}

// AddHiddenTargetElement records the Element IDs from the given Elements in
// new TargetElement values. All new TargetElement values are explicitly set
// as not visible.
func (a *Action) AddHiddenTargetElement(elements ...Element) error {
	visible := false

	return a.AddTargetElement(&visible, elements...)
}

// AddTargetElementID records the given Element ID values in the TargetElements
// collection. A non-empty ID value is required, but the Adaptive Card "tree"
// is not searched for a valid match; it is up to the caller to ensure that
// the given ID value is valid.
//
// The specified visibility setting is used for the new TargetElement values.
//
//   - If true, always show target Element.
//   - If false, always hide target Element.
//   - If nil, allow toggling target Element's visibility.
//
// If the given visibility setting is nil, then the visibility setting for the
// TargetElement values is omitted. This enables toggling visibility for the
// target Elements (e.g., toggle button behavior).
func (a *Action) AddTargetElementID(visible *bool, elementIDs ...string) error {
	for _, id := range elementIDs {
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf(
				"received empty Element ID value: %w",
				ErrMissingValue,
			)
		}

		existingElementIDs := func() []string {
			ids := make([]string, 0, len(a.TargetElements))
			for _, targetElement := range a.TargetElements {
				ids = append(ids, targetElement.ElementID)
			}

			return ids
		}()

		// Assert that the ID is not already in the collection.
		if goteamsnotify.InList(id, existingElementIDs, false) {
			return fmt.Errorf(
				"received duplicate Element ID value %q: %w",
				id,
				ErrInvalidFieldValue,
			)
		}

		a.TargetElements = append(
			a.TargetElements,
			TargetElement{
				ElementID: id,
				Visible:   visible,
			},
		)
	}

	return nil
}

// Validate asserts that fields have valid values.
func (a Action) Validate() error {
	actionValues := supportedActionValues(AdaptiveCardMaxVersion)
	fallbackValues := supportedActionFallbackValues(AdaptiveCardMaxVersion)

	v := validator.Validator{}

	// Some Actions are restricted to later Adaptive Card schema versions.
	v.InList(a.Type, "Type", "action", actionValues, ErrInvalidType)

	switch {
	case a.Type == TypeActionOpenURL:
		v.NotEmptyValue(a.URL, "URL", a.Type, ErrMissingValue)

	case a.Fallback != "":
		v.InList(a.Fallback, "Fallback", "action", fallbackValues, ErrInvalidFieldValue)

	case a.Type == TypeActionToggleVisibility:
		v.NotEmptyCollection("TargetElements", a.Type, ErrMissingValue, a.TargetElements)

	// Optional, but only supported by the Action.ShowCard type.
	case a.Card != nil:
		v.FieldHasSpecificValue(a.Type, "type", TypeActionShowCard, "type", ErrInvalidType)
	}

	// Return the last recorded validation error, or nil if no validation
	// errors occurred.
	return v.Err()
}

// Validate asserts that the collection of Mention values are all valid.
func (m Mentions) Validate() error {
	for _, mention := range m {
		if err := mention.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate asserts that fields have valid values.
//
// Element.Validate() asserts that required Mention.Text content is found for
// each recorded user mention the Card..
func (m Mention) Validate() error {
	if m.Type != TypeMention {
		return fmt.Errorf(
			"invalid Mention type %q; expected %q: %w",
			m.Type,
			TypeMention,
			ErrInvalidType,
		)
	}

	if m.Text == "" {
		return fmt.Errorf(
			"required field Text is empty for Mention: %w",
			ErrMissingValue,
		)
	}

	return nil
}

// Validate asserts that fields have valid values.
func (m Mentioned) Validate() error {
	if m.ID == "" {
		return fmt.Errorf(
			"required field ID is empty: %w",
			ErrMissingValue,
		)
	}

	if m.Name == "" {
		return fmt.Errorf(
			"required field Name is empty: %w",
			ErrMissingValue,
		)
	}

	return nil
}

// Mention uses the provided display name, ID and text values to add a new
// user Mention and TextBlock element to the first Card in the Message.
//
// If no Cards are yet attached to the Message, a new card is created using
// the Mention and TextBlock element. If specified, the new TextBlock element
// is added as the first element of the Card, otherwise it is added last. An
// error is returned if insufficient values are provided.
func (m *Message) Mention(prependElement bool, displayName string, id string, msgText string) error {
	// NOTE: Rely on called functions to validate given arguments.

	switch {
	// If no existing cards, add a new one.
	case len(m.Attachments) == 0:
		mentionCard, err := NewMentionCard(displayName, id, msgText)
		if err != nil {
			return err
		}

		if err := m.Attach(mentionCard); err != nil {
			return err
		}

	// We have at least one Card already, use it.
	default:

		// Build mention.
		mention, err := NewMention(displayName, id)
		if err != nil {
			return fmt.Errorf(
				"add new Mention to Message: %w",
				err,
			)
		}

		textBlock := Element{
			Type: TypeElementTextBlock,

			// TODO: Any issues caused by enabling wrapping? The goal is to
			// prevent the Mention.Text content from pushing user specified
			// text off of the Card, out of sight.
			Wrap: true,

			// The text block contains the mention text string (required) and
			// user-specified message text string. Use the mention text as a
			// "greeting" or lead-in for the user-specified message text.
			Text: mention.Text + " " + msgText,
		}

		switch {
		case prependElement:
			m.Attachments[0].Content.Body = append(
				[]Element{textBlock},
				m.Attachments[0].Content.Body...,
			)
		default:
			m.Attachments[0].Content.Body = append(
				m.Attachments[0].Content.Body,
				textBlock,
			)
		}

		m.Attachments[0].Content.MSTeams.Entities = append(
			m.Attachments[0].Content.MSTeams.Entities,
			mention,
		)
	}

	return nil
}

// Mention uses the given display name, ID and message text to add a new user
// Mention and TextBlock element to the Card. If specified, the new TextBlock
// element is added as the first element of the Card, otherwise it is added
// last. An error is returned if provided values are insufficient to create
// the user mention.
func (c *Card) Mention(displayName string, id string, msgText string, prependElement bool) error {
	if msgText == "" {
		return fmt.Errorf(
			"required msgText argument is empty: %w",
			ErrMissingValue,
		)
	}

	// Rely on this called function to validate the other arguments.
	mention, err := NewMention(displayName, id)
	if err != nil {
		return err
	}

	textBlock := Element{
		Type: TypeElementTextBlock,

		// TODO: Any issues caused by enabling wrapping? The goal is to
		// prevent the Mention.Text content from pushing user specified text
		// off of the Card, out of sight.
		Wrap: true,
		Text: mention.Text + " " + msgText,
	}

	switch {
	case prependElement:
		c.Body = append(c.Body, textBlock)
	default:
		c.Body = append([]Element{textBlock}, c.Body...)
	}

	return nil
}

// AddMention adds one or more provided user mentions to the associated Card
// along with a new TextBlock element. The Text field for the new TextBlock
// element is updated with the Mention Text.
//
// If specified, the new TextBlock element is inserted as the first element in
// the Card body. This effectively creates a dedicated TextBlock that acts as
// a "lead-in" or "announcement block" for other elements in the Card. If
// false, the newly created TextBlock is appended to the Card, effectively
// creating a "CC" list commonly found at the end of an email message.
//
// An error is returned if specified Mention values fail validation.
func (c *Card) AddMention(prepend bool, mentions ...Mention) error {
	textBlock := Element{
		Type: TypeElementTextBlock,

		// The goal is to prevent the Mention.Text from extending off of the
		// Card, out of sight.
		Wrap: true,
	}

	// Whether the mention text is prepended or appended doesn't matter since
	// the TextBlock element we are adding is empty. Likewise, the separator
	// chosen doesn't really matter either as there isn't any existing text
	// that we need to separate from the mention text.
	//
	// NOTE: WE rely on this function to apply validation of user mention
	// values instead of duplicating that logic here.
	err := AddMention(c, &textBlock, true, defaultMentionTextSeparator, mentions...)
	if err != nil {
		return err
	}

	switch prepend {
	case true:
		c.Body = append([]Element{textBlock}, c.Body...)
	case false:
		c.Body = append(c.Body, textBlock)
	}

	return nil
}

// AddElement adds one or more provided Elements to the Body of the associated
// Card. If specified, the Element values are prepended to the Card Body (as a
// contiguous set retaining current order), otherwise appended to the Card
// Body.
//
// An error is returned if specified Element values fail validation.
func (c *Card) AddElement(prepend bool, elements ...Element) error {
	if len(elements) == 0 {
		return fmt.Errorf(
			"received empty collection of elements: %w",
			ErrMissingValue,
		)
	}

	// Validate first before adding to Card Body.
	for _, element := range elements {
		if err := element.Validate(); err != nil {
			return err
		}
	}

	switch prepend {
	case true:
		c.Body = append(elements, c.Body...)
	case false:
		c.Body = append(c.Body, elements...)
	}

	return nil
}

// AddAction adds one or more provided Actions to the associated Card. If
// specified, the Action values are prepended to the Card (as a collection
// retaining current order), otherwise appended.
//
// NOTE: The max display limit for a Card's actions array has been observed to
// be a fixed value for web/desktop app and a matching value as an initial
// display limit for mobile app with the option to expand remaining actions in
// a list.
//
// This value is recorded in this package as "TeamsActionsDisplayLimit".
//
// Consider adding Action values to one or more ActionSet elements as needed
// and include within the Card.Body directly or within a Container to
// workaround this limit.
//
// An error is returned if specified Action values fail validation.
func (c *Card) AddAction(prepend bool, actions ...Action) error {
	if len(actions) == 0 {
		return fmt.Errorf(
			"received empty collection of actions: %w",
			ErrMissingValue,
		)
	}

	for _, action := range actions {
		if err := action.Validate(); err != nil {
			return err
		}
	}

	switch prepend {
	case true:
		c.Actions = append(actions, c.Actions...)
	case false:
		c.Actions = append(c.Actions, actions...)
	}

	return nil
}

// GetElement searches all Element values attached to the Card for the
// specified ID (case sensitive). If found, a pointer to the Element is
// returned, otherwise an error is returned.
func (c *Card) GetElement(id string) (*Element, error) {
	if id == "" {
		return nil, fmt.Errorf(
			"empty ID value specified: %w",
			ErrMissingValue,
		)
	}

	for _, element := range c.Body {
		if element.ID == id {
			return &element, nil
		}

		// If the Element is a Container, we need to evaluate its collection
		// of Elements.
		for _, item := range element.Items {
			if item.ID == id {
				return &element, nil
			}
		}
	}

	return nil, fmt.Errorf(
		"unable to retrieve element id: %w",
		ErrValueNotFound,
	)
}

// AddFactSet adds one or more provided FactSet elements to the Body of the
// associated Card. If specified, the FactSet values are prepended to the Card
// Body (as a contiguous set retaining current order), otherwise appended to
// the Card Body.
//
// An error is returned if specified FactSet values fail validation.
//
// TODO: Is this needed? Should we even have a separate FactSet type that is
// so difficult to work with?
func (c *Card) AddFactSet(prepend bool, factsets ...FactSet) error {
	if len(factsets) == 0 {
		return fmt.Errorf(
			"received empty collection of factsets: %w",
			ErrMissingValue,
		)
	}

	// Convert to base Element type
	factsetElements := make([]Element, 0, len(factsets))
	for _, factset := range factsets {
		element := Element(factset)
		factsetElements = append(factsetElements, element)
	}

	// Validate first before adding to Card Body.
	for _, element := range factsetElements {
		if err := element.Validate(); err != nil {
			return err
		}
	}

	switch prepend {
	case true:
		c.Body = append(factsetElements, c.Body...)
	case false:
		c.Body = append(c.Body, factsetElements...)
	}

	return nil
}

// SetFullWidth enables full width display for the Card.
func (c *Card) SetFullWidth() {
	c.MSTeams.Width = MSTeamsWidthFull
}

// NewMention uses the given display name and ID to create a user Mention
// value for inclusion in a Card. An error is returned if provided values are
// insufficient to create the user mention.
func NewMention(displayName string, id string) (Mention, error) {
	switch {
	case displayName == "":
		return Mention{}, fmt.Errorf(
			"required name argument is empty: %w",
			ErrMissingValue,
		)

	case id == "":
		return Mention{}, fmt.Errorf(
			"required id argument is empty: %w",
			ErrMissingValue,
		)

	default:

		// Build mention.
		mention := Mention{
			Type: TypeMention,
			Text: fmt.Sprintf(MentionTextFormatTemplate, displayName),
			Mentioned: Mentioned{
				ID:   id,
				Name: displayName,
			},
		}

		return mention, nil
	}
}

// AddMention adds one or more provided user mentions to the specified Card.
// The Text field for the specified TextBlock element is updated with the
// Mention Text. If specified, the Mention Text is prepended, otherwise
// appended. If specified, a custom separator is used between the Mention Text
// and the TextBlock Text field, otherwise the default separator is used.
//
// NOTE: This function "registers" the specified Mention values with the Card
// and updates the specified textBlock element, however the caller is
// responsible for ensuring that the specified textBlock element is added to
// the Card.
//
// An error is returned if specified Mention values fail validation, or one of
// Card or Element pointers are null.
func AddMention(card *Card, textBlock *Element, prependText bool, separator string, mentions ...Mention) error {
	if card == nil {
		return fmt.Errorf(
			"specified pointer to Card is nil: %w",
			ErrMissingValue,
		)
	}

	if textBlock == nil {
		return fmt.Errorf(
			"specified pointer to TextBlock element is nil: %w",
			ErrMissingValue,
		)
	}

	if textBlock.Type != TypeElementTextBlock {
		return fmt.Errorf(
			"invalid element type %q; expected %q: %w",
			textBlock.Type,
			TypeElementTextBlock,
			ErrInvalidType,
		)
	}

	if len(mentions) == 0 {
		return fmt.Errorf(
			"received empty collection of mentions: %w",
			ErrMissingValue,
		)
	}

	// Validate all user mentions before modifying Card or Element.
	for _, mention := range mentions {
		if err := mention.Validate(); err != nil {
			return err
		}
	}

	if separator == "" {
		separator = defaultMentionTextSeparator
	}

	mentionsText := make([]string, 0, len(mentions))

	// Record user mentions in the Card and collect all required user mention
	// text values.
	for _, mention := range mentions {
		mentionsText = append(mentionsText, mention.Text)
		card.MSTeams.Entities = append(card.MSTeams.Entities, mention)
	}

	// Update TextBlock element text with required user mention text string.
	switch prependText {
	case true:
		textBlock.Text = strings.Join(mentionsText, " ") + separator + textBlock.Text
	case false:
		textBlock.Text = textBlock.Text + separator + strings.Join(mentionsText, " ")
	}

	// The original text may have been sufficiently short to not be truncated,
	// but once we add the user mention text it is more likely that truncation
	// could occur. Indicate that the text should be wrapped to avoid this.
	textBlock.Wrap = true

	return nil
}

// NewMentionMessage creates a new simple Message. Using the given message
// text, displayName and ID, a user Mention is also created and added to the
// new Message. An error is returned if provided values are insufficient to
// create the user mention.
func NewMentionMessage(displayName string, id string, msgText string) (*Message, error) {
	msg := Message{
		Type: TypeMessage,
	}

	// Rely on function to apply validation instead of duplicating it here.
	mentionCard, err := NewMentionCard(displayName, id, msgText)
	if err != nil {
		return nil, err
	}

	if err := msg.Attach(mentionCard); err != nil {
		return nil, err
	}

	return &msg, nil
}

// NewMentionCard creates a new Card with user Mention using the given
// displayName, ID and message text. An error is returned if provided values
// are insufficient to create the user mention.
func NewMentionCard(displayName string, id string, msgText string) (Card, error) {
	if msgText == "" {
		return Card{}, fmt.Errorf(
			"required msgText argument is empty: %w",
			ErrMissingValue,
		)
	}

	// Build mention.
	mention, err := NewMention(displayName, id)
	if err != nil {
		return Card{}, err
	}

	// Create basic card.
	textCard, err := NewTextBlockCard(msgText, "", true)
	if err != nil {
		return Card{}, err
	}

	// Update the text block so that it contains the mention text string
	// (required) and user-specified message text string. Use the mention
	// text as a "greeting" or lead-in for the user-specified message
	// text.
	textCard.Body[0].Text = mention.Text +
		" " + textCard.Body[0].Text

	textCard.MSTeams.Entities = append(
		textCard.MSTeams.Entities,
		mention,
	)

	return textCard, nil
}

// NewMessageFromCard is a helper function for creating a new Message based
// off of an existing Card value.
func NewMessageFromCard(card Card) (*Message, error) {
	msg := Message{
		Type: TypeMessage,
	}

	if err := msg.Attach(card); err != nil {
		return nil, err
	}

	return &msg, nil
}

// NewContainer creates an empty Container.
func NewContainer() Container {
	container := Container{
		Type: TypeElementContainer,
	}

	return container
}

// NewHiddenContainer creates an empty Container whose initial state is
// set as hidden from view.
func NewHiddenContainer() Container {
	visible := false
	container := Container{
		Type:    TypeElementContainer,
		Visible: &visible,
	}

	return container
}

// NewColumn creates an empty Column.
func NewColumn() Column {
	column := Column{
		Type: TypeColumn,
	}

	return column
}

// NewColumnSet creates an empty Element of type ColumnSet.
func NewColumnSet() Element {
	columnSet := Element{
		Type: TypeElementColumnSet,
	}

	return columnSet
}

// NewActionSet creates an empty ActionSet.
//
// TODO: Should we create a type alias for ActionSet, or keep it as a "base"
// Element type?
func NewActionSet() Element {
	actionSet := Element{
		Type: TypeElementActionSet,
	}

	return actionSet
}

// NewTextBlock creates a new TextBlock element using the optional user
// specified Text. If specified, text wrapping is enabled.
func NewTextBlock(text string, wrap bool) Element {
	textBlock := Element{
		Type: TypeElementTextBlock,
		Wrap: wrap,
		Text: text,
	}

	return textBlock
}

// NewHiddenTextBlock creates a new TextBlock element using the optional user
// specified Text. If specified, text wrapping is enabled.
//
// The new TextBlock is explicitly hidden from view. To view this Element, the
// caller should set an ID value and then allow toggling visibility by
// referencing this TextBlock's ID from a TargetElement associated with a
// ToggleVisibility Action.
func NewHiddenTextBlock(text string, wrap bool) Element {
	isVisible := false
	textBlock := Element{
		Type:    TypeElementTextBlock,
		Wrap:    wrap,
		Text:    text,
		Visible: &isVisible,
	}

	return textBlock
}

// NewTitleTextBlock uses the specified text to create a new TextBlock
// formatted as a "header" or "title" element. If specified, the TextBlock has
// text wrapping enabled. The effect is meant to emulate the visual effects of
// setting a MessageCard.Title field.
func NewTitleTextBlock(title string, wrap bool) Element {
	return Element{
		Type:   TypeElementTextBlock,
		Wrap:   wrap,
		Text:   title,
		Style:  TextBlockStyleHeading,
		Size:   SizeLarge,
		Weight: WeightBolder,
	}
}

// NewTableCellsWithTextBlock accepts a collection of items that can be converted
// to string values and returns a collection of TableCells, each populated
// with a single TextBlock containing one of the given items.
//
// Example usage:
//
// vals := []int{1, 2, 3}
// items := make([]interface{}, len(vals))
//
//	for i := range vals {
//		items[i] = vals[i]
//	}
//
// tableCells := NewTextBlockTableCells(items)
func NewTableCellsWithTextBlock(items []interface{}) (TableCells, error) {
	if len(items) == 0 {
		return TableCells{}, fmt.Errorf("no data provided: %w", ErrMissingValue)
	}

	cells := make(TableCells, len(items))
	for i, item := range items {
		switch {
		// If an input item is nil, insert an empty table cell in its place.
		case item == nil:
			cell := TableCell{
				Type: TypeTableCell,
			}
			cells[i] = cell
		default:
			block := Element{
				Type: TypeElementTextBlock,
				Text: fmt.Sprintf("%v", item),
			}
			cell := TableCell{
				Type:  TypeTableCell,
				Items: []*Element{&block},
			}
			cells[i] = cell
		}
	}

	return cells, nil
}

// NewTableRowFromCells accepts a collection of TableCell values and returns a
// TableRow populated with those TableCells.
func NewTableRowFromCells(cells ...TableCell) (TableRow, error) {
	if len(cells) == 0 {
		return TableRow{}, fmt.Errorf("no data provided: %w", ErrMissingValue)
	}

	if err := TableCells(cells).Validate(); err != nil {
		return TableRow{}, err
	}

	row := TableRow{
		Type:  TypeTableRow,
		Cells: cells,
	}

	return row, nil
}

// NewTable creates an empty Element of Table type.
func NewTable() Element {
	table := Element{
		Type: TypeElementTable,
	}

	return table
}

// NewTableCellFromElement accepts an Element value and returns a TableCell
// populated with that Element.
func NewTableCellFromElement(element Element) (TableCell, error) {
	if err := element.Validate(); err != nil {
		return TableCell{}, err
	}

	cell := TableCell{
		Type:  TypeTableCell,
		Items: []*Element{&element},
	}

	return cell, nil
}

// NewTableCellFromElements accepts a collection of Element values and returns
// a TableCell populated with those Elements.
func NewTableCellFromElements(elements ...Element) (TableCell, error) {
	if len(elements) == 0 {
		return TableCell{}, fmt.Errorf("no data provided: %w", ErrMissingValue)
	}

	if err := Elements(elements).Validate(); err != nil {
		return TableCell{}, err
	}

	cellItems := make([]*Element, len(elements))
	for i := range elements {
		cellItems[i] = &elements[i]
	}

	cell := TableCell{
		Type:  TypeTableCell,
		Items: cellItems,
	}

	return cell, nil
}

// NewTableWithGridFromTableCells accepts a collection of TableCell values and
// the number of cells that should be inserted per table row. Header values
// are not inserted.
func NewTableWithGridFromTableCells(cells []TableCell, perRow int) (Element, error) {
	switch {
	case len(cells) == 0:
		return Element{}, fmt.Errorf("no data provided: %w", ErrMissingValue)

	case perRow < 0:
		return Element{}, fmt.Errorf("invalid per row value %d provided", perRow)
	}

	if err := TableCells(cells).Validate(); err != nil {
		return Element{}, err
	}

	neededRows := func() int {
		// 	d := float64(len(cells)) / float64(perRow)
		// 	return int(math.Ceil(d))

		d := len(cells) / perRow

		// Round up if the per row count doesn't divide evenly into the number
		// of cells. This will leave us with a ragged, but valid number of
		// cells per row.
		if len(cells)%perRow > 0 {
			d++
		}
		return d
	}

	table := Element{
		Type:              TypeElementTable,
		GridStyle:         ContainerStyleAccent,
		ShowGridLines:     func() *bool { hasGridLines := true; return &hasGridLines }(),
		FirstRowAsHeaders: func() *bool { hasHeaders := false; return &hasHeaders }(),
	}

	// Add columns to table.
	for i := 0; i < perRow; i++ {
		c := Column{
			Type:                           TypeTableColumnDefinition,
			Width:                          1,
			HorizontalCellContentAlignment: HorizontalAlignmentCenter,
			VerticalCellContentAlignment:   VerticalAlignmentCenter,
		}
		table.Columns = append(table.Columns, c)
	}

	tableRows := make(TableRows, 0, neededRows())

	// 	cellsChan := make(chan TableCell)
	// 	go func() {
	// 		for _, cell := range cells {
	// 			cellsChan <- cell
	// 		}
	// 		close(cellsChan)
	// 	}()
	//
	// 	for i := 0; i < neededRows(); i++ {
	// 		tableCells := make([]TableCell, 0, perRow)
	// 		for j := 0; j < perRow; j++ {
	// 			cell := <-cellsChan
	// 			tableCells = append(tableCells, cell)
	// 		}
	//
	// 		tableRow := TableRow{
	// 			Type:  TypeTableRow,
	// 			Cells: tableCells,
	// 		}
	//
	// 		tableRows = append(tableRows, tableRow)
	// 	}

	// Opt for non-channel/non-goroutine implementation.
	var cellCtr int
	for i := 0; i < neededRows(); i++ {
		tableCells := make([]TableCell, 0, perRow)
		for j := 0; j < perRow; j++ {
			cell := cells[cellCtr]
			cellCtr++

			tableCells = append(tableCells, cell)
		}

		tableRow := TableRow{
			Type:  TypeTableRow,
			Cells: tableCells,
		}

		tableRows = append(tableRows, tableRow)
	}

	table.Rows = tableRows

	return table, nil
}

// NewTableFromTableCells accepts a multidimensional collection of TableCell
// values, the number of columns that the table should have, a boolean value
// indicating whether the first row should be treated as a header row and
// another boolean value indicating whether grid lines should be displayed for
// the table.
//
// If the specified number of columns is zero then the number of columns will
// be calculated using the number of values in the first row.
//
// The outer slice is the collection of rows and the inner slice is the
// collection of values. The number of cells per row is determined by the
// number of cell values in that row. If a collection of values for a row is
// empty, an empty row is inserted into the generated table.
func NewTableFromTableCells(cells [][]TableCell, numColumns int, firstRowIsHeaders bool, showGridLines bool) (Element, error) {
	if len(cells) == 0 {
		return Element{}, fmt.Errorf("no data provided: %w", ErrMissingValue)
	}

	for _, row := range cells {
		if err := TableCells(row).Validate(); err != nil {
			return Element{}, err
		}
	}

	neededRows := len(cells)
	neededColumns := func() int {
		switch {
		case numColumns == 0:
			return len(cells[0])
		default:
			return numColumns
		}
	}

	table := Element{
		Type:              TypeElementTable,
		GridStyle:         ContainerStyleAccent,
		ShowGridLines:     &showGridLines,
		FirstRowAsHeaders: &firstRowIsHeaders,
	}

	// Add columns to table equal to the number of values in the first row.
	for i := 0; i < neededColumns(); i++ {
		c := Column{
			Type:                           TypeTableColumnDefinition,
			Width:                          1,
			HorizontalCellContentAlignment: HorizontalAlignmentCenter,
			VerticalCellContentAlignment:   VerticalAlignmentCenter,
		}
		table.Columns = append(table.Columns, c)
	}

	tableRows := make(TableRows, 0, neededRows)
	for _, row := range cells {
		var tableRow TableRow
		// If our input row is empty, insert a cell with empty TextBlock in
		// its place.
		switch {
		case len(row) == 0:
			block := Element{
				Type: TypeElementTextBlock,
				Text: "",
			}
			cell := TableCell{
				Type:  TypeTableCell,
				Items: []*Element{&block},
			}
			tableRow = TableRow{
				Type:  TypeTableRow,
				Cells: []TableCell{cell},
			}
		default:
			tableRow = TableRow{
				Type:  TypeTableRow,
				Cells: row,
			}
		}

		tableRows = append(tableRows, tableRow)
	}

	table.Rows = tableRows

	return table, nil
}

// NewFactSet creates an empty FactSet.
func NewFactSet() FactSet {
	factSet := FactSet{
		Type: TypeElementFactSet,
	}

	return factSet
}

// AddFact adds one or many Fact values to a FactSet. An error is returned if
// the Fact fails validation or if AddFact is called on an unsupported Element
// type.
func (fs *FactSet) AddFact(facts ...Fact) error {
	// Fail early if called on the wrong Element type.
	if fs.Type != TypeElementFactSet {
		return fmt.Errorf(
			"unsupported element type %s; expected %s: %w",
			fs.Type,
			TypeElementFactSet,
			ErrInvalidType,
		)
	}

	if len(facts) == 0 {
		return fmt.Errorf(
			"received empty collection of facts: %w",
			ErrMissingValue,
		)
	}

	// Validate all Fact values before adding them to the collection.
	for _, fact := range facts {
		if err := fact.Validate(); err != nil {
			return err
		}
	}

	fs.Facts = append(fs.Facts, facts...)

	return nil
}

// HasMentionText asserts that a supported Element type contains the required
// Mention text string necessary to link a user mention to a specific Element.
func (e Element) HasMentionText(m Mention) bool {
	switch {
	case e.Type == TypeElementTextBlock:
		if strings.Contains(e.Text, m.Text) {
			return true
		}
		return false

	case e.Type == TypeElementFactSet:
		for _, fact := range e.Facts {
			if strings.Contains(fact.Title, m.Text) ||
				strings.Contains(fact.Value, m.Text) {

				return true
			}
		}
		return false

	default:
		return false
	}
}

// AddTableRow adds one or many TableRow values to an Element of Table type.
// An error is returned if a TableRow value fails validation or if AddRow is
// called on any Element type other than a Table.
func (e *Element) AddTableRow(rows ...TableRow) error {
	if e.Type != TypeElementTable {
		return fmt.Errorf(
			"unsupported element type %s; expected %s: %w",
			e.Type,
			TypeElementTable,
			ErrInvalidType,
		)
	}

	if len(rows) == 0 {
		return fmt.Errorf("no data provided: %w", ErrMissingValue)
	}

	e.Rows = append(e.Rows, rows...)

	return nil
}

// NewActionOpenURL creates a new Action.OpenURL value using the provided URL
// and title. An error is returned if invalid values are supplied.
func NewActionOpenURL(url string, title string) (Action, error) {
	// Accept the user-specified values as-is, use Validate() method to do the
	// heavy lifting.
	action := Action{
		Type:  TypeActionOpenURL,
		Title: title,
		URL:   url,
	}

	err := action.Validate()
	if err != nil {
		return Action{}, err
	}

	return action, nil
}

// NewActionToggleVisibility creates a new Action.ToggleVisibility value using
// the (optionally) provided title text.
//
// NOTE: The caller is responsible for adding required TargetElement values to
// meet validation requirements.
func NewActionToggleVisibility(title string) Action {
	return Action{
		Type:  TypeActionToggleVisibility,
		Title: title,
	}
}

// NewActionSetsFromActions creates a new ActionSet for every
// TeamsActionsDisplayLimit count of Actions given. An error is returned if
// the specified Actions do not pass validation.
func NewActionSetsFromActions(actions ...Action) ([]Element, error) {
	if len(actions) == 0 {
		return nil, fmt.Errorf(
			"received empty collection of actions to create ActionSet: %w",
			ErrMissingValue,
		)
	}

	for _, action := range actions {
		if err := action.Validate(); err != nil {
			return nil, err
		}
	}

	// Create a new ActionSet for every TeamsActionsDisplayLimit count of
	// Actions given.
	actionSetsNeeded := int(math.Ceil(float64(len(actions)) / float64(TeamsActionsDisplayLimit)))
	actionSets := make([]Element, 0, actionSetsNeeded)

	stride := TeamsActionsDisplayLimit
	for i := 0; i < len(actions); i += stride {
		// Ensure that we don't stride past the end of the actions slice.
		if stride > len(actions)-i {
			stride = len(actions) - i
		}

		actionSetItems := actions[i : i+stride]
		actionSet := Element{
			Type:    TypeElementActionSet,
			Actions: actionSetItems,
		}

		actionSets = append(actionSets, actionSet)
	}

	return actionSets, nil
}

// AddElement adds the given Element to the collection of Element values in
// the container. If specified, the Element is inserted at the beginning of
// the collection, otherwise appended to the end.
func (c *Container) AddElement(prepend bool, element Element) error {
	if err := element.Validate(); err != nil {
		return err
	}

	switch prepend {
	case true:
		c.Items = append([]Element{element}, c.Items...)
	case false:
		c.Items = append(c.Items, element)
	}

	return nil
}

// AddAction adds one or more provided Action values to the associated
// Container as one or more new ActionSets. The number of actions in each
// newly created ActionSet is limited to the number specified by
// TeamsActionsDisplayLimit.
//
// If specified, the newly created ActionSets are inserted before other
// Elements in the Container, otherwise appended.
//
// If adding an action to be used when the Container is tapped or selected use
// AddSelectAction() instead.
//
// An error is returned if specified Action values fail validation.
func (c *Container) AddAction(prepend bool, actions ...Action) error {
	// Rely on function to apply validation instead of duplicating it here.
	actionSets, err := NewActionSetsFromActions(actions...)
	if err != nil {
		return err
	}

	switch prepend {
	case true:
		c.Items = append(actionSets, c.Items...)
	case false:
		c.Items = append(c.Items, actionSets...)
	}

	return nil
}

// AddSelectAction adds a given Action or ISelectAction value to the
// associated Container. This action will be invoked when the Container is
// tapped or selected.
//
// An error is returned if the given Action or ISelectAction value fails
// validation or if a value other than an Action or ISelectAction is provided.
func (c *Container) AddSelectAction(action interface{}) error {
	switch v := action.(type) {
	case Action:
		// Perform manual conversion to the supported type.
		selectAction := ISelectAction{
			Type:     v.Type,
			ID:       v.ID,
			Title:    v.Title,
			URL:      v.URL,
			Fallback: v.Fallback,
		}

		// Don't touch the new TargetElements field unless the provided Action
		// has specified values.
		if len(v.TargetElements) > 0 {
			selectAction.TargetElements = append(
				selectAction.TargetElements,
				v.TargetElements...,
			)
		}

		c.SelectAction = &selectAction

	case ISelectAction:
		c.SelectAction = &v

	// unsupported value provided
	default:
		return fmt.Errorf(
			"error: unsupported value provided; "+
				" only Action or ISelectAction values are supported: %w",
			ErrInvalidFieldValue,
		)
	}

	return nil
}

// AddContainer adds the given Container Element to the collection of Element
// values for the Card. If specified, the Container Element is inserted at the
// beginning of the collection, otherwise appended to the end.
func (c *Card) AddContainer(prepend bool, container Container) error {
	element := Element(container)

	if err := element.Validate(); err != nil {
		return err
	}

	switch prepend {
	case true:
		c.Body = append([]Element{element}, c.Body...)
	case false:
		c.Body = append(c.Body, element)
	}

	return nil
}

// NewCodeBlock creates a new CodeBlock element with snippet, language, and
// optional firstLine. This is an MSTeams extension element.
//
// Supported languages include:
//
//   - Bash
//   - C
//   - C#
//   - C++
//   - CSS
//   - DOS
//   - Go
//   - GraphQL
//   - HTML
//   - Java
//   - JavaScript
//   - JSON
//   - Perl
//   - PHP
//   - PlainText
//   - PowerShell
//   - Python
//   - SQL
//   - TypeScript
//   - Verilog
//   - VHDL
//   - Visual Basic
//   - XML
//
// See
// https://learn.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-format
// for additional languages that may be supported.
func NewCodeBlock(snippet string, language string, firstLine int) Element {
	codeBlock := Element{
		Type:            TypeElementMSTeamsCodeBlock,
		CodeSnippet:     snippet,
		Language:        language,
		StartLineNumber: firstLine,
	}
	return codeBlock
}

// cardBodyHasMention indicates whether an Adaptive Card body contains all
// specified Mention values. For every user mention, we require at least one
// match in an applicable Element in the Card Body.
func cardBodyHasMention(body []Element, mentions []Mention) bool {
	// If the card body is empty, it cannot contain the required Mention values.
	if body == nil {
		return false
	}

	elementsHaveMention := func(elements []Element, m Mention) bool {
		for _, element := range elements {
			if element.HasMentionText(m) {
				return true
			}
		}
		return false
	}

	for _, mention := range mentions {
		if !elementsHaveMention(body, mention) {
			return false
		}
	}

	return true
}

// assertHeightAlignmentFieldsSetWhenRequired asserts verticalContentAlignment
// is set when minHeight is set; while both are optional fields, both have to
// be set when the other is.
func assertHeightAlignmentFieldsSetWhenRequired(minHeight string, verticalContentAlignment string) error {
	if minHeight != "" && verticalContentAlignment == "" {
		return fmt.Errorf(
			"field MinHeight is set, VerticalContentAlignment is not;"+
				" field VerticalContentAlignment is only optional when MinHeight"+
				" is not set: %w",
			ErrMissingValue,
		)
	}

	return nil
}

// assertCardBodyHasMention asserts that if there are recorded user mentions,
// then Mention.Text is contained (substring match) within an applicable field
// of a supported Element of the Card Body.
//
// At present, this includes the Text field of a TextBlock Element or
// the Title or Value fields of a Fact from a FactSet.
//
// https://docs.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/cards-format#mention-support-within-adaptive-cards
func assertCardBodyHasMention(elements []Element, mentions []Mention) error {
	// User mentions recorded, but no elements in Card Body to potentially
	// contain required text string.
	if len(mentions) > 0 && len(elements) == 0 {
		return fmt.Errorf(
			"user mention text not found in empty Card Body: %w",
			ErrMissingValue,
		)
	}

	// For every user mention, we require at least one match in an applicable
	// Element in the Card Body.
	if len(mentions) > 0 && !cardBodyHasMention(elements, mentions) {
		return fmt.Errorf(
			"user mention text not found in elements of Card Body: %w",
			ErrMissingValue,
		)
	}

	return nil
}

func assertColumnWidthValidValues(c Column) error {
	switch v := c.Width.(type) {
	// Nothing to see here.
	case nil:

	// Assert specific fixed keyword values, empty string or valid pixel
	// width; all other values are invalid.
	case string:
		v = strings.TrimSpace(v)

		switch {
		case v == ColumnWidthAuto:
		case v == ColumnWidthStretch:
		default:
			if err := assertValidPixelSizeOrEmptyValue(v); err != nil {
				return err
			}
		}

	// Number representing relative width of the column.
	case int:

	// Unsupported value.
	default:
		return fmt.Errorf(
			"invalid pixel width %q; "+
				"expected one of keywords %q, int value (e.g., %d) "+
				"or specific pixel width (e.g., %s): %w",
			v,
			strings.Join([]string{
				ColumnWidthAuto,
				ColumnWidthStretch,
			}, ","),
			1,
			PixelSizeExample,
			ErrInvalidFieldValue,
		)
	}

	return nil
}

func assertTableColumnDefinitionWidthValidValues(tcd TableColumnDefinition) error {
	switch v := tcd.Width.(type) {
	// Nothing to see here.
	case nil:

	// Assert valid pixel width or empty string; all other values are invalid.
	case string:
		if err := assertValidPixelSizeOrEmptyValue(v); err != nil {
			return err
		}

	// Number representing relative width of the column.
	case int:

	// Unsupported value.
	default:
		return fmt.Errorf(
			"invalid pixel width %q; "+
				"expected int value (e.g., %d) "+
				"or specific pixel width (e.g., %s): %w",
			v,
			1,
			PixelSizeExample,
			ErrInvalidFieldValue,
		)
	}

	return nil
}

func assertValidPixelSizeOrEmptyValue(val string) error {
	val = strings.TrimSpace(val)

	// An empty string is a special case and is permitted to honor "optional"
	// field value requirement.
	if val == "" {
		return nil
	}

	matched, _ := regexp.MatchString(PixelSizeRegex, val)

	if !matched {
		return fmt.Errorf(
			"invalid pixel width %q; expected value in format %s: %w",
			val,
			PixelSizeExample,
			ErrInvalidFieldValue,
		)
	}

	// TODO: Apply validation to ensure that 0 is not given as a pixel size?

	return nil
}

func assertValidVersionFieldValue(val string) error {
	switch {
	case strings.TrimSpace(val) == "":
		return fmt.Errorf(
			"required field Version is empty for top-level Card: %w",
			ErrMissingValue,
		)
	default:
		// Assert that Version value can be converted to the expected format.
		versionNum, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return fmt.Errorf(
				"value %q incompatible with Version field: %w",
				val,
				ErrInvalidFieldValue,
			)
		}

		// This is a high confidence validation failure.
		if versionNum < AdaptiveCardMinVersion {
			return fmt.Errorf(
				"unsupported version %q;"+
					" expected minimum value of %0.1f: %w",
				val,
				AdaptiveCardMinVersion,
				ErrInvalidFieldValue,
			)
		}

		// This is *NOT* a high confidence validation failure; it is likely
		// that Microsoft Teams will gain support for future versions of the
		// Adaptive Card greater than the current recorded max configured
		// schema version. Because the max value constant is subject to fall
		// out of sync (at least briefly), this is a risky assertion to make.
		//
		// if versionNum < AdaptiveCardMinVersion || versionNum > AdaptiveCardMaxVersion {
		// 	return fmt.Errorf(
		// 		"unsupported version %q;"+
		// 			" expected value between %0.1f and %0.1f: %w",
		// 		tc.Version,
		// 		AdaptiveCardMinVersion,
		// 		AdaptiveCardMaxVersion,
		// 		ErrInvalidFieldValue,
		// 	)
		// }
	}

	return nil
}
