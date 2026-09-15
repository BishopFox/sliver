// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package exslices

func CastFunc[To, From any](source []From, conv func(From) To) []To {
	result := make([]To, len(source))
	for i, v := range source {
		result[i] = conv(v)
	}
	return result
}

func CastFuncFilter[To, From any](source []From, conv func(From) (To, bool)) []To {
	result := make([]To, 0, len(source))
	for _, v := range source {
		res, ok := conv(v)
		if ok {
			result = append(result, res)
		}
	}
	return result
}

func CastToString[To, From ~string](source []From) []To {
	result := make([]To, len(source))
	for i, v := range source {
		result[i] = To(v)
	}
	return result
}

func CastToAny[From any](source []From) []any {
	result := make([]any, len(source))
	for i, v := range source {
		result[i] = v
	}
	return result
}
