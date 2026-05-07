package oauth1

import (
	"errors"
)

// A TokenSource can return a Token.
type TokenSource interface {
	Token() (*Token, error)
}

// Token is an AccessToken (token credential) which allows a consumer (client)
// to access resources from an OAuth1 provider server.
type Token struct {
	Token       string
	TokenSecret string
}

// NewToken returns a new Token with the given token and token secret.
func NewToken(token, tokenSecret string) *Token {
	return &Token{
		Token:       token,
		TokenSecret: tokenSecret,
	}
}

// StaticTokenSource returns a TokenSource which always returns the same Token.
// This is appropriate for tokens which do not have a time expiration.
func StaticTokenSource(token *Token) TokenSource {
	return staticTokenSource{token}
}

// staticTokenSource is a TokenSource that always returns the same Token.
type staticTokenSource struct {
	token *Token
}

func (s staticTokenSource) Token() (*Token, error) {
	if s.token == nil {
		return nil, errors.New("oauth1: Token is nil")
	}
	return s.token, nil
}
