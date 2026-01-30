pushover
=========

[![GoDoc](https://godoc.org/github.com/gregdel/pushover?status.svg)](http://godoc.org/github.com/gregdel/pushover)
[![Build Status](https://travis-ci.org/gregdel/pushover.svg?branch=master)](https://travis-ci.org/gregdel/pushover)
[![Coverage Status](https://coveralls.io/repos/gregdel/pushover/badge.svg?branch=master&service=github)](https://coveralls.io/github/gregdel/pushover?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/gregdel/pushover)](https://goreportcard.com/report/github.com/gregdel/pushover)

pushover is a wrapper around the Superblock's Pushover API written in go.
Based on their [documentation](https://pushover.net/api). It's a convenient way to send notifications from a go program with only a few lines of code.

## Messages

### Send a simple message

Here is a simple example for sending a notification to a recipient. A recipient can be a user or a group. There is no real difference, they both use a notification token.

```go
package main

import (
    "log"

    "github.com/gregdel/pushover"
)

func main() {
    // Create a new pushover app with a token
    app := pushover.New("uQiRzpo4DXghDmr9QzzfQu27cmVRsG")

    // Create a new recipient
    recipient := pushover.NewRecipient("gznej3rKEVAvPUxu9vvNnqpmZpokzF")

    // Create the message to send
    message := pushover.NewMessage("Hello !")

    // Send the message to the recipient
    response, err := app.SendMessage(message, recipient)
    if err != nil {
        log.Panic(err)
    }

    // Print the response if you want
    log.Println(response)
}
```

### Send a message with a title

There is a simple way to create a message with a title. Instead of using pushover.NewMessage you can use pushover.NewMessageWithTitle.

```go
message := pushover.NewMessageWithTitle("My awesome message", "My title")
```

### Send a fancy message

If you want a more detailed message you can still do it.

```go
message := &pushover.Message{
    Message:     "My awesome message",
    Title:       "My title",
    Priority:    pushover.PriorityEmergency,
    URL:         "http://google.com",
    URLTitle:    "Google",
    Timestamp:   time.Now().Unix(),
    Retry:       60 * time.Second,
    Expire:      time.Hour,
    DeviceName:  "SuperDevice",
    CallbackURL: "http://yourapp.com/callback",
    Sound:       pushover.SoundCosmic,
}
```

### Send a message with an attachment

You can send an image attachment along with the message.

```go
file, err := os.Open("/some/image.png")
if err != nil {
  panic(err)
}
defer file.Close()

message := pushover.NewMessage("Hello !")
if err := message.AddAttachment(file); err != nil {
  panic(err)
}
```

## Callbacks and receipts

If you're using an emergency notification you'll have to specify a retry period and an expiration delay. You can get the receipt details using the token in the message response.


```go
...
response, err := app.SendMessage(message, recipient)
if err != nil {
    log.Panic(err)
}

receiptDetails, err := app.GetReceiptDetails(response.Receipt)
if err != nil {
    log.Panic(err)
}

fmt.Println("Acknowledged status :", receiptDetails.Acknowledged)
```

You can also cancel an emergency notification before the expiration time.

```go
response, err := app.CancelEmergencyNotification(response.Receipt)
if err != nil {
    log.Panic(err)
}
```

## User verification

If you want to validate that the recipient token is valid.

```go
...
recipientDetails, err := app.GetRecipientDetails(recipient)
if err != nil {
    log.Panic(err)
}

fmt.Println(recipientDetails)
```
