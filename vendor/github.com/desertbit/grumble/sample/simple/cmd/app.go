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
	"strings"
	"time"

	"github.com/desertbit/grumble"
)

var App = grumble.New(&grumble.Config{
	Name:        "foo",
	Description: "An awesome foo bar",
	Flags: func(f *grumble.Flags) {
		f.String("d", "directory", "DEFAULT", "set an alternative root directory path")
		f.Bool("v", "verbose", false, "enable verbose mode")
	},
})

func init() {
	App.AddCommand(&grumble.Command{
		Name:      "daemon",
		Help:      "run the daemon",
		Aliases:   []string{"run"},
		Usage:     "daemon [OPTIONS]",
		AllowArgs: true,
		Flags: func(f *grumble.Flags) {
			f.Duration("t", "timeout", time.Second, "timeout duration")
		},
		Run: func(c *grumble.Context) error {
			fmt.Println("timeout:", c.Flags.Duration("timeout"))
			fmt.Println("directory:", c.Flags.String("directory"))
			fmt.Println("verbose:", c.Flags.Bool("verbose"))

			// Handle args.
			fmt.Println("args:")
			fmt.Println(strings.Join(c.Args, "\n"))

			return nil
		},
	})

	adminCommand := &grumble.Command{
		Name:     "admin",
		Help:     "admin tools",
		LongHelp: "super administration tools",
	}
	App.AddCommand(adminCommand)

	adminCommand.AddCommand(&grumble.Command{
		Name: "root",
		Help: "root the machine",
		Run: func(c *grumble.Context) error {
			fmt.Println(c.Flags.String("directory"))
			return fmt.Errorf("failed!")
		},
	})
}
