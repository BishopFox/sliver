// Package google provides an implementation of the fantasy AI SDK for Google's language models.
package google

import (
	"context"

	"cloud.google.com/go/auth"
)

type dummyTokenProvider struct{}

// Token implements the auth.TokenProvider interface.
func (dummyTokenProvider) Token(_ context.Context) (*auth.Token, error) {
	return &auth.Token{Value: "dummy-token"}, nil
}
