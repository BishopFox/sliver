package slack

// ContextActionsBlock defines data that is used to hold interactive action elements.
//
// More Information: https://docs.slack.dev/reference/block-kit/blocks/context-actions-block/
type ContextActionsBlock struct {
	Type     MessageBlockType `json:"type"`
	BlockID  string           `json:"block_id,omitempty"`
	Elements *BlockElements   `json:"elements"`
}

// BlockType returns the type of the block
func (s ContextActionsBlock) BlockType() MessageBlockType {
	return s.Type
}

// ID returns the ID of the block
func (s ContextActionsBlock) ID() string {
	return s.BlockID
}

// NewContextActionsBlock returns a new instance of a Context Actions Block
func NewContextActionsBlock(blockID string, elements ...BlockElement) *ContextActionsBlock {
	return &ContextActionsBlock{
		Type:    MBTContextActions,
		BlockID: blockID,
		Elements: &BlockElements{
			ElementSet: elements,
		},
	}
}
