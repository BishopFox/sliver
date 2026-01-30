/*
Package pushover implements a Pushover notifier, allowing messages to be sent to multiple recipients and supports
both users and groups.

Usage:

	package main

	import (
		"context"

		"github.com/nikoksr/notify"
		"github.com/nikoksr/notify/service/pushover"
	)

	func main() {

		notifier := notify.New()

		// Provide your Pushover App token
		pushoverService := pushover.New("APP_TOKEN")

		// Pass user and/or group IDs for where to send the messages
		pushoverService.AddReceivers("USER_ID", "GROUP_ID")

		// Tell our notifier to use the Pushover service. You can repeat the above process
		// for as many services as you like and just tell the notifier to use them.
		notifier.UseServices(pushoverService)

		// Send a message
		_ = notifier.Send(
			context.Background(),
			"Hello!",
			"I am a bot written in Go!",
		)
	}
*/
package pushover
