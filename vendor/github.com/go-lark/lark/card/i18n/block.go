package i18n

import "github.com/go-lark/lark/card"

// LocalizedBlock .
type LocalizedBlock struct {
	locale   string
	elements []card.Element
}

// WithLocale creates a block with locale
func WithLocale(locale string, elements ...card.Element) *LocalizedBlock {
	return &LocalizedBlock{
		locale:   locale,
		elements: elements,
	}
}

// Render .
func (lb *LocalizedBlock) Render() card.Renderer {
	ret := make([]card.Renderer, len(lb.elements))
	for i, v := range lb.elements {
		ret[i] = v.Render()
	}

	return ret
}
