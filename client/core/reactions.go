package core

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"sync"

	consts "github.com/bishopfox/sliver/client/constants"
)

var (
	// Reactions - Manages/tracks reactions
	Reactions = &reactions{
		reactionMap: map[string][]Reaction{},
		mutex:       &sync.RWMutex{},
	}

	// ReactableEvents - A list of reactionable events
	ReactableEvents = []string{
		consts.SessionOpenedEvent,
		consts.SessionUpdateEvent,
		consts.SessionClosedEvent,
		consts.BeaconRegisteredEvent,
		consts.CanaryEvent,
		consts.WatchtowerEvent,
		consts.LootAddedEvent,
		consts.LootRemovedEvent,

		// Not sure if we want to add these or not:
		// consts.JobStartedEvent,
		// consts.JobStoppedEvent,
		// consts.BuildEvent,
		// consts.BuildCompletedEvent,
		// consts.ProfileEvent,
		// consts.WebsiteEvent,
	}
)

type reactions struct {
	reactionMap map[string][]Reaction
	mutex       *sync.RWMutex
	reactionID  int
}

// Add - Add a reaction
func (r *reactions) Add(reaction Reaction) Reaction {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.reactionID++
	reaction.ID = r.reactionID
	if eventActions, ok := r.reactionMap[reaction.EventType]; ok {
		r.reactionMap[reaction.EventType] = append(eventActions, reaction)
	} else {
		r.reactionMap[reaction.EventType] = []Reaction{reaction}
	}
	return reaction
}

// Remove - Remove a reaction, yes we're using linear search but this isn't exactly
// a performance critical piece of code and the map/slice is going to be very small
func (r *reactions) Remove(reactionID int) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	for eventType, eventReactions := range r.reactionMap {
		for index, reaction := range eventReactions {
			if reaction.ID == reactionID {
				r.reactionMap[eventType] = append(eventReactions[:index], eventReactions[index+1:]...)
				return true
			}
		}
	}
	return false
}

// On - Get all reactions of a specific type
func (r *reactions) On(eventType string) []Reaction {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	if reactions, ok := r.reactionMap[eventType]; ok {
		tmp := make([]Reaction, len(reactions))
		copy(tmp, reactions)
		return tmp
	}
	return []Reaction{}
}

// All - Get all reactions (returns a flat list with all event types)
func (r *reactions) All() []Reaction {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	reactions := []Reaction{}
	for _, eventReactions := range r.reactionMap {
		reactions = append(reactions, eventReactions...)
	}
	return reactions
}

// Reaction - Metadata about a portfwd listener
type Reaction struct {
	ID        int      `json:"-"`
	EventType string   `json:"event_type"`
	Commands  []string `json:"commands"`
}
