## v0.26.0 (2025-11-16)

* *(client,appservice)* Deprecated `SendMassagedStateEvent` as `SendStateEvent`
  has been able to do the same for a while now.
* *(client,federation)* Added size limits for responses to make it safer to send
  requests to untrusted servers.
* *(client)* Added wrapper for `/admin/whois` client API
  (thanks to [@nexy7574] in [#411]).
* *(synapseadmin)* Added `force_purge` option to DeleteRoom
  (thanks to [@nexy7574] in [#420]).
* *(statestore)* Added saving join rules for rooms.
* *(bridgev2)* Added optional automatic rollback of room state if bridging the
  change to the remote network fails.
* *(bridgev2)* Added management room notices if transient disconnect state
  doesn't resolve within 3 minutes.
* *(bridgev2)* Added interface to signal that certain participants couldn't be
  invited when creating a group.
* *(bridgev2)* Added `select` type for user input fields in login.
* *(bridgev2)* Added interface to let network connector customize personal
  filtering space.
* *(bridgev2/matrix)* Added checks to avoid sending error messages in reply to
  other bots.
* *(bridgev2/matrix)* Switched to using [MSC4169] to send redactions whenever
  possible.
* *(bridgev2/publicmedia)* Added support for custom path prefixes, file names,
  and encrypted files.
* *(bridgev2/commands)* Added command to resync a single portal.
* *(bridgev2/commands)* Added create group command.
* *(bridgev2/config)* Added option to limit maximum number of logins.
* *(bridgev2)* Changed ghost joining to skip unnecessary invite if portal room
  is public.
* *(bridgev2/disappear)* Changed read receipt handling to only start
  disappearing timers for messages up to the read message (note: may not work in
  all cases if the read receipt points at an unknown event).
* *(event/reply)* Changed plaintext reply fallback removal to only happen when
  an HTML reply fallback is removed successfully.
* *(bridgev2/matrix)* Fixed unnecessary sleep after registering bot on first run.
* *(crypto/goolm)* Fixed panic when processing certain malformed Olm messages.
* *(federation)* Fixed HTTP method for sending transactions
  (thanks to [@nexy7574] in [#426]).
* *(federation)* Fixed response body being closed even when using `DontReadBody`
  parameter.
* *(federation)* Fixed validating auth for requests with query params.
* *(federation/eventauth)* Fixed typo causing restricted joins to not work.

[MSC416]: https://github.com/matrix-org/matrix-spec-proposals/pull/4169
[#411]: github.com/mautrix/go/pull/411
[#420]: github.com/mautrix/go/pull/420
[#426]: github.com/mautrix/go/pull/426

## v0.25.2 (2025-10-16)

* **Breaking change *(id)*** Split `UserID.ParseAndValidate` into
  `ParseAndValidateRelaxed` and `ParseAndValidateStrict`. Strict is the old
  behavior, but most users likely want the relaxed version, as there are real
  users whose user IDs aren't valid under the strict rules.
* *(crypto)* Added helper methods for generating and verifying with recovery
  keys.
* *(bridgev2/matrix)* Added config option to automatically generate a recovery
  key for the bridge bot and self-sign the bridge's device.
* *(bridgev2/matrix)* Added initial support for using appservice/MSC3202 mode
  for encryption with standard servers like Synapse.
* *(bridgev2)* Added optional support for implicit read receipts.
* *(bridgev2)* Added interface for deleting chats on remote network.
* *(bridgev2)* Added local enforcement of media duration and size limits.
* *(bridgev2)* Extended event duration logging to log any event taking too long.
* *(bridgev2)* Improved validation in group creation provisioning API.
* *(event)* Added event type constant for poll end events.
* *(client)* Added wrapper for searching user directory.
* *(client)* Improved support for managing [MSC4140] delayed events.
* *(crypto/helper)* Changed default sync handling to not block on waiting for
  decryption keys. On initial sync, keys won't be requested at all by default.
* *(crypto)* Fixed olm unwedging not working (regressed in v0.25.1).
* *(bridgev2)* Fixed various bugs with migrating to split portals.
* *(event)* Fixed poll start events having incorrect null `m.relates_to`.
* *(client)* Fixed `RespUserProfile` losing standard fields when re-marshaling.
* *(federation)* Fixed various bugs in event auth.

## v0.25.1 (2025-09-16)

* *(client)* Fixed HTTP method of delete devices API call
  (thanks to [@fmseals] in [#393]).
* *(client)* Added wrappers for [MSC4323]: User suspension & locking endpoints
  (thanks to [@nexy7574] in [#407]).
* *(client)* Stabilized support for extensible profiles.
* *(client)* Stabilized support for `state_after` in sync.
* *(client)* Removed deprecated MSC2716 requests.
* *(crypto)* Added fallback to ensure `m.relates_to` is always copied even if
  the content struct doesn't implement `Relatable`.
* *(crypto)* Changed olm unwedging to ignore newly created sessions if they
  haven't been used successfully in either direction.
* *(federation)* Added utilities for generating, parsing, validating and
  authorizing PDUs.
  * Note: the new PDU code depends on `GOEXPERIMENT=jsonv2`
* *(event)* Added `is_animated` flag from [MSC4230] to file info.
* *(event)* Added types for [MSC4332]: In-room bot commands.
* *(event)* Added missing poll end event type for [MSC3381].
* *(appservice)* Fixed URLs not being escaped properly when using unix socket
  for homeserver connections.
* *(format)* Added more helpers for forming markdown links.
* *(event,bridgev2)* Added support for Beeper's disappearing message state event.
* *(bridgev2)* Redesigned group creation interface and added support in commands
  and provisioning API.
* *(bridgev2)* Added GetEvent to Matrix interface to allow network connectors to
  get an old event. The method is best effort only, as some configurations don't
  allow fetching old events.
* *(bridgev2)* Added shared logic for provisioning that can be reused by the
  API, commands and other sources.
* *(bridgev2)* Fixed mentions and URL previews not being copied over when
  caption and media are merged.
* *(bridgev2)* Removed config option to change provisioning API prefix, which
  had already broken in the previous release.

[@fmseals]: https://github.com/fmseals
[#393]: https://github.com/mautrix/go/pull/393
[#407]: https://github.com/mautrix/go/pull/407
[MSC3381]: https://github.com/matrix-org/matrix-spec-proposals/pull/3381
[MSC4230]: https://github.com/matrix-org/matrix-spec-proposals/pull/4230
[MSC4323]: https://github.com/matrix-org/matrix-spec-proposals/pull/4323
[MSC4332]: https://github.com/matrix-org/matrix-spec-proposals/pull/4332

## v0.25.0 (2025-08-16)

* Bumped minimum Go version to 1.24.
* **Breaking change *(appservice,bridgev2,federation)*** Replaced gorilla/mux
  with standard library ServeMux.
* *(client,bridgev2)* Added support for creator power in room v12.
* *(client)* Added option to not set `User-Agent` header for improved Wasm
  compatibility.
* *(bridgev2)* Added support for following tombstones.
* *(bridgev2)* Added interface for getting arbitrary state event from Matrix.
* *(bridgev2)* Added batching to disappearing message queue to ensure it doesn't
  use too many resources even if there are a large number of messages.
* *(bridgev2/commands)* Added support for canceling QR login with `cancel`
  command.
* *(client)* Added option to override HTTP client used for .well-known
  resolution.
* *(crypto/backup)* Added method for encrypting key backup session without
  private keys.
* *(event->id)* Moved room version type and constants to id package.
* *(bridgev2)* Bots in DM portals will now be added to the functional members
  state event to hide them from the room name calculation.
* *(bridgev2)* Changed message delete handling to ignore "delete for me" events
  if there are multiple Matrix users in the room.
* *(format/htmlparser)* Changed text processing to collapse multiple spaces into
  one when outside `pre`/`code` tags.
* *(format/htmlparser)* Removed link suffix in plaintext output when link text
  is only missing protocol part of href.
  * e.g. `<a href="https://example.com">example.com</a>` will turn into
    `example.com` rather than `example.com (https://example.com)`
* *(appservice)* Switched appservice websockets from gorilla/websocket to
  coder/websocket.
* *(bridgev2/matrix)* Fixed encryption key sharing not ignoring ghosts properly.
* *(crypto/attachments)* Fixed hash check when decrypting file streams.
* *(crypto)* Removed unnecessary `AlreadyShared` error in `ShareGroupSession`.
  The function will now act as if it was successful instead.

## v0.24.2 (2025-07-16)

* *(bridgev2)* Added support for return values from portal event handlers. Note
  that the return value will always be "queued" unless the event buffer is
  disabled.
* *(bridgev2)* Added support for [MSC4144] per-message profile passthrough in
  relay mode.
* *(bridgev2)* Added option to auto-reconnect logins after a certain period if
  they hit an `UNKNOWN_ERROR` state.
* *(bridgev2)* Added analytics for event handler panics.
* *(bridgev2)* Changed new room creation to hardcode room v11 to avoid v12 rooms
  being created before proper support for them can be added.
* *(bridgev2)* Changed queuing events to block instead of dropping events if the
  buffer is full.
* *(bridgev2)* Fixed assumption that replies to unknown messages are cross-room.
* *(id)* Fixed server name validation not including ports correctly
  (thanks to [@krombel] in [#392]).
* *(federation)* Fixed base64 algorithm in signature generation.
* *(event)* Fixed [MSC4144] fallbacks not being removed from edits.

[@krombel]: https://github.com/krombel
[#392]: https://github.com/mautrix/go/pull/392

## v0.24.1 (2025-06-16)

* *(commands)* Added framework for using reactions as buttons that execute
  command handlers.
* *(client)* Added wrapper for `/relations` endpoints.
* *(client)* Added support for stable version of room summary endpoint.
* *(client)* Fixed parsing URL preview responses where width/height are strings.
* *(federation)* Fixed bugs in server auth.
* *(id)* Added utilities for validating server names.
* *(event)* Fixed incorrect empty `entity` field when sending hashed moderation
  policy events.
* *(event)* Added [MSC4293] redact events field to member events.
* *(event)* Added support for fallbacks in [MSC4144] per-message profiles.
* *(format)* Added `MarkdownLink` and `MarkdownMention` utility functions for
  generating properly escaped markdown.
* *(synapseadmin)* Added support for synchronous (v1) room delete endpoint.
* *(synapseadmin)* Changed `Client` struct to not embed the `mautrix.Client`.
  This is a breaking change if you were relying on accessing non-admin functions
  from the admin client.
* *(bridgev2/provisioning)* Fixed `/display_and_wait` not passing through errors
  from the network connector properly.
* *(bridgev2/crypto)* Fixed encryption not working if the user's ID had the same
  prefix as the bridge ghosts (e.g. `@whatsappbridgeuser:example.com` with a
  `@whatsapp_` prefix).
* *(bridgev2)* Fixed portals not being saved after creating a DM portal from a
  Matrix DM invite.
* *(bridgev2)* Added config option to determine whether cross-room replies
  should be bridged.
* *(appservice)* Fixed `EnsureRegistered` not being called when sending a custom
  member event for the controlled user.

[MSC4293]: https://github.com/matrix-org/matrix-spec-proposals/pull/4293

## v0.24.0 (2025-05-16)

* *(commands)* Added generic framework for implementing bot commands.
* *(client)* Added support for specifying maximum number of HTTP retries using
  a context value instead of having to call `MakeFullRequest` manually.
* *(client,federation)* Added methods for fetching room directories.
* *(federation)* Added support for server side of request authentication.
* *(synapseadmin)* Added wrapper for the account suspension endpoint.
* *(format)* Added method for safely wrapping a string in markdown inline code.
* *(crypto)* Added method to import key backup without persisting to database,
  to allow the client more control over the process.
* *(bridgev2)* Added viewing chat interface to signal when the user is viewing
  a given chat.
* *(bridgev2)* Added option to pass through transaction ID from client when
  sending messages to remote network.
* *(crypto)* Fixed unnecessary error log when decrypting dummy events used for
  unwedging Olm sessions.
* *(crypto)* Fixed `forwarding_curve25519_key_chain` not being set consistently
  when backing up keys.
* *(event)* Fixed marshaling legacy VoIP events with no version field.
* *(bridgev2)* Fixed disappearing message references not being deleted when the
  portal is deleted.
* *(bridgev2)* Fixed read receipt bridging not ignoring fake message entries
  and causing unnecessary error logs.

## v0.23.3 (2025-04-16)

* *(commands)* Added generic command processing framework for bots.
* *(client)* Added `allowed_room_ids` field to room summary responses
  (thanks to [@nexy7574] in [#367]).
* *(bridgev2)* Added support for custom timeouts on outgoing messages which have
  to wait for a remote echo.
* *(bridgev2)* Added automatic typing stop event if the ghost user had sent a
  typing event before a message.
* *(bridgev2)* The saved management room is now cleared if the user leaves the
  room, allowing the next DM to be automatically marked as a management room.
* *(bridge)* Removed deprecated fallback package for bridge statuses.
  The status package is now only available under bridgev2.

[#367]: https://github.com/mautrix/go/pull/367

## v0.23.2 (2025-03-16)

* **Breaking change *(bridge)*** Removed legacy bridge module.
* **Breaking change *(event)*** Changed `m.federate` field in room create event
  content to a pointer to allow detecting omitted values.
* *(bridgev2/commands)* Added `set-management-room` command to set a new
  management room.
* *(bridgev2/portal)* Changed edit bridging to ignore remote edits if the
  original sender on Matrix can't be puppeted.
* *(bridgv2)* Added config option to disable bridging `m.notice` messages.
* *(appservice/http)* Switched access token validation to use constant time
  comparisons.
* *(event)* Added support for [MSC3765] rich text topics.
* *(event)* Added fields to policy list event contents for [MSC4204] and
  [MSC4205].
* *(client)* Added method for getting the content of a redacted event using
  [MSC2815].
* *(client)* Added methods for sending and updating [MSC4140] delayed events.
* *(client)* Added support for [MSC4222] in sync payloads.
* *(crypto/cryptohelper)* Switched to using `sqlite3-fk-wal` instead of plain
  `sqlite3` by default.
* *(crypto/encryptolm)* Added generic method for encrypting to-device events.
* *(crypto/ssss)* Fixed panic if server-side key metadata is corrupted.
* *(crypto/sqlstore)* Fixed error when marking over 32 thousand device lists
  as outdated on SQLite.

[MSC2815]: https://github.com/matrix-org/matrix-spec-proposals/pull/2815
[MSC3765]: https://github.com/matrix-org/matrix-spec-proposals/pull/3765
[MSC4140]: https://github.com/matrix-org/matrix-spec-proposals/pull/4140
[MSC4204]: https://github.com/matrix-org/matrix-spec-proposals/pull/4204
[MSC4205]: https://github.com/matrix-org/matrix-spec-proposals/pull/4205
[MSC4222]: https://github.com/matrix-org/matrix-spec-proposals/pull/4222

## v0.23.1 (2025-02-16)

* *(client)* Added `FullStateEvent` method to get a state event including
  metadata (using the `?format=event` query parameter).
* *(client)* Added wrapper method for [MSC4194]'s redact endpoint.
* *(pushrules)* Fixed content rules not considering word boundaries and being
  case-sensitive.
* *(crypto)* Fixed bugs that would cause key exports to fail for no reason.
* *(crypto)* Deprecated `ResolveTrust` in favor of `ResolveTrustContext`.
* *(crypto)* Stopped accepting secret shares from unverified devices.
* **Breaking change *(crypto)*** Changed `GetAndVerifyLatestKeyBackupVersion`
  to take an optional private key parameter. The method will now trust the
  public key if it matches the provided private key even if there are no valid
  signatures.
* **Breaking change *(crypto)*** Added context parameter to `IsDeviceTrusted`.

[MSC4194]: https://github.com/matrix-org/matrix-spec-proposals/pull/4194

## v0.23.0 (2025-01-16)

* **Breaking change *(client)*** Changed `JoinRoom` parameters to allow multiple
  `via`s.
* **Breaking change *(bridgev2)*** Updated capability system.
  * The return type of `NetworkAPI.GetCapabilities` is now different.
  * Media type capabilities are enforced automatically by bridgev2.
  * Capabilities are now sent to Matrix rooms using the
    `com.beeper.room_features` state event.
* *(client)* Added `GetRoomSummary` to implement [MSC3266].
* *(client)* Added support for arbitrary profile fields to implement [MSC4133]
  (thanks to [@nexy7574] in [#337]).
* *(crypto)* Started storing olm message hashes to prevent decryption errors
  if messages are repeated (e.g. if the app crashes right after decrypting).
* *(crypto)* Improved olm session unwedging to check when the last session was
  created instead of only relying on an in-memory map.
* *(crypto/verificationhelper)* Fixed emoji verification not doing cross-signing
  properly after a successful verification.
* *(bridgev2/config)* Moved MSC4190 flag from `appservice` to `encryption`.
* *(bridgev2/space)* Fixed failing to add rooms to spaces if the room create
  call was made with a temporary context.
* *(bridgev2/commands)* Changed `help` command to hide commands which require
  interfaces that aren't implemented by the network connector.
* *(bridgev2/matrixinterface)* Moved deterministic room ID generation to Matrix
  connector.
* *(bridgev2)* Fixed service member state event not being set correctly when
  creating a DM by inviting a ghost user.
* *(bridgev2)* Fixed `RemoteReactionSync` events replacing all reactions every
  time instead of only changed ones.

[MSC3266]: https://github.com/matrix-org/matrix-spec-proposals/pull/3266
[MSC4133]: https://github.com/matrix-org/matrix-spec-proposals/pull/4133
[@nexy7574]: https://github.com/nexy7574
[#337]: https://github.com/mautrix/go/pull/337

## v0.22.1 (2024-12-16)

* *(crypto)* Added automatic cleanup when there are too many olm sessions with
  a single device.
* *(crypto)* Added helper for getting cached device list with cross-signing
  status.
* *(crypto/verificationhelper)* Added interface for persisting the state of
  in-progress verifications.
* *(client)* Added `GetMutualRooms` wrapper for [MSC2666].
* *(client)* Switched `JoinRoom` to use the `via` query param instead of
  `server_name` as per [MSC4156].
* *(bridgev2/commands)* Fixed `pm` command not actually starting the chat.
* *(bridgev2/interface)* Added separate network API interface for starting
  chats with a Matrix ghost user. This allows treating internal user IDs
  differently than arbitrary user-input strings.
* *(bridgev2/crypto)* Added support for [MSC4190]
  (thanks to [@onestacked] in [#288]).

[MSC2666]: https://github.com/matrix-org/matrix-spec-proposals/pull/2666
[MSC4156]: https://github.com/matrix-org/matrix-spec-proposals/pull/4156
[MSC4190]: https://github.com/matrix-org/matrix-spec-proposals/pull/4190
[#288]: https://github.com/mautrix/go/pull/288

## v0.22.0 (2024-11-16)

* *(hicli)* Moved package into gomuks repo.
* *(bridgev2/commands)* Fixed cookie unescaping in login commands.
* *(bridgev2/portal)* Added special `DefaultChatName` constant to explicitly
  reset portal names to the default (based on members).
* *(bridgev2/config)* Added options to disable room tag bridging.
* *(bridgev2/database)* Fixed reaction queries not including portal receiver.
* *(appservice)* Updated [MSC2409] stable registration field name from
  `push_ephemeral` to `receive_ephemeral`. Homeserver admins must update
  existing registrations manually.
* *(format)* Added support for `img` tags.
* *(format/mdext)* Added goldmark extensions for Matrix math and custom emojis.
* *(event/reply)* Removed support for generating reply fallbacks ([MSC2781]).
* *(pushrules)* Added support for `sender_notification_permission` condition
  kind (used for `@room` mentions).
* *(crypto)* Added support for `json.RawMessage` in `EncryptMegolmEvent`.
* *(mediaproxy)* Added `GetMediaResponseCallback` and `GetMediaResponseFile`
  to write proxied data directly to http response or temp file instead of
  having to use an `io.Reader`.
* *(mediaproxy)* Dropped support for legacy media download endpoints.
* *(mediaproxy,bridgev2)* Made interface pass through query parameters.

[MSC2781]: https://github.com/matrix-org/matrix-spec-proposals/pull/2781

## v0.21.1 (2024-10-16)

* *(bridgev2)* Added more features and fixed bugs.
* *(hicli)* Added more features and fixed bugs.
* *(appservice)* Removed TLS support. A reverse proxy should be used if TLS
  is needed.
* *(format/mdext)* Added goldmark extension to fix indented paragraphs when
  disabling indented code block parser.
* *(event)* Added `Has` method for `Mentions`.
* *(event)* Added basic support for the unstable version of polls.

## v0.21.0 (2024-09-16)

* **Breaking change *(client)*** Dropped support for unauthenticated media.
  Matrix v1.11 support is now required from the homeserver, although it's not
  enforced using `/versions` as some servers don't advertise it.
* *(bridgev2)* Added more features and fixed bugs.
* *(appservice,crypto)* Added support for using MSC3202 for appservice
  encryption.
* *(crypto/olm)* Made everything into an interface to allow side-by-side
  testing of libolm and goolm, as well as potentially support vodozemac
  in the future.
* *(client)* Fixed requests being retried even after context is canceled.
* *(client)* Added option to move `/sync` request logs to trace level.
* *(error)* Added `Write` and `WithMessage` helpers to `RespError` to make it
  easier to use on servers.
* *(event)* Fixed `org.matrix.msc1767.audio` field allowing omitting the
  duration and waveform.
* *(id)* Changed `MatrixURI` methods to not panic if the receiver is nil.
* *(federation)* Added limit to response size when fetching `.well-known` files.

## v0.20.0 (2024-08-16)

* Bumped minimum Go version to 1.22.
* *(bridgev2)* Added more features and fixed bugs.
* *(event)* Added types for [MSC4144]: Per-message profiles.
* *(federation)* Added implementation of server name resolution and a basic
  client for making federation requests.
* *(crypto/ssss)* Changed recovery key/passphrase verify functions to take the
  key ID as a parameter to ensure it's correctly set even if the key metadata
  wasn't fetched via `GetKeyData`.
* *(format/mdext)* Added goldmark extensions for single-character bold, italic
  and strikethrough parsing (as in `*foo*` -> **foo**, `_foo_` -> _foo_ and
  `~foo~` -> ~~foo~~)
* *(format)* Changed `RenderMarkdown` et al to always include `m.mentions` in
  returned content. The mention list is filled with matrix.to URLs from the
  input by default.

[MSC4144]: https://github.com/matrix-org/matrix-spec-proposals/pull/4144

## v0.19.0 (2024-07-16)

* Renamed `master` branch to `main`.
* *(bridgev2)* Added more features.
* *(crypto)* Fixed bug with copying `m.relates_to` from wire content to
  decrypted content.
* *(mediaproxy)* Added module for implementing simple media repos that proxy
  requests elsewhere.
* *(client)* Changed `Members()` to automatically parse event content for all
  returned events.
* *(bridge)* Added `/register` call if `/versions` fails with `M_FORBIDDEN`.
* *(crypto)* Fixed `DecryptMegolmEvent` sometimes calling database without
  transaction by using the non-context version of `ResolveTrust`.
* *(crypto/attachment)* Implemented `io.Seeker` in `EncryptStream` to allow
  using it in retriable HTTP requests.
* *(event)* Added helper method to add user ID to a `Mentions` object.
* *(event)* Fixed default power level for invites
  (thanks to [@rudis] in [#250]).
* *(client)* Fixed incorrect warning log in `State()` when state store returns
  no error (thanks to [@rudis] in [#249]).
* *(crypto/verificationhelper)* Fixed deadlock when ignoring unknown
  cancellation events (thanks to [@rudis] in [#247]).

[@rudis]: https://github.com/rudis
[#250]: https://github.com/mautrix/go/pull/250
[#249]: https://github.com/mautrix/go/pull/249
[#247]: https://github.com/mautrix/go/pull/247

### beta.1 (2024-06-16)

* *(bridgev2)* Added experimental high-level bridge framework.
* *(hicli)* Added experimental high-level client framework.
* **Slightly breaking changes**
  * *(crypto)* Added room ID and first known index parameters to
    `SessionReceived` callback.
  * *(crypto)* Changed `ImportRoomKeyFromBackup` to return the imported
    session.
  * *(client)* Added `error` parameter to `ResponseHook`.
  * *(client)* Changed `Download` to return entire response instead of just an
    `io.Reader`.
* *(crypto)* Changed initial olm device sharing to save keys before sharing to
  ensure keys aren't accidentally regenerated in case the request fails.
* *(crypto)* Changed `EncryptMegolmEvent` and `ShareGroupSession` to return
  more errors instead of only logging and ignoring them.
* *(crypto)* Added option to completely disable megolm ratchet tracking.
  * The tracking is meant for bots and bridges which may want to delete old
    keys, but for normal clients it's just unnecessary overhead.
* *(crypto)* Changed Megolm session storage methods in `Store` to not take
  sender key as parameter.
  * This causes a breaking change to the layout of the `MemoryStore` struct.
    Using MemoryStore in production is not recommended.
* *(crypto)* Changed `DecryptMegolmEvent` to copy `m.relates_to` in the raw
  content too instead of only in the parsed struct.
* *(crypto)* Exported function to parse megolm message index from raw
  ciphertext bytes.
* *(crypto/sqlstore)* Fixed schema of `crypto_secrets` table to include
  account ID.
* *(crypto/verificationhelper)* Fixed more bugs.
* *(client)* Added `UpdateRequestOnRetry` hook which is called immediately
  before retrying a normal HTTP request.
* *(client)* Added support for MSC3916 media download endpoint.
  * Support is automatically detected from spec versions. The `SpecVersions`
    property can either be filled manually, or `Versions` can be called to
    automatically populate the field with the response.
* *(event)* Added constants for known room versions.

## v0.18.1 (2024-04-16)

* *(format)* Added a `context.Context` field to HTMLParser's Context struct.
* *(bridge)* Added support for handling join rules, knocks, invites and bans
  (thanks to [@maltee1] in [#193] and [#204]).
* *(crypto)* Changed forwarded room key handling to only accept keys with a
  lower first known index than the existing session if there is one.
* *(crypto)* Changed key backup restore to assume own device list is up to date
  to avoid re-requesting device list for every deleted device that has signed
  key backup.
* *(crypto)* Fixed memory cache not being invalidated when storing own
  cross-signing keys

[@maltee1]: https://github.com/maltee1
[#193]: https://github.com/mautrix/go/pull/193
[#204]: https://github.com/mautrix/go/pull/204

## v0.18.0 (2024-03-16)

* **Breaking change *(client, bridge, appservice)*** Dropped support for
  maulogger. Only zerolog loggers are now provided by default.
* *(bridge)* Fixed upload size limit not having a default if the server
  returned no value.
* *(synapseadmin)* Added wrappers for some room and user admin APIs.
  (thanks to [@grvn-ht] in [#181]).
* *(crypto/verificationhelper)* Fixed bugs.
* *(crypto)* Fixed key backup uploading doing too much base64.
* *(crypto)* Changed `EncryptMegolmEvent` to return an error if persisting the
  megolm session fails. This ensures that database errors won't cause messages
  to be sent with duplicate indexes.
* *(crypto)* Changed `GetOrRequestSecret` to use a callback instead of returning
  the value directly. This allows validating the value in order to ignore
  invalid secrets.
* *(id)* Added `ParseCommonIdentifier` function to parse any Matrix identifier
  in the [Common Identifier Format].
* *(federation)* Added simple key server that passes the federation tester.

[@grvn-ht]: https://github.com/grvn-ht
[#181]: https://github.com/mautrix/go/pull/181
[Common Identifier Format]: https://spec.matrix.org/v1.9/appendices/#common-identifier-format

### beta.1 (2024-02-16)

* Bumped minimum Go version to 1.21.
* *(bridge)* Bumped minimum Matrix spec version to v1.4.
* **Breaking change *(crypto)*** Deleted old half-broken interactive
  verification code and replaced it with a new `verificationhelper`.
  * The new verification helper is still experimental.
  * Both QR and emoji verification are supported (in theory).
* *(crypto)* Added support for server-side key backup.
* *(crypto)* Added support for receiving and sending secrets like cross-signing
  private keys via secret sharing.
* *(crypto)* Added support for tracking which devices megolm sessions were
  initially shared to, and allowing re-sharing the keys to those sessions.
* *(client)* Changed cross-signing key upload method to accept a callback for
  user-interactive auth instead of only hardcoding password support.
* *(appservice)* Dropped support for legacy non-prefixed appservice paths
  (e.g. `/transactions` instead of `/_matrix/app/v1/transactions`).
* *(appservice)* Dropped support for legacy `access_token` authorization in
  appservice endpoints.
* *(bridge)* Fixed `RawArgs` field in command events of command state callbacks.
* *(appservice)* Added `CreateFull` helper function for creating an `AppService`
  instance with all the mandatory fields set.

## v0.17.0 (2024-01-16)

* **Breaking change *(bridge)*** Added raw event to portal membership handling
  functions.
* **Breaking change *(everything)*** Added context parameters to all functions
  (started by [@recht] in [#144]).
* **Breaking change *(client)*** Moved event source from sync event handler
  function parameters to the `Mautrix.EventSource` field inside the event
  struct.
* **Breaking change *(client)*** Moved `EventSource` to `event.Source`.
* *(client)* Removed deprecated `OldEventIgnorer`. The non-deprecated version
  (`Client.DontProcessOldEvents`) is still available.
* *(crypto)* Added experimental pure Go Olm implementation to replace libolm
  (thanks to [@DerLukas15] in [#106]).
  * You can use the `goolm` build tag to the new implementation.
* *(bridge)* Added context parameter for bridge command events.
* *(bridge)* Added method to allow custom validation for the entire config.
* *(client)* Changed default syncer to not drop unknown events.
  * The syncer will still drop known events if parsing the content fails.
  * The behavior can be changed by changing the `ParseErrorHandler` function.
* *(crypto)* Fixed some places using math/rand instead of crypto/rand.

[@DerLukas15]: https://github.com/DerLukas15
[@recht]: https://github.com/recht
[#106]: https://github.com/mautrix/go/pull/106
[#144]: https://github.com/mautrix/go/pull/144

## v0.16.2 (2023-11-16)

* *(event)* Added `Redacts` field to `RedactionEventContent` for room v11+.
* *(event)* Added `ReverseTextToHTML` which reverses the changes made by
  `TextToHTML` (i.e. unescapes HTML characters and replaces `<br/>` with `\n`).
* *(bridge)* Added global zerologger to ensure all logs go through the bridge
  logger.
* *(bridge)* Changed encryption error messages to be sent in a thread if the
  message that failed to decrypt was in a thread.

## v0.16.1 (2023-09-16)

* **Breaking change *(id)*** Updated user ID localpart encoding to not encode
  `+` as per [MSC4009].
* *(bridge)* Added bridge utility to handle double puppeting logins.
  * The utility supports automatic logins with all three current methods
    (shared secret, legacy appservice, new appservice).
* *(appservice)* Added warning logs and timeout on appservice event handling.
  * Defaults to warning after 30 seconds and timeout 15 minutes after that.
  * Timeouts can be adjusted or disabled by setting `ExecSync` variables in the
    `EventProcessor`.
* *(crypto/olm)* Added `PkDecryption` wrapper.

[MSC4009]: https://github.com/matrix-org/matrix-spec-proposals/pull/4009

## v0.16.0 (2023-08-16)

* Bumped minimum Go version to 1.20.
* **Breaking change *(util)*** Moved package to [go.mau.fi/util](https://go.mau.fi/util/)
* *(event)* Removed MSC2716 `historical` field in the `m.room.power_levels`
  event content struct.
* *(bridge)* Added `--version-json` flag to print bridge version info as JSON.
* *(appservice)* Added option to use custom transaction handler for websocket mode.

## v0.15.4 (2023-07-16)

* *(client)* Deprecated MSC2716 methods and added new Beeper-specific batch
  send methods, as upstream MSC2716 support has been abandoned.
* *(client)* Added proper error handling and automatic retries to media
  downloads.
* *(crypto, bridge)* Added option to remove all keys that were received before
  the automatic ratcheting was implemented (in v0.15.1).
* *(dbutil)* Added `JSON` utility for writing/reading arbitrary JSON objects to
  the db conveniently without manually de/serializing.

## v0.15.3 (2023-06-16)

* *(synapseadmin)* Added wrappers for some Synapse admin API endpoints.
* *(pushrules)* Implemented new `event_property_is` and `event_property_contains`
  push rule condition kinds as per MSC3758 and MSC3966.
* *(bridge)* Moved websocket code from mautrix-imessage to enable all bridges
  to use appservice websockets easily.
* *(bridge)* Added retrying for appservice pings.
* *(types)* Removed unstable field for MSC3952 (intentional mentions).
* *(client)* Deprecated `OldEventIgnorer` and added `Client.DontProcessOldEvents`
  to replace it.
* *(client)* Added `MoveInviteState` sync handler for moving state events in
  the invite section of sync inside the invite event itself.
* *(crypto)* Added option to not rotate keys when devices change.
* *(crypto)* Added additional duplicate message index check if decryption fails
  because the keys had been ratcheted forward.
* *(client)* Stabilized support for asynchronous uploads.
  * `UnstableCreateMXC` and `UnstableUploadAsync` were renamed to `CreateMXC`
    and `UploadAsync` respectively.
* *(util/dbutil)* Added option to use a separate database connection pool for
  read-only transactions.
  * This is mostly meant for SQLite and it enables read-only transactions that
    don't lock the database, even when normal transactions are configured to
    acquire a write lock immediately.
* *(util/dbutil)* Enabled caller info in zerolog by default.

## v0.15.2 (2023-05-16)

* *(client)* Changed member-fetching methods to clear existing member info in
  state store.
* *(client)* Added support for inserting mautrix-go commit hash into default
  user agent at compile time.
* *(bridge)* Fixed bridge bot intent not having state store set.
* *(client)* Fixed `RespError` marshaling mutating the `ExtraData` map and
  potentially causing panics.
* *(util/dbutil)* Added `DoTxn` method for an easier way to manage database
  transactions.
* *(util)* Added a zerolog `CallerMarshalFunc` implementation that includes the
  function name.
* *(bridge)* Added error reply to encrypted messages if the bridge isn't
  configured to do encryption.

## v0.15.1 (2023-04-16)

* *(crypto, bridge)* Added options to automatically ratchet/delete megolm
  sessions to minimize access to old messages.
* *(pushrules)* Added method to get entire push rule that matched (instead of
  only the list of actions).
* *(pushrules)* Deprecated `NotifySpecified` as there's no reason to read it.
* *(crypto)* Changed `max_age` column in `crypto_megolm_inbound_session` table
  to be milliseconds instead of nanoseconds.
* *(util)* Added method for iterating `RingBuffer`.
* *(crypto/cryptohelper)* Changed decryption errors to request session from all
  own devices in addition to the sender, instead of only asking the sender.
* *(sqlstatestore)* Fixed `FindSharedRooms` throwing an error when using from
  a non-bridge context.
* *(client)* Optimized `AccountDataSyncStore` to not resend save requests if
  the sync token didn't change.
* *(types)* Added `Clone()` method for `PowerLevelEventContent`.

## v0.15.0 (2023-03-16)

### beta.3 (2023-03-15)

* **Breaking change *(appservice)*** Removed `Load()` and `AppService.Init()`
  functions. The struct should just be created with `Create()` and the relevant
  fields should be filled manually.
* **Breaking change *(appservice)*** Removed public `HomeserverURL` field and
  replaced it with a `SetHomeserverURL` method.
* *(appservice)* Added support for unix sockets for homeserver URL and
  appservice HTTP server.
* *(client)* Changed request logging to log durations as floats instead of
  strings (using zerolog's `Dur()`, so the exact output can be configured).
* *(bridge)* Changed zerolog to use nanosecond precision timestamps.
* *(crypto)* Added message index to log after encrypting/decrypting megolm
  events, and when failing to decrypt due to duplicate index.
* *(sqlstatestore)* Fixed warning log for rooms that don't have encryption
  enabled.

### beta.2 (2023-03-02)

* *(bridge)* Fixed building with `nocrypto` tag.
* *(bridge)* Fixed legacy logging config migration not disabling file writer
  when `file_name_format` was empty.
* *(bridge)* Added option to require room power level to run commands.
* *(event)* Added structs for [MSC3952]: Intentional Mentions.
* *(util/variationselector)* Added `FullyQualify` method to add necessary emoji
  variation selectors without adding all possible ones.

[MSC3952]: https://github.com/matrix-org/matrix-spec-proposals/pull/3952

### beta.1 (2023-02-24)

* Bumped minimum Go version to 1.19.
* **Breaking changes**
  * *(all)* Switched to zerolog for logging.
    * The `Client` and `Bridge` structs still include a legacy logger for
      backwards compatibility.
  * *(client, appservice)* Moved `SQLStateStore` from appservice module to the
    top-level (client) module.
  * *(client, appservice)* Removed unused `Typing` map in `SQLStateStore`.
  * *(client)* Removed unused `SaveRoom` and `LoadRoom` methods in `Storer`.
  * *(client, appservice)* Removed deprecated `SendVideo` and `SendImage` methods.
  * *(client)* Replaced `AppServiceUserID` field with `SetAppServiceUserID` boolean.
    The `UserID` field is used as the value for the query param.
  * *(crypto)* Renamed `GobStore` to `MemoryStore` and removed the file saving
    features. The data can still be persisted, but the persistence part must be
    implemented separately.
  * *(crypto)* Removed deprecated `DeviceIdentity` alias
    (renamed to `id.Device` long ago).
  * *(client)* Removed `Stringifable` interface as it's the same as `fmt.Stringer`.
* *(client)* Renamed `Storer` interface to `SyncStore`. A type alias exists for
  backwards-compatibility.
* *(crypto/cryptohelper)* Added package for a simplified crypto interface for clients.
* *(example)* Added e2ee support to example using crypto helper.
* *(client)* Changed default syncer to stop syncing on `M_UNKNOWN_TOKEN` errors.

## v0.14.0 (2023-02-16)

* **Breaking change *(format)*** Refactored the HTML parser `Context` to have
  more data.
* *(id)* Fixed escaping path components when forming matrix.to URLs
  or `matrix:` URIs.
* *(bridge)* Bumped default timeouts for decrypting incoming messages.
* *(bridge)* Added `RawArgs` to commands to allow accessing non-split input.
* *(bridge)* Added `ReplyAdvanced` to commands to allow setting markdown
  settings.
* *(event)* Added `notifications` key to `PowerLevelEventContent`.
* *(event)* Changed `SetEdit` to cut off edit fallback if the message is long.
* *(util)* Added `SyncMap` as a simple generic wrapper for a map with a mutex.
* *(util)* Added `ReturnableOnce` as a wrapper for `sync.Once` with a return
  value.

## v0.13.0 (2023-01-16)

* **Breaking change:** Removed `IsTyping` and `SetTyping` in `appservice.StateStore`
  and removed the `TypingStateStore` struct implementing those methods.
* **Breaking change:** Removed legacy fields in Beeper MSS events.
* Added knocked rooms to sync response structs.
* Added wrapper for `/timestamp_to_event` endpoint added in Matrix v1.6.
* Fixed MSC3870 uploads not failing properly after using up the max retry count.
* Fixed parsing non-positive ordered list start positions in HTML parser.

## v0.12.4 (2022-12-16)

* Added `SendReceipt` to support private read receipts and thread receipts in
  the same function. `MarkReadWithContent` is now deprecated.
* Changed media download methods to return errors if the server returns a
  non-2xx status code.
* Removed legacy `sql_store_upgrade.Upgrade` method. Using `store.DB.Upgrade()`
  after `NewSQLCryptoStore(...)` is recommended instead (the bridge module does
  this automatically).
* Added missing `suggested` field to `m.space.child` content struct.
* Added `device_unused_fallback_key_types` to `/sync` response and appservice
  transaction structs.
* Changed `ReqSetReadMarkers` to omit empty fields.
* Changed bridge configs to force `sqlite3-fk-wal` instead of `sqlite3`.
* Updated bridge helper to close database connection when stopping.
* Fixed read receipt and account data endpoints sending `null` instead of an
  empty object as the body when content isn't provided.

## v0.12.3 (2022-11-16)

* **Breaking change:** Added logging for row iteration in the dbutil package.
  This changes the return type of `Query` methods from `*sql.Rows` to a new
  `dbutil.Rows` interface.
* Added flag to disable wrapping database upgrades in a transaction (e.g. to
  allow setting `PRAGMA`s for advanced table mutations on SQLite).
* Deprecated `MessageEventContent.GetReplyTo` in favor of directly using
  `RelatesTo.GetReplyTo`. RelatesTo methods are nil-safe, so checking if
  RelatesTo is nil is not necessary for using those methods.
* Added wrapper for space hierarchyendpoint (thanks to [@mgcm] in [#100]).
* Added bridge config option to handle transactions asynchronously.
* Added separate channels for to-device events in appservice transaction
  handler to avoid blocking to-device events behind normal events.
* Added `RelatesTo.GetNonFallbackReplyTo` utility method to get the reply event
  ID, unless the reply is a thread fallback.
* Added `event.TextToHTML` as an utility method to HTML-escape a string and
  replace newlines with `<br/>`.
* Added check to bridge encryption helper to make sure the e2ee keys are still
  on the server. Synapse is known to sometimes lose keys randomly.
* Changed bridge crypto syncer to crash on `M_UNKNOWN_TOKEN` errors instead of
  retrying forever pointlessly.
* Fixed verifying signatures of fallback one-time keys.

[@mgcm]: https://github.com/mgcm
[#100]: https://github.com/mautrix/go/pull/100

## v0.12.2 (2022-10-16)

* Added utility method to redact bridge commands.
* Added thread ID field to read receipts to match Matrix v1.4 changes.
* Added automatic fetching of media repo config at bridge startup to make it
  easier for bridges to check homeserver media size limits.
* Added wrapper for the `/register/available` endpoint.
* Added custom user agent to all requests mautrix-go makes. The value can be
  customized by changing the `DefaultUserAgent` variable.
* Implemented [MSC3664], [MSC3862] and [MSC3873] in the push rule evaluator.
* Added workaround for potential race conditions in OTK uploads when using
  appservice encryption ([MSC3202]).
* Fixed generating registrations to use `.+` instead of `[0-9]+` in the
  username regex.
* Fixed panic in megolm session listing methods if the store contains withheld
  key entries.
* Fixed missing header in bridge command help messages.

[MSC3664]: https://github.com/matrix-org/matrix-spec-proposals/pull/3664
[MSC3862]: https://github.com/matrix-org/matrix-spec-proposals/pull/3862
[MSC3873]: https://github.com/matrix-org/matrix-spec-proposals/pull/3873

## v0.12.1 (2022-09-16)

* Bumped minimum Go version to 1.18.
* Added `omitempty` for a bunch of fields in response structs to make them more
  usable for server implementations.
* Added `util.RandomToken` to generate GitHub-style access tokens with checksums.
* Added utilities to call the push gateway API.
* Added `unread_notifications` and [MSC2654] `unread_count` fields to /sync
  response structs.
* Implemented [MSC3870] for uploading and downloading media directly to/from an
  external media storage like S3.
* Fixed dbutil database ownership checks on SQLite.
* Fixed typo in unauthorized encryption key withheld code
  (`m.unauthorized` -> `m.unauthorised`).
* Fixed [MSC2409] support to have a separate field for to-device events.

[MSC2654]: https://github.com/matrix-org/matrix-spec-proposals/pull/2654
[MSC3870]: https://github.com/matrix-org/matrix-spec-proposals/pull/3870

## v0.12.0 (2022-08-16)

* **Breaking change:** Switched `Client.UserTyping` to take a `time.Duration`
  instead of raw `int64` milliseconds.
* **Breaking change:** Removed custom reply relation type and switched to using
  the wire format (nesting in `m.in_reply_to`).
* Added device ID to appservice OTK count map to match updated [MSC3202].
  This is also a breaking change, but the previous incorrect behavior wasn't
  implemented by anything other than mautrix-syncproxy/imessage.
* (There are probably other breaking changes too).
* Added database utility and schema upgrade framework
  * Originally from mautrix-whatsapp, but usable for non-bridges too
  * Includes connection wrapper to log query durations and mutate queries for
    SQLite compatibility (replacing `$x` with `?x`).
* Added bridge utilities similar to mautrix-python. Currently includes:
  * Crypto helper
  * Startup flow
  * Command handling and some standard commands
  * Double puppeting things
  * Generic parts of config, basic config validation
  * Appservice SQL state store
* Added alternative markdown spoiler parsing extension that doesn't support
  reasons, but works better otherwise.
* Added Discord underline markdown parsing extension (`_foo_` -> <u>foo</u>).
* Added support for parsing spoilers and color tags in the HTML parser.
* Added support for mutating plain text nodes in the HTML parser.
* Added room version field to the create room request struct.
* Added empty JSON object as default request body for all non-GET requests.
* Added wrapper for `/capabilities` endpoint.
* Added `omitempty` markers for lots of structs to make the structs easier to
  use on the server side too.
* Added support for registering to-device event handlers via the default
  Syncer's `OnEvent` and `OnEventType` methods.
* Fixed `CreateEventContent` using the wrong field name for the room version
  field.
* Fixed `StopSync` not immediately cancelling the sync loop if it was sleeping
  after a failed sync.
* Fixed `GetAvatarURL` always returning the current user's avatar instead of
  the specified user's avatar (thanks to [@nightmared] in [#83]).
* Improved request logging and added new log when a request finishes.
* Crypto store improvements:
  * Deleted devices are now kept in the database.
  * Made ValidateMessageIndex atomic.
* Moved `appservice.RandomString` to the `util` package and made it use
  `crypto/rand` instead of `math/rand`.
* Significantly improved cross-signing validation code.
  * There are now more options for required trust levels,
    e.g. you can set `SendKeysMinTrust` to `id.TrustStateCrossSignedTOFU`
    to trust the first cross-signing master key seen and require all devices
    to be signed by that key.
  * Trust state of incoming messages is automatically resolved and stored in
    `evt.Mautrix.TrustState`. This can be used to reject incoming messages from
    untrusted devices.

[@nightmared]: https://github.com/nightmared
[#83]: https://github.com/mautrix/go/pull/83

## v0.11.1 (2023-01-15)

* Fixed parsing non-positive ordered list start positions in HTML parser
  (backport of the same fix in v0.13.0).

## v0.11.0 (2022-05-16)

* Bumped minimum Go version to 1.17.
* Switched from `/r0` to `/v3` paths everywhere.
  * The new `v3` paths are implemented since Synapse 1.48, Dendrite 0.6.5, and
    Conduit 0.4.0. Servers older than these are no longer supported.
* Switched from blackfriday to goldmark for markdown parsing in the `format`
  module and added spoiler syntax.
* Added `EncryptInPlace` and `DecryptInPlace` methods for attachment encryption.
  In most cases the plain/ciphertext is not necessary after en/decryption, so
  the old `Encrypt` and `Decrypt` are deprecated.
* Added wrapper for `/rooms/.../aliases`.
* Added utility for adding/removing emoji variation selectors to match
  recommendations on reactions in Matrix.
* Added support for async media uploads ([MSC2246]).
* Added automatic sleep when receiving 429 error
  (thanks to [@ownaginatious] in [#44]).
* Added support for parsing spec version numbers from the `/versions` endpoint.
* Removed unstable prefixed constant used for appservice login.
* Fixed URL encoding not working correctly in some cases.

[MSC2246]: https://github.com/matrix-org/matrix-spec-proposals/pull/2246
[@ownaginatious]: https://github.com/ownaginatious
[#44]: https://github.com/mautrix/go/pull/44

## v0.10.12 (2022-03-16)

* Added option to use a different `Client` to send invites in
  `IntentAPI.EnsureJoined`.
* Changed `MessageEventContent` struct to omit empty `msgtype`s in the output
  JSON, as sticker events shouldn't have that field.
* Fixed deserializing the `thumbnail_file` field in `FileInfo`.
* Fixed bug that broke `SQLCryptoStore.FindDeviceByKey`.

## v0.10.11 (2022-02-16)

* Added automatic updating of state store from `IntentAPI` calls.
* Added option to ignore cache in `IntentAPI.EnsureJoined`.
* Added `GetURLPreview` as a wrapper for the `/preview_url` media repo endpoint.
* Moved base58 module inline to avoid pulling in btcd as a dependency.

## v0.10.10 (2022-01-16)

* Added event types and content structs for server ACLs and moderation policy
  lists (thanks to [@qua3k] in [#59] and [#60]).
* Added optional parameter to `Client.LeaveRoom` to pass a `reason` field.

[#59]: https://github.com/mautrix/go/pull/59
[#60]: https://github.com/mautrix/go/pull/60

## v0.10.9 (2022-01-04)

* **Breaking change:** Changed `Messages()` to take a filter as a parameter
  instead of using the syncer's filter (thanks to [@qua3k] in [#55] and [#56]).
  * The previous filter behavior was completely broken, as it sent a whole
    filter instead of just a RoomEventFilter.
  * Passing `nil` as the filter is fine and will disable filtering
    (which is equivalent to what it did before with the invalid filter).
* Added `Context()` wrapper for the `/context` API (thanks to [@qua3k] in [#54]).
* Added utility for converting media files with ffmpeg.

[#54]: https://github.com/mautrix/go/pull/54
[#55]: https://github.com/mautrix/go/pull/55
[#56]: https://github.com/mautrix/go/pull/56
[@qua3k]: https://github.com/qua3k

## v0.10.8 (2021-12-30)

* Added `OlmSession.Describe()` to wrap `olm_session_describe`.
* Added trace logs to log olm session descriptions when encrypting/decrypting
  to-device messages.
* Added space event types and content structs.
* Added support for power level content override field in `CreateRoom`.
* Fixed ordering of olm sessions which would cause an old session to be used in
  some cases even after a client created a new session.

## v0.10.7 (2021-12-16)

* Changed `Client.RedactEvent` to allow arbitrary fields in redaction request.

## v0.10.5 (2021-12-06)

* Fixed websocket disconnection not clearing all pending requests.
* Added `OlmMachine.SendRoomKeyRequest` as a more direct way of sending room
  key requests.
* Added automatic Olm session recreation if an incoming message fails to decrypt.
* Changed `Login` to only omit request content from logs if there's a password
  or token (appservice logins don't have sensitive content).

## v0.10.4 (2021-11-25)

* Added `reason` field to invite and unban requests
  (thanks to [@ptman] in [#48]).
* Fixed `AppService.HasWebsocket()` returning `true` even after websocket has
  disconnected.

[@ptman]: https://github.com/ptman
[#48]: https://github.com/mautrix/go/pull/48

## v0.10.3 (2021-11-18)

* Added logs about incoming appservice transactions.
* Added support for message send checkpoints (as HTTP requests, similar to the
  bridge state reporting system).

## v0.10.2 (2021-11-15)

* Added utility method for finding the first supported login flow matching any
  of the given types.
* Updated registering appservice ghosts to use `inhibit_login` flag to prevent
  lots of unnecessary access tokens from being created.
  * If you want to log in as an appservice ghost, you should use [MSC2778]'s
    appservice login (e.g. like [mautrix-whatsapp does for e2be](https://github.com/mautrix/whatsapp/blob/v0.2.1/crypto.go#L143-L149)).

## v0.10.1 (2021-11-05)

* Removed direct dependency on `pq`
  * In order to use some more efficient queries on postgres, you must set
    `crypto.PostgresArrayWrapper = pq.Array` if you want to use both postgres
    and e2ee.
* Added temporary hack to ignore state events with the MSC2716 historical flag
  (to be removed after [matrix-org/synapse#11265] is merged)
* Added received transaction acknowledgements for websocket appservice
  transactions.
* Added automatic fallback to move `prev_content` from top level to the
  standard location inside `unsigned`.

[matrix-org/synapse#11265]: https://github.com/matrix-org/synapse/pull/11265

## v0.9.31 (2021-10-27)

* Added `SetEdit` utility function for `MessageEventContent`.

## v0.9.30 (2021-10-26)

* Added wrapper for [MSC2716]'s `/batch_send` endpoint.
* Added `MarshalJSON` method for `Event` struct to prevent empty unsigned
  structs from being included in the JSON.

[MSC2716]: https://github.com/matrix-org/matrix-spec-proposals/pull/2716

## v0.9.29 (2021-09-30)

* Added `client.State` method to get full room state.
* Added bridge info structs and event types ([MSC2346]).
* Made response handling more customizable.
* Fixed type of `AuthType` constants.

[MSC2346]: https://github.com/matrix-org/matrix-spec-proposals/pull/2346

## v0.9.28 (2021-09-30)

* Added `X-Mautrix-Process-ID` to appservice websocket headers to help debug
  issues where multiple instances are connecting to the server at the same time.

## v0.9.27 (2021-09-23)

* Fixed Go 1.14 compatibility (broken in v0.9.25).
* Added GitHub actions CI to build, test and check formatting on Go 1.14-1.17.

## v0.9.26 (2021-09-21)

* Added default no-op logger to `Client` in order to prevent panic when the
  application doesn't set a logger.

## v0.9.25 (2021-09-19)

* Disabled logging request JSON for sensitive requests like `/login`,
  `/register` and other UIA endpoints. Logging can still be enabled by
  setting `MAUTRIX_LOG_SENSITIVE_CONTENT` to `yes`.
* Added option to store new homeserver URL from `/login` response well-known data.
* Added option to stream big sync responses via disk to maybe reduce memory usage.
* Fixed trailing slashes in homeserver URL breaking all requests.

## v0.9.24 (2021-09-03)

* Added write deadline for appservice websocket connection.

## v0.9.23 (2021-08-31)

* Fixed storing e2ee key withheld events in the SQL store.

## v0.9.22 (2021-08-30)

* Updated appservice handler to cache multiple recent transaction IDs
  instead of only the most recent one.

## v0.9.21 (2021-08-25)

* Added liveness and readiness endpoints to appservices.
  * The endpoints are the same as mautrix-python:
    `/_matrix/mau/live` and `/_matrix/mau/ready`
  * Liveness always returns 200 and an empty JSON object by default,
    but it can be turned off by setting `appservice.Live` to `false`.
  * Readiness defaults to returning 500, and it can be switched to 200
    by setting `appservice.Ready` to `true`.

## v0.9.20 (2021-08-19)

* Added crypto store migration for converting all `VARCHAR(255)` columns
  to `TEXT` in Postgres databases.

## v0.9.19 (2021-08-17)

* Fixed HTML parser outputting two newlines after paragraph tags.

## v0.9.18 (2021-08-16)

* Added new `BuildURL` method that does the same as `Client.BuildBaseURL`
  but without requiring the `Client` instance.

## v0.9.17 (2021-07-25)

* Fixed handling OTK counts and device lists coming in through the appservice
  transaction websocket.
* Updated OlmMachine to ignore OTK counts intended for other devices.

## v0.9.15 (2021-07-16)

* Added support for [MSC3202] and the to-device part of [MSC2409] in the
  appservice package.
* Added support for sending commands through appservice websocket.
* Changed error message JSON field name in appservice error responses to
  conform with standard Matrix errors (`message` -> `error`).

[MSC3202]: https://github.com/matrix-org/matrix-spec-proposals/pull/3202

## v0.9.14 (2021-06-17)

* Added default implementation of `PillConverter` in HTML parser utility.

## v0.9.13 (2021-06-15)

* Added support for parsing and generating encoded matrix.to URLs and `matrix:` URIs ([MSC2312](https://github.com/matrix-org/matrix-doc/pull/2312)).
* Updated HTML parser to use new URI parser for parsing user/room pills.

## v0.9.12 (2021-05-18)

* Added new method for sending custom data with read receipts
  (not currently a part of the spec).

## v0.9.11 (2021-05-12)

* Improved debug log for unsupported event types.
* Added VoIP events to GuessClass.
* Added support for parsing strings in VoIP event version field.

## v0.9.10 (2021-04-29)

* Fixed `format.RenderMarkdown()` still allowing HTML when both `allowHTML`
  and `allowMarkdown` are `false`.

## v0.9.9 (2021-04-26)

* Updated appservice `StartWebsocket` to return websocket close info.

## v0.9.8 (2021-04-20)

* Added methods for getting room tags and account data.

## v0.9.7 (2021-04-19)

* **Breaking change (crypto):** `SendEncryptedToDevice` now requires an event
  type parameter. Previously it only allowed sending events of type
  `event.ToDeviceForwardedRoomKey`.
* Added content structs for VoIP events.
* Added global mutex for Olm decryption
  (previously it was only used for encryption).

## v0.9.6 (2021-04-15)

* Added option to retry all HTTP requests when encountering a HTTP network
  error or gateway error response (502/503/504)
  * Disabled by default, you need to set the `DefaultHTTPRetries` field in
    the `AppService` or `Client` struct to enable.
  * Can also be enabled with `FullRequest`s `MaxAttempts` field.

## v0.9.5 (2021-04-06)

* Reverted update of `golang.org/x/sys` which broke Go 1.14 / darwin/arm.

## v0.9.4 (2021-04-06)

* Switched appservices to using shared `http.Client` instance with a in-memory
  cookie jar.

## v0.9.3 (2021-03-26)

* Made user agent headers easier to configure.
* Improved logging when receiving weird/unhandled to-device events.

## v0.9.2 (2021-03-15)

* Fixed type of presence state constants (thanks to [@babolivier] in [#30]).
* Implemented presence state fetching methods (thanks to [@babolivier] in [#29]).
* Added support for sending and receiving commands via appservice transaction websocket.

[@babolivier]: https://github.com/babolivier
[#29]: https://github.com/mautrix/go/pull/29
[#30]: https://github.com/mautrix/go/pull/30

## v0.9.1 (2021-03-11)

* Fixed appservice register request hiding actual errors due to UIA error handling.

## v0.9.0 (2021-03-04)

* **Breaking change (manual API requests):** `MakeFullRequest` now takes a
  `FullRequest` struct instead of individual parameters. `MakeRequest`'s
  parameters are unchanged.
* **Breaking change (manual /sync):** `SyncRequest` now requires a `Context`
  parameter.
* **Breaking change (end-to-bridge encryption):**
  the `uk.half-shot.msc2778.login.application_service` constant used for
  appservice login ([MSC2778]) was renamed from `AuthTypeAppservice`
  to `AuthTypeHalfyAppservice`.
  * The `AuthTypeAppservice` constant now contains `m.login.application_service`,
    which is currently only used for registrations, but will also be used for
    login once MSC2778 lands in the spec.
* Fixed appservice registration requests to include `m.login.application_service`
  as the `type` (re [matrix-org/synapse#9548]).
* Added wrapper for `/logout/all`.

[MSC2778]: https://github.com/matrix-org/matrix-spec-proposals/pull/2778
[matrix-org/synapse#9548]: https://github.com/matrix-org/synapse/pull/9548

## v0.8.6 (2021-03-02)

* Added client-side timeout to `mautrix.Client`'s `http.Client`
  (defaults to 3 minutes).
* Updated maulogger to fix bug where plaintext file logs wouldn't have newlines.

## v0.8.5 (2021-02-26)

* Fixed potential concurrent map writes in appservice `Client` and `Intent`
  methods.

## v0.8.4 (2021-02-24)

* Added option to output appservice logs as JSON.
* Added new methods for validating user ID localparts.

## v0.8.3 (2021-02-21)

* Allowed empty content URIs in parser
* Added functions for device management endpoints
  (thanks to [@edwargix] in [#26]).

[@edwargix]: https://github.com/edwargix
[#26]: https://github.com/mautrix/go/pull/26

## v0.8.2 (2021-02-09)

* Fixed error when removing the user's avatar.

## v0.8.1 (2021-02-09)

* Added AccountDataStore to remove the need for persistent local storage other
  than the access token (thanks to [@daenney] in [#23]).
* Added support for receiving appservice transactions over websocket.
  See <https://github.com/mautrix/wsproxy> for the server-side implementation.
* Fixed error when removing the room avatar.

[@daenney]: https://github.com/daenney
[#23]: https://github.com/mautrix/go/pull/23

## v0.8.0 (2020-12-24)

* **Breaking change:** the `RateLimited` field in the `Registration` struct is
  now a pointer, so that it can be omitted entirely.
* Merged initial SSSS/cross-signing code by [@nikofil]. Interactive verification
  doesn't work, but the other things mostly do.
* Added support for authorization header auth in appservices ([MSC2832]).
* Added support for receiving ephemeral events directly ([MSC2409]).
* Fixed `SendReaction()` and other similar methods in the `Client` struct.
* Fixed crypto cgo code panicking in Go 1.15.3+.
* Fixed olm session locks sometime getting deadlocked.

[MSC2832]: https://github.com/matrix-org/matrix-spec-proposals/pull/2832
[MSC2409]: https://github.com/matrix-org/matrix-spec-proposals/pull/2409
[@nikofil]: https://github.com/nikofil
