/*
Docs:
- https://www.reddit.com/dev/api/
- https://github.com/reddit-archive/reddit/wiki/api
- https://github.com/reddit-archive/reddit/wiki/OAuth2
- https://github.com/reddit-archive/reddit/wiki/OAuth2-Quick-Start-Example

1. Go to https://www.reddit.com/prefs/apps and create an app. There are 3 types of apps:
	- Web app. Service is available over http or https, preferably the latter.
	- Installed app, such as a mobile app on a user's device which you can't control.
	  Redirect the user to a URI after they grant your app permissions.
	- Script (the simplest type of app). Select this if you are the only person who will
	  use the app. Only has access to your account.

Best option for a client like this is to use the script option.

2. After creating the app, you will get a client id and client secret.

3. Send a POST request (with the Content-Type header set to "application/x-www-form-urlencoded")
to https://www.reddit.com/api/v1/access_token with the following form values:
	- grant_type=password
	- username={your Reddit username}
	- password={your Reddit password}

4. You should receive a response body like the following:
{
	"access_token": "70743860-DRhHVNSEOMu1ldlI",
	"token_type": "bearer",
	"expires_in": 3600,
	"scope": "*"
}
*/

package reddit

import (
	"context"
	"net/http"

	"golang.org/x/oauth2"
)

type oauthTokenSource struct {
	ctx                context.Context
	config             *oauth2.Config
	username, password string
}

func (s *oauthTokenSource) Token() (*oauth2.Token, error) {
	return s.config.PasswordCredentialsToken(s.ctx, s.username, s.password)
}

func oauthTransport(client *Client) http.RoundTripper {
	httpClient := &http.Client{Transport: client.client.Transport}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)

	config := &oauth2.Config{
		ClientID:     client.ID,
		ClientSecret: client.Secret,
		Endpoint: oauth2.Endpoint{
			TokenURL:  client.TokenURL.String(),
			AuthStyle: oauth2.AuthStyleInHeader,
		},
	}

	tokenSource := oauth2.ReuseTokenSource(nil, &oauthTokenSource{
		ctx:      ctx,
		config:   config,
		username: client.Username,
		password: client.Password,
	})

	return &oauth2.Transport{
		Source: tokenSource,
		Base:   client.client.Transport,
	}
}
