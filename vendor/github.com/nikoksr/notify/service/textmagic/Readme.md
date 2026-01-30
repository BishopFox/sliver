# TextMagic

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat)](https://docs.textmagic.com/#section/Go/Usage-Example)

## Prerequisites

You will need to have a [TextMagic](https://www.textmagic.com/) account and
`UserName` and `API KEY` from TextMagic.[(api-keys)](https://my.textmagic.com/online/api/rest-api/keys)


```go
package main

import (
  "context"
  "log"

  "github.com/nikoksr/notify"
  "github.com/nikoksr/notify/service/textmagic"
)

func main() {

  textMagicService := textmagic.New("YOUR_USER_NAME", "YOUR_API_KEY")

  textMagicService.AddReceivers("Destination1-Phone-Number")

  notifier := notify.New()
  notifier.UseServices(textMagicService)

  err := notifier.Send(context.Background(), "subject", "message")
  if err != nil {
    log.Fatalf("notifier.Send() failed: %s", err.Error())
  }

  log.Printf("notification sent")
}
```
