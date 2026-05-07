package oauth1

// Endpoint represents an OAuth1 provider's (server's) request token,
// owner authorization, and access token request URLs.
type Endpoint struct {
	// Request URL (Temporary Credential Request URI)
	RequestTokenURL string
	// Authorize URL (Resource Owner Authorization URI)
	AuthorizeURL string
	// Access Token URL (Token Request URI)
	AccessTokenURL string
}
