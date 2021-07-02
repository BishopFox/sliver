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

// ArgMapItem holds the specific arg data.
type ArgMapItem struct {
	Value     interface{}
	IsDefault bool
}

// ArgMap holds all the parsed arg values.
type ArgMap map[string]*ArgMapItem

// String returns the given arg value as string.
// Panics if not present. Args must be registered.
func (a ArgMap) String(name string) string {
	i := a[name]
	if i == nil {
		panic(fmt.Errorf("missing argument value: arg '%s' not registered", name))
	}
	s, ok := i.Value.(string)
	if !ok {
		panic(fmt.Errorf("failed to assert argument '%s' to string", name))
	}
	return s
}

// StringList returns the given arg value as string slice.
// Panics if not present. Args must be registered.
// If optional and not provided, nil is returned.
func (a ArgMap) StringList(long string) []string {
	i := a[long]
	if i == nil {
		panic(fmt.Errorf("missing arg value: arg '%s' not registered", long))
	}
	if i.Value == nil {
		return nil
	}
	s, ok := i.Value.([]string)
	if !ok {
		panic(fmt.Errorf("failed to assert arg '%s' to string list", long))
	}
	return s
}

// Bool returns the given arg value as bool.
// Panics if not present. Args must be registered.
func (a ArgMap) Bool(long string) bool {
	i := a[long]
	if i == nil {
		panic(fmt.Errorf("missing arg value: arg '%s' not registered", long))
	}
	b, ok := i.Value.(bool)
	if !ok {
		panic(fmt.Errorf("failed to assert arg '%s' to bool", long))
	}
	return b
}

// BoolList returns the given arg value as bool slice.
// Panics if not present. Args must be registered.
func (a ArgMap) BoolList(long string) []bool {
	i := a[long]
	if i == nil {
		panic(fmt.Errorf("missing arg value: arg '%s' not registered", long))
	}
	if i.Value == nil {
		return nil
	}
	b, ok := i.Value.([]bool)
	if !ok {
		panic(fmt.Errorf("failed to assert arg '%s' to bool list", long))
	}
	return b
}

// Int returns the given arg value as int.
// Panics if not present. Args must be registered.
func (a ArgMap) Int(long string) int {
	i := a[long]
	if i == nil {
		panic(fmt.Errorf("missing arg value: arg '%s' not registered", long))
	}
	v, ok := i.Value.(int)
	if !ok {
		panic(fmt.Errorf("failed to assert arg '%s' to int", long))
	}
	return v
}

// IntList returns the given arg value as int slice.
// Panics if not present. Args must be registered.
func (a ArgMap) IntList(long string) []int {
	i := a[long]
	if i == nil {
		panic(fmt.Errorf("missing arg value: arg '%s' not registered", long))
	}
	if i.Value == nil {
		return nil
	}
	v, ok := i.Value.([]int)
	if !ok {
		panic(fmt.Errorf("failed to assert arg '%s' to int list", long))
	}
	return v
}

// Int64 returns the given arg value as int64.
// Panics if not present. Args must be registered.
func (a ArgMap) Int64(long string) int64 {
	i := a[long]
	if i == nil {
		panic(fmt.Errorf("missing arg value: arg '%s' not registered", long))
	}
	v, ok := i.Value.(int64)
	if !ok {
		panic(fmt.Errorf("failed to assert arg '%s' to int64", long))
	}
	return v
}

// Int64List returns the given arg value as int64.
// Panics if not present. Args must be registered.
func (a ArgMap) Int64List(long string) []int64 {
	i := a[long]
	if i == nil {
		panic(fmt.Errorf("missing arg value: arg '%s' not registered", long))
	}
	if i.Value == nil {
		return nil
	}
	v, ok := i.Value.([]int64)
	if !ok {
		panic(fmt.Errorf("failed to assert arg '%s' to int64 list", long))
	}
	return v
}

// Uint returns the given arg value as uint.
// Panics if not present. Args must be registered.
func (a ArgMap) Uint(long string) uint {
	i := a[long]
	if i == nil {
		panic(fmt.Errorf("missing arg value: arg '%s' not registered", long))
	}
	v, ok := i.Value.(uint)
	if !ok {
		panic(fmt.Errorf("failed to assert arg '%s' to uint", long))
	}
	return v
}

// UintList returns the given arg value as uint.
// Panics if not present. Args must be registered.
func (a ArgMap) UintList(long string) []uint {
	i := a[long]
	if i == nil {
		panic(fmt.Errorf("missing arg value: arg '%s' not registered", long))
	}
	if i.Value == nil {
		return nil
	}
	v, ok := i.Value.([]uint)
	if !ok {
		panic(fmt.Errorf("failed to assert arg '%s' to uint list", long))
	}
	return v
}

// Uint64 returns the given arg value as uint64.
// Panics if not present. Args must be registered.
func (a ArgMap) Uint64(long string) uint64 {
	i := a[long]
	if i == nil {
		panic(fmt.Errorf("missing arg value: arg '%s' not registered", long))
	}
	v, ok := i.Value.(uint64)
	if !ok {
		panic(fmt.Errorf("failed to assert arg '%s' to uint64", long))
	}
	return v
}

// Uint64List returns the given arg value as uint64.
// Panics if not present. Args must be registered.
func (a ArgMap) Uint64List(long string) []uint64 {
	i := a[long]
	if i == nil {
		panic(fmt.Errorf("missing arg value: arg '%s' not registered", long))
	}
	if i.Value == nil {
		return nil
	}
	v, ok := i.Value.([]uint64)
	if !ok {
		panic(fmt.Errorf("failed to assert arg '%s' to uint64 list", long))
	}
	return v
}

// Float64 returns the given arg value as float64.
// Panics if not present. Args must be registered.
func (a ArgMap) Float64(long string) float64 {
	i := a[long]
	if i == nil {
		panic(fmt.Errorf("missing arg value: arg '%s' not registered", long))
	}
	v, ok := i.Value.(float64)
	if !ok {
		panic(fmt.Errorf("failed to assert arg '%s' to float64", long))
	}
	return v
}

// Float64List returns the given arg value as float64.
// Panics if not present. Args must be registered.
func (a ArgMap) Float64List(long string) []float64 {
	i := a[long]
	if i == nil {
		panic(fmt.Errorf("missing arg value: arg '%s' not registered", long))
	}
	if i.Value == nil {
		return nil
	}
	v, ok := i.Value.([]float64)
	if !ok {
		panic(fmt.Errorf("failed to assert arg '%s' to float64 list", long))
	}
	return v
}

// Duration returns the given arg value as duration.
// Panics if not present. Args must be registered.
func (a ArgMap) Duration(long string) time.Duration {
	i := a[long]
	if i == nil {
		panic(fmt.Errorf("missing arg value: arg '%s' not registered", long))
	}
	v, ok := i.Value.(time.Duration)
	if !ok {
		panic(fmt.Errorf("failed to assert arg '%s' to duration", long))
	}
	return v
}

// DurationList returns the given arg value as duration.
// Panics if not present. Args must be registered.
func (a ArgMap) DurationList(long string) []time.Duration {
	i := a[long]
	if i == nil {
		panic(fmt.Errorf("missing arg value: arg '%s' not registered", long))
	}
	if i.Value == nil {
		return nil
	}
	v, ok := i.Value.([]time.Duration)
	if !ok {
		panic(fmt.Errorf("failed to assert arg '%s' to duration list", long))
	}
	return v
}
