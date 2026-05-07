/*
Package webpush provides a service for sending messages to viber.

Usage:

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
*/
package webpush
