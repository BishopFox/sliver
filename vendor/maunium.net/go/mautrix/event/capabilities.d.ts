/**
 * The content of the `com.beeper.room_features` state event.
 */
export interface RoomFeatures {
	/**
	 * Supported formatting features. If omitted, no formatting is supported.
	 *
	 * Capability level 0 means the corresponding HTML tags/attributes are ignored
	 * and will be treated as if they don't exist, which means that children will
	 * be rendered, but attributes will be dropped.
	 */
	formatting?: Record<FormattingFeature, CapabilitySupportLevel>
	/**
	 * Supported file message types and their features.
	 *
	 * If a message type isn't listed here, it should be treated as support level -2 (will be rejected).
	 */
	file?: Record<CapabilityMsgType, FileFeatures>
	/**
	 * Supported state event types and their parameters. Currently, there are no parameters,
	 * but it is likely there will be some in the future (like max name/topic length, avatar mime types, etc.).
	 *
	 * Events that are not listed or have a support level of zero or below should be treated as unsupported.
	 *
	 * Clients should at least check `m.room.name`, `m.room.topic`, and `m.room.avatar` here.
	 * `m.room.member` will not be listed here, as it's controlled by the member_actions field.
	 * `com.beeper.disappearing_timer` should be listed here, but the parameters are in the disappearing_timer field for now.
	 */
	state?: Record<EventType, StateFeatures>
	/**
	 * Supported member actions and their support levels.
	 *
	 * Actions that are not listed or have a support level of zero or below should be treated as unsupported.
	 */
	member_actions?: Record<MemberAction, CapabilitySupportLevel>

	/** Maximum length of normal text messages. */
	max_text_length?: integer

	/** Whether location messages (`m.location`) are supported. */
	location_message?: CapabilitySupportLevel
	/** Whether polls are supported. */
	poll?: CapabilitySupportLevel
	/** Whether replying in a thread is supported. */
	thread?: CapabilitySupportLevel
	/** Whether replying to a specific message is supported. */
	reply?: CapabilitySupportLevel

	/** Whether edits are supported. */
	edit?: CapabilitySupportLevel
	/** How many times can an individual message be edited. */
	edit_max_count?: integer
	/** How old messages can be edited, in seconds. */
	edit_max_age?: seconds
	/** Whether deleting messages for everyone is supported */
	delete?: CapabilitySupportLevel
	/** How old messages can be deleted for everyone, in seconds. */
	delete_max_age?: seconds
	/** Whether deleting messages just for yourself is supported. No message age limit. */
	delete_for_me?: boolean
	/** Allowed configuration options for disappearing timers. */
	disappearing_timer?: DisappearingTimerCapability

	/** Whether reactions are supported. */
	reaction?: CapabilitySupportLevel
	/** How many reactions can be added to a single message. */
	reaction_count?: integer
	/**
	 * The Unicode emojis allowed for reactions. If omitted, all emojis are allowed.
	 * Emojis in this list must include variation selector 16 if allowed in the Unicode spec.
	 */
	allowed_reactions?: string[]
	/** Whether custom emoji reactions are allowed. */
	custom_emoji_reactions?: boolean

	/** Whether deleting the chat for yourself is supported. */
	delete_chat?: boolean
	/** Whether deleting the chat for all participants is supported. */
	delete_chat_for_everyone?: boolean
}

declare type integer = number
declare type seconds = integer
declare type milliseconds = integer
declare type MIMEClass = "image" | "audio" | "video" | "text" | "font" | "model" | "application"
declare type MIMETypeOrPattern =
	"*/*"
	| `${MIMEClass}/*`
	| `${MIMEClass}/${string}`
	| `${MIMEClass}/${string}; ${string}`

export enum MemberAction {
	Ban = "ban",
	Kick = "kick",
	Leave = "leave",
	RevokeInvite = "revoke_invite",
	Invite = "invite",
}

declare type EventType = string

// This is an object for future extensibility (e.g. max name/topic length)
export interface StateFeatures {
	level: CapabilitySupportLevel
}

export enum CapabilityMsgType {
	// Real message types used in the `msgtype` field
	Image = "m.image",
	File = "m.file",
	Audio = "m.audio",
	Video = "m.video",

