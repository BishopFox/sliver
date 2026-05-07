/*
Package viber provides a service for sending messages to viber.

Usage:

	package main

	import (
	    "context"
	    "log"

	    "github.com/nikoksr/notify"
	    "github.com/nikoksr/notify/service/viber"
	)

	const appKey = "your-viber-token"
	const webhookURL = "https://webhook.com"
	const senderName = "vibersofyana"

	func main() {
	    viberSvc := viber.New(appKey, senderName, "")

	    err := viberSvc.SetWebhook(webhookURL) // this only needs to be called once
	    if err != nil {
	        log.Fatalf("set webhook to viber server failed: %v", err)
	    }

	    viberSvc.AddReceivers("receiver-viber-user-id") // can add as many as required
	    notifier := notify.New()

	    notifier.UseServices(viberSvc)
	    if err := notifier.Send(context.Background(), "TEST", "Message using golang notifier library"); err != nil {
	        log.Fatalf("notifier.Send() failed: %s", err.Error())
	    }

	    log.Println("Notification sent")
	}
*/
package viber
