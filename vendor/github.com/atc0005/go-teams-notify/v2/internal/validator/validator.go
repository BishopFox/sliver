// Copyright 2022 Adam Chalkley
//
// https://github.com/atc0005/go-teams-notify
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package validator

import (
	"fmt"

	goteamsnotify "github.com/atc0005/go-teams-notify/v2"
)

// Validater is the interface shared by all supported types which provide
// validation of their fields.
type Validater interface {
	Validate() error
}

// Validator is used to perform validation of given values. Each validation
// method for this type is designed to exit early in order to preserve any
// prior validation failure. If a previous validation check failure occurred,
// the most recent validation check result will
//
// After performing a validation check, the caller is responsible for checking
// the result to determine if further validation checks should be performed.
//
// Heavily inspired by: https://stackoverflow.com/a/23960293/903870
type Validator struct {
	err error
}

// hasNilValues is a helper function used to determine whether any items in
// the given collection are nil.
func hasNilValues(items []interface{}) bool {
	for _, item := range items {
		if item == nil {
			return true
		}
	}
	return false
}

// SelfValidate asserts that each given item can self-validate.
//
// A true value is returned if the validation step passed. A false value is
// returned if this or a prior validation step failed.
func (v *Validator) SelfValidate(items ...Validater) bool {
	if v.err != nil {
		return false
	}
	for _, item := range items {
		if err := item.Validate(); err != nil {
			v.err = err
			return false
		}
	}
	return true
}

// SelfValidateIfXEqualsY asserts that each given item can self-validate if
// value x is equal to y.
//
// A true value is returned if the validation step passed. A false value is
// returned false if this or a prior validation step failed.
func (v *Validator) SelfValidateIfXEqualsY(x string, y string, items ...Validater) bool {
	if v.err != nil {
		return false
	}

	if x == y {
		v.SelfValidate(items...)
	}

	return true
}

// FieldHasSpecificValue asserts that fieldVal is reqVal. fieldValDesc
// describes the field value being validated (e.g., "Type") and typeDesc
// describes the specific struct or value type whose field we are validating
// (e.g., "Element").
//
// A true value is returned if the validation step passed. A false value is
// returned if this or a prior validation step failed.
func (v *Validator) FieldHasSpecificValue(
	fieldVal string,
	fieldValDesc string,
	reqVal string,
	typeDesc string,
	baseErr error,
) bool {

	switch {
	case v.err != nil:
		return false

	case fieldVal != reqVal:
		v.err = fmt.Errorf(
			// "required %s is empty for %s: %w",
			// "invalid card type %q; expected %q: %w",
			"invalid %s %q for %s; expected %q: %w",
			fieldValDesc,
			fieldVal,
			typeDesc,
			reqVal,
			baseErr,
		)
		return false

	default:
		return true
	}
}

// FieldHasSpecificValueIfFieldNotEmpty asserts that fieldVal is reqVal unless
// fieldVal is empty. fieldValDesc describes the field value being validated
// (e.g., "Type") and typeDesc describes the specific struct or value type
// whose field we are validating (e.g., "Element").
//
// A true value is returned if the validation step passed. A false value is
// returned if this or a prior validation step failed.
func (v *Validator) FieldHasSpecificValueIfFieldNotEmpty(
	fieldVal string,
	fieldValDesc string,
	reqVal string,
	typeDesc string,
	baseErr error,
) bool {

	switch {
	case v.err != nil:
		return false

	case fieldVal != "":
		return v.FieldHasSpecificValue(
			fieldVal,
			fieldValDesc,
			reqVal,
			typeDesc,
			baseErr,
		)

	default:
		return true
	}
}

// NotEmptyValue asserts that fieldVal is not empty. fieldValDesc describes
// the field value being validated (e.g., "Type") and typeDesc describes the
// specific struct or value type whose field we are validating (e.g.,
// "Element").
//
// A true value is returned if the validation step passed. A false value is
// returned if this or a prior validation step failed.
func (v *Validator) NotEmptyValue(fieldVal string, fieldValDesc string, typeDesc string, baseErr error) bool {
	if v.err != nil {
		return false
	}
	if fieldVal == "" {
		v.err = fmt.Errorf(
			"required %s is empty for %s: %w",
			fieldValDesc,
			typeDesc,
			baseErr,
		)
		return false
	}
	return true
}

