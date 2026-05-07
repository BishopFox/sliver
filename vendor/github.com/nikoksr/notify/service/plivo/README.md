# Plivo

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat)](https://pkg.go.dev/github.com/nikoksr/notify/service/plivo)

## Prerequisites

You will need to have a [Plivo](https://www.plivo.com/) account and the
following things:

1. `Auth ID` and `Auth Token` from Plivo [console](https://console.plivo.com/dashboard/).
1. An active rented Plivo [phone number](https://console.plivo.com/active-phone-numbers/).

## Usage

```go
package main

import (
  "context"
  "log"

  "github.com/nikoksr/notify"
  "github.com/nikoksr/notify/service/plivo"
)

func main() {
  plivoSvc, err := plivo.New(
    &plivo.ClientOptions{
      AuthID:    "<Your-Plivo-Auth-Id>",
      AuthToken: "<Your-Plivo-Auth-Token>",
    }, &plivo.MessageOptions{
      Source: "<Your-Plivo-Source-Number>",
    })
  if err != nil {
    log.Fatalf("plivo.New() failed: %s", err.Error())
  }

  plivoSvc.AddReceivers("Destination1")

  notifier := notify.New()
  notifier.UseServices(plivoSvc)

  err = notifier.Send(context.Background(), "subject", "message")
  if err != nil {
    log.Fatalf("notifier.Send() failed: %s", err.Error())
  }

  log.Printf("notification sent")
}
```
