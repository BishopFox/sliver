# Matrix

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat)](https://pkg.go.dev/github.com/nikoksr/notify/service/matrix)

## Prerequisites

You will need to following information to be able to send messages to Matrix.

- Home server url
- User ID
- AccessToken
- Room ID

## Usage

In the current implementation, using this service requires 2 steps:

1. Provide the necessary credentials explicitly
2. Use the Send message to send a message to the specified room.

```go
package main

import (
  "context"
  "log"

  "github.com/nikoksr/notify"
  "github.com/nikoksr/notify/service/matrix"
)

func main() {
  matrixSvc, err := matrix.New("user-id", "room-id", "home-server", "access-token")
  if err != nil {
    log.Fatalf("matrix.New() failed: %s", err.Error())
  }

  notifier := notify.New()
  notifier.UseServices(matrixSvc)

  err = notifier.Send(context.Background(), "", "message")
  if err != nil {
    log.Fatalf("notifier.Send() failed: %s", err.Error())
  }

  log.Println("notification sent")
}
```
