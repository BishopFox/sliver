package completion

import (
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/reeflective/readline/internal/color"
	"github.com/reeflective/readline/internal/term"
)

// group is used to structure different types of completions with different
// display types, autosuffix removal matchers, under their tag heading.
type group struct {
	tag               string        // Printed on top of the group's completions
	rows              [][]Candidate // Values are grouped by aliases/rows, with computed paddings.
	noSpace           SuffixMatcher // Suffixes to remove if a space or non-nil character is entered after the completion.
	columnsWidth      []int         // Computed width for each column of completions, when aliases
	descriptionsWidth []int         // Computed width for each column of completions, when aliases
	listSeparator     string        // This is used to separate completion candidates from their descriptions.
	list              bool          // Force completions to be listed instead of grided
	noSort            bool          // Don't sort completions
	aliased           bool          // Are their aliased completions
	preserveEscapes   bool          // Preserve escape sequences in the completion inserted values.
	isCurrent         bool          // Currently cycling through this group, for highlighting choice
	longestValue      int           // Used when display is map/list, for determining message width
	longestDesc       int           // Used to know how much descriptions can use when there are aliases.
	maxDescAllowed    int           // Maximum ALLOWED description width.
	termWidth         int           // Term size queried at beginning of computes by the engine.

	// Selectors (position/bounds) management
	posX int
	posY int
	maxX int
	maxY int
}

// newCompletionGroup initializes a group of completions to be displayed in the same area/header.
func (e *Engine) newCompletionGroup(comps Values, tag string, vals RawValues, descriptions []string) {
	grp := &group{
		tag:          tag,
		noSpace:      comps.NoSpace,
		posX:         -1,
		posY:         -1,
		columnsWidth: []int{0},
		termWidth:    term.GetWidth(),
		longestDesc:  longest(descriptions, true),
	}

	// Initialize all options for the group.
	grp.initOptions(e, &comps, tag, vals)

	// Global actions to take on all values.
	if !grp.noSort {
		sort.Stable(vals)
	}

	// Initial processing of our assigned values:
	// Compute color/no-color sizes, some max/min, etc.
	grp.prepareValues(vals)

	// Generate the full grid of completions.
	// Special processing is needed when some values
	// share a common description, they are "aliased".
	if completionsAreAliases(vals) {
		grp.initCompletionAliased(vals)
	} else {
		grp.initCompletionsGrid(vals)
	}

	e.groups = append(e.groups, grp)
}

// initOptions checks for global or group-specific options (display, behavior, grouping, etc).
func (g *group) initOptions(eng *Engine, comps *Values, tag string, vals RawValues) {
	// Override grid/list displays
	_, g.list = comps.ListLong[tag]
	if _, all := comps.ListLong["*"]; all && len(comps.ListLong) == 1 {
		g.list = true
	}

	// Description list separator
	listSep, err := strconv.Unquote(eng.config.GetString("completion-list-separator"))
	if err != nil {
		g.listSeparator = "--"
	} else {
		g.listSeparator = listSep
	}

	// Strip escaped characters in the value component.
	g.preserveEscapes = comps.Escapes[g.tag]
	if !g.preserveEscapes {
		g.preserveEscapes = comps.Escapes["*"]
	}

	// Always list long commands when they have descriptions.
	if strings.HasSuffix(g.tag, "commands") && len(vals) > 0 && vals[0].Description != "" {
		g.list = true
	}

	// Description list separator
	listSep, found := comps.ListSep[tag]
	if !found {
		if allSep, found := comps.ListSep["*"]; found {
			g.listSeparator = allSep
		}
	} else {
		g.listSeparator = listSep
	}

	// Override sorting or sort if needed
	g.noSort = comps.NoSort[tag]
	if noSort, all := comps.NoSort["*"]; noSort && all && len(comps.NoSort) == 1 {
		g.noSort = true
	}
}

// initCompletionsGrid arranges completions when there are no aliases.
func (g *group) initCompletionsGrid(comps RawValues) {
	if len(comps) == 0 {
		return
	}

	pairLength := g.longestValueDescribed(comps)
	if pairLength > g.termWidth {
		pairLength = g.termWidth
	}

	maxColumns := g.termWidth / pairLength
	if g.list || maxColumns < 0 {
		maxColumns = 1
	}

	rowCount := int(math.Ceil(float64(len(comps)) / (float64(maxColumns))))

	g.rows = createGrid(comps, rowCount, maxColumns)
	g.calculateMaxColumnWidths(g.rows)
}

