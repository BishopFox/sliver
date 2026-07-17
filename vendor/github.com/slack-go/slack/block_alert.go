package slack

// AlertLevel defines the severity for an AlertBlock.
type AlertLevel string

const (
	AlertLevelDefault AlertLevel = "default"
	AlertLevelInfo    AlertLevel = "info"
	AlertLevelWarning AlertLevel = "warning"
	AlertLevelError   AlertLevel = "error"
	AlertLevelSuccess AlertLevel = "success"
)

// AlertBlock defines a block of type alert used to surface a notification
// message with an optional severity level.
//
// Surface: modal only. Slack rejects alert blocks sent via chat.postMessage
// or the streaming APIs — use OpenView / UpdateView / PushView with a
// ModalViewRequest whose Blocks include the alert.
//
// More Information: https://docs.slack.dev/reference/block-kit/blocks/alert-block/
type AlertBlock struct {
	Type    MessageBlockType `json:"type"`
	Text    *TextBlockObject `json:"text"`
	Level   AlertLevel       `json:"level,omitempty"`
	BlockID string           `json:"block_id,omitempty"`
}

// BlockType returns the type of the block
func (s AlertBlock) BlockType() MessageBlockType {
	return s.Type
}

// ID returns the ID of the block
func (s AlertBlock) ID() string {
	return s.BlockID
}

// AlertBlockOption allows configuration of options for a new alert block
type AlertBlockOption func(*AlertBlock)

// AlertBlockOptionLevel sets the severity level for the alert block
func AlertBlockOptionLevel(level AlertLevel) AlertBlockOption {
	return func(block *AlertBlock) {
		block.Level = level
	}
}

// AlertBlockOptionBlockID sets the block ID for the alert block
func AlertBlockOptionBlockID(blockID string) AlertBlockOption {
	return func(block *AlertBlock) {
		block.BlockID = blockID
	}
}

// NewAlertBlock returns a new instance of an alert block
func NewAlertBlock(text *TextBlockObject, options ...AlertBlockOption) *AlertBlock {
	block := AlertBlock{
		Type: MBTAlert,
		Text: text,
	}

	for _, option := range options {
		if option != nil {
			option(&block)
		}
	}

	return &block
}
