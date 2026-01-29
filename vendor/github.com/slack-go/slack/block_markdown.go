package slack

// MarkdownBlock defines a block that lets you use markdown to format your text.
//
// This block can be used with AI apps when you expect a markdown response from an LLM
// that can get lost in translation rendering in Slack. Providing it in a markdown block
// leaves the translating to Slack to ensure your message appears as intended. Note that
// passing a single block may result in multiple blocks after translation.
//
// More Information: https://api.slack.com/reference/block-kit/blocks#markdown
type MarkdownBlock struct {
	Type    MessageBlockType `json:"type"`
	BlockID string           `json:"block_id,omitempty"`
	Text    string           `json:"text"`
}

// BlockType returns the type of the block
func (s MarkdownBlock) BlockType() MessageBlockType {
	return s.Type
}

// ID returns the ID of the block
func (s MarkdownBlock) ID() string {
	return s.BlockID
}

// NewMarkdownBlock returns an instance of a new Markdown Block type
func NewMarkdownBlock(blockID, text string) *MarkdownBlock {
	return &MarkdownBlock{
		Type:    MBTMarkdown,
		BlockID: blockID,
		Text:    text,
	}
}
