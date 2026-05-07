# WeChat SDK for Go

![Go](https://github.com/silenceper/wechat/actions/workflows/go.yml/badge.svg?branch=v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/silenceper/wechat/v2)](https://goreportcard.com/report/github.com/silenceper/wechat/v2)
[![pkg](https://img.shields.io/badge/dev-reference-007d9c?logo=go&logoColor=white&style=flat)](https://pkg.go.dev/github.com/silenceper/wechat/v2?tab=doc)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/silenceper/wechat?sort=semver)
![star](https://gitcode.com/silenceper/wechat/star/badge.svg)


使用Golang开发的微信SDK，简单、易用。

## 文档 && 例子

[API列表](https://github.com/silenceper/wechat/tree/v2/doc/api)

[Wechat SDK 2.0 文档](https://silenceper.com/wechat)

[Wechat SDK 2.0 例子](https://github.com/gowechat/example)

## 快速开始

```
import "github.com/silenceper/wechat/v2"
```

以下是一个微信公众号处理消息接收以及回复的例子：

```go
// 使用memcache保存access_token，也可选择redis或自定义cache
wc := wechat.NewWechat()
memory := cache.NewMemory()
cfg := &offConfig.Config{
    AppID:     "xxx",
    AppSecret: "xxx",
    Token:     "xxx",
    // EncodingAESKey: "xxxx",
    Cache: memory,
}
officialAccount := wc.GetOfficialAccount(cfg)

// 传入request和responseWriter
server := officialAccount.GetServer(req, rw)
// 设置接收消息的处理方法
server.SetMessageHandler(func(msg *message.MixMessage) *message.Reply {

    // 回复消息：演示回复用户发送的消息
    text := message.NewText(msg.Content)
    return &message.Reply{MsgType: message.MsgTypeText, MsgData: text}
})

// 处理消息接收以及回复
err := server.Serve()
if err != nil {
    fmt.Println(err)
    return
}
// 发送回复的消息
server.Send()

```

## 目录说明

- officialaccount: 微信公众号API
- miniprogram: 小程序API
- minigame:小游戏API
- pay:微信支付API
- openplatform:开放平台API
- work:企业微信
- aispeech:智能对话
- doc: api文档

## 贡献

- 在[API列表](https://github.com/silenceper/wechat/tree/v2/doc/api)中查看哪些API未实现
- 提交issue，描述需要贡献的内容
- 完成更改后，提交PR


## 感谢以下贡献者

<a href="https://opencollective.com/gowechat"><img src="https://opencollective.com/gowechat/contributors.svg?width=890" /></a>


## 作者公众号

![img](https://silenceper.oss-cn-beijing.aliyuncs.com/qrcode/search_study_program.png)

## License

Apache License, Version 2.0
