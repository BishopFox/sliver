# Go/Golang package for Viber messaging and chatbot [![GoDoc](https://godoc.org/github.com/mileusna/viber?status.svg)](https://godoc.org/github.com/mileusna/viber)

With this package you can use [Viber REST API](https://developers.viber.com/docs/api/rest-bot-api/) to send and receive messages from Viber platform.

All structs used in package represents the structs identical to [Viber REST API](https://developers.viber.com/docs/api/rest-bot-api/). To fully understand Viber messaging platform and this package as well, you should read [Viber REST API](https://developers.viber.com/docs/api/rest-bot-api/).

Before using this package you will need [Viber public account](https://support.viber.com/customer/en/portal/articles/2733413-create-a-public-account) and your App key which you can find in Edit section of your Viber public account.

  * [Instalation](#instalation)
  * [Hello World example](#helloworld)
  * [Set Webhook](#webhook)
  * [Send message to the user](#messaging)
  * [Carousel messages](#carousel)
  * [Send message to the Public Account](#pamessaging)
  * [Public Account info](#accountinfo)
  * [User details](#userdetails)
  * [Receiving messages and callbacks](#callbacks)

## Installation <a id="installation"></a>
```
go get github.com/mileusna/viber
```

## Hello World example<a id="helloworld"></a>

```go
package main 

import (
    "fmt"
    "log"

    "github.com/mileusna/viber"
)

func main() {
    v := viber.New("YOUR-APP-KEY-FROM-VIBER", "MyPage", "https://mysite.com/img/avatar.jpg")

    // you really need this only once, remove after you set the webhook
    v.SetWebhook("https://mysite.com/viber/webhook/", nil)

    userID := "Goxxuipn9xKKRqkFOOwKnw==" // fake user ID, use the real one

    // send text message
    token, err := v.SendTextMessage(userID, "Hello, World!")
    if err != nil {
        log.Println("Viber error:", err)
        return
    }
    fmt.Println("Message sent, message token:", token)
}
```

At the begining you neew to declare your viber struct with you app key. Sender is default sender which will be used as default for sending all messages to the users and public account. But, for each message you can specify different sender if you like.

```go
v := viber.New("YOUR-APP-KEY-FROM-VIBER", "MyPage", "https://mysite.com/img/avatar.jpg")
```

Read more about _SetWebhook_ in following [chapter](#webhook).

After that, you can [send different kind of messages to the user](#messaging).

[Here](#callbacks) you can read how to handle message receiving.

## Webhook <a id="webhook"></a>

To be able to receive messages and notifications from Viber you have to specify your webhook. Webhook is the URL where Viber will send you all messages and notification. You only have to do this once in a lifetime. URL of webhook have to be online in moment you call _SetWebhook_ since Viber will send http request to webhook URL expecting HTTP status code 200. For more info visit [Viber documentation on Webhooks](https://developers.viber.com/docs/api/rest-bot-api/#webhooks).

```go
// if eventTypes is nil, all callbacks will be set to webhook
// if eventTypes is empty []string mandatory callbacks will be set
// Mandatory callbacks: "message", "subscribed", "unsubscribed"
// All possible callbacks: "message", "subscribed",  "unsubscribed", "delivered", "seen", "failed", "conversation_started"
v.SetWebhook("https://mysite.com/viber/webhook/", nil)
```

## Messaging <a id="messaging"></a>

You can send message in different ways. The easiest way is to use shortcut functions like _SendTextMessage_ or _SendURLMessage_:
```go
v.SendTextMessage(userID, "Hello, World!")

v.SendURLMessage(userID, "Visit my site", "http://mysite.com/")

v.SendPictureMessage(userID, "Take a look at this photo", "http://mysite.com/photo.jpg")
```

This function will send messages to userID (you will get userID when you [receive message from user](#callbacks)) using default sender specified in declaration.

You can create individual message, change some settings and then send it using _SendMessage_

```go
// create message, change the sender for this message and then send id
m := v.NewTextMessage("Hello, world!")
m.Sender = viber.Sender{
    Name:   "SomeOtherName",
    Avatar: "https://mysite.com/img/other_avatar.jpg",
}
v.SendMessage(userID, m)
```

## Carousel messages <a id="carousel"></a>

Documentation coming soon.

## Send Messages to Public Account <a id="pamessaging"></a>

In previous examples you send messages directly to the user subscribed to your public account. If you want to send message to the Public Account which will be seen by all PA followers, use te _SendPublicMessage_ function.
```go
// adminID is the ID of PA administrator
m := v.NewTextMessage("Hello to everyone")
v.SendPublicMessage(adminID, m)

imgMsg := v.NewImageMessage("Photo for everyone", "http://mysite.com/photo.jpg")
v.SendPublicMessage(adminID, imgMsg)
```

## Account info <a id="accountinfo"></a>

In previous example you have to use ID of administrator of Public account. To obtain Public Acount info and the list of administrators, use the _AccountInfo_ function.
```go
a, err := v.AccountInfo()
if err != nil {
    log.Println("AccountInfo Viber error:", err)
    return
}

// print all admministrators
for _, m := range a.Members {
    fmt.Println(m.ID, m.Name, m.Role)
}
```

# User details <a id="userdetails"></a>
When receiving message from user, or when user starts the conversation, you will receive some bacis user info. To obtain full user details use the _UserDetails_ function.

```go
u, err := v.UserDetails(userID)
if err != nil {
    log.Println("User details viber error:", err)
    return
}

fmt.Println("Details:", u.Name, u.Avatar, u.Country, u.Language, u.DeviceType, u.PrimaryDeviceOs)
```

## Receiving messages / Callbacks <a id="callbacks"></a>
To receive messages and other callbacks, you have to run you viber app on webhook URL you specified using _SetWebhook_. You can easily manage all Viber callbacks (_Message, Subscribed, Unsubscribed, Delivered, Seen, Failed_) using this package by specifying your own functions which will be called on event.

```go
package main

import (
    "fmt"
    "log"
    "net/http"
    "time"

    "github.com/mileusna/viber"
)

func main() {
    v := &viber.Viber{
        AppKey: "YOUR-APP-KEY-FROM-VIBER",
        Sender: viber.Sender{
            Name:   "MyPage",
            Avatar: "https://mysite.com/img/avatar.jpg",
        },
        Message:   myMsgReceivedFunc,  // your function for handling messages
        Delivered: myDeliveredFunc,    // your function for delivery report
    }
    v.Seen = mySeenFunc   // or assign events after declaration
    
    // this have to be your webhook, pass it your viber app as http handler
    http.Handle("/viber/webhook/", v)
    http.ListenAndServe(":80", nil)    
}

// myMsgReceivedFunc will be called everytime when user send us a message
func myMsgReceivedFunc(v *viber.Viber, u viber.User, m viber.Message, token uint64, t time.Time) {
    switch m.(type) {

    case *viber.TextMessage:
        v.SendTextMessage(u.ID, "Thank you for your message")
        txt := m.(*viber.TextMessage).Text
        v.SendTextMessage(u.ID, "This is the text you have sent to me "+txt)

    case *viber.URLMessage:
        url := m.(*viber.URLMessage).Media
        v.SendTextMessage(u.ID, "You have sent me an interesting link "+url)

    case *viber.PictureMessage:
        v.SendTextMessage(u.ID, "Nice pic!")

    }
}

func myDeliveredFunc(v *viber.Viber, userID string, token uint64, t time.Time) {
    log.Println("Message ID", token, "delivered to user ID", userID)
}

func mySeenFunc(v *viber.Viber, userID string, token uint64, t time.Time) {
    log.Println("Message ID", token, "seen by user ID", userID)
}


// All events that you can assign your function, declarations must match
// ConversationStarted func(v *Viber, u User, conversationType, context string, subscribed bool, token uint64, t time.Time) Message
// Message             func(v *Viber, u User, m Message, token uint64, t time.Time)
// Subscribed          func(v *Viber, u User, token uint64, t time.Time)
// Unsubscribed        func(v *Viber, userID string, token uint64, t time.Time)
// Delivered           func(v *Viber, userID string, token uint64, t time.Time)
// Seen                func(v *Viber, userID string, token uint64, t time.Time)
// Failed              func(v *Viber, userID string, token uint64, descr string, t time.Time) 
```
