package core

import (
	"sync"
)

var (
	// Reactions - Manages/tracks reactions
	Reactions = &reactions{
		reactionMap: map[string][]Reaction{},
		mutex:       &sync.RWMutex{},
	}
)

type reactions struct {
	reactionMap map[string][]Reaction
	mutex       *sync.RWMutex
	reactionID  int
}

// Start - Start a reaction
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
	ID        int
	EventType string   `json:"event_type"`
	Commands  []string `json:"commands"`
}
