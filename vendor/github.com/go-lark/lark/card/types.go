package card

// Element 所有元素均需实现的方法
type Element interface {
	Render() Renderer
}

// Renderer 渲染接口标记，用于保存成通用的树形结构
type Renderer interface{}

// ElementTag 标记元素的Tag
type ElementTag struct {
	Tag string `json:"tag"`
}
