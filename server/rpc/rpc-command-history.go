package rpc

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	Thiv program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	pb "github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/assets"
)

// GetHistory - A console requests a command history list, for the user or also a Session.
func (c *Server) GetHistory(in context.Context, req *pb.HistoryRequest) (res *pb.History, err error) {

	hist := &pb.History{Response: &commonpb.Response{}}

	// Get user name, and set responses
	name := c.getClientCommonName(in)

	// We always send the user/server history.
	hist.User, hist.UserHistLength, err = getUserHistory(name)
	if err != nil {
		hist.Response.Err = err.Error()
	}

	// Also send the session history file if there is a current implant being used
	if req.Session != nil {
		hist.Sliver, hist.SliverHistLength, err = getSessionHistory(req.Session)
		if err != nil {
			hist.Response.Err = err.Error()
		}
	}

	return hist, nil
}

func getUserHistory(name string) (lines []string, length int32, err error) {
	var filename string
	if name == "" {
		filename = filepath.Join(assets.GetRootAppDir(), ".history")
	} else {
		path := assets.GetUserDirectory(name)
		filename = filepath.Join(path, ".history")
	}
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	lines = strings.Split(string(data), "\n")
	length = int32(len(lines))

	return
}

func getSessionHistory(sess *pb.Session) (lines []string, length int32, err error) {
	sliverPath := assets.GetSliverDirectory(sess)

	// The same history is shared by all sessions who have the same target user,
	// and the same UUID. (mostly machine identifiers, except for Windows where there's more)
	histFile := fmt.Sprintf("%s_%s.history", sess.Username, sess.UUID)

	// Make the whole
	filename := filepath.Join(sliverPath, histFile)

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	lines = strings.Split(string(data), "\n")
	length = int32(len(lines))

	return
}

// AddToHistory - A client has sent a new command input line to be saved.
func (c *Server) AddToHistory(in context.Context, req *pb.AddCmdHistoryRequest) (res *pb.AddCmdHistory, err error) {

	res = &pb.AddCmdHistory{Response: &commonpb.Response{}}

	// Get an ID/operator name for this client, so that the Comms system knows
	// where to route back connections that are meant for this client proxy/portfwd utilities.
	name := c.getClientCommonName(in)

	// Filter various useless commands, depending on the context (user or session)
	// Always do it for the server, anyway. Do not save any empty line.
	if req.Session != nil && commandBanned(strings.TrimSpace(req.Line), uselessCmdsSession) {
		res.Doublon = true
	} else if commandBanned(strings.TrimSpace(req.Line), uselessCmdsServer) {
		res.Doublon = true
	}
	if strings.TrimSpace(req.Line) == "" {
		res.Doublon = true
	}

	// If there are no doublons, we check to which history file we need to save.
	if !res.Doublon {
		if req.Session != nil {
			err = writeSessionHistory(req.Session, req.Line)
			if err != nil {
				res.Response.Err = err.Error()
			}
		} else {
			err = writeUserHistory(name, req.Line)
		}

	}

	// Send back updated history sources
	res.User, _, err = getUserHistory(name)
	if err != nil {
		res.Response.Err = err.Error()
	}
	if req.Session != nil {
		res.Sliver, _, err = getSessionHistory(req.Session)
		if err != nil {
			res.Response.Err = err.Error()
		}
	}

	return res, nil
}

func writeUserHistory(name string, line string) (err error) {
	var filename string
	if name == "" {
		filename = filepath.Join(assets.GetRootAppDir(), ".history")
	} else {
		path := assets.GetUserDirectory(name)
		filename = filepath.Join(path, ".history")
	}

	// Write to client history file
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return errors.New("server could not find your client when requesting history: " + err.Error())
	}
	if _, err = f.WriteString(line + "\n"); err != nil {
		return errors.New("server could not find your client when requesting history: " + err.Error())
	}
	f.Close()

	return
}

func writeSessionHistory(sess *pb.Session, line string) (err error) {
	sliverPath := assets.GetSliverDirectory(sess)

	// The same history is shared by all sessions who have the same target user,
	// and the same UUID. (mostly machine identifiers, except for Windows where there's more)
	histFile := fmt.Sprintf("%s_%s.history", sess.Username, sess.UUID)

	// Make the whole
	filename := filepath.Join(sliverPath, histFile)

	// Write to session history file
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return errors.New("server could not find your client when requesting history: " + err.Error())
	}
	if _, err = f.WriteString(line + "\n"); err != nil {
		return errors.New("server could not find your client when requesting history: " + err.Error())
	}
	f.Close()

	return
}

// If any of these patterns are found in a command line, they are dropped, for user/server history.
var uselessCmdsServer = []string{
	"exit",
	"players",
	"-h",
	"--help",
	"cd",
	"ls",
	"cat",
	"jobs",
	"pwd",
	"use",
	"config",
}

// If any of these patterns are found in a command line, they are dropped, for session history.
var uselessCmdsSession = []string{
	"exit",
	"players",
	"-h",
	"--help",
	"help",
	"ls",
	"cat",
	"jobs",
	"pwd",
	"use",
	"config",
	"background",
	"info",
}

// commandBanned - If any of the words above apppear in a command, ON THEIR OWN (we don't match patterns),
// we do not save the corresponding input line in the history source.
func commandBanned(a string, list []string) bool {
	for _, b := range list {
		if strings.Contains(a, " "+b+" ") || strings.TrimSpace(a) == b {
			return true
		}
	}
	return false
}
