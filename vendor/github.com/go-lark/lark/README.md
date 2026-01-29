# go-lark

[![build](https://github.com/go-lark/lark/actions/workflows/ci.yml/badge.svg)](https://github.com/go-lark/lark/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/go-lark/lark/branch/main/graph/badge.svg)](https://codecov.io/gh/go-lark/lark)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-lark/lark)](https://goreportcard.com/report/github.com/go-lark/lark)
[![Go Module](https://badge.fury.io/go/github.com%2Fgo-lark%2Flark.svg)](https://badge.fury.io/go/github.com%2Fgo-lark%2Flark.svg)
[![Go Reference](https://pkg.go.dev/badge/github.com/go-lark/lark.svg)](https://pkg.go.dev/github.com/go-lark/lark)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)

[简体中文](/README_zhCN.md)

go-lark is an easy-to-use SDK for Feishu and Lark Open Platform,
which implements messaging APIs, with full-fledged supports on building Chat Bot and Notification Bot.

It is widely used and tested by ~650 ByteDance in-house developers with over 3k Go packages.

## Features

- Notification bot & chat bot supported
- Send messages (Group, Private, Rich Text, and Card)
- Quick to build message with `MsgBuffer`
- Easy to create incoming message hook
- Encryption and token verification supported
- Middleware support for Gin & Hertz web framework
- Highly extensible
- Documentation & tests

## Installation

```shell
go get github.com/go-lark/lark
```

## Quick Start

### Prerequisite

There are two types of bot that is supported by go-lark. We need to create a bot manually.

Chat Bot:

- Feishu: create from [Feishu Open Platform](https://open.feishu.cn/).
- Lark: create from [Lark Developer](https://open.larksuite.com/).
- App ID and App Secret are required to init a `ChatBot`.

Notification Bot:

- Create from group chat.
- Web Hook URL is required.

### Sending Message

Chat Bot:

```go
import "github.com/go-lark/lark"

func main() {
    bot := lark.NewChatBot("<App ID>", "<App Secret>")
    bot.StartHeartbeat()
    bot.PostText("hello, world", lark.WithEmail("someone@example.com"))
}
```

Notification Bot:

```go
import "github.com/go-lark/lark"

func main() {
    bot := lark.NewNotificationBot("<WEB HOOK URL>")
    bot.PostNotificationV2(lark.NewMsgBuffer(lark.MsgText).Text("hello, wolrd").Build())
}
```

Feishu/Lark API offers more features, please refers to [Usage](#usage) for further documentation.

## Limits

- go-lark is tested on Feishu endpoints, which literally compats Lark endpoints,
  because Feishu and Lark basically shares the same API specification.
  We do not guarantee all of the APIs work well with Lark, until we have tested it on Lark.
- go-lark only supports Custom App. Marketplace App is not supported yet.
- go-lark implements messaging, group chat, and bot API, other APIs such as Lark Doc, Calendar and so on are not supported.

### Switch to Lark Endpoints

The default API endpoints are for Feishu, in order to switch to Lark, we should use `SetDomain`:

```go
bot := lark.NewChatBot("<App ID>", "<App Secret>")
bot.SetDomain(lark.DomainLark)
```

## Usage

### Auth

Auto-renewable authentication:

```go
// initialize a chat bot with appID and appSecret
bot := lark.NewChatBot(appID, appSecret)
// Renew access token periodically
bot.StartHeartbeat()
// Stop renewal
bot.StopHeartbeat()
```

Single-pass authentication:

```go
bot := lark.NewChatBot(appID, appSecret)
resp, err := bot.GetTenantAccessTokenInternal(true)
// and we can now access the token value with `bot.TenantAccessToken()`
```

Example: [examples/auth](https://github.com/go-lark/examples/tree/main/auth)

### Messaging

For Chat Bot, we can send simple messages with the following method:

- `PostText`
- `PostTextMention`
- `PostTextMentionAll`
- `PostImage`
- `PostShareChatCard`
- `ReplyMessage`
- `AddReaction`
- `DeleteReaction`

Basic message examples: [examples/basic-message](https://github.com/go-lark/examples/tree/main/basic-message)

To build rich messages, we may use [Message Buffer](#message-buffer) (or simply `MsgBuffer`),
which builds message conveniently with chaining methods.

### Examples

Apart from the general auth and messaging chapter, there are comprehensive examples for almost all APIs.
Here is a collection of ready-to-run examples for each part of `go-lark`:

- [examples/auth](https://github.com/go-lark/examples/tree/main/auth)
- [examples/basic-message](https://github.com/go-lark/examples/tree/main/basic-message)
- [examples/rich-text-message](https://github.com/go-lark/examples/tree/main/rich-text-message)
- [examples/interactive-message](https://github.com/go-lark/examples/tree/main/interactive-message)
- [examples/image-message](https://github.com/go-lark/examples/tree/main/image-message)
- [examples/share-chat](https://github.com/go-lark/examples/tree/main/share-chat)
- [examples/group](https://github.com/go-lark/examples/tree/main/group)

### Message Buffer

We can build message body with `MsgBuffer` and send with `PostMessage`, which supports the following message types:

- `MsgText`: Text
- `MsgPost`: Rich Text
- `MsgInteractive`: Interactive Card
- `MsgShareCard`: Group Share Card
- `MsgShareUser`: User Share Card
- `MsgImage`: Image
- `MsgFile`: File
- `MsgAudio`: Audio
- `MsgMedia`: Media
- `MsgSticker`: Sticker

`MsgBuffer` provides binding functions and content functions.

Binding functions:

| Function    | Usage               | Comment                                                                     |
| ----------- | ------------------- | --------------------------------------------------------------------------- |
| BindChatID  | Bind a chat ID      | Either `OpenID`, `UserID`, `Email`, `ChatID` or `UnionID` should be present |
| BindOpenID  | Bind a user open ID |                                                                             |
| BindUserID  | Bind a user ID      |                                                                             |
| BindUnionID | Bind a union ID     |                                                                             |
| BindEmail   | Bind a user email   |                                                                             |
| BindReply   | Bind a reply ID     | Required when reply a message                                               |

Content functions pair with message content types. If it mismatched, it would not have sent successfully.
Content functions:

| Function  | Message Type     | Usage                   | Comment                                                          |
| --------- | ---------------- | ----------------------- | ---------------------------------------------------------------- |
| Text      | `MsgText`        | Append plain text       | May build with `TextBuilder`                                     |
| Post      | `MsgPost`        | Append rich text        | May build with `PostBuilder`                                     |
| Card      | `MsgInteractive` | Append interactive card | May build with [`CardBuilder`](card/README.md)                   |
| Template  | `MsgInteractive` | Append card template    | Required to build with [CardKit](https://open.feishu.cn/cardkit) |
| ShareChat | `MsgShareCard`   | Append group share card |                                                                  |
| ShareUser | `MsgShareUser`   | Append user share card  |                                                                  |
| Image     | `MsgImage`       | Append image            | Required to upload to Lark server in advance                     |
| File      | `MsgFile`        | Append file             | Required to upload to Lark server in advance                     |
| Audio     | `MsgAudio`       | Append audio            | Required to upload to Lark server in advance                     |
| Media     | `MsgMedia`       | Append media            | Required to upload to Lark server in advance                     |
| Sticker   | `MsgSticker`     | Append sticker          | Required to upload to Lark server in advance                     |

### Error Handling

Each `go-lark` API function returns `response` and `err`.
`err` is the error from HTTP client, when it was not `nil`, HTTP might have gone wrong.

While `response` is HTTP response from Lark API server, in which `Code` and `OK` represent whether it succeeds.
The meaning of `Code` is defined [here](https://open.feishu.cn/document/ukTMukTMukTM/ugjM14COyUjL4ITN).

### Event

Lark provides a number of [events](https://open.feishu.cn/document/ukTMukTMukTM/uUTNz4SN1MjL1UzM) and they are in two different schema (1.0/2.0).
go-lark now only implements a few of them, which are needed for interacting between bot and Lark server:

- URL Challenge
- Receiving Messages

We recommend HTTP middlewares to handle these events.

### Middlewares

We have already implemented HTTP middlewares to support event handling:

- [Gin Middleware](https://github.com/go-lark/lark-gin)
- [Hertz Middleware](https://github.com/go-lark/lark-hertz)

Example: [examples/gin-middleware](https://github.com/go-lark/examples/tree/main/gin-middleware) [examples/hertz-middleware](https://github.com/go-lark/examples/tree/main/hertz-middleware)

#### URL Challenge

```go
r := gin.Default()
middleware := larkgin.NewLarkMiddleware()
middleware.BindURLPrefix("/handle") // supposed URL is http://your.domain.com/handle
r.Use(middleware.LarkChallengeHandler())
```

#### Event V2

Lark has provided event v2 and it applied automatically to newly created bots.

```go
r := gin.Default()
middleware := larkgin.NewLarkMiddleware()
r.Use(middleware.LarkEventHandler())
```

Get the event (e.g. Message):

```go
r.POST("/", func(c *gin.Context) {
    if evt, ok := middleware.GetEvent(c); ok { // => GetEvent instead of GetMessage
        if evt.Header.EventType == lark.EventTypeMessageReceived {
            if msg, err := evt.GetMessageReceived(); err == nil {
                fmt.Println(msg.Message.Content)
            }
        }
    }
})
```

#### Card Callback

We may also setup callback for card actions (e.g. button). The URL challenge part is the same.

We may use `LarkCardHandler` to handle the actions:

```go
r.Use(middleware.LarkCardHandler())
r.POST("/callback", func(c *gin.Context) {
    if card, ok := middleware.GetCardCallback(c); ok {
    }
})
```

#### Receiving Message (Event V1)

For older bots, please use v1:

```go
r := gin.Default()
middleware := larkgin.NewLarkMiddleware()
middleware.BindURLPrefix("/handle") // supposed URL is http://your.domain.com/handle
r.POST("/handle", func(c *gin.Context) {
    if msg, ok := middleware.GetMessage(c); ok && msg != nil {
        text := msg.Event.Text
        // your awesome logic
    }
})
```

### Security & Encryption

Lark Open Platform offers AES encryption and token verification to ensure security for events.

- AES Encryption: when switch on, all traffic will be encrypted with AES.
- Token Verification: simple token verification for incoming messages.

We recommend you to enable token verification. If HTTPS is not available on your host, then enable AES encryption.

```go
middleware.WithTokenVerfication("<verification-token>")
middleware.WithEncryption("<encryption-key>")
```

### Debugging

Lark does not provide messaging API debugger officially. Thus, we have to debug with real Lark conversation.
We recommend [ngrok](https://ngrok.com/) to debug events.

And we add `PostEvent` to simulate message sending to make it even easier.
`PostEvent` can also be used to redirect events, which acts like a reverse proxy.

## Development

### Test

1. Dotenv Setup

   go-lark uses `godotenv` test locally. You may have to create a `.env` file in repo directory, which contains environmental variables:

   ```bash
   LARK_APP_ID
   LARK_APP_SECRET
   LARK_USER_EMAIL
   LARK_USER_ID
   LARK_UNION_ID
   LARK_OPEN_ID
   LARK_CHAT_ID
   LARK_WEBHOOK_V2
   LARK_WEBHOOK_V2_SIGNED
   ```

   `LARK_APP_ID` and `LARK_APP_SECRET` are mandatory. Others are required only by specific API tests.

2. Run Test

   ```bash
   GO_LARK_TEST_MODE=local ./scripts/test.sh
   ```

### Extensions

go-lark's dev utilities (authentication, HTTP handling, and etc.) are capable for easily implementing most of APIs provided by Lark Open Platform.
And we may use that as an extension for go-lark.

Here is an example that implementing a Lark Doc API with go-lark:

```go
package lark

import "github.com/go-lark/lark"

const copyFileAPIPattern = "/open-apis/drive/explorer/v2/file/copy/files/%s"

// CopyFileResponse .
type CopyFileResponse struct {
	lark.BaseResponse

	Data CopyFileData `json:"data"`
}

// CopyFileData .
type CopyFileData struct {
	FolderToken string `json:"folderToken"`
	Revision    int64  `json:"revision"`
	Token       string `json:"token"`
	Type        string `json:"type"`
	URL         string `json:"url"`
}

// CopyFile implementation
func CopyFile(bot *lark.Bot, fileToken, dstFolderToken, dstName string) (*CopyFileResponse, error) {
	var respData model.CopyFileResponse
	err := bot.PostAPIRequest(
		"CopyFile",
		fmt.Sprintf(copyFileAPIPattern, fileToken),
		true,
		map[string]interface{}{
			"type":             "doc",
			"dstFolderToken":   dstFolderToken,
			"dstName":          dstName,
			"permissionNeeded": true,
			"CommentNeeded":    false,
		},
		&respData,
	)
	return &respData, err
}
```

## FAQ

- I got `99991401` when sending messages
  - remove IP Whitelist from dashboard
- My bot failed sending messages
  1. check authentication.
  2. not invite to the group.
  3. API permission not applied.
- Does go-lark support interactive message card?
  - Yes, use a CardBuilder.

## Contributing

- If you think you've found a bug with go-lark, please file an issue.
- Pull Request is welcomed.

## License

Copyright (c) David Zhang, 2018-2024. Licensed under MIT License.