// InList reports whether fieldVal is in validVals. fieldValDesc describes the
// field value being validated (e.g., "Type") and typeDesc describes the
// specific struct or value type whose field we are validating (e.g.,
// "Element").
//
// A true value is returned if fieldVal is is in validVals.
//
// A false value is returned if any of:
//   - a prior validation step failed
//   - fieldVal is empty
//   - fieldVal is non-empty and not in validVals
//   - the validVals collection to compare against is empty
func (v *Validator) InList(fieldVal string, fieldValDesc string, typeDesc string, validVals []string, baseErr error) bool {
	switch {
	case v.err != nil:
		return false

	case fieldVal == "":
		return false

	case !goteamsnotify.InList(fieldVal, validVals, false):
		switch {
		case len(validVals) == 0 && baseErr != nil:
			v.err = fmt.Errorf(
				"invalid %s %q for %s; empty list of valid values: %w",
				fieldValDesc,
				fieldVal,
				typeDesc,
				baseErr,
			)
		case len(validVals) == 0:
			v.err = fmt.Errorf(
				"invalid %s %q for %s; no known valid values",
				fieldValDesc,
				fieldVal,
				typeDesc,
			)
		case baseErr != nil:
			v.err = fmt.Errorf(
				"invalid %s %q for %s; expected one of %v: %w",
				fieldValDesc,
				fieldVal,
				typeDesc,
				validVals,
				baseErr,
			)
		default:
			v.err = fmt.Errorf(
				"invalid %s %q for %s; expected one of %v",
				fieldValDesc,
				fieldVal,
				typeDesc,
				validVals,
			)
		}

		return false

	// Validation is good.
	default:
		return true
	}
}

// InListIfFieldValNotEmpty reports whether fieldVal is in validVals if
// fieldVal is not empty. fieldValDesc describes the field value being
// validated (e.g., "Type") and typeDesc describes the specific struct or
// value type whose field we are validating (e.g., "Element").
//
// A true value is returned if fieldVal is empty or is in validVals.
//
// A false value is returned if any of:
//   - a prior validation step failed
//   - fieldVal is not empty and is not in validVals
//   - the validVals collection to compare against is empty
func (v *Validator) InListIfFieldValNotEmpty(fieldVal string, fieldValDesc string, typeDesc string, validVals []string, baseErr error) bool {
	switch {
	case v.err != nil:
		return false

	case fieldVal != "" && !goteamsnotify.InList(fieldVal, validVals, false):
		switch {
		case len(validVals) == 0 && baseErr != nil:
			v.err = fmt.Errorf(
				"invalid %s %q for %s; empty list of valid values: %w",
				fieldValDesc,
				fieldVal,
				typeDesc,
				baseErr,
			)
		case len(validVals) == 0:
			v.err = fmt.Errorf(
				"invalid %s %q for %s; no known valid values",
				fieldValDesc,
				fieldVal,
				typeDesc,
			)
		case baseErr != nil:
			v.err = fmt.Errorf(
				"invalid %s %q for %s; expected one of %v: %w",
				fieldValDesc,
				fieldVal,
				typeDesc,
				validVals,
				baseErr,
			)
		default:
			v.err = fmt.Errorf(
				"invalid %s %q for %s; expected one of %v",
				fieldValDesc,
				fieldVal,
				typeDesc,
				validVals,
			)
		}

		return false

	// Validation is good.
	default:
		return true
	}
}

// FieldInListIfTypeValIs reports whether fieldVal is in validVals if fieldVal
// is not empty. fieldValDesc describes the field value being validated (e.g.,
// "Type") and typeDesc describes the specific struct or value type whose
// field we are validating (e.g., "Element").
//
// A true value is returned if fieldVal is empty or is in validVals. A false
// value is returned if a prior validation step failed or if fieldVal is not
// empty and is not in validVals.
// func (v *Validator) FieldInListIfTypeValIs(
// 	fieldVal string,
// 	fieldDesc string,
// 	typeVal string,
// 	typeDesc string,
// 	validVals []string,
// 	baseErr error,
// ) bool {
// 	switch {
// 	case v.err != nil:
// 		return false
//
// 	case fieldVal != "" && !goteamsnotify.InList(fieldVal, validVals, false):
// 		v.err = fmt.Errorf(
// 			"invalid %s %q for %s; expected one of %v",
// 			fieldValDesc,
// 			fieldVal,
// 			typeDesc,
// 			validVals,
// 		)
//
// 		if baseErr != nil {
// 			v.err = fmt.Errorf(
// 				"invalid %s %q for %s; expected one of %v: %w",
// 				fieldValDesc,
// 				fieldVal,
// 				typeDesc,
// 				validVals,
// 				baseErr,
// 			)
// 		}
//
// 		return false
//
// 	// Validation is good.
// 	default:
// 		return true
// 	}
// }

