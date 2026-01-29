// Copyright 2022 Google Inc. All Rights Reserved.
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

// Package appcheck provides functionality for verifying App Check tokens.
package appcheck

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc"
	"github.com/golang-jwt/jwt/v4"

	"firebase.google.com/go/v4/internal"
)

// JWKSUrl is the URL of the JWKS used to verify App Check tokens.
var JWKSUrl = "https://firebaseappcheck.googleapis.com/v1beta/jwks"

const appCheckIssuer = "https://firebaseappcheck.googleapis.com/"

var (
	// ErrIncorrectAlgorithm is returned when the token is signed with a non-RSA256 algorithm.
	ErrIncorrectAlgorithm = errors.New("token has incorrect algorithm")
	// ErrTokenType is returned when the token is not a JWT.
	ErrTokenType = errors.New("token has incorrect type")
	// ErrTokenClaims is returned when the token claims cannot be decoded.
	ErrTokenClaims = errors.New("token has incorrect claims")
	// ErrTokenAudience is returned when the token audience does not match the current project.
	ErrTokenAudience = errors.New("token has incorrect audience")
	// ErrTokenIssuer is returned when the token issuer does not match Firebase's App Check service.
	ErrTokenIssuer = errors.New("token has incorrect issuer")
	// ErrTokenSubject is returned when the token subject is empty or missing.
	ErrTokenSubject = errors.New("token has empty or missing subject")
)

// DecodedAppCheckToken represents a verified App Check token.
//
// DecodedAppCheckToken provides typed accessors to the common JWT fields such as Audience (aud)
// and ExpiresAt (exp). Additionally it provides an AppID field, which indicates the application ID to which this
// token belongs. Any additional JWT claims can be accessed via the Claims map of DecodedAppCheckToken.
type DecodedAppCheckToken struct {
	Issuer    string
	Subject   string
	Audience  []string
	ExpiresAt time.Time
	IssuedAt  time.Time
	AppID     string
	Claims    map[string]interface{}
}

// Client is the interface for the Firebase App Check service.
type Client struct {
	projectID string
	jwks      *keyfunc.JWKS
}

// NewClient creates a new instance of the Firebase App Check Client.
//
// This function can only be invoked from within the SDK. Client applications should access the
// the App Check service through firebase.App.
func NewClient(ctx context.Context, conf *internal.AppCheckConfig) (*Client, error) {
	// TODO: Add support for overriding the HTTP client using the App one.
	jwks, err := keyfunc.Get(JWKSUrl, keyfunc.Options{
		Ctx:             ctx,
		RefreshInterval: 6 * time.Hour,
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		projectID: conf.ProjectID,
		jwks:      jwks,
	}, nil
}

// VerifyToken verifies the given App Check token.
//
// VerifyToken considers an App Check token string to be valid if all the following conditions are met:
//   - The token string is a valid RS256 JWT.
//   - The JWT contains valid issuer (iss) and audience (aud) claims that match the issuerPrefix
//     and projectID of the tokenVerifier.
//   - The JWT contains a valid subject (sub) claim.
//   - The JWT is not expired, and it has been issued some time in the past.
//   - The JWT is signed by a Firebase App Check backend server as determined by the keySource.
//
// If any of the above conditions are not met, an error is returned. Otherwise a pointer to a
// decoded App Check token is returned.
func (c *Client) VerifyToken(token string) (*DecodedAppCheckToken, error) {
	// References for checks:
	// https://firebase.googleblog.com/2021/10/protecting-backends-with-app-check.html
	// https://github.com/firebase/firebase-admin-node/blob/master/src/app-check/token-verifier.ts#L106

	// The standard JWT parser also validates the expiration of the token
	// so we do not need dedicated code for that.
	decodedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if t.Header["alg"] != "RS256" {
			return nil, ErrIncorrectAlgorithm
		}
		if t.Header["typ"] != "JWT" {
			return nil, ErrTokenType
		}
		return c.jwks.Keyfunc(t)
	})
	if err != nil {
		return nil, err
	}

	claims, ok := decodedToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrTokenClaims
	}

	rawAud := claims["aud"].([]interface{})
	aud := []string{}
	for _, v := range rawAud {
		aud = append(aud, v.(string))
	}

	if !contains(aud, "projects/"+c.projectID) {
		return nil, ErrTokenAudience
	}

	// We check the prefix to make sure this token was issued
	// by the Firebase App Check service, but we do not check the
	// Project Number suffix because the Golang SDK only has project ID.
	//
	// This is consistent with the Firebase Admin Node SDK.
	if !strings.HasPrefix(claims["iss"].(string), appCheckIssuer) {
		return nil, ErrTokenIssuer
	}

	if val, ok := claims["sub"].(string); !ok || val == "" {
		return nil, ErrTokenSubject
	}

	appCheckToken := DecodedAppCheckToken{
		Issuer:    claims["iss"].(string),
		Subject:   claims["sub"].(string),
		Audience:  aud,
		ExpiresAt: time.Unix(int64(claims["exp"].(float64)), 0),
		IssuedAt:  time.Unix(int64(claims["iat"].(float64)), 0),
		AppID:     claims["sub"].(string),
	}

	// Remove all the claims we've already parsed.
	for _, usedClaim := range []string{"iss", "sub", "aud", "exp", "iat", "sub"} {
		delete(claims, usedClaim)
	}
	appCheckToken.Claims = claims

	return &appCheckToken, nil
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}
