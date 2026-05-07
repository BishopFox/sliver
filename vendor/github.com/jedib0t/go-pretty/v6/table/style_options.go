package table

// Options defines the global options that determine how the Table is
// rendered.
type Options struct {
	// DoNotColorBordersAndSeparators disables coloring all the borders and row
	// or column separators.
	DoNotColorBordersAndSeparators bool

	// DoNotRenderSeparatorWhenEmpty disables rendering the separator row after
	// headers when there are no data rows (for example when only headers and/or
	// footers are present).
	DoNotRenderSeparatorWhenEmpty bool

	// DrawBorder enables or disables drawing the border around the Table.
	// Example of a table where it is disabled:
	//     # │ FIRST NAME │ LAST NAME │ SALARY │
	//  ─────┼────────────┼───────────┼────────┼─────────────────────────────
	//     1 │ Arya       │ Stark     │   3000 │
	//    20 │ Jon        │ Snow      │   2000 │ You know nothing, Jon Snow!
	//   300 │ Tyrion     │ Lannister │   5000 │
	//  ─────┼────────────┼───────────┼────────┼─────────────────────────────
	//       │            │ TOTAL     │  10000 │
	DrawBorder bool

	// SeparateColumns enables or disable drawing border between columns.
	// Example of a table where it is disabled:
	//  ┌─────────────────────────────────────────────────────────────────┐
	//  │   #  FIRST NAME  LAST NAME  SALARY                              │
	//  ├─────────────────────────────────────────────────────────────────┤
	//  │   1  Arya        Stark        3000                              │
	//  │  20  Jon         Snow         2000  You know nothing, Jon Snow! │
	//  │ 300  Tyrion      Lannister    5000                              │
	//  │                  TOTAL       10000                              │
	//  └─────────────────────────────────────────────────────────────────┘
	SeparateColumns bool

	// SeparateFooter enables or disable drawing border between the footer and
	// the rows. Example of a table where it is disabled:
	//  ┌─────┬────────────┬───────────┬────────┬─────────────────────────────┐
	//  │   # │ FIRST NAME │ LAST NAME │ SALARY │                             │
	//  ├─────┼────────────┼───────────┼────────┼─────────────────────────────┤
	//  │   1 │ Arya       │ Stark     │   3000 │                             │
	//  │  20 │ Jon        │ Snow      │   2000 │ You know nothing, Jon Snow! │
	//  │ 300 │ Tyrion     │ Lannister │   5000 │                             │
	//  │     │            │ TOTAL     │  10000 │                             │
	//  └─────┴────────────┴───────────┴────────┴─────────────────────────────┘
	SeparateFooter bool

	// SeparateHeader enables or disable drawing border between the header and
	// the rows. Example of a table where it is disabled:
	//  ┌─────┬────────────┬───────────┬────────┬─────────────────────────────┐
	//  │   # │ FIRST NAME │ LAST NAME │ SALARY │                             │
	//  │   1 │ Arya       │ Stark     │   3000 │                             │
	//  │  20 │ Jon        │ Snow      │   2000 │ You know nothing, Jon Snow! │
	//  │ 300 │ Tyrion     │ Lannister │   5000 │                             │
	//  ├─────┼────────────┼───────────┼────────┼─────────────────────────────┤
	//  │     │            │ TOTAL     │  10000 │                             │
	//  └─────┴────────────┴───────────┴────────┴─────────────────────────────┘
	SeparateHeader bool

	// SeparateRows enables or disables drawing separators between each row.
	// Example of a table where it is enabled:
	//  ┌─────┬────────────┬───────────┬────────┬─────────────────────────────┐
	//  │   # │ FIRST NAME │ LAST NAME │ SALARY │                             │
	//  ├─────┼────────────┼───────────┼────────┼─────────────────────────────┤
	//  │   1 │ Arya       │ Stark     │   3000 │                             │
	//  ├─────┼────────────┼───────────┼────────┼─────────────────────────────┤
	//  │  20 │ Jon        │ Snow      │   2000 │ You know nothing, Jon Snow! │
	//  ├─────┼────────────┼───────────┼────────┼─────────────────────────────┤
	//  │ 300 │ Tyrion     │ Lannister │   5000 │                             │
	//  ├─────┼────────────┼───────────┼────────┼─────────────────────────────┤
	//  │     │            │ TOTAL     │  10000 │                             │
	//  └─────┴────────────┴───────────┴────────┴─────────────────────────────┘
	SeparateRows bool
}

var (
	// OptionsDefault defines sensible global options.
	OptionsDefault = Options{
		DoNotColorBordersAndSeparators: false,
		DrawBorder:                     true,
		SeparateColumns:                true,
		SeparateFooter:                 true,
		SeparateHeader:                 true,
		SeparateRows:                   false,
	}

	// OptionsNoBorders sets up a table without any borders.
	OptionsNoBorders = Options{
		DoNotColorBordersAndSeparators: false,
		DrawBorder:                     false,
		SeparateColumns:                true,
		SeparateFooter:                 true,
		SeparateHeader:                 true,
		SeparateRows:                   false,
	}

	// OptionsNoBordersAndSeparators sets up a table without any borders or
	// separators.
	OptionsNoBordersAndSeparators = Options{
		DoNotColorBordersAndSeparators: false,
		DrawBorder:                     false,
		SeparateColumns:                false,
		SeparateFooter:                 false,
		SeparateHeader:                 false,
		SeparateRows:                   false,
	}
)
