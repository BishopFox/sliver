// Copyright (c) 2021 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package id

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// UserID represents a Matrix user ID.
// https://matrix.org/docs/spec/appendices#user-identifiers
type UserID string

const UserIDMaxLength = 255

func NewUserID(localpart, homeserver string) UserID {
	return UserID(fmt.Sprintf("@%s:%s", localpart, homeserver))
}

func NewEncodedUserID(localpart, homeserver string) UserID {
	return NewUserID(EncodeUserLocalpart(localpart), homeserver)
}

var (
	ErrInvalidUserID          = errors.New("is not a valid user ID")
	ErrNoncompliantLocalpart  = errors.New("contains characters that are not allowed")
	ErrUserIDTooLong          = errors.New("the given user ID is longer than 255 characters")
	ErrEmptyLocalpart         = errors.New("empty localparts are not allowed")
	ErrNoncompliantServerPart = errors.New("is not a valid server name")
)

// ParseCommonIdentifier parses a common identifier according to https://spec.matrix.org/v1.9/appendices/#common-identifier-format
func ParseCommonIdentifier[Stringish ~string](identifier Stringish) (sigil byte, localpart, homeserver string) {
	if len(identifier) == 0 {
		return
	}
	sigil = identifier[0]
	strIdentifier := string(identifier)
	colonIdx := strings.IndexByte(strIdentifier, ':')
	if colonIdx > 0 {
		localpart = strIdentifier[1:colonIdx]
		homeserver = strIdentifier[colonIdx+1:]
	} else {
		localpart = strIdentifier[1:]
	}
	return
}

// Parse parses the user ID into the localpart and server name.
//
// Note that this only enforces very basic user ID formatting requirements: user IDs start with
// a @, and contain a : after the @. If you want to enforce localpart validity, see the
// ParseAndValidate and ValidateUserLocalpart functions.
func (userID UserID) Parse() (localpart, homeserver string, err error) {
	var sigil byte
	sigil, localpart, homeserver = ParseCommonIdentifier(userID)
	if sigil != '@' || homeserver == "" {
		err = fmt.Errorf("'%s' %w", userID, ErrInvalidUserID)
	}
	return
}

func (userID UserID) Localpart() string {
	localpart, _, _ := userID.Parse()
	return localpart
}

func (userID UserID) Homeserver() string {
	_, homeserver, _ := userID.Parse()
	return homeserver
}

// URI returns the user ID as a MatrixURI struct, which can then be stringified into a matrix: URI or a matrix.to URL.
//
// This does not parse or validate the user ID. Use the ParseAndValidate method if you want to ensure the user ID is valid first.
func (userID UserID) URI() *MatrixURI {
	if userID == "" {
		return nil
	}
	return &MatrixURI{
		Sigil1: '@',
		MXID1:  string(userID)[1:],
	}
}

var ValidLocalpartRegex = regexp.MustCompile("^[0-9a-z-.=_/+]+$")

// ValidateUserLocalpart validates a Matrix user ID localpart using the grammar
// in https://matrix.org/docs/spec/appendices#user-identifier
func ValidateUserLocalpart(localpart string) error {
	if len(localpart) == 0 {
		return ErrEmptyLocalpart
	} else if !ValidLocalpartRegex.MatchString(localpart) {
		return fmt.Errorf("'%s' %w", localpart, ErrNoncompliantLocalpart)
	}
	return nil
}

// ParseAndValidateStrict is a stricter version of ParseAndValidateRelaxed that checks the localpart to only allow non-historical localparts.
// This should be used with care: there are real users still using historical localparts.
func (userID UserID) ParseAndValidateStrict() (localpart, homeserver string, err error) {
	localpart, homeserver, err = userID.ParseAndValidateRelaxed()
	if err == nil {
		err = ValidateUserLocalpart(localpart)
	}
	return
}

// ParseAndValidateRelaxed parses the user ID into the localpart and server name like Parse,
// and also validates that the user ID is not too long and that the server name is valid.
func (userID UserID) ParseAndValidateRelaxed() (localpart, homeserver string, err error) {
	if len(userID) > UserIDMaxLength {
		err = ErrUserIDTooLong
		return
	}
	localpart, homeserver, err = userID.Parse()
	if err == nil && !ValidateServerName(homeserver) {
		err = fmt.Errorf("%q %q", homeserver, ErrNoncompliantServerPart)
	}
	return
}

func (userID UserID) ParseAndDecode() (localpart, homeserver string, err error) {
	localpart, homeserver, err = userID.ParseAndValidateStrict()
	if err == nil {
		localpart, err = DecodeUserLocalpart(localpart)
	}
	return
}

