/*
Package lark provides message notification integration for Lark. Two kinds of
bots on Lark are supported -- webhooks and custom apps. For information on
webhook bots, see
https://open.larksuite.com/document/uAjLw4CM/ukTMukTMukTM/bot-v3/use-custom-bots-in-a-group,
and for info on custom apps, see
https://open.larksuite.com/document/home/develop-a-bot-in-5-minutes/create-an-app.

Usage:

	package main

	import (
	  "context"
	  "log"

	  "github.com/nikoksr/notify"
	  "github.com/nikoksr/notify/service/lark"
	)

	const (
	  webhookURL = "https://open.feishu.cn/open-apis/bot/v2/hook/xxx"
	  appId      = "xxx"
	  appSecret  = "xxx"
	)

	func main() {
	  // Two types of services are available depending on your requirements.
	  larkWebhookService := lark.NewWebhookService(webhookURL)
	  larkCustomAppService := lark.NewCustomAppService(appId, appSecret)

	  // Lark implements five types of receiver IDs. You'll need to specify the
	  // type using the respective helper functions when adding them as receivers
	  // for the custom app service.
	  larkCustomAppService.AddReceivers(
	    lark.OpenID("xxx"),
	    lark.UserID("xxx"),
	    lark.UnionID("xxx"),
	    lark.Email("xyz@example.com"),
	    lark.ChatID("xxx"),
	  )

	  notifier := notify.New()
	  notifier.UseServices(larkWebhookService, larkCustomAppService)

	  if err := notifier.Send(context.Background(), "subject", "message"); err != nil {
	    log.Fatalf("notifier.Send() failed: %s", err.Error())
	  }

	  log.Println("notification sent")
	}
*/
package lark
