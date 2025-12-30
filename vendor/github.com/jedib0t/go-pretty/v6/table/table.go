package table

import (
	"fmt"
	"io"
	"strings"
	"time"
	"unicode"

	"github.com/jedib0t/go-pretty/v6/text"
)

// Table helps print a 2-dimensional array in a human-readable pretty-table.
type Table struct {
	// allowedRowLength is the max allowed length for a row (or line of output)
	allowedRowLength int
	// enable automatic indexing of the rows and columns like a spreadsheet?
	autoIndex bool
	// autoIndexVIndexMaxLength denotes the length in chars for the last row
	autoIndexVIndexMaxLength int
	// caption stores the text to be rendered just below the table; and doesn't
	// get used when rendered as a CSV
	caption string
	// columnIsNonNumeric stores if a column contains non-numbers in all rows
	columnIsNonNumeric []bool
	// columnConfigs stores the custom-configuration for 1 or more columns
	columnConfigs []ColumnConfig
	// columnConfigMap stores the custom-configuration by column
	// number and is generated before rendering
	columnConfigMap map[int]ColumnConfig
	// firstRowOfPage tells if the renderer is on the first row of a page?
	firstRowOfPage bool
	// htmlCSSClass stores the HTML CSS Class to use on the <table> node
	htmlCSSClass string
	// indexColumn stores the number of the column considered as the "index"
	indexColumn int
	// maxColumnLengths stores the length of the longest line in each column
	maxColumnLengths []int
	// maxMergedColumnLengths stores the longest lengths for merged columns
	// endIndex -> startIndex -> maxMergedLength
	maxMergedColumnLengths map[int]map[int]int
	// maxRowLength stores the length of the longest row
	maxRowLength int
	// numColumns stores the (max.) number of columns seen
	numColumns int
	// numLinesRendered keeps track of the number of lines rendered and helps in
	// paginating long tables
	numLinesRendered int
	// outputMirror stores an io.Writer where the "Render" functions would write
	outputMirror io.Writer
	// pager controls how the output is separated into pages
	pager pager
	// rows stores the rows that make up the body (in string form)
	rows []rowStr
	// rowsColors stores the text.Colors over-rides for each row as defined by
	// rowPainter or rowPainterWithAttributes
	rowsColors []text.Colors
	// rowsConfigs stores RowConfig for each row
	rowsConfigMap map[int]RowConfig
	// rowsRaw stores the rows that make up the body
	rowsRaw []Row
	// rowsFooter stores the rows that make up the footer (in string form)
	rowsFooter []rowStr
	// rowsFooterConfigs stores RowConfig for each footer row
	rowsFooterConfigMap map[int]RowConfig
	// rowsFooterRaw stores the rows that make up the footer
	rowsFooterRaw []Row
	// rowsHeader stores the rows that make up the header (in string form)
	rowsHeader []rowStr
	// rowsHeaderConfigs stores RowConfig for each header row
	rowsHeaderConfigMap map[int]RowConfig
	// rowsHeaderRaw stores the rows that make up the header
	rowsHeaderRaw []Row
	// rowPainter is a custom function that given a Row, returns the colors to
	// use on the entire row
	rowPainter RowPainter
	// rowPainterWithAttributes is same as rowPainter, but with attributes
	rowPainterWithAttributes RowPainterWithAttributes
	// rowSeparator is a dummy row that contains the separator columns (dashes
	// that make up the separator between header/body/footer
	rowSeparator rowStr
	// separators is used to keep track of all rowIndices after which a
	// separator has to be rendered
	separators map[int]bool
	// sortBy stores a map of Column
	sortBy []SortBy
	// sortedRowIndices is the output of sorting
	sortedRowIndices []int
	// style contains all the strings used to draw the table, and more
	style *Style
	// suppressEmptyColumns hides columns which have no content on all regular
	// rows
	suppressEmptyColumns bool
	// suppressTrailingSpaces removes all trailing spaces from the end of the last column
	suppressTrailingSpaces bool
	// title contains the text to appear above the table
	title string
}

