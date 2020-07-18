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
	"os"
	"sort"

	"github.com/desertbit/columnize"
)

func defaultInterruptHandler(a *App, count int) {
	if count >= 2 {
		a.Println("interrupted")
		os.Exit(1)
	}
	a.Println("input Ctrl-c once more to exit")
}

func defaultPrintHelp(a *App, shell bool) {
	// Columnize options.
	config := columnize.DefaultConfig()
	config.Delim = "|"
	config.Glue = "  "
	config.Prefix = "  "

	// ASCII logo.
	if a.printASCIILogo != nil {
		a.printASCIILogo(a)
	}

	// Description.
	if (len(a.config.Description)) > 0 {
		a.Printf("\n%s\n", a.config.Description)
	}

	// Usage.
	if !shell {
		a.Println()
		printHeadline(a, "Usage:")
		a.Printf("  %s [command]\n", a.config.Name)
	}

	// Group the commands by their help group if present.
	groups := make(map[string]*Commands)
	for _, c := range a.commands.list {
		key := c.HelpGroup
		if len(key) == 0 {
			key = "Commands:"
		}
		cc := groups[key]
		if cc == nil {
			cc = new(Commands)
			groups[key] = cc
		}
		cc.Add(c)
	}

	// Sort the map by the keys.
	var keys []string
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Print each commands group.
	for _, headline := range keys {
		cc := groups[headline]
		cc.Sort()

		var output []string
		for _, c := range cc.list {
			name := c.Name
			for _, a := range c.Aliases {
				name += ", " + a
			}
			output = append(output, fmt.Sprintf("%s | %v", name, c.Help))
		}

		if len(output) > 0 {
			a.Println()
			printHeadline(a, headline)
			a.Printf("%s\n", columnize.Format(output, config))
		}
	}

	// Sub Commands.
	if a.config.HelpSubCommands {
		// Check if there is at least one sub command.
		hasSubCmds := false
		for _, c := range a.commands.list {
			if len(c.commands.list) > 0 {
				hasSubCmds = true
				break
			}
		}
		if hasSubCmds {
			// Headline.
			a.Println()
			printHeadline(a, "Sub Commands:")
			hp := headlinePrinter(a)

			// Only print the first level of sub commands.
			for _, c := range a.commands.list {
				if len(c.commands.list) == 0 {
					continue
				}

				var output []string
				for _, c := range c.commands.list {
					name := c.Name
					for _, a := range c.Aliases {
						name += ", " + a
					}
					output = append(output, fmt.Sprintf("%s | %v", name, c.Help))
				}

				a.Println()
				_, _ = hp(c.Name + ":")
				a.Printf("%s\n", columnize.Format(output, config))
			}
		}
	}

	// Flags.
	if !shell {
		printFlags(a, &a.flags)
	}

	a.Println()
}

func defaultPrintCommandHelp(a *App, cmd *Command, shell bool) {
	// Columnize options.
	config := columnize.DefaultConfig()
	config.Delim = "|"
	config.Glue = "  "
	config.Prefix = "  "

	// Help description.
	if (len(cmd.LongHelp)) > 0 {
		a.Printf("\n%s\n", cmd.LongHelp)
	} else {
		a.Printf("\n%s\n", cmd.Help)
	}

	// Usage
	if len(cmd.Usage) > 0 {
		a.Println()
		printHeadline(a, "Usage:")
		a.Printf("  %s\n", cmd.Usage)
	}

	// Flags.
	printFlags(a, &cmd.flags)

	// Sub Commands.
	if len(cmd.commands.list) > 0 {
		// Only print the first level of sub commands.
		var output []string
		for _, c := range cmd.commands.list {
			name := c.Name
			for _, a := range c.Aliases {
				name += ", " + a
			}
			output = append(output, fmt.Sprintf("%s | %v", name, c.Help))
		}

		a.Println()
		printHeadline(a, "Sub Commands:")
		a.Printf("%s\n", columnize.Format(output, config))
	}

	a.Println()
}

func headlinePrinter(a *App) func(v ...interface{}) (int, error) {
	if a.config.NoColor || a.config.HelpHeadlineColor == nil {
		return a.Println
	}
	return func(v ...interface{}) (int, error) {
		return a.config.HelpHeadlineColor.Fprintln(a, v...)
	}
}

func printHeadline(a *App, s string) {
	hp := headlinePrinter(a)
	if a.config.HelpHeadlineUnderline {
		_, _ = hp(s)
		u := ""
		for i := 0; i < len(s); i++ {
			u += "="
		}
		_, _ = hp(u)
	} else {
		_, _ = hp(s)
	}
}

func printFlags(a *App, flags *Flags) {
	// Columnize options.
	config := columnize.DefaultConfig()
	config.Delim = "|"
	config.Glue = " "
	config.Prefix = "  "

	flags.sort()

	var output []string
	for _, f := range flags.list {
		long := "--" + f.Long
		short := ""
		if len(f.Short) > 0 {
			short = "-" + f.Short + ","
		}

		defaultValue := ""
		if f.Default != nil && f.HelpShowDefault && len(fmt.Sprintf("%v", f.Default)) > 0 {
			defaultValue = fmt.Sprintf("(default: %v)", f.Default)
		}

		output = append(output, fmt.Sprintf("%s | %s | %s |||| %s %s", short, long, f.HelpArgs, f.Help, defaultValue))
	}

	if len(output) > 0 {
		a.Println()
		printHeadline(a, "Flags:")
		a.Printf("%s\n", columnize.Format(output, config))
	}
}
