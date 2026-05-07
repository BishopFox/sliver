# Examples

All the examples below are going to start with the following block, although
nothing except a single Row is mandatory for the `Render()` function to render
something:
```golang
package main

import (
    "os"

    "github.com/jedib0t/go-pretty/v6/table"
)

func main() {
    t := table.NewWriter()
    t.SetOutputMirror(os.Stdout)
    t.AppendHeader(table.Row{"#", "First Name", "Last Name", "Salary"})
    t.AppendRows([]table.Row{
        {1, "Arya", "Stark", 3000},
        {20, "Jon", "Snow", 2000, "You know nothing, Jon Snow!"},
    })
    t.AppendSeparator()
    t.AppendRow([]interface{}{300, "Tyrion", "Lannister", 5000})
    t.AppendFooter(table.Row{"", "", "Total", 10000})
    t.Render()
}
```
Running the above will result in:
```
+-----+------------+-----------+--------+-----------------------------+
|   # | FIRST NAME | LAST NAME | SALARY |                             |
+-----+------------+-----------+--------+-----------------------------+
|   1 | Arya       | Stark     |   3000 |                             |
|  20 | Jon        | Snow      |   2000 | You know nothing, Jon Snow! |
+-----+------------+-----------+--------+-----------------------------+
| 300 | Tyrion     | Lannister |   5000 |                             |
+-----+------------+-----------+--------+-----------------------------+
|     |            | TOTAL     |  10000 |                             |
+-----+------------+-----------+--------+-----------------------------+
```

---

<details>
<summary><strong>Ready-to-use Styles</strong></summary>

Table comes with a bunch of ready-to-use Styles that make the table look really
good. Set or Change the style using:
```golang
    t.SetStyle(table.StyleLight)
    t.Render()
```
to get:
```
┌─────┬────────────┬───────────┬────────┬─────────────────────────────┐
│   # │ FIRST NAME │ LAST NAME │ SALARY │                             │
├─────┼────────────┼───────────┼────────┼─────────────────────────────┤
│   1 │ Arya       │ Stark     │   3000 │                             │
│  20 │ Jon        │ Snow      │   2000 │ You know nothing, Jon Snow! │
├─────┼────────────┼───────────┼────────┼─────────────────────────────┤
│ 300 │ Tyrion     │ Lannister │   5000 │                             │
├─────┼────────────┼───────────┼────────┼─────────────────────────────┤
│     │            │ TOTAL     │  10000 │                             │
└─────┴────────────┴───────────┴────────┴─────────────────────────────┘
```

Or if you want to use a full-color mode, and don't care for boxes, use:
```golang
    t.SetStyle(table.StyleColoredBright)
    t.Render()
```
to get:

<img src="images/table-StyleColoredBright.png" width="640px" alt="Colored Table"/>

</details>

---

<details>
<summary><strong>Roll your own Style</strong></summary>

You can also roll your own style:
```golang
    t.SetStyle(table.Style{
        Name: "myNewStyle",
        Box: table.BoxStyle{
            BottomLeft:       "\\",
            BottomRight:      "/",
            BottomSeparator:  "v",
            Left:             "[",
            LeftSeparator:    "{",
            MiddleHorizontal: "-",
            MiddleSeparator:  "+",
            MiddleVertical:   "|",
            PaddingLeft:      "<",
            PaddingRight:     ">",
            Right:            "]",
            RightSeparator:   "}",
            TopLeft:          "(",
            TopRight:         ")",
            TopSeparator:     "^",
            UnfinishedRow:    " ~~~",
        },
        Color: table.ColorOptions{
            IndexColumn:     text.Colors{text.BgCyan, text.FgBlack},
            Footer:          text.Colors{text.BgCyan, text.FgBlack},
            Header:          text.Colors{text.BgHiCyan, text.FgBlack},
            Row:             text.Colors{text.BgHiWhite, text.FgBlack},
            RowAlternate:    text.Colors{text.BgWhite, text.FgBlack},
        },
        Format: table.FormatOptions{
            Footer: text.FormatUpper,
            Header: text.FormatUpper,
            Row:    text.FormatDefault,
        },
        Options: table.Options{
            DrawBorder:      true,
            SeparateColumns: true,
            SeparateFooter:  true,
            SeparateHeader:  true,
            SeparateRows:    false,
        },
    })
```

