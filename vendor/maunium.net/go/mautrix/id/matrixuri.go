// Copyright (c) 2021 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package id

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Errors that can happen when parsing matrix: URIs
var (
	ErrInvalidScheme       = errors.New("matrix URI scheme must be exactly 'matrix'")
	ErrInvalidPartCount    = errors.New("matrix URIs must have exactly 2 or 4 segments")
	ErrInvalidFirstSegment = errors.New("invalid identifier in first segment of matrix URI")
	ErrEmptySecondSegment  = errors.New("the second segment of the matrix URI must not be empty")
	ErrInvalidThirdSegment = errors.New("invalid identifier in third segment of matrix URI")
	ErrEmptyFourthSegment  = errors.New("the fourth segment of the matrix URI must not be empty when the third segment is present")
)

// Errors that can happen when parsing matrix.to URLs
var (
	ErrNotMatrixTo                        = errors.New("that URL is not a matrix.to URL")
	ErrInvalidMatrixToPartCount           = errors.New("matrix.to URLs must have exactly 1 or 2 segments")
	ErrEmptyMatrixToPrimaryIdentifier     = errors.New("the primary identifier in the matrix.to URL is empty")
	ErrInvalidMatrixToPrimaryIdentifier   = errors.New("the primary identifier in the matrix.to URL has an invalid sigil")
	ErrInvalidMatrixToSecondaryIdentifier = errors.New("the secondary identifier in the matrix.to URL has an invalid sigil")
)

var ErrNotMatrixToOrMatrixURI = errors.New("that URL is not a matrix.to URL nor matrix: URI")

// MatrixURI contains the result of parsing a matrix: URI using ParseMatrixURI
type MatrixURI struct {
	Sigil1 rune
	Sigil2 rune
	MXID1  string
	MXID2  string
	Via    []string
	Action string
}

// SigilToPathSegment contains a mapping from Matrix identifier sigils to matrix: URI path segments.
var SigilToPathSegment = map[rune]string{
	'$': "e",
	'#': "r",
	'!': "roomid",
	'@': "u",
}

func (uri *MatrixURI) getQuery() url.Values {
	q := make(url.Values)
	if uri.Via != nil && len(uri.Via) > 0 {
		q["via"] = uri.Via
	}
	if len(uri.Action) > 0 {
		q.Set("action", uri.Action)
	}
	return q
}

// String converts the parsed matrix: URI back into the string representation.
func (uri *MatrixURI) String() string {
	if uri == nil {
		return ""
	}
	parts := []string{
		SigilToPathSegment[uri.Sigil1],
		url.PathEscape(uri.MXID1),
	}
	if uri.Sigil2 != 0 {
		parts = append(parts, SigilToPathSegment[uri.Sigil2], url.PathEscape(uri.MXID2))
	}
	return (&url.URL{
		Scheme:   "matrix",
		Opaque:   strings.Join(parts, "/"),
		RawQuery: uri.getQuery().Encode(),
	}).String()
}

// MatrixToURL converts to parsed matrix: URI into a matrix.to URL
func (uri *MatrixURI) MatrixToURL() string {
	if uri == nil {
		return ""
	}
	fragment := fmt.Sprintf("#/%s", url.PathEscape(uri.PrimaryIdentifier()))
	if uri.Sigil2 != 0 {
		fragment = fmt.Sprintf("%s/%s", fragment, url.PathEscape(uri.SecondaryIdentifier()))
	}
	query := uri.getQuery().Encode()
	if len(query) > 0 {
		fragment = fmt.Sprintf("%s?%s", fragment, query)
	}
	// It would be nice to use URL{...}.String() here, but figuring out the Fragment vs RawFragment stuff is a pain
	return fmt.Sprintf("https://matrix.to/%s", fragment)
}

// PrimaryIdentifier returns the first Matrix identifier in the URI.
// Currently room IDs, room aliases and user IDs can be in the primary identifier slot.
func (uri *MatrixURI) PrimaryIdentifier() string {
	if uri == nil {
		return ""
	}
	return fmt.Sprintf("%c%s", uri.Sigil1, uri.MXID1)
}

// SecondaryIdentifier returns the second Matrix identifier in the URI.
// Currently only event IDs can be in the secondary identifier slot.
func (uri *MatrixURI) SecondaryIdentifier() string {
	if uri == nil || uri.Sigil2 == 0 {
		return ""
	}
	return fmt.Sprintf("%c%s", uri.Sigil2, uri.MXID2)
}

// UserID returns the user ID from the URI if the primary identifier is a user ID.
func (uri *MatrixURI) UserID() UserID {
	if uri != nil && uri.Sigil1 == '@' {
		return UserID(uri.PrimaryIdentifier())
	}
	return ""
}

// RoomID returns the room ID from the URI if the primary identifier is a room ID.
func (uri *MatrixURI) RoomID() RoomID {
	if uri != nil && uri.Sigil1 == '!' {
		return RoomID(uri.PrimaryIdentifier())
	}
	return ""
}

// RoomAlias returns the room alias from the URI if the primary identifier is a room alias.
func (uri *MatrixURI) RoomAlias() RoomAlias {
	if uri != nil && uri.Sigil1 == '#' {
		return RoomAlias(uri.PrimaryIdentifier())
	}
	return ""
}

// EventID returns the event ID from the URI if the primary identifier is a room ID or alias and the secondary identifier is an event ID.
func (uri *MatrixURI) EventID() EventID {
	if uri != nil && (uri.Sigil1 == '!' || uri.Sigil1 == '#') && uri.Sigil2 == '$' {
		return EventID(uri.SecondaryIdentifier())
	}
	return ""
}

