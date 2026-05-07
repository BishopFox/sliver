package viber

// Keyboard struct
type Keyboard struct {
	Type          string   `json:"Type"`
	DefaultHeight bool     `json:"DefaultHeight,omitempty"`
	BgColor       string   `json:"BgColor,omitempty"`
	Buttons       []Button `json:"Buttons"`
}

// AddButton to keyboard
func (k *Keyboard) AddButton(b *Button) {
	k.Buttons = append(k.Buttons, *b)
}

// NewKeyboard struct with attribs init
func (v *Viber) NewKeyboard(bgcolor string, defaultHeight bool) *Keyboard {
	return &Keyboard{
		Type:          "keyboard",
		DefaultHeight: defaultHeight,
		BgColor:       bgcolor,
	}
}
