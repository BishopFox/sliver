package table

import (
	"io"
)

// Writer declares the interfaces that can be used to set up and render a table.
type Writer interface {
	AppendFooter(row Row, configs ...RowConfig)
	AppendHeader(row Row, configs ...RowConfig)
	AppendRow(row Row, configs ...RowConfig)
	AppendRows(rows []Row, configs ...RowConfig)
	AppendSeparator()
	FilterBy(filterBy []FilterBy)
	ImportGrid(grid interface{}) bool
	Length() int
	Pager(opts ...PagerOption) Pager
	Render() string
	RenderCSV() string
	RenderHTML() string
	RenderMarkdown() string
	RenderTSV() string
	ResetFooters()
	ResetHeaders()
	ResetRows()
	SetAutoIndex(autoIndex bool)
	SetCaption(format string, a ...interface{})
	SetColumnConfigs(configs []ColumnConfig)
	SetIndexColumn(colNum int)
	SetOutputMirror(mirror io.Writer)
	SetRowPainter(painter interface{})
	SetStyle(style Style)
	SetTitle(format string, a ...interface{})
	SortBy(sortBy []SortBy)
	Style() *Style
	SuppressEmptyColumns()
	SuppressTrailingSpaces()

	// deprecated; in favor if Style().Size.WidthMax
	SetAllowedRowLength(length int)
	// deprecated; in favor of Style().HTML.CSSClass
	SetHTMLCSSClass(cssClass string)
	// deprecated; in favor of Pager()
	SetPageSize(numLines int)
}

// NewWriter initializes and returns a Writer.
func NewWriter() Writer {
	return &Table{}
}
