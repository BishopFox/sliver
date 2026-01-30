# CHANGELOG

## v1.16.0

- feat(base): add base error (#78)
- feat(im): support buzz message (#77)

## v1.15.0

- feat(im): support card template (#75)

## v1.14.1

- feat(api): UploadFile support uploads binary file

## v1.14.0

- refactor(im): build reply and update message with standalone methods
- feat(im): UpdateMessage supports text and post in addition to card
- feat(im): ReplyMessage supports reply in thread

## v1.13.3

- feat(event): support message_recalled, message_reaction_created, and message_reaction_deleted

## v1.13.2

- fix(event): chat_id and message_id

## v1.13.1

- feat(event): support card callback (#68)

## v1.13.0

- feat(contact): support user info (#67)

## v1.12.0

- feat(im): support forward message
- feat(im): change reaction API names (#66)

## v1.11.0

- feat(im): support reactions (#62)
- ci: run tests under private tenant to avoid applying permissions (#65)

## v1.10.2

- fix(notification): remove chat_id and uid_type from outcoming message (#63)
- fix(notification): timestamp should be string
- feat(im): drop v1 update_multi

## v1.10.1

- feat(chat): support chat list and chat (#60)

## v1.10.0

- feat(im): support card with i18n (#59)

## v1.9.0

- feat(im): support column set (#57)

## v1.8.0

- feat(im): IMMessage use Sender rather than Sendor (#54)
- feat(chat): set and delete notice (#53)
- feat(im): pin and unpin message (#47)
- fix: update chat avatar field (#46)
- feat(event): add more events (#38)
- fix(sign): fix missing timestamp in signed message

## v1.7.4

- feat(message): support UUID (#40)

## v1.7.3

- feat(notification): support sign

## v1.7.2

- fix(http): remove shared context for requests
- feat: improve heartbeat
- feat(notification): allow update url

## v1.7.1

- feat(card): support update_multi in config

## v1.7.0

- feat(message): support update message card (#31)
- feat(chat): add more chat APIs (#29)
- feat: support event v2 (#4)
- fix(chat): allow set user id type
- feat: api im (#19)

## v1.6.1

- docs: add extension guide [ci skip] (#18)

## v1.6.0

- chore: recall message uses base response
- feat: delete ephemeral message
- feat: add ephemeral card
- docs: add goreportcard [ci skip]

## v1.5.0

- fix: div.text, option.text & tests (#13)
- feat: card builder (#12)
- feat(auth): make heartbeat thread safe (#10)
- chore: add editorconfig [ci skip]
- ci: switch to revive

## v1.4.2

- feat: add domain method

## v1.4.1

- fix: should pass http header
- feat: add context to logger & client (#9)

## v1.3.0

- refactor: better logger interface (#8)

## v1.2.1

- fix: http custom client test
- refactor: improve http wrapper (#7)

## v1.1.0

- feat: open more api (#6)
- feat(message/post): patch i18n support
- docs: add godoc badge [ci skip]
- ci: add codecov (#2)
- chore: add .vimrc [ci skip]
- feat: add alternative domains

## v1.0.1

- feat: drop api user v4

## v1.0.0

- feat: init go-lark/lark oss version
