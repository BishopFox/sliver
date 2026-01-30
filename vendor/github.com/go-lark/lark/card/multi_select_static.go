package card

var _ Element = (*MultiSelectMenuBlock)(nil)

// MultiSelectMenuBlock 菜单元素
type MultiSelectMenuBlock struct {
	name        string
	tag         string
	placeholder string
	options     []Element
	required    bool
	disabled    bool
}

type multiselectMenuRenderer struct {
	ElementTag
	Name          string        `json:"name,omitempty"`
	Placeholder   Renderer      `json:"placeholder,omitempty"`
	Options       []Renderer    `json:"options,omitempty"`
	SelectedValue []interface{} `json:"selected_value"`
	Required      bool          `json:"required"`
	Disabled      bool          `json:"disabled"`
	Width         string        `json:"width,omitempty"`
}

// Render 渲染为 Renderer
func (s *MultiSelectMenuBlock) Render() Renderer {
	ret := multiselectMenuRenderer{
		ElementTag: ElementTag{
			Tag: s.tag,
		},
		Name:          s.name,
		Required:      s.required,
		Disabled:      s.disabled,
		SelectedValue: []interface{}{},
		Width:         "fill",
		Options:       renderElements(s.options),
		Placeholder:   Text(s.placeholder).Render(),
	}
	return ret
}

// MultiSelectMenu 菜单组件
func MultiSelectMenu(name string, opt ...*OptionBlock) *MultiSelectMenuBlock {
	ret := &MultiSelectMenuBlock{
		tag:      "multi_select_static",
		name:     name,
		options:  make([]Element, len(opt)),
		required: false,
		disabled: false,
	}
	for i, v := range opt {
		ret.options[i] = v
	}
	return ret
}

// Placeholder 未选中时展示的内容，无默认选项时必须设置
func (s *MultiSelectMenuBlock) Placeholder(p string) *MultiSelectMenuBlock {
	s.placeholder = p
	return s
}

// Required 是否必选
func (s *MultiSelectMenuBlock) Required(r bool) *MultiSelectMenuBlock {
	s.required = r
	return s
}

// Disabled 是否禁用
func (s *MultiSelectMenuBlock) Disabled(d bool) *MultiSelectMenuBlock {
	s.disabled = d
	return s
}

// Name 按钮的标识
func (s *MultiSelectMenuBlock) Name(n string) *MultiSelectMenuBlock {
	s.name = n
	return s
}
