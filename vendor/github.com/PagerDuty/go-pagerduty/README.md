[![GoDoc](https://godoc.org/github.com/PagerDuty/go-pagerduty?status.svg)](http://godoc.org/github.com/PagerDuty/go-pagerduty) [![Go Report Card](https://goreportcard.com/badge/github.com/PagerDuty/go-pagerduty)](https://goreportcard.com/report/github.com/PagerDuty/go-pagerduty) [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/gojp/goreportcard/blob/master/LICENSE)
# go-pagerduty

go-pagerduty is a CLI and [go](https://golang.org/) client library for the [PagerDuty API](https://developer.pagerduty.com/api-reference/).

## Installation

To add the latest stable version to your project:
```cli
go get github.com/PagerDuty/go-pagerduty@v1.8
```

If you instead wish to work with the latest code from main:
```cli
go get github.com/PagerDuty/go-pagerduty@latest
```

## Usage

### CLI

The CLI requires an [authentication token](https://v2.developer.pagerduty.com/docs/authentication), which can be specified in `.pd.yml`
file in the home directory of the user, or passed as a command-line argument.
Example of config file:

```yaml
---
authtoken: fooBar
```

#### Commands
`pd` command provides a single entrypoint for all the API endpoints, with individual
API represented by their own sub commands. For an exhaustive list of sub-commands, try:
```
pd --help
```

An example of the `service` sub-command

```
pd service list
```

### Client Library

#### NOTICE: Breaking API Changes in v1.5.0

As part of the `v1.5.0` release, we have fixed features that have never worked
correctly and require a breaking API change to fix. One example is the issue
reported in [\#232](https://github.com/PagerDuty/go-pagerduty/issues/232), as
well as a handful of other examples within the [v1.5.0
milestone](https://github.com/PagerDuty/go-pagerduty/milestone/2).

If you are impacted by a breaking change in this release, you should audit the
functionality you depended on as it may not have been working. If you cannot
upgrade for some reason, the `v1.4.x` line of releases should still work. At the
time of writing `v1.4.3` was the latest, and we intend to backport any critical
fixes for the time being.

#### Example Usage

```go
package main

import (
	"fmt"
	"github.com/PagerDuty/go-pagerduty"
)

var	authtoken = "" // Set your auth token here

func main() {
	ctx := context.Background()
	client := pagerduty.NewClient(authtoken)

	var opts pagerduty.ListEscalationPoliciesOptions
	eps, err := client.ListEscalationPoliciesWithContext(ctx, opts)
	if err != nil {
		panic(err)
	}
	for _, p := range eps.EscalationPolicies {
		fmt.Println(p.Name)
	}
}
```

The PagerDuty API client also exposes its HTTP client as the `HTTPClient` field.
If you need to use your own HTTP client, for doing things like defining your own
transport settings, you can replace the default HTTP client with your own by
simply by setting a new value in the `HTTPClient` field.

#### API Error Responses

For cases where your request results in an error from the API, you can use the
`errors.As()` function from the standard library to extract the
`pagerduty.APIError` error value and inspect more details about the error,
including the HTTP response code and PagerDuty API Error Code.

```go
package main

import (
	"fmt"
	"github.com/PagerDuty/go-pagerduty"
)

var	authtoken = "" // Set your auth token here

func main() {
	ctx := context.Background()
	client := pagerduty.NewClient(authtoken)
	user, err := client.GetUserWithContext(ctx, "NOTREAL", pagerduty.GetUserOptions{})
	if err != nil {
		var aerr pagerduty.APIError

		if errors.As(err, &aerr) {
			if aerr.RateLimited() {
				fmt.Println("rate limited")
				return
			}

			fmt.Println("unknown status code:", aerr.StatusCode)

			return
		}

		panic(err)
	}
	fmt.Println(user)
}
```

#### Extending and Debugging Client

##### Extending The Client

The `*pagerduty.Client` has a `Do` method which allows consumers to wrap the
client, and make their own requests to the PagerDuty API. The method signature
is similar to that of the `http.Client.Do` method, except it also includes a
`bool` to incidate whether the API endpoint is authenticated (i.e., the REST
API). When the API is authenticated, the client will annotate the request with
the appropriate headers to be authenticated by the API.

If the PagerDuty client doesn't natively expose functionality that you wish to
use, such as undocumented JSON fields, you can use the `Do()` method to issue
your own request that you can parse the response of.

Likewise, you can use it to issue requests to the API for the purposes of
debugging. However, that's not the only mechanism for debugging.

##### Debugging the Client

The `*pagerduty.Client` has a method that allows consumers to enable debug
functionality, including interception of PagerDuty API responses. This is done
by using the `SetDebugFlag()` method using the `pagerduty.DebugFlag` unsigned
integer type. There are also exported constants to help consumers enable
specific debug behaviors.

###### Capturing Last PagerDuty Response

If you're not getting the response you expect from the PagerDuty Go client, you
can enable the `DebugCaptureLastResponse` debug flag to capture the HTTP
responses. You can then use one of the methods to make an API call, and then
inspect the API response received. For example:

```Go
client := pagerduty.NewClient("example")

client.SetDebugFlag(pagerduty.DebugCaptureLastResponse)

oncalls, err := client.ListOnCallsWithContext(ctx, pagerduty.ListOnCallOptions{})

resp, ok := client.LastAPIResponse()
if ok { // resp is an *http.Response we can inspect
	body, err := httputil.DumpResponse(resp, true)
    // ...
}
```

#### Included Packages

##### webhookv3

Support for V3 of PagerDuty Webhooks is provided via the `webhookv3` package.
The intent is for this package to provide signature verification and decoding
helpers.

## Contributing

1. Fork it ( https://github.com/PagerDuty/go-pagerduty/fork )
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Commit your changes (`git commit -am 'Add some feature'`)
4. Push to the branch (`git push origin my-new-feature`)
5. Create a new Pull Request

## License
[Apache 2](http://www.apache.org/licenses/LICENSE-2.0)