// AppendFooter appends the row to the List of footers to render.
//
// Only the first item in the "config" will be tagged against this row.
func (t *Table) AppendFooter(row Row, config ...RowConfig) {
	t.rowsFooterRaw = append(t.rowsFooterRaw, row)
	if len(config) > 0 {
		if t.rowsFooterConfigMap == nil {
			t.rowsFooterConfigMap = make(map[int]RowConfig)
		}
		t.rowsFooterConfigMap[len(t.rowsFooterRaw)-1] = config[0]
	}
}

// AppendHeader appends the row to the List of headers to render.
//
// Only the first item in the "config" will be tagged against this row.
func (t *Table) AppendHeader(row Row, config ...RowConfig) {
	t.rowsHeaderRaw = append(t.rowsHeaderRaw, row)
	if len(config) > 0 {
		if t.rowsHeaderConfigMap == nil {
			t.rowsHeaderConfigMap = make(map[int]RowConfig)
		}
		t.rowsHeaderConfigMap[len(t.rowsHeaderRaw)-1] = config[0]
	}
}

// AppendRow appends the row to the List of rows to render.
//
// Only the first item in the "config" will be tagged against this row.
func (t *Table) AppendRow(row Row, config ...RowConfig) {
	t.rowsRaw = append(t.rowsRaw, row)
	if len(config) > 0 {
		if t.rowsConfigMap == nil {
			t.rowsConfigMap = make(map[int]RowConfig)
		}
		t.rowsConfigMap[len(t.rowsRaw)-1] = config[0]
	}
}

// AppendRows appends the rows to the List of rows to render.
//
// Only the first item in the "config" will be tagged against all the rows.
func (t *Table) AppendRows(rows []Row, config ...RowConfig) {
	for _, row := range rows {
		t.AppendRow(row, config...)
	}
}

// AppendSeparator helps render a separator row after the current last row. You
// could call this function over and over, but it will be a no-op unless you
// call AppendRow or AppendRows in between. Likewise, if the last thing you
// append is a separator, it will not be rendered in addition to the usual table
// separator.
//
// ******************************************************************************
// Please note the following caveats:
//  1. SetPageSize(): this may end up creating consecutive separator rows near
//     the end of a page or at the beginning of a page
//  2. SortBy(): since SortBy could inherently alter the ordering of rows, the
//     separators may not appear after the row it was originally intended to
//     follow
//
// ******************************************************************************
func (t *Table) AppendSeparator() {
	if t.separators == nil {
		t.separators = make(map[int]bool)
	}
	if len(t.rowsRaw) > 0 {
		t.separators[len(t.rowsRaw)-1] = true
	}
}

// ImportGrid helps import 1d or 2d arrays as rows.
func (t *Table) ImportGrid(grid interface{}) bool {
	rows := objAsSlice(grid)
	if rows == nil {
		return false
	}
	addedRows := false
	for _, row := range rows {
		rowAsSlice := objAsSlice(row)
		if rowAsSlice != nil {
			t.AppendRow(rowAsSlice)
		} else if row != nil {
			t.AppendRow(Row{row})
		}
		addedRows = true
	}
	return addedRows
}

// Length returns the number of rows to be rendered.
func (t *Table) Length() int {
	return len(t.rowsRaw)
}

// Pager returns an object that splits the table output into pages and
// lets you move back and forth through them.
func (t *Table) Pager(opts ...PagerOption) Pager {
	for _, opt := range opts {
		opt(t)
	}

	// use a temporary page separator for splitting up the pages
	tempPageSep := fmt.Sprintf("%p // page separator // %d", t.rows, time.Now().UnixNano())

	// backup
	origOutputMirror, origPageSep := t.outputMirror, t.Style().Box.PageSeparator
	// restore on exit
	defer func() {
		t.outputMirror = origOutputMirror
		t.Style().Box.PageSeparator = origPageSep
	}()
	// override
	t.outputMirror = nil
	t.Style().Box.PageSeparator = tempPageSep
	// render
	t.pager.pages = strings.Split(t.Render(), tempPageSep)

	return &t.pager
}

// ResetFooters resets and clears all the Footer rows appended earlier.
func (t *Table) ResetFooters() {
	t.rowsFooterRaw = nil
}

