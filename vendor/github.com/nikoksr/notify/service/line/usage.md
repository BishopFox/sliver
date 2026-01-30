# Line Usage

Install notifier using:

* `go get -u github.com/nikoksr/notify`


## Sample Code

```go
package main

import (
  "github.com/nikoksr/notify"
  "github.com/nikoksr/notify/service/line"
  "golang.org/x/net/context"
)

func main() {
  // Assuming you already have a line messaging API credential
  // Provide your channel secret and access token
  lineService, _ := line.New("channelSecret", "channelAccessToken")

  // Add id from various receivers
  // You can try to use your own line id for testing
  lineService.AddReceivers("userID1", "groupID1")

  notifier := notify.New()

  // Tell our notifier to use the line service. You can repeat the above process
  // for as many services as you like and just tell the notifier to use them.
  notifier.UseServices(lineService)

  // Send a message
  err := notifier.Send(
    context.Background(),
    "Welcome",
    "I am a bot written in Go!",
  )

  if err != nil {
    panic(err)
  }
}
```

## Sample Code for Line Notify

```go
package main

import (
  "github.com/nikoksr/notify"
  "github.com/nikoksr/notify/service/line"
  "golang.org/x/net/context"
)

func main() {
  // Assuming you already have a line messaging API credential
  // Provide your channel secret and access token
  lineNotifyService := line.NewNotify()

  // Add id from various receivers
  // You can try to use your own line id for testing
  lineNotifyService.AddReceivers("receiverToken")

  notifier := notify.New()

  // Tell our notifier to use the line service. You can repeat the above process
  // for as many services as you like and just tell the notifier to use them.
  notifier.UseServices(lineNotifyService)

  // Send a message
  err := notifier.Send(
    context.Background(),
    "Welcome",
    "I am a bot written in Go!",
  )

  if err != nil {
    panic(err)
  }
}
```
