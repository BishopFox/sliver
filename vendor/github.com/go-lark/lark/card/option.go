package card

var _ Element = (*OptionBlock)(nil)

// OptionBlock 选项元素
type OptionBlock struct {
	text, value, url string
	multiURL         *URLBlock
}

type optionRenderer struct {
	Text     Renderer `json:"text,omitempty"`
	Value    string   `json:"value"`
	URL      string   `json:"url,omitempty"`
	MultiURL Renderer `json:"multi_url,omitempty"`
}

// Render 渲染为 Renderer
func (o *OptionBlock) Render() Renderer {
	ret := optionRenderer{
		Value: o.value,
		URL:   o.url,
	}
	if o.text != "" {
		ret.Text = Text(o.text).Render()
	}
	if o.multiURL != nil {
		ret.MultiURL = o.multiURL.Render()
	}
	return ret
}

// Option 选项模块，可用于 SelectMenu 和 Overflow
func Option(value string) *OptionBlock {
	return &OptionBlock{
		value: value,
		text:  value,
	}
}

// Text 选项显示内容
func (o *OptionBlock) Text(s string) *OptionBlock {
	o.text = s
	return o
}

// URL 选项跳转链接
func (o *OptionBlock) URL(u string) *OptionBlock {
	o.url = u
	return o
}

// MultiURL 选项多端跳转链接
func (o *OptionBlock) MultiURL(u *URLBlock) *OptionBlock {
	o.multiURL = u
	return o
}
