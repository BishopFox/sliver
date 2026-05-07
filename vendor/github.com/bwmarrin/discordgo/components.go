package discordgo

import (
	"encoding/json"
	"fmt"
)

// ComponentType is type of component.
type ComponentType uint

// MessageComponent types.
const (
	ActionsRowComponent            ComponentType = 1
	ButtonComponent                ComponentType = 2
	SelectMenuComponent            ComponentType = 3
	TextInputComponent             ComponentType = 4
	UserSelectMenuComponent        ComponentType = 5
	RoleSelectMenuComponent        ComponentType = 6
	MentionableSelectMenuComponent ComponentType = 7
	ChannelSelectMenuComponent     ComponentType = 8
	SectionComponent               ComponentType = 9
	TextDisplayComponent           ComponentType = 10
	ThumbnailComponent             ComponentType = 11
	MediaGalleryComponent          ComponentType = 12
	FileComponentType              ComponentType = 13
	SeparatorComponent             ComponentType = 14
	ContainerComponent             ComponentType = 17
)

// MessageComponent is a base interface for all message components.
type MessageComponent interface {
	json.Marshaler
	Type() ComponentType
}

type unmarshalableMessageComponent struct {
	MessageComponent
}

// UnmarshalJSON is a helper function to unmarshal MessageComponent object.
func (umc *unmarshalableMessageComponent) UnmarshalJSON(src []byte) error {
	var v struct {
		Type ComponentType `json:"type"`
	}
	err := json.Unmarshal(src, &v)
	if err != nil {
		return err
	}

	switch v.Type {
	case ActionsRowComponent:
		umc.MessageComponent = &ActionsRow{}
	case ButtonComponent:
		umc.MessageComponent = &Button{}
	case SelectMenuComponent, ChannelSelectMenuComponent, UserSelectMenuComponent,
		RoleSelectMenuComponent, MentionableSelectMenuComponent:
		umc.MessageComponent = &SelectMenu{}
	case TextInputComponent:
		umc.MessageComponent = &TextInput{}
	case SectionComponent:
		umc.MessageComponent = &Section{}
	case TextDisplayComponent:
		umc.MessageComponent = &TextDisplay{}
	case ThumbnailComponent:
		umc.MessageComponent = &Thumbnail{}
	case MediaGalleryComponent:
		umc.MessageComponent = &MediaGallery{}
	case FileComponentType:
		umc.MessageComponent = &FileComponent{}
	case SeparatorComponent:
		umc.MessageComponent = &Separator{}
	case ContainerComponent:
		umc.MessageComponent = &Container{}
	default:
		return fmt.Errorf("unknown component type: %d", v.Type)
	}
	return json.Unmarshal(src, umc.MessageComponent)
}

// MessageComponentFromJSON is a helper function for unmarshaling message components
func MessageComponentFromJSON(b []byte) (MessageComponent, error) {
	var u unmarshalableMessageComponent
	err := u.UnmarshalJSON(b)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal into MessageComponent: %w", err)
	}
	return u.MessageComponent, nil
}

// ActionsRow is a top-level container component for displaying a row of interactive components.
type ActionsRow struct {
	// Can contain Button, SelectMenu and TextInput.
	// NOTE: maximum of 5.
	Components []MessageComponent `json:"components"`
	// Unique identifier for the component; auto populated through increment if not provided.
	ID int `json:"id,omitempty"`
}

// MarshalJSON is a method for marshaling ActionsRow to a JSON object.
func (r ActionsRow) MarshalJSON() ([]byte, error) {
	type actionsRow ActionsRow

	return Marshal(struct {
		actionsRow
		Type ComponentType `json:"type"`
	}{
		actionsRow: actionsRow(r),
		Type:       r.Type(),
	})
}

// UnmarshalJSON is a helper function to unmarshal Actions Row.
func (r *ActionsRow) UnmarshalJSON(data []byte) error {
	type actionsRow ActionsRow
	var v struct {
		actionsRow
		RawComponents []unmarshalableMessageComponent `json:"components"`
	}
	err := json.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	*r = ActionsRow(v.actionsRow)

	r.Components = make([]MessageComponent, len(v.RawComponents))
	for i, v := range v.RawComponents {
		r.Components[i] = v.MessageComponent
	}

	return err
}

// Type is a method to get the type of a component.
func (r ActionsRow) Type() ComponentType {
	return ActionsRowComponent
}

// ButtonStyle is style of button.
type ButtonStyle uint

// Button styles.
const (
	// PrimaryButton is a button with blurple color.
	PrimaryButton ButtonStyle = 1
	// SecondaryButton is a button with grey color.
	SecondaryButton ButtonStyle = 2
	// SuccessButton is a button with green color.
	SuccessButton ButtonStyle = 3
	// DangerButton is a button with red color.
	DangerButton ButtonStyle = 4
	// LinkButton is a special type of button which navigates to a URL. Has grey color.
	LinkButton ButtonStyle = 5
	// PremiumButton is a special type of button with a blurple color that links to a SKU.
	PremiumButton ButtonStyle = 6
)

