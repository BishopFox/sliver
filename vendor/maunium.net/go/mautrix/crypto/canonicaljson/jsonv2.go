// Copyright (c) 2026 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build goexperiment.jsonv2

package canonicaljson

import (
	"bytes"
	"cmp"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"slices"
	"sync"
)

var canonicalOpts = []json.Options{
	jsontext.CanonicalizeRawInts(true),
	jsontext.CanonicalizeRawFloats(true),
	jsontext.AllowDuplicateNames(false),
	jsontext.AllowInvalidUTF8(false),
	jsontext.ReorderRawObjects(false), // we want UTF-8 ordering rather than UTF-16
}

func Canonicalize(input *jsontext.Value) error {
	err := input.Format(canonicalOpts...)
	if err != nil {
		return err
	}
	return reorderObjectsAndValidate(nil, *input, &[]byte{})
}

func Marshal(v any) (jsontext.Value, error) {
	data, err := json.Marshal(v, canonicalOpts...)
	if err != nil {
		return nil, err
	}
	return data, reorderObjectsAndValidate(nil, data, &[]byte{})
}

type objectMember struct {
	name   []byte
	buffer []byte
}

func (x objectMember) Compare(y objectMember) int {
	return bytes.Compare(x.name, y.name)
}

var objectMemberPool = sync.Pool{
	New: func() any {
		return new([]objectMember)
	},
}

func getObjectMembers() *[]objectMember {
	ns := objectMemberPool.Get().(*[]objectMember)
	*ns = (*ns)[:0]
	return ns
}

func putObjectMembers(ns *[]objectMember) {
	if cap(*ns) < 1<<10 {
		clear(*ns) // avoid pinning name and buffer
		objectMemberPool.Put(ns)
	}
}

// TODO replace with jsontext.Kind* after dropping Go 1.25 support
const (
	KindNull        = 'n'
	KindFalse       = 'f'
	KindTrue        = 't'
	KindString      = '"'
	KindNumber      = '0'
	KindBeginObject = '{'
	KindEndObject   = '}'
	KindBeginArray  = '['
	KindEndArray    = ']'
)

