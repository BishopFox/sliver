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

// ArgOption can be supplied to modify an argument.
type ArgOption func(*argItem)

// Min sets the minimum required number of elements for a list argument.
func Min(m int) ArgOption {
	if m < 0 {
		panic("min must be >= 0")
	}

	return func(i *argItem) {
		if !i.isList {
			panic("min option only valid for list arguments")
		}

		i.listMin = m
	}
}

// Max sets the maximum required number of elements for a list argument.
func Max(m int) ArgOption {
	if m < 1 {
		panic("max must be >= 1")
	}

	return func(i *argItem) {
		if !i.isList {
			panic("max option only valid for list arguments")
		}

		i.listMax = m
	}
}

// Default sets a default value for the argument.
// The argument becomes optional then.
func Default(v interface{}) ArgOption {
	if v == nil {
		panic("nil default value not allowed")
	}

	return func(i *argItem) {
		i.Default = v
		i.optional = true
	}
}
