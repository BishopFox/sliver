// Copyright (c) 2022 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package id

import (
	"fmt"
	"strings"
)

// TrustState determines how trusted a device is.
type TrustState int

const (
	TrustStateBlacklisted          TrustState = -100
	TrustStateUnset                TrustState = 0
	TrustStateUnknownDevice        TrustState = 10
	TrustStateForwarded            TrustState = 20
	TrustStateCrossSignedUntrusted TrustState = 50
	TrustStateCrossSignedTOFU      TrustState = 100
	TrustStateCrossSignedVerified  TrustState = 200
	TrustStateVerified             TrustState = 300
	TrustStateInvalid              TrustState = (1 << 31) - 1
)

func (ts *TrustState) UnmarshalText(data []byte) error {
	strData := string(data)
	state := ParseTrustState(strData)
	if state == TrustStateInvalid {
		return fmt.Errorf("invalid trust state %q", strData)
	}
	*ts = state
	return nil
}

func (ts *TrustState) MarshalText() ([]byte, error) {
	return []byte(ts.String()), nil
}

func ParseTrustState(val string) TrustState {
	switch strings.ToLower(val) {
	case "blacklisted":
		return TrustStateBlacklisted
	case "unverified":
		return TrustStateUnset
	case "cross-signed-untrusted":
		return TrustStateCrossSignedUntrusted
	case "unknown-device":
		return TrustStateUnknownDevice
	case "forwarded":
		return TrustStateForwarded
	case "cross-signed-tofu", "cross-signed":
		return TrustStateCrossSignedTOFU
	case "cross-signed-verified", "cross-signed-trusted":
		return TrustStateCrossSignedVerified
	case "verified":
		return TrustStateVerified
	default:
		return TrustStateInvalid
	}
}

func (ts TrustState) String() string {
	switch ts {
	case TrustStateBlacklisted:
		return "blacklisted"
	case TrustStateUnset:
		return "unverified"
	case TrustStateCrossSignedUntrusted:
		return "cross-signed-untrusted"
	case TrustStateUnknownDevice:
		return "unknown-device"
	case TrustStateForwarded:
		return "forwarded"
	case TrustStateCrossSignedTOFU:
		return "cross-signed-tofu"
	case TrustStateCrossSignedVerified:
		return "cross-signed-verified"
	case TrustStateVerified:
		return "verified"
	default:
		return "invalid"
	}
}
