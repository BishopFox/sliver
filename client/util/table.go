package util

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/evilsocket/islazy/tui"
	"github.com/olekukonko/tablewriter"
)

// Table - A wrapper around tablewriter.Table, for behavior customization
type Table struct {
	*tablewriter.Table
	maxColumnWidth int
	title          string

	// Callback are used to match certain values and output them with colors.
	headers   []string
	callbacks map[string]func(string) tablewriter.Colors
}

// NewTable - Constructor method with default behavior
func NewTable(title string) *Table {
	t := &Table{
		tablewriter.NewWriter(os.Stdout), // New table writer
		0,                                // Max column width
		title,                            // Table title
		[]string{},                       // Empty table
		map[string]func(string) tablewriter.Colors{}, // Callbacks
	}

	// Appearance
	t.Table.SetCenterSeparator(fmt.Sprintf("%s|%s", tui.FOREBLACK, tui.RESET))
	t.Table.SetColumnSeparator(fmt.Sprintf("%s|%s", tui.FOREBLACK, tui.RESET))
	t.Table.SetRowSeparator(fmt.Sprintf("%s-%s", tui.FOREBLACK, tui.RESET))
	t.Table.SetAlignment(tablewriter.ALIGN_LEFT)
	t.Table.SetBorder(false)

	// Format
	t.Table.SetAutoWrapText(false)
	t.Table.SetAutoFormatHeaders(false)
	t.Table.SetReflowDuringAutoWrap(false)

	// The column width is the default maximum column width
	t.Table.SetColWidth(70)
	t.maxColumnWidth = 70

	return t
}

// SetColumns - Set the headers (and their widths) for a table.
func (t *Table) SetColumns(names []string, widths []int) error {

	// Both list must have the same length.
	if len(names) != len(widths) {
		return errors.New("Unmatching number of column widths and column headers")
	}

	t.Table.SetHeader(names)
	t.headers = names

	// Headers: filters against a list of Column titles, against which we apply special coloring,
	// in either headers and/or columns, like for IDs, IP addresses, status.
	var headerColors []tablewriter.Colors
	var columnColors []tablewriter.Colors

	// For each header name, apply coloring. By default we have grey titles.
	for range names {
		headerColors = append(headerColors, tablewriter.Colors{
			tablewriter.Normal,
			tablewriter.FgHiBlackColor,
		})
	}
	t.Table.SetHeaderColor(headerColors...)

	// Column content automatic coloring
	for _, name := range names {
		switch name {

		// Add color callbacks to Status strings in the table.
		case "Status", "Health":
			statusCb := func(status string) tablewriter.Colors {
				switch status {
				case "ALIVE", "Alive":
					return tablewriter.Colors{tablewriter.Normal, tablewriter.FgGreenColor}
				case "DEAD", "Dead":
					return tablewriter.Colors{tablewriter.Normal, tablewriter.FgRedColor}
				case "PENDING", "Pending":
					return tablewriter.Colors{tablewriter.Normal, tablewriter.FgYellowColor}
				case "ASLEEP", "Asleep":
					return tablewriter.Colors{tablewriter.Normal, tablewriter.FgCyanColor}
				default:
					return tablewriter.Colors{tablewriter.Italic, tablewriter.FgHiBlackColor}
				}
			}
			t.callbacks[name] = statusCb

			columnColors = append(columnColors, tablewriter.Colors{
				tablewriter.Normal,
			})

		case "Value":
			// All module/extension values are bolded
			columnColors = append(columnColors, tablewriter.Colors{
				tablewriter.Bold,
			})

		// Implant connection status: green/yellow/red depending on value.
		case "Required", "Req":
			columnColors = append(columnColors, tablewriter.Colors{
				tablewriter.Normal,
				tablewriter.FgHiBlackColor,
			})

		case "ID", "SessionID":
			columnColors = append(columnColors, tablewriter.Colors{
				tablewriter.Normal,
				tablewriter.FgHiBlackColor,
			})

		default:
			// By default colum contents are normal
			columnColors = append(columnColors, tablewriter.Colors{
				tablewriter.Normal,
			})
		}
	}
	t.Table.SetColumnColor(columnColors...)

	// Length: in order to optimize printing
	for i, width := range widths {
		// If width is non-zero, simply set the column.
		if width != 0 {
			t.Table.SetColMinWidth(i, width)
			continue
		}
	}

	return nil
}

// ApplyCurrentRowColor - We want to signal this row is of importance in a given context.
func (t *Table) ApplyCurrentRowColor(items []string, color string) (colored []string) {
	for _, item := range items {
		colored = append(colored, fmt.Sprintf("%s%s", color, item))
	}
	return
}

// AppendRow - Add a row of items. Each of them is being applied against
// its column settings and/or any callbacks, before being added.
func (t *Table) AppendRow(items []string) error {

	var rowColors []tablewriter.Colors // Item coloring
	var rowItems []string              //Processed items

	// For each item (column) apply coloring and wrapping
	for i, item := range items {

		// Check for a callback first and apply it if found.
		if cb, ok := t.callbacks[t.headers[i]]; ok {
			rowColors = append(rowColors, cb(item))
		} else {
			// By default colum contents are normal
			rowColors = append(rowColors, tablewriter.Colors{
				tablewriter.Normal,
			})
		}

		// Wrapping
		if len(item) > t.maxColumnWidth {
			rowItems = append(rowItems, Wrap(item, t.maxColumnWidth))
		} else {
			rowItems = append(rowItems, item)
		}
	}

	// Push to table buffer
	t.Table.Rich(rowItems, rowColors)

	return nil
}

// Output - Render the table, and its title if non-nil
func (t *Table) Output() {
	if t.title != "" {
		fmt.Println(" " + t.title)
		// fmt.Println(" " + t.title + "\n")
	}
	t.Table.Render()
}

// SortMapInt - Sort a map with int keys
func SortMapInt(in map[int]interface{}) (out map[int]interface{}) {
	out = map[int]interface{}{}

	// Sort keys
	var keys []int
	for k := range in {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	// Make map
	for k := range keys {
		out[k] = in[k]
	}

	return
}

// SortMapUint32 - Sort a map with uint32 keys
func SortMapUint32(in map[uint32]interface{}) (out map[uint32]interface{}) {
	out = map[uint32]interface{}{}

	// Sort keys
	var keys []int
	for k := range in {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	// Make map
	for k := range keys {
		out[uint32(k)] = in[uint32(k)]
	}

	return
}

// SortMapString - Sort a map with string keys
func SortMapString(in map[string]interface{}) (out map[string]interface{}) {

	out = map[string]interface{}{}

	// Sort keys
	var keys []string
	for k := range in {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Make map
	for _, k := range keys {
		out[k] = in[k]
	}

	return
}