	// Pseudo types only used in capabilities
	/** An `m.audio` message that has `"org.matrix.msc3245.voice": {}` */
	Voice = "org.matrix.msc3245.voice",
	/** An `m.video` message that has `"info": {"fi.mau.gif": true}`, or an `m.image` message of type `image/gif` */
	GIF = "fi.mau.gif",
	/** An `m.sticker` event, no `msgtype` field */
	Sticker = "m.sticker",
}

export interface FileFeatures {
	/**
	 * The supported MIME types or type patterns and their support levels.
	 *
	 * If a mime type doesn't match any pattern provided,
	 * it should be treated as support level -2 (will be rejected).
	 */
	mime_types: Record<MIMETypeOrPattern, CapabilitySupportLevel>

	/** The support level for captions within this file message type */
	caption?: CapabilitySupportLevel
	/** The maximum length for captions (only applicable if captions are supported). */
	max_caption_length?: integer
	/** The maximum file size as bytes. */
	max_size?: integer
	/** For images and videos, the maximum width as pixels. */
	max_width?: integer
	/** For images and videos, the maximum height as pixels. */
	max_height?: integer
	/** For videos and audio files, the maximum duration as seconds. */
	max_duration?: seconds

	/** Can this type of file be sent as view-once media? */
	view_once?: boolean
}

export enum DisappearingType {
	None = "",
	AfterRead = "after_read",
	AfterSend = "after_send",
}

export interface DisappearingTimerCapability {
	types: DisappearingType[]
	/** Allowed timer values. If omitted, any timer is allowed. */
	timers?: milliseconds[]
	/**
	 * Whether clients should omit the empty disappearing_timer object in messages that they don't want to disappear
	 *
	 * Generally, bridged rooms will want the object to be always present, while native Matrix rooms don't,
	 * so the hardcoded features for Matrix rooms should set this to true, while bridges will not.
	 */
	omit_empty_timer?: true
}

/**
 * The support level for a feature. These are integers rather than booleans
 * to accurately represent what the bridge is doing and hopefully make the
 * state event more generally useful. Our clients should check for > 0 to
 * determine if the feature should be allowed.
 */
export enum CapabilitySupportLevel {
	/** The feature is unsupported and messages using it will be rejected. */
	Rejected = -2,
	/** The feature is unsupported and has no fallback. The message will go through, but data may be lost. */
	Dropped = -1,
	/** The feature is unsupported, but may have a fallback. The nature of the fallback depends on the context. */
	Unsupported = 0,
	/** The feature is partially supported (e.g. it may be converted to a different format). */
	PartialSupport = 1,
	/** The feature is fully supported and can be safely used. */
	FullySupported = 2,
}

/**
 * A formatting feature that consists of specific HTML tags and/or attributes.
 */
export enum FormattingFeature {
	Bold = "bold", // strong, b
	Italic = "italic", // em, i
	Underline = "underline", // u
	Strikethrough = "strikethrough", // del, s
	InlineCode = "inline_code", // code
	CodeBlock = "code_block", // pre + code
	SyntaxHighlighting = "code_block.syntax_highlighting", // <pre><code class="language-...">
	Blockquote = "blockquote", // blockquote
	InlineLink = "inline_link", // a
	UserLink = "user_link", // <a href="https://matrix.to/#/@...">
	RoomLink = "room_link", // <a href="https://matrix.to/#/#...">
	EventLink = "event_link", // <a href="https://matrix.to/#/!.../$...">
	AtRoomMention = "at_room_mention", // @room (no html tag)
	UnorderedList = "unordered_list", // ul + li
	OrderedList = "ordered_list", // ol + li
	ListStart = "ordered_list.start", // <ol start="N">
	ListJumpValue = "ordered_list.jump_value", // <li value="N">
	CustomEmoji = "custom_emoji", // <img data-mx-emoticon>
	Spoiler = "spoiler", // <span data-mx-spoiler>
	SpoilerReason = "spoiler.reason", // <span data-mx-spoiler="...">
	TextForegroundColor = "color.foreground", // <span data-mx-color="#...">
	TextBackgroundColor = "color.background", // <span data-mx-bg-color="#...">
	HorizontalLine = "horizontal_line", // hr
	Headers = "headers", // h1, h2, h3, h4, h5, h6
	Superscript = "superscript", // sup
	Subscript = "subscript", // sub
	Math = "math", // <span data-mx-maths="...">
	DetailsSummary = "details_summary", // <details><summary>...</summary>...</details>
	Table = "table", // table, thead, tbody, tr, th, td
}
