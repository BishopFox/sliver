# 智能对话

[官方文档](https://developers.weixin.qq.com/doc/aispeech/confapi/dialog/token.html)

## 快速入门

```go
import (
	"github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/aispeech/config"
	"github.com/silenceper/wechat/v2/aispeech/dialog"
	"github.com/silenceper/wechat/v2/cache"
)

wc := wechat.NewWechat()
memory := cache.NewMemory()

ai := wc.GetAISpeech(&config.Config{
	AppID:   "xxx",
	Token:   "xxx",
	AESKey:  "xxx",
	Account: "admin",
	Cache:   memory,
})

dialogClient := ai.GetDialog()

accessToken, err := dialogClient.GetAccessToken()
if err != nil {
	return err
}

res, err := dialogClient.Query(&dialog.QueryRequest{
	Query:  "你好",
	Env:    "online",
	UserID: "user-1",
})
```

## 简单问答导入与发布

```go
task, err := dialogClient.ImportJSON(&dialog.ImportJSONRequest{
	Mode: 0,
	Data: []dialog.BotIntent{{
		Skill:     "售前咨询",
		Intent:    "查询营业时间",
		Disable:   false,
		Questions: []string{"你们几点开门", "营业时间是什么时候"},
		Answers:   []string{"我们的营业时间是周一至周五 9:00-18:00"},
	}},
})
if err != nil {
	return err
}

asyncResult, err := dialogClient.FetchAsync(&dialog.FetchAsyncRequest{
	TaskID: task.TaskID,
})
if err != nil {
	return err
}

publish, err := dialogClient.Publish()
if err != nil {
	return err
}

progress, err := dialogClient.GetEffectiveProgress(&dialog.EffectiveProgressRequest{
	Env: "online",
})
```

当前封装微信智能对话开放接口中“接入智能对话”的 6 个接口：

- `POST /v2/token`
- `POST /v2/bot/import/json`
- `POST /v2/async/fetch`
- `POST /v2/bot/publish`
- `POST /v2/bot/effective_progress`
- `POST /v2/bot/query`

## 真实接口验证

默认单元测试不会访问真实微信接口。如需验证 token 和 query 只读链路，可设置以下环境变量后运行：

```bash
AISPEECH_INTEGRATION=1
AISPEECH_APPID=xxx
AISPEECH_TOKEN=xxx
AISPEECH_AES_KEY=xxx
go test ./aispeech/dialog -run TestIntegrationAccessTokenAndQuery
```

如需验证导入、异步任务查询、发布、发布进度查询和命中回答的完整链路，可显式开启变更型测试。该测试会导入一个 `CodexSmokeTest` 临时问答并发布到线上机器人：

```bash
AISPEECH_INTEGRATION_MUTATION=1
AISPEECH_APPID=xxx
AISPEECH_TOKEN=xxx
AISPEECH_AES_KEY=xxx
go test ./aispeech/dialog -run TestIntegrationFullDialogFlow -v
```
