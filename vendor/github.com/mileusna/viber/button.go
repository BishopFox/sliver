package viber

// Button for carousel and keyboards
type Button struct {
	Columns             int        `json:"Columns"`
	Rows                int        `json:"Rows"`
	ActionType          ActionType `json:"ActionType"`
	ActionBody          string     `json:"ActionBody"`
	Image               string     `json:"Image,omitempty"`
	Text                string     `json:"Text,omitempty"`
	TextSize            TextSize   `json:"TextSize,omitempty"`
	TextVAlign          TextVAlign `json:"TextVAlign,omitempty"`
	TextHAlign          TextHAlign `json:"TextHAlign,omitempty"`
	TextOpacity         int8       `json:"TextOpacity,omitempty"`
	TextBgGradientColor string     `json:"TextBgGradientColor,omitempty"`
	BgColor             string     `json:"BgColor,omitempty"`
	BgMediaType         string     `json:"BgMediaType,omitempty"`
	BgMedia             string     `json:"BgMedia,omitempty"`
	BgLoop              bool       `json:"BgLoop,omitempty"`
	Silent              bool       `json:"Silent,omitempty"`
}

// NewButton helper function for creating button with text and image
func (v *Viber) NewButton(cols, rows int, typ ActionType, actionBody string, text, image string) *Button {
	return &Button{
		Columns:    cols,
		Rows:       rows,
		ActionType: typ,
		ActionBody: actionBody,
		Text:       text,
		Image:      image,
	}
}

// NewImageButton helper function for creating image button struct with common params
func (v *Viber) NewImageButton(cols, rows int, typ ActionType, actionBody string, image string) *Button {
	return &Button{
		Columns:    cols,
		Rows:       rows,
		ActionType: typ,
		ActionBody: actionBody,
		Image:      image,
	}
}

// NewTextButton helper function for creating image button struct with common params
func (v *Viber) NewTextButton(cols, rows int, t ActionType, actionBody, text string) *Button {
	return &Button{
		Columns:    cols,
		Rows:       rows,
		ActionType: t,
		ActionBody: actionBody,
		Text:       text,
	}
}

// TextSize for carousel buttons
// viber.Small
// viber.Medium (synonym to regular)
// viber.Large
// viber.Regular (default)
type TextSize string

// TextSize values
const (
	Small   = TextSize("small")
	Medium  = TextSize("medium")
	Large   = TextSize("large")
	Regular = TextSize("regular")
)

// TextSizeSmall for button text
func (b *Button) TextSizeSmall() *Button {
	b.TextSize = Small
	return b
}

// TextSizeMedium for button text, synonym to Regular
func (b *Button) TextSizeMedium() *Button {
	b.TextSize = Medium
	return b
}

// TextSizeRegular for button text, synonym to Medium
func (b *Button) TextSizeRegular() *Button {
	b.TextSize = Regular
	return b
}

// TextSizeLarge for button text
func (b *Button) TextSizeLarge() *Button {
	b.TextSize = Large
	return b
}

// ActionType for carousel buttons
// viber.Reply
// viber.OpenURL
type ActionType string

// ActionType values
const (
	Reply   = ActionType("reply")
	OpenURL = ActionType("open-url")
	None    = ActionType("none")
)

// TextVAlign for carousel buttons
// viber.Top
// viber.Middle (default)
// viber.Bottom
type TextVAlign string

// TextVAlign values
const (
	Top    = TextVAlign("top")
	Middle = TextVAlign("middle")
	Bottom = TextVAlign("bottom")
)

// TextVAlignTop vertically align text to the top
func (b *Button) TextVAlignTop() *Button {
	b.TextVAlign = Top
	return b
}

// TextVAlignMiddle vertically align text to the middle
func (b *Button) TextVAlignMiddle() *Button {
	b.TextVAlign = Middle
	return b
}

// TextVAlignBottom vertically align text to the bottom
func (b *Button) TextVAlignBottom() *Button {
	b.TextVAlign = Bottom
	return b
}

// TextHAlign for carousel buttons
// viber.Left
// viber.Center (default)
// viber.Middle
type TextHAlign string

// TextHAlign values
const (
	Left   = TextHAlign("left")
	Center = TextHAlign("center")
	Right  = TextHAlign("right")
)

// TextHAlignLeft horizontaly center text left
func (b *Button) TextHAlignLeft() *Button {
	b.TextHAlign = Left
	return b
}

// TextHAlignCenter horizontaly center text
func (b *Button) TextHAlignCenter() *Button {
	b.TextHAlign = Center
	return b
}

// TextHAlignRight horizontaly align text right
func (b *Button) TextHAlignRight() *Button {
	b.TextHAlign = Right
	return b
}

// SetSilent response from button
func (b *Button) SetSilent() *Button {
	b.Silent = true
	return b
}

// SetBgColor for button
func (b *Button) SetBgColor(hex string) *Button {
	b.BgColor = hex
	return b
}

// SetTextOpacity 0-100
func (b *Button) SetTextOpacity(o int8) *Button {
	if o >= 0 && o <= 100 {
		b.TextOpacity = o
	}
	return b
}

// BgMediaGIF set BgMedia to GIF with loop param
func (b *Button) BgMediaGIF(gifURL string, loop bool) *Button {
	b.BgMediaType = "gif"
	b.BgMedia = gifURL
	b.BgLoop = loop
	return b
}

// BgMediaPicture to set background to PNG or JPG. Use BgMediaGIF for GIF background
func (b *Button) BgMediaPicture(picURL string) *Button {
	b.BgMediaType = "picture"
	b.BgMedia = picURL
	return b
}
