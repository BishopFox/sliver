package settings

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	tableStyles = map[string]table.Style{
		// Sliver styles
		SliverDefault.Name: SliverDefault,

		// Go Pretty styles
		table.StyleBold.Name:                    table.StyleBold,
		table.StyleColoredBright.Name:           table.StyleColoredBright,
		table.StyleLight.Name:                   table.StyleLight,
		table.StyleColoredDark.Name:             table.StyleColoredDark,
		table.StyleColoredBlackOnBlueWhite.Name: table.StyleColoredBlackOnBlueWhite,
	}

	SliverDefault = table.Style{
		Name: "SliverDefault",
		Box: table.BoxStyle{
			BottomLeft:       " ",
			BottomRight:      " ",
			BottomSeparator:  " ",
			Left:             " ",
			LeftSeparator:    " ",
			MiddleHorizontal: "=",
			MiddleSeparator:  " ",
			MiddleVertical:   " ",
			PaddingLeft:      " ",
			PaddingRight:     " ",
			Right:            " ",
			RightSeparator:   " ",
			TopLeft:          " ",
			TopRight:         " ",
			TopSeparator:     " ",
			UnfinishedRow:    "~~",
		},
		Color: table.ColorOptions{
			IndexColumn:  text.Colors{},
			Footer:       text.Colors{},
			Header:       text.Colors{},
			Row:          text.Colors{},
			RowAlternate: text.Colors{},
		},
		Format: table.FormatOptions{
			Footer: text.FormatDefault,
			Header: text.FormatTitle,
			Row:    text.FormatDefault,
		},
		Options: table.Options{
			DrawBorder:      false,
			SeparateColumns: true,
			SeparateFooter:  false,
			SeparateHeader:  true,
			SeparateRows:    false,
		},
	}
)

// GetTableStyle - Get the current table style
func GetTableStyle(con *console.SliverConsoleClient) table.Style {
	if con.Settings == nil {
		con.Settings, _ = assets.LoadSettings()
	}
	if con.Settings != nil {
		if value, ok := tableStyles[con.Settings.TableStyle]; ok {
			return value
		}
	}
	return tableStyles[SliverDefault.Name]
}

// GetPageSize - Page size for tables
func GetPageSize() int {
	return 10
}

// PagesOf - Return the pages of a table
func PagesOf(tw table.Writer) [][]string {
	lines := strings.Split(tw.Render(), "\n")
	if len(lines) < 2 {
		return [][]string{}
	}
	token := lines[0]
	pages := [][]string{}
	page := []string{token}
	for _, line := range lines[1:] {
		if line != token {
			page = append(page, line)
		} else {
			pages = append(pages, page)
			page = []string{token}
		}
	}
	pages = append(pages, page)
	return pages
}

// PaginateTable - Render paginated table to console
func PaginateTable(tw table.Writer, skipPages int, overflow bool, interactive bool, con *console.SliverConsoleClient) {
	if !overflow {
		width, height, err := terminal.GetSize(0)
		if err == nil {
			if 7 < height {
				tw.SetPageSize(height - 6)
				tw.SetAllowedRowLength(width)
			}
		} else {
			tw.SetAllowedRowLength(150)
		}
	}

	pages := PagesOf(tw)
	for pageNumber, page := range pages {
		if pageNumber+1 < skipPages {
			continue
		}
		for _, line := range page {
			if len(line) == 0 {
				continue
			}
			con.Printf("%s\n", line)
		}
		con.Println()
		if interactive {
			if 1 < len(pages) {
				nextPage := false
				prompt := &survey.Confirm{
					Message: fmt.Sprintf("[%d/%d] Continue?", pageNumber+1, len(pages)),
				}
				survey.AskOne(prompt, &nextPage)
				if !nextPage {
					break
				}
				con.Println()
			}
		} else {
			if 1 < len(pages) {
				con.Printf(console.Bold+"Page [%d/%d]\n", pageNumber+1, len(pages))
			}
			break
		}
	}
}
