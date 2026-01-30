// Discordgo - Discord bindings for Go
// Available at https://github.com/bwmarrin/discordgo

// Copyright 2015-2016 Bruce Marriner <bruce@sqls.net>.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file contains all structures for the discordgo package.  These
// may be moved about later into separate files but I find it easier to have
// them all located together.

package discordgo

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// A Session represents a connection to the Discord API.
type Session struct {
	sync.RWMutex

	// General configurable settings.

	// Authentication token for this session
	// TODO: Remove Below, Deprecated, Use Identify struct
	Token string

	MFA bool

	// Debug for printing JSON request/responses
	Debug    bool // Deprecated, will be removed.
	LogLevel int

	// Should the session reconnect the websocket on errors.
	ShouldReconnectOnError bool

	// Should voice connections reconnect on a session reconnect.
	ShouldReconnectVoiceOnSessionError bool

	// Should the session retry requests when rate limited.
	ShouldRetryOnRateLimit bool

	// Identify is sent during initial handshake with the discord gateway.
	// https://discord.com/developers/docs/topics/gateway#identify
	Identify Identify

	// TODO: Remove Below, Deprecated, Use Identify struct
	// Should the session request compressed websocket data.
	Compress bool

	// Sharding
	ShardID    int
	ShardCount int

	// Should state tracking be enabled.
	// State tracking is the best way for getting the users
	// active guilds and the members of the guilds.
	StateEnabled bool

	// Whether or not to call event handlers synchronously.
	// e.g. false = launch event handlers in their own goroutines.
	SyncEvents bool

	// Exposed but should not be modified by User.

	// Whether the Data Websocket is ready
	DataReady bool // NOTE: Maye be deprecated soon

	// Max number of REST API retries
	MaxRestRetries int

	// Status stores the current status of the websocket connection
	// this is being tested, may stay, may go away.
	status int32

	// Whether the Voice Websocket is ready
	VoiceReady bool // NOTE: Deprecated.

	// Whether the UDP Connection is ready
	UDPReady bool // NOTE: Deprecated

	// Stores a mapping of guild id's to VoiceConnections
	VoiceConnections map[string]*VoiceConnection

	// Managed state object, updated internally with events when
	// StateEnabled is true.
	State *State

	// The http client used for REST requests
	Client *http.Client

	// The dialer used for WebSocket connection
	Dialer *websocket.Dialer

	// The user agent used for REST APIs
	UserAgent string

	// Stores the last HeartbeatAck that was received (in UTC)
	LastHeartbeatAck time.Time

	// Stores the last Heartbeat sent (in UTC)
	LastHeartbeatSent time.Time

	// used to deal with rate limits
	Ratelimiter *RateLimiter

	// Event handlers
	handlersMu   sync.RWMutex
	handlers     map[string][]*eventHandlerInstance
	onceHandlers map[string][]*eventHandlerInstance

	// The websocket connection.
	wsConn *websocket.Conn

	// When nil, the session is not listening.
	listening chan interface{}

	// sequence tracks the current gateway api websocket sequence number
	sequence *int64

	// stores sessions current Discord Gateway
	gateway string

	// stores session ID of current Gateway connection
	sessionID string

	// used to make sure gateway websocket writes do not happen concurrently
	wsMutex sync.Mutex
}

// ApplicationIntegrationType dictates where application can be installed and its available interaction contexts.
type ApplicationIntegrationType uint

const (
	// ApplicationIntegrationGuildInstall indicates that app is installable to guilds.
	ApplicationIntegrationGuildInstall ApplicationIntegrationType = 0
	// ApplicationIntegrationUserInstall indicates that app is installable to users.
	ApplicationIntegrationUserInstall ApplicationIntegrationType = 1
)

// ApplicationInstallParams represents application's installation parameters
// for default in-app oauth2 authorization link.
type ApplicationInstallParams struct {
	Scopes      []string `json:"scopes"`
	Permissions int64    `json:"permissions,string"`
}

// ApplicationIntegrationTypeConfig represents application's configuration for a particular integration type.
type ApplicationIntegrationTypeConfig struct {
	OAuth2InstallParams *ApplicationInstallParams `json:"oauth2_install_params,omitempty"`
}

// Application stores values for a Discord Application
type Application struct {
	ID                     string                                                           `json:"id,omitempty"`
	Name                   string                                                           `json:"name"`
	Icon                   string                                                           `json:"icon,omitempty"`
	Description            string                                                           `json:"description,omitempty"`
	RPCOrigins             []string                                                         `json:"rpc_origins,omitempty"`
	BotPublic              bool                                                             `json:"bot_public,omitempty"`
	BotRequireCodeGrant    bool                                                             `json:"bot_require_code_grant,omitempty"`
	TermsOfServiceURL      string                                                           `json:"terms_of_service_url"`
	PrivacyProxyURL        string                                                           `json:"privacy_policy_url"`
	Owner                  *User                                                            `json:"owner"`
	Summary                string                                                           `json:"summary"`
	VerifyKey              string                                                           `json:"verify_key"`
	Team                   *Team                                                            `json:"team"`
	GuildID                string                                                           `json:"guild_id"`
	PrimarySKUID           string                                                           `json:"primary_sku_id"`
	Slug                   string                                                           `json:"slug"`
	CoverImage             string                                                           `json:"cover_image"`
	Flags                  int                                                              `json:"flags,omitempty"`
	IntegrationTypesConfig map[ApplicationIntegrationType]*ApplicationIntegrationTypeConfig `json:"integration_types,omitempty"`
}

// ApplicationRoleConnectionMetadataType represents the type of application role connection metadata.
type ApplicationRoleConnectionMetadataType int

// Application role connection metadata types.
const (
	ApplicationRoleConnectionMetadataIntegerLessThanOrEqual     ApplicationRoleConnectionMetadataType = 1
	ApplicationRoleConnectionMetadataIntegerGreaterThanOrEqual  ApplicationRoleConnectionMetadataType = 2
	ApplicationRoleConnectionMetadataIntegerEqual               ApplicationRoleConnectionMetadataType = 3
	ApplicationRoleConnectionMetadataIntegerNotEqual            ApplicationRoleConnectionMetadataType = 4
	ApplicationRoleConnectionMetadataDatetimeLessThanOrEqual    ApplicationRoleConnectionMetadataType = 5
	ApplicationRoleConnectionMetadataDatetimeGreaterThanOrEqual ApplicationRoleConnectionMetadataType = 6
	ApplicationRoleConnectionMetadataBooleanEqual               ApplicationRoleConnectionMetadataType = 7
	ApplicationRoleConnectionMetadataBooleanNotEqual            ApplicationRoleConnectionMetadataType = 8
)

// ApplicationRoleConnectionMetadata stores application role connection metadata.
type ApplicationRoleConnectionMetadata struct {
	Type                     ApplicationRoleConnectionMetadataType `json:"type"`
	Key                      string                                `json:"key"`
	Name                     string                                `json:"name"`
	NameLocalizations        map[Locale]string                     `json:"name_localizations"`
	Description              string                                `json:"description"`
	DescriptionLocalizations map[Locale]string                     `json:"description_localizations"`
}

// ApplicationRoleConnection represents the role connection that an application has attached to a user.
type ApplicationRoleConnection struct {
	PlatformName     string            `json:"platform_name"`
	PlatformUsername string            `json:"platform_username"`
	Metadata         map[string]string `json:"metadata"`
}

// UserConnection is a Connection returned from the UserConnections endpoint
type UserConnection struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Type         string         `json:"type"`
	Revoked      bool           `json:"revoked"`
	Integrations []*Integration `json:"integrations"`
}

// Integration stores integration information
type Integration struct {
	ID                string             `json:"id"`
	Name              string             `json:"name"`
	Type              string             `json:"type"`
	Enabled           bool               `json:"enabled"`
	Syncing           bool               `json:"syncing"`
	RoleID            string             `json:"role_id"`
	EnableEmoticons   bool               `json:"enable_emoticons"`
	ExpireBehavior    ExpireBehavior     `json:"expire_behavior"`
	ExpireGracePeriod int                `json:"expire_grace_period"`
	User              *User              `json:"user"`
	Account           IntegrationAccount `json:"account"`
	SyncedAt          time.Time          `json:"synced_at"`
}

// ExpireBehavior of Integration
// https://discord.com/developers/docs/resources/guild#integration-object-integration-expire-behaviors
type ExpireBehavior int

// Block of valid ExpireBehaviors
const (
	ExpireBehaviorRemoveRole ExpireBehavior = 0
	ExpireBehaviorKick       ExpireBehavior = 1
)

// IntegrationAccount is integration account information
// sent by the UserConnections endpoint
type IntegrationAccount struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// A VoiceRegion stores data for a specific voice region server.
// https://discord.com/developers/docs/resources/voice#voice-region-object
type VoiceRegion struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Optimal    bool   `json:"optimal"`
	Deprecated bool   `json:"deprecated"`
	Custom     bool   `json:"custom"`
}

// InviteTargetType indicates the type of target of an invite
// https://discord.com/developers/docs/resources/invite#invite-object-invite-target-types
type InviteTargetType uint8

// Invite target types
const (
	InviteTargetStream              InviteTargetType = 1
	InviteTargetEmbeddedApplication InviteTargetType = 2
)

// A Invite stores all data related to a specific Discord Guild or Channel invite.
type Invite struct {
	Guild             *Guild           `json:"guild"`
	Channel           *Channel         `json:"channel"`
	Inviter           *User            `json:"inviter"`
	Code              string           `json:"code"`
	CreatedAt         time.Time        `json:"created_at"`
	MaxAge            int              `json:"max_age"`
	Uses              int              `json:"uses"`
	MaxUses           int              `json:"max_uses"`
	Revoked           bool             `json:"revoked"`
	Temporary         bool             `json:"temporary"`
	Unique            bool             `json:"unique"`
	TargetUser        *User            `json:"target_user"`
	TargetType        InviteTargetType `json:"target_type"`
	TargetApplication *Application     `json:"target_application"`

	// will only be filled when using InviteWithCounts
	ApproximatePresenceCount int `json:"approximate_presence_count"`
	ApproximateMemberCount   int `json:"approximate_member_count"`

	ExpiresAt *time.Time `json:"expires_at"`
}

// ChannelType is the type of a Channel
type ChannelType int

// Block contains known ChannelType values
const (
	ChannelTypeGuildText          ChannelType = 0
	ChannelTypeDM                 ChannelType = 1
	ChannelTypeGuildVoice         ChannelType = 2
	ChannelTypeGroupDM            ChannelType = 3
	ChannelTypeGuildCategory      ChannelType = 4
	ChannelTypeGuildNews          ChannelType = 5
	ChannelTypeGuildStore         ChannelType = 6
	ChannelTypeGuildNewsThread    ChannelType = 10
	ChannelTypeGuildPublicThread  ChannelType = 11
	ChannelTypeGuildPrivateThread ChannelType = 12
	ChannelTypeGuildStageVoice    ChannelType = 13
	ChannelTypeGuildDirectory     ChannelType = 14
	ChannelTypeGuildForum         ChannelType = 15
	ChannelTypeGuildMedia         ChannelType = 16
)

// ChannelFlags represent flags of a channel/thread.
type ChannelFlags int

// Block containing known ChannelFlags values.
const (
	// ChannelFlagPinned indicates whether the thread is pinned in the forum channel.
	// NOTE: forum threads only.
	ChannelFlagPinned ChannelFlags = 1 << 1
	// ChannelFlagRequireTag indicates whether a tag is required to be specified when creating a thread.
	// NOTE: forum channels only.
	ChannelFlagRequireTag ChannelFlags = 1 << 4
)

// ForumSortOrderType represents sort order of a forum channel.
type ForumSortOrderType int

const (
	// ForumSortOrderLatestActivity sorts posts by activity.
	ForumSortOrderLatestActivity ForumSortOrderType = 0
	// ForumSortOrderCreationDate sorts posts by creation time (from most recent to oldest).
	ForumSortOrderCreationDate ForumSortOrderType = 1
)

// ForumLayout represents layout of a forum channel.
type ForumLayout int

const (
	// ForumLayoutNotSet represents no default layout.
	ForumLayoutNotSet ForumLayout = 0
	// ForumLayoutListView displays forum posts as a list.
	ForumLayoutListView ForumLayout = 1
	// ForumLayoutGalleryView displays forum posts as a collection of tiles.
	ForumLayoutGalleryView ForumLayout = 2
)

// A Channel holds all data related to an individual Discord channel.
type Channel struct {
	// The ID of the channel.
	ID string `json:"id"`

	// The ID of the guild to which the channel belongs, if it is in a guild.
	// Else, this ID is empty (e.g. DM channels).
	GuildID string `json:"guild_id"`

	// The name of the channel.
	Name string `json:"name"`

	// The topic of the channel.
	Topic string `json:"topic"`

	// The type of the channel.
	Type ChannelType `json:"type"`

	// The ID of the last message sent in the channel. This is not
	// guaranteed to be an ID of a valid message.
	LastMessageID string `json:"last_message_id"`

	// The timestamp of the last pinned message in the channel.
	// nil if the channel has no pinned messages.
	LastPinTimestamp *time.Time `json:"last_pin_timestamp"`

	// An approximate count of messages in a thread, stops counting at 50
	MessageCount int `json:"message_count"`
	// An approximate count of users in a thread, stops counting at 50
	MemberCount int `json:"member_count"`

	// Whether the channel is marked as NSFW.
	NSFW bool `json:"nsfw"`

	// Icon of the group DM channel.
	Icon string `json:"icon"`

	// The position of the channel, used for sorting in client.
	Position int `json:"position"`

	// The bitrate of the channel, if it is a voice channel.
	Bitrate int `json:"bitrate"`

	// The recipients of the channel. This is only populated in DM channels.
	Recipients []*User `json:"recipients"`

	// The messages in the channel. This is only present in state-cached channels,
	// and State.MaxMessageCount must be non-zero.
	Messages []*Message `json:"-"`

	// A list of permission overwrites present for the channel.
	PermissionOverwrites []*PermissionOverwrite `json:"permission_overwrites"`

	// The user limit of the voice channel.
	UserLimit int `json:"user_limit"`

	// The ID of the parent channel, if the channel is under a category. For threads - id of the channel thread was created in.
	ParentID string `json:"parent_id"`

	// Amount of seconds a user has to wait before sending another message or creating another thread (0-21600)
	// bots, as well as users with the permission manage_messages or manage_channel, are unaffected
	RateLimitPerUser int `json:"rate_limit_per_user"`

	// ID of the creator of the group DM or thread
	OwnerID string `json:"owner_id"`

	// ApplicationID of the DM creator Zeroed if guild channel or not a bot user
	ApplicationID string `json:"application_id"`

	// Thread-specific fields not needed by other channels
	ThreadMetadata *ThreadMetadata `json:"thread_metadata,omitempty"`
	// Thread member object for the current user, if they have joined the thread, only included on certain API endpoints
	Member *ThreadMember `json:"thread_member"`

	// All thread members. State channels only.
	Members []*ThreadMember `json:"-"`

	// Channel flags.
	Flags ChannelFlags `json:"flags"`

	// The set of tags that can be used in a forum channel.
	AvailableTags []ForumTag `json:"available_tags"`

	// The IDs of the set of tags that have been applied to a thread in a forum channel.
	AppliedTags []string `json:"applied_tags"`

	// Emoji to use as the default reaction to a forum post.
	DefaultReactionEmoji ForumDefaultReaction `json:"default_reaction_emoji"`

	// The initial RateLimitPerUser to set on newly created threads in a channel.
	// This field is copied to the thread at creation time and does not live update.
	DefaultThreadRateLimitPerUser int `json:"default_thread_rate_limit_per_user"`

	// The default sort order type used to order posts in forum channels.
	// Defaults to null, which indicates a preferred sort order hasn't been set by a channel admin.
	DefaultSortOrder *ForumSortOrderType `json:"default_sort_order"`

	// The default forum layout view used to display posts in forum channels.
	// Defaults to ForumLayoutNotSet, which indicates a layout view has not been set by a channel admin.
	DefaultForumLayout ForumLayout `json:"default_forum_layout"`
}