// initCompletionsGrid arranges completions when some of them share the same description.
func (g *group) initCompletionAliased(domains []Candidate) {
	g.aliased = true

	// Filter out all duplicates: group aliased completions together.
	grid, descriptions := g.createDescribedRows(domains)
	g.calculateMaxColumnWidths(grid)
	g.wrapExcessAliases(grid, descriptions)

	g.maxY = len(g.rows)
	g.maxX = len(g.columnsWidth)
}

// This createDescribedRows function takes a list of values, a list of descriptions, and the
// terminal width as input, and returns a list of rows based on the provided requirements:.
func (g *group) createDescribedRows(values []Candidate) ([][]Candidate, []string) {
	descriptionMap := make(map[string][]Candidate)
	uniqueDescriptions := make([]string, 0)
	rows := make([][]Candidate, 0)

	// Separate duplicates and store them.
	for i, description := range values {
		if slices.Contains(uniqueDescriptions, description.Description) {
			descriptionMap[description.Description] = append(descriptionMap[description.Description], values[i])
		} else {
			uniqueDescriptions = append(uniqueDescriptions, description.Description)
			descriptionMap[description.Description] = []Candidate{values[i]}
		}
	}

	// Sorting helps with easier grids.
	for _, description := range uniqueDescriptions {
		row := descriptionMap[description]
		// slices.Sort(row)
		// slices.Reverse(row)
		rows = append(rows, row)
	}

	return rows, uniqueDescriptions
}

// Wraps all rows for which there are too many aliases to be displayed on a single one.
func (g *group) wrapExcessAliases(grid [][]Candidate, descriptions []string) {
	breakeven := 0
	maxColumns := len(g.columnsWidth)

	for i, width := range g.columnsWidth {
		if (breakeven + width + 1) > g.termWidth/2 {
			maxColumns = i
			break
		}

		breakeven += width + 1
	}

	var rows [][]Candidate

	for rowIndex := range grid {
		row := grid[rowIndex]

		for len(row) > maxColumns {
			rows = append(rows, row[:maxColumns])
			row = row[maxColumns:]
		}

		rows = append(rows, row)
	}

	g.rows = rows
	g.columnsWidth = g.columnsWidth[:maxColumns]
}

// prepareValues ensures all of them have a display, and starts
// gathering information on longest/shortest values, etc.
func (g *group) prepareValues(vals RawValues) RawValues {
	for pos, value := range vals {
		if value.Display == "" {
			value.Display = value.Value
		}

		// Only pass for colors regex should be here.
		value.displayLen = len(color.Strip(value.Display))
		value.descLen = len(color.Strip(value.Description))

		if value.displayLen > g.longestValue {
			g.longestValue = value.displayLen
		}

		if value.descLen > g.longestDesc {
			g.longestDesc = value.descLen
		}

		vals[pos] = value
	}

	return vals
}

func (g *group) setMaximumSizes(col int) int {
	// Get the length of the longest description in the same column.
	maxDescLen := g.descriptionsWidth[col]
	valuesRealLen := sum(g.columnsWidth) + len(g.columnsWidth) + len(g.listSep())

	if valuesRealLen+maxDescLen > g.termWidth {
		maxDescLen = g.termWidth - valuesRealLen
	} else if valuesRealLen+maxDescLen < g.termWidth {
		maxDescLen = g.termWidth - valuesRealLen
	}

	return maxDescLen
}

// calculateMaxColumnWidths is in charge of optimizing the sizes of rows/columns.
func (g *group) calculateMaxColumnWidths(grid [][]Candidate) {
	var numColumns int

	// Get the row with the greatest number of columns.
	for _, row := range grid {
		if len(row) > numColumns {
			numColumns = len(row)
		}
	}

	// First, all columns are as wide as the longest of their elements,
	// regardless of if this longest element is longer than terminal
	values := make([]int, numColumns)
	descriptions := make([]int, numColumns)

	for _, row := range grid {
		for columnIndex, value := range row {
			if value.displayLen+1 > values[columnIndex] {
				values[columnIndex] = value.displayLen + 1
			}

			if value.descLen+1 > descriptions[columnIndex] {
				descriptions[columnIndex] = value.descLen + 1
			}
		}
	}

	// If we have only one row, it means that the number of columns
	// multiplied by the size on the longest one will fit into the
	// terminal, so we can just
	if len(grid) == 1 && len(grid[0]) <= numColumns && sum(descriptions) == 0 {
		for i := range values {
			values[i] = g.longestValue
		}
	}

	// Last time adjustment: try to reallocate any space modulo to each column.
	shouldPad := len(grid) > 1 && numColumns > 1 && sum(descriptions) == 0
	intraColumnSpace := (numColumns * 2)
	totalSpaceUsed := sum(values) + sum(descriptions) + intraColumnSpace
	freeSpace := g.termWidth - totalSpaceUsed

	if shouldPad && !g.aliased && freeSpace >= numColumns {
		each := freeSpace / numColumns

		for i := range values {
			values[i] += each
		}
	}

	// The group is mostly ready to print and select its values for completion.
	g.maxY = len(g.rows)
	g.maxX = len(values)
	g.columnsWidth = values
	g.descriptionsWidth = descriptions
}

