# Usage

Usage examples for SendGrid REST library

## Initialization

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sendgrid/rest"
)

// Build the URL
const host = "https://api.sendgrid.com"
endpoint := "/v3/api_keys"
baseURL := host + endpoint

// Build the request headers
key := os.Getenv("SENDGRID_API_KEY")
Headers := make(map[string]string)
Headers["Authorization"] = "Bearer " + key
```

## Table of Contents

- [GET](#get)
- [DELETE](#delete)
- [POST](#post)
- [PUT](#put)
- [PATCH](#patch)

<a name="get"></a>
## GET

#### GET Single

```go
method = rest.Get

// Make the API call
request = rest.Request{
    Method:  method,
    BaseURL: baseURL + "/" + apiKey,
    Headers: Headers,
}
response, err = rest.API(request)
if err != nil {
    fmt.Println(err)
} else {
    fmt.Println(response.StatusCode)
    fmt.Println(response.Body)
    fmt.Println(response.Headers)
}
```

#### GET Collection

```go
method := rest.Get

// Build the query parameters
queryParams := make(map[string]string)
queryParams["limit"] = "100"
queryParams["offset"] = "0"

// Make the API call
request := rest.Request{
    Method:      method,
    BaseURL:     baseURL,
    Headers:     Headers,
    QueryParams: queryParams,
}
response, err := rest.API(request)
if err != nil {
    fmt.Println(err)
} else {
    fmt.Println(response.StatusCode)
    fmt.Println(response.Body)
    fmt.Println(response.Headers)
}
```
<a name="delete"></a>
## DELETE

```go
method = rest.Delete

// Make the API call
request = rest.Request{
	Method:      method,
	BaseURL:     baseURL + "/" + apiKey,
	Headers:     Headers,
	QueryParams: queryParams,
}
response, err = rest.API(request)
if err != nil {
	fmt.Println(err)
} else {
	fmt.Println(response.StatusCode)
	fmt.Println(response.Headers)
}
```

<a name="post"></a>
## POST

```go
method = rest.Post

// Build the request body
var Body = []byte(`{
    "name": "My API Key",
    "scopes": [
        "mail.send",
        "alerts.create",
        "alerts.read"
    ]
}`)

// Make the API call
request = rest.Request{
	Method:      method,
	BaseURL:     baseURL,
	Headers:     Headers,
	QueryParams: queryParams,
	Body:        Body,
}
response, err = rest.API(request)
if err != nil {
	fmt.Println(err)
} else {
	fmt.Println(response.StatusCode)
	fmt.Println(response.Body)
	fmt.Println(response.Headers)
}

// Get a particular return value.
// Note that you can unmarshall into a struct if
// you know the JSON structure in advance.
b := []byte(response.Body)
var f interface{}
err = json.Unmarshal(b, &f)
if err != nil {
	fmt.Println(err)
}
m := f.(map[string]interface{})
apiKey := m["api_key_id"].(string)
```
<a name="put"></a>
## PUT

```go
method = rest.Put

// Build the request body
Body = []byte(`{
    "name": "A New Hope",
    "scopes": [
        "user.profile.read",
        "user.profile.update"
    ]
}`)

// Make the API call
request = rest.Request{
	Method:  method,
	BaseURL: baseURL + "/" + apiKey,
	Headers: Headers,
	Body:    Body,
}
response, err = rest.API(request)
if err != nil {
	fmt.Println(err)
} else {
	fmt.Println(response.StatusCode)
	fmt.Println(response.Body)
	fmt.Println(response.Headers)
}
```
<a name="patch"></a>
## PATCH

```go
method = rest.Patch

// Build the request body
Body = []byte(`{
    "name": "A New Hope"
}`)

// Make the API call
request = rest.Request{
	Method:  method,
	BaseURL: baseURL + "/" + apiKey,
	Headers: Headers,
	Body:    Body,
}
response, err = rest.API(request)
if err != nil {
	fmt.Println(err)
} else {
	fmt.Println(response.StatusCode)
	fmt.Println(response.Body)
	fmt.Println(response.Headers)
}
```