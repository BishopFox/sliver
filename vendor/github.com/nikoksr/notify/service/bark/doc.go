/*
Package bark provides a service for sending messages to bark.

Usage:

	package main

	import (
	    "context"
	    "log"

	    "github.com/nikoksr/notify"
	    "github.com/nikoksr/notify/service/bark"
	)

	func main() {
	    // Create a bark service. `device key` is generated when you install the application. You can use the
	    // `bark.NewWithServers` function to create a service with a custom server.
	    barkService := bark.NewWithServers("your bark device key", bark.DefaultServerURL)

	    // Or use `bark.New` to create a service with the default server.
	    barkService = bark.New("your bark device key")

	    // Tell our notifier to use the bark service.
	    notify.UseServices(barkService)

	    // Send a test message.
	    _ = notify.Send(
	        context.Background(),
	        "Subject/Title",
	        "The actual message - Hello, you awesome gophers! :)",
	    )
	}
*/
package bark