// Mention returns a string which mentions the channel
func (c *Channel) Mention() string {
	return fmt.Sprintf("<#%s>", c.ID)
}

// IsThread is a helper function to determine if channel is a thread or not
func (c *Channel) IsThread() bool {
	return c.Type == ChannelTypeGuildPublicThread || c.Type == ChannelTypeGuildPrivateThread || c.Type == ChannelTypeGuildNewsThread
}

// A ChannelEdit holds Channel Field data for a channel edit.
type ChannelEdit struct {
	Name                          string                 `json:"name,omitempty"`
	Topic                         string                 `json:"topic,omitempty"`
	NSFW                          *bool                  `json:"nsfw,omitempty"`
	Position                      *int                   `json:"position,omitempty"`
	Bitrate                       int                    `json:"bitrate,omitempty"`
	UserLimit                     int                    `json:"user_limit,omitempty"`
	PermissionOverwrites          []*PermissionOverwrite `json:"permission_overwrites,omitempty"`
	ParentID                      string                 `json:"parent_id,omitempty"`
	RateLimitPerUser              *int                   `json:"rate_limit_per_user,omitempty"`
	Flags                         *ChannelFlags          `json:"flags,omitempty"`
	DefaultThreadRateLimitPerUser *int                   `json:"default_thread_rate_limit_per_user,omitempty"`

	// NOTE: threads only

	Archived            *bool `json:"archived,omitempty"`
	AutoArchiveDuration int   `json:"auto_archive_duration,omitempty"`
	Locked              *bool `json:"locked,omitempty"`
	Invitable           *bool `json:"invitable,omitempty"`

	// NOTE: forum channels only

	AvailableTags        *[]ForumTag           `json:"available_tags,omitempty"`
	DefaultReactionEmoji *ForumDefaultReaction `json:"default_reaction_emoji,omitempty"`
	DefaultSortOrder     *ForumSortOrderType   `json:"default_sort_order,omitempty"` // TODO: null
	DefaultForumLayout   *ForumLayout          `json:"default_forum_layout,omitempty"`

	// NOTE: forum threads only
	AppliedTags *[]string `json:"applied_tags,omitempty"`
}

// A ChannelFollow holds data returned after following a news channel
type ChannelFollow struct {
	ChannelID string `json:"channel_id"`
	WebhookID string `json:"webhook_id"`
}

// PermissionOverwriteType represents the type of resource on which
// a permission overwrite acts.
type PermissionOverwriteType int

// The possible permission overwrite types.
const (
	PermissionOverwriteTypeRole   PermissionOverwriteType = 0
	PermissionOverwriteTypeMember PermissionOverwriteType = 1
)

// A PermissionOverwrite holds permission overwrite data for a Channel
type PermissionOverwrite struct {
	ID    string                  `json:"id"`
	Type  PermissionOverwriteType `json:"type"`
	Deny  int64                   `json:"deny,string"`
	Allow int64                   `json:"allow,string"`
}

// ThreadStart stores all parameters you can use with MessageThreadStartComplex or ThreadStartComplex
type ThreadStart struct {
	Name                string      `json:"name"`
	AutoArchiveDuration int         `json:"auto_archive_duration,omitempty"`
	Type                ChannelType `json:"type,omitempty"`
	Invitable           bool        `json:"invitable"`
	RateLimitPerUser    int         `json:"rate_limit_per_user,omitempty"`

	// NOTE: forum threads only
	AppliedTags []string `json:"applied_tags,omitempty"`
}

// ThreadMetadata contains a number of thread-specific channel fields that are not needed by other channel types.
type ThreadMetadata struct {
	// Whether the thread is archived
	Archived bool `json:"archived"`
	// Duration in minutes to automatically archive the thread after recent activity, can be set to: 60, 1440, 4320, 10080
	AutoArchiveDuration int `json:"auto_archive_duration"`
	// Timestamp when the thread's archive status was last changed, used for calculating recent activity
	ArchiveTimestamp time.Time `json:"archive_timestamp"`
	// Whether the thread is locked; when a thread is locked, only users with MANAGE_THREADS can unarchive it
	Locked bool `json:"locked"`
	// Whether non-moderators can add other non-moderators to a thread; only available on private threads
	Invitable bool `json:"invitable"`
}

// ThreadMember is used to indicate whether a user has joined a thread or not.
// NOTE: ID and UserID are empty (omitted) on the member sent within each thread in the GUILD_CREATE event.
type ThreadMember struct {
	// The id of the thread
	ID string `json:"id,omitempty"`
	// The id of the user
	UserID string `json:"user_id,omitempty"`
	// The time the current user last joined the thread
	JoinTimestamp time.Time `json:"join_timestamp"`
	// Any user-thread settings, currently only used for notifications
	Flags int `json:"flags"`
	// Additional information about the user.
	// NOTE: only present if the withMember parameter is set to true
	// when calling Session.ThreadMembers or Session.ThreadMember.
	Member *Member `json:"member,omitempty"`
}

// ThreadsList represents a list of threads alongisde with thread member objects for the current user.
type ThreadsList struct {
	Threads []*Channel      `json:"threads"`
	Members []*ThreadMember `json:"members"`
	HasMore bool            `json:"has_more"`
}

// AddedThreadMember holds information about the user who was added to the thread
type AddedThreadMember struct {
	*ThreadMember
	Member   *Member   `json:"member"`
	Presence *Presence `json:"presence"`
}

// ForumDefaultReaction specifies emoji to use as the default reaction to a forum post.
// NOTE: Exactly one of EmojiID and EmojiName must be set.
type ForumDefaultReaction struct {
	// The id of a guild's custom emoji.
	EmojiID string `json:"emoji_id,omitempty"`
	// The unicode character of the emoji.
	EmojiName string `json:"emoji_name,omitempty"`
}

// ForumTag represents a tag that is able to be applied to a thread in a forum channel.
type ForumTag struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	Moderated bool   `json:"moderated"`
	EmojiID   string `json:"emoji_id,omitempty"`
	EmojiName string `json:"emoji_name,omitempty"`
}

// Emoji struct holds data related to Emoji's
type Emoji struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Roles         []string `json:"roles"`
	User          *User    `json:"user"`
	RequireColons bool     `json:"require_colons"`
	Managed       bool     `json:"managed"`
	Animated      bool     `json:"animated"`
	Available     bool     `json:"available"`
}

// EmojiRegex is the regex used to find and identify emojis in messages
var (
	EmojiRegex = regexp.MustCompile(`<(a|):[A-Za-z0-9_~]+:[0-9]{18,20}>`)
)

// MessageFormat returns a correctly formatted Emoji for use in Message content and embeds
func (e *Emoji) MessageFormat() string {
	if e.ID != "" && e.Name != "" {
		if e.Animated {
			return "<a:" + e.APIName() + ">"
		}

		return "<:" + e.APIName() + ">"
	}

	return e.APIName()
}

// APIName returns an correctly formatted API name for use in the MessageReactions endpoints.
func (e *Emoji) APIName() string {
	if e.ID != "" && e.Name != "" {
		return e.Name + ":" + e.ID
	}
	if e.Name != "" {
		return e.Name
	}
	return e.ID
}

// EmojiParams represents parameters needed to create or update an Emoji.
type EmojiParams struct {
	// Name of the emoji
	Name string `json:"name,omitempty"`
	// A base64 encoded emoji image, has to be smaller than 256KB.
	// NOTE: can be only set on creation.
	Image string `json:"image,omitempty"`
	// Roles for which this emoji will be available.
	// NOTE: can not be used with application emoji endpoints.
	Roles []string `json:"roles,omitempty"`
}

// StickerFormat is the file format of the Sticker.
type StickerFormat int

// Defines all known Sticker types.
const (
	StickerFormatTypePNG    StickerFormat = 1
	StickerFormatTypeAPNG   StickerFormat = 2
	StickerFormatTypeLottie StickerFormat = 3
	StickerFormatTypeGIF    StickerFormat = 4
)

// StickerType is the type of sticker.
type StickerType int

// Defines Sticker types.
const (
	StickerTypeStandard StickerType = 1
	StickerTypeGuild    StickerType = 2
)

