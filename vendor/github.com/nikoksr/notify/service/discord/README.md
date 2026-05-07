# Discord

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat)](https://pkg.go.dev/github.com/nikoksr/notify/service/discord)

## Prerequisites

To use the Discord notification service, you will need:

- A Discord Bot Token or OAuth2 Token
- Channel IDs where you want to send messages

You can create a Discord bot and get your token from the [Discord Developer Portal](https://discord.com/developers/applications).

## Usage

### Basic Usage

```go
package main

import (
	"context"
	"log"

	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/discord"
)

func main() {
	discordSvc := discord.New()

	// Authenticate with your bot token
	if err := discordSvc.AuthenticateWithBotToken("your-bot-token"); err != nil {
		log.Fatalf("failed to authenticate: %s", err.Error())
	}

	// Add channel IDs to send messages to
	discordSvc.AddReceivers("channel-id-1", "channel-id-2")

	notifier := notify.New()
	notifier.UseServices(discordSvc)

	if err := notifier.Send(context.Background(), "Subject", "Message body"); err != nil {
		log.Fatalf("failed to send notification: %s", err.Error())
	}

	log.Println("notification sent")
}
```

### OAuth2 Authentication

You can also authenticate using an OAuth2 token:

```go
discordSvc := discord.New()

if err := discordSvc.AuthenticateWithOAuth2Token("your-oauth2-token"); err != nil {
	log.Fatalf("failed to authenticate: %s", err.Error())
}
```

### Advanced Configuration

For advanced use cases such as proxy configuration or custom HTTP client settings, you can use `SetClient` with `DefaultSession`:

```go
package main

import (
	"context"
	"log"
	"net/http"
	"net/url"

	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/discord"
)

func main() {
	// Create a custom HTTP client with proxy
	proxyURL, _ := url.Parse("http://proxy.example.com:8080")
	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}

	// Get a Discord session with default configuration
	session := discord.DefaultSession()
	session.Client = httpClient

	// Create Discord service and set the custom session
	discordSvc := discord.New()
	discordSvc.SetClient(session)

	// Authenticate - this will preserve the custom HTTP client
	if err := discordSvc.AuthenticateWithBotToken("your-bot-token"); err != nil {
		log.Fatalf("failed to authenticate: %s", err.Error())
	}

	discordSvc.AddReceivers("channel-id")

	notifier := notify.New()
	notifier.UseServices(discordSvc)

	if err := notifier.Send(context.Background(), "Subject", "Message via proxy"); err != nil {
		log.Fatalf("failed to send notification: %s", err.Error())
	}

	log.Println("notification sent")
}
```

The `SetClient` method allows full control over the Discord session configuration, including:

- Custom HTTP client (for proxies, custom timeouts, etc.)
- Shard configuration
- Connection settings
- Rate limiting behavior

Note that `DefaultSession()` allows you to configure the Discord session without importing the `discordgo` package directly.
