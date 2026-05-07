package linebot

// Mention type
type Mention struct {
	Mentionees []*Mentionee `json:"mentionees"`
}

// Mentionee type
type Mentionee struct {
	Index  int    `json:"index"`
	Length int    `json:"length"`
	UserID string `json:"userId,omitempty"`
}