// Sticker represents a sticker object that can be sent in a Message.
type Sticker struct {
	ID          string        `json:"id"`
	PackID      string        `json:"pack_id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Tags        string        `json:"tags"`
	Type        StickerType   `json:"type"`
	FormatType  StickerFormat `json:"format_type"`
	Available   bool          `json:"available"`
	GuildID     string        `json:"guild_id"`
	User        *User         `json:"user"`
	SortValue   int           `json:"sort_value"`
}

// StickerItem represents the smallest amount of data required to render a sticker. A partial sticker object.
type StickerItem struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	FormatType StickerFormat `json:"format_type"`
}

// StickerPack represents a pack of standard stickers.
type StickerPack struct {
	ID             string     `json:"id"`
	Stickers       []*Sticker `json:"stickers"`
	Name           string     `json:"name"`
	SKUID          string     `json:"sku_id"`
	CoverStickerID string     `json:"cover_sticker_id"`
	Description    string     `json:"description"`
	BannerAssetID  string     `json:"banner_asset_id"`
}

// VerificationLevel type definition
type VerificationLevel int

// Constants for VerificationLevel levels from 0 to 4 inclusive
const (
	VerificationLevelNone     VerificationLevel = 0
	VerificationLevelLow      VerificationLevel = 1
	VerificationLevelMedium   VerificationLevel = 2
	VerificationLevelHigh     VerificationLevel = 3
	VerificationLevelVeryHigh VerificationLevel = 4
)

// ExplicitContentFilterLevel type definition
type ExplicitContentFilterLevel int

// Constants for ExplicitContentFilterLevel levels from 0 to 2 inclusive
const (
	ExplicitContentFilterDisabled            ExplicitContentFilterLevel = 0
	ExplicitContentFilterMembersWithoutRoles ExplicitContentFilterLevel = 1
	ExplicitContentFilterAllMembers          ExplicitContentFilterLevel = 2
)

// GuildNSFWLevel type definition
type GuildNSFWLevel int

// Constants for GuildNSFWLevel levels from 0 to 3 inclusive
const (
	GuildNSFWLevelDefault       GuildNSFWLevel = 0
	GuildNSFWLevelExplicit      GuildNSFWLevel = 1
	GuildNSFWLevelSafe          GuildNSFWLevel = 2
	GuildNSFWLevelAgeRestricted GuildNSFWLevel = 3
)

// MfaLevel type definition
type MfaLevel int

// Constants for MfaLevel levels from 0 to 1 inclusive
const (
	MfaLevelNone     MfaLevel = 0
	MfaLevelElevated MfaLevel = 1
)

// PremiumTier type definition
type PremiumTier int

// Constants for PremiumTier levels from 0 to 3 inclusive
const (
	PremiumTierNone PremiumTier = 0
	PremiumTier1    PremiumTier = 1
	PremiumTier2    PremiumTier = 2
	PremiumTier3    PremiumTier = 3
)

// A Guild holds all data related to a specific Discord Guild.  Guilds are also
// sometimes referred to as Servers in the Discord client.
type Guild struct {
	// The ID of the guild.
	ID string `json:"id"`

	// The name of the guild. (2–100 characters)
	Name string `json:"name"`

	// The hash of the guild's icon. Use Session.GuildIcon
	// to retrieve the icon itself.
	Icon string `json:"icon"`

	// The voice region of the guild.
	Region string `json:"region"`

	// The ID of the AFK voice channel.
	AfkChannelID string `json:"afk_channel_id"`

	// The user ID of the owner of the guild.
	OwnerID string `json:"owner_id"`

	// If we are the owner of the guild
	Owner bool `json:"owner"`

	// The time at which the current user joined the guild.
	// This field is only present in GUILD_CREATE events and websocket
	// update events, and thus is only present in state-cached guilds.
	JoinedAt time.Time `json:"joined_at"`

	// The hash of the guild's discovery splash.
	DiscoverySplash string `json:"discovery_splash"`

	// The hash of the guild's splash.
	Splash string `json:"splash"`

	// The timeout, in seconds, before a user is considered AFK in voice.
	AfkTimeout int `json:"afk_timeout"`

	// The number of members in the guild.
	// This field is only present in GUILD_CREATE events and websocket
	// update events, and thus is only present in state-cached guilds.
	MemberCount int `json:"member_count"`

	// The verification level required for the guild.
	VerificationLevel VerificationLevel `json:"verification_level"`

	// Whether the guild is considered large. This is
	// determined by a member threshold in the identify packet,
	// and is currently hard-coded at 250 members in the library.
	Large bool `json:"large"`

	// The default message notification setting for the guild.
	DefaultMessageNotifications MessageNotifications `json:"default_message_notifications"`

	// A list of roles in the guild.
	Roles []*Role `json:"roles"`

	// A list of the custom emojis present in the guild.
	Emojis []*Emoji `json:"emojis"`

	// A list of the custom stickers present in the guild.
	Stickers []*Sticker `json:"stickers"`

	// A list of the members in the guild.
	// This field is only present in GUILD_CREATE events and websocket
	// update events, and thus is only present in state-cached guilds.
	Members []*Member `json:"members"`

	// A list of partial presence objects for members in the guild.
	// This field is only present in GUILD_CREATE events and websocket
	// update events, and thus is only present in state-cached guilds.
	Presences []*Presence `json:"presences"`

	// The maximum number of presences for the guild (the default value, currently 25000, is in effect when null is returned)
	MaxPresences int `json:"max_presences"`

	// The maximum number of members for the guild
	MaxMembers int `json:"max_members"`

	// A list of channels in the guild.
	// This field is only present in GUILD_CREATE events and websocket
	// update events, and thus is only present in state-cached guilds.
	Channels []*Channel `json:"channels"`

	// A list of all active threads in the guild that current user has permission to view
	// This field is only present in GUILD_CREATE events and websocket
	// update events and thus is only present in state-cached guilds.
	Threads []*Channel `json:"threads"`

	// A list of voice states for the guild.
	// This field is only present in GUILD_CREATE events and websocket
	// update events, and thus is only present in state-cached guilds.
	VoiceStates []*VoiceState `json:"voice_states"`

	// Whether this guild is currently unavailable (most likely due to outage).
	// This field is only present in GUILD_CREATE events and websocket
	// update events, and thus is only present in state-cached guilds.
	Unavailable bool `json:"unavailable"`

	// The explicit content filter level
	ExplicitContentFilter ExplicitContentFilterLevel `json:"explicit_content_filter"`

	// The NSFW Level of the guild
	NSFWLevel GuildNSFWLevel `json:"nsfw_level"`

	// The list of enabled guild features
	Features []GuildFeature `json:"features"`

	// Required MFA level for the guild
	MfaLevel MfaLevel `json:"mfa_level"`

	// The application id of the guild if bot created.
	ApplicationID string `json:"application_id"`

	// Whether or not the Server Widget is enabled
	WidgetEnabled bool `json:"widget_enabled"`

	// The Channel ID for the Server Widget
	WidgetChannelID string `json:"widget_channel_id"`

	// The Channel ID to which system messages are sent (eg join and leave messages)
	SystemChannelID string `json:"system_channel_id"`

	// The System channel flags
	SystemChannelFlags SystemChannelFlag `json:"system_channel_flags"`

	// The ID of the rules channel ID, used for rules.
	RulesChannelID string `json:"rules_channel_id"`

	// the vanity url code for the guild
	VanityURLCode string `json:"vanity_url_code"`

	// the description for the guild
	Description string `json:"description"`

	// The hash of the guild's banner
	Banner string `json:"banner"`

	// The premium tier of the guild
	PremiumTier PremiumTier `json:"premium_tier"`

	// The total number of users currently boosting this server
	PremiumSubscriptionCount int `json:"premium_subscription_count"`

	// The preferred locale of a guild with the "PUBLIC" feature; used in server discovery and notices from Discord; defaults to "en-US"
	PreferredLocale string `json:"preferred_locale"`

	// The id of the channel where admins and moderators of guilds with the "PUBLIC" feature receive notices from Discord
	PublicUpdatesChannelID string `json:"public_updates_channel_id"`

	// The maximum amount of users in a video channel
	MaxVideoChannelUsers int `json:"max_video_channel_users"`

	// Approximate number of members in this guild, returned from the GET /guild/<id> endpoint when with_counts is true
	ApproximateMemberCount int `json:"approximate_member_count"`

	// Approximate number of non-offline members in this guild, returned from the GET /guild/<id> endpoint when with_counts is true
	ApproximatePresenceCount int `json:"approximate_presence_count"`

	// Permissions of our user
	Permissions int64 `json:"permissions,string"`

	// Stage instances in the guild
	StageInstances []*StageInstance `json:"stage_instances"`
}

// A GuildPreview holds data related to a specific public Discord Guild, even if the user is not in the guild.
type GuildPreview struct {
	// The ID of the guild.
	ID string `json:"id"`

	// The name of the guild. (2–100 characters)
	Name string `json:"name"`

	// The hash of the guild's icon. Use Session.GuildIcon
	// to retrieve the icon itself.
	Icon string `json:"icon"`

	// The hash of the guild's splash.
	Splash string `json:"splash"`

	// The hash of the guild's discovery splash.
	DiscoverySplash string `json:"discovery_splash"`

	// A list of the custom emojis present in the guild.
	Emojis []*Emoji `json:"emojis"`

	// The list of enabled guild features
	Features []string `json:"features"`

	// Approximate number of members in this guild
	// NOTE: this field is only filled when using GuildWithCounts
	ApproximateMemberCount int `json:"approximate_member_count"`

	// Approximate number of non-offline members in this guild
	// NOTE: this field is only filled when using GuildWithCounts
	ApproximatePresenceCount int `json:"approximate_presence_count"`

	// the description for the guild
	Description string `json:"description"`
}

// IconURL returns a URL to the guild's icon.
//
//	size:    The size of the desired icon image as a power of two
//	         Image size can be any power of two between 16 and 4096.
func (g *GuildPreview) IconURL(size string) string {
	return iconURL(g.Icon, EndpointGuildIcon(g.ID, g.Icon), EndpointGuildIconAnimated(g.ID, g.Icon), size)
}

// GuildScheduledEvent is a representation of a scheduled event in a guild. Only for retrieval of the data.
// https://discord.com/developers/docs/resources/guild-scheduled-event#guild-scheduled-event
type GuildScheduledEvent struct {
	// The ID of the scheduled event
	ID string `json:"id"`
	// The guild id which the scheduled event belongs to
	GuildID string `json:"guild_id"`
	// The channel id in which the scheduled event will be hosted, or null if scheduled entity type is EXTERNAL
	ChannelID string `json:"channel_id"`
	// The id of the user that created the scheduled event
	CreatorID string `json:"creator_id"`
	// The name of the scheduled event (1-100 characters)
	Name string `json:"name"`
	// The description of the scheduled event (1-1000 characters)
	Description string `json:"description"`
	// The time the scheduled event will start
	ScheduledStartTime time.Time `json:"scheduled_start_time"`
	// The time the scheduled event will end, required only when entity_type is EXTERNAL
	ScheduledEndTime *time.Time `json:"scheduled_end_time"`
	// The privacy level of the scheduled event
	PrivacyLevel GuildScheduledEventPrivacyLevel `json:"privacy_level"`
	// The status of the scheduled event
	Status GuildScheduledEventStatus `json:"status"`
	// Type of the entity where event would be hosted
	// See field requirements
	// https://discord.com/developers/docs/resources/guild-scheduled-event#guild-scheduled-event-object-field-requirements-by-entity-type
	EntityType GuildScheduledEventEntityType `json:"entity_type"`
	// The id of an entity associated with a guild scheduled event
	EntityID string `json:"entity_id"`
	// Additional metadata for the guild scheduled event
	EntityMetadata GuildScheduledEventEntityMetadata `json:"entity_metadata"`
	// The user that created the scheduled event
	Creator *User `json:"creator"`
	// The number of users subscribed to the scheduled event
	UserCount int `json:"user_count"`
	// The cover image hash of the scheduled event
	// see https://discord.com/developers/docs/reference#image-formatting for more
	// information about image formatting
	Image string `json:"image"`
}

// GuildScheduledEventParams are the parameters allowed for creating or updating a scheduled event
// https://discord.com/developers/docs/resources/guild-scheduled-event#create-guild-scheduled-event
type GuildScheduledEventParams struct {
	// The channel id in which the scheduled event will be hosted, or null if scheduled entity type is EXTERNAL
	ChannelID string `json:"channel_id,omitempty"`
	// The name of the scheduled event (1-100 characters)
	Name string `json:"name,omitempty"`
	// The description of the scheduled event (1-1000 characters)
	Description string `json:"description,omitempty"`
	// The time the scheduled event will start
	ScheduledStartTime *time.Time `json:"scheduled_start_time,omitempty"`
	// The time the scheduled event will end, required only when entity_type is EXTERNAL
	ScheduledEndTime *time.Time `json:"scheduled_end_time,omitempty"`
	// The privacy level of the scheduled event
	PrivacyLevel GuildScheduledEventPrivacyLevel `json:"privacy_level,omitempty"`
	// The status of the scheduled event
	Status GuildScheduledEventStatus `json:"status,omitempty"`
	// Type of the entity where event would be hosted
	// See field requirements
	// https://discord.com/developers/docs/resources/guild-scheduled-event#guild-scheduled-event-object-field-requirements-by-entity-type
	EntityType GuildScheduledEventEntityType `json:"entity_type,omitempty"`
	// Additional metadata for the guild scheduled event
	EntityMetadata *GuildScheduledEventEntityMetadata `json:"entity_metadata,omitempty"`
	// The cover image hash of the scheduled event
	// see https://discord.com/developers/docs/reference#image-formatting for more
	// information about image formatting
	Image string `json:"image,omitempty"`
}

// MarshalJSON is a helper function to marshal GuildScheduledEventParams
func (p GuildScheduledEventParams) MarshalJSON() ([]byte, error) {
	type guildScheduledEventParams GuildScheduledEventParams

	if p.EntityType == GuildScheduledEventEntityTypeExternal && p.ChannelID == "" {
		return Marshal(struct {
			guildScheduledEventParams
			ChannelID json.RawMessage `json:"channel_id"`
		}{
			guildScheduledEventParams: guildScheduledEventParams(p),
			ChannelID:                 json.RawMessage("null"),
		})
	}

	return Marshal(guildScheduledEventParams(p))
}

// GuildScheduledEventEntityMetadata holds additional metadata for guild scheduled event.
type GuildScheduledEventEntityMetadata struct {
	// location of the event (1-100 characters)
	// required for events with 'entity_type': EXTERNAL
	Location string `json:"location"`
}

// GuildScheduledEventPrivacyLevel is the privacy level of a scheduled event.
// https://discord.com/developers/docs/resources/guild-scheduled-event#guild-scheduled-event-object-guild-scheduled-event-privacy-level
type GuildScheduledEventPrivacyLevel int

const (
	// GuildScheduledEventPrivacyLevelGuildOnly makes the scheduled
	// event is only accessible to guild members
	GuildScheduledEventPrivacyLevelGuildOnly GuildScheduledEventPrivacyLevel = 2
)

// GuildScheduledEventStatus is the status of a scheduled event
// Valid Guild Scheduled Event Status Transitions :
// SCHEDULED --> ACTIVE --> COMPLETED
// SCHEDULED --> CANCELED
// https://discord.com/developers/docs/resources/guild-scheduled-event#guild-scheduled-event-object-guild-scheduled-event-status
type GuildScheduledEventStatus int

const (
	// GuildScheduledEventStatusScheduled represents the current event is in scheduled state
	GuildScheduledEventStatusScheduled GuildScheduledEventStatus = 1
	// GuildScheduledEventStatusActive represents the current event is in active state
	GuildScheduledEventStatusActive GuildScheduledEventStatus = 2
	// GuildScheduledEventStatusCompleted represents the current event is in completed state
	GuildScheduledEventStatusCompleted GuildScheduledEventStatus = 3
	// GuildScheduledEventStatusCanceled represents the current event is in canceled state
	GuildScheduledEventStatusCanceled GuildScheduledEventStatus = 4
)

// GuildScheduledEventEntityType is the type of entity associated with a guild scheduled event.
// https://discord.com/developers/docs/resources/guild-scheduled-event#guild-scheduled-event-object-guild-scheduled-event-entity-types
type GuildScheduledEventEntityType int

const (
	// GuildScheduledEventEntityTypeStageInstance represents a stage channel
	GuildScheduledEventEntityTypeStageInstance GuildScheduledEventEntityType = 1
	// GuildScheduledEventEntityTypeVoice represents a voice channel
	GuildScheduledEventEntityTypeVoice GuildScheduledEventEntityType = 2
	// GuildScheduledEventEntityTypeExternal represents an external event
	GuildScheduledEventEntityTypeExternal GuildScheduledEventEntityType = 3
)

// GuildScheduledEventUser is a user subscribed to a scheduled event.
// https://discord.com/developers/docs/resources/guild-scheduled-event#guild-scheduled-event-user-object
type GuildScheduledEventUser struct {
	GuildScheduledEventID string  `json:"guild_scheduled_event_id"`
	User                  *User   `json:"user"`
	Member                *Member `json:"member"`
}

// GuildOnboardingMode defines the criteria used to satisfy constraints that are required for enabling onboarding.
// https://discord.com/developers/docs/resources/guild#guild-onboarding-object-onboarding-mode
type GuildOnboardingMode int

// Block containing known GuildOnboardingMode values.
const (
	// GuildOnboardingModeDefault counts default channels towards constraints.
	GuildOnboardingModeDefault GuildOnboardingMode = 0
	// GuildOnboardingModeAdvanced counts default channels and questions towards constraints.
	GuildOnboardingModeAdvanced GuildOnboardingMode = 1
)

// GuildOnboarding represents the onboarding flow for a guild.
// https://discord.com/developers/docs/resources/guild#guild-onboarding-object
type GuildOnboarding struct {
	// ID of the guild this onboarding flow is part of.
	GuildID string `json:"guild_id,omitempty"`

	// Prompts shown during onboarding and in the customize community (Channels & Roles) tab.
	Prompts *[]GuildOnboardingPrompt `json:"prompts,omitempty"`

	// Channel IDs that members get opted into automatically.
	DefaultChannelIDs []string `json:"default_channel_ids,omitempty"`

	// Whether onboarding is enabled in the guild.
	Enabled *bool `json:"enabled,omitempty"`

	// Mode of onboarding.
	Mode *GuildOnboardingMode `json:"mode,omitempty"`
}

// GuildOnboardingPromptType is the type of an onboarding prompt.
// https://discord.com/developers/docs/resources/guild#guild-onboarding-object-prompt-types
type GuildOnboardingPromptType int

// Block containing known GuildOnboardingPromptType values.
const (
	GuildOnboardingPromptTypeMultipleChoice GuildOnboardingPromptType = 0
	GuildOnboardingPromptTypeDropdown       GuildOnboardingPromptType = 1
)

// GuildOnboardingPrompt is a prompt shown during onboarding and in the customize community (Channels & Roles) tab.
// https://discord.com/developers/docs/resources/guild#guild-onboarding-object-onboarding-prompt-structure
type GuildOnboardingPrompt struct {
	// ID of the prompt.
	// NOTE: always requires to be a valid snowflake (e.g. "0"), see
	// https://github.com/discord/discord-api-docs/issues/6320 for more information.
	ID string `json:"id,omitempty"`

	// Type of the prompt.
	Type GuildOnboardingPromptType `json:"type"`

	// Options available within the prompt.
	Options []GuildOnboardingPromptOption `json:"options"`

	// Title of the prompt.
	Title string `json:"title"`

	// Indicates whether users are limited to selecting one option for the prompt.
	SingleSelect bool `json:"single_select"`

	// Indicates whether the prompt is required before a user completes the onboarding flow.
	Required bool `json:"required"`

	// Indicates whether the prompt is present in the onboarding flow.
	// If false, the prompt will only appear in the customize community (Channels & Roles) tab.
	InOnboarding bool `json:"in_onboarding"`
}

// GuildOnboardingPromptOption is an option available within an onboarding prompt.
// https://discord.com/developers/docs/resources/guild#guild-onboarding-object-prompt-option-structure
type GuildOnboardingPromptOption struct {
	// ID of the prompt option.
	ID string `json:"id,omitempty"`

	// IDs for channels a member is added to when the option is selected.
	ChannelIDs []string `json:"channel_ids"`

	// IDs for roles assigned to a member when the option is selected.
	RoleIDs []string `json:"role_ids"`

	// Emoji of the option.
	// NOTE: when creating or updating a prompt option
	// EmojiID, EmojiName and EmojiAnimated should be used instead.
	Emoji *Emoji `json:"emoji,omitempty"`

	// Title of the option.
	Title string `json:"title"`

	// Description of the option.
	Description string `json:"description"`

	// ID of the option's emoji.
	// NOTE: only used when creating or updating a prompt option.
	EmojiID string `json:"emoji_id,omitempty"`
	// Name of the option's emoji.
	// NOTE: only used when creating or updating a prompt option.
	EmojiName string `json:"emoji_name,omitempty"`
	// Whether the option's emoji is animated.
	// NOTE: only used when creating or updating a prompt option.
	EmojiAnimated *bool `json:"emoji_animated,omitempty"`
}

// A GuildTemplate represents a replicable template for guild creation
type GuildTemplate struct {
	// The unique code for the guild template
	Code string `json:"code"`

	// The name of the template
	Name string `json:"name,omitempty"`

	// The description for the template
	Description *string `json:"description,omitempty"`

	// The number of times this template has been used
	UsageCount int `json:"usage_count"`

	// The ID of the user who created the template
	CreatorID string `json:"creator_id"`

	// The user who created the template
	Creator *User `json:"creator"`

	// The timestamp of when the template was created
	CreatedAt time.Time `json:"created_at"`

	// The timestamp of when the template was last synced
	UpdatedAt time.Time `json:"updated_at"`

	// The ID of the guild the template was based on
	SourceGuildID string `json:"source_guild_id"`

	// The guild 'snapshot' this template contains
	SerializedSourceGuild *Guild `json:"serialized_source_guild"`

	// Whether the template has unsynced changes
	IsDirty bool `json:"is_dirty"`
}

// GuildTemplateParams stores the data needed to create or update a GuildTemplate.
type GuildTemplateParams struct {
	// The name of the template (1-100 characters)
	Name string `json:"name,omitempty"`
	// The description of the template (0-120 characters)
	Description string `json:"description,omitempty"`
}

// MessageNotifications is the notification level for a guild
// https://discord.com/developers/docs/resources/guild#guild-object-default-message-notification-level
type MessageNotifications int

// Block containing known MessageNotifications values
const (
	MessageNotificationsAllMessages  MessageNotifications = 0
	MessageNotificationsOnlyMentions MessageNotifications = 1
)

// SystemChannelFlag is the type of flags in the system channel (see SystemChannelFlag* consts)
// https://discord.com/developers/docs/resources/guild#guild-object-system-channel-flags
type SystemChannelFlag int

// Block containing known SystemChannelFlag values
const (
	SystemChannelFlagsSuppressJoinNotifications          SystemChannelFlag = 1 << 0
	SystemChannelFlagsSuppressPremium                    SystemChannelFlag = 1 << 1
	SystemChannelFlagsSuppressGuildReminderNotifications SystemChannelFlag = 1 << 2
	SystemChannelFlagsSuppressJoinNotificationReplies    SystemChannelFlag = 1 << 3
)

// IconURL returns a URL to the guild's icon.
//
//	size:    The size of the desired icon image as a power of two
//	         Image size can be any power of two between 16 and 4096.
func (g *Guild) IconURL(size string) string {
	return iconURL(g.Icon, EndpointGuildIcon(g.ID, g.Icon), EndpointGuildIconAnimated(g.ID, g.Icon), size)
}

// BannerURL returns a URL to the guild's banner.
//
//	size:    The size of the desired banner image as a power of two
//	         Image size can be any power of two between 16 and 4096.
func (g *Guild) BannerURL(size string) string {
	return bannerURL(g.Banner, EndpointGuildBanner(g.ID, g.Banner), EndpointGuildBannerAnimated(g.ID, g.Banner), size)
}

// A UserGuild holds a brief version of a Guild
type UserGuild struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Icon        string         `json:"icon"`
	Owner       bool           `json:"owner"`
	Permissions int64          `json:"permissions,string"`
	Features    []GuildFeature `json:"features"`

	// Approximate number of members in this guild.
	// NOTE: this field is only filled when withCounts is true.
	ApproximateMemberCount int `json:"approximate_member_count"`

	// Approximate number of non-offline members in this guild.
	// NOTE: this field is only filled when withCounts is true.
	ApproximatePresenceCount int `json:"approximate_presence_count"`
}

// GuildFeature indicates the presence of a feature in a guild
type GuildFeature string

// Constants for GuildFeature
const (
	GuildFeatureAnimatedBanner                        GuildFeature = "ANIMATED_BANNER"
	GuildFeatureAnimatedIcon                          GuildFeature = "ANIMATED_ICON"
	GuildFeatureApplicationCommandPermissionV2        GuildFeature = "APPLICATION_COMMAND_PERMISSIONS_V2"
	GuildFeatureAutoModeration                        GuildFeature = "AUTO_MODERATION"
	GuildFeatureBanner                                GuildFeature = "BANNER"
	GuildFeatureCommunity                             GuildFeature = "COMMUNITY"
	GuildFeatureCreatorMonetizableProvisional         GuildFeature = "CREATOR_MONETIZABLE_PROVISIONAL"
	GuildFeatureCreatorStorePage                      GuildFeature = "CREATOR_STORE_PAGE"
	GuildFeatureDeveloperSupportServer                GuildFeature = "DEVELOPER_SUPPORT_SERVER"
	GuildFeatureDiscoverable                          GuildFeature = "DISCOVERABLE"
	GuildFeatureFeaturable                            GuildFeature = "FEATURABLE"
	GuildFeatureInvitesDisabled                       GuildFeature = "INVITES_DISABLED"
	GuildFeatureInviteSplash                          GuildFeature = "INVITE_SPLASH"
	GuildFeatureMemberVerificationGateEnabled         GuildFeature = "MEMBER_VERIFICATION_GATE_ENABLED"
	GuildFeatureMoreSoundboard                        GuildFeature = "MORE_SOUNDBOARD"
	GuildFeatureMoreStickers                          GuildFeature = "MORE_STICKERS"
	GuildFeatureNews                                  GuildFeature = "NEWS"
	GuildFeaturePartnered                             GuildFeature = "PARTNERED"
	GuildFeaturePreviewEnabled                        GuildFeature = "PREVIEW_ENABLED"
	GuildFeatureRaidAlertsDisabled                    GuildFeature = "RAID_ALERTS_DISABLED"
	GuildFeatureRoleIcons                             GuildFeature = "ROLE_ICONS"
	GuildFeatureRoleSubscriptionsAvailableForPurchase GuildFeature = "ROLE_SUBSCRIPTIONS_AVAILABLE_FOR_PURCHASE"
	GuildFeatureRoleSubscriptionsEnabled              GuildFeature = "ROLE_SUBSCRIPTIONS_ENABLED"
	GuildFeatureSoundboard                            GuildFeature = "SOUNDBOARD"
	GuildFeatureTicketedEventsEnabled                 GuildFeature = "TICKETED_EVENTS_ENABLED"
	GuildFeatureVanityURL                             GuildFeature = "VANITY_URL"
	GuildFeatureVerified                              GuildFeature = "VERIFIED"
	GuildFeatureVipRegions                            GuildFeature = "VIP_REGIONS"
	GuildFeatureWelcomeScreenEnabled                  GuildFeature = "WELCOME_SCREEN_ENABLED"
)

// A GuildParams stores all the data needed to update discord guild settings
type GuildParams struct {
	Name                        string             `json:"name,omitempty"`
	Region                      string             `json:"region,omitempty"`
	VerificationLevel           *VerificationLevel `json:"verification_level,omitempty"`
	DefaultMessageNotifications int                `json:"default_message_notifications,omitempty"` // TODO: Separate type?
	ExplicitContentFilter       int                `json:"explicit_content_filter,omitempty"`
	AfkChannelID                string             `json:"afk_channel_id,omitempty"`
	AfkTimeout                  int                `json:"afk_timeout,omitempty"`
	Icon                        string             `json:"icon,omitempty"`
	OwnerID                     string             `json:"owner_id,omitempty"`
	Splash                      string             `json:"splash,omitempty"`
	DiscoverySplash             string             `json:"discovery_splash,omitempty"`
	Banner                      string             `json:"banner,omitempty"`
	SystemChannelID             string             `json:"system_channel_id,omitempty"`
	SystemChannelFlags          SystemChannelFlag  `json:"system_channel_flags,omitempty"`
	RulesChannelID              string             `json:"rules_channel_id,omitempty"`
	PublicUpdatesChannelID      string             `json:"public_updates_channel_id,omitempty"`
	PreferredLocale             Locale             `json:"preferred_locale,omitempty"`
	Features                    []GuildFeature     `json:"features,omitempty"`
	Description                 string             `json:"description,omitempty"`
	PremiumProgressBarEnabled   *bool              `json:"premium_progress_bar_enabled,omitempty"`
}

// A Role stores information about Discord guild member roles.
type Role struct {
	// The ID of the role.
	ID string `json:"id"`

	// The name of the role.
	Name string `json:"name"`

	// Whether this role is managed by an integration, and
	// thus cannot be manually added to, or taken from, members.
	Managed bool `json:"managed"`

	// Whether this role is mentionable.
	Mentionable bool `json:"mentionable"`

	// Whether this role is hoisted (shows up separately in member list).
	Hoist bool `json:"hoist"`

	// The hex color of this role.
	Color int `json:"color"`

	// The position of this role in the guild's role hierarchy.
	Position int `json:"position"`

	// The permissions of the role on the guild (doesn't include channel overrides).
	// This is a combination of bit masks; the presence of a certain permission can
	// be checked by performing a bitwise AND between this int and the permission.
	Permissions int64 `json:"permissions,string"`

	// The hash of the role icon. Use Role.IconURL to retrieve the icon's URL.
	Icon string `json:"icon"`

	// The emoji assigned to this role.
	UnicodeEmoji string `json:"unicode_emoji"`

	// The flags of the role, which describe its extra features.
	// This is a combination of bit masks; the presence of a certain flag can
	// be checked by performing a bitwise AND between this int and the flag.
	Flags RoleFlags `json:"flags"`
}

// RoleFlags represent the flags of a Role.
// https://discord.com/developers/docs/topics/permissions#role-object-role-flags
type RoleFlags int

// Block containing known RoleFlags values.
const (
	// RoleFlagInPrompt indicates whether the Role is selectable by members in an onboarding prompt.
	RoleFlagInPrompt RoleFlags = 1 << 0
)

// Mention returns a string which mentions the role
func (r *Role) Mention() string {
	return fmt.Sprintf("<@&%s>", r.ID)
}

// IconURL returns the URL of the role's icon.
//
//	size:    The size of the desired role icon as a power of two
//	         Image size can be any power of two between 16 and 4096.
func (r *Role) IconURL(size string) string {
	if r.Icon == "" {
		return ""
	}

	URL := EndpointRoleIcon(r.ID, r.Icon)

	if size != "" {
		return URL + "?size=" + size
	}
	return URL
}

// RoleParams represents the parameters needed to create or update a Role
type RoleParams struct {
	// The role's name
	Name string `json:"name,omitempty"`
	// The color the role should have (as a decimal, not hex)
	Color *int `json:"color,omitempty"`
	// Whether to display the role's users separately
	Hoist *bool `json:"hoist,omitempty"`
	// The overall permissions number of the role
	Permissions *int64 `json:"permissions,omitempty,string"`
	// Whether this role is mentionable
	Mentionable *bool `json:"mentionable,omitempty"`
	// The role's unicode emoji.
	// NOTE: can only be set if the guild has the ROLE_ICONS feature.
	UnicodeEmoji *string `json:"unicode_emoji,omitempty"`
	// The role's icon image encoded in base64.
	// NOTE: can only be set if the guild has the ROLE_ICONS feature.
	Icon *string `json:"icon,omitempty"`
}

// Roles are a collection of Role
type Roles []*Role

func (r Roles) Len() int {
	return len(r)
}

func (r Roles) Less(i, j int) bool {
	return r[i].Position > r[j].Position
}

func (r Roles) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

// A VoiceState stores the voice states of Guilds
type VoiceState struct {
	GuildID                 string     `json:"guild_id"`
	ChannelID               string     `json:"channel_id"`
	UserID                  string     `json:"user_id"`
	Member                  *Member    `json:"member"`
	SessionID               string     `json:"session_id"`
	Deaf                    bool       `json:"deaf"`
	Mute                    bool       `json:"mute"`
	SelfDeaf                bool       `json:"self_deaf"`
	SelfMute                bool       `json:"self_mute"`
	SelfStream              bool       `json:"self_stream"`
	SelfVideo               bool       `json:"self_video"`
	Suppress                bool       `json:"suppress"`
	RequestToSpeakTimestamp *time.Time `json:"request_to_speak_timestamp"`
}

// A Presence stores the online, offline, or idle and game status of Guild members.
type Presence struct {
	User         *User        `json:"user"`
	Status       Status       `json:"status"`
	Activities   []*Activity  `json:"activities"`
	Since        *int         `json:"since"`
	ClientStatus ClientStatus `json:"client_status"`
}

// A TimeStamps struct contains start and end times used in the rich presence "playing .." Game
type TimeStamps struct {
	EndTimestamp   int64 `json:"end,omitempty"`
	StartTimestamp int64 `json:"start,omitempty"`
}

// UnmarshalJSON unmarshals JSON into TimeStamps struct
func (t *TimeStamps) UnmarshalJSON(b []byte) error {
	temp := struct {
		End   float64 `json:"end,omitempty"`
		Start float64 `json:"start,omitempty"`
	}{}
	err := Unmarshal(b, &temp)
	if err != nil {
		return err
	}
	t.EndTimestamp = int64(temp.End)
	t.StartTimestamp = int64(temp.Start)
	return nil
}

// An Assets struct contains assets and labels used in the rich presence "playing .." Game
type Assets struct {
	LargeImageID string `json:"large_image,omitempty"`
	SmallImageID string `json:"small_image,omitempty"`
	LargeText    string `json:"large_text,omitempty"`
	SmallText    string `json:"small_text,omitempty"`
}

// MemberFlags represent flags of a guild member.
// https://discord.com/developers/docs/resources/guild#guild-member-object-guild-member-flags
type MemberFlags int

// Block containing known MemberFlags values.
const (
	// MemberFlagDidRejoin indicates whether the Member has left and rejoined the guild.
	MemberFlagDidRejoin MemberFlags = 1 << 0
	// MemberFlagCompletedOnboarding indicates whether the Member has completed onboarding.
	MemberFlagCompletedOnboarding MemberFlags = 1 << 1
	// MemberFlagBypassesVerification indicates whether the Member is exempt from guild verification requirements.
	MemberFlagBypassesVerification MemberFlags = 1 << 2
	// MemberFlagStartedOnboarding indicates whether the Member has started onboarding.
	MemberFlagStartedOnboarding MemberFlags = 1 << 3
)

// A Member stores user information for Guild members. A guild
// member represents a certain user's presence in a guild.
type Member struct {
	// The guild ID on which the member exists.
	GuildID string `json:"guild_id"`

	// The time at which the member joined the guild.
	JoinedAt time.Time `json:"joined_at"`

	// The nickname of the member, if they have one.
	Nick string `json:"nick"`

	// Whether the member is deafened at a guild level.
	Deaf bool `json:"deaf"`

	// Whether the member is muted at a guild level.
	Mute bool `json:"mute"`

	// The hash of the avatar for the guild member, if any.
	Avatar string `json:"avatar"`

	// The hash of the banner for the guild member, if any.
	Banner string `json:"banner"`

	// The underlying user on which the member is based.
	User *User `json:"user"`

	// A list of IDs of the roles which are possessed by the member.
	Roles []string `json:"roles"`

	// When the user used their Nitro boost on the server
	PremiumSince *time.Time `json:"premium_since"`

	// The flags of this member. This is a combination of bit masks; the presence of a certain
	// flag can be checked by performing a bitwise AND between this int and the flag.
	Flags MemberFlags `json:"flags"`

	// Is true while the member hasn't accepted the membership screen.
	Pending bool `json:"pending"`

	// Total permissions of the member in the channel, including overrides, returned when in the interaction object.
	Permissions int64 `json:"permissions,string"`

	// The time at which the member's timeout will expire.
	// Time in the past or nil if the user is not timed out.
	CommunicationDisabledUntil *time.Time `json:"communication_disabled_until"`
}

// Mention creates a member mention
func (m *Member) Mention() string {
	return "<@!" + m.User.ID + ">"
}

// AvatarURL returns the URL of the member's avatar
//
//	size:    The size of the user's avatar as a power of two
//	         if size is an empty string, no size parameter will
//	         be added to the URL.
func (m *Member) AvatarURL(size string) string {
	if m.Avatar == "" {
		return m.User.AvatarURL(size)
	}
	// The default/empty avatar case should be handled by the above condition
	return avatarURL(m.Avatar, "", EndpointGuildMemberAvatar(m.GuildID, m.User.ID, m.Avatar),
		EndpointGuildMemberAvatarAnimated(m.GuildID, m.User.ID, m.Avatar), size)

}

// BannerURL returns the URL of the member's banner image.
//
//	size:    The size of the desired banner image as a power of two
//	         Image size can be any power of two between 16 and 4096.
func (m *Member) BannerURL(size string) string {
	if m.Banner == "" {
		return m.User.BannerURL(size)
	}
	return bannerURL(
		m.Banner,
		EndpointGuildMemberBanner(m.GuildID, m.User.ID, m.Banner),
		EndpointGuildMemberBannerAnimated(m.GuildID, m.User.ID, m.Banner),
		size,
	)
}

// DisplayName returns the member's guild nickname if they have one,
// otherwise it returns their discord display name.
func (m *Member) DisplayName() string {
	if m.Nick != "" {
		return m.Nick
	}
	return m.User.DisplayName()
}

// ClientStatus stores the online, offline, idle, or dnd status of each device of a Guild member.
type ClientStatus struct {
	Desktop Status `json:"desktop"`
	Mobile  Status `json:"mobile"`
	Web     Status `json:"web"`
}

// Status type definition
type Status string

// Constants for Status with the different current available status
const (
	StatusOnline       Status = "online"
	StatusIdle         Status = "idle"
	StatusDoNotDisturb Status = "dnd"
	StatusInvisible    Status = "invisible"
	StatusOffline      Status = "offline"
)

// A TooManyRequests struct holds information received from Discord
// when receiving a HTTP 429 response.
type TooManyRequests struct {
	Bucket     string        `json:"bucket"`
	Message    string        `json:"message"`
	RetryAfter time.Duration `json:"retry_after"`
}

// UnmarshalJSON helps support translation of a milliseconds-based float
// into a time.Duration on TooManyRequests.
func (t *TooManyRequests) UnmarshalJSON(b []byte) error {
	u := struct {
		Bucket     string  `json:"bucket"`
		Message    string  `json:"message"`
		RetryAfter float64 `json:"retry_after"`
	}{}
	err := Unmarshal(b, &u)
	if err != nil {
		return err
	}

	t.Bucket = u.Bucket
	t.Message = u.Message
	whole, frac := math.Modf(u.RetryAfter)
	t.RetryAfter = time.Duration(whole)*time.Second + time.Duration(frac*1000)*time.Millisecond
	return nil
}

// A ReadState stores data on the read state of channels.
type ReadState struct {
	MentionCount  int    `json:"mention_count"`
	LastMessageID string `json:"last_message_id"`
	ID            string `json:"id"`
}

// A GuildRole stores data for guild roles.
type GuildRole struct {
	Role    *Role  `json:"role"`
	GuildID string `json:"guild_id"`
}

// A GuildBan stores data for a guild ban.
type GuildBan struct {
	Reason string `json:"reason"`
	User   *User  `json:"user"`
}

// AutoModerationRule stores data for an auto moderation rule.
type AutoModerationRule struct {
	ID              string                         `json:"id,omitempty"`
	GuildID         string                         `json:"guild_id,omitempty"`
	Name            string                         `json:"name,omitempty"`
	CreatorID       string                         `json:"creator_id,omitempty"`
	EventType       AutoModerationRuleEventType    `json:"event_type,omitempty"`
	TriggerType     AutoModerationRuleTriggerType  `json:"trigger_type,omitempty"`
	TriggerMetadata *AutoModerationTriggerMetadata `json:"trigger_metadata,omitempty"`
	Actions         []AutoModerationAction         `json:"actions,omitempty"`
	Enabled         *bool                          `json:"enabled,omitempty"`
	ExemptRoles     *[]string                      `json:"exempt_roles,omitempty"`
	ExemptChannels  *[]string                      `json:"exempt_channels,omitempty"`
}

// AutoModerationRuleEventType indicates in what event context a rule should be checked.
type AutoModerationRuleEventType int

// Auto moderation rule event types.
const (
	// AutoModerationEventMessageSend is checked when a member sends or edits a message in the guild
	AutoModerationEventMessageSend AutoModerationRuleEventType = 1
)

// AutoModerationRuleTriggerType represents the type of content which can trigger the rule.
type AutoModerationRuleTriggerType int

// Auto moderation rule trigger types.
const (
	AutoModerationEventTriggerKeyword       AutoModerationRuleTriggerType = 1
	AutoModerationEventTriggerHarmfulLink   AutoModerationRuleTriggerType = 2
	AutoModerationEventTriggerSpam          AutoModerationRuleTriggerType = 3
	AutoModerationEventTriggerKeywordPreset AutoModerationRuleTriggerType = 4
)

// AutoModerationKeywordPreset represents an internally pre-defined wordset.
type AutoModerationKeywordPreset uint

// Auto moderation keyword presets.
const (
	AutoModerationKeywordPresetProfanity     AutoModerationKeywordPreset = 1
	AutoModerationKeywordPresetSexualContent AutoModerationKeywordPreset = 2
	AutoModerationKeywordPresetSlurs         AutoModerationKeywordPreset = 3
)

// AutoModerationTriggerMetadata represents additional metadata used to determine whether rule should be triggered.
type AutoModerationTriggerMetadata struct {
	// Substrings which will be searched for in content.
	// NOTE: should be only used with keyword trigger type.
	KeywordFilter []string `json:"keyword_filter,omitempty"`
	// Regular expression patterns which will be matched against content (maximum of 10).
	// NOTE: should be only used with keyword trigger type.
	RegexPatterns []string `json:"regex_patterns,omitempty"`

	// Internally pre-defined wordsets which will be searched for in content.
	// NOTE: should be only used with keyword preset trigger type.
	Presets []AutoModerationKeywordPreset `json:"presets,omitempty"`

	// Substrings which should not trigger the rule.
	// NOTE: should be only used with keyword or keyword preset trigger type.
	AllowList *[]string `json:"allow_list,omitempty"`

	// Total number of unique role and user mentions allowed per message.
	// NOTE: should be only used with mention spam trigger type.
	MentionTotalLimit int `json:"mention_total_limit,omitempty"`
}

// AutoModerationActionType represents an action which will execute whenever a rule is triggered.
type AutoModerationActionType int

// Auto moderation actions types.
const (
	AutoModerationRuleActionBlockMessage     AutoModerationActionType = 1
	AutoModerationRuleActionSendAlertMessage AutoModerationActionType = 2
	AutoModerationRuleActionTimeout          AutoModerationActionType = 3
)

// AutoModerationActionMetadata represents additional metadata needed during execution for a specific action type.
type AutoModerationActionMetadata struct {
	// Channel to which user content should be logged.
	// NOTE: should be only used with send alert message action type.
	ChannelID string `json:"channel_id,omitempty"`

	// Timeout duration in seconds (maximum of 2419200 - 4 weeks).
	// NOTE: should be only used with timeout action type.
	Duration int `json:"duration_seconds,omitempty"`

	// Additional explanation that will be shown to members whenever their message is blocked (maximum of 150 characters).
	// NOTE: should be only used with block message action type.
	CustomMessage string `json:"custom_message,omitempty"`
}

// AutoModerationAction stores data for an auto moderation action.
type AutoModerationAction struct {
	Type     AutoModerationActionType      `json:"type"`
	Metadata *AutoModerationActionMetadata `json:"metadata,omitempty"`
}

// A GuildEmbed stores data for a guild embed.
type GuildEmbed struct {
	Enabled   *bool  `json:"enabled,omitempty"`
	ChannelID string `json:"channel_id,omitempty"`
}

// A GuildAuditLog stores data for a guild audit log.
// https://discord.com/developers/docs/resources/audit-log#audit-log-object-audit-log-structure
type GuildAuditLog struct {
	Webhooks        []*Webhook       `json:"webhooks,omitempty"`
	Users           []*User          `json:"users,omitempty"`
	AuditLogEntries []*AuditLogEntry `json:"audit_log_entries"`
	Integrations    []*Integration   `json:"integrations"`
}

// AuditLogEntry for a GuildAuditLog
// https://discord.com/developers/docs/resources/audit-log#audit-log-entry-object-audit-log-entry-structure
type AuditLogEntry struct {
	TargetID   string            `json:"target_id"`
	Changes    []*AuditLogChange `json:"changes"`
	UserID     string            `json:"user_id"`
	ID         string            `json:"id"`
	ActionType *AuditLogAction   `json:"action_type"`
	Options    *AuditLogOptions  `json:"options"`
	Reason     string            `json:"reason"`
}

// AuditLogChange for an AuditLogEntry
type AuditLogChange struct {
	NewValue interface{}        `json:"new_value"`
	OldValue interface{}        `json:"old_value"`
	Key      *AuditLogChangeKey `json:"key"`
}

// AuditLogChangeKey value for AuditLogChange
// https://discord.com/developers/docs/resources/audit-log#audit-log-change-object-audit-log-change-key
type AuditLogChangeKey string

// Block of valid AuditLogChangeKey
const (
	// AuditLogChangeKeyAfkChannelID is sent when afk channel changed (snowflake) - guild
	AuditLogChangeKeyAfkChannelID AuditLogChangeKey = "afk_channel_id"
	// AuditLogChangeKeyAfkTimeout is sent when afk timeout duration changed (int) - guild
	AuditLogChangeKeyAfkTimeout AuditLogChangeKey = "afk_timeout"
	// AuditLogChangeKeyAllow is sent when a permission on a text or voice channel was allowed for a role (string) - role
	AuditLogChangeKeyAllow AuditLogChangeKey = "allow"
	// AudirChangeKeyApplicationID is sent when application id of the added or removed webhook or bot (snowflake) - channel
	AuditLogChangeKeyApplicationID AuditLogChangeKey = "application_id"
	// AuditLogChangeKeyArchived is sent when thread was archived/unarchived (bool) - thread
	AuditLogChangeKeyArchived AuditLogChangeKey = "archived"
	// AuditLogChangeKeyAsset is sent when asset is changed (string) - sticker
	AuditLogChangeKeyAsset AuditLogChangeKey = "asset"
	// AuditLogChangeKeyAutoArchiveDuration is sent when auto archive duration changed (int) - thread
	AuditLogChangeKeyAutoArchiveDuration AuditLogChangeKey = "auto_archive_duration"
	// AuditLogChangeKeyAvailable is sent when availability of sticker changed (bool) - sticker
	AuditLogChangeKeyAvailable AuditLogChangeKey = "available"
	// AuditLogChangeKeyAvatarHash is sent when user avatar changed (string) - user
	AuditLogChangeKeyAvatarHash AuditLogChangeKey = "avatar_hash"
	// AuditLogChangeKeyBannerHash is sent when guild banner changed (string) - guild
	AuditLogChangeKeyBannerHash AuditLogChangeKey = "banner_hash"
	// AuditLogChangeKeyBitrate is sent when voice channel bitrate changed (int) - channel
	AuditLogChangeKeyBitrate AuditLogChangeKey = "bitrate"
	// AuditLogChangeKeyChannelID is sent when channel for invite code or guild scheduled event changed (snowflake) - invite or guild scheduled event
	AuditLogChangeKeyChannelID AuditLogChangeKey = "channel_id"
	// AuditLogChangeKeyCode is sent when invite code changed (string) - invite
	AuditLogChangeKeyCode AuditLogChangeKey = "code"
	// AuditLogChangeKeyColor is sent when role color changed (int) - role
	AuditLogChangeKeyColor AuditLogChangeKey = "color"
	// AuditLogChangeKeyCommunicationDisabledUntil is sent when member timeout state changed (ISO8601 timestamp) - member
	AuditLogChangeKeyCommunicationDisabledUntil AuditLogChangeKey = "communication_disabled_until"
	// AuditLogChangeKeyDeaf is sent when user server deafened/undeafened (bool) - member
	AuditLogChangeKeyDeaf AuditLogChangeKey = "deaf"
	// AuditLogChangeKeyDefaultAutoArchiveDuration is sent when default auto archive duration for newly created threads changed (int) - channel
	AuditLogChangeKeyDefaultAutoArchiveDuration AuditLogChangeKey = "default_auto_archive_duration"
	// AuditLogChangeKeyDefaultMessageNotification is sent when default message notification level changed (int) - guild
	AuditLogChangeKeyDefaultMessageNotification AuditLogChangeKey = "default_message_notifications"
	// AuditLogChangeKeyDeny is sent when a permission on a text or voice channel was denied for a role (string) - role
	AuditLogChangeKeyDeny AuditLogChangeKey = "deny"
	// AuditLogChangeKeyDescription is sent when description changed (string) - guild, sticker, or guild scheduled event
	AuditLogChangeKeyDescription AuditLogChangeKey = "description"
	// AuditLogChangeKeyDiscoverySplashHash is sent when discovery splash changed (string) - guild
	AuditLogChangeKeyDiscoverySplashHash AuditLogChangeKey = "discovery_splash_hash"
	// AuditLogChangeKeyEnableEmoticons is sent when integration emoticons enabled/disabled (bool) - integration
	AuditLogChangeKeyEnableEmoticons AuditLogChangeKey = "enable_emoticons"
	// AuditLogChangeKeyEntityType is sent when entity type of guild scheduled event was changed (int) - guild scheduled event
	AuditLogChangeKeyEntityType AuditLogChangeKey = "entity_type"
	// AuditLogChangeKeyExpireBehavior is sent when integration expiring subscriber behavior changed (int) - integration
	AuditLogChangeKeyExpireBehavior AuditLogChangeKey = "expire_behavior"
	// AuditLogChangeKeyExpireGracePeriod is sent when integration expire grace period changed (int) - integration
	AuditLogChangeKeyExpireGracePeriod AuditLogChangeKey = "expire_grace_period"
	// AuditLogChangeKeyExplicitContentFilter is sent when change in whose messages are scanned and deleted for explicit content in the server is made (int) - guild
	AuditLogChangeKeyExplicitContentFilter AuditLogChangeKey = "explicit_content_filter"
	// AuditLogChangeKeyFormatType is sent when format type of sticker changed (int - sticker format type) - sticker
	AuditLogChangeKeyFormatType AuditLogChangeKey = "format_type"
	// AuditLogChangeKeyGuildID is sent when guild sticker is in changed (snowflake) - sticker
	AuditLogChangeKeyGuildID AuditLogChangeKey = "guild_id"
	// AuditLogChangeKeyHoist is sent when role is now displayed/no longer displayed separate from online users (bool) - role
	AuditLogChangeKeyHoist AuditLogChangeKey = "hoist"
	// AuditLogChangeKeyIconHash is sent when icon changed (string) - guild or role
	AuditLogChangeKeyIconHash AuditLogChangeKey = "icon_hash"
	// AuditLogChangeKeyID is sent when the id of the changed entity - sometimes used in conjunction with other keys (snowflake) - any
	AuditLogChangeKeyID AuditLogChangeKey = "id"
	// AuditLogChangeKeyInvitable is sent when private thread is now invitable/uninvitable (bool) - thread
	AuditLogChangeKeyInvitable AuditLogChangeKey = "invitable"
	// AuditLogChangeKeyInviterID is sent when person who created invite code changed (snowflake) - invite
	AuditLogChangeKeyInviterID AuditLogChangeKey = "inviter_id"
	// AuditLogChangeKeyLocation is sent when channel id for guild scheduled event changed (string) - guild scheduled event
	AuditLogChangeKeyLocation AuditLogChangeKey = "location"
	// AuditLogChangeKeyLocked is sent when thread was locked/unlocked (bool) - thread
	AuditLogChangeKeyLocked AuditLogChangeKey = "locked"
	// AuditLogChangeKeyMaxAge is sent when invite code expiration time changed (int) - invite
	AuditLogChangeKeyMaxAge AuditLogChangeKey = "max_age"
	// AuditLogChangeKeyMaxUses is sent when max number of times invite code can be used changed (int) - invite
	AuditLogChangeKeyMaxUses AuditLogChangeKey = "max_uses"
	// AuditLogChangeKeyMentionable is sent when role is now mentionable/unmentionable (bool) - role
	AuditLogChangeKeyMentionable AuditLogChangeKey = "mentionable"
	// AuditLogChangeKeyMfaLevel is sent when two-factor auth requirement changed (int - mfa level) - guild
	AuditLogChangeKeyMfaLevel AuditLogChangeKey = "mfa_level"
	// AuditLogChangeKeyMute is sent when user server muted/unmuted (bool) - member
	AuditLogChangeKeyMute AuditLogChangeKey = "mute"
	// AuditLogChangeKeyName is sent when name changed (string) - any
	AuditLogChangeKeyName AuditLogChangeKey = "name"
	// AuditLogChangeKeyNick is sent when user nickname changed (string) - member
	AuditLogChangeKeyNick AuditLogChangeKey = "nick"
	// AuditLogChangeKeyNSFW is sent when channel nsfw restriction changed (bool) - channel
	AuditLogChangeKeyNSFW AuditLogChangeKey = "nsfw"
	// AuditLogChangeKeyOwnerID is sent when owner changed (snowflake) - guild
	AuditLogChangeKeyOwnerID AuditLogChangeKey = "owner_id"
	// AuditLogChangeKeyPermissionOverwrite is sent when permissions on a channel changed (array of channel overwrite objects) - channel
	AuditLogChangeKeyPermissionOverwrite AuditLogChangeKey = "permission_overwrites"
	// AuditLogChangeKeyPermissions is sent when permissions for a role changed (string) - role
	AuditLogChangeKeyPermissions AuditLogChangeKey = "permissions"
	// AuditLogChangeKeyPosition is sent when text or voice channel position changed (int) - channel
	AuditLogChangeKeyPosition AuditLogChangeKey = "position"
	// AuditLogChangeKeyPreferredLocale is sent when preferred locale changed (string) - guild
	AuditLogChangeKeyPreferredLocale AuditLogChangeKey = "preferred_locale"
	// AuditLogChangeKeyPrivacylevel is sent when privacy level of the stage instance changed (integer - privacy level) - stage instance or guild scheduled event
	AuditLogChangeKeyPrivacylevel AuditLogChangeKey = "privacy_level"
	// AuditLogChangeKeyPruneDeleteDays is sent when number of days after which inactive and role-unassigned members are kicked changed (int) - guild
	AuditLogChangeKeyPruneDeleteDays AuditLogChangeKey = "prune_delete_days"
	// AuditLogChangeKeyPublicUpdatesChannelID is sent when id of the public updates channel changed (snowflake) - guild
	AuditLogChangeKeyPublicUpdatesChannelID AuditLogChangeKey = "public_updates_channel_id"
	// AuditLogChangeKeyRateLimitPerUser is sent when amount of seconds a user has to wait before sending another message changed (int) - channel
	AuditLogChangeKeyRateLimitPerUser AuditLogChangeKey = "rate_limit_per_user"
	// AuditLogChangeKeyRegion is sent when region changed (string) - guild
	AuditLogChangeKeyRegion AuditLogChangeKey = "region"
	// AuditLogChangeKeyRulesChannelID is sent when id of the rules channel changed (snowflake) - guild
	AuditLogChangeKeyRulesChannelID AuditLogChangeKey = "rules_channel_id"
	// AuditLogChangeKeySplashHash is sent when invite splash page artwork changed (string) - guild
	AuditLogChangeKeySplashHash AuditLogChangeKey = "splash_hash"
	// AuditLogChangeKeyStatus is sent when status of guild scheduled event was changed (int - guild scheduled event status) - guild scheduled event
	AuditLogChangeKeyStatus AuditLogChangeKey = "status"
	// AuditLogChangeKeySystemChannelID is sent when id of the system channel changed (snowflake) - guild
	AuditLogChangeKeySystemChannelID AuditLogChangeKey = "system_channel_id"
	// AuditLogChangeKeyTags is sent when related emoji of sticker changed (string) - sticker
	AuditLogChangeKeyTags AuditLogChangeKey = "tags"
	// AuditLogChangeKeyTemporary is sent when invite code is now temporary or never expires (bool) - invite
	AuditLogChangeKeyTemporary AuditLogChangeKey = "temporary"
	// TODO: remove when compatibility is not required
	AuditLogChangeKeyTempoary = AuditLogChangeKeyTemporary
	// AuditLogChangeKeyTopic is sent when text channel topic or stage instance topic changed (string) - channel or stage instance
	AuditLogChangeKeyTopic AuditLogChangeKey = "topic"
	// AuditLogChangeKeyType is sent when type of entity created (int or string) - any
	AuditLogChangeKeyType AuditLogChangeKey = "type"
	// AuditLogChangeKeyUnicodeEmoji is sent when role unicode emoji changed (string) - role
	AuditLogChangeKeyUnicodeEmoji AuditLogChangeKey = "unicode_emoji"
	// AuditLogChangeKeyUserLimit is sent when new user limit in a voice channel set (int) - voice channel
	AuditLogChangeKeyUserLimit AuditLogChangeKey = "user_limit"
	// AuditLogChangeKeyUses is sent when number of times invite code used changed (int) - invite
	AuditLogChangeKeyUses AuditLogChangeKey = "uses"
	// AuditLogChangeKeyVanityURLCode is sent when guild invite vanity url changed (string) - guild
	AuditLogChangeKeyVanityURLCode AuditLogChangeKey = "vanity_url_code"
	// AuditLogChangeKeyVerificationLevel is sent when required verification level changed (int - verification level) - guild
	AuditLogChangeKeyVerificationLevel AuditLogChangeKey = "verification_level"
	// AuditLogChangeKeyWidgetChannelID is sent when channel id of the server widget changed (snowflake) - guild
	AuditLogChangeKeyWidgetChannelID AuditLogChangeKey = "widget_channel_id"
	// AuditLogChangeKeyWidgetEnabled is sent when server widget enabled/disabled (bool) - guild
	AuditLogChangeKeyWidgetEnabled AuditLogChangeKey = "widget_enabled"
	// AuditLogChangeKeyRoleAdd is sent when new role added (array of partial role objects) - guild
	AuditLogChangeKeyRoleAdd AuditLogChangeKey = "$add"
	// AuditLogChangeKeyRoleRemove is sent when role removed (array of partial role objects) - guild
	AuditLogChangeKeyRoleRemove AuditLogChangeKey = "$remove"
)

// AuditLogOptions optional data for the AuditLog
// https://discord.com/developers/docs/resources/audit-log#audit-log-entry-object-optional-audit-entry-info
type AuditLogOptions struct {
	DeleteMemberDays              string               `json:"delete_member_days"`
	MembersRemoved                string               `json:"members_removed"`
	ChannelID                     string               `json:"channel_id"`
	MessageID                     string               `json:"message_id"`
	Count                         string               `json:"count"`
	ID                            string               `json:"id"`
	Type                          *AuditLogOptionsType `json:"type"`
	RoleName                      string               `json:"role_name"`
	ApplicationID                 string               `json:"application_id"`
	AutoModerationRuleName        string               `json:"auto_moderation_rule_name"`
	AutoModerationRuleTriggerType string               `json:"auto_moderation_rule_trigger_type"`
	IntegrationType               string               `json:"integration_type"`
}

// AuditLogOptionsType of the AuditLogOption
// https://discord.com/developers/docs/resources/audit-log#audit-log-entry-object-optional-audit-entry-info
type AuditLogOptionsType string

// Valid Types for AuditLogOptionsType
const (
	AuditLogOptionsTypeRole   AuditLogOptionsType = "0"
	AuditLogOptionsTypeMember AuditLogOptionsType = "1"
)

// AuditLogAction is the Action of the AuditLog (see AuditLogAction* consts)
// https://discord.com/developers/docs/resources/audit-log#audit-log-entry-object-audit-log-events
type AuditLogAction int

// Block contains Discord Audit Log Action Types
const (
	AuditLogActionGuildUpdate AuditLogAction = 1

	AuditLogActionChannelCreate          AuditLogAction = 10
	AuditLogActionChannelUpdate          AuditLogAction = 11
	AuditLogActionChannelDelete          AuditLogAction = 12
	AuditLogActionChannelOverwriteCreate AuditLogAction = 13
	AuditLogActionChannelOverwriteUpdate AuditLogAction = 14
	AuditLogActionChannelOverwriteDelete AuditLogAction = 15

	AuditLogActionMemberKick       AuditLogAction = 20
	AuditLogActionMemberPrune      AuditLogAction = 21
	AuditLogActionMemberBanAdd     AuditLogAction = 22
	AuditLogActionMemberBanRemove  AuditLogAction = 23
	AuditLogActionMemberUpdate     AuditLogAction = 24
	AuditLogActionMemberRoleUpdate AuditLogAction = 25
	AuditLogActionMemberMove       AuditLogAction = 26
	AuditLogActionMemberDisconnect AuditLogAction = 27
	AuditLogActionBotAdd           AuditLogAction = 28

	AuditLogActionRoleCreate AuditLogAction = 30
	AuditLogActionRoleUpdate AuditLogAction = 31
	AuditLogActionRoleDelete AuditLogAction = 32

	AuditLogActionInviteCreate AuditLogAction = 40
	AuditLogActionInviteUpdate AuditLogAction = 41
	AuditLogActionInviteDelete AuditLogAction = 42

	AuditLogActionWebhookCreate AuditLogAction = 50
	AuditLogActionWebhookUpdate AuditLogAction = 51
	AuditLogActionWebhookDelete AuditLogAction = 52

	AuditLogActionEmojiCreate AuditLogAction = 60
	AuditLogActionEmojiUpdate AuditLogAction = 61
	AuditLogActionEmojiDelete AuditLogAction = 62

	AuditLogActionMessageDelete     AuditLogAction = 72
	AuditLogActionMessageBulkDelete AuditLogAction = 73
	AuditLogActionMessagePin        AuditLogAction = 74
	AuditLogActionMessageUnpin      AuditLogAction = 75

	AuditLogActionIntegrationCreate   AuditLogAction = 80
	AuditLogActionIntegrationUpdate   AuditLogAction = 81
	AuditLogActionIntegrationDelete   AuditLogAction = 82
	AuditLogActionStageInstanceCreate AuditLogAction = 83
	AuditLogActionStageInstanceUpdate AuditLogAction = 84
	AuditLogActionStageInstanceDelete AuditLogAction = 85

	AuditLogActionStickerCreate AuditLogAction = 90
	AuditLogActionStickerUpdate AuditLogAction = 91
	AuditLogActionStickerDelete AuditLogAction = 92

	AuditLogGuildScheduledEventCreate AuditLogAction = 100
	AuditLogGuildScheduledEventUpdate AuditLogAction = 101
	AuditLogGuildScheduledEventDelete AuditLogAction = 102

	AuditLogActionThreadCreate AuditLogAction = 110
	AuditLogActionThreadUpdate AuditLogAction = 111
	AuditLogActionThreadDelete AuditLogAction = 112

	AuditLogActionApplicationCommandPermissionUpdate AuditLogAction = 121

	AuditLogActionAutoModerationRuleCreate                AuditLogAction = 140
	AuditLogActionAutoModerationRuleUpdate                AuditLogAction = 141
	AuditLogActionAutoModerationRuleDelete                AuditLogAction = 142
	AuditLogActionAutoModerationBlockMessage              AuditLogAction = 143
	AuditLogActionAutoModerationFlagToChannel             AuditLogAction = 144
	AuditLogActionAutoModerationUserCommunicationDisabled AuditLogAction = 145

	AuditLogActionCreatorMonetizationRequestCreated AuditLogAction = 150
	AuditLogActionCreatorMonetizationTermsAccepted  AuditLogAction = 151

	AuditLogActionOnboardingPromptCreate AuditLogAction = 163
	AuditLogActionOnboardingPromptUpdate AuditLogAction = 164
	AuditLogActionOnboardingPromptDelete AuditLogAction = 165
	AuditLogActionOnboardingCreate       AuditLogAction = 166
	AuditLogActionOnboardingUpdate       AuditLogAction = 167

	AuditLogActionHomeSettingsCreate = 190
	AuditLogActionHomeSettingsUpdate = 191
)

// GuildMemberParams stores data needed to update a member
// https://discord.com/developers/docs/resources/guild#modify-guild-member
type GuildMemberParams struct {
	// Value to set user's nickname to.
	Nick string `json:"nick,omitempty"`
	// Array of role ids the member is assigned.
	Roles *[]string `json:"roles,omitempty"`
	// ID of channel to move user to (if they are connected to voice).
	// Set to "" to remove user from a voice channel.
	ChannelID *string `json:"channel_id,omitempty"`
	// Whether the user is muted in voice channels.
	Mute *bool `json:"mute,omitempty"`
	// Whether the user is deafened in voice channels.
	Deaf *bool `json:"deaf,omitempty"`
	// When the user's timeout will expire and the user will be able
	// to communicate in the guild again (up to 28 days in the future).
	// Set to time.Time{} to remove timeout.
	CommunicationDisabledUntil *time.Time `json:"communication_disabled_until,omitempty"`
}

// MarshalJSON is a helper function to marshal GuildMemberParams.
func (p GuildMemberParams) MarshalJSON() (res []byte, err error) {
	type guildMemberParams GuildMemberParams
	v := struct {
		guildMemberParams
		ChannelID                  json.RawMessage `json:"channel_id,omitempty"`
		CommunicationDisabledUntil json.RawMessage `json:"communication_disabled_until,omitempty"`
	}{guildMemberParams: guildMemberParams(p)}

	if p.ChannelID != nil {
		if *p.ChannelID == "" {
			v.ChannelID = json.RawMessage(`null`)
		} else {
			res, err = json.Marshal(p.ChannelID)
			if err != nil {
				return
			}
			v.ChannelID = res
		}
	}

	if p.CommunicationDisabledUntil != nil {
		if p.CommunicationDisabledUntil.IsZero() {
			v.CommunicationDisabledUntil = json.RawMessage(`null`)
		} else {
			res, err = json.Marshal(p.CommunicationDisabledUntil)
			if err != nil {
				return
			}
			v.CommunicationDisabledUntil = res
		}
	}

	return json.Marshal(v)
}

// GuildMemberAddParams stores data needed to add a user to a guild.
// NOTE: All fields are optional, except AccessToken.
type GuildMemberAddParams struct {
	// Valid access_token for the user.
	AccessToken string `json:"access_token"`
	// Value to set users nickname to.
	Nick string `json:"nick,omitempty"`
	// A list of role ID's to set on the member.
	Roles []string `json:"roles,omitempty"`
	// Whether the user is muted.
	Mute bool `json:"mute,omitempty"`
	// Whether the user is deafened.
	Deaf bool `json:"deaf,omitempty"`
}

// An APIErrorMessage is an api error message returned from discord
type APIErrorMessage struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MessageReaction stores the data for a message reaction.
type MessageReaction struct {
	UserID    string `json:"user_id"`
	MessageID string `json:"message_id"`
	Emoji     Emoji  `json:"emoji"`
	ChannelID string `json:"channel_id"`
	GuildID   string `json:"guild_id,omitempty"`
}

// GatewayBotResponse stores the data for the gateway/bot response
type GatewayBotResponse struct {
	URL               string             `json:"url"`
	Shards            int                `json:"shards"`
	SessionStartLimit SessionInformation `json:"session_start_limit"`
}

// SessionInformation provides the information for max concurrency sharding
type SessionInformation struct {
	Total          int `json:"total,omitempty"`
	Remaining      int `json:"remaining,omitempty"`
	ResetAfter     int `json:"reset_after,omitempty"`
	MaxConcurrency int `json:"max_concurrency,omitempty"`
}

// GatewayStatusUpdate is sent by the client to indicate a presence or status update
// https://discord.com/developers/docs/topics/gateway#update-status-gateway-status-update-structure
type GatewayStatusUpdate struct {
	Since  int      `json:"since"`
	Game   Activity `json:"game"`
	Status string   `json:"status"`
	AFK    bool     `json:"afk"`
}

// Activity defines the Activity sent with GatewayStatusUpdate
// https://discord.com/developers/docs/topics/gateway#activity-object
type Activity struct {
	Name          string       `json:"name"`
	Type          ActivityType `json:"type"`
	URL           string       `json:"url,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
	ApplicationID string       `json:"application_id,omitempty"`
	State         string       `json:"state,omitempty"`
	Details       string       `json:"details,omitempty"`
	Timestamps    TimeStamps   `json:"timestamps,omitempty"`
	Emoji         Emoji        `json:"emoji,omitempty"`
	Party         Party        `json:"party,omitempty"`
	Assets        Assets       `json:"assets,omitempty"`
	Secrets       Secrets      `json:"secrets,omitempty"`
	Instance      bool         `json:"instance,omitempty"`
	Flags         int          `json:"flags,omitempty"`
}

