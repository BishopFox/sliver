// Package card provides declarative card builder
package card

import (
	"encoding/json"
)

var _ Element = (*Block)(nil)

// Block 卡片元素
type Block struct {
	elements       []Element
	disableForward bool
	updateMulti    bool
	title          string
	template       string
	links          *URLBlock
}

// MarshalJSON implements json.Marshaler
func (b *Block) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(b.Render(), "", "  ")
}

// String implements fmt.Stringer
func (b *Block) String() string {
	bytes, _ := b.MarshalJSON()
	return string(bytes)
}

type cardConfigRenderer struct {
	WideScreenMode bool `json:"wide_screen_mode"`
	EnableForward  bool `json:"enable_forward"`
	UpdateMulti    bool `json:"update_multi"`
}

type cardHeaderRenderer struct {
	Title    Renderer `json:"title"`
	Template string   `json:"template,omitempty"`
}

type cardRenderer struct {
	Config   cardConfigRenderer `json:"config,omitempty"`
	Header   cardHeaderRenderer `json:"header,omitempty"`
	CardLink Renderer           `json:"card_link,omitempty"`
	Elements []Renderer         `json:"elements,omitempty"`
}

// Render 渲染为 Renderer
func (b *Block) Render() Renderer {
	ret := cardRenderer{
		Config: cardConfigRenderer{
			WideScreenMode: true,
			EnableForward:  !b.disableForward,
			UpdateMulti:    b.updateMulti,
		},
		Header: cardHeaderRenderer{
			Title:    Text(b.title).Render(),
			Template: b.template,
		},
		Elements: renderElements(b.elements),
	}
	if b.links != nil {
		ret.CardLink = b.links.Render()
	}
	return ret
}

// Card 包裹了最外层的卡片结构
func Card(el ...Element) *Block {
	return &Block{elements: el}
}

// NoForward 设置后，卡片将不可转发
func (b *Block) NoForward() *Block {
	b.disableForward = true
	return b
}

// UpdateMulti set card can be updated
func (b *Block) UpdateMulti(updateMulti bool) *Block {
	b.updateMulti = updateMulti
	return b
}

// Title 卡片标题
func (b *Block) Title(title string) *Block {
	b.title = title
	return b
}

// Link 设置卡片跳转链接
func (b *Block) Link(href *URLBlock) *Block {
	b.links = href
	return b
}

// Blue 设置卡片标题栏颜色（蓝色）
func (b *Block) Blue() *Block {
	b.template = "blue"
	return b
}

// Wathet 设置卡片标题栏颜色（浅蓝色）
func (b *Block) Wathet() *Block {
	b.template = "wathet"
	return b
}

// Turquoise 设置卡片标题栏颜色（松石绿）
func (b *Block) Turquoise() *Block {
	b.template = "turquoise"
	return b
}

// Green 设置卡片标题栏颜色（绿色）
func (b *Block) Green() *Block {
	b.template = "green"
	return b
}

// Yellow 设置卡片标题栏颜色（黄色）
func (b *Block) Yellow() *Block {
	b.template = "yellow"
	return b
}

// Orange 设置卡片标题栏颜色（橙色）
func (b *Block) Orange() *Block {
	b.template = "orange"
	return b
}

// Red 设置卡片标题栏颜色（红色）
func (b *Block) Red() *Block {
	b.template = "red"
	return b
}

// Carmine 设置卡片标题栏颜色（洋红色）
func (b *Block) Carmine() *Block {
	b.template = "carmine"
	return b
}

// Violet 设置卡片标题栏颜色（紫红色）
func (b *Block) Violet() *Block {
	b.template = "violet"
	return b
}

// Purple 设置卡片标题栏颜色（紫色）
func (b *Block) Purple() *Block {
	b.template = "purple"
	return b
}

// Indigo 设置卡片标题栏颜色（靛青色）
func (b *Block) Indigo() *Block {
	b.template = "indigo"
	return b
}

// Grey 设置卡片标题栏颜色（灰色）
func (b *Block) Grey() *Block {
	b.template = "grey"
	return b
}