Or you can use one of the ready-to-use Styles, and just make a few tweaks:
```golang
    t.SetStyle(table.StyleLight)
    t.Style().Color.Header = text.Colors{text.BgHiCyan, text.FgBlack}
    t.Style().Color.IndexColumn = text.Colors{text.BgHiCyan, text.FgBlack}
    t.Style().Format.Footer = text.FormatLower
    t.Style().Options.DrawBorder = false
```

</details>

---

<details>
<summary><strong>Customize Horizontal Separators</strong></summary>

For more granular control over horizontal lines in your table, you can use `BoxStyleHorizontal` to customize different separator types independently. This allows you to have different horizontal line styles for titles, headers, rows, and footers.

The `BoxStyleHorizontal` struct provides 10 customizable separator types:
- `TitleTop` / `TitleBottom` - Lines above/below the title
- `HeaderTop` / `HeaderMiddle` / `HeaderBottom` - Lines in the header section
- `RowTop` / `RowMiddle` / `RowBottom` - Lines in the data rows section
- `FooterTop` / `FooterMiddle` / `FooterBottom` - Lines in the footer section

You can customize each separator type using:

```golang
    tw := table.NewWriter()
    tw.AppendHeader(table.Row{"#", "First Name", "Last Name", "Salary"})
    tw.AppendRows([]table.Row{
        {1, "Arya", "Stark", 3000},
        {20, "Jon", "Snow", 2000, "You know nothing, Jon Snow!"},
        {300, "Tyrion", "Lannister", 5000},
    })
    tw.AppendFooter(table.Row{"", "", "Total", 10000})
    tw.SetStyle(table.StyleDefault)
    tw.Style().Box.Horizontal = &table.BoxStyleHorizontal{
        HeaderTop:    "=", // Thicker line above header
        HeaderMiddle: "-",
        HeaderBottom: "~", // Thicker line below header
        RowTop:       "-",
        RowMiddle:    "- ",
        RowBottom:    "-",
        FooterTop:    "~", // Thicker line above footer
        FooterMiddle: "-",
        FooterBottom: "=", // Thicker line below footer
    }
    tw.Style().Options.SeparateRows = true
    fmt.Println(tw.Render())
```
to get something like:
```
+=====+============+===========+========+=============================+
|   # | FIRST NAME | LAST NAME | SALARY |                             |
+~~~~~+~~~~~~~~~~~~+~~~~~~~~~~~+~~~~~~~~+~~~~~~~~~~~~~~~~~~~~~~~~~~~~~+
|   1 | Arya       | Stark     |   3000 |                             |
+- - -+- - - - - - +- - - - - -+- - - - +- - - - - - - - - - - - - - -+
|  20 | Jon        | Snow      |   2000 | You know nothing, Jon Snow! |
+- - -+- - - - - - +- - - - - -+- - - - +- - - - - - - - - - - - - - -+
| 300 | Tyrion     | Lannister |   5000 |                             |
+~~~~~+~~~~~~~~~~~~+~~~~~~~~~~~+~~~~~~~~+~~~~~~~~~~~~~~~~~~~~~~~~~~~~~+
|     |            | TOTAL     |  10000 |                             |
+=====+============+===========+========+=============================+
```

When `BoxStyle.Horizontal` is set to a non-nil value, it overrides the `MiddleHorizontal` string for all horizontal separators. If `Horizontal` is nil, the table will automatically use `MiddleHorizontal` for all separator types.

</details>

---

<details>
<summary><strong>Auto-Merge</strong></summary>

