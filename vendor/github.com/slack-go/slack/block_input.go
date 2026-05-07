package slack

// InputBlock defines data that is used to display user input fields.
//
// More Information: https://api.slack.com/reference/block-kit/blocks#input
type InputBlock struct {
	Type           MessageBlockType `json:"type"`
	BlockID        string           `json:"block_id,omitempty"`
	Label          *TextBlockObject `json:"label"`
	Element        BlockElement     `json:"element"`
	Hint           *TextBlockObject `json:"hint,omitempty"`
	Optional       bool             `json:"optional,omitempty"`
	DispatchAction bool             `json:"dispatch_action,omitempty"`
}

// BlockType returns the type of the block
func (s InputBlock) BlockType() MessageBlockType {
	return s.Type
}

// ID returns the ID of the block
func (s InputBlock) ID() string {
	return s.BlockID
}

// NewInputBlock returns a new instance of an input block
func NewInputBlock(blockID string, label, hint *TextBlockObject, element BlockElement) *InputBlock {
	return &InputBlock{
		Type:    MBTInput,
		BlockID: blockID,
		Label:   label,
		Element: element,
		Hint:    hint,
	}
}

// WithOptional sets the optional flag on the input block
func (s *InputBlock) WithOptional(optional bool) *InputBlock {
	s.Optional = optional
	return s
}

// WithDispatchAction sets the dispatch action flag on the input block
func (s *InputBlock) WithDispatchAction(dispatchAction bool) *InputBlock {
	s.DispatchAction = dispatchAction
	return s
}