// UnmarshalJSON is a custom unmarshaljson to make CreatedAt a time.Time instead of an int
func (activity *Activity) UnmarshalJSON(b []byte) error {
	temp := struct {
		Name          string       `json:"name"`
		Type          ActivityType `json:"type"`
		URL           string       `json:"url,omitempty"`
		CreatedAt     int64        `json:"created_at"`
		ApplicationID json.Number  `json:"application_id,omitempty"`
		State         string       `json:"state,omitempty"`
		Details       string       `json:"details,omitempty"`
		Timestamps    TimeStamps   `json:"timestamps,omitempty"`
		Emoji         Emoji        `json:"emoji,omitempty"`
		Party         Party        `json:"party,omitempty"`
		Assets        Assets       `json:"assets,omitempty"`
		Secrets       Secrets      `json:"secrets,omitempty"`
		Instance      bool         `json:"instance,omitempty"`
		Flags         int          `json:"flags,omitempty"`
	}{}
	err := Unmarshal(b, &temp)
	if err != nil {
		return err
	}
	activity.ApplicationID = temp.ApplicationID.String()
	activity.CreatedAt = time.Unix(0, temp.CreatedAt*1000000)
	activity.Assets = temp.Assets
	activity.Details = temp.Details
	activity.Emoji = temp.Emoji
	activity.Flags = temp.Flags
	activity.Instance = temp.Instance
	activity.Name = temp.Name
	activity.Party = temp.Party
	activity.Secrets = temp.Secrets
	activity.State = temp.State
	activity.Timestamps = temp.Timestamps
	activity.Type = temp.Type
	activity.URL = temp.URL
	return nil
}

