/*
Package reddit implements a Reddit notifier, allowing messages to be sent to multiple recipients

Usage:

	package main

	import (
		"context"

		"github.com/nikoksr/notify"
		"github.com/nikoksr/notify/service/reddit"
	)

	func main() {

		notifier := notify.New()

		// Provide your Reddit app credentials and username/password
		redditService := reddit.New("clientID", "clientSecret", "username", "password")

		// Pass the usernames for where to send the messages
		pushoverService.AddReceivers("User1", "User2")

		// Tell our notifier to use the Reddit service. You can repeat the above process
		// for as many services as you like and just tell the notifier to use them.
		notifier.UseServices(redditService)

		// Send a message
		_ = notifier.Send(
			context.Background(),
			"Hello!",
			"I am a bot written in Go!",
		)
	}
*/
package reddit
