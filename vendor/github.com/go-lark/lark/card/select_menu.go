package card

var _ Element = (*SelectMenuBlock)(nil)

// SelectMenuBlock 菜单元素
type SelectMenuBlock struct {
	tag           string
	placeholder   string
	initialOption string
	options       []Element
	value         map[string]interface{}
	confirm       *ConfirmBlock
}

type selectMenuRenderer struct {
	ElementTag
	Placeholder   Renderer               `json:"placeholder,omitempty"`
	InitialOption string                 `json:"initial_option,omitempty"`
	Options       []Renderer             `json:"options,omitempty"`
	Value         map[string]interface{} `json:"value,omitempty"`
	Confirm       Renderer               `json:"confirm,omitempty"`
}

// Render 渲染为 Renderer
func (s *SelectMenuBlock) Render() Renderer {
	ret := selectMenuRenderer{
		ElementTag: ElementTag{
			Tag: s.tag,
		},
		InitialOption: s.initialOption,
		Options:       renderElements(s.options),
		Value:         s.value,
		Placeholder:   Text(s.placeholder).Render(),
	}
	if s.confirm != nil {
		ret.Confirm = s.confirm.Render()
	}
	return ret
}

// SelectMenu 菜单组件
func SelectMenu(opt ...*OptionBlock) *SelectMenuBlock {
	ret := &SelectMenuBlock{
		tag:     "select_static",
		options: make([]Element, len(opt)),
	}
	for i, v := range opt {
		ret.options[i] = v
	}
	return ret
}

// SelectPerson 选人模式，value应设置为人员的open_id，options 为空则候选人员为当前群组
func (s *SelectMenuBlock) SelectPerson() *SelectMenuBlock {
	s.tag = "select_person"
	return s
}

// InitialOption 默认选项的 value 字段值
func (s *SelectMenuBlock) InitialOption(o string) *SelectMenuBlock {
	s.initialOption = o
	return s
}

// Placeholder 未选中时展示的内容，无默认选项时必须设置
func (s *SelectMenuBlock) Placeholder(p string) *SelectMenuBlock {
	s.placeholder = p
	return s
}

// Value 选中后发送给业务方的数据
func (s *SelectMenuBlock) Value(v map[string]interface{}) *SelectMenuBlock {
	s.value = v
	return s
}

// Confirm 选中后二次确认的弹框
func (s *SelectMenuBlock) Confirm(title, text string) *SelectMenuBlock {
	s.confirm = Confirm(title, text)
	return s
}