// Party defines the Party field in the Activity struct
// https://discord.com/developers/docs/topics/gateway#activity-object
type Party struct {
	ID   string `json:"id,omitempty"`
	Size []int  `json:"size,omitempty"`
}

// Secrets defines the Secrets field for the Activity struct
// https://discord.com/developers/docs/topics/gateway#activity-object
type Secrets struct {
	Join     string `json:"join,omitempty"`
	Spectate string `json:"spectate,omitempty"`
	Match    string `json:"match,omitempty"`
}

// ActivityType is the type of Activity (see ActivityType* consts) in the Activity struct
// https://discord.com/developers/docs/topics/gateway#activity-object-activity-types
type ActivityType int

// Valid ActivityType values
const (
	ActivityTypeGame      ActivityType = 0
	ActivityTypeStreaming ActivityType = 1
	ActivityTypeListening ActivityType = 2
	ActivityTypeWatching  ActivityType = 3
	ActivityTypeCustom    ActivityType = 4
	ActivityTypeCompeting ActivityType = 5
)

// Identify is sent during initial handshake with the discord gateway.
// https://discord.com/developers/docs/topics/gateway#identify
type Identify struct {
	Token          string              `json:"token"`
	Properties     IdentifyProperties  `json:"properties"`
	Compress       bool                `json:"compress"`
	LargeThreshold int                 `json:"large_threshold"`
	Shard          *[2]int             `json:"shard,omitempty"`
	Presence       GatewayStatusUpdate `json:"presence,omitempty"`
	Intents        Intent              `json:"intents"`
}

