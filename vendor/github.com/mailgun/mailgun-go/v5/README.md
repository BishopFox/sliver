# Mailgun with Go

[![GoDoc](https://godoc.org/github.com/mailgun/mailgun-go?status.svg)](https://godoc.org/github.com/mailgun/mailgun-go/v5)
[![Build Status](https://github.com/mailgun/mailgun-go/workflows/CI/badge.svg)](https://github.com/mailgun/mailgun-go/actions/workflows/main.yml?query=branch%3Amaster)

Go library for interacting with the [Mailgun](https://mailgun.com/) [API](https://documentation.mailgun.com/).

## Installation

If you are using [Go Modules](https://go.dev/wiki/Modules) make sure you
include the `/v5` at the end of your import paths
```bash
go get github.com/mailgun/mailgun-go/v5
```

## Usage
### Send a message
```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mailgun/mailgun-go/v5"
)

// Your available domain names can be found here:
// (https://app.mailgun.com/app/domains)
var yourDomain = "your-domain-name" // e.g. mg.yourcompany.com

// You can find the Private API Key in your Account Menu, under "Settings":
// (https://app.mailgun.com/settings/api_security)
var privateAPIKey = "your-private-key"

func main() {
	// Create an instance of the Mailgun Client
	mg := mailgun.NewMailgun(privateAPIKey)

	// When you have an EU domain, you must specify the endpoint:
	// err := mg.SetAPIBase(mailgun.APIBaseEU)

	sender := "sender@example.com"
	subject := "Fancy subject!"
	body := "Hello from Mailgun Go!"
	recipient := "recipient@example.com"

	// The message object allows you to add attachments and Bcc recipients
	message := mailgun.NewMessage(yourDomain, sender, subject, body, recipient)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Send the message with a 10-second timeout
	resp, err := mg.Send(ctx, message)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("ID: %s Resp: %s\n", resp.ID, resp.Message)
}
```

### Get Events
```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mailgun/mailgun-go/v5"
	"github.com/mailgun/mailgun-go/v5/events"
)

func main() {
	// You can find the Private API Key in your Account Menu, under "Settings":
	// (https://app.mailgun.com/settings/api_security)
	mg := mailgun.NewMailgun("your-private-key")

	it := mg.ListEvents("your-domain.com", &mailgun.ListEventOptions{Limit: 100})

	var page []events.Event

	// The entire operation should not take longer than 30 seconds
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	// For each page of 100 events
	for it.Next(ctx, &page) {
		for _, e := range page {
			// You can access some fields via the interface
			fmt.Printf("Event: '%s' TimeStamp: '%s'\n", e.GetName(), e.GetTimestamp())

			// and you can act upon each event by type
			switch event := e.(type) {
			case *events.Accepted:
				fmt.Printf("Accepted: auth: %t\n", event.Flags.IsAuthenticated)
			case *events.Delivered:
				fmt.Printf("Delivered transport: %s\n", event.Envelope.Transport)
			case *events.Failed:
				fmt.Printf("Failed reason: %s\n", event.Reason)
			case *events.Clicked:
				fmt.Printf("Clicked GeoLocation: %s\n", event.GeoLocation.Country)
			case *events.Opened:
				fmt.Printf("Opened GeoLocation: %s\n", event.GeoLocation.Country)
			case *events.Rejected:
				fmt.Printf("Rejected reason: %s\n", event.Reject.Reason)
			case *events.Stored:
				fmt.Printf("Stored URL: %s\n", event.Storage.URL)
			case *events.Unsubscribed:
				fmt.Printf("Unsubscribed client OS: %s\n", event.ClientInfo.ClientOS)
			}
		}
	}
}
```

### Event Polling
The mailgun library has built-in support for polling the events api
```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/mailgun/mailgun-go/v5"
	"github.com/mailgun/mailgun-go/v5/events"
)

func main() {
	// You can find the Private API Key in your Account Menu, under "Settings":
	// (https://app.mailgun.com/settings/api_security)
	mg := mailgun.NewMailgun("your-private-key")

	begin := time.Now().Add(time.Second * -3)

	// Very short poll interval
	it := mg.PollEvents("your-domain.com", &mailgun.ListEventOptions{
		// Only events with a timestamp after this date/time will be returned
		Begin: begin,
		// How often we poll the api for new events
		PollInterval: time.Second * 30,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Poll until our email event arrives
	var page []events.Event
	for it.Poll(ctx, &page) {
		for _, e := range page {
			log.Printf("Got an event: %q (%q)", e.GetName(), e.GetID())
			// Do something with event
		}
	}
}
```

### Email Validations
```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mailgun/mailgun-go/v5"
)

// Your plan should include email validations.
// Use your Mailgun API key. You can find the Mailgun API keys in your Account Menu, under "Settings":
// (https://app.mailgun.com/settings/api_security)
var apiKey = "your-api-key"

func main() {
	// Create an instance of the Validator
	mg := mailgun.NewMailgun("your-private-key")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	email, err := mg.ValidateEmail(ctx, "recipient@example.com", false)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Risk: %t\n", email.Risk)
}
```

### Webhook Handling
```go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/mailgun/mailgun-go/v5"
	"github.com/mailgun/mailgun-go/v5/events"
	"github.com/mailgun/mailgun-go/v5/mtypes"
)

func main() {
	// You can find the Private API Key in your Account Menu, under "Settings":
	// (https://app.mailgun.com/settings/api_security)
	mg := mailgun.NewMailgun("private-api-key")
	mg.SetWebhookSigningKey("webhook-signing-key")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var payload mtypes.WebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			fmt.Printf("decode JSON error: %s", err)
			w.WriteHeader(http.StatusNotAcceptable)
			return
		}

		verified, err := mg.VerifyWebhookSignature(payload.Signature)
		if err != nil {
			fmt.Printf("verify error: %s\n", err)
			w.WriteHeader(http.StatusNotAcceptable)
			return
		}

		if !verified {
			w.WriteHeader(http.StatusNotAcceptable)
			fmt.Printf("failed verification %+v\n", payload.Signature)
			return
		}

		fmt.Printf("Verified Signature\n")

		// Parse the event provided by the webhook payload
		e, err := events.ParseEvent(payload.EventData)
		if err != nil {
			fmt.Printf("parse event error: %s\n", err)
			return
		}

		switch event := e.(type) {
		case *events.Accepted:
			fmt.Printf("Accepted: auth: %t\n", event.Flags.IsAuthenticated)
		case *events.Delivered:
			fmt.Printf("Delivered transport: %s\n", event.Envelope.Transport)
		}
	})

	fmt.Println("Serve on :9090...")
	if err := http.ListenAndServe(":9090", nil); err != nil {
		fmt.Printf("serve error: %s\n", err)
		os.Exit(1)
	}
}
```

### Sending HTML templates

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mailgun/mailgun-go/v5"
)

// Your available domain names can be found here:
// (https://app.mailgun.com/app/domains)
var yourDomain = "your-domain-name" // e.g. mg.yourcompany.com

// You can find the Private API Key in your Account Menu, under "Settings":
// (https://app.mailgun.com/settings/api_security)
var privateAPIKey = "your-private-key"

func main() {
	// Create an instance of the Mailgun Client
	mg := mailgun.NewMailgun(privateAPIKey)

	sender := "sender@example.com"
	subject := "HTML email!"
	recipient := "recipient@example.com"

	message := mailgun.NewMessage(yourDomain, sender, subject, "", recipient)
	body := `
<html>
<body>
	<h1>Sending HTML emails with Mailgun</h1>
	<p style="color:blue; font-size:30px;">Hello world</p>
	<p style="font-size:30px;">More examples can be found <a href="https://documentation.mailgun.com/en/latest/api-sending.html#examples">here</a></p>
</body>
</html>
`

	message.SetHTML(body)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Send the message with a 10-second timeout
	resp, err := mg.Send(ctx, message)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("ID: %s Resp: %s\n", resp.ID, resp.Message)
}
```

### Using Templates

Templates enable you to create message templates on your Mailgun account and then populate the data variables at send-time. This allows you to have your layout and design managed on the server and handle the data on the client. The template variables are added as a JSON stringified `X-Mailgun-Variables` header. For example, if you have a template to send a password reset link, you could do the following:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mailgun/mailgun-go/v5"
)

// Your available domain names can be found here:
// (https://app.mailgun.com/app/domains)
var yourDomain = "your-domain-name" // e.g. mg.yourcompany.com

// You can find the Private API Key in your Account Menu, under "Settings":
// (https://app.mailgun.com/settings/api_security)
var privateAPIKey = "your-private-key"

func main() {
	// Create an instance of the Mailgun Client
	mg := mailgun.NewMailgun(privateAPIKey)

	sender := "sender@example.com"
	subject := "Fancy subject!"
	body := ""
	recipient := "recipient@example.com"

	// The message object allows you to add attachments and Bcc recipients
	message := mailgun.NewMessage(yourDomain, sender, subject, body, recipient)
	message.SetTemplate("passwordReset")
	err := message.AddTemplateVariable("passwordResetLink", "some link to your site unique to your user")
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Send the message with a 10-second timeout
	resp, err := mg.Send(ctx, message)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("ID: %s Resp: %s\n", resp.ID, resp.Message)
}
```

The official mailgun documentation includes examples using this library. Go
[here](https://documentation.mailgun.com/docs/mailgun/api-reference/)
and click on the "Go" button at the top of the page.

## EU Region
European customers will need to change the default API Base to access your domains

```go
mg := mailgun.NewMailgun("private-api-key")
mg.SetAPIBase(mailgun.APIBaseEU)
```

## Testing

*WARNING* - running the tests will cost you money!

To run the tests various environment variables must be set. These are:

* `MG_DOMAIN` is the domain name - this is a value registered in the Mailgun admin interface.
* `MG_API_KEY` is the Private API key - you can get this value from the Mailgun [security page](https://app.mailgun.com/settings/api_security)
* `MG_EMAIL_TO` is the email address used in various sending tests.

and finally

* `MG_SPEND_MONEY` if this value is set the part of the test that use the API to actually send email will be run - be aware *this will count on your quota* and *this _will_ cost you money*.

The code is released under a 3-clause BSD license. See the LICENSE file for more information.
