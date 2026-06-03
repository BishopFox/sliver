package slack

// CallBlock defines data that is used to display a call in slack.
//
// More Information: https://api.slack.com/apis/calls#post_to_channel
type CallBlock struct {
	Type    MessageBlockType `json:"type"`
	BlockID string           `json:"block_id,omitempty"`
	CallID  string           `json:"call_id"`
	// Call is populated by Slack when retrieving messages containing a call block.
	// When creating a call block to post, only CallID is required.
	// Note: The structure differs from the Call type used in API responses.
	Call                   *CallBlockData `json:"call,omitempty"`
	APIDecorationAvailable bool           `json:"api_decoration_available,omitempty"`
}

// CallBlockData represents the call data structure as it appears in CallBlocks.
// This differs from the Call type used in API responses - CallBlock data is nested under V1.
type CallBlockData struct {
	V1               *CallBlockDataV1 `json:"v1,omitempty"`
	MediaBackendType string           `json:"media_backend_type,omitempty"`
}

// CallBlockDataV1 contains the actual call information within a CallBlock.
type CallBlockDataV1 struct {
	ID                 string             `json:"id"`
	AppID              string             `json:"app_id,omitempty"`
	AppIconURLs        *CallBlockIconURLs `json:"app_icon_urls,omitempty"`
	DateStart          int64              `json:"date_start"`
	DateEnd            int64              `json:"date_end"`
	ActiveParticipants []CallParticipant  `json:"active_participants,omitempty"`
	AllParticipants    []CallParticipant  `json:"all_participants,omitempty"`
	DisplayID          string             `json:"display_id,omitempty"`
	JoinURL            string             `json:"join_url,omitempty"`
	DesktopAppJoinURL  string             `json:"desktop_app_join_url,omitempty"`
	Name               string             `json:"name,omitempty"`
	CreatedBy          string             `json:"created_by,omitempty"`
	Channels           []string           `json:"channels,omitempty"`
	IsDMCall           bool               `json:"is_dm_call"`
	WasRejected        bool               `json:"was_rejected"`
	WasMissed          bool               `json:"was_missed"`
	WasAccepted        bool               `json:"was_accepted"`
	HasEnded           bool               `json:"has_ended"`
}

// CallBlockIconURLs contains app icon URLs at various sizes for a call integration.
type CallBlockIconURLs struct {
	Image32       string `json:"image_32,omitempty"`
	Image36       string `json:"image_36,omitempty"`
	Image48       string `json:"image_48,omitempty"`
	Image64       string `json:"image_64,omitempty"`
	Image72       string `json:"image_72,omitempty"`
	Image96       string `json:"image_96,omitempty"`
	Image128      string `json:"image_128,omitempty"`
	Image192      string `json:"image_192,omitempty"`
	Image512      string `json:"image_512,omitempty"`
	Image1024     string `json:"image_1024,omitempty"`
	ImageOriginal string `json:"image_original,omitempty"`
}

// BlockType returns the type of the block
func (s CallBlock) BlockType() MessageBlockType {
	return s.Type
}

// ID returns the ID of the block
func (s CallBlock) ID() string {
	return s.BlockID
}

// CallBlockOption allows configuration of options for a new call block
type CallBlockOption func(*CallBlock)

// CallBlockOptionBlockID sets the block_id for the call block
func CallBlockOptionBlockID(blockID string) CallBlockOption {
	return func(block *CallBlock) {
		block.BlockID = blockID
	}
}

// NewCallBlock returns a new instance of a call block
func NewCallBlock(callID string, options ...CallBlockOption) *CallBlock {
	block := &CallBlock{
		Type:   MBTCall,
		CallID: callID,
	}

	for _, option := range options {
		option(block)
	}

	return block
}
