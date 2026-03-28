package models

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

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
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// AIConversation - A server-side AI conversation thread.
type AIConversation struct {
	ID                           uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt                    time.Time `gorm:"->;<-:create;"`
	UpdatedAt                    time.Time
	OperatorName                 string `gorm:"index"`
	Provider                     string `gorm:"index"`
	Model                        string
	ThinkingLevel                string
	Title                        string
	Summary                      string
	SystemPrompt                 string `gorm:"type:text;"`
	ActiveTurnID                 string `gorm:"index"`
	TurnState                    int32
	TargetSessionID              string `gorm:"index"`
	TargetBeaconID               string `gorm:"index"`
	ContextInputTokens           int64
	ContextOutputTokens          int64
	ContextTotalTokens           int64
	ContextWindowTokens          int64
	ContextWindowTokensEstimated bool

	Messages []AIConversationMessage `gorm:"foreignKey:ConversationID;constraint:OnDelete:CASCADE;"`
}

// BeforeCreate - GORM hook.
func (a *AIConversation) BeforeCreate(tx *gorm.DB) (err error) {
	a.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	now := time.Now()
	a.CreatedAt = now
	a.UpdatedAt = now
	return nil
}

// ToProtobuf - Convert to protobuf.
func (a *AIConversation) ToProtobuf() *clientpb.AIConversation {
	messages := make([]*clientpb.AIConversationMessage, 0, len(a.Messages))
	for _, message := range a.Messages {
		messages = append(messages, message.ToProtobuf())
	}

	return &clientpb.AIConversation{
		ID:              a.ID.String(),
		CreatedAt:       a.CreatedAt.Unix(),
		UpdatedAt:       a.UpdatedAt.Unix(),
		OperatorName:    a.OperatorName,
		Provider:        a.Provider,
		Model:           a.Model,
		ThinkingLevel:   a.ThinkingLevel,
		Title:           a.Title,
		Summary:         a.Summary,
		SystemPrompt:    a.SystemPrompt,
		Messages:        messages,
		ActiveTurnID:    a.ActiveTurnID,
		TurnState:       clientpb.AIConversationTurnState(a.TurnState),
		TargetSessionID: a.TargetSessionID,
		TargetBeaconID:  a.TargetBeaconID,
		ContextWindowUsage: aiContextWindowUsageToProtobuf(
			a.ContextInputTokens,
			a.ContextOutputTokens,
			a.ContextTotalTokens,
			a.ContextWindowTokens,
			a.ContextWindowTokensEstimated,
		),
	}
}

// AIConversationFromProtobuf - Convert a protobuf conversation to a model.
func AIConversationFromProtobuf(pbConversation *clientpb.AIConversation) *AIConversation {
	if pbConversation == nil {
		return &AIConversation{}
	}

	id, _ := uuid.FromString(pbConversation.ID)

	return &AIConversation{
		ID:                           id,
		OperatorName:                 pbConversation.OperatorName,
		Provider:                     pbConversation.Provider,
		Model:                        pbConversation.Model,
		ThinkingLevel:                pbConversation.ThinkingLevel,
		Title:                        pbConversation.Title,
		Summary:                      pbConversation.Summary,
		SystemPrompt:                 pbConversation.SystemPrompt,
		ActiveTurnID:                 pbConversation.ActiveTurnID,
		TurnState:                    int32(pbConversation.TurnState),
		TargetSessionID:              pbConversation.TargetSessionID,
		TargetBeaconID:               pbConversation.TargetBeaconID,
		ContextInputTokens:           pbConversation.GetContextWindowUsage().GetInputTokens(),
		ContextOutputTokens:          pbConversation.GetContextWindowUsage().GetOutputTokens(),
		ContextTotalTokens:           pbConversation.GetContextWindowUsage().GetTotalTokens(),
		ContextWindowTokens:          pbConversation.GetContextWindowUsage().GetContextWindowTokens(),
		ContextWindowTokensEstimated: pbConversation.GetContextWindowUsage().GetContextWindowTokensEstimated(),
	}
}

func aiContextWindowUsageToProtobuf(inputTokens, outputTokens, totalTokens, windowTokens int64, estimated bool) *clientpb.AIContextWindowUsage {
	if inputTokens == 0 && outputTokens == 0 && totalTokens == 0 && windowTokens == 0 && !estimated {
		return nil
	}
	return &clientpb.AIContextWindowUsage{
		InputTokens:                  inputTokens,
		OutputTokens:                 outputTokens,
		TotalTokens:                  totalTokens,
		ContextWindowTokens:          windowTokens,
		ContextWindowTokensEstimated: estimated,
	}
}

