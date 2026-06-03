package slack

import "encoding/json"

// UnknownBlock represents a block type that is not yet known. This block type
// exists to prevent Slack from introducing new and unknown block types that
// break this library. It preserves the raw JSON so that unrecognized blocks
// survive round-trip marshaling.
//
// If you encounter an UnknownBlock for a block type that Slack documents,
// please open an issue at https://github.com/slack-go/slack/issues so we can
// add first-class support for it.
type UnknownBlock struct {
	Type    MessageBlockType `json:"type"`
	BlockID string           `json:"block_id,omitempty"`
	raw     json.RawMessage
}

// BlockType returns the type of the block
func (b UnknownBlock) BlockType() MessageBlockType {
	return b.Type
}

// ID returns the ID of the block
func (s UnknownBlock) ID() string {
	return s.BlockID
}

// MarshalJSON returns the original raw JSON if available, preserving all fields
func (b UnknownBlock) MarshalJSON() ([]byte, error) {
	if b.raw != nil {
		return b.raw, nil
	}
	type alias UnknownBlock
	return json.Marshal(alias(b))
}
