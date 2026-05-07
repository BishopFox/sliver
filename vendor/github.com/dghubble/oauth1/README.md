# OAuth1
[![GoDoc](https://pkg.go.dev/badge/github.com/dghubble/oauth1.svg)](https://pkg.go.dev/github.com/dghubble/oauth1)
[![Workflow](https://github.com/dghubble/oauth1/actions/workflows/test.yaml/badge.svg)](https://github.com/dghubble/oauth1/actions/workflows/test.yaml?query=branch%3Amain)
[![Sponsors](https://img.shields.io/github/sponsors/dghubble?logo=github)](https://github.com/sponsors/dghubble)
[![Mastodon](https://img.shields.io/badge/follow-news-6364ff?logo=mastodon)](https://fosstodon.org/@typhoon)

<img align="right" src="https://storage.googleapis.com/dghubble/oauth1.png">

Package `oauth1` provides a Go implementation of the [OAuth 1 spec](https://tools.ietf.org/html/rfc5849) to allow end-users to authorize a client (i.e. consumer) to access protected resources on his/her behalf.

`oauth1` takes design cues from [golang.org/x/oauth2](https://godoc.org/golang.org/x/oauth2), to provide an analogous API and an `http.Client` with a Transport which signs/authorizes requests.

## Install

```
go get github.com/dghubble/oauth1
```

## Docs

Read [GoDoc](https://godoc.org/github.com/dghubble/oauth1)

## Usage

Package `oauth1` implements the OAuth1 authorization flow and provides an `http.Client` which can sign and authorize OAuth1 requests.

To implement "Login with X", use the [gologin](https://github.com/dghubble/gologin) packages which provide login handlers for OAuth1 and OAuth2 providers.

To call the Twitter, Digits, or Tumblr OAuth1 APIs, use the higher level Go API clients.

* [Twitter](https://github.com/dghubble/go-twitter)
* [Digits](https://github.com/dghubble/go-digits)
* [Tumblr](https://github.com/benfb/go-tumblr)

### Authorization Flow

Perform the OAuth 1 authorization flow to ask a user to grant an application access to his/her resources via an access token.

```go
import (
    "github.com/dghubble/oauth1"
    "github.com/dghubble/oauth1/twitter"
)
...

config := oauth1.Config{
    ConsumerKey:    "consumerKey",
    ConsumerSecret: "consumerSecret",
    CallbackURL:    "http://mysite.com/oauth/twitter/callback",
    Endpoint:       twitter.AuthorizeEndpoint,
}
```

1. When a user performs an action (e.g. "Login with X" button calls "/login" route) get an OAuth1 request token (temporary credentials).

    ```go
    requestToken, requestSecret, err = config.RequestToken()
    // handle err
    ```

2. Obtain authorization from the user by redirecting them to the OAuth1 provider's authorization URL to grant the application access.

    ```go
    authorizationURL, err := config.AuthorizationURL(requestToken)
    // handle err
    http.Redirect(w, req, authorizationURL.String(), http.StatusFound)
    ```

    Receive the callback from the OAuth1 provider in a handler.

    ```go
    requestToken, verifier, err := oauth1.ParseAuthorizationCallback(req)
    // handle err
    ```

3. Acquire the access token (token credentials) which can later be used to make requests on behalf of the user.

    ```go
    accessToken, accessSecret, err := config.AccessToken(requestToken, requestSecret, verifier)
    // handle error
    token := oauth1.NewToken(accessToken, accessSecret)
    ```

Check the [examples](examples) to see this authorization flow in action from the command line, with Twitter PIN-based login and Tumblr login.

### Authorized Requests

Use an access `Token` to make authorized requests on behalf of a user.

```go
import (
    "github.com/dghubble/oauth1"
)

func main() {
    config := oauth1.NewConfig("consumerKey", "consumerSecret")
    token := oauth1.NewToken("token", "tokenSecret")

    // httpClient will automatically authorize http.Request's
    httpClient := config.Client(oauth1.NoContext, token)

    // example Twitter API request
    path := "https://api.twitter.com/1.1/statuses/home_timeline.json?count=2"
    resp, _ := httpClient.Get(path)
    defer resp.Body.Close()
    body, _ := ioutil.ReadAll(resp.Body)
    fmt.Printf("Raw Response Body:\n%v\n", string(body))
}
```

Check the [examples](examples) to see Twitter and Tumblr requests in action.

### Concepts

An `Endpoint` groups an OAuth provider's token and authorization URL endpoints.Endpoints for common providers are provided in subpackages.

A `Config` stores a consumer application's consumer key and secret, the registered callback URL, and the `Endpoint` to which the consumer is registered. It provides OAuth1 authorization flow methods.

An OAuth1 `Token` is an access token which can be used to make signed requests on behalf of a user. See [Authorized Requests](#authorized-requests) for details.

If you've used the [golang.org/x/oauth2](https://godoc.org/golang.org/x/oauth2) package for OAuth2 before, this organization should be familiar.

## Contributing

See the [Contributing Guide](https://gist.github.com/dghubble/be682c123727f70bcfe7).

## License

[MIT License](LICENSE)
