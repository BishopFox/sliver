package viber

// RichMediaMessage / Carousel
type RichMediaMessage struct {
	AuthToken     string      `json:"auth_token"`
	Receiver      string      `json:"receiver,omitempty"`
	Type          MessageType `json:"type"`
	MinAPIVersion int         `json:"min_api_version"`
	RichMedia     RichMedia   `json:"rich_media"`
	AltText       string      `json:"alt_text,omitempty"`
	Keyboard      *Keyboard   `json:"keyboard,omitempty"`
	TrackingData  string      `json:"tracking_data,omitempty"`
}

// RichMedia for carousel
type RichMedia struct {
	Type                MessageType `json:"Type"`
	ButtonsGroupColumns int         `json:"ButtonsGroupColumns"`
	ButtonsGroupRows    int         `json:"ButtonsGroupRows"`
	BgColor             string      `json:"BgColor"`
	Buttons             []Button    `json:"Buttons"`
}

// AddButton to rich media message
func (rm *RichMediaMessage) AddButton(b *Button) {
	rm.RichMedia.Buttons = append(rm.RichMedia.Buttons, *b)
}

// NewRichMediaMessage creates new empty carousel message
func (v *Viber) NewRichMediaMessage(cols, rows int, bgColor string) *RichMediaMessage {
	return &RichMediaMessage{
		MinAPIVersion: 2,
		AuthToken:     v.AppKey,
		Type:          TypeRichMediaMessage,
		RichMedia: RichMedia{
			Type:                TypeRichMediaMessage,
			ButtonsGroupColumns: cols,
			ButtonsGroupRows:    rows,
			BgColor:             bgColor,
		},
	}
}

// SetKeyboard for text message
func (rm *RichMediaMessage) SetKeyboard(k *Keyboard) {
	// TODO
	rm.Keyboard = k
}

// SetReceiver for RichMedia message
func (rm *RichMediaMessage) SetReceiver(r string) {
	rm.Receiver = r
}

// SetFrom to satisfy interface although RichMedia messages can't be sent to publich chat and don't have From
func (rm *RichMediaMessage) SetFrom(from string) {}
