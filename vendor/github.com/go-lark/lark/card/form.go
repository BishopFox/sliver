package card

var _ Element = (*FormBlock)(nil)

// FormBlock 表单元素
type FormBlock struct {
	name     string
	elements []Element
	value    map[string]interface{}
}

type formRenderer struct {
	ElementTag
	Name     string                 `json:"name"`
	Elements []Renderer             `json:"elements,omitempty"`
	Value    map[string]interface{} `json:"value,omitempty"`
}

// Render 渲染为 Renderer
func (b *FormBlock) Render() Renderer {
	ret := formRenderer{
		ElementTag: ElementTag{
			Tag: "form",
		},
		Name:     b.name,
		Elements: renderElements(b.elements),
		Value:    b.value,
	}
	return ret
}

// Form 表单交互元素
func Form(name string, el ...Element) *FormBlock {
	ret := &FormBlock{
		name:     name,
		elements: make([]Element, len(el))}
	for i, v := range el {
		ret.elements[i] = v
	}
	return ret
}

// Value 点击后发送给业务方的数据
func (b *FormBlock) Value(v map[string]interface{}) *FormBlock {
	b.value = v
	return b
}
