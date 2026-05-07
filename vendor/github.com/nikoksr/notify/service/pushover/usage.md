# Pushover Usage

Ensure that you have already navigated to your GOPATH and installed the following packages:

* `go get -u github.com/nikoksr/notify`

## Steps for Pushover App

These are general and very high level instructions

1. Create a new Pushover App by visiting [here](https://pushover.net/apps/build)
2. Copy your *App token* for usage below
3. Copy the *User ID* or *Group ID* for where you'd like to send messages
4. Now you should be good to use the code below

## Sample Code

```go
package main

import (
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
```
