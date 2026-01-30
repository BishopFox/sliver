# WeChat

## Prerequisites

To use the service you need to apply for a [WeChat Official Account](https://mp.weixin.qq.com).
Application approval is a manual process and might take a while. In the meantime, you
can test your code using the [sandbox](https://developers.weixin.qq.com/sandbox).

You need the following configuration information to sent out text messages with an Official Account:

1. AppID
2. AppSecret
3. Token
4. EncodingAESKey
5. Verification URL

The `AppID` and `AppSecret` are provided to you by WeChat. The `Token`, `EncodingAESKey` and
the Verifications URL, you set yourself and are needed by the authentication method. More on
this [here](https://developers.weixin.qq.com/doc/offiaccount/en/Basic_Information/Access_Overview.html).

## Getting Started

Until your application is approved, sign in to the sandbox to get an `AppID`, an `AppSecret` and
set the `Token` and the Verification URL. Typically, you need a service like [ngrok](https://ngrok.com/)
for the latter. You don't need to/cannot set the `EncodingAESKey`, because it's not required in the
sandbox environment:

![wechat_sandbox_1.png](img/wechat_sandbox_1.png)

You also need a user subscribed to your Official Account. You can use your own:

![wechat_sandbox_2.png](img/wechat_sandbox_2.png)

## Usage

```go
package main

import (
  "github.com/silenceper/wechat/v2/cache"
  "log"
  "context"
  "fmt"
  "net/http"
  "github.com/nikoksr/notify"
  "github.com/nikoksr/notify/service/wechat"
)

func main() {

  wechatSvc := wechat.New(&wechat.Config{
    AppID:          "abcdefghi",
    AppSecret:      "jklmnopqr",
    Token:          "mytoken",
    EncodingAESKey: "IGNORED-IN-SANDBOX",
    Cache:          cache.NewMemory(),
  })

  // do this only once, or when settings are updated
  devMode := true
  wechatSvc.WaitForOneOffVerification(":7999", devMode, func(r *http.Request, verified bool) {
    if !verified {
      fmt.Println("unknown or failed verification call")
    } else {
      fmt.Println("verification call done")
    }
  })

  wechatSvc.AddReceivers("some-user-openid")

  notifier := notify.New()
  notifier.UseServices(wechatSvc)

  err := notifier.Send(context.Background(), "subject", "message")
  if err != nil {
    log.Fatalf("notifier.Send() failed: %s", err.Error())
  }

  log.Println("notification sent")
}
```

![wechat_usage.png](img/wechat_usage.png)

