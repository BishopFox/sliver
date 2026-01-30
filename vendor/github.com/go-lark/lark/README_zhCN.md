# go-lark

[![build](https://github.com/go-lark/lark/actions/workflows/ci.yml/badge.svg)](https://github.com/go-lark/lark/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/go-lark/lark/branch/main/graph/badge.svg)](https://codecov.io/gh/go-lark/lark)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-lark/lark)](https://goreportcard.com/report/github.com/go-lark/lark)
[![Go Module](https://badge.fury.io/go/github.com%2Fgo-lark%2Flark.svg)](https://badge.fury.io/go/github.com%2Fgo-lark%2Flark.svg)
[![Go Reference](https://pkg.go.dev/badge/github.com/go-lark/lark.svg)](https://pkg.go.dev/github.com/go-lark/lark)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)

一个简单、开发者友好的 Lark 开放平台机器人 SDK。

## 介绍

go-lark 主要实现了消息类 API，提供完整的聊天机器人和通知机器人支持。在字节跳动公司内部得到广泛应用，有大约 450 开发者和超过 1500 个 Go 仓库使用。

## 功能

- 聊天机器人和通知机器人
- 发送各类消息（群发、私聊、富文本、卡片消息）
- 快速消息体构造 `MsgBuffer`
- 一站式解决服务器 Challenge 和聊天消息响应
- 支持加密和校验
- 支持 Gin 和 Hertz 框架中间件
- 高可扩展性
- 文档、测试覆盖

## 安装

```shell
go get github.com/go-lark/lark
```

## 快速入门

### 前置准备

我们支持两种类型的机器人，需要分别用以下方式创建：

聊天机器人：

- 飞书: 通过[飞书开放平台](https://open.feishu.cn/)创建。
- Lark: 通过 [Lark Developer](https://open.larksuite.com/) 创建。
- 需要 App ID 和 App Secret 来初始化 `ChatBot`。

通知机器人：

- 通过群聊创建-群机器人创建。
- 需要使用 WebHook URL。

### 消息发送

聊天机器人：

```go
import "github.com/go-lark/lark"

func main() {
    bot := lark.NewChatBot("<App ID>", "<App Secret>")
    bot.StartHeartbeat()
    bot.PostText("hello, world", lark.WithEmail("someone@example.com"))
}
```

通知机器人：

```go
import "github.com/go-lark/lark"

func main() {
    bot := lark.NewNotificationBot("<WEB HOOK URL>")
    bot.PostNotificationV2(lark.NewMsgBuffer(lark.MsgText).Text("hello, wolrd").Build())
}
```

## 限制

- go-lark 基于飞书域名进行测试，理论上可以完全兼容 Lark 平台（API 定义一致）。但我们不保证在 Lark 下完全可用，因为账户限于，没有专门测试过。
- go-lark 仅支持企业自建应用，不支持应用商店应用（ISV）。
- go-lark 仅实现了消息、群组和机器人 API，对于飞书文档、日历等功能，并不支持。

### 切换到 Lark 域名

go-lark 默认使用飞书 API 域名，我们需要调用`SetDomain`来切换到 Lark：

```go
bot := lark.NewChatBot("<App ID>", "<App Secret>")
bot.SetDomain(lark.DomainLark)
```

## 用法

### 鉴权

自动更新授权：

```go
// initialize a chat bot with appID and appSecret
bot := lark.NewChatBot(appID, appSecret)
// Renew access token periodically
bot.StartHeartbeat()
// Stop renewal
bot.StopHeartbeat()
```

单次授权：

```go
bot := lark.NewChatBot(appID, appSecret)
resp, err := bot.GetTenantAccessTokenInternal(true)
// and we can now access the token value with `bot.TenantAccessToken()`
```

参考实例：[鉴权](https://github.com/go-lark/examples/tree/main/auth)

### 消息

简单消息可以以下接口直接通过：

- `PostText`
- `PostTextMention`
- `PostTextMentionAll`
- `PostImage`
- `PostShareChatCard`
- `ReplyMessage`
- `AddReaction`
- `DeleteReaction`

参考实例：[基本消息](https://github.com/go-lark/examples/tree/main/basic-message)。

对于复杂消息，可以使用 [Message Buffer](#message-buffer) 进行链式构造。

### 参考实例

- [鉴权](https://github.com/go-lark/examples/tree/main/auth)
- [基本消息](https://github.com/go-lark/examples/tree/main/basic-message)
- [富文本消息](https://github.com/go-lark/examples/tree/main/rich-text-message)
- [交互卡片](https://github.com/go-lark/examples/tree/main/interactive-message)
- [图片消息](https://github.com/go-lark/examples/tree/main/image-message)
- [分享群卡片](https://github.com/go-lark/examples/tree/main/share-chat)
- [群操作](https://github.com/go-lark/examples/tree/main/group)

### Message Buffer

发送消息需要先通过 MsgBuffer 构造消息体，然后调用 `PostMessage` 进行发送。

MsgBuffer 支持多种类型的消息：

- `MsgText`：文本
- `MsgPost`：富文本
- `MsgInteractive`：交互式卡片
- `MsgShareUser`: 用户名片
- `MsgShareCard`：群名片
- `MsgImage`：图片
- `MsgFile`: 文件
- `MsgAudio`: 音频
- `MsgMedia`: 媒体
- `MsgSticker`: 表情

MsgBuffer 主要有两类函数，Bind 函数和内容函数。

Bind 函数：

| 函数        | 作用         | 备注                                          |
| ----------- | ------------ | --------------------------------------------- |
| BindChatID  | 绑定 ChatID  | OpenID/UserID/Email/ChatID/UnionID 选一个即可 |
| BindOpenID  | 绑定 OpenID  |                                               |
| BindUserID  | 绑定 UserID  |                                               |
| BindUnionID | 绑定 UnionID |                                               |
| BindEmail   | 绑定邮箱     |                                               |
| BindReply   | 绑定回复     | 回复他人时需要                                |

内容函数大多跟消息类型是强关联的，类型错误不会生效。内容函数：

| 函数      | 适用范围         | 作用             | 备注                                                         |
| --------- | ---------------- | ---------------- | ------------------------------------------------------------ |
| Text      | `MsgText`        | 添加文本内容     | 可使用 `TextBuilder` 构造                                    |
| Post      | `MsgPost`        | 添加富文本内容   | 可使用 `PostBuilder` 构造                                    |
| Card      | `MsgInteractive` | 添加交互式卡片   | 可使用 [`CardBuilder`](card/README_zhCN.md) 构造             |
| Template  | `MsgInteractive` | 添加卡片模板     | 可使用 [可视化搭建工具](https://open.feishu.cn/cardkit) 构造 |
| ShareChat | `MsgShareCard`   | 添加分享群卡片   |                                                              |
| ShareUser | `MsgShareUser`   | 添加分享用户卡片 |                                                              |
| Image     | `MsgImage`       | 添加图片         | 需要先上传到飞书服务器                                       |
| File      | `MsgFile`        | 添加文件         | 需要先上传到飞书服务器                                       |
| Audio     | `MsgAudio`       | 添加音频         | 需要先上传到飞书服务器                                       |
| Media     | `MsgMedia`       | 添加媒体         | 需要先上传到飞书服务器                                       |
| Sticker   | `MsgSticker`     | 添加表情         | 需要先上传到飞书服务器                                       |

### 异常处理

每个 API 都会返回 `response` 和 `error`。`error` 是 HTTP 客户端返回，`response` 是开放平台接口返回。一般来说，每个接口的 `response` 都会有 `code` 字段，如果非 0 则表示有错误。具体错误码含义，请查看[官方文档](https://open.feishu.cn/document/ukTMukTMukTM/ugjM14COyUjL4ITN)。

## 事件处理

事件是飞书机器人用于实现机器人交互的机制，创建聊天机器人后我们并不具有和机器人交互的能力，需要通过开放平台的挑战和消息相应完成交互。

飞书开放平台提供多种事件，并且有两种版本的格式（1.0 和 2.0）。go-lark 只实现了 1.0 中的两种事件。

在开发交互机器人过程中，我们主要需要用到这两类事件：

- URL 挑战
- 接收消息

我们推荐使用 HTTP 中间件处理事件。

### 中间件

我们实现了 Gin 和 Hertz 框架的中间件：

- [Gin Middleware](https://github.com/go-lark/lark-gin)
- [Hertz Middleware](https://github.com/go-lark/lark-hertz)

实例：[examples/gin-middleware](https://github.com/go-lark/examples/tree/main/gin-middleware) [examples/hertz-middleware](https://github.com/go-lark/examples/tree/main/hertz-middleware)

#### URL 挑战

```go
r := gin.Default()
middleware := larkgin.NewLarkMiddleware()
middleware.BindURLPrefix("/handle") // 假设 URL 是 http://your.domain.com/handle
r.Use(middleware.LarkChallengeHandler())
```

#### 事件 2.0

飞书开放平台默认事件类似目前 v2，会自动在新创建的机器人中启用。

```go
r := gin.Default()
middleware := larkgin.NewLarkMiddleware()
r.Use(middleware.LarkEventHandler())
```

获取事件详情：

```go
r.POST("/", func(c *gin.Context) {
    if evt, ok := middleware.GetEvent(c); ok { // => GetEvent instead of GetMessage
        if evt.Header.EventType == lark.EventTypeMessageReceived {
            if msg, err := evt.GetMessageReceived(); err == nil {
                fmt.Println(msg.Message.Content)
            }
            // you may have to parse other events
        }
    }
})
```

#### 卡片回调

我们可以使用卡片回调接受卡片的用户操作（如：按钮点击），URL 挑战部分同步上。

我们可以使用 `LarkCardHandler` 来接受操作事件：

```go
r.Use(middleware.LarkCardHandler())
r.POST("/callback", func(c *gin.Context) {
    if card, ok := middleware.GetCardCallback(c); ok {
    }
})
```

#### 接收消息（事件 1.0）

对于较早常见的机器人，我们需要使用 v1 版本：

```go
r := gin.Default()
middleware := larkgin.NewLarkMiddleware()
middleware.BindURLPrefix("/handle") // supposed URL is http://your.domain.com/handle
r.POST("/handle", func(c *gin.Context) {
    if msg, ok := middleware.GetMessage(c); ok && msg != nil {
        text := msg.Event.Text
        // 你的业务逻辑
    }
})
```

### 加密安全

飞书开放平台目前有两种加密安全策略（可以同时启用），分别是 AES 加密和 Token 校验。

- AES 加密：需要在验证 Challenge 时就开启，此后所有收到的消息都会走 AES 加密。
- Token 校验：验证消息来自 Lark 开放平台。

我们建议开启 Token 校验。如果没有使用 HTTPS 协议，则开启 AES。

```go
middleware.WithTokenVerfication("<verification-token>")
middleware.WithEncryption("<encryption-key>")
```

### 调试

飞书官方没有提供发消息工具，如果测试消息交互的话不得不在飞书上发消息，直接在“线上” URL 调试，很不方便。
推荐使用 [ngrok](https://ngrok.com/) 进行代理调试。

同时，我们还加入了线下模拟消息事件的 `PostEvent`，通过它可以在任何地方进行调试。当然，模拟消息的包体需要自己构造。`PostEvent` 也可以用于事件转发，对事件进行反向代理。

## 开发

### 测试

1. Dotenv 配置

   go-lark 使用 `godotenv` 进行本地测试。测试前需要在代码目录下创建一个 `.env` 文件，包含如下环境变量：

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

   其中，`LARK_APP_ID` 和 `LARK_APP_SECRET` 必须配置，其它字段根据不同的测试可选择配置。

2. 运行测试

   ```bash
   GO_LARK_TEST_MODE=local ./scripts/test.sh
   ```

### 扩展

go-lark 的开发设施（鉴权、HTTP 处理等）可以很方便的用来实现大部分开放平台提供的 API 能力。我们可以通过这种方式扩展 go-lark。

这里是一个使用 go-lark 扩展实现飞书文档 API 的例子：

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

- 调用接口发消息报错，错误码 99991401
  - 在开发者后台“安全”中取消“IP 白名单”
- 机器人发消息失败了
  - 常见原因：1，忘了开启授权；2，没进群发群消息；3，其它权限类问题
- go-lark 可以发消息卡片吗？怎么发？
  - 可以，可以使用 CardBuilder 构建

## 贡献

- 如果在使用 go-lark 时遇到 Bug，请提交 Issue。
- 欢迎通过 Pull Request 提交功能或 Bug 修复。

## 协议

Copyright (c) David Zhang, 2018-2024. Licensed under MIT License.