You can auto-merge cells horizontally and vertically, but you have request for
it specifically for each row/column using `RowConfig` or `ColumnConfig`.

```golang
    rowConfigAutoMerge := table.RowConfig{AutoMerge: true}

    t := table.NewWriter()
    t.AppendHeader(table.Row{"Node IP", "Pods", "Namespace", "Container", "RCE", "RCE"}, rowConfigAutoMerge)
    t.AppendHeader(table.Row{"Node IP", "Pods", "Namespace", "Container", "EXE", "RUN"})
    t.AppendRow(table.Row{"1.1.1.1", "Pod 1A", "NS 1A", "C 1", "Y", "Y"}, rowConfigAutoMerge)
    t.AppendRow(table.Row{"1.1.1.1", "Pod 1A", "NS 1A", "C 2", "Y", "N"}, rowConfigAutoMerge)
    t.AppendRow(table.Row{"1.1.1.1", "Pod 1A", "NS 1B", "C 3", "N", "N"}, rowConfigAutoMerge)
    t.AppendRow(table.Row{"1.1.1.1", "Pod 1B", "NS 2", "C 4", "N", "N"}, rowConfigAutoMerge)
    t.AppendRow(table.Row{"1.1.1.1", "Pod 1B", "NS 2", "C 5", "Y", "N"}, rowConfigAutoMerge)
    t.AppendRow(table.Row{"2.2.2.2", "Pod 2", "NS 3", "C 6", "Y", "Y"}, rowConfigAutoMerge)
    t.AppendRow(table.Row{"2.2.2.2", "Pod 2", "NS 3", "C 7", "Y", "Y"}, rowConfigAutoMerge)
    t.AppendFooter(table.Row{"", "", "", 7, 5, 3})
    t.SetAutoIndex(true)
    t.SetColumnConfigs([]table.ColumnConfig{
        {Number: 1, AutoMerge: true},
        {Number: 2, AutoMerge: true},
        {Number: 3, AutoMerge: true},
        {Number: 4, AutoMerge: true},
        {Number: 5, Align: text.AlignCenter, AlignFooter: text.AlignCenter, AlignHeader: text.AlignCenter},
        {Number: 6, Align: text.AlignCenter, AlignFooter: text.AlignCenter, AlignHeader: text.AlignCenter},
    })
    t.SetStyle(table.StyleLight)
    t.Style().Options.SeparateRows = true
    fmt.Println(t.Render())
```
to get:
```
┌───┬─────────┬────────┬───────────┬───────────┬───────────┐
│   │ NODE IP │ PODS   │ NAMESPACE │ CONTAINER │    RCE    │
│   │         │        │           │           ├─────┬─────┤
│   │         │        │           │           │ EXE │ RUN │
├───┼─────────┼────────┼───────────┼───────────┼─────┴─────┤
│ 1 │ 1.1.1.1 │ Pod 1A │ NS 1A     │ C 1       │     Y     │
├───┤         │        │           ├───────────┼─────┬─────┤
│ 2 │         │        │           │ C 2       │  Y  │  N  │
├───┤         │        ├───────────┼───────────┼─────┴─────┤
│ 3 │         │        │ NS 1B     │ C 3       │     N     │
├───┤         ├────────┼───────────┼───────────┼───────────┤
│ 4 │         │ Pod 1B │ NS 2      │ C 4       │     N     │
├───┤         │        │           ├───────────┼─────┬─────┤
│ 5 │         │        │           │ C 5       │  Y  │  N  │
├───┼─────────┼────────┼───────────┼───────────┼─────┴─────┤
│ 6 │ 2.2.2.2 │ Pod 2  │ NS 3      │ C 6       │     Y     │
├───┤         │        │           ├───────────┼───────────┤
│ 7 │         │        │           │ C 7       │     Y     │
├───┼─────────┼────────┼───────────┼───────────┼─────┬─────┤
│   │         │        │           │ 7         │  5  │  3  │
└───┴─────────┴────────┴───────────┴───────────┴─────┴─────┘
```