func (g *group) longestValueDescribed(vals []Candidate) int {
	var longestDesc, longestVal int

	// Equivalent to `<completion> -- <Description>`,
	// asuuming no trailing spaces in the completion
	// nor leading spaces in the description.
	descSeparatorLen := 1 + len(g.listSeparator) + 1

	// Get the length of the longest value
	// and the length of the longest description.
	for _, val := range vals {
		if val.displayLen > longestVal {
			longestVal = val.displayLen
		}

		if val.descLen > longestDesc {
			longestDesc = val.descLen
		}

		if val.descLen > longestDesc {
			longestDesc = val.descLen
		}
	}

	if longestDesc > 0 {
		longestDesc += descSeparatorLen
	}

	if longestDesc > 0 {
		longestDesc += descSeparatorLen
	}

	// Always add one: there is at least one space between each column.
	return longestVal + longestDesc + 2
}

func (g *group) trimDisplay(comp Candidate, pad, col int) (candidate, padded string) {
	val := comp.Display

	// No display value means padding.
	if val == "" {
		return "", padSpace(pad)
	}

	// Get the allowed length for this column.
	// The display can never be longer than terminal width.
	maxDisplayWidth := g.columnsWidth[col] + 1

	if maxDisplayWidth > g.termWidth {
		maxDisplayWidth = g.termWidth
	}

	val = sanitizer.Replace(val)

	if comp.displayLen > maxDisplayWidth {
		val = color.Trim(val, maxDisplayWidth-trailingValueLen)
		val += "..." // 3 dots + 1 safety space = -3

		return val, " "
	}

	return val, padSpace(pad)
}

func (g *group) trimDesc(val Candidate, pad int) (desc, padded string) {
	desc = val.Description
	if desc == "" {
		return desc, padSpace(pad)
	}

	// We don't compare against the terminal width:
	// the correct padding should have been computed
	// based on the space taken by all candidates
	// described by our current string.
	if pad > g.maxDescAllowed {
		pad = g.maxDescAllowed - val.descLen
	}

	desc = sanitizer.Replace(desc)

	// Trim the description accounting for escapes.
	if val.descLen > g.maxDescAllowed && g.maxDescAllowed > 0 {
		desc = color.Trim(desc, g.maxDescAllowed-trailingDescLen)
		desc += "..." // 3 dots =  -3

		return g.listSep() + desc, ""
	}

	if val.descLen+pad > g.maxDescAllowed {
		pad = g.maxDescAllowed - val.descLen
	}

	return g.listSep() + desc, padSpace(pad)
}

func (g *group) getPad(value Candidate, columnIndex int, desc bool) int {
	columns := g.columnsWidth
	valLen := value.displayLen - 1

	if desc {
		columns = g.descriptionsWidth
		valLen = value.descLen
	}

	// Ensure we never longer or even fully equal
	// to the terminal size: we are not really sure
	// of where offsets might be set in the code...
	column := columns[columnIndex]
	if column > g.termWidth-1 {
		column = g.termWidth - 1
	}

	padding := column - valLen

	if padding < 0 {
		return 0
	}

	return padding
}

func (g *group) listSep() string {
	return g.listSeparator + " "
}

//
// Usage-time functions (selecting/writing) -----------------------------------------------------------------
//

// updateIsearch - When searching through all completion groups (whether it be command history or not),
// we ask each of them to filter its own items and return the results to the shell for aggregating them.
// The rx parameter is passed, as the shell already checked that the search pattern is valid.
func (g *group) updateIsearch(eng *Engine) {
	if eng.IsearchRegex == nil {
		return
	}

	suggs := make([]Candidate, 0)

	for i := range g.rows {
		row := g.rows[i]

		for _, val := range row {
			if eng.IsearchRegex.MatchString(val.Value) {
				suggs = append(suggs, val)
			} else if val.Description != "" && eng.IsearchRegex.MatchString(val.Description) {
				suggs = append(suggs, val)
			}
		}
	}

	// Reset the group parameters
	g.rows = make([][]Candidate, 0)
	g.posX = -1
	g.posY = -1

	// Initial processing of our assigned values:
	// Compute color/no-color sizes, some max/min, etc.
	suggs = g.prepareValues(suggs)

	// Generate the full grid of completions.
	// Special processing is needed when some values
	// share a common description, they are "aliased".
	if completionsAreAliases(suggs) {
		g.initCompletionAliased(suggs)
	} else {
		g.initCompletionsGrid(suggs)
	}
}

