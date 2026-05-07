package reddit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// WidgetService handles communication with the widget
// related methods of the Reddit API.
//
// Reddit API docs: https://www.reddit.com/dev/api/#section_widgets
type WidgetService struct {
	client *Client
}

// Widget is a section of useful content on a subreddit.
// They can feature information such as rules, links, the origins of the subreddit, etc.
// Read about them here: https://mods.reddithelp.com/hc/en-us/articles/360010364372-Sidebar-Widgets
type Widget interface {
	// kind returns the widget kind.
	// having un unexported method on an exported interface means it cannot be implemented by a client.
	kind() string
	// GetID returns the widget's id.
	GetID() string
}

const (
	widgetKindTextArea         = "textarea"
	widgetKindButton           = "button"
	widgetKindImage            = "image"
	widgetKindCommunityList    = "community-list"
	widgetKindMenu             = "menu"
	widgetKindCommunityDetails = "id-card"
	widgetKindModerators       = "moderators"
	widgetKindSubredditRules   = "subreddit-rules"
	widgetKindCustom           = "custom"
)

type rootWidget struct {
	Data Widget
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (w *rootWidget) UnmarshalJSON(data []byte) error {
	root := new(struct {
		Kind string `json:"kind"`
	})

	err := json.Unmarshal(data, root)
	if err != nil {
		return err
	}

	switch root.Kind {
	case widgetKindTextArea:
		w.Data = new(TextAreaWidget)
	case widgetKindButton:
		w.Data = new(ButtonWidget)
	case widgetKindImage:
		w.Data = new(ImageWidget)
	case widgetKindCommunityList:
		w.Data = new(CommunityListWidget)
	case widgetKindMenu:
		w.Data = new(MenuWidget)
	case widgetKindCommunityDetails:
		w.Data = new(CommunityDetailsWidget)
	case widgetKindModerators:
		w.Data = new(ModeratorsWidget)
	case widgetKindSubredditRules:
		w.Data = new(SubredditRulesWidget)
	case widgetKindCustom:
		w.Data = new(CustomWidget)
	default:
		return fmt.Errorf("unrecognized widget kind: %q", root.Kind)
	}

	return json.Unmarshal(data, w.Data)
}

// WidgetList is a list of widgets.
type WidgetList []Widget

// UnmarshalJSON implements the json.Unmarshaler interface.
func (l *WidgetList) UnmarshalJSON(data []byte) error {
	var widgetMap map[string]json.RawMessage
	err := json.Unmarshal(data, &widgetMap)
	if err != nil {
		return err
	}

	*l = make(WidgetList, 0, len(widgetMap))
	for _, w := range widgetMap {
		root := new(rootWidget)
		err = json.Unmarshal(w, root)
		if err != nil {
			return err
		}

		*l = append(*l, root.Data)
	}

	return nil
}

// common widget fields
type widget struct {
	ID    string       `json:"id,omitempty"`
	Kind  string       `json:"kind,omitempty"`
	Style *WidgetStyle `json:"styles,omitempty"`
}

func (w *widget) kind() string  { return w.Kind }
func (w *widget) GetID() string { return w.ID }

// TextAreaWidget displays a box of text in the subreddit.
type TextAreaWidget struct {
	widget

	Name string `json:"shortName,omitempty"`
	Text string `json:"text,omitempty"`
}

// ButtonWidget displays up to 10 button style links with customizable font colors for each button.
type ButtonWidget struct {
	widget

	Name        string          `json:"shortName,omitempty"`
	Description string          `json:"description,omitempty"`
	Buttons     []*WidgetButton `json:"buttons,omitempty"`
}

// ImageWidget display a random image from up to 10 selected images.
// The image can be clickable links.
type ImageWidget struct {
	widget

	Name   string             `json:"shortName,omitempty"`
	Images []*WidgetImageLink `json:"data,omitempty"`
}

// CommunityListWidget display a list of up to 10 other communities (subreddits).
type CommunityListWidget struct {
	widget

	Name        string             `json:"shortName,omitempty"`
	Communities []*WidgetCommunity `json:"data,omitempty"`
}

// MenuWidget displays tabs for your community's menu. These can be direct links or submenus that
// create a drop-down menu to multiple links.
type MenuWidget struct {
	widget

	ShowWiki bool           `json:"showWiki"`
	Links    WidgetLinkList `json:"data,omitempty"`
}

// CommunityDetailsWidget displays your subscriber count, users online, and community description,
// as defined in your subreddit settings. You can customize the displayed text for subscribers and
// users currently viewing the community.
type CommunityDetailsWidget struct {
	widget

	Name        string `json:"shortName,omitempty"`
	Description string `json:"description,omitempty"`

	Subscribers      int `json:"subscribersCount"`
	CurrentlyViewing int `json:"currentlyViewingCount"`

	SubscribersText      string `json:"subscribersText,omitempty"`
	CurrentlyViewingText string `json:"currentlyViewingText,omitempty"`
}

// ModeratorsWidget displays the list of moderators of the subreddit.
type ModeratorsWidget struct {
	widget

	Mods  []string `json:"mods"`
	Total int      `json:"totalMods"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (w *ModeratorsWidget) UnmarshalJSON(data []byte) error {
	root := new(struct {
		widget

		Mods []struct {
			Name string `json:"name"`
		} `json:"mods"`
		Total int `json:"totalMods"`
	})

	err := json.Unmarshal(data, root)
	if err != nil {
		return err
	}

	w.widget = root.widget
	w.Total = root.Total
	for _, mod := range root.Mods {
		w.Mods = append(w.Mods, mod.Name)
	}

	return nil
}

// SubredditRulesWidget displays your community rules.
type SubredditRulesWidget struct {
	widget

	Name string `json:"shortName,omitempty"`
	// One of: full (includes description), compact (rule is collapsed).
	Display string   `json:"display,omitempty"`
	Rules   []string `json:"rules,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (w *SubredditRulesWidget) UnmarshalJSON(data []byte) error {
	root := new(struct {
		widget

		Name    string `json:"shortName"`
		Display string `json:"display"`
		Rules   []struct {
			Description string `json:"description"`
		} `json:"data"`
	})

	err := json.Unmarshal(data, root)
	if err != nil {
		return err
	}

	w.widget = root.widget
	w.Name = root.Name
	w.Display = root.Display
	for _, r := range root.Rules {
		w.Rules = append(w.Rules, r.Description)
	}

	return nil
}

// CustomWidget is a custom widget.
type CustomWidget struct {
	widget

	Name string `json:"shortName,omitempty"`
	Text string `json:"text,omitempty"`

	StyleSheet    string         `json:"css,omitempty"`
	StyleSheetURL string         `json:"stylesheetUrl,omitempty"`
	Images        []*WidgetImage `json:"imageData,omitempty"`
}

// WidgetStyle contains style information for the widget.
type WidgetStyle struct {
	HeaderColor     string `json:"headerColor,omitempty"`
	BackgroundColor string `json:"backgroundColor,omitempty"`
}

// WidgetImage is an image in a widget.
type WidgetImage struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// WidgetLink is a link or a group of links that's part of a widget.
type WidgetLink interface {
	// single returns whether or not the widget holds just one single link.
	// having un unexported method on an exported interface means it cannot be implemented by a client.
	single() bool
}

// WidgetLinkSingle is a link that's part of a widget.
type WidgetLinkSingle struct {
	Text string `json:"text,omitempty"`
	URL  string `json:"url,omitempty"`
}

func (l *WidgetLinkSingle) single() bool { return true }

// WidgetLinkMultiple is a dropdown of multiple links that's part of a widget.
type WidgetLinkMultiple struct {
	Text string              `json:"text,omitempty"`
	URLs []*WidgetLinkSingle `json:"children,omitempty"`
}

func (l *WidgetLinkMultiple) single() bool { return false }

// WidgetLinkList is a list of widgets links.
type WidgetLinkList []WidgetLink

// UnmarshalJSON implements the json.Unmarshaler interface.
func (l *WidgetLinkList) UnmarshalJSON(data []byte) error {
	var dataMap []json.RawMessage
	err := json.Unmarshal(data, &dataMap)
	if err != nil {
		return err
	}

	*l = make(WidgetLinkList, 0, len(dataMap))
	for _, d := range dataMap {
		var widgetLinkDataMap map[string]json.RawMessage
		err = json.Unmarshal(d, &widgetLinkDataMap)
		if err != nil {
			return err
		}

		var wl WidgetLink
		if _, ok := widgetLinkDataMap["children"]; ok {
			wl = new(WidgetLinkMultiple)
		} else {
			wl = new(WidgetLinkSingle)
		}

		err = json.Unmarshal(d, wl)
		if err != nil {
			return err
		}

		*l = append(*l, wl)
	}

	return nil
}

// WidgetImageLink is an image that links to an URL within a widget.
type WidgetImageLink struct {
	URL     string `json:"url,omitempty"`
	LinkURL string `json:"linkURL,omitempty"`
}

// WidgetCommunity is a community (subreddit) that's displayed in a widget.
type WidgetCommunity struct {
	Name        string `json:"name,omitempty"`
	Subscribers int    `json:"subscribers"`
	Subscribed  bool   `json:"isSubscribed"`
	NSFW        bool   `json:"isNSFW"`
}

// WidgetButton is a button that's part of a widget.
type WidgetButton struct {
	Text      string `json:"text,omitempty"`
	URL       string `json:"url,omitempty"`
	TextColor string `json:"textColor,omitempty"`
	FillColor string `json:"fillColor,omitempty"`
	// The color of the button's "outline".
	StrokeColor string                  `json:"color,omitempty"`
	HoverState  *WidgetButtonHoverState `json:"hoverState,omitempty"`
}

// WidgetButtonHoverState is the behaviour of a button that's part of a widget when it's hovered over with the mouse.
type WidgetButtonHoverState struct {
	Text      string `json:"text,omitempty"`
	TextColor string `json:"textColor,omitempty"`
	FillColor string `json:"fillColor,omitempty"`
	// The color of the button's "outline".
	StrokeColor string `json:"color,omitempty"`
}

// WidgetCreateRequest represents a request to create a widget.
type WidgetCreateRequest interface {
	requestKind() string
}

// TextAreaWidgetCreateRequest represents a requets to create a text area widget.
type TextAreaWidgetCreateRequest struct {
	Style *WidgetStyle `json:"styles,omitempty"`
	// No longer than 30 characters.
	Name string `json:"shortName,omitempty"`
	// Raw markdown text.
	Text string `json:"text,omitempty"`
}

func (*TextAreaWidgetCreateRequest) requestKind() string { return widgetKindTextArea }

// MarshalJSON implements the json.Marshaler interface.
func (r *TextAreaWidgetCreateRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Kind  string       `json:"kind"`
		Style *WidgetStyle `json:"styles,omitempty"`
		Name  string       `json:"shortName,omitempty"`
		Text  string       `json:"text,omitempty"`
	}{r.requestKind(), r.Style, r.Name, r.Text})
}