// NotEmptyCollection asserts that the specified items collection is not
// empty. fieldValueDesc describes the field for this collection being
// validated (e.g., "Facts") and typeDesc describes the specific struct or
// value type whose field we are validating (e.g., "Element").
//
// A true value is returned if the collection is not empty. A false value is
// returned if a prior validation step failed or if the items collection is
// empty.
func (v *Validator) NotEmptyCollection(fieldValueDesc string, typeDesc string, baseErr error, items ...interface{}) bool {
	if v.err != nil {
		return false
	}
	if len(items) == 0 {
		switch {
		case baseErr != nil:
			v.err = fmt.Errorf(
				"required %s collection is empty for %s: %w",
				fieldValueDesc,
				typeDesc,
				baseErr,
			)
		default:
			v.err = fmt.Errorf(
				"required %s collection is empty for %s",
				fieldValueDesc,
				typeDesc,
			)
		}

		return false
	}
	return true
}

// NoNilValuesInCollection asserts that the specified items collection does
// not contain any nil values. fieldValueDesc describes the field for this
// collection being validated (e.g., "Facts") and typeDesc describes the
// specific struct or value type whose field we are validating (e.g.,
// "Element").
//
// A true value is returned if the collection does not contain any nil values
// (even if the collection itself has no values). A false value is returned if
// a prior validation step failed or if any items in the collection are nil.
func (v *Validator) NoNilValuesInCollection(fieldValueDesc string, typeDesc string, baseErr error, items ...interface{}) bool {
	if v.err != nil {
		return false
	}

	switch {
	case hasNilValues(items):
		switch {
		case baseErr != nil:
			v.err = fmt.Errorf(
				"required %s collection contains nil values for %s: %w",
				fieldValueDesc,
				typeDesc,
				baseErr,
			)
		default:
			v.err = fmt.Errorf(
				"required %s collection contains nil values for for %s",
				fieldValueDesc,
				typeDesc,
			)
		}

		return false

	default:
		return true
	}
}

// NotEmptyCollectionIfFieldValNotEmpty asserts that the specified items
// collection is not empty if fieldVal is not empty. fieldValueDesc describes
// the field for this collection being validated (e.g., "Facts") and typeDesc
// describes the specific struct or value type whose field we are validating
// (e.g., "Element").
//
// A true value is returned if the collection is not empty. A false value is
// returned if a prior validation step failed or if the items collection is
// empty.
func (v *Validator) NotEmptyCollectionIfFieldValNotEmpty(
	fieldVal string,
	fieldValueDesc string,
	typeDesc string,
	baseErr error,
	items ...interface{},
) bool {

	switch {
	case v.err != nil:
		return false

	case fieldVal != "" && len(items) == 0:
		switch {
		case baseErr != nil:
			v.err = fmt.Errorf(
				"required %s collection is empty for %s: %w",
				fieldValueDesc,
				typeDesc,
				baseErr,
			)
		default:
			v.err = fmt.Errorf(
				"required %s collection is empty for %s",
				fieldValueDesc,
				typeDesc,
			)
		}

		return false

	default:
		return true
	}
}

// SuccessfulFuncCall accepts fn, a function that returns an error. fn is
// called in order to determine validation results.
//
// A true value is returned if fn was successful. A false value is returned if
// a prior validation step failed or if fn returned an error.
func (v *Validator) SuccessfulFuncCall(fn func() error) bool {
	if v.err != nil {
		return false
	}

	if err := fn(); err != nil {
		v.err = err
		return false
	}

	return true
}

// IsValid indicates whether validation checks performed thus far have all
// passed.
func (v *Validator) IsValid() bool {
	return v.err != nil
}

// Error returns the error string from the last recorded validation error.
func (v *Validator) Error() string {
	return v.err.Error()
}

// Err returns the last recorded validation error.
func (v *Validator) Err() error {
	return v.err
}
