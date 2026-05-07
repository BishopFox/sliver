package slack

// DividerBlock for displaying a divider line between blocks (similar to <hr> tag in html)
//
// More Information: https://api.slack.com/reference/messaging/blocks#divider
type DividerBlock struct {
	Type    MessageBlockType `json:"type"`
	BlockID string           `json:"block_id,omitempty"`
}

// BlockType returns the type of the block
func (s DividerBlock) BlockType() MessageBlockType {
	return s.Type
}

// ID returns the ID of the block
func (s DividerBlock) ID() string {
	return s.BlockID
}

// NewDividerBlock returns a new instance of a divider block
func NewDividerBlock() *DividerBlock {
	return &DividerBlock{
		Type: MBTDivider,
	}
}
