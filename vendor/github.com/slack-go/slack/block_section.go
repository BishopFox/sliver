package slack

// SectionBlock defines a new block of type section
//
// More Information: https://api.slack.com/reference/messaging/blocks#section
type SectionBlock struct {
	Type      MessageBlockType   `json:"type"`
	Text      *TextBlockObject   `json:"text,omitempty"`
	BlockID   string             `json:"block_id,omitempty"`
	Fields    []*TextBlockObject `json:"fields,omitempty"`
	Accessory *Accessory         `json:"accessory,omitempty"`
	Expand    bool               `json:"expand,omitempty"`
}

// BlockType returns the type of the block
func (s SectionBlock) BlockType() MessageBlockType {
	return s.Type
}

// ID returns the ID of the block
func (s SectionBlock) ID() string {
	return s.BlockID
}

// SectionBlockOption allows configuration of options for a new section block
type SectionBlockOption func(*SectionBlock)

func SectionBlockOptionBlockID(blockID string) SectionBlockOption {
	return func(block *SectionBlock) {
		block.BlockID = blockID
	}
}

// SectionBlockOptionExpand allows long text to be auto-expanded when displaying
//
// @see https://api.slack.com/reference/block-kit/blocks#section
func SectionBlockOptionExpand(shouldExpand bool) SectionBlockOption {
	return func(block *SectionBlock) {
		block.Expand = shouldExpand
	}
}

// NewSectionBlock returns a new instance of a section block to be rendered
func NewSectionBlock(textObj *TextBlockObject, fields []*TextBlockObject, accessory *Accessory, options ...SectionBlockOption) *SectionBlock {
	block := SectionBlock{
		Type:      MBTSection,
		Text:      textObj,
		Fields:    fields,
		Accessory: accessory,
	}

	for _, option := range options {
		option(&block)
	}

	return &block
}