// ComponentEmoji represents button emoji, if it does have one.
type ComponentEmoji struct {
	Name     string `json:"name,omitempty"`
	ID       string `json:"id,omitempty"`
	Animated bool   `json:"animated,omitempty"`
}

// Button represents button component.
type Button struct {
	Label    string          `json:"label"`
	Style    ButtonStyle     `json:"style"`
	Disabled bool            `json:"disabled"`
	Emoji    *ComponentEmoji `json:"emoji,omitempty"`

	// NOTE: Only button with LinkButton style can have link. Also, URL is mutually exclusive with CustomID.
	URL      string `json:"url,omitempty"`
	CustomID string `json:"custom_id,omitempty"`
	// Identifier for a purchasable SKU. Only available when using premium-style buttons.
	SKUID string `json:"sku_id,omitempty"`
	// Unique identifier for the component; auto populated through increment if not provided.
	ID int `json:"id,omitempty"`
}

// MarshalJSON is a method for marshaling Button to a JSON object.
func (b Button) MarshalJSON() ([]byte, error) {
	type button Button

	if b.Style == 0 {
		b.Style = PrimaryButton
	}

	return Marshal(struct {
		button
		Type ComponentType `json:"type"`
	}{
		button: button(b),
		Type:   b.Type(),
	})
}

// Type is a method to get the type of a component.
func (Button) Type() ComponentType {
	return ButtonComponent
}

// SelectMenuOption represents an option for a select menu.
type SelectMenuOption struct {
	Label       string          `json:"label,omitempty"`
	Value       string          `json:"value"`
	Description string          `json:"description"`
	Emoji       *ComponentEmoji `json:"emoji,omitempty"`
	// Determines whenever option is selected by default or not.
	Default bool `json:"default"`
}

// SelectMenuDefaultValueType represents the type of an entity selected by default in auto-populated select menus.
type SelectMenuDefaultValueType string

// SelectMenuDefaultValue types.
const (
	SelectMenuDefaultValueUser    SelectMenuDefaultValueType = "user"
	SelectMenuDefaultValueRole    SelectMenuDefaultValueType = "role"
	SelectMenuDefaultValueChannel SelectMenuDefaultValueType = "channel"
)

// SelectMenuDefaultValue represents an entity selected by default in auto-populated select menus.
type SelectMenuDefaultValue struct {
	// ID of the entity.
	ID string `json:"id"`
	// Type of the entity.
	Type SelectMenuDefaultValueType `json:"type"`
}

// SelectMenuType represents select menu type.
type SelectMenuType ComponentType

// SelectMenu types.
const (
	StringSelectMenu      = SelectMenuType(SelectMenuComponent)
	UserSelectMenu        = SelectMenuType(UserSelectMenuComponent)
	RoleSelectMenu        = SelectMenuType(RoleSelectMenuComponent)
	MentionableSelectMenu = SelectMenuType(MentionableSelectMenuComponent)
	ChannelSelectMenu     = SelectMenuType(ChannelSelectMenuComponent)
)

// SelectMenu represents select menu component.
type SelectMenu struct {
	// Type of the select menu.
	MenuType SelectMenuType `json:"type,omitempty"`
	// CustomID is a developer-defined identifier for the select menu.
	CustomID string `json:"custom_id,omitempty"`
	// The text which will be shown in the menu if there's no default options or all options was deselected and component was closed.
	Placeholder string `json:"placeholder"`
	// This value determines the minimal amount of selected items in the menu.
	MinValues *int `json:"min_values,omitempty"`
	// This value determines the maximal amount of selected items in the menu.
	// If MaxValues or MinValues are greater than one then the user can select multiple items in the component.
	MaxValues int `json:"max_values,omitempty"`
	// List of default values for auto-populated select menus.
	// NOTE: Number of entries should be in the range defined by MinValues and MaxValues.
	DefaultValues []SelectMenuDefaultValue `json:"default_values,omitempty"`

	Options  []SelectMenuOption `json:"options,omitempty"`
	Disabled bool               `json:"disabled"`

	// NOTE: Can only be used in SelectMenu with Channel menu type.
	ChannelTypes []ChannelType `json:"channel_types,omitempty"`

	// Unique identifier for the component; auto populated through increment if not provided.
	ID int `json:"id,omitempty"`
}

// Type is a method to get the type of a component.
func (s SelectMenu) Type() ComponentType {
	if s.MenuType != 0 {
		return ComponentType(s.MenuType)
	}
	return SelectMenuComponent
}

