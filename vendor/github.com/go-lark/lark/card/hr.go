package card

var _ Element = (*HrBlock)(nil)

// HrBlock 分割线元素
type HrBlock struct{}

// Hr 分割线模块
func Hr() *HrBlock {
	return &HrBlock{}
}

// Render 渲染为 Renderer
func (h *HrBlock) Render() Renderer {
	return ElementTag{
		Tag: "hr",
	}
}
