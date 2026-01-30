# Pushbullet Usage

Ensure that you have already navigated to your GOPATH and installed the following packages:

* `go get -u github.com/nikoksr/notify`
* `go get github.com/cschomburg/go-pushbullet` - You might need this one too

## Steps for Pushbullet App

These are general and very high level instructions

1. Create a Pushbullet account
2. Download Pushbullet on any devices which are to receive notifications
3. Copy your *Access Token* for usage below form https://www.pushbullet.com/#settings
4. Copy the *Device Nickname* of the device you want to post a message to. See https://www.pushbullet.com/#settings/devices

## Sample Code

```go
package main

import (
    "github.com/nikoksr/notify"
    "github.com/nikoksr/notify/service/pushbullet"
)

func main() {

    notifier := notify.New()

    // Provide your Access Token
    service := pushbullet.New("AccessToken")

    // Passing a device nickname as receiver for our messages.
    service.AddReceivers("DeviceNickname")

    // Tell our notifier to use the Pushbullet service. You can repeat the above process
    // for as many services as you like and just tell the notifier to use them.
    notifier.UseServices(service)

    // Send a message
    err = notifier.Send(
        "Hello\n",
        "I am a bot written in Go!",
    )

    if err != nil {
        panic(err)
    }
}
```


# Steps for Pushbullet SMS

1. Follow the above instructions, ensuring that Pushbullet is installed on an Android device.
2. Enable 'SMS Sync' is turned on (under the SMS tab on the Android app)

## Sample Code

```go
package main

import (
    "github.com/nikoksr/notify"
    "github.com/nikoksr/notify/service/pushbullet"
)

func main() {

    notifier := notify.New()

    // Provide your Access Token and Pushbullet nickname for your mobile device
    service, err := pushbullet.NewSMS("AccessToken", "PhoneNickname")
    if err != nil {
        panic(err)
    }

    // Passing a phone number as receiver for our messages.
    service.AddReceivers("PhoneNumber")

    // Tell our notifier to use the Pushbullet service. You can repeat the above process
    // for as many services as you like and just tell the notifier to use them.
    notifier.UseServices(service)

    // Send a message
    err = notifier.Send(
        "Hello\n",
        "I am a bot written in Go!",
    )

    if err != nil {
        panic(err)
    }
}
```