// MarshalJSON is a method for marshaling SelectMenu to a JSON object.
func (s SelectMenu) MarshalJSON() ([]byte, error) {
	type selectMenu SelectMenu

	return Marshal(struct {
		selectMenu
		Type ComponentType `json:"type"`
	}{
		selectMenu: selectMenu(s),
		Type:       s.Type(),
	})
}

// TextInput represents text input component.
type TextInput struct {
	CustomID    string         `json:"custom_id"`
	Label       string         `json:"label"`
	Style       TextInputStyle `json:"style"`
	Placeholder string         `json:"placeholder,omitempty"`
	Value       string         `json:"value,omitempty"`
	Required    bool           `json:"required"`
	MinLength   int            `json:"min_length,omitempty"`
	MaxLength   int            `json:"max_length,omitempty"`

	// Unique identifier for the component; auto populated through increment if not provided.
	ID int `json:"id,omitempty"`
}

// Type is a method to get the type of a component.
func (TextInput) Type() ComponentType {
	return TextInputComponent
}

// MarshalJSON is a method for marshaling TextInput to a JSON object.
func (m TextInput) MarshalJSON() ([]byte, error) {
	type inputText TextInput

	return Marshal(struct {
		inputText
		Type ComponentType `json:"type"`
	}{
		inputText: inputText(m),
		Type:      m.Type(),
	})
}

// TextInputStyle is style of text in TextInput component.
type TextInputStyle uint

// Text styles
const (
	TextInputShort     TextInputStyle = 1
	TextInputParagraph TextInputStyle = 2
)

// Section is a top-level layout component that allows you to join text contextually with an accessory.
type Section struct {
	// Unique identifier for the component; auto populated through increment if not provided.
	ID int `json:"id,omitempty"`
	// Array of text display components; max of 3.
	Components []MessageComponent `json:"components"`
	// Can be Button or Thumbnail
	Accessory MessageComponent `json:"accessory"`
}

// UnmarshalJSON is a method for unmarshaling Section from JSON
func (s *Section) UnmarshalJSON(data []byte) error {
	type section Section

	var v struct {
		section
		RawComponents []unmarshalableMessageComponent `json:"components"`
		RawAccessory  unmarshalableMessageComponent   `json:"accessory"`
	}

	err := json.Unmarshal(data, &v)
	if err != nil {
		return err
	}

	*s = Section(v.section)
	s.Accessory = v.RawAccessory.MessageComponent
	s.Components = make([]MessageComponent, len(v.RawComponents))
	for i, v := range v.RawComponents {
		s.Components[i] = v.MessageComponent
	}

	return nil
}

// Type is a method to get the type of a component.
func (Section) Type() ComponentType {
	return SectionComponent
}

// MarshalJSON is a method for marshaling Section to a JSON object.
func (s Section) MarshalJSON() ([]byte, error) {
	type section Section

	return Marshal(struct {
		section
		Type ComponentType `json:"type"`
	}{
		section: section(s),
		Type:    s.Type(),
	})
}

// TextDisplay is a top-level component that allows you to add markdown-formatted text to the message.
type TextDisplay struct {
	Content string `json:"content"`
}

// Type is a method to get the type of a component.
func (TextDisplay) Type() ComponentType {
	return TextDisplayComponent
}

// MarshalJSON is a method for marshaling TextDisplay to a JSON object.
func (t TextDisplay) MarshalJSON() ([]byte, error) {
	type textDisplay TextDisplay

	return Marshal(struct {
		textDisplay
		Type ComponentType `json:"type"`
	}{
		textDisplay: textDisplay(t),
		Type:        t.Type(),
	})
}

// Thumbnail component can be used as an accessory for a section component.
type Thumbnail struct {
	// Unique identifier for the component; auto populated through increment if not provided.
	ID          int               `json:"id,omitempty"`
	Media       UnfurledMediaItem `json:"media"`
	Description *string           `json:"description,omitempty"`
	Spoiler     bool              `json:"spoiler,omitemoty"`
}

// Type is a method to get the type of a component.
func (Thumbnail) Type() ComponentType {
	return ThumbnailComponent
}

// MarshalJSON is a method for marshaling Thumbnail to a JSON object.
func (t Thumbnail) MarshalJSON() ([]byte, error) {
	type thumbnail Thumbnail

	return Marshal(struct {
		thumbnail
		Type ComponentType `json:"type"`
	}{
		thumbnail: thumbnail(t),
		Type:      t.Type(),
	})
}

// MediaGallery is a top-level component allows you to group images, videos or gifs into a gallery grid.
type MediaGallery struct {
	// Unique identifier for the component; auto populated through increment if not provided.
	ID int `json:"id,omitempty"`
	// Array of media gallery items; max of 10.
	Items []MediaGalleryItem `json:"items"`
}

// Type is a method to get the type of a component.
func (MediaGallery) Type() ComponentType {
	return MediaGalleryComponent
}

