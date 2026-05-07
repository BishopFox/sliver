/*
Package fcm provides message notification integration for Firebase Cloud Messaging (FCM).

Usage:

	package main

	import (
		"context"
		"log"

		"github.com/nikoksr/notify"
		"github.com/nikoksr/notify/service/fcm"
	)

	func main() {
		fcmSvc, err := fcm.New("server_api_key")
		if err != nil {
			log.Fatalf("fcm.New() failed: %s", err.Error())
		}

		fcmSvc.AddReceivers("deviceToken1")

		notifier := notify.New()
		notifier.UseServices(fcmSvc)

		// Use context.Background() if you want to send a simple notification message.
		ctx := context.Background()

		// Optionally, you can include additional data in the message payload by adding the corresponding value to the
		// context.
		ctxWithData := context.WithValue(ctx, fcm.DataKey, map[string]interface{}{
			"some-key":  "some-value",
			"other-key": "other-value",
		})

		// Optionally, you can specify a total of retry attempts per each message by adding the corresponding value to
		// the context.
		ctxWithDataAndRetries := context.WithValue(ctxWithData, fcm.RetriesKey, 3)

		err = notifier.Send(ctxWithDataAndRetries, "subject", "message")
		if err != nil {
			log.Fatalf("notifier.Send() failed: %s", err.Error())
		}

		log.Println("notification sent")
	}
*/
package fcm