// CommunityListWidgetCreateRequest represents a requets to create a community list widget.
type CommunityListWidgetCreateRequest struct {
	Style *WidgetStyle `json:"styles,omitempty"`
	// No longer than 30 characters.
	Name        string   `json:"shortName,omitempty"`
	Communities []string `json:"data,omitempty"`
}

func (*CommunityListWidgetCreateRequest) requestKind() string { return widgetKindCommunityList }

// MarshalJSON implements the json.Marshaler interface.
func (r *CommunityListWidgetCreateRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Kind        string       `json:"kind"`
		Style       *WidgetStyle `json:"styles,omitempty"`
		Name        string       `json:"shortName,omitempty"`
		Communities []string     `json:"data,omitempty"`
	}{r.requestKind(), r.Style, r.Name, r.Communities})
}

// Get the subreddit's widgets.
func (s *WidgetService) Get(ctx context.Context, subreddit string) ([]Widget, *Response, error) {
	path := fmt.Sprintf("r/%s/api/widgets?progressive_images=true", subreddit)
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(struct {
		Widgets WidgetList `json:"items"`
	})
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.Widgets, resp, nil
}

// Create a widget for the subreddit.
func (s *WidgetService) Create(ctx context.Context, subreddit string, request WidgetCreateRequest) (Widget, *Response, error) {
	if request == nil {
		return nil, nil, errors.New("WidgetCreateRequest: cannot be nil")
	}

	path := fmt.Sprintf("r/%s/api/widget", subreddit)
	req, err := s.client.NewJSONRequest(http.MethodPost, path, request)
	if err != nil {
		return nil, nil, err
	}

	root := new(rootWidget)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.Data, resp, nil
}

// Delete a widget via its id.
func (s *WidgetService) Delete(ctx context.Context, subreddit, id string) (*Response, error) {
	path := fmt.Sprintf("r/%s/api/widget/%s", subreddit, id)
	req, err := s.client.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

// Reorder the widgets in the subreddit.
// The order should contain every single widget id in the subreddit; omitting any id will result in an error.
// The id list should only contain sidebar widgets. It should exclude the community details and moderators widgets.
func (s *WidgetService) Reorder(ctx context.Context, subreddit string, ids []string) (*Response, error) {
	path := fmt.Sprintf("r/%s/api/widget_order/sidebar", subreddit)
	req, err := s.client.NewJSONRequest(http.MethodPatch, path, ids)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}
