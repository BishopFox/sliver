# RocketChat Usage

Install notifier using:

* `go get -u github.com/nikoksr/notify`


## Steps to follow

These are general and very high level instructions

1. Login to rocketchat and create a personal token `Profile -> My Account -> Security -> Personal Access Tokens`
2. Copy your *UserID and Token* for usage below
3. Add the user to channels where you want to send message
4. Note down *Channel Names* where you want to post a messages. Channel names are  *Case Sensitive*.
5. Grab the URL for rocketchat server, for this example we are going to user `localhost` and `scheme` is http.
6. Incase endpoint is exposed on a different port then default on localhost
   you can input the serverURL with port i.e `localhost:3000`



## Sample Code

```go
package main

import (
  "github.com/nikoksr/notify"
  "github.com/nikoksr/notify/service/rocketchat"
  "golang.org/x/net/context"
)

func main() {
  // Assuming you already have a rocketchat personal token and userID
  // Provide your server endpoint , protocol , userID  and token
  rocketChatSvc, err := rocketchat.New("localhost", "http", "pLcfBy8zgFDYryQtG", "kNdevpAnDPxh3vwjGELFStFFOI0m0nU_AIN4B0BydtN")
if err != nil {
		panic(err)
	}
  // Add channel names where message is being sent
  // Channel names are case sensitive
  rocketChatSvc.AddReceivers("general", "Notify")

  notifier := notify.New()

  // Tell notifier to use the rocketchat service. You can repeat the above process
  // for as many services as you like and just tell the notifier to use them.
  notifier.UseServices(rocketChatSvc)

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
