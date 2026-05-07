# Lark

## Prerequisites

Depending on your requirements, you'll need either a custom app or a Lark group 
chat webhook. The latter is easier to set up, but can only send messages to the 
group it is in. You may refer to the doc 
[here](https://open.larksuite.com/document/uAjLw4CM/ukTMukTMuhttps://open.larksuite.com/document/home/develop-a-bot-in-5-minutes/create-an-appkTM/bot-v3/use-custom-bots-in-a-group) 
to set up a webhook bot, and the doc 
[here](https://open.larksuite.com/document/home/develop-a-bot-in-5-minutes/create-an-app) 
to set up a custom app.

## Usage

### Webhook

For webhook bots, we only need the webhook URL, which might look something like 
`https://open.feishu.cn/open-apis/bot/v2/hook/xxx`. Note that there is no 
method to configure receivers, because the webhook bot can only send messages 
to the group in which it was created.

```go
package main

import (
	"context"
	"log"

	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/lark"
)

// Replace this with your own webhook URL.
const webHookURL = "https://open.feishu.cn/open-apis/bot/v2/hook/xxx"

func main() {
	larkWebhookSvc := lark.NewWebhookService(webHookURL)

	notifier := notify.New()
	notifier.UseServices(larkWebhookSvc)

	if err := notifier.Send(context.Background(), "subject", "message"); err != nil {
		log.Fatalf("notifier.Send() failed: %s", err.Error())
	}

	log.Println("notification sent")
}
```

### Custom App

For custom apps, we need to pass in the App ID and App Secret when creating a 
new notification service. When adding receivers, the type of the receiver ID 
must be specified, as shown in the example below. You may refer to the section 
entitled "Query parameters" in the doc 
[here](https://open.larksuite.com/document/uAjLw4CM/ukTMukTMukTM/reference/im-v1/message/create) 
for more information about the different ID types.

```go
package main

import (
	"context"
	"log"

	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/lark"
)

// Replace these with the credentials from your custom app.
const (
	appId     = "xxx"
	appSecret = "xxx"
)

func main() {
	larkCustomAppService := lark.NewCustomAppService(appId, appSecret)

	// Lark implements five types of receiver IDs. You'll need to specify the
	// type using the respective helper functions when adding them as receivers.
	larkCustomAppService.AddReceivers(
		lark.OpenID("xxx"),
		lark.UserID("xxx"),
		lark.UnionID("xxx"),
		lark.Email("xyz@example.com"),
		lark.ChatID("xxx"),
	)

	notifier := notify.New()
	notifier.UseServices(larkCustomAppService)

	if err := notifier.Send(context.Background(), "subject", "message"); err != nil {
		log.Fatalf("notifier.Send() failed: %s", err.Error())
	}

	log.Println("notification sent")
}
```

