package lark

import (
	"github.com/go-lark/lark/card"
	"github.com/go-lark/lark/card/i18n"
)

type i18nCardBuilder struct{}

// CardBuilder .
type CardBuilder struct {
	I18N *i18nCardBuilder
}

// Card wraps i18n card
func (i18nCardBuilder) Card(blocks ...*i18n.LocalizedBlock) *i18n.Block {
	return i18n.Card(blocks...)
}

func (i18nCardBuilder) WithLocale(locale string, elements ...card.Element) *i18n.LocalizedBlock {
	return i18n.WithLocale(locale, elements...)
}

// Title wraps i18n title block
func (i18nCardBuilder) LocalizedText(locale, s string) *i18n.LocalizedTextBlock {
	return i18n.LocalizedText(locale, s)
}

// NewCardBuilder 新建卡片构造器
func NewCardBuilder() *CardBuilder {
	return &CardBuilder{
		I18N: &i18nCardBuilder{},
	}
}

// Card 包裹了最外层的卡片结构
func (CardBuilder) Card(elements ...card.Element) *card.Block {
	return card.Card(elements...)
}

// Action 交互元素，可添加 Button, SelectMenu, Overflow, DatePicker, TimePicker, DatetimePicker
func (CardBuilder) Action(actions ...card.Element) *card.ActionBlock {
	return card.Action(actions...)
}

// Button 按钮交互元素
func (CardBuilder) Button(text *card.TextBlock) *card.ButtonBlock {
	return card.Button(text)
}

// Confirm 用于交互元素的二次确认
func (CardBuilder) Confirm(title, text string) *card.ConfirmBlock {
	return card.Confirm(title, text)
}

// DatePicker 日期选择器
func (CardBuilder) DatePicker() *card.DatePickerBlock {
	return card.DatePicker()
}

// TimePicker 时间选择器
func (CardBuilder) TimePicker() *card.TimePickerBlock {
	return card.TimePicker()
}

// DatetimePicker 日期时间选择器
func (CardBuilder) DatetimePicker() *card.DatetimePickerBlock {
	return card.DatetimePicker()
}

// Div 内容模块
func (CardBuilder) Div(fields ...*card.FieldBlock) *card.DivBlock {
	return card.Div(fields...)
}

// Field 内容模块的排版元素
func (CardBuilder) Field(text *card.TextBlock) *card.FieldBlock {
	return card.Field(text)
}

// Hr 分割线模块
func (CardBuilder) Hr() *card.HrBlock {
	return card.Hr()
}

// Img 图片展示模块
func (CardBuilder) Img(key string) *card.ImgBlock {
	return card.Img(key)
}

// Note 备注模块
func (CardBuilder) Note() *card.NoteBlock {
	return card.Note()
}

// Option 选项模块，可用于 SelectMenu 和 Overflow
func (CardBuilder) Option(value string) *card.OptionBlock {
	return card.Option(value)
}

// Overflow 折叠按钮菜单组件
func (CardBuilder) Overflow(options ...*card.OptionBlock) *card.OverflowBlock {
	return card.Overflow(options...)
}

// SelectMenu 菜单组件
func (CardBuilder) SelectMenu(options ...*card.OptionBlock) *card.SelectMenuBlock {
	return card.SelectMenu(options...)
}

// Text 文本模块
func (CardBuilder) Text(s string) *card.TextBlock {
	return card.Text(s)
}

// Markdown 单独使用的 Markdown 文本模块
func (CardBuilder) Markdown(s string) *card.MarkdownBlock {
	return card.Markdown(s)
}

// URL 链接模块
func (CardBuilder) URL() *card.URLBlock {
	return card.URL()
}

// ColumnSet column set module
func (CardBuilder) ColumnSet(columns ...*card.ColumnBlock) *card.ColumnSetBlock {
	return card.ColumnSet(columns...)
}

// Column column module
func (CardBuilder) Column(elements ...card.Element) *card.ColumnBlock {
	return card.Column(elements...)
}

// ColumnSetAction column action module
func (CardBuilder) ColumnSetAction(url *card.URLBlock) *card.ColumnSetActionBlock {
	return card.ColumnSetAction(url)
}
