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

	"github.com/fatih/color"
)

const (
	defaultMultiPrompt = "... "
)

// Config specifies the application options.
type Config struct {
	// Name specifies the application name. This field is required.
	Name string

	// Description specifies the application description.
	Description string

	// Define all app command flags within this function.
	Flags func(f *Flags)

	// Persist readline historys to file if specified.
	HistoryFile string

	// Specify the max length of historys, it's 500 by default, set it to -1 to disable history.
	HistoryLimit int

	// NoColor defines if color output should be disabled.
	NoColor bool

	// Prompt defines the shell prompt.
	Prompt      string
	PromptColor *color.Color

	// MultiPrompt defines the prompt shown on multi readline.
	MultiPrompt      string
	MultiPromptColor *color.Color

	// Some more optional color settings.
	ASCIILogoColor *color.Color
	ErrorColor     *color.Color

	// Help styling.
	HelpHeadlineUnderline bool
	HelpSubCommands       bool
	HelpHeadlineColor     *color.Color
}

// SetDefaults sets the default values if not set.
func (c *Config) SetDefaults() {
	if c.HistoryLimit == 0 {
		c.HistoryLimit = 500
	}
	if c.PromptColor == nil {
		c.PromptColor = color.New(color.FgYellow, color.Bold)
	}
	if len(c.Prompt) == 0 {
		c.Prompt = c.Name + " » "
	}
	if c.MultiPromptColor == nil {
		c.MultiPromptColor = c.PromptColor
	}
	if len(c.MultiPrompt) == 0 {
		c.MultiPrompt = defaultMultiPrompt
	}
	if c.ASCIILogoColor == nil {
		c.ASCIILogoColor = c.PromptColor
	}
	if c.ErrorColor == nil {
		c.ErrorColor = color.New(color.FgRed, color.Bold)
	}
}

// Validate the required config fields.
func (c *Config) Validate() error {
	if len(c.Name) == 0 {
		return fmt.Errorf("application name is not set")
	}
	return nil
}

func (c *Config) prompt() string {
	if c.NoColor {
		return c.Prompt
	}
	return c.PromptColor.Sprint(c.Prompt)
}

func (c *Config) multiPrompt() string {
	if c.NoColor {
		return c.MultiPrompt
	}
	return c.MultiPromptColor.Sprint(c.MultiPrompt)
}