func (g *group) selected() (comp Candidate) {
	defer func() {
		if !g.preserveEscapes {
			comp.Value = color.Strip(comp.Value)
		}
	}()

	if g.posY == -1 || g.posX == -1 {
		return g.rows[0][0]
	}

	return g.rows[g.posY][g.posX]
}

func (g *group) moveSelector(x, y int) (done, next bool) {
	// When the group has not yet been used, adjust
	if g.posX == -1 && g.posY == -1 {
		if x != 0 {
			g.posY++
		} else {
			g.posX++
		}
	}

	g.posX += x
	g.posY += y
	reverse := (x < 0 || y < 0)

	// 1) Ensure columns is minimum one, if not, either
	// go to previous row, or go to previous group.
	if g.posX < 0 {
		if g.posY == 0 && reverse {
			g.posX = 0
			g.posY = 0

			return true, false
		}

		g.posY--
		g.posX = len(g.rows[g.posY]) - 1
	}

	// 2) If we are reverse-cycling and currently on the first candidate,
	// we are done with this group. Stay on those coordinates still.
	if g.posY < 0 {
		if g.posX == 0 {
			g.posX = 0
			g.posY = 0

			return true, false
		}

		g.posY = len(g.rows) - 1
		g.posX--
	}

	// 3) If we are on the last row, we might have to move to
	// the next column, if there is another one.
	if g.posY > g.maxY-1 {
		g.posY = 0
		if g.posX < g.maxX-1 {
			g.posX++
		} else {
			return true, true
		}
	}

	// 4) If we are on the last column, go to next row or next group
	if g.posX > len(g.rows[g.posY])-1 {
		if g.aliased {
			return g.findFirstCandidate(x, y)
		}

		g.posX = 0

		if g.posY < g.maxY-1 {
			g.posY++
		} else {
			return true, true
		}
	}

	// By default, come back to this group for next item.
	return false, false
}

// Check that there is indeed a completion in the column for a given row,
// otherwise loop in the direction wished until one is found, or go next/
// previous column, and so on.
func (g *group) findFirstCandidate(x, y int) (done, next bool) {
	for g.posX > len(g.rows[g.posY])-1 {
		g.posY += y
		g.posY += x

		// Previous column or group
		if g.posY < 0 {
			if g.posX == 0 {
				g.posX = 0
				g.posY = 0

				return true, false
			}

			g.posY = len(g.rows) - 1
			g.posX--
		}

		// Next column or group
		if g.posY > g.maxY-1 {
			g.posY = 0
			if g.posX < len(g.columnsWidth)-1 {
				g.posX++
			} else {
				return true, true
			}
		}
	}

	return
}

func (g *group) firstCell() {
	g.posX = 0
	g.posY = 0
}

func (g *group) lastCell() {
	g.posY = len(g.rows) - 1
	g.posX = len(g.columnsWidth) - 1

	if g.aliased {
		g.findFirstCandidate(0, -1)
	} else {
		g.posX = len(g.rows[g.posY]) - 1
	}
}

func completionsAreAliases(values []Candidate) bool {
	oddValueMap := make(map[string]bool)

	for _, value := range values {
		if value.Description == "" {
			continue
		}

		if _, found := oddValueMap[value.Description]; found {
			return true
		}

		oddValueMap[value.Description] = true
	}

	return false
}

func createGrid(values []Candidate, rowCount, maxColumns int) [][]Candidate {
	if rowCount < 0 {
		rowCount = 0
	}

	grid := make([][]Candidate, rowCount)

	for i := 0; i < rowCount; i++ {
		grid[i] = createRow(values, maxColumns, i)
	}

	return grid
}

func createRow(domains []Candidate, maxColumns, rowIndex int) []Candidate {
	rowStart := rowIndex * maxColumns
	rowEnd := (rowIndex + 1) * maxColumns

	if rowEnd > len(domains) {
		rowEnd = len(domains)
	}

	return domains[rowStart:rowEnd]
}

func padSpace(times int) string {
	if times > 0 {
		return strings.Repeat(" ", times)
	}

	return ""
}
