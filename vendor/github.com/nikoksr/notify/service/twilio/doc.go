/*
Package twilio provides message notification integration for Twilio (Message Service).

Usage:

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
*/
package twilio