// ResetHeaders resets and clears all the Header rows appended earlier.
func (t *Table) ResetHeaders() {
	t.rowsHeaderRaw = nil
}

// ResetRows resets and clears all the rows appended earlier.
func (t *Table) ResetRows() {
	t.rowsRaw = nil
	t.separators = nil
}

// SetAllowedRowLength sets the maximum allowed length or a row (or line of
// output) when rendered as a table. Rows that are longer than this limit will
// be "snipped" to the length. Length has to be a positive value to take effect.
//
// Deprecated: in favor if Style().Size.WidthMax
func (t *Table) SetAllowedRowLength(length int) {
	t.allowedRowLength = length
}

// SetAutoIndex adds a generated header with columns such as "A", "B", "C", etc.
// and a leading column with the row number similar to what you'd see on any
// spreadsheet application. NOTE: Appending a Header will void this
// functionality.
func (t *Table) SetAutoIndex(autoIndex bool) {
	t.autoIndex = autoIndex
}

// SetCaption sets the text to be rendered just below the table. This will not
// show up when the Table is rendered as a CSV.
func (t *Table) SetCaption(format string, a ...interface{}) {
	t.caption = fmt.Sprintf(format, a...)
}

// SetColumnConfigs sets the configs for each Column.
func (t *Table) SetColumnConfigs(configs []ColumnConfig) {
	t.columnConfigs = configs
}

// SetHTMLCSSClass sets the HTML CSS Class to use on the <table> node
// when rendering the Table in HTML format.
//
// Deprecated: in favor of Style().HTML.CSSClass
func (t *Table) SetHTMLCSSClass(cssClass string) {
	t.htmlCSSClass = cssClass
}

// SetIndexColumn sets the given Column # as the column that has the row
// "Number". Valid values range from 1 to N. Note that this is not 0-indexed.
func (t *Table) SetIndexColumn(colNum int) {
	t.indexColumn = colNum
}

// SetOutputMirror sets an io.Writer for all the Render functions to "Write" to
// in addition to returning a string.
func (t *Table) SetOutputMirror(mirror io.Writer) {
	t.outputMirror = mirror
	t.pager.SetOutputMirror(mirror)
}

// SetPageSize sets the maximum number of lines to render before rendering the
// header rows again. This can be useful when dealing with tables containing a
// long list of rows that can span pages. Please note that the pagination logic
// will not consider Header/Footer lines for paging.
func (t *Table) SetPageSize(numLines int) {
	t.pager.size = numLines
}

// SetRowPainter sets up the function which determines the colors to use on a
// row. Before rendering, this function is invoked on all rows and the color
// of each row is determined. This color takes precedence over other ways to
// set color (ColumnConfig.Color*, SetColor*()).
func (t *Table) SetRowPainter(painter interface{}) {
	// TODO: fix interface on major version bump to accept only
	// one type of RowPainter: RowPainterWithAttributes renamed to RowPainter

	// reset both so only one is set at any given time
	t.rowPainter = nil
	t.rowPainterWithAttributes = nil

	// if called as SetRowPainter(RowPainter(func...))
	switch painter.(type) {
	case RowPainter:
		t.rowPainter = painter.(RowPainter)
		return
	case RowPainterWithAttributes:
		t.rowPainterWithAttributes = painter.(RowPainterWithAttributes)
		return
	}

	// if called as SetRowPainter(func...)
	switch fmt.Sprintf("%T", painter) {
	case "func(table.Row) text.Colors":
		t.rowPainter = painter.(func(row Row) text.Colors)
		return
	case "func(table.Row, table.RowAttributes) text.Colors":
		t.rowPainterWithAttributes = painter.(func(row Row, attr RowAttributes) text.Colors)
		return
	}
}

// SetStyle overrides the DefaultStyle with the provided one.
func (t *Table) SetStyle(style Style) {
	t.style = &style
}

// SetTitle sets the title text to be rendered above the table.
func (t *Table) SetTitle(format string, a ...interface{}) {
	t.title = fmt.Sprintf(format, a...)
}

// SortBy sets the rules for sorting the Rows in the order specified. i.e., the
// first SortBy instruction takes precedence over the second and so on. Any
// duplicate instructions on the same column will be discarded while sorting.
func (t *Table) SortBy(sortBy []SortBy) {
	t.sortBy = sortBy
}

