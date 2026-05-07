// Copyright (c) 2022 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package exerrors

import (
	"errors"
	"fmt"
)

type DualError struct {
	High error
	Low  error
}

func NewDualError(high, low error) DualError {
	return DualError{high, low}
}

func (err DualError) Is(other error) bool {
	return errors.Is(other, err.High) || errors.Is(other, err.Low)
}

func (err DualError) Unwrap() error {
	return err.Low
}

func (err DualError) Error() string {
	return fmt.Sprintf("%v: %v", err.High, err.Low)
}
