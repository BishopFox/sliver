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
	"context"
	"errors"
	"strings"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
)

type implantHistory struct {
	con    *SliverClient
	items  []*clientpb.ImplantCommand
	Stream rpcpb.SliverRPC_ImplantHistoryClient
	pos    int
	user   bool
}

// SaveCommandLine sends a command-line to the server for saving,
// with information about the currrent active target and user.
// Called in the implant-command tree root persistent post-runner.
func (s *ActiveTarget) SaveCommandLine(args []string) {
	if s.hist == nil {
		return
	}

	cmdline := strings.Join(args, " ")

	s.hist.Write(cmdline)
}

func (con *SliverClient) newImplantHistory(user bool) (*implantHistory, error) {
	hist := &implantHistory{
		con:  con,
		user: user,
	}

	// Always refresh our cache when connecting.
	defer hist.Dump()

	// Important; in Write(), user should not use this tream.
	if hist.user {
		return hist, nil
	}

	stream, err := con.Rpc.ImplantHistory(context.Background())
	if err != nil {
		return nil, err
	}

	// Refresh the list.
	hist.Stream = stream

	return hist, nil
}

// Write - Sends the last command to the server for saving.
// Some commands are not saved (background, exit, etc)
func (h *implantHistory) Write(cmdline string) (int, error) {
	sess, beac := h.con.ActiveTarget.Get()
	if sess == nil && beac == nil {
		return len(h.items), nil
	}

	cmdline = strings.TrimSpace(cmdline)

	// Don't save queries for the list of commands.
	if isTrivialCommand(cmdline) {
		return len(h.items), nil
	}

	// Populate a command line with its context.
	cmd := &clientpb.ImplantCommand{}

	cmd.Block = cmdline
	cmd.ExecutedAt = time.Now().Unix()

	if sess != nil {
		cmd.ImplantID = sess.ID
		cmd.ImplantName = sess.Name
	} else if beac != nil {
		cmd.ImplantID = beac.ID
		cmd.ImplantName = beac.Name
	}

	// Save it in memory
	h.items = append(h.items, cmd)

	if h.user {
		return len(h.items), nil
	}

	err := h.Stream.Send(cmd)

	return len(h.items), err
}

// GetLine returns a line from history.
func (h *implantHistory) GetLine(pos int) (string, error) {
	if pos < 0 {
		return "", nil
	}

	// Refresh the command history.
	if len(h.items) == 0 {
		h.Dump()
	}

	if pos >= len(h.items) {
		return "", errors.New("Invalid history index")
	}

	return h.items[pos].Block, nil
}

// Len returns the number of lines in history.
func (h *implantHistory) Len() int {
	h.Dump()
	return len(h.items)
}

// Dump returns the entire history, and caches it
// internally to avoid queries when possible.
func (h *implantHistory) Dump() interface{} {
	sess, beac := h.con.ActiveTarget.Get()
	if sess == nil && beac == nil {
		return h.items
	}

	req := &clientpb.HistoryRequest{
		UserOnly: h.user,
	}

	if sess != nil {
		req.ImplantID = sess.ID
		req.ImplantName = sess.Name
	} else if beac != nil {
		req.ImplantID = beac.ID
		req.ImplantName = beac.Name
	}

	history, err := h.con.Rpc.GetImplantHistory(context.Background(), req)
	if err != nil {
		return h.items
	}

	h.items = history.Commands

	return h.items
}

// Close closes the implant history stream.
func (h *implantHistory) Close() error {
	if h.Stream == nil {
		return nil
	}

	_, err := h.Stream.CloseAndRecv()
	return err
}

// isTrivialCommand returns true for commands that don't
// need to be saved in a given implant command history.
func isTrivialCommand(cmdline string) bool {
	ignoreCmds := map[string]bool{
		"":           true,
		"background": true,
		"history":    true,
	}

	for key := range ignoreCmds {
		if key != "" && strings.HasPrefix(cmdline, key) {
			return true
		}
	}

	ignore, found := ignoreCmds[cmdline]
	if !found {
		return false
	}

	return ignore
}
