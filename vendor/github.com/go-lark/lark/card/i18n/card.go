// Package i18n internationalization support for card
package i18n

import (
	"encoding/json"

	"github.com/go-lark/lark/card"
)

// Block 卡片元素
type Block struct {
	blocks         []*LocalizedBlock
	disableForward bool
	updateMulti    bool
	template       string
	links          *card.URLBlock
	title          *TextBlock
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
	Title    card.Renderer `json:"title"`
	Template string        `json:"template,omitempty"`
}

type cardRenderer struct {
	Config       cardConfigRenderer       `json:"config,omitempty"`
	Header       cardHeaderRenderer       `json:"header,omitempty"`
	CardLink     card.Renderer            `json:"card_link,omitempty"`
	I18NElements map[string]card.Renderer `json:"i18n_elements,omitempty"`
}

// Render 渲染为 Renderer
func (b *Block) Render() card.Renderer {
	ret := cardRenderer{
		Config: cardConfigRenderer{
			WideScreenMode: true,
			EnableForward:  !b.disableForward,
			UpdateMulti:    b.updateMulti,
		},
	}
	// render i18n
	ret.I18NElements = make(map[string]card.Renderer)
	for _, el := range b.blocks {
		ret.I18NElements[el.locale] = el.Render()
	}
	ret.Header = cardHeaderRenderer{
		Template: b.template,
		Title:    b.title.Render(),
	}
	if b.links != nil {
		ret.CardLink = b.links.Render()
	}
	return ret
}

// Card creates a i18n card block
func Card(blocks ...*LocalizedBlock) *Block {
	return &Block{
		blocks: blocks,
	}
}

// Title set title with locale
func (b *Block) Title(texts ...*LocalizedTextBlock) *Block {
	b.title = Text(texts...)
	return b
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

// Link 设置卡片跳转链接
func (b *Block) Link(href *card.URLBlock) *Block {
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
