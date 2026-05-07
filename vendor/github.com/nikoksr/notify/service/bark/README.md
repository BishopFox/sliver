# Bark

[Bark](https://apps.apple.com/us/app/bark-customed-notifications/id1403753865) is an application allows you to push customed notifications to your iPhone

## Usage

```go
// Create a bark service. `device key` is generated when you install the application. You can use the
// `bark.NewWithServers()` function to create a service with a custom server.
barkService := bark.NewWithServers("your bark device key", bark.DefaultServerURL)

// Or use `bark.New()` to create a service with the default server.
barkService = bark.New("your bark device key")

// Add more servers
barkService.AddReceivers("https://your-bark-server.com")

// Tell our notifier to use the bark service.
notify.UseServices(barkService)

// Send a test message.
_ = notify.Send(
    context.Background(),
    "Subject/Title",
    "The actual message - Hello, you awesome gophers! :)",
)
```

