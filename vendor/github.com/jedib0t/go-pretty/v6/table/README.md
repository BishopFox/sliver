# Table
[![Go Reference](https://pkg.go.dev/badge/github.com/jedib0t/go-pretty/v6/table.svg)](https://pkg.go.dev/github.com/jedib0t/go-pretty/v6/table)

Pretty-print tables into ASCII/Unicode strings.

## Sample Table Rendering

```
+---------------------------------------------------------------------+
| Game of Thrones                                                     +
+-----+------------+-----------+--------+-----------------------------+
|   # | FIRST NAME | LAST NAME | SALARY |                             |
+-----+------------+-----------+--------+-----------------------------+
|   1 | Arya       | Stark     |   3000 |                             |
|  20 | Jon        | Snow      |   2000 | You know nothing, Jon Snow! |
| 300 | Tyrion     | Lannister |   5000 |                             |
+-----+------------+-----------+--------+-----------------------------+
|     |            | TOTAL     |  10000 |                             |
+-----+------------+-----------+--------+-----------------------------+
```

A demonstration of all the capabilities can be found here:
[../cmd/demo-table](../cmd/demo-table)

If you want very specific examples, look at the [EXAMPLES.md](EXAMPLES.md) file.

## Features

### Core Table Building

  - Add Rows one-by-one or as a group (`AppendRow`/`AppendRows`)
  - Add Header(s) and Footer(s) (`AppendHeader`/`AppendFooter`)
  - Add a Separator manually after any Row (`AppendSeparator`)
  - Add Title above the table (`SetTitle`)
  - Add Caption below the table (`SetCaption`)
  - Import 1D or 2D arrays/grids as rows (`ImportGrid`)
  - Reset Headers/Rows/Footers at will to reuse the same Table Writer (`Reset*`)

### Indexing & Navigation

  - Auto Index Rows (1, 2, 3 ...) and Columns (A, B, C, ...) (`SetAutoIndex`)
  - Set which column is the index column (`SetIndexColumn`)
  - Pager interface for navigating through paged output (`Pager()`)
    - `GoTo(pageNum)` - Jump to specific page
    - `Next()` - Move to next page
    - `Prev()` - Move to previous page
    - `Location()` - Get current page number
    - `Render()` - Render current page
    - `SetOutputMirror()` - Mirror output to io.Writer

### Auto Merge

  - Auto Merge cells (_not supported in CSV/Markdown/TSV modes_)
    - Cells in a Row (`RowConfig.AutoMerge`)
    - Columns (`ColumnConfig.AutoMerge`) (_not supported in HTML mode_)
    - Custom alignment for merged cells (`RowConfig.AutoMergeAlign`)

### Size & Width Control

  - Limit the length of Rows (`SetAllowedRowLength` or `Style().Size.WidthMax`)
  - Auto-size Rows (`Style().Size.WidthMin` and `Style().Size.WidthMax`)
  - Column width control (`ColumnConfig.WidthMin` and `ColumnConfig.WidthMax`)
  - Custom width enforcement functions (`ColumnConfig.WidthMaxEnforcer`)
    - Default: `text.WrapText`
    - Options: `text.WrapSoft`, `text.WrapHard`, `text.Trim`, or custom function

### Alignment

  - **Horizontal Alignment**
    - Auto (numeric columns aligned Right, text aligned Left)
    - Custom per column (`ColumnConfig.Align`, `AlignHeader`, `AlignFooter`)
    - Options: Left, Center, Right, Justify, Auto
  - **Vertical Alignment**
    - Custom per column with multi-line cell support (`ColumnConfig.VAlign`, `VAlignHeader`, `VAlignFooter`)
    - Options: Top, Middle, Bottom

### Sorting & Filtering

  - **Sorting**
    - Sort by one or more Columns (`SortBy`)
    - Multiple column sorting support
    - Various sort modes: Alphabetical, Numeric, Alpha-numeric, Numeric-alpha
    - Case-insensitive sorting option (`IgnoreCase`)
    - Custom sorting functions (`CustomLess`) for advanced sorting logic
  - **Filtering**
    - Filter by one or more Columns (`FilterBy`)
    - Multiple filters with AND logic (all must match)
    - Various filter operators:
      - Equality: Equal, NotEqual
      - Numeric: GreaterThan, GreaterThanOrEqual, LessThan, LessThanOrEqual
      - String: Contains, NotContains, StartsWith, EndsWith
      - Regex: RegexMatch, RegexNotMatch
    - Case-insensitive filtering option (`IgnoreCase`)
    - Custom filter functions (`CustomFilter`) for advanced filtering logic
    - Filters are applied before sorting
  - Suppress/hide columns with no content (`SuppressEmptyColumns`)
  - Hide specific columns (`ColumnConfig.Hidden`)
  - Suppress trailing spaces in the last column (`SuppressTrailingSpaces`)

### Customization & Styling

  - **Row Coloring**
    - Custom row painter function (`SetRowPainter`)
    - Row painter with attributes (`RowPainterWithAttributes`)
    - Access to row number and sorted position
  - **Cell Transformation**
    - Customizable Cell rendering per Column (`ColumnConfig.Transformer`, `TransformerHeader`, `TransformerFooter`)
    - Use built-in transformers from `text` package (Number, JSON, Time, URL, etc.)
  - **Column Styling**
    - Per-column colors (`ColumnConfig.Colors`, `ColorsHeader`, `ColorsFooter`)
    - Per-column alignment (horizontal and vertical)
    - Per-column width constraints
  - **Completely customizable styles** (`SetStyle`/`Style`)
    - Many ready-to-use styles: [style.go](style.go)
      - `StyleDefault` - Classic ASCII borders
      - `StyleLight` - Light box-drawing characters
      - `StyleBold` - Bold box-drawing characters
      - `StyleDouble` - Double box-drawing characters
      - `StyleRounded` - Rounded box-drawing characters
      - `StyleColoredBright` - Bright colors, no borders
      - `StyleColoredDark` - Dark colors, no borders
      - Many more colored variants (Blue, Cyan, Green, Magenta, Red, Yellow)
    - Colorize Headers/Body/Footers using [../text/color.go](../text/color.go)
    - Custom text-case for Headers/Body/Footers
    - Enable/disable separators between rows
    - Render table with or without borders
    - Customize box-drawing characters
      - Horizontal separators per section (title, header, rows, footer) using `BoxStyleHorizontal`
    - Title and caption styling options
    - HTML rendering options (CSS class, escaping, newlines, color conversion)
    - Bidirectional text support (`Style().Format.Direction`)

### Output Formats

  - **Render as:**
    - (ASCII/Unicode) Table - Human-readable pretty format
    - CSV - Comma-separated values
    - HTML Table - With custom CSS Class and options
    - Markdown Table - Markdown-compatible format
    - TSV - Tab-separated values
  - Mirror output to an `io.Writer` (ex. `os.StdOut`) (`SetOutputMirror`)
