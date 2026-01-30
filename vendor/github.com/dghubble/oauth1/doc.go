/*
Package oauth1 is a Go implementation of the OAuth1 spec RFC 5849.

It allows end-users to authorize a client (consumer) to access protected
resources on their behalf (e.g. login) and allows clients to make signed and
authorized requests on behalf of a user (e.g. API calls).

It takes design cues from golang.org/x/oauth2, providing an http.Client which
handles request signing and authorization.

# Usage

Package oauth1 implements the OAuth1 authorization flow and provides an
http.Client which can sign and authorize OAuth1 requests.

To implement "Login with X", use the https://github.com/dghubble/gologin
packages which provide login handlers for OAuth1 and OAuth2 providers.

To call the Twitter, Digits, or Tumblr OAuth1 APIs, use the higher level Go API
clients.

* https://github.com/dghubble/go-twitter
* https://github.com/dghubble/go-digits
* https://github.com/benfb/go-tumblr

# Authorization Flow

Perform the OAuth 1 authorization flow to ask a user to grant an application
access to his/her resources via an access token.

	import (
		"github.com/dghubble/oauth1"
		"github.com/dghubble/oauth1/twitter""
	)
	...

	config := oauth1.Config{
		ConsumerKey:    "consumerKey",
		ConsumerSecret: "consumerSecret",
		CallbackURL:    "http://mysite.com/oauth/twitter/callback",
		Endpoint:       twitter.AuthorizeEndpoint,
	}

1. When a user performs an action (e.g. "Login with X" button calls "/login"
route) get an OAuth1 request token (temporary credentials).

	requestToken, requestSecret, err = config.RequestToken()
	// handle err

2. Obtain authorization from the user by redirecting them to the OAuth1
provider's authorization URL to grant the application access.

	authorizationURL, err := config.AuthorizationURL(requestToken)
	// handle err
	http.Redirect(w, req, authorizationURL.String(), http.StatusFound)

Receive the callback from the OAuth1 provider in a handler.

	requestToken, verifier, err := oauth1.ParseAuthorizationCallback(req)
	// handle err

3. Acquire the access token (token credentials) which can later be used
to make requests on behalf of the user.

	accessToken, accessSecret, err := config.AccessToken(requestToken, requestSecret, verifier)
	// handle error
	token := oauth1.NewToken(accessToken, accessSecret)

Check the examples to see this authorization flow in action from the command
line, with Twitter PIN-based login and Tumblr login.

# Authorized Requests

Use an access Token to make authorized requests on behalf of a user.

	import (
		"github.com/dghubble/oauth1"
	)

	func main() {
	    config := oauth1.NewConfig("consumerKey", "consumerSecret")
	    token := oauth1.NewToken("token", "tokenSecret")

	    // httpClient will automatically authorize http.Request's
	    httpClient := config.Client(token)

	    // example Twitter API request
	    path := "https://api.twitter.com/1.1/statuses/home_timeline.json?count=2"
	    resp, _ := httpClient.Get(path)
	    defer resp.Body.Close()
	    body, _ := ioutil.ReadAll(resp.Body)
	    fmt.Printf("Raw Response Body:\n%v\n", string(body))
	}

Check the examples to see Twitter and Tumblr requests in action.
*/
package oauth1