// IdentifyProperties contains the "properties" portion of an Identify packet
// https://discord.com/developers/docs/topics/gateway#identify-identify-connection-properties
type IdentifyProperties struct {
	OS              string `json:"$os"`
	Browser         string `json:"$browser"`
	Device          string `json:"$device"`
	Referer         string `json:"$referer"`
	ReferringDomain string `json:"$referring_domain"`
}

// StageInstance holds information about a live stage.
// https://discord.com/developers/docs/resources/stage-instance#stage-instance-resource
type StageInstance struct {
	// The id of this Stage instance
	ID string `json:"id"`
	// The guild id of the associated Stage channel
	GuildID string `json:"guild_id"`
	// The id of the associated Stage channel
	ChannelID string `json:"channel_id"`
	// The topic of the Stage instance (1-120 characters)
	Topic string `json:"topic"`
	// The privacy level of the Stage instance
	// https://discord.com/developers/docs/resources/stage-instance#stage-instance-object-privacy-level
	PrivacyLevel StageInstancePrivacyLevel `json:"privacy_level"`
	// Whether or not Stage Discovery is disabled (deprecated)
	DiscoverableDisabled bool `json:"discoverable_disabled"`
	// The id of the scheduled event for this Stage instance
	GuildScheduledEventID string `json:"guild_scheduled_event_id"`
}

