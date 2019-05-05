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
	"github.com/desertbit/grumble"
)

func init() {
	promptCommand := &grumble.Command{
		Name: "prompt",
		Help: "set a custom prompt",
	}
	App.AddCommand(promptCommand)

	promptCommand.AddCommand(&grumble.Command{
		Name: "set",
		Help: "set a custom prompt",
		Run: func(c *grumble.Context) error {
			c.App.SetPrompt("CUSTOM PROMPT >> ")
			return nil
		},
	})

	promptCommand.AddCommand(&grumble.Command{
		Name: "reset",
		Help: "reset to default prompt",
		Run: func(c *grumble.Context) error {
			c.App.SetDefaultPrompt()
			return nil
		},
	})
}
