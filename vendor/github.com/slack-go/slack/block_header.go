package slack

// HeaderBlock defines a new block of type header
//
// More Information: https://api.slack.com/reference/messaging/blocks#header
type HeaderBlock struct {
	Type    MessageBlockType `json:"type"`
	Text    *TextBlockObject `json:"text,omitempty"`
	BlockID string           `json:"block_id,omitempty"`
	// Level sets the heading level. Values 1-4 correspond to H1-H4 heading
	// levels, respectively.
	Level int `json:"level,omitempty"`
}

// BlockType returns the type of the block
func (s HeaderBlock) BlockType() MessageBlockType {
	return s.Type
}

// ID returns the ID of the block
func (s HeaderBlock) ID() string {
	return s.BlockID
}

// HeaderBlockOption allows configuration of options for a new header block
type HeaderBlockOption func(*HeaderBlock)

func HeaderBlockOptionBlockID(blockID string) HeaderBlockOption {
	return func(block *HeaderBlock) {
		block.BlockID = blockID
	}
}

// HeaderBlockOptionLevel sets the heading level of the header block. Values 1-4
// correspond to H1-H4 heading levels, respectively.
func HeaderBlockOptionLevel(level int) HeaderBlockOption {
	return func(block *HeaderBlock) {
		block.Level = level
	}
}

// NewHeaderBlock returns a new instance of a header block to be rendered
func NewHeaderBlock(textObj *TextBlockObject, options ...HeaderBlockOption) *HeaderBlock {
	block := HeaderBlock{
		Type: MBTHeader,
		Text: textObj,
	}

	for _, option := range options {
		if option != nil {
			option(&block)
		}
	}

	return &block
}
