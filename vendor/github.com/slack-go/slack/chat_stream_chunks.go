package slack

import (
	"encoding/json"
)

// StreamChunkType identifies a chunk in the chat.startStream / chat.appendStream
// / chat.stopStream streaming-message protocol.
//
// More information: https://docs.slack.dev/reference/methods/chat.appendStream/
type StreamChunkType string

const (
	StreamChunkMarkdownText StreamChunkType = "markdown_text"
	StreamChunkTaskUpdate   StreamChunkType = "task_update"
	StreamChunkPlanUpdate   StreamChunkType = "plan_update"
	StreamChunkBlocks       StreamChunkType = "blocks"
)

// StreamChunk represents a single chunk in the streaming-message chunks array.
type StreamChunk interface {
	ChunkType() StreamChunkType
}

// MarkdownTextChunk streams markdown-formatted text.
type MarkdownTextChunk struct {
	Type StreamChunkType `json:"type"`
	Text string          `json:"text"`
}

func (c MarkdownTextChunk) ChunkType() StreamChunkType { return c.Type }

// NewMarkdownTextChunk returns a markdown_text chunk.
func NewMarkdownTextChunk(text string) MarkdownTextChunk {
	return MarkdownTextChunk{Type: StreamChunkMarkdownText, Text: text}
}

// TaskUpdateChunk streams a task status update that renders as a task card.
type TaskUpdateChunk struct {
	Type    StreamChunkType  `json:"type"`
	ID      string           `json:"id"`
	Title   string           `json:"title"`
	Status  TaskCardStatus   `json:"status,omitempty"`
	Details string           `json:"details,omitempty"`
	Output  string           `json:"output,omitempty"`
	Sources []TaskCardSource `json:"sources,omitempty"`
}

func (c TaskUpdateChunk) ChunkType() StreamChunkType { return c.Type }

// NewTaskUpdateChunk returns a task_update chunk with the given id and title.
func NewTaskUpdateChunk(id, title string) TaskUpdateChunk {
	return TaskUpdateChunk{Type: StreamChunkTaskUpdate, ID: id, Title: title}
}

// PlanUpdateChunk streams an update to the current plan's title.
type PlanUpdateChunk struct {
	Type  StreamChunkType `json:"type"`
	Title string          `json:"title"`
}

func (c PlanUpdateChunk) ChunkType() StreamChunkType { return c.Type }

// NewPlanUpdateChunk returns a plan_update chunk.
func NewPlanUpdateChunk(title string) PlanUpdateChunk {
	return PlanUpdateChunk{Type: StreamChunkPlanUpdate, Title: title}
}

// BlocksChunk streams a group of Block Kit blocks. Up to 50 blocks per chunk.
type BlocksChunk struct {
	Type   StreamChunkType `json:"type"`
	Blocks []Block         `json:"blocks"`
}

func (c BlocksChunk) ChunkType() StreamChunkType { return c.Type }

// NewBlocksChunk returns a blocks chunk containing the given blocks.
func NewBlocksChunk(blocks ...Block) BlocksChunk {
	return BlocksChunk{Type: StreamChunkBlocks, Blocks: blocks}
}

// MsgOptionChunks sets the `chunks` parameter for the streaming chat methods
// (chat.startStream / chat.appendStream / chat.stopStream). It is the
// transport for Block Kit agent-UI blocks (Alert, Card, Carousel, etc.) which
// chat.postMessage rejects as "Unsupported block type".
func MsgOptionChunks(chunks ...StreamChunk) MsgOption {
	return func(config *sendConfig) error {
		encoded, err := json.Marshal(chunks)
		if err != nil {
			return err
		}
		config.values.Set("chunks", string(encoded))
		return nil
	}
}
