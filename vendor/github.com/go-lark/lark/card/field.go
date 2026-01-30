package card

var _ Element = (*FieldBlock)(nil)

// FieldBlock 排版模块元素
type FieldBlock struct {
	short bool
	text  *TextBlock
}

type fieldRenderer struct {
	IsShort bool     `json:"is_short"`
	Text    Renderer `json:"text"`
}

// Render 渲染为 Renderer
func (f *FieldBlock) Render() Renderer {
	return fieldRenderer{
		IsShort: f.short,
		Text:    f.text.Render(),
	}
}

// Field 内容模块的排版元素
func Field(text *TextBlock) *FieldBlock {
	return &FieldBlock{text: text}
}

// Short 设置后，将会使用并排布局
func (f *FieldBlock) Short() *FieldBlock {
	f.short = true
	return f
}
