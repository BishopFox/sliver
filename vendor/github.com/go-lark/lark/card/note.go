package card

var _ Element = (*NoteBlock)(nil)

// NoteBlock 备注元素
type NoteBlock struct {
	elements []Element
}

type noteRenderer struct {
	ElementTag
	Elements []Renderer `json:"elements"`
}

// Render 渲染为 Renderer
func (n *NoteBlock) Render() Renderer {
	return noteRenderer{
		ElementTag: ElementTag{
			Tag: "note",
		},
		Elements: renderElements(n.elements),
	}
}

// Note 备注模块
func Note() *NoteBlock {
	return &NoteBlock{}
}

// AddText 添加一个文本模块
func (n *NoteBlock) AddText(t *TextBlock) *NoteBlock {
	n.elements = append(n.elements, t)
	return n
}

// AddImage 添加一个图片模块
func (n *NoteBlock) AddImage(i *ImgBlock) *NoteBlock {
	n.elements = append(n.elements, i)
	return n
}