// Style returns the current style.
func (t *Table) Style() *Style {
	if t.style == nil {
		tempStyle := StyleDefault
		t.style = &tempStyle
	}
	// override WidthMax with allowedRowLength until allowedRowLength is
	// removed from code
	if t.allowedRowLength > 0 {
		t.style.Size.WidthMax = t.allowedRowLength
	}
	return t.style
}

// SuppressEmptyColumns hides columns when the column is empty in ALL the
// regular rows.
func (t *Table) SuppressEmptyColumns() {
	t.suppressEmptyColumns = true
}

// SuppressTrailingSpaces removes all trailing spaces from the output.
func (t *Table) SuppressTrailingSpaces() {
	t.suppressTrailingSpaces = true
}

func (t *Table) getAlign(colIdx int, hint renderHint) text.Align {
	align := text.AlignDefault
	if cfg, ok := t.columnConfigMap[colIdx]; ok {
		if hint.isHeaderRow {
			align = cfg.AlignHeader
		} else if hint.isFooterRow {
			align = cfg.AlignFooter
		} else {
			align = cfg.Align
		}
	}
	if align == text.AlignDefault {
		if !t.columnIsNonNumeric[colIdx] {
			align = text.AlignRight
		} else if hint.isAutoIndexRow {
			align = text.AlignCenter
		} else if hint.isHeaderRow {
			align = t.style.Format.HeaderAlign
		} else if hint.isFooterRow {
			align = t.style.Format.FooterAlign
		} else {
			align = t.style.Format.RowAlign
		}
	}
	return align
}

func (t *Table) getAutoIndexColumnIDs() rowStr {
	row := make(rowStr, t.numColumns)
	for colIdx := range row {
		row[colIdx] = AutoIndexColumnID(colIdx)
	}
	return row
}

func (t *Table) getBorderColors(hint renderHint) text.Colors {
	if t.style.Options.DoNotColorBordersAndSeparators {
		return nil
	} else if t.style.Color.Border != nil {
		return t.style.Color.Border
	} else if hint.isTitleRow {
		return t.style.Title.Colors
	} else if hint.isHeaderRow {
		return t.style.Color.Header
	} else if hint.isFooterRow {
		return t.style.Color.Footer
	} else if t.autoIndex {
		return t.style.Color.IndexColumn
	} else if hint.rowNumber%2 == 0 && t.style.Color.RowAlternate != nil {
		return t.style.Color.RowAlternate
	}
	return t.style.Color.Row
}

func (t *Table) getBorderLeft(hint renderHint) string {
	border := t.style.Box.Left
	if hint.isBorderTop {
		if t.title != "" {
			border = t.style.Box.LeftSeparator
		} else {
			border = t.style.Box.TopLeft
		}
	} else if hint.isBorderBottom {
		border = t.style.Box.BottomLeft
	} else if hint.isSeparatorRow {
		if t.autoIndex && hint.isHeaderOrFooterSeparator() {
			border = t.style.Box.Left
		} else if !t.autoIndex && t.shouldMergeCellsVerticallyAbove(0, hint) {
			border = t.style.Box.Left
		} else {
			border = t.style.Box.LeftSeparator
		}
	}
	return border
}

func (t *Table) getBorderRight(hint renderHint) string {
	border := t.style.Box.Right
	if hint.isBorderTop {
		if t.title != "" {
			border = t.style.Box.RightSeparator
		} else {
			border = t.style.Box.TopRight
		}
	} else if hint.isBorderBottom {
		border = t.style.Box.BottomRight
	} else if hint.isSeparatorRow {
		if t.shouldMergeCellsVerticallyAbove(t.numColumns-1, hint) {
			border = t.style.Box.Right
		} else {
			border = t.style.Box.RightSeparator
		}
	}
	return border
}

