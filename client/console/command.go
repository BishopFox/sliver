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
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/reeflective/console"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/spin"
)

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

// Filters returns list of constants describing which types of commands
// should NOT be available for the current target, eg. beacon commands if
// the target is a session, Windows commands if the target host is Linux.
func (s *ActiveTarget) Filters() []string {
	if s.session == nil && s.beacon == nil {
		return nil
	}

	filters := make([]string, 0)

	// Target type.
	switch {
	case s.session != nil:
		session := s.session

		// Forbid all beacon-only commands.
		filters = append(filters, consts.BeaconCmdsFilter)

		// Operating system
		if session.OS != "windows" {
			filters = append(filters, consts.WindowsCmdsFilter)
		}

		// C2 stack
		if session.Transport != "wg" {
			filters = append(filters, consts.WireguardCmdsFilter)
		}

	case s.beacon != nil:
		beacon := s.beacon

		// Forbid all session-only commands.
		filters = append(filters, consts.SessionCmdsFilter)

		// Operating system
		if beacon.OS != "windows" {
			filters = append(filters, consts.WindowsCmdsFilter)
		}

		// C2 stack
		if beacon.Transport != "wg" {
			filters = append(filters, consts.WireguardCmdsFilter)
		}
	}

	return filters
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

// SpinUntil starts a console display spinner in the background (non-blocking)
func (con *SliverClient) SpinUntil(message string, ctrl chan bool) {
	go spin.Until(os.Stdout, message, ctrl)
}

// WaitSignal listens for os.Signals and returns when receiving one of the following:
// SIGINT, SIGTERM, SIGQUIT.
//
// This can be used for commands which should block if executed in an exec-once CLI run:
// if the command is ran in the closed-loop console, this function will not monitor signals
// and return immediately.
func (con *SliverClient) WaitSignal() error {
	if !con.isCLI {
		return nil
	}

	sigchan := make(chan os.Signal, 1)

	signal.Notify(
		sigchan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		// syscall.SIGKILL,
	)

	sig := <-sigchan
	con.PrintInfof("Received signal %s\n", sig)

	return nil
}

func (con *SliverClient) waitSignalOrClose() error {
	if !con.isCLI {
		return nil
	}

	sigchan := make(chan os.Signal, 1)

	signal.Notify(
		sigchan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		// syscall.SIGKILL,
	)

	if con.waitingResult == nil {
		con.waitingResult = make(chan bool)
	}

	select {
	case sig := <-sigchan:
		con.PrintInfof("Received signal %s\n", sig)
	case <-con.waitingResult:
		con.waitingResult = make(chan bool)
		return nil
	}

	return nil
}