// This is based on the standard implementation of [jsontext.ReorderRawObjects]
// in https://github.com/golang/go/blob/go1.26.3/src/encoding/json/jsontext/value.go#L298-L395
// It has been adjusted to:
// * work outside the standard library
// * assume the data is already minified rather than supporting extra whitespace
// * have additional matrix-specific validation for numbers (reject floats and enforce limits)
// * most importantly: sort objects based on UTF-8 instead of UTF-16
func reorderObjectsAndValidate(d *jsontext.Decoder, buf []byte, scratch *[]byte) error {
	if d == nil {
		d = jsontext.NewDecoder(
			bytes.NewBuffer(buf),
			// This should improve performance slightly since the Format call earlier already rejected invalid data
			jsontext.AllowDuplicateNames(true),
			jsontext.AllowInvalidUTF8(true),
		)
	}
	switch tok, err := d.ReadToken(); tok.Kind() {
	case KindBeginObject:
		// Iterate and collect the name and offsets for every object member.
		members := getObjectMembers()
		defer putObjectMembers(members)
		var prevMember objectMember
		isSorted := true

		beforeBody := d.InputOffset() // offset after '{'
		for d.PeekKind() != KindEndObject {
			beforeName := d.InputOffset()
			name, err := d.ReadValue()
			if err != nil {
				return fmt.Errorf("failed to read key: %w", err)
			}
			if !bytes.ContainsRune(name, '\\') {
				name = name[1 : len(name)-1] // unquote without copying if possible
			} else {
				name, err = jsontext.AppendUnquote(nil, name)
				if err != nil {
					return fmt.Errorf("failed to unquote key: %w", err)
				}
			}
			err = reorderObjectsAndValidate(d, buf, scratch)
			if err != nil {
				return fmt.Errorf("failed to reorder %s: %w", name, err)
			}
			afterValue := d.InputOffset()

			currMember := objectMember{name, buf[beforeName:afterValue]}
			if isSorted && len(*members) > 0 {
				isSorted = objectMember.Compare(prevMember, currMember) < 0
			}
			*members = append(*members, currMember)
			prevMember = currMember
		}
		afterBody := d.InputOffset() // offset before '}'
		_, err = d.ReadToken()
		if err != nil {
			return fmt.Errorf("failed to read end of object: %w", err)
		}

		// Sort the members; return early if it's already sorted.
		if isSorted {
			return nil
		}
		firstBufferBeforeSorting := (*members)[0].buffer
		slices.SortFunc(*members, objectMember.Compare)

		// Append the reordered members to a new buffer,
		// then copy the reordered members back over the original members.
		// Avoid swapping in place since each member may be a different size
		// where moving a member over a smaller member may corrupt the data
		// for subsequent members before they have been moved.
		//
		// The following invariant must hold:
		//	sum([m.after-m.before for m in members]) == afterBody-beforeBody
		sorted := (*scratch)[:0]
		for i, member := range *members {
			switch {
			case i == 0 && &member.buffer[0] != &firstBufferBeforeSorting[0]:
				// First member after sorting is not the first member before sorting, cut off the leading comma
				if member.buffer[0] != ',' {
					return fmt.Errorf("expected newly sorted first member buffer to start with a comma")
				}
				sorted = append(sorted, member.buffer[1:]...)
			case i != 0 && &member.buffer[0] == &firstBufferBeforeSorting[0]:
				// Later member after sorting is the first member before sorting, add leading comma
				if member.buffer[0] == ',' {
					return fmt.Errorf("expected newly sorted later member buffer to not start with a comma")
				}
				sorted = append(sorted, ',')
				sorted = append(sorted, member.buffer...)
			default:
				sorted = append(sorted, member.buffer...)
			}
		}
		if int(afterBody-beforeBody) != len(sorted) {
			return fmt.Errorf("BUG: length invariant violated")
		}
		copy(buf[beforeBody:afterBody], sorted)

		// Update scratch buffer to the largest amount ever used.
		if len(sorted) > len(*scratch) {
			*scratch = sorted
		}
		return nil
	case KindBeginArray:
		for d.PeekKind() != KindEndArray {
			err = reorderObjectsAndValidate(d, buf, scratch)
			if err != nil {
				return err
			}
		}
		_, err = d.ReadToken()
		return err
	case KindNumber:
		str := tok.String()
		if str == "-0" {
			return fmt.Errorf("invalid number: -0")
		} else if str == "0" {
			return nil
		} else if len(str) > 17 {
			return fmt.Errorf("too long number: %q", str)
		}
		var val uint64
		firstDigitIdx := 0
		for i, bt := range []byte(str) {
			if i == 0 && bt == '-' {
				firstDigitIdx = 1
				continue
			} else if (bt >= '1' && bt <= '9') || (bt == '0' && i != firstDigitIdx) {
				// ok
			} else {
				return fmt.Errorf("invalid character %c in number: %q", bt, str)
			}
			val = val*10 + uint64(bt-'0')
		}
		// val is always positive since we just ignore the leading -
		if val >= 1<<53 {
			return fmt.Errorf("number too large: %q", str)
		}
		return nil
	case KindNull, KindFalse, KindTrue, KindString:
		return err
	default:
		// This probably can't happen
		return cmp.Or(err, fmt.Errorf("unexpected token: %s", tok))
	}
}

// Deprecated: Use the new Canonicalize or Marshal functions
func CanonicalJSONAssumeValid(input []byte) []byte {
	out, _ := CanonicalJSON(input)
	return out
}

// Deprecated: Use the new Canonicalize or Marshal functions
func CanonicalJSON(input []byte) ([]byte, error) {
	out := jsontext.Value(bytes.Clone(input))
	err := Canonicalize(&out)
	return out, err
}
