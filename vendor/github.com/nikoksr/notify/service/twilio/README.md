# Twilio (Message Service)

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat)](https://pkg.go.dev/github.com/nikoksr/notify/service/twilio)

## Prerequisites

Navigate to Twilio [console](https://console.twilio.com/), create a new account or login with an existing one.
You will find the `Account SID` and the `Auth Token` under the `Account Info` tab. You may also request a Twilio phone number, if required.

To test the integration with a phone number you can just use the sample code below.

## Usage

```go
package main

import (
	"context"
	"log"

	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/twilio"
)

func main() {
	twilioSvc, err := twilio.New("account_sid", "auth_token", "your_phone_number")
	if err != nil {
		log.Fatalf("twilio.New() failed: %s", err.Error())
	}

	twilioSvc.AddReceivers("recipient_phone_number")

	notifier := notify.New()
	notifier.UseServices(twilioSvc)

	err = notifier.Send(context.Background(), "subject", "message")
	if err != nil {
		log.Fatalf("notifier.Send() failed: %s", err.Error())
	}

	log.Println("notification sent")
}
```
