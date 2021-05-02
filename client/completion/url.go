package completion

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"strings"

	"github.com/bishopfox/sliver/client/constants"
	"github.com/maxlandon/gonsole"
	"github.com/maxlandon/readline"
)

var (
	proxyUpdateShemes = []string{
		"https",
		"http",
	}
	stagerShemes = []string{
		"tcp",
		"http",
		"https",
	}
)

// urlCompleter - A little wrapper used to more a more menu/context/command sensitive
// URL completion system, with server and/or/with sliver addresses, etc.
type urlCompleter struct {
	name           string        // will be appended to the completion group name
	schemes        []string      // valid shemes
	menu           *gonsole.Menu // the menu we will use to make decisions
	useCurrentMenu bool          // do we need to load the current menu to make them ?
}

// NewURLCompleterFromSchemes - A generic completer, using the current menu for making
// interface completion decisions, that may be fed with a given set of URL shemes.
func NewURLCompleterFromSchemes(schemes []string) (c *urlCompleter) {
	c = &urlCompleter{
		name:           "",
		schemes:        schemes,
		useCurrentMenu: true,
	}
	return
}

// NewURLCompleterStager - A URL completer that is used for stager listeners
func NewURLCompleterStager() (c *urlCompleter) {
	c = &urlCompleter{
		name:           "stager",
		schemes:        stagerShemes,
		menu:           Console.GetMenu(constants.ServerMenu),
		useCurrentMenu: false,
	}
	return
}

// NewURLCompleterProxyUpdate - A completer used for the update command (its proxy option)
func NewURLCompleterProxyUpdate() (c *urlCompleter) {
	c = &urlCompleter{
		name:           "proxy",
		schemes:        proxyUpdateShemes,
		menu:           Console.GetMenu(constants.ServerMenu),
		useCurrentMenu: false,
	}
	return
}

// CompleteURL - A user wants to input a URL. We complete shemes and hosts. We pass an indication
// about the current context, so as to refine the list of hosts we give, and we also
// pass in a list of valid shemes for this precise completion: some commands will be
// compatible with only a handful of URL schemes, like meterpreter stagers.
func (c *urlCompleter) CompleteURL(last string) (prefix string, completions []*readline.CompletionGroup) {

	// If the URL completer is notified that it does not have to
	// get the current context, just use the cached one.
	if c.useCurrentMenu {
		c.menu = Console.CurrentMenu()
	} else if c.menu == nil {
		c.menu = Console.CurrentMenu()
	}

	// Normally, when we enter the function the last input should be nil
	// because we don't "append" a URL to anything  without a space (usually).
	// If its the case, just return schemes with ://
	// Also, we ask the shell not to add a space when inserting the completion
	// because we are not done with the URL yet.
	if last == "" || last == " " {
		completion := &readline.CompletionGroup{
			Name:        c.name + " protocol",
			MaxLength:   5,
			DisplayType: readline.TabDisplayGrid,
			NoSpace:     true,
		}
		for _, sch := range c.schemes {
			completion.Suggestions = append(completion.Suggestions, sch+"://")
		}
		return last, []*readline.CompletionGroup{completion}
	}

	// If not empty, and the last is suffixed by ://
	// return a list of hosts like for other completions.
	if strings.Contains(last, "://") {

		// Trim the input from the scheme://, otherwise no hosts will match
		parts := strings.Split(last, "://")
		var host string
		if len(parts) > 1 {
			host = parts[1] // Just keep the host, if empty its not a problem.
		}

		if c.menu.Name == constants.ServerMenu {
			// Client IPv4/IPv6 interfaces
			comp := Console.Completer.ClientInterfaceAddrs()
			for _, grp := range comp {
				var suggs []string
				for _, sugg := range grp.Suggestions {
					if strings.HasPrefix(sugg, host) {
						suggs = append(suggs, sugg)

					}
				}
				grp.Suggestions = suggs
			}
			completions = append(completions, comp...)

			return host, completions
		}

		if c.menu.Name == constants.SliverMenu {
			// Else we are starting a handler, or something like. Separate group for active session
			comp := ActiveSessionIfaceAddrs()
			for _, grp := range comp {
				var suggs []string
				for _, sugg := range grp.Suggestions {
					if strings.HasPrefix(sugg, host) {
						suggs = append(suggs, sugg)

					}
				}
				grp.Suggestions = suggs
			}
			completions = append(completions, comp...)

			return host, completions
		}
	} else {
		completion := &readline.CompletionGroup{
			Name:        c.name + " protocol",
			MaxLength:   5,
			DisplayType: readline.TabDisplayGrid,
			NoSpace:     true,
		}
		for _, sch := range c.schemes {
			if strings.HasPrefix(sch, last) {
				completion.Suggestions = append(completion.Suggestions, sch+"://")
			}
		}
		return last, []*readline.CompletionGroup{completion}
	}

	return
}