// AIConversationMessage - A single message stored within a conversation.
type AIConversationMessage struct {
	ID                uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	ConversationID    uuid.UUID `gorm:"type:uuid;index"`
	CreatedAt         time.Time `gorm:"->;<-:create;"`
	UpdatedAt         time.Time
	OperatorName      string `gorm:"index"`
	Provider          string `gorm:"index"`
	Model             string
	Sequence          uint32 `gorm:"index"`
	Role              string
	Content           string `gorm:"type:text;"`
	ProviderMessageID string
	FinishReason      string
	Kind              int32
	Visibility        int32
	State             int32
	TurnID            string `gorm:"index"`
	ItemID            string `gorm:"index"`
	ToolCallID        string `gorm:"index"`
	ToolName          string
	ToolArguments     string `gorm:"type:text;"`
	ToolResult        string `gorm:"type:text;"`
	ErrorText         string `gorm:"type:text;"`
	IncludeInContext  *bool
}

// BeforeCreate - GORM hook.
func (a *AIConversationMessage) BeforeCreate(tx *gorm.DB) (err error) {
	a.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	now := time.Now()
	a.CreatedAt = now
	a.UpdatedAt = now
	return nil
}

// ToProtobuf - Convert to protobuf.
func (a *AIConversationMessage) ToProtobuf() *clientpb.AIConversationMessage {
	return &clientpb.AIConversationMessage{
		ID:                a.ID.String(),
		ConversationID:    a.ConversationID.String(),
		CreatedAt:         a.CreatedAt.Unix(),
		UpdatedAt:         a.UpdatedAt.Unix(),
		OperatorName:      a.OperatorName,
		Provider:          a.Provider,
		Model:             a.Model,
		Sequence:          a.Sequence,
		Role:              a.Role,
		Content:           a.Content,
		ProviderMessageID: a.ProviderMessageID,
		FinishReason:      a.FinishReason,
		Kind:              clientpb.AIConversationMessageKind(a.Kind),
		Visibility:        clientpb.AIConversationMessageVisibility(a.Visibility),
		State:             clientpb.AIConversationMessageState(a.State),
		TurnID:            a.TurnID,
		ItemID:            a.ItemID,
		ToolCallID:        a.ToolCallID,
		ToolName:          a.ToolName,
		ToolArguments:     a.ToolArguments,
		ToolResult:        a.ToolResult,
		ErrorText:         a.ErrorText,
		IncludeInContext:  resolveAIConversationMessageIncludeInContext(a.IncludeInContext, a.Visibility),
	}
}

// AIConversationMessageFromProtobuf - Convert a protobuf message to a model.
func AIConversationMessageFromProtobuf(pbMessage *clientpb.AIConversationMessage) *AIConversationMessage {
	if pbMessage == nil {
		return &AIConversationMessage{}
	}

	id, _ := uuid.FromString(pbMessage.ID)
	conversationID, _ := uuid.FromString(pbMessage.ConversationID)

	return &AIConversationMessage{
		ID:                id,
		ConversationID:    conversationID,
		OperatorName:      pbMessage.OperatorName,
		Provider:          pbMessage.Provider,
		Model:             pbMessage.Model,
		Sequence:          pbMessage.Sequence,
		Role:              pbMessage.Role,
		Content:           pbMessage.Content,
		ProviderMessageID: pbMessage.ProviderMessageID,
		FinishReason:      pbMessage.FinishReason,
		Kind:              int32(pbMessage.Kind),
		Visibility:        int32(pbMessage.Visibility),
		State:             int32(pbMessage.State),
		TurnID:            pbMessage.TurnID,
		ItemID:            pbMessage.ItemID,
		ToolCallID:        pbMessage.ToolCallID,
		ToolName:          pbMessage.ToolName,
		ToolArguments:     pbMessage.ToolArguments,
		ToolResult:        pbMessage.ToolResult,
		ErrorText:         pbMessage.ErrorText,
		IncludeInContext:  cloneOptionalBool(pbMessage.IncludeInContext),
	}
}

func resolveAIConversationMessageIncludeInContext(value *bool, visibility int32) *bool {
	if value != nil {
		return cloneOptionalBool(value)
	}
	include := clientpb.AIConversationMessageVisibility(visibility) == clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_CONTEXT
	return &include
}

func cloneOptionalBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
