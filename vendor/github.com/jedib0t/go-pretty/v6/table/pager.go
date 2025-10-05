package table

import (
	"io"
)

// Pager lets you interact with the table rendering in a paged manner.
type Pager interface {
	// GoTo moves to the given 1-indexed page number.
	GoTo(pageNum int) string
	// Location returns the current page number in 1-indexed form.
	Location() int
	// Next moves to the next available page and returns the same.
	Next() string
	// Prev moves to the previous available page and returns the same.
	Prev() string
	// Render returns the current page.
	Render() string
	// SetOutputMirror sets up the writer to which Render() will write the
	// output other than returning.
	SetOutputMirror(mirror io.Writer)
}

type pager struct {
	index        int // 0-indexed
	pages        []string
	outputMirror io.Writer
	size         int
}

func (p *pager) GoTo(pageNum int) string {
	if pageNum < 1 {
		pageNum = 1
	}
	if pageNum > len(p.pages) {
		pageNum = len(p.pages)
	}
	p.index = pageNum - 1
	return p.pages[p.index]
}

func (p *pager) Location() int {
	return p.index + 1
}

func (p *pager) Next() string {
	if p.index < len(p.pages)-1 {
		p.index++
	}
	return p.pages[p.index]
}

func (p *pager) Prev() string {
	if p.index > 0 {
		p.index--
	}
	return p.pages[p.index]
}

func (p *pager) Render() string {
	pageToWrite := p.pages[p.index]
	if p.outputMirror != nil {
		_, _ = p.outputMirror.Write([]byte(pageToWrite))
	}
	return pageToWrite
}

func (p *pager) SetOutputMirror(mirror io.Writer) {
	p.outputMirror = mirror
}
