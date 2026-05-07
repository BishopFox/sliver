package table

import "github.com/jedib0t/go-pretty/v6/text"

// FormatOptions defines the text-formatting to perform on parts of the Table.
type FormatOptions struct {
	Direction    text.Direction // (forced) BiDi direction for each Column
	Footer       text.Format    // default text format
	FooterAlign  text.Align     // default horizontal align
	FooterVAlign text.VAlign    // default vertical align
	Header       text.Format    // default text format
	HeaderAlign  text.Align     // default horizontal align
	HeaderVAlign text.VAlign    // default vertical align
	Row          text.Format    // default text format
	RowAlign     text.Align     // default horizontal align
	RowVAlign    text.VAlign    // default vertical align
}

var (
	// FormatOptionsDefault defines sensible formatting options.
	FormatOptionsDefault = FormatOptions{
		Footer:       text.FormatUpper,
		FooterAlign:  text.AlignDefault,
		FooterVAlign: text.VAlignDefault,
		Header:       text.FormatUpper,
		HeaderAlign:  text.AlignDefault,
		HeaderVAlign: text.VAlignDefault,
		Row:          text.FormatDefault,
		RowAlign:     text.AlignDefault,
		RowVAlign:    text.VAlignDefault,
	}
)