func (t *Table) getColumnColors(colIdx int, hint renderHint) text.Colors {
	if hint.isBorderOrSeparator() {
		if colors := t.getColumnColorsForBorderOrSeparator(hint); colors != nil {
			return colors
		}
	}
	if t.hasRowPainter() && hint.isRegularNonSeparatorRow() && !t.isIndexColumn(colIdx, hint) {
		if colors := t.rowsColors[hint.rowNumber-1]; colors != nil {
			return colors
		}
	}
	if cfg, ok := t.columnConfigMap[colIdx]; ok {
		if hint.isSeparatorRow {
			return nil
		} else if hint.isHeaderRow {
			return cfg.ColorsHeader
		} else if hint.isFooterRow {
			return cfg.ColorsFooter
		}
		return cfg.Colors
	}
	return nil
}

func (t *Table) getColumnColorsForBorderOrSeparator(hint renderHint) text.Colors {
	if t.style.Options.DoNotColorBordersAndSeparators {
		return text.Colors{} // not nil to force caller to paint with no colors
	}
	if (hint.isBorderBottom || hint.isBorderTop) && t.style.Color.Border != nil {
		return t.style.Color.Border
	}
	if hint.isSeparatorRow && t.style.Color.Separator != nil {
		return t.style.Color.Separator
	}
	return nil
}

func (t *Table) getColumnSeparator(row rowStr, colIdx int, hint renderHint) string {
	separator := t.style.Box.MiddleVertical
	if hint.isSeparatorRow {
		if hint.isBorderTop {
			if t.shouldMergeCellsHorizontallyBelow(row, colIdx, hint) {
				separator = t.style.Box.MiddleHorizontal
			} else {
				separator = t.style.Box.TopSeparator
			}
		} else if hint.isBorderBottom {
			if t.shouldMergeCellsHorizontallyAbove(row, colIdx, hint) {
				separator = t.style.Box.MiddleHorizontal
			} else {
				separator = t.style.Box.BottomSeparator
			}
		} else {
			sm1 := t.shouldMergeCellsHorizontallyAbove(row, colIdx, hint)
			sm2 := t.shouldMergeCellsHorizontallyBelow(row, colIdx, hint)
			separator = t.getColumnSeparatorNonBorder(sm1, sm2, colIdx, hint)
		}
	}
	return separator
}

func (t *Table) getColumnSeparatorNonBorder(mergeCellsAbove bool, mergeCellsBelow bool, colIdx int, hint renderHint) string {
	mergeNextCol := t.shouldMergeCellsVerticallyAbove(colIdx, hint)
	if hint.isAutoIndexColumn {
		return t.getColumnSeparatorNonBorderAutoIndex(mergeNextCol, hint)
	}

	mergeCurrCol := t.shouldMergeCellsVerticallyAbove(colIdx-1, hint)
	return t.getColumnSeparatorNonBorderNonAutoIndex(mergeCellsAbove, mergeCellsBelow, mergeCurrCol, mergeNextCol)
}

func (t *Table) getColumnSeparatorNonBorderAutoIndex(mergeNextCol bool, hint renderHint) string {
	if hint.isHeaderOrFooterSeparator() {
		if mergeNextCol {
			return t.style.Box.MiddleVertical
		}
		return t.style.Box.LeftSeparator
	} else if mergeNextCol {
		return t.style.Box.RightSeparator
	}
	return t.style.Box.MiddleSeparator
}

func (t *Table) getColumnSeparatorNonBorderNonAutoIndex(mergeCellsAbove bool, mergeCellsBelow bool, mergeCurrCol bool, mergeNextCol bool) string {
	if mergeCellsAbove && mergeCellsBelow && mergeCurrCol && mergeNextCol {
		return t.style.Box.EmptySeparator
	} else if mergeCellsAbove && mergeCellsBelow {
		return t.style.Box.MiddleHorizontal
	} else if mergeCellsAbove {
		return t.style.Box.TopSeparator
	} else if mergeCellsBelow {
		return t.style.Box.BottomSeparator
	} else if mergeCurrCol && mergeNextCol {
		return t.style.Box.MiddleVertical
	} else if mergeCurrCol {
		return t.style.Box.LeftSeparator
	} else if mergeNextCol {
		return t.style.Box.RightSeparator
	}
	return t.style.Box.MiddleSeparator
}

