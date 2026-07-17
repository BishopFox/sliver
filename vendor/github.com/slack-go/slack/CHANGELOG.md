# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.24.0]

### Added

- Block Kit: `DataTableBlock` for the [`data_table`](https://docs.slack.dev/reference/block-kit/blocks/data-table-block/)
  block, with `NewDataTableBlock`, `AddRow`, raw-text/raw-number/rich-text cell
  constructors, and `WithPageSize` / `WithRowHeaderColumnIndex` builders.

### Changed

- `NewTaskCardBlock` and `NewPlanBlock` nil-guard their variadic options,
  matching the other block constructors (#1236).

## [0.23.1] - 2026-05-10

### Fixed

- `NewSecretsVerifier` now rejects empty signing secrets to avoid accepting forged request
  signatures when applications are misconfigured.

## [0.23.0] - 2026-04-22

### Added

- **Block Kit: `CardBlock` and `CarouselBlock`** — Support for two of the new
  agent-UI blocks announced in the
  [April 16 Slack changelog](https://docs.slack.dev/changelog/2026/04/16/block-kit-new-blocks).
  `CardBlock` is constructed via `NewCardBlock` with a functional-options
  pattern and fluent `With*` builders (`WithTitle`, `WithSubtitle`, `WithBody`,
  `WithIcon`, `WithHeroImage`, `WithActions`). `CarouselBlock` is constructed
  via `NewCarouselBlock` with a variadic `*CardBlock` list plus `WithBlockID`
  and `AddCard` helpers. Both blocks wire into `Blocks.UnmarshalJSON` for
  round-trip fidelity, and reuse existing `ImageBlockElement` /
  `ButtonBlockElement` / `BlockElements` types rather than introducing new
  composition objects.
- **Block Kit: `AlertBlock`** — Support for the third of the new agent-UI
  blocks from the
  [April 16 Slack changelog](https://docs.slack.dev/changelog/2026/04/16/block-kit-new-blocks).
  `AlertBlock` is constructed via `NewAlertBlock` with a `*TextBlockObject`
  body and a functional-options pattern. Severity is set via
  `AlertBlockOptionLevel` (`AlertLevelDefault`, `AlertLevelInfo`,
  `AlertLevelWarning`, `AlertLevelError`, `AlertLevelSuccess`) and the block
  ID via `AlertBlockOptionBlockID`. Wires into `Blocks.UnmarshalJSON` for
  round-trip fidelity. Must be delivered via the streaming chunks API —
  `chat.postMessage` rejects it as an unsupported block type.
- **Streaming-message chunks API** — `chat.startStream` / `chat.appendStream` /
  `chat.stopStream` now accept a `chunks` parameter. Added `MsgOptionChunks`
  along with a `StreamChunk` interface and four chunk types:
  `MarkdownTextChunk`, `TaskUpdateChunk`, `PlanUpdateChunk`, and `BlocksChunk`
  (each with a `New*Chunk` constructor). This is the supported transport for
  streaming Block Kit content and the new agent-UI blocks in particular
  (which `chat.postMessage` rejects as `Unsupported block type`).
- **`MsgOptionTaskDisplayMode`** — New option for `chat.startStream` controlling
  whether task chunks render as a sequential timeline or a grouped plan.
  Accepts `TaskDisplayModeTimeline` or `TaskDisplayModePlan`.
- Added `Username`, `IconURL`, and `IconEmoji` fields to
  `AssistantThreadsSetStatusParameters`, forwarded by
  `SetAssistantThreadsStatusContext`, matching the new optional parameters on
  [`assistant.threads.setStatus`](https://docs.slack.dev/reference/methods/assistant.threads.setStatus)
  for customising the status-update presentation.
- Exposed `SocketmodeHandler.DispatchEvent` (previously the unexported
  `dispatcher`), enabling integration tests to exercise registered handlers
  without a live WebSocket connection. The unexported `dispatcher` is kept as
  a thin wrapper for backwards compatibility. Closes #1549.

## [0.22.0] - 2026-04-12

### Added

- Added missing parameters to `assistant.search.context` (`Sort`, `SortDir`, `Before`,
  `After`, `Highlight`, `IncludeContextMessages`, `IncludeDeletedUsers`,
  `IncludeMessageBlocks`, `IncludeArchivedChannels`, `DisableSemanticSearch`, `Modifiers`,
  `TermClauses`) and new response types (`AssistantSearchContextFile`,
  `AssistantSearchContextChannel`, `AssistantSearchContextMessageContext`) to match the
  full Real-Time Search API surface.
- Added `Underline`, `Highlight`, `ClientHighlight`, and `Unlink` fields to
  `RichTextSectionTextStyle`. Added `Style` field to `RichTextSectionUserGroupElement`.
- Added `BotOptional` and `UserOptional` fields to `OAuthScopes` for app manifests.
- Added PKCE support for OAuth: `OAuthOptionCodeVerifier` option for
  `GetOAuthV2Response`, `GenerateCodeVerifier()` and `GenerateCodeChallenge()`
  helper functions (RFC 7636). `client_secret` is now conditionally omitted when
  empty in both `GetOAuthV2ResponseContext` and `RefreshOAuthV2TokenContext`.

### Fixed

- `ChannelTypes` and `ContentTypes` now send comma-separated values instead of repeated
  form keys, matching the convention used by every other method in the library.
- In `socketmode` malformed JSON messages no longer force an unnecessary reconnect.
  Instead the error is emitted and the connection continues as normal.

## [0.21.1] - 2026-04-08

### Added

- **`slackevents.ChannelType*` constants and `MessageEvent` helpers** — Added
  `ChannelTypeChannel`, `ChannelTypeGroup`, `ChannelTypeIM`, `ChannelTypeMPIM` constants
  and `IsChannel()`, `IsGroup()`, `IsIM()`, `IsMpIM()` methods on `MessageEvent` so
  callers no longer need to compare raw strings.

### Fixed

- **Duplicate attachment/block serialization in `MsgOptionAttachments` / `MsgOptionBlocks`** —
  Attachments and blocks were serialized twice: once into typed struct fields (for the JSON
  response-URL path) and again into `url.Values` (for the form POST path). Serialization for
  the form path now happens inside `formSender.BuildRequestContext`, so each sender owns its
  own marshalling. This fixes the long-standing FIXME and eliminates redundant `json.Marshal`
  calls in the option functions. ([#1547])

  > [!NOTE]
  > `UnsafeApplyMsgOptions` returns `config.values` directly. After this change,
  > `attachments` and `blocks` keys are no longer present in those values since
  > marshalling is deferred to send time. This function is documented as unsupported.

## [0.21.0] - 2026-04-05

### Deprecated

- **`slackevents.ParseActionEvent`** — Cannot parse `block_actions` payloads (returns
  unmarshalling error). Use `slack.InteractionCallback` with `json.Unmarshal` instead,
  or `slack.InteractionCallbackParse` for HTTP requests. `InteractionCallback` handles
  all interaction types. ([#596])
- **`slackevents.MessageAction`**, **`MessageActionEntity`**, **`MessageActionResponse`** —
  Associated types that only support legacy `interactive_message` payloads.

### Removed

- **`IM` struct** — Removed the `IM` struct (and unused internal types `imChannel`,
  `imResponseFull`). The `IsUserDeleted` field has been moved to `Conversation`, where it
  is populated for IM-type conversations. Code using `IM` should switch to `Conversation`.

  > [!NOTE]
  > In practice no user should be affected — `IM` was never returned by any public API
  > method in this library, so there was no way to obtain one outside of manual construction.

- **`Info.GetBotByID`, `GetUserByID`, `GetChannelByID`, `GetGroupByID`, `GetIMByID`** —
  These methods were deprecated and returned `nil` unconditionally. They have been removed.

  > [!WARNING]
  > **Breaking change.** If you are calling any of these methods, remove those calls — they
  > were already no-ops.

### Added

- **`admin.teams.settings.*` API support** — `AdminTeamsSettingsInfo`,
  `AdminTeamsSettingsSetDefaultChannels`, `AdminTeamsSettingsSetDescription`,
  `AdminTeamsSettingsSetDiscoverability`, `AdminTeamsSettingsSetIcon`, and
  `AdminTeamsSettingsSetName`. Includes `TeamDiscoverability` enum with `Open`,
  `InviteOnly`, `Closed`, and `Unlisted` variants. ([#960])
- **`OAuthOptionAPIURL` for package-level OAuth functions** — All package-level OAuth
  functions (`GetOAuthV2Response`, `GetOpenIDConnectToken`, `RefreshOAuthV2Token`, etc.)
  now accept variadic `OAuthOption` arguments. Use `OAuthOptionAPIURL(url)` to override
  the default Slack API URL, enabling integration tests against a local HTTP server.
  Existing callers are unaffected. ([#744])
- **`GetOpenIDConnectUserInfo` / `GetOpenIDConnectUserInfoContext`** — Returns identity
  information about the user associated with the token via `openid.connect.userInfo`.
  Complements the existing `GetOpenIDConnectToken` method. ([#967])
- **HTTP response headers** — Slack API response headers (e.g. `X-OAuth-Scopes`,
  `X-Accepted-OAuth-Scopes`, `X-Ratelimit-*`) are now accessible. `AuthTestResponse`
  exposes a `Header` field directly. For all other methods, use
  `OptionOnResponseHeaders(func(method string, headers http.Header))` to register a
  callback that fires after every API call. ([#1076])
- **`DNDOptionTeamID`** — `GetDNDInfo` and `GetDNDTeamInfo` now accept optional
  `ParamOption` arguments. Use `DNDOptionTeamID("T...")` to pass `team_id`, which is
  required after workspace migration (Slack returns `missing_argument` without it).
  ([#1157])
- **`UpdateUserGroupMembersList` / `UpdateUserGroupMembersListContext`** — Convenience
  wrappers around `UpdateUserGroupMembers` that accept `[]string` instead of a
  comma-separated string, enabling clean chaining with `GetUserGroupMembers`. ([#1172])
- **`SetUserProfile` / `SetUserProfileContext`** — Set multiple user profile fields in a
  single API call by passing a `*UserProfile` struct to `users.profile.set`. Complements
  the existing single-field methods (`SetUserRealName`, `SetUserCustomStatus`, etc.).
  ([#1158])
- **API warning callbacks** — Slack API responses may include a `warnings` field with
  deprecation notices or usage hints. Use `OptionWarnings(func(warnings []string))` to
  register a callback that receives these warnings. ([#1540])
- **RTM support for `user_status_changed`, `user_huddle_changed`, `user_profile_changed`
  events** — these events are now mapped in `EventMapping` with dedicated structs
  (`UserStatusChangedEvent`, `UserHuddleChangedEvent`, `UserProfileChangedEvent`).
  Previously they triggered `UnmarshallingErrorEvent`. ([#1541])
- **RTM support for `sh_room_join`, `sh_room_leave`, `sh_room_update`, `channel_updated`
  events** — Slack Call/Huddle room events and channel property updates are now mapped with
  dedicated structs (`SHRoomJoinEvent`, `SHRoomLeaveEvent`, `SHRoomUpdateEvent`,
  `ChannelUpdatedEvent`). ([#858])
- **`CacheTS` and `EventTS` fields on `UserChangeEvent`** — these fields were sent by Slack
  but silently dropped during unmarshalling.
- **`workflows.featured` API support** — add, list, remove, and set featured workflows on
  channels via `WorkflowsFeaturedAdd`, `WorkflowsFeaturedList`, `WorkflowsFeaturedRemove`,
  and `WorkflowsFeaturedSet`
- **`IsConnectorBot` and `IsWorkflowBot` in `User`** — boolean flags for connector and
  workflow bot users
- **`GuestInvitedBy` in `UserProfile`** — user ID of whoever invited a guest user
- **`Blocks` field on `MessageEvent`** — block data from webhook payloads is now directly
  accessible via `event.Blocks` instead of only through `event.Message.Blocks`. ([#1257])
- **`Username` field on `User`** — Slack's interaction payloads (block_actions, shortcuts)
  include a `username` field in the user object that was previously dropped during
  unmarshalling. ([#1218])
- **`Blocks`, `Attachments`, `Files`, `Upload` fields on `AppMentionEvent`** — these fields
  are sent by Slack in `app_mention` event payloads but were silently dropped. ([#961])
- **`HandleShortcut`, `HandleViewSubmission`, `HandleViewClosed` in socketmode handler** —
  Level 3 handlers that dispatch `shortcut`/`message_action`, `view_submission`, and
  `view_closed` interactions by `CallbackID`, matching the pattern of
  `HandleInteractionBlockAction` and `HandleSlashCommand`. ([#1161])
- **`BlockFromJSON` / `MustBlockFromJSON`** — Create blocks from raw JSON strings, enabling
  direct use of output from Slack's [Block Kit Builder](https://app.slack.com/block-kit-builder)
  or quick adoption of new block types before the library adds typed support. The original
  JSON is preserved through marshalling. ([#1497])

### Documentation

- **`ViewSubmissionResponse` constructors** — `NewClearViewSubmissionResponse`,
  `NewUpdateViewSubmissionResponse`, `NewPushViewSubmissionResponse`, and
  `NewErrorsViewSubmissionResponse` now have doc comments explaining the HTTP response
  pattern (write JSON and return promptly) and the Socket Mode pattern (pass as Ack
  payload). `NewErrorsViewSubmissionResponse` additionally documents that map keys must
  be `BlockID`s of `InputBlock` elements. ([#726], [#1013])
- **Socket Mode examples** — `examples/socketmode/` and `examples/socketmode_handler/` now
  have doc comments explaining the two-token requirement: app-level token (`xapp-`) for the
  WebSocket connection and bot token (`xoxb-`) for API calls. ([#941])

### Fixed

- **`UnknownBlock` round-trip data loss** — Unrecognized block types (e.g. new Slack block
  types not yet supported by this library) now preserve their full JSON through
  unmarshal/marshal cycles. Previously only `type` and `block_id` were retained, silently
  discarding all other fields.

### Changed

- Adjusted some `admin` errors that started with uppercase to be lowercase per go
  conventions.

  > [!WARNING]
  > **Breaking change.** If you are matching the error content in your code, this is a
  > BREAKING CHANGE.
- **`WebhookMessage.UnfurlLinks` and `UnfurlMedia` are now `*bool`** — Previously these
  were `bool` with `omitempty`, which meant `false` was silently stripped from the JSON
  payload. Users could not explicitly disable link or media unfurling via webhooks. The
  fields are now `*bool` so that `nil` (omit), `false`, and `true` all serialize correctly.
  ([#1231])

  > [!WARNING]
  > **Breaking change.** Code that sets these fields directly must be updated:
  >
  > ```go
  > // Before
  > msg := slack.WebhookMessage{UnfurlLinks: true}
  >
  > // After — use a helper or a variable
  > t := true
  > msg := slack.WebhookMessage{UnfurlLinks: &t}
  > ```
  >
  > Leaving the fields unset (`nil`) preserves the previous default behavior — Slack's
  > server-side defaults apply (`unfurl_links=false`, `unfurl_media=true`).

- **`User.Has2FA` is now `*bool`** — When using a bot token, Slack's `users.list` API omits
  `has_2fa` entirely. With a plain `bool`, this was indistinguishable from explicitly `false`.
  Now `nil` means absent/unknown, `false` means explicitly disabled, `true` means enabled.
  ([#1121])

  > [!WARNING]
  > **Breaking change.** Code that reads `Has2FA` must handle the pointer:
  >
  > ```go
  > // Before
  > if user.Has2FA { ... }
  >
  > // After
  > if user.Has2FA != nil && *user.Has2FA { ... }
  > ```

- **`ListReactions` now uses cursor-based pagination** — `ListReactionsParameters` replaces
  `Count`/`Page` with `Cursor`/`Limit`, and `ListReactions`/`ListReactionsContext` now return
  `([]ReactedItem, string, error)` where the string is the next cursor, instead of
  `([]ReactedItem, *Paging, error)`. ([#825])

  > [!WARNING]
  > **Breaking change.** Both the parameters and return signature have changed:
  >
  > ```go
  > // Before
  > params := slack.NewListReactionsParameters()
  > params.Count = 100
  > params.Page = 2
  > items, paging, err := api.ListReactions(params)
  >
  > // After
  > params := slack.NewListReactionsParameters()
  > params.Limit = 100
  > items, nextCursor, err := api.ListReactions(params)
  > // Use nextCursor for the next page: params.Cursor = nextCursor
  > ```

- **`ListStars`/`GetStarred` now use cursor-based pagination** — `StarsParameters` replaces
  `Count`/`Page` with `Cursor`/`Limit` (and adds `TeamID`), and `ListStars`/`ListStarsContext`/
  `GetStarred`/`GetStarredContext` now return `string` (next cursor) instead of `*Paging`.
  Slack's `stars.list` API no longer returns `paging` data — only `response_metadata.next_cursor`.

  > [!WARNING]
  > **Breaking change.** Both the parameters and return signature have changed:
  >
  > ```go
  > // Before
  > params := slack.NewStarsParameters()
  > params.Count = 100
  > params.Page = 2
  > items, paging, err := api.ListStars(params)
  >
  > // After
  > params := slack.NewStarsParameters()
  > params.Limit = 100
  > items, nextCursor, err := api.ListStars(params)
  > // Use nextCursor for the next page: params.Cursor = nextCursor
  > ```

- **`GetAccessLogs` now uses cursor-based pagination** — `AccessLogParameters` replaces
  `Count`/`Page` with `Cursor`/`Limit` (and adds `Before`), and `GetAccessLogs`/
  `GetAccessLogsContext` now return `string` (next cursor) instead of `*Paging`.
  Slack's `team.accessLogs` API warns `use_cursor_pagination_instead` when using the old
  parameters.

  > [!WARNING]
  > **Breaking change.** Both the parameters and return signature have changed:
  >
  > ```go
  > // Before
  > params := slack.NewAccessLogParameters()
  > params.Count = 100
  > params.Page = 2
  > logins, paging, err := api.GetAccessLogs(params)
  >
  > // After
  > params := slack.NewAccessLogParameters()
  > params.Limit = 100
  > logins, nextCursor, err := api.GetAccessLogs(params)
  > // Use nextCursor for the next page: params.Cursor = nextCursor
  > ```

### Fixed

- **Socket Mode: large Ack payloads no longer silently fail** — Two issues caused `Ack()`
  payloads to be silently dropped by Slack. First, gorilla/websocket's default 4KB write
  buffer fragmented messages into WebSocket continuation frames that Slack does not
  reassemble. The library now uses a 32KB write buffer. Second, Slack silently drops
  Socket Mode responses at or above 20KB — `Ack()`, `Send()`, and `SendCtx()` now return
  an error when the serialized response reaches this limit. ([#1196])

  > [!WARNING]
  > **Breaking change.** `Ack()` and `Send()` now return `error`. Existing call sites that
  > don't capture the return value continue to compile without changes.

- **`MsgOptionBlocks()` with no arguments now sends `blocks=[]`** — Previously, calling
  `MsgOptionBlocks()` with no arguments or a nil spread was a silent no-op, making it
  impossible to clear blocks from a message via `chat.update`. The Slack API requires an
  explicit `blocks=[]` to reliably remove blocks. ([#1214])

  > [!WARNING]
  > **Breaking change.** `MsgOptionBlocks()` with no arguments now sends `blocks=[]` instead
  > of being a no-op. If you were relying on this to be a no-op, remove the option entirely:
  >
  > ```go
  > // Before — this was a no-op, now it sends blocks=[]
  > api.PostMessage(ch, slack.MsgOptionBlocks(), slack.MsgOptionText("text", false))
  >
  > // After — omit MsgOptionBlocks entirely to not set blocks
  > api.PostMessage(ch, slack.MsgOptionText("text", false))
  > ```

- **`WorkflowButtonBlockElement` missing from `UnmarshalJSON`** — `workflow_button` blocks
  now unmarshal correctly through `BlockElements`, `InputBlock`, and `Accessory` paths.
  Also adds missing `multi_*_select` and `file_input` cases to `BlockElements.UnmarshalJSON`,
  and fixes `toBlockElement` for `RichTextInputElement` and `WorkflowButtonElement`. ([#1539])
- **`NewBlockHeader` nil pointer dereference** — passing a nil text object no longer panics. ([#1236])
- **`ValidateUniqueBlockID` rejects empty block IDs** — multiple input blocks with no
  explicit `block_id` set (empty string) were incorrectly flagged as duplicates, causing
  `OpenView` to fail. ([#1184])

## [0.20.0] - 2026-03-21

> [!WARNING]
> `trigger_id` and `workflow_id` are NOT in any documentation or in any of the official
libraries, so exercise caution if you use these.

### Added

- **`workflow_id` and `trigger_id` in `Message`** — It seems that some types of messages,
    e.g: `bot_message`, can carry `trigger_id` and `workflow_id`.
- **`RichTextQuote.Border` field** — optional border toggle (matches the docs now)
- **`RichTextPreformatted.Language` field** — enables syntax highlighting for preformatted
  blocks

### Fixed

- **Remove embedding of `RichTextSection`** — `RichTextQuote` and `RichTextPreformatted`
  are now flattened as they should have always been. This is a breaking change for anyone
  using these structs directly.

## [0.19.0] - 2026-03-04

### Added

- **Optional HTTP retry for Web API** — Retries are off by default. Enable with `OptionRetry(n)` for 429-only retries or `OptionRetryConfig(cfg)` for full control including 5xx and connection errors with exponential backoff. ([#1532])
- **`task_card` and `plan` agent blocks** — New block types for task cards and plan agent blocks. ([#1536])

### Changed

- CI: bumped `actions/stale` from 10.1.1 to 10.2.0. ([#1534])
- Use `golangci-lint` in Makefile. ([#1533])

## [0.18.0] - 2026-02-21

### Added

- **`focus_on_load` support for remaining block elements** — Static/external/users/conversations/channels select, multi-select variants, datepicker, timepicker, plain_text_input, checkboxes, radio_buttons, and number_input. ([#1519])
- **`PlainText` and `PreviewPlainText` fields on `File`** — Email file objects now include the plain text body fields instead of silently discarding them. ([#1522])
- **Missing fields on `User`, `UserProfile`, and `EnterpriseUser`** — `who_can_share_contact_card`, `always_active`, `pronouns`, `image_1024`, `is_custom_image`, `status_text_canonical`, `huddle_state`, `huddle_state_expiration_ts`, `start_date`, and `is_primary_owner`. ([#1526])
- **Work Objects support** — Chat unfurl with Work Object metadata, entity details (flexpane), `entity_details_requested` event, and associated types (`WorkObjectMetadata`, `WorkObjectEntity`, `WorkObjectExternalRef`). ([#1529])
- **`admin.roles.*` API methods** — `admin.roles.listAssignments`, `admin.roles.addAssignments`, and `admin.roles.removeAssignments`. ([#1520])

### Fixed

- **`UserProfile.Skype` JSON tag** — Corrected typo from `"skyp"` to `"skype"`. ([#1524])
- **`assistant.threads.setSuggestedPrompts` title parameter** — Title is now sent when non-empty. ([#1528])

### Changed

- CI test matrix updated: dropped Go 1.24, added Go 1.26; bumped golangci-lint to v2.10.1. ([#1530])

## [0.18.0-rc2] - 2026-01-28

### Added

- **Audit Logs example** - New example demonstrating how to use the Audit Logs API. ([#1144])
- **Admin Conversations API support** - Comprehensive support for `admin.conversations.*`
  methods including core operations (archive, unarchive, create, delete, rename, invite,
  search, lookup, getTeams, convertToPrivate, convertToPublic, disconnectShared, setTeams),
  bulk operations (bulkArchive, bulkDelete, bulkMove), preferences, retention management,
  restrict access controls, and EKM channel info. ([#1329])

### Changed

- **BREAKING**: Removed deprecated `UploadFile`, `UploadFileContext`, and
  `FileUploadParameters`. The `files.upload` API was discontinued by Slack on November
  12, 2025. ([#1481])
- **BREAKING**: Renamed `UploadFileV2` → `UploadFile`, `UploadFileV2Context` →
  `UploadFileContext`, and `UploadFileV2Parameters` → `UploadFileParameters`. The "V2"
  suffix is no longer needed now that the old API is removed. ([#1481])

### Fixed

- **File upload error wrapping** - `UploadFile` now wraps errors with the step name
  (`GetUploadURLExternal`, `UploadToURL`, or `CompleteUploadExternal`) so callers can
  identify which of the three upload steps failed. ([#1491])
- **Audit Logs API endpoint** - Fixed `GetAuditLogs` to use the correct endpoint
  (`api.slack.com`) instead of the regular API endpoint (`slack.com/api`). The Audit
  Logs API requires a different base URL. Added `OptionAuditAPIURL` for testing. ([#1144])
- **Socket mode websocket dial debugging** - Added debug logging when a custom dialer is
  used including HTTP response status on dial failures. This helps diagnose proxy/TLS
  issues like "bad handshake" errors. ([#1360])
- **`MsgOptionPostMessageParameters` now passes `MetaData`** - Previously, metadata was
  silently dropped when using `PostMessageParameters`. ([#1343])

## [0.18.0-rc1] - 2026-01-26

### Added

- **Huddle support** - New `HuddleRoom`, `HuddleParticipantEvent`, and `HuddleRecording`
  types for handling Slack huddle events (`huddle_thread` subtype messages).
- **Call block data parsing** - `CallBlock` now includes full call data when retrieved
  from Slack messages, with new `CallBlockData`, `CallBlockDataV1`, and `CallBlockIconURLs`
  types. ([#897])
- **Chat Streaming API support** - New streaming API for real-time chat interactions
  with example usage. ([#1506])
- **Data Access API support** - Full support for Slack's Data Access API with
  example implementation. ([#1439])
- **Cursor-based pagination for `GetUsers`** - More efficient user retrieval
  with cursor pagination. ([#1465])
- **`GetAllConversations` with pagination** - Retrieve all conversations with
  automatic pagination handling, including rate limit and server error handling. ([#1463])
- **Table blocks support** - Parse and create table blocks with proper
  unmarshaling. ([#1490], [#1511])
- **Context actions block support** - New `context_actions` block type. ([#1495])
- **Workflow button block element** - Support for `workflow_button` in block
  elements. ([#1499])
- **`loading_messages` parameter for `SetAssistantThreadsStatus`** - Optional
  parameter to customize loading state messages. ([#1489])
- **Attachment image fields** - Added `ImageBytes`, `ImageHeight`, and `ImageWidth`
  fields to attachments. ([#1516])
- **`RecordChannel` to conversation properties** - New property for conversation
  metadata. ([#1513])
- **Title argument for `CreateChannelCanvas`** - Canvas creation now supports
  custom titles. ([#1483])
- **`PostEphemeral` handler for slacktest** - Audit outgoing ephemeral messages
  in test environments. ([#1517])
- **`PreviewImageName` for remote files** - Customize preview image filename
  instead of using the default `preview.jpg`.

### Fixed

- **`PublishView` no longer sends empty hash** - Prevents unnecessary payload
  when hash is empty. ([#1515])
- **`ImageBlockElement` validation** - Now properly validates that either
  `imageURL` or `SlackFile` is provided. ([#1488])
- **Rich text section channel return** - Correctly returns channel for section
  channel rich text elements. ([#1472])
- **`KickUserFromConversation` error handling** - Errors are now properly parsed
  as a map structure. ([#1471])

### Changed

- **BREAKING**: `GetReactions` now returns `ReactedItem` instead of `[]ItemReaction`.
  This aligns the response with the actual Slack API, which includes the item itself
  (message, file, or file_comment) alongside reactions. To migrate, use `resp.Reactions`
  to access the slice of reactions. ([#1480])
- **BREAKING**: `Settings` struct fields `Interactivity` and `EventSubscriptions`
  are now pointers, allowing them to be omitted when empty. ([#1461])
- Minimum Go version bumped to 1.24. ([#1504])

## [0.17.3] - 2025-07-04

Previous release. See [GitHub releases](https://github.com/slack-go/slack/releases/tag/v0.17.3)
for details.

[#897]: https://github.com/slack-go/slack/issues/897
[#1236]: https://github.com/slack-go/slack/issues/1236
[#1257]: https://github.com/slack-go/slack/issues/1257
[#1144]: https://github.com/slack-go/slack/issues/1144
[#1329]: https://github.com/slack-go/slack/issues/1329
[#1343]: https://github.com/slack-go/slack/issues/1343
[#1360]: https://github.com/slack-go/slack/issues/1360
[#1439]: https://github.com/slack-go/slack/pull/1439
[#1461]: https://github.com/slack-go/slack/pull/1461
[#1463]: https://github.com/slack-go/slack/pull/1463
[#1465]: https://github.com/slack-go/slack/pull/1465
[#1471]: https://github.com/slack-go/slack/pull/1471
[#1472]: https://github.com/slack-go/slack/pull/1472
[#1480]: https://github.com/slack-go/slack/pull/1480
[#1483]: https://github.com/slack-go/slack/pull/1483
[#1488]: https://github.com/slack-go/slack/pull/1488
[#1489]: https://github.com/slack-go/slack/pull/1489
[#1490]: https://github.com/slack-go/slack/pull/1490
[#1491]: https://github.com/slack-go/slack/issues/1491
[#1495]: https://github.com/slack-go/slack/pull/1495
[#1497]: https://github.com/slack-go/slack/pull/1497
[#1499]: https://github.com/slack-go/slack/pull/1499
[#1504]: https://github.com/slack-go/slack/pull/1504
[#1506]: https://github.com/slack-go/slack/pull/1506
[#1511]: https://github.com/slack-go/slack/pull/1511
[#1513]: https://github.com/slack-go/slack/pull/1513
[#1515]: https://github.com/slack-go/slack/pull/1515
[#1516]: https://github.com/slack-go/slack/pull/1516
[#1517]: https://github.com/slack-go/slack/pull/1517
[#1519]: https://github.com/slack-go/slack/pull/1519
[#1520]: https://github.com/slack-go/slack/pull/1520
[#1522]: https://github.com/slack-go/slack/pull/1522
[#1524]: https://github.com/slack-go/slack/pull/1524
[#1526]: https://github.com/slack-go/slack/pull/1526
[#1528]: https://github.com/slack-go/slack/pull/1528
[#1529]: https://github.com/slack-go/slack/pull/1529
[#1530]: https://github.com/slack-go/slack/pull/1530
[#1532]: https://github.com/slack-go/slack/pull/1532
[#1533]: https://github.com/slack-go/slack/pull/1533
[#1534]: https://github.com/slack-go/slack/pull/1534
[#1536]: https://github.com/slack-go/slack/pull/1536
[#596]: https://github.com/slack-go/slack/issues/596
[#1541]: https://github.com/slack-go/slack/issues/1541
[#1172]: https://github.com/slack-go/slack/issues/1172
[#1076]: https://github.com/slack-go/slack/issues/1076
[#1157]: https://github.com/slack-go/slack/issues/1157
[#1196]: https://github.com/slack-go/slack/issues/1196
[#1547]: https://github.com/slack-go/slack/pull/1547

[Unreleased]: https://github.com/slack-go/slack/compare/v0.24.0...HEAD
[0.24.0]: https://github.com/slack-go/slack/compare/v0.23.1...v0.24.0
[0.23.1]: https://github.com/slack-go/slack/compare/v0.23.0...v0.23.1
[0.23.0]: https://github.com/slack-go/slack/compare/v0.22.0...v0.23.0
[0.22.0]: https://github.com/slack-go/slack/compare/v0.21.1...0.22.0
[0.21.1]: https://github.com/slack-go/slack/compare/v0.21.0...v0.21.1
[0.21.0]: https://github.com/slack-go/slack/compare/v0.20.0...v0.21.0
[0.20.0]: https://github.com/slack-go/slack/compare/v0.19.0...v0.20.0
[0.19.0]: https://github.com/slack-go/slack/compare/v0.18.0...v0.19.0
[0.18.0]: https://github.com/slack-go/slack/compare/v0.18.0-rc2...v0.18.0
[0.18.0-rc2]: https://github.com/slack-go/slack/releases/tag/v0.18.0-rc2
[0.18.0-rc1]: https://github.com/slack-go/slack/releases/tag/v0.18.0-rc1
[0.17.3]: https://github.com/slack-go/slack/releases/tag/v0.17.3