// MarshalJSON is a method for marshaling MediaGallery to a JSON object.
func (m MediaGallery) MarshalJSON() ([]byte, error) {
	type mediaGallery MediaGallery

	return Marshal(struct {
		mediaGallery
		Type ComponentType `json:"type"`
	}{
		mediaGallery: mediaGallery(m),
		Type:         m.Type(),
	})
}

// MediaGalleryItem represents an item used in MediaGallery.
type MediaGalleryItem struct {
	Media       UnfurledMediaItem `json:"media"`
	Description *string           `json:"description,omitempty"`
	Spoiler     bool              `json:"spoiler"`
}

// FileComponent is a top-level component that allows you to display an uploaded file as an attachment to the message and reference it in the component.
type FileComponent struct {
	// Unique identifier for the component; auto populated through increment if not provided.
	ID      int               `json:"id,omitempty"`
	File    UnfurledMediaItem `json:"file"`
	Spoiler bool              `json:"spoiler"`
}

// Type is a method to get the type of a component.
func (FileComponent) Type() ComponentType {
	return FileComponentType
}

// MarshalJSON is a method for marshaling FileComponent to a JSON object.
func (f FileComponent) MarshalJSON() ([]byte, error) {
	type fileComponent FileComponent

	return Marshal(struct {
		fileComponent
		Type ComponentType `json:"type"`
	}{
		fileComponent: fileComponent(f),
		Type:          f.Type(),
	})
}

// SeparatorSpacingSize represents spacing size around the separator.
type SeparatorSpacingSize uint

// Separator spacing sizes.
const (
	SeparatorSpacingSizeSmall SeparatorSpacingSize = 1
	SeparatorSpacingSizeLarge SeparatorSpacingSize = 2
)

// Separator is a top-level layout component that adds vertical padding and visual division between other components.
type Separator struct {
	// Unique identifier for the component; auto populated through increment if not provided.
	ID int `json:"id,omitempty"`

	Divider *bool                 `json:"divider,omitempty"`
	Spacing *SeparatorSpacingSize `json:"spacing,omitempty"`
}

// Type is a method to get the type of a component.
func (Separator) Type() ComponentType {
	return SeparatorComponent
}

// MarshalJSON is a method for marshaling Separator to a JSON object.
func (s Separator) MarshalJSON() ([]byte, error) {
	type separator Separator

	return Marshal(struct {
		separator
		Type ComponentType `json:"type"`
	}{
		separator: separator(s),
		Type:      s.Type(),
	})
}

// Container is a top-level layout component.
// Containers are visually distinct from surrounding components and have an optional customizable color bar (similar to embeds).
type Container struct {
	// Unique identifier for the component; auto populated through increment if not provided.
	ID          int                `json:"id,omitempty"`
	AccentColor *int               `json:"accent_color,omitempty"`
	Spoiler     bool               `json:"spoiler"`
	Components  []MessageComponent `json:"components"`
}

// Type is a method to get the type of a component.
func (Container) Type() ComponentType {
	return ContainerComponent
}

// UnmarshalJSON is a method for unmarshaling Container from JSON
func (c *Container) UnmarshalJSON(data []byte) error {
	type container Container

	var v struct {
		container
		RawComponents []unmarshalableMessageComponent `json:"components"`
	}

	err := json.Unmarshal(data, &v)
	if err != nil {
		return err
	}

	*c = Container(v.container)
	c.Components = make([]MessageComponent, len(v.RawComponents))
	for i, v := range v.RawComponents {
		c.Components[i] = v.MessageComponent
	}

	return nil
}

// MarshalJSON is a method for marshaling Container to a JSON object.
func (c Container) MarshalJSON() ([]byte, error) {
	type container Container

	return Marshal(struct {
		container
		Type ComponentType `json:"type"`
	}{
		container: container(c),
		Type:      c.Type(),
	})
}

// UnfurledMediaItem represents an unfurled media item.
type UnfurledMediaItem struct {
	URL string `json:"url"`
}

// UnfurledMediaItemLoadingState is the loading state of the unfurled media item.
type UnfurledMediaItemLoadingState uint

// Unfurled media item loading states.
const (
	UnfurledMediaItemLoadingStateUnknown        UnfurledMediaItemLoadingState = 0
	UnfurledMediaItemLoadingStateLoading        UnfurledMediaItemLoadingState = 1
	UnfurledMediaItemLoadingStateLoadingSuccess UnfurledMediaItemLoadingState = 2
	UnfurledMediaItemLoadingStateLoadedNotFound UnfurledMediaItemLoadingState = 3
)

// ResolvedUnfurledMediaItem represents a resolved unfurled media item.
type ResolvedUnfurledMediaItem struct {
	URL         string `json:"url"`
	ProxyURL    string `json:"proxy_url"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	ContentType string `json:"content_type"`
}