func (t *Table) getColumnTransformer(colIdx int, hint renderHint) text.Transformer {
	var transformer text.Transformer
	if cfg, ok := t.columnConfigMap[colIdx]; ok {
		if hint.isHeaderRow {
			transformer = cfg.TransformerHeader
		} else if hint.isFooterRow {
			transformer = cfg.TransformerFooter
		} else {
			transformer = cfg.Transformer
		}
	}
	return transformer
}

func (t *Table) getColumnWidthMax(colIdx int) int {
	if cfg, ok := t.columnConfigMap[colIdx]; ok {
		return cfg.WidthMax
	}
	return 0
}

func (t *Table) getColumnWidthMin(colIdx int) int {
	if cfg, ok := t.columnConfigMap[colIdx]; ok {
		return cfg.WidthMin
	}
	return 0
}

func (t *Table) getFormat(hint renderHint) text.Format {
	if hint.isSeparatorRow {
		return text.FormatDefault
	} else if hint.isHeaderRow {
		return t.style.Format.Header
	} else if hint.isFooterRow {
		return t.style.Format.Footer
	}
	return t.style.Format.Row
}

func (t *Table) getMaxColumnLengthForMerging(colIdx int) int {
	maxColumnLength := t.maxColumnLengths[colIdx]
	maxColumnLength += text.StringWidthWithoutEscSequences(t.style.Box.PaddingRight + t.style.Box.PaddingLeft)
	if t.style.Options.SeparateColumns {
		maxColumnLength += text.StringWidthWithoutEscSequences(t.style.Box.EmptySeparator)
	}
	return maxColumnLength
}

// getMergedColumnIndices returns a map of colIdx values to all the other colIdx
// values (that are being merged) and their lengths.
func (t *Table) getMergedColumnIndices(row rowStr, hint renderHint) mergedColumnIndices {
	if !t.getRowConfig(hint).AutoMerge {
		return nil
	}

	mci := make(mergedColumnIndices)
	for colIdx := 0; colIdx < t.numColumns-1; colIdx++ {
		for otherColIdx := colIdx + 1; otherColIdx < len(row); otherColIdx++ {
			colsEqual := row[colIdx] == row[otherColIdx]
			if !colsEqual {
				lastEqual := otherColIdx - 1
				if colIdx != lastEqual {
					mci[colIdx] = lastEqual
					colIdx = lastEqual
				}
				break
			} else if colsEqual && otherColIdx == len(row)-1 {
				mci[colIdx] = otherColIdx
				colIdx = otherColIdx
			}
		}
	}
	return mci
}

func (t *Table) getRow(rowIdx int, hint renderHint) rowStr {
	switch {
	case hint.isHeaderRow:
		if rowIdx >= 0 && rowIdx < len(t.rowsHeader) {
			return t.rowsHeader[rowIdx]
		}
	case hint.isFooterRow:
		if rowIdx >= 0 && rowIdx < len(t.rowsFooter) {
			return t.rowsFooter[rowIdx]
		}
	default:
		if rowIdx >= 0 && rowIdx < len(t.rows) {
			return t.rows[rowIdx]
		}
	}
	return rowStr{}
}

func (t *Table) getRowConfig(hint renderHint) RowConfig {
	rowIdx := hint.rowNumber - 1
	if rowIdx < 0 {
		rowIdx = 0
	}

	switch {
	case hint.isHeaderRow:
		return t.rowsHeaderConfigMap[rowIdx]
	case hint.isFooterRow:
		return t.rowsFooterConfigMap[rowIdx]
	default:
		return t.rowsConfigMap[rowIdx]
	}
}

func (t *Table) getSeparatorColors(hint renderHint) text.Colors {
	if t.style.Options.DoNotColorBordersAndSeparators {
		return nil
	} else if (hint.isBorderBottom || hint.isBorderTop) && t.style.Color.Border != nil {
		return t.style.Color.Border
	} else if t.style.Color.Separator != nil {
		return t.style.Color.Separator
	} else if hint.isHeaderRow {
		return t.style.Color.Header
	} else if hint.isFooterRow {
		return t.style.Color.Footer
	} else if hint.isAutoIndexColumn {
		return t.style.Color.IndexColumn
	} else if hint.rowNumber > 0 && hint.rowNumber%2 == 0 {
		return t.style.Color.RowAlternate
	}
	return t.style.Color.Row
}