// StageInstanceParams represents the parameters needed to create or edit a stage instance
type StageInstanceParams struct {
	// ChannelID represents the id of the Stage channel
	ChannelID string `json:"channel_id,omitempty"`
	// Topic of the Stage instance (1-120 characters)
	Topic string `json:"topic,omitempty"`
	// PrivacyLevel of the Stage instance (default GUILD_ONLY)
	PrivacyLevel StageInstancePrivacyLevel `json:"privacy_level,omitempty"`
	// SendStartNotification will notify @everyone that a Stage instance has started
	SendStartNotification bool `json:"send_start_notification,omitempty"`
}

// StageInstancePrivacyLevel represents the privacy level of a Stage instance
// https://discord.com/developers/docs/resources/stage-instance#stage-instance-object-privacy-level
type StageInstancePrivacyLevel int

const (
	// StageInstancePrivacyLevelPublic The Stage instance is visible publicly. (deprecated)
	StageInstancePrivacyLevelPublic StageInstancePrivacyLevel = 1
	// StageInstancePrivacyLevelGuildOnly The Stage instance is visible to only guild members.
	StageInstancePrivacyLevelGuildOnly StageInstancePrivacyLevel = 2
)

// PollLayoutType represents the layout of a poll.
type PollLayoutType int

// Valid PollLayoutType values.
const (
	PollLayoutTypeDefault PollLayoutType = 1
)

// PollMedia contains common data used by question and answers.
type PollMedia struct {
	Text  string          `json:"text,omitempty"`
	Emoji *ComponentEmoji `json:"emoji,omitempty"` // TODO: rename the type
}

// PollAnswer represents a single answer in a poll.
type PollAnswer struct {
	// NOTE: should not be set on creation.
	AnswerID int        `json:"answer_id,omitempty"`
	Media    *PollMedia `json:"poll_media"`
}

// PollAnswerCount stores counted poll votes for a single answer.
type PollAnswerCount struct {
	ID      int  `json:"id"`
	Count   int  `json:"count"`
	MeVoted bool `json:"me_voted"`
}

// PollResults contains voting results on a poll.
type PollResults struct {
	Finalized    bool               `json:"is_finalized"`
	AnswerCounts []*PollAnswerCount `json:"answer_counts"`
}

// Poll contains all poll related data.
type Poll struct {
	Question         PollMedia      `json:"question"`
	Answers          []PollAnswer   `json:"answers"`
	AllowMultiselect bool           `json:"allow_multiselect"`
	LayoutType       PollLayoutType `json:"layout_type,omitempty"`

	// NOTE: should be set only on creation, when fetching use Expiry.
	Duration int `json:"duration,omitempty"`

	// NOTE: available only when fetching.

	Results *PollResults `json:"results,omitempty"`
	// NOTE: as Discord documentation notes, this field might be null even when fetching.
	Expiry *time.Time `json:"expiry,omitempty"`
}

// SKUType is the type of SKU (see SKUType* consts)
// https://discord.com/developers/docs/monetization/skus
type SKUType int

// Valid SKUType values
const (
	SKUTypeDurable      SKUType = 2
	SKUTypeConsumable   SKUType = 3
	SKUTypeSubscription SKUType = 5
	// SKUTypeSubscriptionGroup is a system-generated group for each subscription SKU.
	SKUTypeSubscriptionGroup SKUType = 6
)

// SKUFlags is a bitfield of flags used to differentiate user and server subscriptions (see SKUFlag* consts)
// https://discord.com/developers/docs/monetization/skus#sku-object-sku-flags
type SKUFlags int

const (
	// SKUFlagAvailable indicates that the SKU is available for purchase.
	SKUFlagAvailable SKUFlags = 1 << 2
	// SKUFlagGuildSubscription indicates that the SKU is a guild subscription.
	SKUFlagGuildSubscription SKUFlags = 1 << 7
	// SKUFlagUserSubscription indicates that the SKU is a user subscription.
	SKUFlagUserSubscription SKUFlags = 1 << 8
)

// SKU (stock-keeping units) represent premium offerings
type SKU struct {
	// The ID of the SKU
	ID string `json:"id"`

	// The Type of the SKU
	Type SKUType `json:"type"`

	// The ID of the parent application
	ApplicationID string `json:"application_id"`

	// Customer-facing name of the SKU.
	Name string `json:"name"`

	// System-generated URL slug based on the SKU's name.
	Slug string `json:"slug"`

	// SKUFlags combined as a bitfield. The presence of a certain flag can be checked
	// by performing a bitwise AND operation between this int and the flag.
	Flags SKUFlags `json:"flags"`
}

// Subscription represents a user making recurring payments for at least one SKU over an ongoing period.
// https://discord.com/developers/docs/resources/subscription#subscription-object
type Subscription struct {
	// ID of the subscription
	ID string `json:"id"`

	// ID of the user who is subscribed
	UserID string `json:"user_id"`

	// List of SKUs subscribed to
	SKUIDs []string `json:"sku_ids"`

	// List of entitlements granted for this subscription
	EntitlementIDs []string `json:"entitlement_ids"`

	// List of SKUs that this user will be subscribed to at renewal
	RenewalSKUIDs []string `json:"renewal_sku_ids,omitempty"`

	// Start of the current subscription period
	CurrentPeriodStart time.Time `json:"current_period_start"`

	// End of the current subscription period
	CurrentPeriodEnd time.Time `json:"current_period_end"`

	// Current status of the subscription
	Status SubscriptionStatus `json:"status"`

	// When the subscription was canceled. Only present if the subscription has been canceled.
	CanceledAt *time.Time `json:"canceled_at,omitempty"`

	// ISO3166-1 alpha-2 country code of the payment source used to purchase the subscription. Missing unless queried with a private OAuth scope.
	Country string `json:"country,omitempty"`
}

// SubscriptionStatus is the current status of a Subscription Object
// https://discord.com/developers/docs/resources/subscription#subscription-statuses
type SubscriptionStatus int

// Valid SubscriptionStatus values
const (
	SubscriptionStatusActive   = 0
	SubscriptionStatusEnding   = 1
	SubscriptionStatusInactive = 2
)

// EntitlementType is the type of entitlement (see EntitlementType* consts)
// https://discord.com/developers/docs/monetization/entitlements#entitlement-object-entitlement-types
type EntitlementType int

// Valid EntitlementType values
const (
	EntitlementTypePurchase                = 1
	EntitlementTypePremiumSubscription     = 2
	EntitlementTypeDeveloperGift           = 3
	EntitlementTypeTestModePurchase        = 4
	EntitlementTypeFreePurchase            = 5
	EntitlementTypeUserGift                = 6
	EntitlementTypePremiumPurchase         = 7
	EntitlementTypeApplicationSubscription = 8
)

// Entitlement represents that a user or guild has access to a premium offering
// in your application.
type Entitlement struct {
	// The ID of the entitlement
	ID string `json:"id"`

	// The ID of the SKU
	SKUID string `json:"sku_id"`

	// The ID of the parent application
	ApplicationID string `json:"application_id"`

	// The ID of the user that is granted access to the entitlement's sku
	// Only available for user subscriptions.
	UserID string `json:"user_id,omitempty"`

	// The type of the entitlement
	Type EntitlementType `json:"type"`

	// The entitlement was deleted
	Deleted bool `json:"deleted"`

	// The start date at which the entitlement is valid.
	// Not present when using test entitlements.
	StartsAt *time.Time `json:"starts_at,omitempty"`

	// The date at which the entitlement is no longer valid.
	// Not present when using test entitlements or when receiving an ENTITLEMENT_CREATE event.
	EndsAt *time.Time `json:"ends_at,omitempty"`

	// The ID of the guild that is granted access to the entitlement's sku.
	// Only available for guild subscriptions.
	GuildID string `json:"guild_id,omitempty"`

	// Whether or not the entitlement has been consumed.
	// Only available for consumable items.
	Consumed *bool `json:"consumed,omitempty"`

	// The SubscriptionID of the entitlement.
	// Not present when using test entitlements.
	SubscriptionID string `json:"subscription_id,omitempty"`
}

// EntitlementOwnerType is the type of entitlement (see EntitlementOwnerType* consts)
type EntitlementOwnerType int

// Valid EntitlementOwnerType values
const (
	EntitlementOwnerTypeGuildSubscription EntitlementOwnerType = 1
	EntitlementOwnerTypeUserSubscription  EntitlementOwnerType = 2
)

// EntitlementTest is used to test granting an entitlement to a user or guild
type EntitlementTest struct {
	// The ID of the SKU to grant the entitlement to
	SKUID string `json:"sku_id"`

	// The ID of the guild or user to grant the entitlement to
	OwnerID string `json:"owner_id"`

	// OwnerType is the type of which the entitlement should be created
	OwnerType EntitlementOwnerType `json:"owner_type"`
}

// EntitlementFilterOptions are the options for filtering Entitlements
type EntitlementFilterOptions struct {
	// Optional user ID to look up for.
	UserID string

	// Optional array of SKU IDs to check for.
	SkuIDs []string

	// Optional timestamp to retrieve Entitlements before this time.
	Before *time.Time

	// Optional timestamp to retrieve Entitlements after this time.
	After *time.Time

	// Optional maximum number of entitlements to return (1-100, default 100).
	Limit int

	// Optional guild ID to look up for.
	GuildID string

	// Optional whether or not ended entitlements should be omitted.
	ExcludeEnded bool
}

// Constants for the different bit offsets of text channel permissions
const (
	// Deprecated: PermissionReadMessages has been replaced with PermissionViewChannel for text and voice channels
	PermissionReadMessages = 1 << 10

	// Allows for sending messages in a channel and creating threads in a forum (does not allow sending messages in threads).
	PermissionSendMessages = 1 << 11

	// Allows for sending of /tts messages.
	PermissionSendTTSMessages = 1 << 12

	// Allows for deletion of other users messages.
	PermissionManageMessages = 1 << 13

	// Links sent by users with this permission will be auto-embedded.
	PermissionEmbedLinks = 1 << 14

	// Allows for uploading images and files.
	PermissionAttachFiles = 1 << 15

	// Allows for reading of message history.
	PermissionReadMessageHistory = 1 << 16

	// Allows for using the @everyone tag to notify all users in a channel, and the @here tag to notify all online users in a channel.
	PermissionMentionEveryone = 1 << 17

	// Allows the usage of custom emojis from other servers.
	PermissionUseExternalEmojis = 1 << 18

	// Deprecated: PermissionUseSlashCommands has been replaced by PermissionUseApplicationCommands
	PermissionUseSlashCommands = 1 << 31

	// Allows members to use application commands, including slash commands and context menu commands.
	PermissionUseApplicationCommands = 1 << 31

	// Allows for deleting and archiving threads, and viewing all private threads.
	PermissionManageThreads = 1 << 34

	// Allows for creating public and announcement threads.
	PermissionCreatePublicThreads = 1 << 35

	// Allows for creating private threads.
	PermissionCreatePrivateThreads = 1 << 36

	// Allows the usage of custom stickers from other servers.
	PermissionUseExternalStickers = 1 << 37

	// Allows for sending messages in threads.
	PermissionSendMessagesInThreads = 1 << 38

	// Allows sending voice messages.
	PermissionSendVoiceMessages = 1 << 46

	// Allows sending polls.
	PermissionSendPolls = 1 << 49

	// Allows user-installed apps to send public responses. When disabled, users will still be allowed to use their apps but the responses will be ephemeral. This only applies to apps not also installed to the server.
	PermissionUseExternalApps = 1 << 50
)

// Constants for the different bit offsets of voice permissions
const (
	// Allows for using priority speaker in a voice channel.
	PermissionVoicePrioritySpeaker = 1 << 8

	// Allows the user to go live.
	PermissionVoiceStreamVideo = 1 << 9

	// Allows for joining of a voice channel.
	PermissionVoiceConnect = 1 << 20

	// Allows for speaking in a voice channel.
	PermissionVoiceSpeak = 1 << 21

	// Allows for muting members in a voice channel.
	PermissionVoiceMuteMembers = 1 << 22

	// Allows for deafening of members in a voice channel.
	PermissionVoiceDeafenMembers = 1 << 23

	// Allows for moving of members between voice channels.
	PermissionVoiceMoveMembers = 1 << 24

	// Allows for using voice-activity-detection in a voice channel.
	PermissionVoiceUseVAD = 1 << 25

	// Allows for requesting to speak in stage channels.
	PermissionVoiceRequestToSpeak = 1 << 32

	// Deprecated: PermissionUseActivities has been replaced by PermissionUseEmbeddedActivities.
	PermissionUseActivities = 1 << 39

	// Allows for using Activities (applications with the EMBEDDED flag) in a voice channel.
	PermissionUseEmbeddedActivities = 1 << 39

	// Allows for using soundboard in a voice channel.
	PermissionUseSoundboard = 1 << 42

	// Allows the usage of custom soundboard sounds from other servers.
	PermissionUseExternalSounds = 1 << 45
)

// Constants for general management.
const (
	// Allows for modification of own nickname.
	PermissionChangeNickname = 1 << 26

	// Allows for modification of other users nicknames.
	PermissionManageNicknames = 1 << 27

	// Allows management and editing of roles.
	PermissionManageRoles = 1 << 28

	// Allows management and editing of webhooks.
	PermissionManageWebhooks = 1 << 29

	// Deprecated: PermissionManageEmojis has been replaced by PermissionManageGuildExpressions.
	PermissionManageEmojis = 1 << 30

	// Allows for editing and deleting emojis, stickers, and soundboard sounds created by all users.
	PermissionManageGuildExpressions = 1 << 30

	// Allows for editing and deleting scheduled events created by all users.
	PermissionManageEvents = 1 << 33

	// Allows for viewing role subscription insights.
	PermissionViewCreatorMonetizationAnalytics = 1 << 41

	// Allows for creating emojis, stickers, and soundboard sounds, and editing and deleting those created by the current user.
	PermissionCreateGuildExpressions = 1 << 43

	// Allows for creating scheduled events, and editing and deleting those created by the current user.
	PermissionCreateEvents = 1 << 44
)

