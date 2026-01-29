## Table of Contents

* [Viewing the Request Body](#request-body)


<a name="request-body"></a>
## Viewing the Request Body

When debugging or testing, it may be useful to examine the raw request body to compare against the [documented format](https://sendgrid.com/docs/API_Reference/api_v3.html).

Example Code
```go
package main

import "github.com/sendgrid/rest"
import "fmt"

func main() {
	const host = "https://api.example.com"
	param := "myparam"
	endpoint := "/your/api/" + param + "/call"
	baseURL := host + endpoint
	Headers := make(map[string]string)
	key := os.Getenv("API_KEY")
	Headers["Authorization"] = "Bearer " + key
	Headers["X-Test"] = "Test"
	var Body = []byte(`{"some": 0, "awesome": 1, "data": 3}`)
	queryParams := make(map[string]string)
	queryParams["hello"] = "0"
	queryParams["world"] = "1"
	method := rest.Post
	request = rest.Request{
		Method:      method,
		BaseURL:     baseURL,
		Headers:     Headers,
		QueryParams: queryParams,
		Body:        Body,
	}
	response, err := rest.API(request)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(response.StatusCode)
		fmt.Println(response.Body)
		fmt.Println(response.Headers)
	}
}
```

You can do this right before you call 
`response, err := rest.API(request)` like so:

```go
fmt.Printf("Request Body: %v \n", string(request.Body))

req, e := BuildRequestObject(request)
requestDump, err := httputil.DumpRequest(req, true)
if err != nil {
	t.Errorf("Error : %v", err)
}
fmt.Printf("Request : %v \n", string(requestDump))
```