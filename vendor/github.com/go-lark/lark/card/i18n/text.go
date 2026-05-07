package i18n

import "github.com/go-lark/lark/card"

// TextBlock contains LocalizedTextBlock and render it to I18N
type TextBlock struct {
	texts []*LocalizedTextBlock
}

// Text .
func Text(texts ...*LocalizedTextBlock) *TextBlock {
	return &TextBlock{
		texts: texts,
	}
}

// LocalizedTextBlock contains text with locale
type LocalizedTextBlock struct {
	locale string
	text   string
}

// LocalizedText .
func LocalizedText(locale, text string) *LocalizedTextBlock {
	return &LocalizedTextBlock{
		locale: locale,
		text:   text,
	}
}

type textBlockRenderer struct {
	card.ElementTag
	I18N map[string]card.Renderer `json:"i18n"`
}

// Render .
func (t *TextBlock) Render() card.Renderer {
	ret := textBlockRenderer{
		ElementTag: card.ElementTag{
			Tag: "plain_text",
		},
		I18N: make(map[string]card.Renderer),
	}
	for _, tt := range t.texts {
		ret.I18N[tt.locale] = tt.text
	}
	return ret
}
