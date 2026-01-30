package card

var _ Element = (*PickerDatetimeBlock)(nil)

// PickerDatetimeBlock 输入元素
type PickerDatetimeBlock struct {
	name            string
	width           string
	placeholder     string
	initialDatetime string
	value           map[string]interface{}
}

type PickerDatetimeBlockRenderer struct {
	ElementTag
	Name            string                 `json:"name,omitempty"`
	Width           string                 `json:"width,omitempty"`
	Placeholder     Renderer               `json:"placeholder,omitempty"`
	InitialDatetime string                 `json:"initialDatetime,omitempty"`
	Value           map[string]interface{} `json:"value,omitempty"`
}

// Render 渲染为 Renderer
func (s *PickerDatetimeBlock) Render() Renderer {
	ret := PickerDatetimeBlockRenderer{
		ElementTag: ElementTag{
			Tag: "picker_datetime",
		},
		Name:            s.name,
		Placeholder:     Text(s.placeholder).Render(),
		InitialDatetime: s.initialDatetime,
		Value:           s.value,
		Width:           "fill",
	}
	return ret
}

// PickerDatetime 时间组件
func PickerDatetime(name string) *PickerDatetimeBlock {
	return &PickerDatetimeBlock{
		name: name,
	}
}

// Value 点击后发送给业务方的数据
func (s *PickerDatetimeBlock) Value(v map[string]interface{}) *PickerDatetimeBlock {
	s.value = v
	return s
}

// Placeholder 未选中值时展示的内容，无默认值时必填
func (s *PickerDatetimeBlock) Placeholder(p string) *PickerDatetimeBlock {
	s.placeholder = p
	return s
}

// InitialDatetime 默认选中值
func (s *PickerDatetimeBlock) InitialDatetime(d string) *PickerDatetimeBlock {
	s.initialDatetime = d
	return s
}

// Name 名字
func (s *PickerDatetimeBlock) Name(n string) *PickerDatetimeBlock {
	s.name = n
	return s
}