</details>

---

<details>
<summary><strong>Paging</strong></summary>

You can limit the number of lines rendered in a single "Page". This logic
can handle rows with multiple lines too. The recommended way is to use the
`Pager()` interface:

```golang
    pager := t.Pager(PageSize(1))
    pager.Render()  // Render first page
    pager.Next()    // Move to next page and render
    pager.Prev()    // Move to previous page and render
    pager.GoTo(3)   // Jump to page 3
    pager.Location() // Get current page number
```

Or use the deprecated `SetPageSize()` method for simple cases:
```golang
    t.SetPageSize(1)
    t.Render()
```
to get:
```
+-----+------------+-----------+--------+-----------------------------+
|   # | FIRST NAME | LAST NAME | SALARY |                             |
+-----+------------+-----------+--------+-----------------------------+
|   1 | Arya       | Stark     |   3000 |                             |
+-----+------------+-----------+--------+-----------------------------+
|     |            | TOTAL     |  10000 |                             |
+-----+------------+-----------+--------+-----------------------------+

+-----+------------+-----------+--------+-----------------------------+
|   # | FIRST NAME | LAST NAME | SALARY |                             |
+-----+------------+-----------+--------+-----------------------------+
|  20 | Jon        | Snow      |   2000 | You know nothing, Jon Snow! |
+-----+------------+-----------+--------+-----------------------------+
|     |            | TOTAL     |  10000 |                             |
+-----+------------+-----------+--------+-----------------------------+

+-----+------------+-----------+--------+-----------------------------+
|   # | FIRST NAME | LAST NAME | SALARY |                             |
+-----+------------+-----------+--------+-----------------------------+
| 300 | Tyrion     | Lannister |   5000 |                             |
+-----+------------+-----------+--------+-----------------------------+
|     |            | TOTAL     |  10000 |                             |
+-----+------------+-----------+--------+-----------------------------+
```

</details>

---

<details>
<summary><strong>Filtering</strong></summary>

Filtering can be done on one or more columns. All filters are applied with AND logic (all must match).
Filters are applied before sorting.

```golang
    t.FilterBy([]table.FilterBy{
        {Name: "Salary", Operator: table.GreaterThan, Value: 2000},
        {Name: "First Name", Operator: table.Contains, Value: "on"},
    })
```

The `Operator` field in `FilterBy` supports various filtering operators:
- `Equal` / `NotEqual` - Exact match
- `GreaterThan` / `GreaterThanOrEqual` - Numeric comparisons
- `LessThan` / `LessThanOrEqual` - Numeric comparisons
- `Contains` / `NotContains` - String search
- `StartsWith` / `EndsWith` - String prefix/suffix matching
- `RegexMatch` / `RegexNotMatch` - Regular expression matching

You can make string comparisons case-insensitive by setting `IgnoreCase: true`:
```golang
    t.FilterBy([]table.FilterBy{
        {Name: "First Name", Operator: table.Equal, Value: "JON", IgnoreCase: true},
    })
```

For advanced filtering requirements, you can provide a custom filter function:
```golang
    t.FilterBy([]table.FilterBy{
        {
            Number: 2,
            CustomFilter: func(cellValue string) bool {
                // Custom logic: include rows where first name length > 3
                return len(cellValue) > 3
            },
        },
    })
```

Example: Filter by salary and name
```golang
    t := table.NewWriter()
    t.AppendHeader(table.Row{"#", "First Name", "Last Name", "Salary"})
    t.AppendRows([]table.Row{
        {1, "Arya", "Stark", 3000},
        {20, "Jon", "Snow", 2000},
        {300, "Tyrion", "Lannister", 5000},
        {400, "Sansa", "Stark", 2500},
    })
    t.FilterBy([]table.FilterBy{
        {Number: 4, Operator: table.GreaterThan, Value: 2000},
        {Number: 3, Operator: table.Contains, Value: "Stark"},
    })
    t.Render()
```
to get:
```
+-----+------------+-----------+--------+
|   # | FIRST NAME | LAST NAME | SALARY |
+-----+------------+-----------+--------+
|   1 | Arya       | Stark     |   3000 |
| 400 | Sansa      | Stark     |   2500 |
+-----+------------+-----------+--------+
```

