package completers

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

	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

var (
	// Cache The unique completion cache for this console client.
	Cache = &CompletionCache{
		Sessions: map[uint32]*SessionCompCache{},
		mutex:    sync.RWMutex{},
	}
)

// CompletionCache - In order to avoid making too many requests to implants,
// we use a global cache that stores many items that we might have to retrieve
// from implants. This cache is also responsible for actually requesting data
// to implants, either when forced to, or when it determines the values are too
// old to be reliable. We can also request the cache to update part or fully.
type CompletionCache struct {
	Sessions map[uint32]*SessionCompCache
	mutex    sync.RWMutex
}

// AddSessionCache - Create a new cache for a newly registered session.
func (c *CompletionCache) AddSessionCache(sess *clientpb.Session) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	cache := &SessionCompCache{
		sess:        sess,
		CurrentDirs: map[string]*sliverpb.Ls{},
	}
	c.Sessions[sess.ID] = cache
}

// GetSessionCache - A completer needs the cache of a session.
func (c *CompletionCache) GetSessionCache(ID uint32) (cache *SessionCompCache) {
	cache, found := c.Sessions[ID]
	if !found {
		// Create a new cache if it does not exist
		c.AddSessionCache(GetAllSessions()[ID])
		cache, found = c.Sessions[ID]
		return cache
	}
	return cache
}

// RemoveSessionData - Usually, when receiving a SessionKilled event,
// we request the cache to clear all data related to this session.
func (c *CompletionCache) RemoveSessionData(sess *clientpb.Session) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.Sessions, sess.ID)
}

// Reset - After each input loop (command executed) reset
// parts or all of session completion data caches
func (c *CompletionCache) Reset() {
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
	Processes      []*commonpb.Process
	procLastUpdate time.Time

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

	if sc.Interfaces != nil {
		return sc.Interfaces
	}

	ifconfig, err := transport.RPC.Ifconfig(context.Background(), &sliverpb.IfconfigReq{
		Request: cctx.Request(sc.sess),
	})
	if err != nil {
		return
	}

	// Cache data
	sc.Interfaces = ifconfig

	return ifconfig
}

// Reset - The session completion cache resets all or most of its items.
// This function is usually called at the end of each input command, because
// we might have modified the filesystem, processes may have new IDs pretty fast, etc...
// If all is false, we don't reset things like net Interfaces.
func (sc *SessionCompCache) Reset(all bool) {

	// Reset directory contents
	sc.CurrentDirs = map[string]*sliverpb.Ls{}

	// Reset processes
	sc.Processes = []*commonpb.Process{}

	// More stable (usually) values are only cleared if all is true
	if all {
		sc.Interfaces = &sliverpb.Ifconfig{}
	}
}
