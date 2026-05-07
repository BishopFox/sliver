package card

var _ Element = (*DivBlock)(nil)

// DivBlock 内容元素
type DivBlock struct {
	fields []Element
	text   *TextBlock
	extra  Element
}

type divRenderer struct {
	ElementTag
	Text   Renderer   `json:"text,omitempty"`
	Fields []Renderer `json:"fields,omitempty"`
	Extra  Renderer   `json:"extra,omitempty"`
}

// Render 渲染为 Renderer
func (d *DivBlock) Render() Renderer {
	ret := divRenderer{
		ElementTag: ElementTag{
			Tag: "div",
		},
		Fields: renderElements(d.fields),
	}
	if d.text != nil {
		ret.Text = d.text.Render()
	}
	if d.extra != nil {
		ret.Extra = d.extra.Render()
	}
	return ret
}

// Div 内容模块
func Div(fields ...*FieldBlock) *DivBlock {
	ret := &DivBlock{fields: make([]Element, len(fields))}
	for i, v := range fields {
		ret.fields[i] = v
	}
	return ret
}

// Text 单个文本的展示，和 fields 至少要有一个
func (d *DivBlock) Text(t *TextBlock) *DivBlock {
	d.text = t
	return d
}

// Extra 模块的附加元素，展示在内容右侧。可附加 Img, Button, SelectMenu, Overflow, DatePicker, TimePicker, DatetimePicker
func (d *DivBlock) Extra(e Element) *DivBlock {
	d.extra = e
	return d
}
