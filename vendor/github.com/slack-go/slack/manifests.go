package slack

import (
	"context"
	"encoding/json"
	"net/url"
)

// Manifest is an application manifest schema
type Manifest struct {
	Metadata    ManifestMetadata `json:"_metadata,omitempty" yaml:"_metadata,omitempty"`
	Display     Display          `json:"display_information" yaml:"display_information"`
	Settings    Settings         `json:"settings,omitempty" yaml:"settings,omitempty"`
	Features    Features         `json:"features,omitempty" yaml:"features,omitempty"`
	OAuthConfig OAuthConfig      `json:"oauth_config,omitempty" yaml:"oauth_config,omitempty"`
}

// CreateManifest creates an app from an app manifest.
// For more details, see CreateManifestContext documentation.
func (api *Client) CreateManifest(manifest *Manifest, token string) (*ManifestResponse, error) {
	return api.CreateManifestContext(context.Background(), manifest, token)
}

// CreateManifestContext creates an app from an app manifest with a custom context.
// Slack API docs: https://api.slack.com/methods/apps.manifest.create
func (api *Client) CreateManifestContext(ctx context.Context, manifest *Manifest, token string) (*ManifestResponse, error) {
	if token == "" {
		token = api.configToken
	}

	jsonBytes, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}

	values := url.Values{
		"token":    {token},
		"manifest": {string(jsonBytes)},
	}

	response := &ManifestResponse{}
	err = api.postMethod(ctx, "apps.manifest.create", values, response)
	if err != nil {
		return nil, err
	}

	return response, response.Err()
}

// DeleteManifest permanently deletes an app created through app manifests.
// For more details, see DeleteManifestContext documentation.
func (api *Client) DeleteManifest(token string, appId string) (*SlackResponse, error) {
	return api.DeleteManifestContext(context.Background(), token, appId)
}

// DeleteManifestContext permanently deletes an app created through app manifests with a custom context.
// Slack API docs: https://api.slack.com/methods/apps.manifest.delete
func (api *Client) DeleteManifestContext(ctx context.Context, token string, appId string) (*SlackResponse, error) {
	if token == "" {
		token = api.configToken
	}

	values := url.Values{
		"token":  {token},
		"app_id": {appId},
	}

	response := &SlackResponse{}
	err := api.postMethod(ctx, "apps.manifest.delete", values, response)
	if err != nil {
		return nil, err
	}

	return response, response.Err()
}

// ExportManifest exports an app manifest from an existing app.
// For more details, see ExportManifestContext documentation.
func (api *Client) ExportManifest(token string, appId string) (*Manifest, error) {
	return api.ExportManifestContext(context.Background(), token, appId)
}

// ExportManifestContext exports an app manifest from an existing app with a custom context.
// Slack API docs: https://api.slack.com/methods/apps.manifest.export
func (api *Client) ExportManifestContext(ctx context.Context, token string, appId string) (*Manifest, error) {
	if token == "" {
		token = api.configToken
	}

	values := url.Values{
		"token":  {token},
		"app_id": {appId},
	}

	response := &ExportManifestResponse{}
	err := api.postMethod(ctx, "apps.manifest.export", values, response)
	if err != nil {
		return nil, err
	}

	return &response.Manifest, response.Err()
}

// UpdateManifest updates an app from an app manifest.
// For more details, see UpdateManifestContext documentation.
func (api *Client) UpdateManifest(manifest *Manifest, token string, appId string) (*UpdateManifestResponse, error) {
	return api.UpdateManifestContext(context.Background(), manifest, token, appId)
}

// UpdateManifestContext updates an app from an app manifest with a custom context.
// Slack API docs: https://api.slack.com/methods/apps.manifest.update
func (api *Client) UpdateManifestContext(ctx context.Context, manifest *Manifest, token string, appId string) (*UpdateManifestResponse, error) {
	if token == "" {
		token = api.configToken
	}

	jsonBytes, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}

	values := url.Values{
		"token":    {token},
		"app_id":   {appId},
		"manifest": {string(jsonBytes)},
	}

	response := &UpdateManifestResponse{}
	err = api.postMethod(ctx, "apps.manifest.update", values, response)
	if err != nil {
		return nil, err
	}

	return response, response.Err()
}

// ValidateManifest sends a request to apps.manifest.validate to validate your app manifest.
// For more details, see ValidateManifestContext documentation.
func (api *Client) ValidateManifest(manifest *Manifest, token string, appId string) (*ManifestResponse, error) {
	return api.ValidateManifestContext(context.Background(), manifest, token, appId)
}

// ValidateManifestContext sends a request to apps.manifest.validate to validate your app manifest with a custom context.
// Slack API docs: https://api.slack.com/methods/apps.manifest.validate
func (api *Client) ValidateManifestContext(ctx context.Context, manifest *Manifest, token string, appId string) (*ManifestResponse, error) {
	if token == "" {
		token = api.configToken
	}

	// Marshal manifest into string
	jsonBytes, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}

	values := url.Values{
		"token":    {token},
		"manifest": {string(jsonBytes)},
	}

	if appId != "" {
		values.Add("app_id", appId)
	}

	response := &ManifestResponse{}
	err = api.postMethod(ctx, "apps.manifest.validate", values, response)
	if err != nil {
		return nil, err
	}

	return response, response.Err()
}

// ManifestMetadata is a group of settings that describe the manifest
type ManifestMetadata struct {
	MajorVersion int `json:"major_version,omitempty" yaml:"major_version,omitempty"`
	MinorVersion int `json:"minor_version,omitempty" yaml:"minor_version,omitempty"`
}