// Constants for the different bit offsets of general permissions
const (
	// Allows creation of instant invites.
	PermissionCreateInstantInvite = 1 << 0

	// Allows kicking members.
	PermissionKickMembers = 1 << 1

	// Allows banning members.
	PermissionBanMembers = 1 << 2

	// Allows all permissions and bypasses channel permission overwrites.
	PermissionAdministrator = 1 << 3

	// Allows management and editing of channels.
	PermissionManageChannels = 1 << 4

	// Deprecated: PermissionManageServer has been replaced by PermissionManageGuild.
	PermissionManageServer = 1 << 5

	// Allows management and editing of the guild.
	PermissionManageGuild = 1 << 5

	// Allows for the addition of reactions to messages.
	PermissionAddReactions = 1 << 6

	// Allows for viewing of audit logs.
	PermissionViewAuditLogs = 1 << 7

	// Allows guild members to view a channel, which includes reading messages in text channels and joining voice channels.
	PermissionViewChannel = 1 << 10

	// Allows for viewing guild insights.
	PermissionViewGuildInsights = 1 << 19

	// Allows for timing out users to prevent them from sending or reacting to messages in chat and threads, and from speaking in voice and stage channels.
	PermissionModerateMembers = 1 << 40

	PermissionAllText = PermissionViewChannel |
		PermissionSendMessages |
		PermissionSendTTSMessages |
		PermissionManageMessages |
		PermissionEmbedLinks |
		PermissionAttachFiles |
		PermissionReadMessageHistory |
		PermissionMentionEveryone
	PermissionAllVoice = PermissionViewChannel |
		PermissionVoiceConnect |
		PermissionVoiceSpeak |
		PermissionVoiceMuteMembers |
		PermissionVoiceDeafenMembers |
		PermissionVoiceMoveMembers |
		PermissionVoiceUseVAD |
		PermissionVoicePrioritySpeaker
	PermissionAllChannel = PermissionAllText |
		PermissionAllVoice |
		PermissionCreateInstantInvite |
		PermissionManageRoles |
		PermissionManageChannels |
		PermissionAddReactions |
		PermissionViewAuditLogs
	PermissionAll = PermissionAllChannel |
		PermissionKickMembers |
		PermissionBanMembers |
		PermissionManageServer |
		PermissionAdministrator |
		PermissionManageWebhooks |
		PermissionManageEmojis
)

// Block contains Discord JSON Error Response codes
const (
	ErrCodeGeneralError = 0

	ErrCodeUnknownAccount                        = 10001
	ErrCodeUnknownApplication                    = 10002
	ErrCodeUnknownChannel                        = 10003
	ErrCodeUnknownGuild                          = 10004
	ErrCodeUnknownIntegration                    = 10005
	ErrCodeUnknownInvite                         = 10006
	ErrCodeUnknownMember                         = 10007
	ErrCodeUnknownMessage                        = 10008
	ErrCodeUnknownOverwrite                      = 10009
	ErrCodeUnknownProvider                       = 10010
	ErrCodeUnknownRole                           = 10011
	ErrCodeUnknownToken                          = 10012
	ErrCodeUnknownUser                           = 10013
	ErrCodeUnknownEmoji                          = 10014
	ErrCodeUnknownWebhook                        = 10015
	ErrCodeUnknownWebhookService                 = 10016
	ErrCodeUnknownSession                        = 10020
	ErrCodeUnknownBan                            = 10026
	ErrCodeUnknownSKU                            = 10027
	ErrCodeUnknownStoreListing                   = 10028
	ErrCodeUnknownEntitlement                    = 10029
	ErrCodeUnknownBuild                          = 10030
	ErrCodeUnknownLobby                          = 10031
	ErrCodeUnknownBranch                         = 10032
	ErrCodeUnknownStoreDirectoryLayout           = 10033
	ErrCodeUnknownRedistributable                = 10036
	ErrCodeUnknownGiftCode                       = 10038
	ErrCodeUnknownStream                         = 10049
	ErrCodeUnknownPremiumServerSubscribeCooldown = 10050
	ErrCodeUnknownGuildTemplate                  = 10057
	ErrCodeUnknownDiscoveryCategory              = 10059
	ErrCodeUnknownSticker                        = 10060
	ErrCodeUnknownInteraction                    = 10062
	ErrCodeUnknownApplicationCommand             = 10063
	ErrCodeUnknownApplicationCommandPermissions  = 10066
	ErrCodeUnknownStageInstance                  = 10067
	ErrCodeUnknownGuildMemberVerificationForm    = 10068
	ErrCodeUnknownGuildWelcomeScreen             = 10069
	ErrCodeUnknownGuildScheduledEvent            = 10070
	ErrCodeUnknownGuildScheduledEventUser        = 10071
	ErrUnknownTag                                = 10087

	ErrCodeBotsCannotUseEndpoint                                            = 20001
	ErrCodeOnlyBotsCanUseEndpoint                                           = 20002
	ErrCodeExplicitContentCannotBeSentToTheDesiredRecipients                = 20009
	ErrCodeYouAreNotAuthorizedToPerformThisActionOnThisApplication          = 20012
	ErrCodeThisActionCannotBePerformedDueToSlowmodeRateLimit                = 20016
	ErrCodeOnlyTheOwnerOfThisAccountCanPerformThisAction                    = 20018
	ErrCodeMessageCannotBeEditedDueToAnnouncementRateLimits                 = 20022
	ErrCodeChannelHasHitWriteRateLimit                                      = 20028
	ErrCodeTheWriteActionYouArePerformingOnTheServerHasHitTheWriteRateLimit = 20029
	ErrCodeStageTopicContainsNotAllowedWordsForPublicStages                 = 20031
	ErrCodeGuildPremiumSubscriptionLevelTooLow                              = 20035

	ErrCodeMaximumGuildsReached                                     = 30001
	ErrCodeMaximumPinsReached                                       = 30003
	ErrCodeMaximumNumberOfRecipientsReached                         = 30004
	ErrCodeMaximumGuildRolesReached                                 = 30005
	ErrCodeMaximumNumberOfWebhooksReached                           = 30007
	ErrCodeMaximumNumberOfEmojisReached                             = 30008
	ErrCodeTooManyReactions                                         = 30010
	ErrCodeMaximumNumberOfGuildChannelsReached                      = 30013
	ErrCodeMaximumNumberOfAttachmentsInAMessageReached              = 30015
	ErrCodeMaximumNumberOfInvitesReached                            = 30016
	ErrCodeMaximumNumberOfAnimatedEmojisReached                     = 30018
	ErrCodeMaximumNumberOfServerMembersReached                      = 30019
	ErrCodeMaximumNumberOfGuildDiscoverySubcategoriesReached        = 30030
	ErrCodeGuildAlreadyHasATemplate                                 = 30031
	ErrCodeMaximumNumberOfThreadParticipantsReached                 = 30033
	ErrCodeMaximumNumberOfBansForNonGuildMembersHaveBeenExceeded    = 30035
	ErrCodeMaximumNumberOfBansFetchesHasBeenReached                 = 30037
	ErrCodeMaximumNumberOfUncompletedGuildScheduledEventsReached    = 30038
	ErrCodeMaximumNumberOfStickersReached                           = 30039
	ErrCodeMaximumNumberOfPruneRequestsHasBeenReached               = 30040
	ErrCodeMaximumNumberOfGuildWidgetSettingsUpdatesHasBeenReached  = 30042
	ErrCodeMaximumNumberOfEditsToMessagesOlderThanOneHourReached    = 30046
	ErrCodeMaximumNumberOfPinnedThreadsInForumChannelHasBeenReached = 30047
	ErrCodeMaximumNumberOfTagsInForumChannelHasBeenReached          = 30048

	ErrCodeUnauthorized                           = 40001
	ErrCodeActionRequiredVerifiedAccount          = 40002
	ErrCodeOpeningDirectMessagesTooFast           = 40003
	ErrCodeSendMessagesHasBeenTemporarilyDisabled = 40004
	ErrCodeRequestEntityTooLarge                  = 40005
	ErrCodeFeatureTemporarilyDisabledServerSide   = 40006
	ErrCodeUserIsBannedFromThisGuild              = 40007
	ErrCodeTargetIsNotConnectedToVoice            = 40032
	ErrCodeMessageAlreadyCrossposted              = 40033
	ErrCodeAnApplicationWithThatNameAlreadyExists = 40041
	ErrCodeInteractionHasAlreadyBeenAcknowledged  = 40060
	ErrCodeTagNamesMustBeUnique                   = 40061

	ErrCodeMissingAccess                                                = 50001
	ErrCodeInvalidAccountType                                           = 50002
	ErrCodeCannotExecuteActionOnDMChannel                               = 50003
	ErrCodeEmbedDisabled                                                = 50004
	ErrCodeGuildWidgetDisabled                                          = 50004
	ErrCodeCannotEditFromAnotherUser                                    = 50005
	ErrCodeCannotSendEmptyMessage                                       = 50006
	ErrCodeCannotSendMessagesToThisUser                                 = 50007
	ErrCodeCannotSendMessagesInVoiceChannel                             = 50008
	ErrCodeChannelVerificationLevelTooHigh                              = 50009
	ErrCodeOAuth2ApplicationDoesNotHaveBot                              = 50010
	ErrCodeOAuth2ApplicationLimitReached                                = 50011
	ErrCodeInvalidOAuthState                                            = 50012
	ErrCodeMissingPermissions                                           = 50013
	ErrCodeInvalidAuthenticationToken                                   = 50014
	ErrCodeTooFewOrTooManyMessagesToDelete                              = 50016
	ErrCodeCanOnlyPinMessageToOriginatingChannel                        = 50019
	ErrCodeInviteCodeWasEitherInvalidOrTaken                            = 50020
	ErrCodeCannotExecuteActionOnSystemMessage                           = 50021
	ErrCodeCannotExecuteActionOnThisChannelType                         = 50024
	ErrCodeInvalidOAuth2AccessTokenProvided                             = 50025
	ErrCodeMissingRequiredOAuth2Scope                                   = 50026
	ErrCodeInvalidWebhookTokenProvided                                  = 50027
	ErrCodeInvalidRole                                                  = 50028
	ErrCodeInvalidRecipients                                            = 50033
	ErrCodeMessageProvidedTooOldForBulkDelete                           = 50034
	ErrCodeInvalidFormBody                                              = 50035
	ErrCodeInviteAcceptedToGuildApplicationsBotNotIn                    = 50036
	ErrCodeInvalidAPIVersionProvided                                    = 50041
	ErrCodeFileUploadedExceedsTheMaximumSize                            = 50045
	ErrCodeInvalidFileUploaded                                          = 50046
	ErrCodeInvalidGuild                                                 = 50055
	ErrCodeInvalidMessageType                                           = 50068
	ErrCodeCannotDeleteAChannelRequiredForCommunityGuilds               = 50074
	ErrCodeInvalidStickerSent                                           = 50081
	ErrCodePerformedOperationOnArchivedThread                           = 50083
	ErrCodeBeforeValueIsEarlierThanThreadCreationDate                   = 50085
	ErrCodeCommunityServerChannelsMustBeTextChannels                    = 50086
	ErrCodeThisServerIsNotAvailableInYourLocation                       = 50095
	ErrCodeThisServerNeedsMonetizationEnabledInOrderToPerformThisAction = 50097
	ErrCodeThisServerNeedsMoreBoostsToPerformThisAction                 = 50101
	ErrCodeTheRequestBodyContainsInvalidJSON                            = 50109

	ErrCodeNoUsersWithDiscordTagExist = 80004

	ErrCodeReactionBlocked = 90001

	ErrCodeAPIResourceIsCurrentlyOverloaded = 130000

	ErrCodeTheStageIsAlreadyOpen = 150006

	ErrCodeCannotReplyWithoutPermissionToReadMessageHistory = 160002
	ErrCodeThreadAlreadyCreatedForThisMessage               = 160004
	ErrCodeThreadIsLocked                                   = 160005
	ErrCodeMaximumNumberOfActiveThreadsReached              = 160006
	ErrCodeMaximumNumberOfActiveAnnouncementThreadsReached  = 160007

	ErrCodeInvalidJSONForUploadedLottieFile                    = 170001
	ErrCodeUploadedLottiesCannotContainRasterizedImages        = 170002
	ErrCodeStickerMaximumFramerateExceeded                     = 170003
	ErrCodeStickerFrameCountExceedsMaximumOfOneThousandFrames  = 170004
	ErrCodeLottieAnimationMaximumDimensionsExceeded            = 170005
	ErrCodeStickerFrameRateOutOfRange                          = 170006
	ErrCodeStickerAnimationDurationExceedsMaximumOfFiveSeconds = 170007

	ErrCodeCannotUpdateAFinishedEvent             = 180000
	ErrCodeFailedToCreateStageNeededForStageEvent = 180002

	ErrCodeCannotEnableOnboardingRequirementsAreNotMet  = 350000
	ErrCodeCannotUpdateOnboardingWhileBelowRequirements = 350001
)

// Intent is the type of a Gateway Intent
// https://discord.com/developers/docs/topics/gateway#gateway-intents
type Intent int

// Constants for the different bit offsets of intents
const (
	IntentGuilds                      Intent = 1 << 0
	IntentGuildMembers                Intent = 1 << 1
	IntentGuildModeration             Intent = 1 << 2
	IntentGuildEmojis                 Intent = 1 << 3
	IntentGuildIntegrations           Intent = 1 << 4
	IntentGuildWebhooks               Intent = 1 << 5
	IntentGuildInvites                Intent = 1 << 6
	IntentGuildVoiceStates            Intent = 1 << 7
	IntentGuildPresences              Intent = 1 << 8
	IntentGuildMessages               Intent = 1 << 9
	IntentGuildMessageReactions       Intent = 1 << 10
	IntentGuildMessageTyping          Intent = 1 << 11
	IntentDirectMessages              Intent = 1 << 12
	IntentDirectMessageReactions      Intent = 1 << 13
	IntentDirectMessageTyping         Intent = 1 << 14
	IntentMessageContent              Intent = 1 << 15
	IntentGuildScheduledEvents        Intent = 1 << 16
	IntentAutoModerationConfiguration Intent = 1 << 20
	IntentAutoModerationExecution     Intent = 1 << 21
	IntentGuildMessagePolls           Intent = 1 << 24
	IntentDirectMessagePolls          Intent = 1 << 25

	// TODO: remove when compatibility is not needed

	IntentGuildBans Intent = IntentGuildModeration

	IntentsGuilds                 Intent = 1 << 0
	IntentsGuildMembers           Intent = 1 << 1
	IntentsGuildBans              Intent = 1 << 2
	IntentsGuildEmojis            Intent = 1 << 3
	IntentsGuildIntegrations      Intent = 1 << 4
	IntentsGuildWebhooks          Intent = 1 << 5
	IntentsGuildInvites           Intent = 1 << 6
	IntentsGuildVoiceStates       Intent = 1 << 7
	IntentsGuildPresences         Intent = 1 << 8
	IntentsGuildMessages          Intent = 1 << 9
	IntentsGuildMessageReactions  Intent = 1 << 10
	IntentsGuildMessageTyping     Intent = 1 << 11
	IntentsDirectMessages         Intent = 1 << 12
	IntentsDirectMessageReactions Intent = 1 << 13
	IntentsDirectMessageTyping    Intent = 1 << 14
	IntentsMessageContent         Intent = 1 << 15
	IntentsGuildScheduledEvents   Intent = 1 << 16

	IntentsAllWithoutPrivileged = IntentGuilds |
		IntentGuildBans |
		IntentGuildEmojis |
		IntentGuildIntegrations |
		IntentGuildWebhooks |
		IntentGuildInvites |
		IntentGuildVoiceStates |
		IntentGuildMessages |
		IntentGuildMessageReactions |
		IntentGuildMessageTyping |
		IntentDirectMessages |
		IntentDirectMessageReactions |
		IntentDirectMessageTyping |
		IntentGuildScheduledEvents |
		IntentAutoModerationConfiguration |
		IntentAutoModerationExecution

	IntentsAll = IntentsAllWithoutPrivileged |
		IntentGuildMembers |
		IntentGuildPresences |
		IntentMessageContent

	IntentsNone Intent = 0
)

// MakeIntent used to help convert a gateway intent value for use in the Identify structure;
// this was useful to help support the use of a pointer type when intents were optional.
// This is now a no-op, and is not necessary to use.
func MakeIntent(intents Intent) Intent {
	return intents
}