// ParseMatrixURIOrMatrixToURL parses the given matrix.to URL or matrix: URI into a unified representation.
func ParseMatrixURIOrMatrixToURL(uri string) (*MatrixURI, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URI: %w", err)
	}
	if parsed.Scheme == "matrix" {
		return ProcessMatrixURI(parsed)
	} else if strings.HasSuffix(parsed.Hostname(), "matrix.to") {
		return ProcessMatrixToURL(parsed)
	} else {
		return nil, ErrNotMatrixToOrMatrixURI
	}
}

// ParseMatrixURI implements the matrix: URI parsing algorithm.
//
// Currently specified in https://github.com/matrix-org/matrix-doc/blob/master/proposals/2312-matrix-uri.md#uri-parsing-algorithm
func ParseMatrixURI(uri string) (*MatrixURI, error) {
	// Step 1: parse the URI according to RFC 3986
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URI: %w", err)
	}
	return ProcessMatrixURI(parsed)
}

// ProcessMatrixURI implements steps 2-7 of the matrix: URI parsing algorithm
// (i.e. everything except parsing the URI itself, which is done with url.Parse or ParseMatrixURI)
func ProcessMatrixURI(uri *url.URL) (*MatrixURI, error) {
	// Step 2: check that scheme is exactly `matrix`
	if uri.Scheme != "matrix" {
		return nil, ErrInvalidScheme
	}

	// Step 3: split the path into segments separated by /
	parts := strings.Split(uri.Opaque, "/")

	// Step 4: Check that the URI contains either 2 or 4 segments
	if len(parts) != 2 && len(parts) != 4 {
		return nil, ErrInvalidPartCount
	}

	var parsed MatrixURI

	// Step 5: Construct the top-level Matrix identifier
	//         a: find the sigil from the first segment
	switch parts[0] {
	case "u", "user":
		parsed.Sigil1 = '@'
	case "r", "room":
		parsed.Sigil1 = '#'
	case "roomid":
		parsed.Sigil1 = '!'
	default:
		return nil, fmt.Errorf("%w: '%s'", ErrInvalidFirstSegment, parts[0])
	}
	// b: find the identifier from the second segment
	if len(parts[1]) == 0 {
		return nil, ErrEmptySecondSegment
	}
	var err error
	parsed.MXID1, err = url.PathUnescape(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to url decode second segment %q: %w", parts[1], err)
	}

	// Step 6: if the first part is a room and the URI has 4 segments, construct a second level identifier
	if parsed.Sigil1 == '!' && len(parts) == 4 {
		// a: find the sigil from the third segment
		switch parts[2] {
		case "e", "event":
			parsed.Sigil2 = '$'
		default:
			return nil, fmt.Errorf("%w: '%s'", ErrInvalidThirdSegment, parts[0])
		}

		// b: find the identifier from the fourth segment
		if len(parts[3]) == 0 {
			return nil, ErrEmptyFourthSegment
		}
		parsed.MXID2, err = url.PathUnescape(parts[3])
		if err != nil {
			return nil, fmt.Errorf("failed to url decode fourth segment %q: %w", parts[3], err)
		}
	}

	// Step 7: parse the query and extract via and action items
	via, ok := uri.Query()["via"]
	if ok && len(via) > 0 {
		parsed.Via = via
	}
	action, ok := uri.Query()["action"]
	if ok && len(action) > 0 {
		parsed.Action = action[len(action)-1]
	}

	return &parsed, nil
}

// ParseMatrixToURL parses a matrix.to URL into the same container as ParseMatrixURI parses matrix: URIs.
func ParseMatrixToURL(uri string) (*MatrixURI, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}
	return ProcessMatrixToURL(parsed)
}

// ProcessMatrixToURL is the equivalent of ProcessMatrixURI for matrix.to URLs.
func ProcessMatrixToURL(uri *url.URL) (*MatrixURI, error) {
	if !strings.HasSuffix(uri.Hostname(), "matrix.to") {
		return nil, ErrNotMatrixTo
	}

	initialSplit := strings.SplitN(uri.Fragment, "?", 2)
	parts := strings.Split(initialSplit[0], "/")
	if len(initialSplit) > 1 {
		uri.RawQuery = initialSplit[1]
	}

	if len(parts) < 2 || len(parts) > 3 {
		return nil, ErrInvalidMatrixToPartCount
	}

	if len(parts[1]) == 0 {
		return nil, ErrEmptyMatrixToPrimaryIdentifier
	}

	var parsed MatrixURI

	parsed.Sigil1 = rune(parts[1][0])
	parsed.MXID1 = parts[1][1:]
	_, isKnown := SigilToPathSegment[parsed.Sigil1]
	if !isKnown {
		return nil, ErrInvalidMatrixToPrimaryIdentifier
	}

	if len(parts) == 3 && len(parts[2]) > 0 {
		parsed.Sigil2 = rune(parts[2][0])
		parsed.MXID2 = parts[2][1:]
		_, isKnown = SigilToPathSegment[parsed.Sigil2]
		if !isKnown {
			return nil, ErrInvalidMatrixToSecondaryIdentifier
		}
	}

	via, ok := uri.Query()["via"]
	if ok && len(via) > 0 {
		parsed.Via = via
	}
	action, ok := uri.Query()["action"]
	if ok && len(action) > 0 {
		parsed.Action = action[len(action)-1]
	}

	return &parsed, nil
}