// Display is a group of settings that describe parts of an app's appearance within Slack
type Display struct {
	Name            string `json:"name" yaml:"name"`
	Description     string `json:"description,omitempty" yaml:"description,omitempty"`
	LongDescription string `json:"long_description,omitempty" yaml:"long_description,omitempty"`
	BackgroundColor string `json:"background_color,omitempty" yaml:"background_color,omitempty"`
}

// Settings is a group of settings corresponding to the Settings section of the app config pages.
type Settings struct {
	AllowedIPAddressRanges []string           `json:"allowed_ip_address_ranges,omitempty" yaml:"allowed_ip_address_ranges,omitempty"`
	EventSubscriptions     EventSubscriptions `json:"event_subscriptions,omitempty" yaml:"event_subscriptions,omitempty"`
	Interactivity          Interactivity      `json:"interactivity,omitempty" yaml:"interactivity,omitempty"`
	OrgDeployEnabled       bool               `json:"org_deploy_enabled,omitempty" yaml:"org_deploy_enabled,omitempty"`
	SocketModeEnabled      bool               `json:"socket_mode_enabled,omitempty" yaml:"socket_mode_enabled,omitempty"`
}

// EventSubscriptions is a group of settings that describe the Events API configuration
type EventSubscriptions struct {
	RequestUrl string   `json:"request_url,omitempty" yaml:"request_url,omitempty"`
	BotEvents  []string `json:"bot_events,omitempty" yaml:"bot_events,omitempty"`
	UserEvents []string `json:"user_events,omitempty" yaml:"user_events,omitempty"`
}

// Interactivity is a group of settings that describe the interactivity configuration
type Interactivity struct {
	IsEnabled             bool   `json:"is_enabled" yaml:"is_enabled"`
	RequestUrl            string `json:"request_url,omitempty" yaml:"request_url,omitempty"`
	MessageMenuOptionsUrl string `json:"message_menu_options_url,omitempty" yaml:"message_menu_options_url,omitempty"`
}

// Features is a group of settings corresponding to the Features section of the app config pages
type Features struct {
	AppHome       AppHome                `json:"app_home,omitempty" yaml:"app_home,omitempty"`
	BotUser       BotUser                `json:"bot_user,omitempty" yaml:"bot_user,omitempty"`
	Shortcuts     []Shortcut             `json:"shortcuts,omitempty" yaml:"shortcuts,omitempty"`
	SlashCommands []ManifestSlashCommand `json:"slash_commands,omitempty" yaml:"slash_commands,omitempty"`
	WorkflowSteps []WorkflowStep         `json:"workflow_steps,omitempty" yaml:"workflow_steps,omitempty"`
}

// AppHome is a group of settings that describe the App Home configuration
type AppHome struct {
	HomeTabEnabled             bool `json:"home_tab_enabled,omitempty" yaml:"home_tab_enabled,omitempty"`
	MessagesTabEnabled         bool `json:"messages_tab_enabled,omitempty" yaml:"messages_tab_enabled,omitempty"`
	MessagesTabReadOnlyEnabled bool `json:"messages_tab_read_only_enabled,omitempty" yaml:"messages_tab_read_only_enabled,omitempty"`
}

// BotUser is a group of settings that describe bot user configuration
type BotUser struct {
	DisplayName  string `json:"display_name" yaml:"display_name"`
	AlwaysOnline bool   `json:"always_online,omitempty" yaml:"always_online,omitempty"`
}

// Shortcut is a group of settings that describes shortcut configuration
type Shortcut struct {
	Name        string       `json:"name" yaml:"name"`
	CallbackID  string       `json:"callback_id" yaml:"callback_id"`
	Description string       `json:"description" yaml:"description"`
	Type        ShortcutType `json:"type" yaml:"type"`
}

// ShortcutType is a new string type for the available types of shortcuts
type ShortcutType string

const (
	MessageShortcut ShortcutType = "message"
	GlobalShortcut  ShortcutType = "global"
)

// ManifestSlashCommand is a group of settings that describes slash command configuration
type ManifestSlashCommand struct {
	Command      string `json:"command" yaml:"command"`
	Description  string `json:"description" yaml:"description"`
	ShouldEscape bool   `json:"should_escape,omitempty" yaml:"should_escape,omitempty"`
	Url          string `json:"url,omitempty" yaml:"url,omitempty"`
	UsageHint    string `json:"usage_hint,omitempty" yaml:"usage_hint,omitempty"`
}

// WorkflowStep is a group of settings that describes workflow steps configuration
type WorkflowStep struct {
	Name       string `json:"name" yaml:"name"`
	CallbackID string `json:"callback_id" yaml:"callback_id"`
}

// OAuthConfig is a group of settings that describe OAuth configuration for the app
type OAuthConfig struct {
	RedirectUrls []string    `json:"redirect_urls,omitempty" yaml:"redirect_urls,omitempty"`
	Scopes       OAuthScopes `json:"scopes,omitempty" yaml:"scopes,omitempty"`
}

// OAuthScopes is a group of settings that describe permission scopes configuration
type OAuthScopes struct {
	Bot  []string `json:"bot,omitempty" yaml:"bot,omitempty"`
	User []string `json:"user,omitempty" yaml:"user,omitempty"`
}

// ManifestResponse is the response returned by the API for apps.manifest.x endpoints
type ManifestResponse struct {
	Errors []ManifestValidationError `json:"errors,omitempty"`
	SlackResponse
}

// ManifestValidationError is an error message returned for invalid manifests
type ManifestValidationError struct {
	Message string `json:"message"`
	Pointer string `json:"pointer"`
}

type ExportManifestResponse struct {
	Manifest Manifest `json:"manifest,omitempty"`
	SlackResponse
}

type UpdateManifestResponse struct {
	AppId              string `json:"app_id,omitempty"`
	PermissionsUpdated bool   `json:"permissions_updated,omitempty"`
	ManifestResponse
}
