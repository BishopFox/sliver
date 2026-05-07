// Copyright (c) 2020 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package event

import (
	"encoding/json"

	"maunium.net/go/mautrix/id"
)

type RelationType string

const (
	RelReplace             RelationType = "m.replace"
	RelReference           RelationType = "m.reference"
	RelAnnotation          RelationType = "m.annotation"
	RelThread              RelationType = "m.thread"
	RelBeeperTranscription RelationType = "com.beeper.transcription"
)

type RelatesTo struct {
	Type    RelationType `json:"rel_type,omitempty"`
	EventID id.EventID   `json:"event_id,omitempty"`
	Key     string       `json:"key,omitempty"`

	InReplyTo     *InReplyTo `json:"m.in_reply_to,omitempty"`
	IsFallingBack bool       `json:"is_falling_back,omitempty"`
}

type InReplyTo struct {
	EventID id.EventID `json:"event_id,omitempty"`

	UnstableRoomID id.RoomID `json:"com.beeper.cross_room_id,omitempty"`
}

func (rel *RelatesTo) Copy() *RelatesTo {
	if rel == nil {
		return nil
	}
	cp := *rel
	return &cp
}

func (rel *RelatesTo) GetReplaceID() id.EventID {
	if rel != nil && rel.Type == RelReplace {
		return rel.EventID
	}
	return ""
}

func (rel *RelatesTo) GetReferenceID() id.EventID {
	if rel != nil && rel.Type == RelReference {
		return rel.EventID
	}
	return ""
}

func (rel *RelatesTo) GetThreadParent() id.EventID {
	if rel != nil && rel.Type == RelThread {
		return rel.EventID
	}
	return ""
}

func (rel *RelatesTo) GetReplyTo() id.EventID {
	if rel != nil && rel.InReplyTo != nil {
		return rel.InReplyTo.EventID
	}
	return ""
}

func (rel *RelatesTo) GetNonFallbackReplyTo() id.EventID {
	if rel != nil && rel.InReplyTo != nil && (rel.Type != RelThread || !rel.IsFallingBack) {
		return rel.InReplyTo.EventID
	}
	return ""
}

func (rel *RelatesTo) GetAnnotationID() id.EventID {
	if rel != nil && rel.Type == RelAnnotation {
		return rel.EventID
	}
	return ""
}

func (rel *RelatesTo) GetAnnotationKey() string {
	if rel != nil && rel.Type == RelAnnotation {
		return rel.Key
	}
	return ""
}

func (rel *RelatesTo) SetReplace(mxid id.EventID) *RelatesTo {
	rel.Type = RelReplace
	rel.EventID = mxid
	return rel
}

func (rel *RelatesTo) SetReplyTo(mxid id.EventID) *RelatesTo {
	if rel.Type != RelThread {
		rel.Type = ""
		rel.EventID = ""
	}
	rel.InReplyTo = &InReplyTo{EventID: mxid}
	rel.IsFallingBack = false
	return rel
}

func (rel *RelatesTo) SetThread(mxid, fallback id.EventID) *RelatesTo {
	rel.Type = RelThread
	rel.EventID = mxid
	if fallback != "" && rel.GetReplyTo() == "" {
		rel.SetReplyTo(fallback)
		rel.IsFallingBack = true
	}
	return rel
}

func (rel *RelatesTo) SetAnnotation(mxid id.EventID, key string) *RelatesTo {
	rel.Type = RelAnnotation
	rel.EventID = mxid
	rel.Key = key
	return rel
}

type RelationChunkItem struct {
	Type    RelationType `json:"type"`
	EventID string       `json:"event_id,omitempty"`
	Key     string       `json:"key,omitempty"`
	Count   int          `json:"count,omitempty"`
}

type RelationChunk struct {
	Chunk []RelationChunkItem `json:"chunk"`

	Limited bool `json:"limited"`
	Count   int  `json:"count"`
}

type AnnotationChunk struct {
	RelationChunk
	Map map[string]int `json:"-"`
}

type serializableAnnotationChunk AnnotationChunk

func (ac *AnnotationChunk) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, (*serializableAnnotationChunk)(ac)); err != nil {
		return err
	}
	ac.Map = make(map[string]int)
	for _, item := range ac.Chunk {
		if item.Key != "" {
			ac.Map[item.Key] += item.Count
		}
	}
	return nil
}

func (ac *AnnotationChunk) Serialize() RelationChunk {
	ac.Chunk = make([]RelationChunkItem, len(ac.Map))
	i := 0
	for key, count := range ac.Map {
		ac.Chunk[i] = RelationChunkItem{
			Type:  RelAnnotation,
			Key:   key,
			Count: count,
		}
		i++
	}
	return ac.RelationChunk
}

type EventIDChunk struct {
	RelationChunk
	List []string `json:"-"`
}

type serializableEventIDChunk EventIDChunk

func (ec *EventIDChunk) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, (*serializableEventIDChunk)(ec)); err != nil {
		return err
	}
	for _, item := range ec.Chunk {
		ec.List = append(ec.List, item.EventID)
	}
	return nil
}

func (ec *EventIDChunk) Serialize(typ RelationType) RelationChunk {
	ec.Chunk = make([]RelationChunkItem, len(ec.List))
	for i, eventID := range ec.List {
		ec.Chunk[i] = RelationChunkItem{
			Type:    typ,
			EventID: eventID,
		}
	}
	return ec.RelationChunk
}

type Relations struct {
	Raw map[RelationType]RelationChunk `json:"-"`

	Annotations AnnotationChunk `json:"m.annotation,omitempty"`
	References  EventIDChunk    `json:"m.reference,omitempty"`
	Replaces    EventIDChunk    `json:"m.replace,omitempty"`
}

type serializableRelations Relations

func (relations *Relations) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &relations.Raw); err != nil {
		return err
	}
	return json.Unmarshal(data, (*serializableRelations)(relations))
}

func (relations *Relations) MarshalJSON() ([]byte, error) {
	if relations.Raw == nil {
		relations.Raw = make(map[RelationType]RelationChunk)
	}
	relations.Raw[RelAnnotation] = relations.Annotations.Serialize()
	relations.Raw[RelReference] = relations.References.Serialize(RelReference)
	relations.Raw[RelReplace] = relations.Replaces.Serialize(RelReplace)
	for key, item := range relations.Raw {
		if !item.Limited {
			item.Count = len(item.Chunk)
		}
		if item.Count == 0 {
			delete(relations.Raw, key)
		}
	}
	return json.Marshal(relations.Raw)
}