func (t *Table) getVAlign(colIdx int, hint renderHint) text.VAlign {
	vAlign := text.VAlignDefault
	if cfg, ok := t.columnConfigMap[colIdx]; ok {
		if hint.isHeaderRow {
			vAlign = cfg.VAlignHeader
		} else if hint.isFooterRow {
			vAlign = cfg.VAlignFooter
		} else {
			vAlign = cfg.VAlign
		}
	}
	if vAlign == text.VAlignDefault {
		if hint.isHeaderRow {
			vAlign = t.style.Format.HeaderVAlign
		} else if hint.isFooterRow {
			vAlign = t.style.Format.FooterVAlign
		} else {
			vAlign = t.style.Format.RowVAlign
		}
	}
	return vAlign
}

func (t *Table) hasHiddenColumns() bool {
	for _, cc := range t.columnConfigMap {
		if cc.Hidden {
			return true
		}
	}
	return false
}

func (t *Table) hasRowPainter() bool {
	return t.rowPainter != nil || t.rowPainterWithAttributes != nil
}

func (t *Table) hideColumns() map[int]int {
	colIdxMap := make(map[int]int)
	numColumns := 0
	hideColumnsInRows := func(rows []rowStr) []rowStr {
		var rsp []rowStr
		for _, row := range rows {
			var rowNew rowStr
			for colIdx, col := range row {
				cc := t.columnConfigMap[colIdx]
				if !cc.Hidden {
					rowNew = append(rowNew, col)
					colIdxMap[colIdx] = len(rowNew) - 1
				}
			}
			if len(rowNew) > numColumns {
				numColumns = len(rowNew)
			}
			rsp = append(rsp, rowNew)
		}
		return rsp
	}

	// hide columns as directed
	t.rows = hideColumnsInRows(t.rows)
	t.rowsFooter = hideColumnsInRows(t.rowsFooter)
	t.rowsHeader = hideColumnsInRows(t.rowsHeader)

	// reset numColumns to the new number of columns
	t.numColumns = numColumns

	return colIdxMap
}

func (t *Table) isIndexColumn(colIdx int, hint renderHint) bool {
	return t.indexColumn == colIdx+1 || hint.isAutoIndexColumn
}

func (t *Table) render(out *strings.Builder) string {
	outStr := out.String()
	if t.suppressTrailingSpaces {
		var trimmed []string
		for _, line := range strings.Split(outStr, "\n") {
			trimmed = append(trimmed, strings.TrimRightFunc(line, unicode.IsSpace))
		}
		outStr = strings.Join(trimmed, "\n")
	}
	if t.outputMirror != nil && len(outStr) > 0 {
		_, _ = t.outputMirror.Write([]byte(outStr))
		_, _ = t.outputMirror.Write([]byte("\n"))
	}
	return outStr
}

func (t *Table) shouldMergeCellsHorizontallyAbove(row rowStr, colIdx int, hint renderHint) bool {
	if hint.isAutoIndexColumn || hint.isAutoIndexRow {
		return false
	}

	rowConfig := t.getRowConfig(hint)
	if hint.isSeparatorRow {
		if hint.isHeaderRow && hint.rowNumber == 1 {
			rowConfig = t.getRowConfig(hint)
			row = t.getRow(hint.rowNumber-1, hint)
		} else if hint.isFooterRow && hint.isFirstRow {
			rowConfig = t.getRowConfig(renderHint{isLastRow: true, rowNumber: len(t.rows)})
			row = t.getRow(len(t.rows)-1, renderHint{})
		} else if hint.isFooterRow && hint.isBorderBottom {
			row = t.getRow(len(t.rowsFooter)-1, renderHint{isFooterRow: true})
		} else {
			row = t.getRow(hint.rowNumber-1, hint)
		}
	}

	if rowConfig.AutoMerge {
		return row.areEqual(colIdx-1, colIdx)
	}
	return false
}

