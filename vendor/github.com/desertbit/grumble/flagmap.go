/*
 * The MIT License (MIT)
 *
 * Copyright (c) 2018 Roland Singer [roland.singer@deserbit.com]
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package grumble

import (
	"fmt"
	"time"
)

// FlagMapItem holds the specific flag data.
type FlagMapItem struct {
	Value     interface{}
	IsDefault bool
}

// FlagMap holds all the parsed flag values.
type FlagMap map[string]*FlagMapItem

// copyMissingValues adds all missing values to the flags map.
func (f FlagMap) copyMissingValues(m FlagMap, copyDefault bool) {
	for k, v := range m {
		if _, ok := f[k]; !ok {
			if !copyDefault && v.IsDefault {
				continue
			}
			f[k] = v
		}
	}
}

// String returns the given flag value as string.
// Panics if not present. Flags must be registered.
func (f FlagMap) String(long string) string {
	i := f[long]
	if i == nil {
		panic(fmt.Errorf("missing flag value: flag '%s' not registered", long))
	}
	s, ok := i.Value.(string)
	if !ok {
		panic(fmt.Errorf("failed to assert flag '%s' to string", long))
	}
	return s
}

// Bool returns the given flag value as boolean.
// Panics if not present. Flags must be registered.
func (f FlagMap) Bool(long string) bool {
	i := f[long]
	if i == nil {
		panic(fmt.Errorf("missing flag value: flag '%s' not registered", long))
	}
	b, ok := i.Value.(bool)
	if !ok {
		panic(fmt.Errorf("failed to assert flag '%s' to bool", long))
	}
	return b
}

// Int returns the given flag value as int.
// Panics if not present. Flags must be registered.
func (f FlagMap) Int(long string) int {
	i := f[long]
	if i == nil {
		panic(fmt.Errorf("missing flag value: flag '%s' not registered", long))
	}
	v, ok := i.Value.(int)
	if !ok {
		panic(fmt.Errorf("failed to assert flag '%s' to int", long))
	}
	return v
}

// Int64 returns the given flag value as int64.
// Panics if not present. Flags must be registered.
func (f FlagMap) Int64(long string) int64 {
	i := f[long]
	if i == nil {
		panic(fmt.Errorf("missing flag value: flag '%s' not registered", long))
	}
	v, ok := i.Value.(int64)
	if !ok {
		panic(fmt.Errorf("failed to assert flag '%s' to int64", long))
	}
	return v
}

// Uint returns the given flag value as uint.
// Panics if not present. Flags must be registered.
func (f FlagMap) Uint(long string) uint {
	i := f[long]
	if i == nil {
		panic(fmt.Errorf("missing flag value: flag '%s' not registered", long))
	}
	v, ok := i.Value.(uint)
	if !ok {
		panic(fmt.Errorf("failed to assert flag '%s' to uint", long))
	}
	return v
}

// Uint64 returns the given flag value as uint64.
// Panics if not present. Flags must be registered.
func (f FlagMap) Uint64(long string) uint64 {
	i := f[long]
	if i == nil {
		panic(fmt.Errorf("missing flag value: flag '%s' not registered", long))
	}
	v, ok := i.Value.(uint64)
	if !ok {
		panic(fmt.Errorf("failed to assert flag '%s' to uint64", long))
	}
	return v
}

// Float64 returns the given flag value as float64.
// Panics if not present. Flags must be registered.
func (f FlagMap) Float64(long string) float64 {
	i := f[long]
	if i == nil {
		panic(fmt.Errorf("missing flag value: flag '%s' not registered", long))
	}
	v, ok := i.Value.(float64)
	if !ok {
		panic(fmt.Errorf("failed to assert flag '%s' to float64", long))
	}
	return v
}

// Duration returns the given flag value as duration.
// Panics if not present. Flags must be registered.
func (f FlagMap) Duration(long string) time.Duration {
	i := f[long]
	if i == nil {
		panic(fmt.Errorf("missing flag value: flag '%s' not registered", long))
	}
	v, ok := i.Value.(time.Duration)
	if !ok {
		panic(fmt.Errorf("failed to assert flag '%s' to duration", long))
	}
	return v
}
