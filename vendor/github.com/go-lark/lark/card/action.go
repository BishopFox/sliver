package card

var _ Element = (*ActionBlock)(nil)

// ActionBlock 交互元素
type ActionBlock struct {
	actions []Element
	layout  string
}

type actionRenderer struct {
	ElementTag
	Actions []Renderer `json:"actions"`
	Layout  string     `json:"layout,omitempty"`
}

// Render 渲染为 Renderer
func (a *ActionBlock) Render() Renderer {
	return actionRenderer{
		ElementTag: ElementTag{
			Tag: "action",
		},
		Actions: renderElements(a.actions),
		Layout:  a.layout,
	}
}

// Action 交互元素，可添加 Button, SelectMenu, Overflow, DatePicker, TimePicker, DatetimePicker
func Action(actions ...Element) *ActionBlock {
	return &ActionBlock{
		actions: actions,
	}
}

// BisectedLayout 二等分布局排列
func (a *ActionBlock) BisectedLayout() *ActionBlock {
	a.layout = "bisected"
	return a
}

// TrisectionLayout 三等分布局排列
func (a *ActionBlock) TrisectionLayout() *ActionBlock {
	a.layout = "trisection"
	return a
}

// FlowLayout 自适应流式布局
func (a *ActionBlock) FlowLayout() *ActionBlock {
	a.layout = "flow"
	return a
}
