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
	"context"
	"sync"
	"time"

	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

const (
	// ANSI Colors
	normal    = "\033[0m"
	black     = "\033[30m"
	red       = "\033[31m"
	green     = "\033[32m"
	orange    = "\033[33m"
	blue      = "\033[34m"
	purple    = "\033[35m"
	cyan      = "\033[36m"
	gray      = "\033[37m"
	bold      = "\033[1m"
	clearln   = "\r\x1b[2K"
	upN       = "\033[%dA"
	downN     = "\033[%dB"
	underline = "\033[4m"

	// Info - Display colorful information
	Info = bold + cyan + "[*] " + normal
	// Warn - Warn a user
	Warn = bold + red + "[!] " + normal
	// Debug - Display debug information
	Debug = bold + purple + "[-] " + normal
	// Woot - Display success
	Woot = bold + green + "[$] " + normal

	// ensure that nothing remains when we refresh the prompt
	seqClearScreenBelow = "\x1b[0J"
)

var (
	// Cache The unique completion cache for this console client.
	Cache = &cache{
		Sessions: map[uint32]*SessionCompCache{},
		mutex:    sync.RWMutex{},
	}
)

// cache - In order to avoid making too many requests to implants,
// we use a global cache that stores many items that we might have to retrieve
// from implants. This cache is also responsible for actually requesting data
// to implants, either when forced to, or when it determines the values are too
// old to be reliable. We can also request the cache to update part or fully.
type cache struct {
	Sessions map[uint32]*SessionCompCache
	mutex    sync.RWMutex
}

// AddSessionCache - Create a new cache for a newly registered session.
func (c *cache) AddSessionCache(sess *clientpb.Session) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	cache := &SessionCompCache{
		sess:        sess,
		CurrentDirs: map[string]*sliverpb.Ls{},
	}
	c.Sessions[sess.ID] = cache
}

// GetSessionCache - A completer needs the cache of a session.
func (c *cache) GetSessionCache(ID uint32) (cache *SessionCompCache) {
	cache, found := c.Sessions[ID]
	if !found {
		// Create a new cache if it does not exist
		c.AddSessionCache(getAllSessions()[ID])
		cache, found = c.Sessions[ID]
		return cache
	}
	return cache
}

// RemoveSessionData - Usually, when receiving a SessionKilled event,
// we request the cache to clear all data related to this session.
func (c *cache) RemoveSessionData(sess *clientpb.Session) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.Sessions, sess.ID)
}

// Reset - After each input loop (command executed) reset
// parts or all of session completion data caches
func (c *cache) Reset() {
	for _, cache := range c.Sessions {
		cache.Reset(false)
	}
}

//SessionCompCache - A cache of data dedicated to a single Session.
type SessionCompCache struct {
	// Base info
	sess *clientpb.Session

	// Network
	Interfaces      *sliverpb.Ifconfig
	ifaceLastUpdate time.Time

	// Processes. Usually we request to implant if processes have
	// not been updated in the last 30 seconds, as might move fast.
	Processes      *sliverpb.Ps
	procLastUpdate time.Time

	// Environment
	Env           *sliverpb.EnvInfo
	envLastUpdate time.Time

	// Directories and files.
	// The mere fact of backspacing with completions
	// will trigger a request to dir. Before doing this
	// we check if we have the current directory listing
	// If not the SessionCompCache will provide it to us.
	CurrentDirs map[string]*sliverpb.Ls
}

// GetDirectoryContents - A completer wants a directory list. If we have it and its fresh enough, return
// it directly. Otherwise, make the request on behalf of the completer, store results and return them.
func (sc *SessionCompCache) GetDirectoryContents(path string) (files *sliverpb.Ls) {

	// Check cache first
	if files, found := sc.CurrentDirs[path]; found {
		return files
	}

	// Else, request files to the implant.
	dirList, err := transport.RPC.Ls(context.Background(), &sliverpb.LsReq{
		Request: &commonpb.Request{
			SessionID: sc.sess.ID,
		},
		Path: path,
	})
	if err != nil {
		return
	}

	// Cache the data first
	sc.CurrentDirs[path] = dirList

	// And then return it
	return dirList
}

// GetNetInterfaces - Returns the net interfaces for an implant, either cached or requested.
func (sc *SessionCompCache) GetNetInterfaces() (ifaces *sliverpb.Ifconfig) {

	if time.Since(sc.ifaceLastUpdate) < (5*time.Minute) && sc.Interfaces != nil {
		return sc.Interfaces
	}

	ifconfig, err := transport.RPC.Ifconfig(context.Background(), &sliverpb.IfconfigReq{
		Request: core.SessionRequest(sc.sess),
	})
	if err != nil {
		return
	}

	// Cache data and reset timer
	sc.Interfaces = ifconfig
	sc.ifaceLastUpdate = time.Now()

	return ifconfig
}

// GetEnvironmentVariables - Returns the list of environment variables found on the host
func (sc *SessionCompCache) GetEnvironmentVariables() (env *sliverpb.EnvInfo) {

	if time.Since(sc.envLastUpdate) < (5*time.Minute) && sc.Env != nil {
		return sc.Env
	}

	envInfo, err := transport.RPC.GetEnv(context.Background(), &sliverpb.EnvReq{
		Name:    "",
		Request: core.ActiveSessionRequest(),
	})
	if err != nil {
		return
	}

	// Cache data and reset timer
	sc.Env = envInfo
	sc.envLastUpdate = time.Now()

	return envInfo
}

// GetProcesses - Returns the list of processes running on the session host.
func (sc *SessionCompCache) GetProcesses() (procs *sliverpb.Ps) {

	if time.Since(sc.procLastUpdate) < (5*time.Minute) && sc.Processes != nil {
		return sc.Processes
	}

	ps, err := transport.RPC.Ps(context.Background(), &sliverpb.PsReq{
		Request: core.ActiveSessionRequest(),
	})
	if err != nil {
		return
	}

	// Cache data and reset timer
	sc.Processes = ps
	sc.procLastUpdate = time.Now()

	return ps
}

// Reset - The session completion cache resets all or most of its items.
// This function is usually called at the end of each input command, because
// we might have modified the filesystem, processes may have new IDs pretty fast, etc...
// If all is false, we don't reset things like net Interfaces.
func (sc *SessionCompCache) Reset(all bool) {

	sc.CurrentDirs = map[string]*sliverpb.Ls{} // Reset directory contents
	sc.Processes = nil                         // Reset processes

	// More stable (usually) values are only cleared if all is true
	if all {
		sc.Interfaces = nil
		sc.Env = nil
	}
}

// getAllSessions - Get a map of all sessions
func getAllSessions() (sessionsMap map[uint32]*clientpb.Session) {
	sessions, err := transport.RPC.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		return
	}
	sessionsMap = map[uint32]*clientpb.Session{}
	for _, session := range sessions.GetSessions() {
		sessionsMap[session.ID] = session
	}

	return
}
