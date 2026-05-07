/*
Package twitter provides a Client for the Twitter API.


The twitter package provides a Client for accessing the Twitter API. Here are
some example requests.

	// Twitter client
	client := twitter.NewClient(httpClient)
	// Home Timeline
	tweets, resp, err := client.Timelines.HomeTimeline(&HomeTimelineParams{})
	// Send a Tweet
	tweet, resp, err := client.Statuses.Update("just setting up my twttr", nil)
	// Status Show
	tweet, resp, err := client.Statuses.Show(585613041028431872, nil)
	// User Show
	params := &twitter.UserShowParams{ScreenName: "dghubble"}
	user, resp, err := client.Users.Show(params)
	// Followers
	followers, resp, err := client.Followers.List(&FollowerListParams{})

Required parameters are passed as positional arguments. Optional parameters
are passed in a typed params struct (or pass nil).

Authentication

By design, the Twitter Client accepts any http.Client so user auth (OAuth1) or
application auth (OAuth2) requests can be made by using the appropriate
authenticated client. Use the https://github.com/dghubble/oauth1 and
https://github.com/golang/oauth2 packages to obtain an http.Client which
transparently authorizes requests.

For example, make requests as a consumer application on behalf of a user who
has granted access, with OAuth1.

	// OAuth1
	import (
		"github.com/dghubble/go-twitter/twitter"
		"github.com/dghubble/oauth1"
	)

	config := oauth1.NewConfig("consumerKey", "consumerSecret")
	token := oauth1.NewToken("accessToken", "accessSecret")
	// http.Client will automatically authorize Requests
	httpClient := config.Client(oauth1.NoContext, token)

	// twitter client
	client := twitter.NewClient(httpClient)

If no user auth context is needed, make requests as your application with
application auth.

	// OAuth2
	import (
		"github.com/dghubble/go-twitter/twitter"
		"golang.org/x/oauth2"
		"golang.org/x/oauth2/clientcredentials"
	)

	// oauth2 configures a client that uses app credentials to keep a fresh token
	config := &clientcredentials.Config{
		ClientID:     flags.consumerKey,
		ClientSecret: flags.consumerSecret,
		TokenURL:     "https://api.twitter.com/oauth2/token",
	}
	// http.Client will automatically authorize Requests
	httpClient := config.Client(oauth2.NoContext)

	// Twitter client
	client := twitter.NewClient(httpClient)

To implement Login with Twitter, see https://github.com/dghubble/gologin.

*/
package twitter