</details>

---

<details>
<summary><strong>Sorting</strong></summary>

Sorting can be done on one or more columns. The following code will make the
rows be sorted first by "First Name" and then by "Last Name" (in case of similar
"First Name" entries).
```golang
    t.SortBy([]table.SortBy{
	    {Name: "First Name", Mode: table.Asc},
	    {Name: "Last Name", Mode: table.Asc},
    })
```

The `Mode` field in `SortBy` supports various sorting modes:
- `Asc` / `Dsc` - Alphabetical ascending/descending
- `AscNumeric` / `DscNumeric` - Numerical ascending/descending
- `AscAlphaNumeric` / `DscAlphaNumeric` - Alphabetical first, then numerical
- `AscNumericAlpha` / `DscNumericAlpha` - Numerical first, then alphabetical

You can also make sorting case-insensitive by setting `IgnoreCase: true`:
```golang
    t.SortBy([]table.SortBy{
	    {Name: "First Name", Mode: table.Asc, IgnoreCase: true},
    })
```

</details>

---

<details>
<summary><strong>Sorting Customization</strong></summary>

For advanced sorting requirements, you can provide a custom comparison function
using `CustomLess`. This function overrides the `Mode` and `IgnoreCase` settings
and gives you full control over the sorting logic.

The `CustomLess` function receives two string values (the cell contents converted
to strings) and must return:
- `-1` when the first value should come before the second
- `0` when the values are considered equal (sorting continues to the next column)
- `1` when the first value should come after the second

Example: Custom numeric sorting that handles string numbers correctly.
```golang
    t.SortBy([]table.SortBy{
	    {
		    Number: 1,
		    CustomLess: func(iStr string, jStr string) int {
			    iNum, iErr := strconv.Atoi(iStr)
			    jNum, jErr := strconv.Atoi(jStr)
			    if iErr != nil || jErr != nil {
				    // Fallback to string comparison if not numeric
				    if iStr < jStr {
					    return -1
				    }
				    if iStr > jStr {
					    return 1
				    }
				    return 0
			    }
			    if iNum < jNum {
				    return -1
			    }
			    if iNum > jNum {
				    return 1
			    }
			    return 0
		    },
	    },
    })
```

Example: Combining custom sorting with default sorting modes.
```golang
    t.SortBy([]table.SortBy{
	    {
		    Number: 1,
		    CustomLess: func(iStr string, jStr string) int {
			    // Custom logic: "same" values come first
			    if iStr == "same" && jStr != "same" {
				    return -1
			    }
			    if iStr != "same" && jStr == "same" {
				    return 1
			    }
			    return 0 // Equal, continue to next column
		    },
	    },
	    {Number: 2, Mode: table.Asc},        // Default alphabetical sort
	    {Number: 3, Mode: table.AscNumeric}, // Default numeric sort
    })
```
</details>

---

<details>
<summary><strong>Wrapping (or) Row/Column Width restrictions</strong></summary>

You can restrict the maximum (text) width for a Row:
```golang
    t.SetAllowedRowLength(50)
    t.Render()
```
to get:
```
+-----+------------+-----------+--------+------- ~
|   # | FIRST NAME | LAST NAME | SALARY |        ~
+-----+------------+-----------+--------+------- ~
|   1 | Arya       | Stark     |   3000 |        ~
|  20 | Jon        | Snow      |   2000 | You kn ~
+-----+------------+-----------+--------+------- ~
| 300 | Tyrion     | Lannister |   5000 |        ~
+-----+------------+-----------+--------+------- ~
|     |            | TOTAL     |  10000 |        ~
+-----+------------+-----------+--------+------- ~
```