func (t *Table) shouldMergeCellsHorizontallyBelow(row rowStr, colIdx int, hint renderHint) bool {
	if hint.isAutoIndexColumn || hint.isAutoIndexRow {
		return false
	}

	var rowConfig RowConfig
	if hint.isSeparatorRow {
		if hint.isRegularRow() {
			rowConfig = t.getRowConfig(renderHint{rowNumber: hint.rowNumber + 1})
			row = t.getRow(hint.rowNumber, renderHint{})
		} else if hint.isHeaderRow && hint.rowNumber == 0 {
			rowConfig = t.getRowConfig(renderHint{isHeaderRow: true, rowNumber: 1})
			row = t.getRow(0, hint)
		} else if hint.isHeaderRow && hint.isLastRow {
			rowConfig = t.getRowConfig(renderHint{rowNumber: 1})
			row = t.getRow(0, renderHint{})
		} else if hint.isHeaderRow {
			rowConfig = t.getRowConfig(renderHint{isHeaderRow: true, rowNumber: hint.rowNumber + 1})
			row = t.getRow(hint.rowNumber, hint)
		} else if hint.isFooterRow && hint.rowNumber >= 0 {
			rowConfig = t.getRowConfig(renderHint{isFooterRow: true, rowNumber: 1})
			row = t.getRow(hint.rowNumber, renderHint{isFooterRow: true})
		}
	}

	if rowConfig.AutoMerge {
		return row.areEqual(colIdx-1, colIdx)
	}
	return false
}

func (t *Table) shouldMergeCellsVerticallyAbove(colIdx int, hint renderHint) bool {
	if !t.firstRowOfPage && t.columnConfigMap[colIdx].AutoMerge && colIdx < t.numColumns {
		if hint.isSeparatorRow {
			rowPrev := t.getRow(hint.rowNumber-1, hint)
			rowNext := t.getRow(hint.rowNumber, hint)
			if colIdx < len(rowPrev) && colIdx < len(rowNext) {
				return rowPrev[colIdx] == rowNext[colIdx]
			}
		} else {
			rowPrev := t.getRow(hint.rowNumber-2, hint)
			rowCurr := t.getRow(hint.rowNumber-1, hint)
			if colIdx < len(rowPrev) && colIdx < len(rowCurr) {
				return rowPrev[colIdx] == rowCurr[colIdx]
			}
		}
	}
	return false
}

func (t *Table) shouldMergeCellsVerticallyBelow(colIdx int, hint renderHint) int {
	numRowsToMerge := 0
	if t.columnConfigMap[colIdx].AutoMerge && colIdx < t.numColumns {
		numRowsToMerge = 1
		rowCurr := t.getRow(hint.rowNumber-1, hint)
		for rowIdx := hint.rowNumber; rowIdx < len(t.rows); rowIdx++ {
			rowNext := t.getRow(rowIdx, hint)
			if colIdx < len(rowCurr) && colIdx < len(rowNext) && rowNext[colIdx] == rowCurr[colIdx] {
				numRowsToMerge++
			} else {
				break
			}
		}
	}
	return numRowsToMerge
}

func (t *Table) shouldSeparateRows(rowIdx int, numRows int) bool {
	// not asked to separate rows and no manually added separator
	if !t.style.Options.SeparateRows && !t.separators[rowIdx] {
		return false
	}

	pageSize := numRows
	if t.pager.size > 0 {
		pageSize = t.pager.size
	}
	if rowIdx%pageSize == pageSize-1 { // last row of page
		return false
	}
	if rowIdx == numRows-1 { // last row of table
		return false
	}
	return true
}

func (t *Table) wrapRow(row rowStr) (int, rowStr) {
	colMaxLines := 0
	rowWrapped := make(rowStr, len(row))
	for colIdx, colStr := range row {
		widthEnforcer := t.columnConfigMap[colIdx].getWidthMaxEnforcer()
		maxWidth := t.getColumnWidthMax(colIdx)
		if maxWidth == 0 {
			maxWidth = t.maxColumnLengths[colIdx]
		}
		rowWrapped[colIdx] = widthEnforcer(colStr, maxWidth)
		colNumLines := strings.Count(rowWrapped[colIdx], "\n") + 1
		if colNumLines > colMaxLines {
			colMaxLines = colNumLines
		}
	}
	return colMaxLines, rowWrapped
}
