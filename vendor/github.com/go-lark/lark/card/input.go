package card

var _ Element = (*InputBlock)(nil)

// InputBlock 输入元素
type InputBlock struct {
	name        string
	tag         string
	placeholder string
}

type InputBlockRenderer struct {
	ElementTag
	Name          string   `json:"name"`
	Placeholder   Renderer `json:"placeholder,omitempty"`
	LabelPosition string   `json:"label_position"`
	Label         label    `json:"label,omitempty"`
	MaxLength     int      `json:"max_length"`
}

type label struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

// Render 渲染为 Renderer
func (s *InputBlock) Render() Renderer {
	ret := InputBlockRenderer{
		ElementTag: ElementTag{
			Tag: "input",
		},
		Name:          s.name,
		LabelPosition: "left",
		MaxLength:     120,
		Label: label{
			Tag:     "plain_text",
			Content: "Custom Input:",
		},
		Placeholder: Text(s.placeholder).Render(),
	}
	return ret
}

// Input 输入组件
func Input(name string) *InputBlock {
	return &InputBlock{
		name: name,
	}
}

// Placeholder 默认展示内容
func (s *InputBlock) Placeholder(str string) *InputBlock {
	s.placeholder = str
	return s
}
