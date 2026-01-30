# Slack Usage

Ensure that you have already navigated to your GOPATH and installed the following packages:

* `go get -u github.com/nikoksr/notify`
* `go get github.com/slack-go/slack` - You might need this one too

## Steps for Slack App

These are general and very high level instructions

1. Create a new Slack App
2. Give your bot/app the following OAuth permission scopes: `chat:write`, `chat:write.public`
3. Copy your *Bot User OAuth Access Token* for usage below
4. Copy the *Channel ID* of the channel you want to post a message to. You can grab the *Channel ID* by right clicking a channel and selecting `copy link`. Your *Channel ID* will be in that link.
5. Now you should be good to use the code below

## Sample Code

```go
package main

import (
    "github.com/nikoksr/notify"
    "github.com/nikoksr/notify/service/slack"
)

func main() {

    notifier := notify.New()

    // Provide your Slack OAuth Access Token
    slackService := slack.New("OAUTH_TOKEN")

    // Passing a Slack channel id as receiver for our messages.
    // Where to send our messages.
    slackService.AddReceivers("CHANNEL_ID")

    // Tell our notifier to use the Slack service. You can repeat the above process
    // for as many services as you like and just tell the notifier to use them.
    notifier.UseServices(slackService)

    // Send a message
    _ = notifier.Send(
        context.Background(),
        "Hello :wave:\n",
        "I am a bot written in Go!",
    )

    // Code isn't working and need to debug? Use this code below:
    // x := notifier.Send(
    //  context.Background(),
    //  "Hello :wave:\n",
    //  "I am a bot written in Go!",
    // )

    // if x != nil {
    //  fmt.Println(x)
    // }

}
```
