[comment]: <> (HEAD)
# TextMagic Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/textmagic/textmagic-rest-go-v2/v3.svg)](https://pkg.go.dev/github.com/textmagic/textmagic-rest-go-v2/v3)
[![Go Report Card](https://goreportcard.com/badge/github.com/textmagic/textmagic-rest-go-v2/v3)](https://goreportcard.com/report/github.com/textmagic/textmagic-rest-go-v2/v3)

This library provides you with an easy way of sending SMS and receiving replies by integrating the TextMagic SMS Gateway into your Go application.

## What Is TextMagic?
TextMagic's application programming interface (API) provides the communication link between your application and TextMagic's SMS Gateway, allowing you to send and receive text messages and to check the delivery status of text messages you've already sent.

[comment]: <> (/HEAD)

## Requirements

- **Go 1.23** or higher
- Go modules support

## Installation

### Via go get (Recommended)

```bash
go get -u github.com/textmagic/textmagic-rest-go-v2/v3@latest
```

### Via go.mod

Add to your `go.mod`:
```go
require github.com/textmagic/textmagic-rest-go-v2/v3 v3.0.43885
```

Then run:
```bash
go mod download
```

### Package Registry

This package is available on [pkg.go.dev](https://pkg.go.dev/github.com/textmagic/textmagic-rest-go-v2/v3) - the official Go package registry.

## Usage Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    tm "github.com/textmagic/textmagic-rest-go-v2/v3"
)

func main() {
    // Create configuration
    cfg := tm.NewConfiguration()
    cfg.Servers = tm.ServerConfigurations{
        {
            URL: "https://rest.textmagic.com",
            Description: "TextMagic REST API",
        },
    }

    client := tm.NewAPIClient(cfg)

    // Set up authentication
    // Get your credentials from https://my.textmagic.com/online/api/rest-api/keys
    auth := context.WithValue(context.Background(), tm.ContextBasicAuth, tm.BasicAuth{
        UserName: "YOUR_USERNAME",
        Password: "YOUR_API_KEY",
    })

    // Simple ping request example
    pingResponse, _, err := client.TextMagicAPI.Ping(auth).Execute()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Ping:", pingResponse.Ping)

    // Send a new message request example
    sendMessageInput := tm.SendMessageInputObject{
        Text:   "I love TextMagic!",
        Phones: "+19998887766",
    }

    sendMessageResponse, _, err := client.TextMagicAPI.SendMessage(auth).SendMessageInputObject(sendMessageInput).Execute()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Message ID:", sendMessageResponse.Id)

    // Get all outgoing messages request example
    page := int32(1)
    limit := int32(250)

    getAllOutboundMessageResponse, _, err := client.TextMagicAPI.GetAllOutboundMessages(auth).
        Page(page).
        Limit(limit).
        Execute()

    if err != nil {
        log.Fatal(err)
    }

    if len(getAllOutboundMessageResponse.Resources) > 0 {
        fmt.Println("First message ID:", getAllOutboundMessageResponse.Resources[0].Id)
    }
}
```

## Features

- ✅ Full TextMagic API support
- ✅ Type-safe API calls
- ✅ Context-based authentication
- ✅ File upload support
- ✅ Comprehensive error handling
- ✅ Modern Go idioms (Go 1.23+)
- ✅ No external dependencies for optional parameters
- ✅ Fluent API for method chaining

## Error Handling

The SDK returns standard Go errors that should be properly handled:

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    tm "github.com/textmagic/textmagic-rest-go-v2/v3"
)

func main() {
    cfg := tm.NewConfiguration()
    cfg.Servers = tm.ServerConfigurations{
        {
            URL: "https://rest.textmagic.com",
            Description: "TextMagic REST API",
        },
    }
    
    client := tm.NewAPIClient(cfg)
    auth := context.WithValue(context.Background(), tm.ContextBasicAuth, tm.BasicAuth{
        UserName: "YOUR_USERNAME",
        Password: "YOUR_API_KEY",
    })

    sendMessageInput := tm.SendMessageInputObject{
        Text:   "Test message",
        Phones: "+1234567890",
    }
    
    response, httpResp, err := client.TextMagicAPI.SendMessage(auth).
        SendMessageInputObject(sendMessageInput).
        Execute()
        
    if err != nil {
        // Handle different types of errors
        if httpResp != nil {
            switch httpResp.StatusCode {
            case 401:
                log.Fatal("Authentication failed: Invalid credentials")
            case 400:
                log.Fatal("Bad request: Invalid parameters")
            case 404:
                log.Fatal("Resource not found")
            case 429:
                log.Fatal("Rate limit exceeded")
            default:
                log.Fatalf("API error (status %d): %v", httpResp.StatusCode, err)
            }
        } else {
            log.Fatalf("Network error: %v", err)
        }
    }
    
    fmt.Printf("Message sent successfully! ID: %s\n", response.Id)
}
```

### Common Error Codes

- **401 Unauthorized** - Invalid credentials
- **400 Bad Request** - Invalid parameters
- **404 Not Found** - Resource not found
- **429 Too Many Requests** - Rate limit exceeded
- **500 Internal Server Error** - Server error

## API Documentation

For complete API documentation, including all available methods, parameters, and response formats, please visit:

- 📖 **[TextMagic API Documentation](https://docs.textmagic.com/)**
- 🔗 **[API Reference](https://docs.textmagic.com/#tag/TextMagic-API)**
- 📦 **[Go Package Documentation](https://pkg.go.dev/github.com/textmagic/textmagic-rest-go-v2/v3)** - Official Go package registry with full SDK reference

### Available API Methods

The SDK provides access to all TextMagic API endpoints through the `TextMagicAPI` interface. Some commonly used methods include:

**Messaging:**
- `SendMessage(ctx)` - Send SMS messages
- `GetAllOutboundMessages(ctx)` - Get sent messages
- `GetAllInboundMessages(ctx)` - Get received messages
- `DeleteMessage(ctx, id)` - Delete a message
- `GetMessagesBySessionId(ctx, id)` - Get messages by session ID

**Contacts:**
- `CreateContact(ctx)` - Create a new contact
- `GetContact(ctx, id)` - Get contact details
- `UpdateContact(ctx, id)` - Update contact information
- `DeleteContact(ctx, id)` - Delete a contact
- `GetAllContacts(ctx)` - Get all contacts

**Lists:**
- `CreateList(ctx)` - Create a contact list
- `GetList(ctx, id)` - Get list details
- `GetAllLists(ctx)` - Get all lists
- `AssignContactsToList(ctx, id)` - Add contacts to a list

**Account:**
- `Ping(ctx)` - Test API connection
- `GetUser(ctx)` - Get account information
- `GetUserBalance(ctx)` - Get account balance

For a complete list of available methods, please refer to the generated SDK documentation in the `docs/` directory.

## Migration Guide from v1.x to v2.x

### Breaking Changes

#### Go Version Requirement

**v1.x (Swagger Codegen):**
```go
go 1.13
```

**v2.x (OpenAPI Generator):**
```go
go 1.23
```

**Action Required:** Upgrade your Go version to 1.23 or later.

```bash
# Check your Go version
go version

# Should output: go1.23.x or higher
```

#### Dependencies Update

**v1.x (Swagger Codegen):**
```go
import (
    "github.com/antihax/optional"
    tm "github.com/textmagic/textmagic-rest-go-v2"
)

// Optional parameters required antihax/optional
page := optional.NewInt32(1)
limit := optional.NewInt32(10)
```

**v2.x (OpenAPI Generator):**
```go
import (
    tm "github.com/textmagic/textmagic-rest-go-v2/v3"
)

// Optional parameters use fluent API
page := int32(1)
limit := int32(10)
```

**Note:**
- The `antihax/optional` dependency is **no longer required**
- OpenAPI Generator 7.x uses built-in Go types for optional parameters
- Fluent API for method chaining

#### Import Path Change

**v1.x:**
```go
import tm "github.com/textmagic/textmagic-rest-go-v2"
```

**v2.x:**
```go
import tm "github.com/textmagic/textmagic-rest-go-v2/v3"
```

**Note:** The `/v3` suffix is required for Go modules compatibility.

#### API Usage Changes

**v1.x (Swagger Codegen):**
```go
api := tm.NewAPIClient(cfg)

// Optional parameters with antihax/optional
opts := &tm.GetAllOutboundMessagesOpts{
    Page:  optional.NewInt32(1),
    Limit: optional.NewInt32(10),
}

result, _, err := api.TextMagicApi.GetAllOutboundMessages(ctx, opts)
```

**v2.x (OpenAPI Generator):**
```go
client := tm.NewAPIClient(cfg)

// Fluent API with method chaining
result, _, err := client.TextMagicAPI.GetAllOutboundMessages(ctx).
    Page(1).
    Limit(10).
    Execute()
```

### What Stays the Same

✅ **Authentication** - Configuration remains the same:
```go
auth := context.WithValue(context.Background(), tm.ContextBasicAuth, tm.BasicAuth{
    UserName: "YOUR_USERNAME",
    Password: "YOUR_API_KEY",
})
```

✅ **API Methods** - All methods remain the same (with improved syntax)
✅ **Models** - All data structures remain compatible
✅ **Error Handling** - Standard Go error handling

### Step-by-Step Migration

1. **Upgrade Go to 1.23+**
   ```bash
   # Download and install Go 1.23
   # https://go.dev/dl/
   
   # Verify
   go version
   ```

2. **Update Import Path**
   ```bash
   # Update go.mod
   go get -u github.com/textmagic/textmagic-rest-go-v2/v3@latest
   
   # Remove old version
   go mod tidy
   ```

3. **Update Your Code**

   **Before (v1.x):**
   ```go
   import (
       "github.com/antihax/optional"
       tm "github.com/textmagic/textmagic-rest-go-v2"
   )
   
   opts := &tm.GetAllOutboundMessagesOpts{
       Page:  optional.NewInt32(1),
       Limit: optional.NewInt32(10),
   }
   result, _, err := api.TextMagicApi.GetAllOutboundMessages(ctx, opts)
   ```

   **After (v2.x):**
   ```go
   import tm "github.com/textmagic/textmagic-rest-go-v2/v3"
   
   result, _, err := client.TextMagicAPI.GetAllOutboundMessages(ctx).
       Page(1).
       Limit(10).
       Execute()
   ```

4. **Remove antihax/optional**
   ```bash
   # Remove from imports and code
   # Run go mod tidy to clean up
   go mod tidy
   ```

5. **Test Your Application**
   ```bash
   # Run your tests
   go test ./...
   
   # Build your application
   go build
   ```

### Compatibility Matrix

|    Feature    | v1.x (Swagger) | v2.x (OpenAPI) | Compatible? |
|---------------|----------------|----------------|-------------|
| Go 1.13-1.22  |       ✅       |       ❌       |    ❌ No    |
| Go 1.23+      |       ✅       |       ✅       |    ✅ Yes   |
| API Methods   |      Same      |      Same      |    ✅ Yes   |
| Models        |      Same      |      Same      |    ✅ Yes   |
| Authentication|      Same      |      Same      |    ✅ Yes   |
| File Upload   |       ❌       |       ✅       |    ⚠️ New   |
| Optional Params| antihax/optional | Fluent API   |    ⚠️ Changed |
| Import Path   | `/textmagic-rest-go-v2` | `/textmagic-rest-go-v2/v3` | ⚠️ Changed |

### Need Help?

- 📖 [Full Documentation](https://docs.textmagic.com/)
- 💬 [Support](https://www.textmagic.com/support/)
- 🐛 [Report Issues](https://github.com/textmagic/textmagic-rest-go-v2/issues)

[comment]: <> (FOOTER)
## License
The library is available as open source under the terms of the [MIT License](http://opensource.org/licenses/MIT).

[comment]: <> (/FOOTER)
