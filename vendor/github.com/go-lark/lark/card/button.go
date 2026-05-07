package card

var _ Element = (*ButtonBlock)(nil)

// ButtonBlock 按钮元素
type ButtonBlock struct {
	text     *TextBlock
	name     string
	url      string
	multiURL *URLBlock
	btnType  string
	value    map[string]interface{}
	confirm  *ConfirmBlock
}

type buttonRenderer struct {
	ElementTag
	Text     Renderer               `json:"text"`
	Name     string                 `json:"name,omitempty"`
	URL      string                 `json:"url,omitempty"`
	MultiURL Renderer               `json:"multi_url,omitempty"`
	Type     string                 `json:"type,omitempty"`
	Value    map[string]interface{} `json:"value,omitempty"`
	Confirm  Renderer               `json:"confirm,omitempty"`
}

// Render 渲染为 Renderer
func (b *ButtonBlock) Render() Renderer {
	ret := buttonRenderer{
		ElementTag: ElementTag{
			Tag: "button",
		},
		Name:  b.name,
		Text:  b.text.Render(),
		URL:   b.url,
		Type:  b.btnType,
		Value: b.value,
	}
	if b.multiURL != nil {
		ret.MultiURL = b.multiURL.Render()
	}
	if b.confirm != nil {
		ret.Confirm = b.confirm.Render()
	}
	return ret
}

// Button 按钮交互元素
func Button(text *TextBlock) *ButtonBlock {
	return (&ButtonBlock{text: text}).Default()
}

// Name 按钮的标识
func (b *ButtonBlock) Name(n string) *ButtonBlock {
	b.name = n
	return b
}

// URL 按钮的跳转链接
func (b *ButtonBlock) URL(u string) *ButtonBlock {
	b.url = u
	return b
}

// MultiURL 按钮的多端差异跳转链接
func (b *ButtonBlock) MultiURL(u *URLBlock) *ButtonBlock {
	b.multiURL = u
	return b
}

// Value 点击后发送给业务方的数据
func (b *ButtonBlock) Value(v map[string]interface{}) *ButtonBlock {
	b.value = v
	return b
}

// Confirm 点击后二次确认的弹框
func (b *ButtonBlock) Confirm(title, text string) *ButtonBlock {
	b.confirm = Confirm(title, text)
	return b
}

// Default 设置按钮样式（次要按钮）
func (b *ButtonBlock) Default() *ButtonBlock {
	b.btnType = "default"
	return b
}

// Primary 设置按钮样式（主要按钮）
func (b *ButtonBlock) Primary() *ButtonBlock {
	b.btnType = "primary"
	return b
}

// Danger 设置按钮样式（警示按钮）
func (b *ButtonBlock) Danger() *ButtonBlock {
	b.btnType = "danger"
	return b
}