</details>

---

<details>
<summary><strong>Column Control - Alignment, Colors, Width and more</strong></summary>

You can control a lot of things about individual cells/columns which overrides
global properties/styles using the `SetColumnConfig()` interface:
- Alignment (horizontal & vertical)
- Colorization
- Transform individual cells based on the content
- Visibility
- Width (minimum & maximum)

```golang
    nameTransformer := text.Transformer(func(val interface{}) string {
        return text.Bold.Sprint(val)
    })

    t.SetColumnConfigs([]ColumnConfig{
        {
            Name:              "First Name",
            Align:             text.AlignLeft,
            AlignFooter:       text.AlignLeft,
            AlignHeader:       text.AlignLeft,
            Colors:            text.Colors{text.BgBlack, text.FgRed},
            ColorsHeader:      text.Colors{text.BgRed, text.FgBlack, text.Bold},
            ColorsFooter:      text.Colors{text.BgRed, text.FgBlack},
            Hidden:            false,
            Transformer:       nameTransformer,
            TransformerFooter: nameTransformer,
            TransformerHeader: nameTransformer,
            VAlign:            text.VAlignMiddle,
            VAlignFooter:      text.VAlignTop,
            VAlignHeader:      text.VAlignBottom,
            WidthMin:          6,
            WidthMax:          64,
        }
    })
```

</details>

---

<details>
<summary><strong>CSV</strong></summary>

```golang
    t.RenderCSV()
```
to get:
```
,First Name,Last Name,Salary,
1,Arya,Stark,3000,
20,Jon,Snow,2000,"You know nothing\, Jon Snow!"
300,Tyrion,Lannister,5000,
,,Total,10000,
```

</details>

---

<details>
<summary><strong>HTML Table</strong></summary>

```golang
    t.Style().HTML = table.HTMLOptions{
        CSSClass:             "game-of-thrones",
        EmptyColumn:          "&nbsp;",
        EscapeText:           true,
        Newline:              "<br/>",
        ConvertColorsToSpans: true, // Convert ANSI escape sequences to HTML <span> tags with CSS classes
    }
    t.RenderHTML()
```
to get:
```html
<table class="game-of-thrones">
  <thead>
  <tr>
    <th align="right">#</th>
    <th>First Name</th>
    <th>Last Name</th>
    <th align="right">Salary</th>
    <th>&nbsp;</th>
  </tr>
  </thead>
  <tbody>
  <tr>
    <td align="right">1</td>
    <td>Arya</td>
    <td>Stark</td>
    <td align="right">3000</td>
    <td>&nbsp;</td>
  </tr>
  <tr>
    <td align="right">20</td>
    <td>Jon</td>
    <td>Snow</td>
    <td align="right">2000</td>
    <td>You know nothing, Jon Snow!</td>
  </tr>
  <tr>
    <td align="right">300</td>
    <td>Tyrion</td>
    <td>Lannister</td>
    <td align="right">5000</td>
    <td>&nbsp;</td>
  </tr>
  </tbody>
  <tfoot>
  <tr>
    <td align="right">&nbsp;</td>
    <td>&nbsp;</td>
    <td>Total</td>
    <td align="right">10000</td>
    <td>&nbsp;</td>
  </tr>
  </tfoot>
</table>
```

</details>

---

<details>
<summary><strong>Markdown Table</strong></summary>

```golang
    t.RenderMarkdown()
```
to get:
```markdown
| # | First Name | Last Name | Salary |  |
| ---:| --- | --- | ---:| --- |
| 1 | Arya | Stark | 3000 |  |
| 20 | Jon | Snow | 2000 | You know nothing, Jon Snow! |
| 300 | Tyrion | Lannister | 5000 |  |
|  |  | Total | 10000 |  |
```

</details>

---
