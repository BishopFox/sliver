// Copyright 2026 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package phonenumberverification provides functionality for verifying Firebase Phone Number Verification (FPNV) tokens.
package phonenumberverification

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc"
	"github.com/golang-jwt/jwt/v4"

	"firebase.google.com/go/v4/internal"
)

const (
	jwksURL      = "https://fpnv.googleapis.com/v1beta/jwks"
	issuerPrefix = "https://fpnv.googleapis.com/projects/"
	algorithm    = "ES256"
	headerTyp    = "JWT"
)

var (
	// ErrProjectIDRequired is returned when the project ID is not available.
	ErrProjectIDRequired = errors.New("project ID is required to access phone number verification client")
	// ErrEmptyToken is returned when the provided token is empty.
	ErrEmptyToken = errors.New("token must not be empty")
	// ErrTokenHeaderKid is returned when the token has no 'kid' claim.
	ErrTokenHeaderKid = errors.New("token has no 'kid' claim")
	// ErrIncorrectAlgorithm is returned when the token is signed with a non-ES256 algorithm.
	ErrIncorrectAlgorithm = errors.New("token must be signed with ES256")
	// ErrTokenType is returned when the token is not a JWT.
	ErrTokenType = errors.New("token is not a JWT")
	// ErrTokenClaims is returned when the token claims cannot be decoded.
	ErrTokenClaims = errors.New("token has incorrect claims")
	// ErrTokenEmptyAudience is returned when the token has no audience.
	ErrTokenEmptyAudience = errors.New("token has no 'aud' claim")
	// ErrTokenAudience is returned when the token audience does not match the current project.
	ErrTokenAudience = errors.New("token audience must be the project name")
	// ErrTokenIssuer is returned when the token issuer does not match phone number verification service.
	ErrTokenIssuer = errors.New("token was not issued by the PNV service")
	// ErrTokenSubject is returned when the token subject is empty or missing.
	ErrTokenSubject = errors.New("token has empty or missing subject")
	// ErrTokenExpiresAt is returned when the token has an invalid expiresAt.
	ErrTokenExpiresAt = errors.New("token has an invalid expiresAt")
	// ErrTokenIssuedAt is returned when the token has an invalid issuedAt.
	ErrTokenIssuedAt = errors.New("token has an invalid issuedAt")
)

// DecodedVerificationToken represents a verified FPNV token.
//
// DecodedVerificationToken provides typed accessors to the common JWT fields such as Audience (aud)
// and ExpiresAt (exp). Additionally, it provides an PhoneNumber field,
// which is an alias for Subject (sub).
// Any additional JWT claims can be accessed via the Claims map of DecodedVerificationToken.
type DecodedVerificationToken struct {
	Issuer      string
	Subject     string
	Audience    []string
	ExpiresAt   time.Time
	IssuedAt    time.Time
	PhoneNumber string
	Claims      map[string]interface{}
}

// Client is the client for the Firebase Phone Number Verification service.
type Client struct {
	projectID string
	jwks      *keyfunc.JWKS
}

// NewClient creates a new instance of the Firebase Phone Number Verification Client.
//
// This function can only be invoked from within the SDK. Client applications should access the
// FPNV service through firebase.App.
func NewClient(ctx context.Context, conf *internal.PhoneNumberVerificationConfig) (*Client, error) {
	// TODO: Add support for overriding the HTTP client using the App one.
	jwks, err := keyfunc.Get(jwksURL, keyfunc.Options{
		Ctx:               ctx,
		RefreshUnknownKID: true,
		RefreshRateLimit:  5 * time.Minute, // Prevent network floods from malicious tokens
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		projectID: conf.ProjectID,
		jwks:      jwks,
	}, nil
}

// VerifyToken verifies the given Firebase Phone Number Verification (FPNV) token.
//
// VerifyToken considers a Firebase Phone Number Verification token string to be valid
// if all the following conditions are met:
//   - The token string is a valid ES256 JWT.
//   - The JWT contains valid issuer (iss) and audience (aud) claims that match the issuerPrefix
//     and projectID of the tokenVerifier.
//   - The JWT is not expired, and it has been issued some time in the past.
//
// If any of the above conditions are not met, an error is returned.
// Otherwise, a pointer to a decoded FPNV token is returned.
func (c *Client) VerifyToken(token string) (*DecodedVerificationToken, error) {
	if c.projectID == "" {
		return nil, ErrProjectIDRequired
	}

	if token == "" {
		return nil, ErrEmptyToken
	}

	// The standard JWT parser also validates the expiration of the token
	// so we do not need dedicated code for that.

	// Header part
	decodedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		kid, ok := t.Header["kid"].(string)
		if !ok || kid == "" {
			return nil, ErrTokenHeaderKid
		}
		alg, ok := t.Header["alg"].(string)
		if !ok || alg != algorithm {
			return nil, ErrIncorrectAlgorithm
		}
		typ, ok := t.Header["typ"].(string)
		if !ok || typ != headerTyp {
			return nil, ErrTokenType
		}
		return c.jwks.Keyfunc(t)
	})

	if err != nil {
		return nil, err
	}

	// Payload part
	claims, ok := decodedToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrTokenClaims
	}

	_, okAud := claims["aud"]
	if !okAud {
		return nil, ErrTokenEmptyAudience
	}

	var aud []string
	switch v := claims["aud"].(type) {
	case string:
		aud = []string{v}
	case []interface{}:
		for _, s := range v {
			if str, ok := s.(string); ok {
				aud = append(aud, str)
			}
		}
	}

	if !slices.Contains(aud, issuerPrefix+c.projectID) {
		return nil, ErrTokenAudience
	}

	// Prepare claims for DecodedVerificationToken
	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return nil, ErrTokenSubject
	}
	iss, ok := claims["iss"].(string)
	// We check the prefix to make sure this token was issued
	// by the Firebase Phone Number Verification service, but we do not check the
	// Project Number suffix because the Golang SDK only has project ID.
	//
	// This is consistent with the Firebase Admin Node SDK.
	if !ok || !strings.HasPrefix(iss, issuerPrefix) {
		return nil, ErrTokenIssuer
	}
	exp, ok := claims["exp"].(float64)
	if !ok || exp == 0 {
		return nil, ErrTokenExpiresAt
	}
	iat, ok := claims["iat"].(float64)
	if !ok || iat == 0 {
		return nil, ErrTokenIssuedAt
	}

	decodedVerificationToken := DecodedVerificationToken{
		Issuer:      iss,
		Subject:     sub,
		Audience:    aud,
		ExpiresAt:   time.Unix(int64(exp), 0),
		IssuedAt:    time.Unix(int64(iat), 0),
		PhoneNumber: sub,
	}

	// Remove all the claims we've already parsed.
	for _, usedClaim := range []string{"iss", "sub", "aud", "exp", "iat"} {
		delete(claims, usedClaim)
	}
	decodedVerificationToken.Claims = claims

	return &decodedVerificationToken, nil
}
