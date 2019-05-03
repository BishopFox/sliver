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

package cmd

import (
	"fmt"
	"time"

	"github.com/desertbit/grumble"
)

func init() {
	App.AddCommand(&grumble.Command{
		Name: "flags",
		Help: "test flags",
		Flags: func(f *grumble.Flags) {
			f.Duration("d", "duration", time.Second, "duration test")
			f.Int("i", "int", 1, "test int")
			f.Int64("l", "int64", 2, "test int64")
			f.Uint("u", "uint", 3, "test uint")
			f.Uint64("j", "uint64", 4, "test uint64")
			f.Float64("f", "float", 5.55, "test float64")
		},
		Run: func(c *grumble.Context) error {
			fmt.Println("duration ", c.Flags.Duration("duration"))
			fmt.Println("int      ", c.Flags.Int("int"))
			fmt.Println("int64    ", c.Flags.Int64("int64"))
			fmt.Println("uint     ", c.Flags.Uint("uint"))
			fmt.Println("uint64   ", c.Flags.Uint64("uint64"))
			fmt.Println("float    ", c.Flags.Float64("float"))
			return nil
		},
	})
}
