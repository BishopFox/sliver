package console

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

	"github.com/spf13/cobra"

	"github.com/reeflective/console"

	consts "github.com/bishopfox/sliver/client/constants"
)

// FilterCommands - The active target may have various transport stacks,
// run on different hosts and operating systems, have networking tools, etc.
//
// Given a tree of commands which may or may not all act on a given target,
// the implant adds a series of annotations and hide directives to those which
// should not be available in the current state of things.
func (s *ActiveTarget) FilterCommands(rootCmd *cobra.Command) {
	targetFilters := s.Filters()

	for _, cmd := range rootCmd.Commands() {
		// Don't override commands if they are already hidden
		if cmd.Hidden {
			continue
		}

		if isFiltered(cmd, targetFilters) {
			cmd.Hidden = true
		}
	}
}

// FilterCommands shows/hides commands if the active target does support them (or not).
// Ex; to hide Windows commands on Linux implants, Wireguard tools on HTTP C2, etc.
// Both the cmd *cobra.Command passed and the filters can be nil, in which case the
// filters are recomputed by the console application for the current context.
func (con *SliverClient) FilterCommands(cmd *cobra.Command, filters ...string) {
	con.App.ShowCommands()

	if con.isCLI {
		filters = append(filters, consts.ConsoleCmdsFilter)
	}

	sess, beac := con.ActiveTarget.Get()
	if sess != nil || beac != nil {
		filters = append(filters, con.ActiveTarget.Filters()...)
	}

	con.App.HideCommands(filters...)

	if cmd != nil {
		for _, cmd := range cmd.Commands() {
			if cmd.Hidden {
				continue
			}

			if isFiltered(cmd, filters) {
				cmd.Hidden = true
			}
		}
	}
}

// AddPreRunner should be considered part of the temporary API.
// It is used by the Sliver client to run hooks before running its own pre-connect
// handlers, and can thus be used to register server-only pre-run routines.
func (con *SliverClient) AddPreRuns(hooks ...func(_ *cobra.Command, _ []string) error) {
	con.preRunners = append(con.preRunners, hooks...)
}

// runPreConnectHooks is also a function which might be temporary, and currently used
// to run "server-side provided" command pre-runners (for assets setup, jobs, etc)
func (con *SliverClient) runPreConnectHooks(cmd *cobra.Command, args []string) error {
	for _, hook := range con.preRunners {
		if hook == nil {
			continue
		}

		if err := hook(cmd, args); err != nil {
			return err
		}
	}

	return nil
}

// WARN: this is the premise of a big burden. Please bear this in mind.
// If I haven't speaked to you about it, or if you're not sure of what
// that means, ping me up and ask.
func (con *SliverClient) isOffline(cmd *cobra.Command) bool {
	// Teamclient configuration import does not need network.
	ts, _, err := cmd.Root().Find([]string{"teamserver", "client", "import"})
	if err == nil && ts != nil && ts == cmd {
		return true
	}

	tc, _, err := cmd.Root().Find([]string{"teamclient", "import"})
	if err == nil && ts != nil && tc == cmd {
		return true
	}

	return false
}

func isFiltered(cmd *cobra.Command, targetFilters []string) bool {
	if cmd.Annotations == nil {
		return false
	}

	// Get the filters on the command
	filterStr := cmd.Annotations[console.CommandFilterKey]
	filters := strings.Split(filterStr, ",")

	for _, cmdFilter := range filters {
		for _, filter := range targetFilters {
			if cmdFilter != "" && cmdFilter == filter {
				return true
			}
		}
	}

	return false
}
