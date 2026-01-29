package card

var _ Element = (*ColumnSetBlock)(nil)
var _ Element = (*ColumnBlock)(nil)
var _ Element = (*ColumnSetActionBlock)(nil)

// ColumnSetBlock column set element
type ColumnSetBlock struct {
	flexMode          string
	backgroundStyle   string
	horizontalSpacing string
	columns           []*ColumnBlock
	action            *ColumnSetActionBlock
}

type columnSetRenderer struct {
	ElementTag
	FlexMode          string     `json:"flex_mode"`
	BackgroundStyle   string     `json:"background_style,omitempty"`
	HorizontalSpacing string     `json:"horizontal_spacing,omitempty"`
	Columns           []Renderer `json:"columns,omitempty"`
	Action            Renderer   `json:"action,omitempty"`
}

// Render .
func (c *ColumnSetBlock) Render() Renderer {
	ret := columnSetRenderer{
		ElementTag:        ElementTag{"column_set"},
		FlexMode:          c.flexMode,
		BackgroundStyle:   c.backgroundStyle,
		HorizontalSpacing: c.horizontalSpacing,
	}
	for _, col := range c.columns {
		ret.Columns = append(ret.Columns, col.Render())
	}
	if c.action != nil {
		ret.Action = c.action.Render()
	}

	return ret
}

// ColumnSet .
func ColumnSet(columns ...*ColumnBlock) *ColumnSetBlock {
	return &ColumnSetBlock{
		columns:  columns,
		flexMode: "none",
	}
}

// FlexMode set flex mode
func (c *ColumnSetBlock) FlexMode(mode string) *ColumnSetBlock {
	c.flexMode = mode
	return c
}

// BackgroundStyle set background style
func (c *ColumnSetBlock) BackgroundStyle(style string) *ColumnSetBlock {
	c.backgroundStyle = style
	return c
}

// HorizontalSpacing set horizontal spacing
func (c *ColumnSetBlock) HorizontalSpacing(hs string) *ColumnSetBlock {
	c.horizontalSpacing = hs
	return c
}

// Action add column set action
func (c *ColumnSetBlock) Action(action *ColumnSetActionBlock) *ColumnSetBlock {
	c.action = action
	return c
}

// ColumnBlock column element
type ColumnBlock struct {
	width         string
	weight        int
	verticalAlign string
	elements      []Element
}

type columnRenderer struct {
	ElementTag
	Width         string     `json:"width,omitempty"`
	Weight        int        `json:"weight,omitempty"`
	VerticalAlign string     `json:"vertical_align,omitempty"`
	Elements      []Renderer `json:"elements,omitempty"`
}

// Column .
func Column(els ...Element) *ColumnBlock {
	return &ColumnBlock{
		elements: els,
	}
}

// Width .
func (c *ColumnBlock) Width(width string) *ColumnBlock {
	c.width = width
	return c
}

// Weight .
func (c *ColumnBlock) Weight(weight int) *ColumnBlock {
	c.weight = weight
	return c
}

// VerticalAlign .
func (c *ColumnBlock) VerticalAlign(align string) *ColumnBlock {
	c.verticalAlign = align
	return c
}

// Render .
func (c *ColumnBlock) Render() Renderer {
	ret := columnRenderer{
		ElementTag:    ElementTag{"column"},
		Width:         c.width,
		Weight:        c.weight,
		VerticalAlign: c.verticalAlign,
		Elements:      renderElements(c.elements),
	}
	return ret
}

// ColumnSetActionBlock column action element
type ColumnSetActionBlock struct {
	multiURL *URLBlock
}

type columnActionRenderer struct {
	MultiURL Renderer `json:"multi_url,omitempty"`
}

// ColumnSetAction .
func ColumnSetAction(url *URLBlock) *ColumnSetActionBlock {
	return &ColumnSetActionBlock{
		multiURL: url,
	}
}

// Render .
func (c *ColumnSetActionBlock) Render() Renderer {
	ret := columnActionRenderer{
		MultiURL: c.multiURL.Render(),
	}
	return ret
}
