/*
Package mattermost provides message notification integration for mattermost.com.

Usage:

	package main

	import (
		"os"

		"github.com/nikoksr/notify"
		"github.com/nikoksr/notify/service/mattermost"
	)

	func main() {

		// Init notifier
		notifier := notify.New()
		ctx := context.Background()

		// Provide your Mattermost server url
		mattermostService := mattermost.New("https://myserver.cloud.mattermost.com")

		// Provide username as loginID and password to login into above server.
		// NOTE: This generates auth token which will get expired, invoking this method again
		// after expiry will generate new token and uses for further requests.
		err := mattermostService.LoginWithCredentials(ctx, "someone@gmail.com", "somepassword")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Passing a Mattermost channel/chat id as receiver for our messages.
		// Where to send our messages.
		mattermostService.AddReceivers("CHANNEL_ID")

		// Tell our notifier to use the Mattermost service. You can repeat the above process
		// for as many services as you like and just tell the notifier to use them.
		notifier.UseServices(mattermostService)

		// Add presend and postsend hooks that you need to execute before every requests and after
		// every response respectively. Multiple presend and postsend are executed in the order defined here.
		// refer service/http for the more info.
		// PreSend hook
		mattermostService.PreSend(func(req *stdhttp.Request) error {
			log.Printf("Sending message to %s server", req.URL)
			return nil
		})
		// PostSend hook
		mattermostService.PostSend(func(req *stdhttp.Request, resp *stdhttp.Response) error {
			log.Printf("Message sent to %s server with status %d", req.URL, resp.StatusCode)
			return nil
		})

		// Send a message
		err = notifier.Send(
			ctx,
			"Hello from notify :wave:",
			"Message written in Go!",
		)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
*/
package mattermost
