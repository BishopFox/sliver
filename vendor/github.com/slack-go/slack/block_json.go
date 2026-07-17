package slack

import (
	"encoding/json"
	"fmt"
)

// RawJSONBlock represents a block created from raw JSON that preserves
// the original JSON structure. This is useful for testing new Slack block types
// before the library has full support, or for using blocks copied from Block Kit Builder.
//
// The block stores the original JSON and outputs it unchanged during marshalling,
// ensuring no data is lost through the unmarshal/marshal cycle.
type RawJSONBlock struct {
	Type    MessageBlockType `json:"-"`
	BlockID string           `json:"-"`
	raw     json.RawMessage
}

// BlockType returns the type of the block
func (r RawJSONBlock) BlockType() MessageBlockType {
	return r.Type
}

// ID returns the block_id of the block
func (r RawJSONBlock) ID() string {
	return r.BlockID
}

// MarshalJSON outputs the original JSON unchanged
func (r RawJSONBlock) MarshalJSON() ([]byte, error) {
	return r.raw, nil
}

// BlockFromJSON creates a RawJSONBlock from a JSON string that preserves
// the original JSON. This is useful for quickly testing blocks from Slack's
// Block Kit Builder or for incorporating new block types before the library
// has full support.
//
// The JSON can be either a single block object or an array of blocks.
// If an array is provided, only the first block is returned.
//
// The returned block stores the original JSON and outputs it unchanged during
// marshalling, ensuring no data is lost.
//
// Returns an error if the JSON is invalid, empty, or missing required fields.
//
// Example:
//
//	block, err := slack.BlockFromJSON(`{"type": "section", "text": {"type": "mrkdwn", "text": "Hello"}}`)
//	if err != nil {
//	    return err
//	}
//	blocks = append(blocks, block)
func BlockFromJSON(jsonStr string) (Block, error) {
	var rawJSON json.RawMessage
	var isArray bool

	// Try to unmarshal as an array first
	var arrayTest []json.RawMessage
	if err := json.Unmarshal([]byte(jsonStr), &arrayTest); err == nil && len(arrayTest) > 0 {
		rawJSON = arrayTest[0]
		isArray = true
	} else {
		// Try as a single block object
		if err := json.Unmarshal([]byte(jsonStr), &rawJSON); err != nil {
			return nil, fmt.Errorf("failed to unmarshal block JSON: %w", err)
		}
		isArray = false
	}

	if !isArray && len(rawJSON) == 0 {
		return nil, fmt.Errorf("no blocks found in JSON")
	}

	// Extract minimal fields for Block interface
	var minimal struct {
		Type    string `json:"type"`
		BlockID string `json:"block_id"`
	}
	if err := json.Unmarshal(rawJSON, &minimal); err != nil {
		return nil, fmt.Errorf("failed to extract block type: %w", err)
	}

	if minimal.Type == "" {
		return nil, fmt.Errorf("block missing required 'type' field")
	}

	return RawJSONBlock{
		Type:    MessageBlockType(minimal.Type),
		BlockID: minimal.BlockID,
		raw:     rawJSON,
	}, nil
}

// MustBlockFromJSON creates a Block from a JSON string and panics if there's an error.
// This is primarily intended for use in tests or examples where the JSON is known to be valid.
// For production code, use BlockFromJSON which returns an error instead.
//
// Example:
//
//	block := slack.MustBlockFromJSON(`{"type": "divider"}`)
//	msg := slack.NewBlockMessage(block)
func MustBlockFromJSON(jsonStr string) Block {
	block, err := BlockFromJSON(jsonStr)
	if err != nil {
		panic(fmt.Sprintf("MustBlockFromJSON: %v", err))
	}
	return block
}