func (userID UserID) String() string {
	return string(userID)
}

const lowerhex = "0123456789abcdef"

// encode the given byte using quoted-printable encoding (e.g "=2f")
// and writes it to the buffer
// See https://golang.org/src/mime/quotedprintable/writer.go
func encode(buf *bytes.Buffer, b byte) {
	buf.WriteByte('=')
	buf.WriteByte(lowerhex[b>>4])
	buf.WriteByte(lowerhex[b&0x0f])
}

// escape the given alpha character and writes it to the buffer
func escape(buf *bytes.Buffer, b byte) {
	buf.WriteByte('_')
	if b == '_' {
		buf.WriteByte('_') // another _
	} else {
		buf.WriteByte(b + 0x20) // ASCII shift A-Z to a-z
	}
}

func shouldEncode(b byte) bool {
	return b != '-' && b != '.' && b != '_' && b != '+' && !(b >= '0' && b <= '9') && !(b >= 'a' && b <= 'z') && !(b >= 'A' && b <= 'Z')
}

func shouldEscape(b byte) bool {
	return (b >= 'A' && b <= 'Z') || b == '_'
}

func isValidByte(b byte) bool {
	return isValidEscapedChar(b) || (b >= '0' && b <= '9') || b == '.' || b == '=' || b == '-' || b == '+'
}

func isValidEscapedChar(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z')
}

// EncodeUserLocalpart encodes the given string into Matrix-compliant user ID localpart form.
// See https://spec.matrix.org/v1.2/appendices/#mapping-from-other-character-sets
//
// This returns a string with only the characters "a-z0-9._=-". The uppercase range A-Z
// are encoded using leading underscores ("_"). Characters outside the aforementioned ranges
// (including literal underscores ("_") and equals ("=")) are encoded as UTF8 code points (NOT NCRs)
// and converted to lower-case hex with a leading "=". For example:
//
//	Alph@Bet_50up  => _alph=40_bet=5f50up
func EncodeUserLocalpart(str string) string {
	strBytes := []byte(str)
	var outputBuffer bytes.Buffer
	for _, b := range strBytes {
		if shouldEncode(b) {
			encode(&outputBuffer, b)
		} else if shouldEscape(b) {
			escape(&outputBuffer, b)
		} else {
			outputBuffer.WriteByte(b)
		}
	}
	return outputBuffer.String()
}

// DecodeUserLocalpart decodes the given string back into the original input string.
// Returns an error if the given string is not a valid user ID localpart encoding.
// See https://spec.matrix.org/v1.2/appendices/#mapping-from-other-character-sets
//
// This decodes quoted-printable bytes back into UTF8, and unescapes casing. For
// example:
//
//	_alph=40_bet=5f50up  =>  Alph@Bet_50up
//
// Returns an error if the input string contains characters outside the
// range "a-z0-9._=-", has an invalid quote-printable byte (e.g. not hex), or has
// an invalid _ escaped byte (e.g. "_5").
func DecodeUserLocalpart(str string) (string, error) {
	strBytes := []byte(str)
	var outputBuffer bytes.Buffer
	for i := 0; i < len(strBytes); i++ {
		b := strBytes[i]
		if !isValidByte(b) {
			return "", fmt.Errorf("invalid encoded byte at position %d: %c", i, b)
		}

		if b == '_' { // next byte is a-z and should be upper-case or is another _ and should be a literal _
			if i+1 >= len(strBytes) {
				return "", fmt.Errorf("unexpected end of string after underscore at %d", i)
			}
			if !isValidEscapedChar(strBytes[i+1]) { // invalid escaping
				return "", fmt.Errorf("unexpected byte %c after underscore at %d", strBytes[i+1], i)
			}
			if strBytes[i+1] == '_' {
				outputBuffer.WriteByte('_')
			} else {
				outputBuffer.WriteByte(strBytes[i+1] - 0x20) // ASCII shift a-z to A-Z
			}
			i++ // skip next byte since we just handled it
		} else if b == '=' { // next 2 bytes are hex and should be buffered ready to be read as utf8
			if i+2 >= len(strBytes) {
				return "", fmt.Errorf("unexpected end of string after equals sign at %d", i)
			}
			dst := make([]byte, 1)
			_, err := hex.Decode(dst, strBytes[i+1:i+3])
			if err != nil {
				return "", err
			}
			outputBuffer.WriteByte(dst[0])
			i += 2 // skip next 2 bytes since we just handled it
		} else { // pass through
			outputBuffer.WriteByte(b)
		}
	}
	return outputBuffer.String(), nil
}
