# Webpush Notifications

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat)](https://pkg.go.dev/github.com/nikoksr/notify/service/webpush)


## Prerequisites

Generate VAPID Public and Private Keys for the notification service. This can be done using many tools, one of which is [`GenerateVAPIDKeys`](https://pkg.go.dev/github.com/SherClockHolmes/webpush-go#GenerateVAPIDKeys) from [webpush-go](https://github.com/SherClockHolmes/webpush-go/).

### Compatibility

This service is compatible with the [Web Push Protocol](https://tools.ietf.org/html/rfc8030) and [VAPID](https://tools.ietf.org/html/rfc8292).

For a list of compatible browsers, see [this](https://caniuse.com/push-api) for the Push API and [this](https://caniuse.com/notifications) for the Web Notifications.

## Usage
```go
package main

import (
	"context"
	"log"

	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/webpush"
)

const vapidPublicKey = "..."  // Add a vapidPublicKey
const vapidPrivateKey = "..." // Add a vapidPrivateKey

func main() {
	subscription := webpush.Subscription{
		Endpoint: "https://your-endpoint",
		Keys: {
			Auth:   "...",
			P256dh: "...",
		},
	}

	webpushSvc := webpush.New(vapidPublicKey, vapidPrivateKey)
	webpushSvc.AddReceivers(subscription)

	notifier := notify.NewWithServices(webpushSvc)

	if err := notifier.Send(context.Background(), "TEST", "Message using golang notifier library"); err != nil {
		log.Fatalf("notifier.Send() failed: %s", err.Error())
	}

	log.Println("Notification sent successfully")
}
```
