package display

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/reeflective/readline/internal/color"
	"github.com/reeflective/readline/internal/core"
)

// highlightLine applies visual/selection highlighting to a line.
// The provided line might already have been highlighted by a user-provided
// highlighter: this function accounts for any embedded color sequences.
func (e *Engine) highlightLine(line []rune, selection core.Selection) string {
	// Sort regions and extract colors/positions.
	sorted := sortHighlights(selection)
	colors := e.getHighlights(line, sorted)

	var highlighted string

	// And apply highlighting before each rune.
	for i, r := range line {
		if highlight, found := colors[i]; found {
			highlighted += string(highlight)
		}

		highlighted += string(r)
	}

	// Finally, highlight comments using a regex.
	comment := strings.Trim(e.opts.GetString("comment-begin"), "\"")
	commentPattern := fmt.Sprintf(`(^|\s)%s.*`, comment)

	if commentsMatch, err := regexp.Compile(commentPattern); err == nil {
		commentColor := color.SGRStart + color.Fg + "244" + color.SGREnd
		highlighted = commentsMatch.ReplaceAllString(highlighted, fmt.Sprintf("%s${0}%s", commentColor, color.Reset))
	}

	highlighted += color.Reset

	return highlighted
}

func sortHighlights(vhl core.Selection) []core.Selection {
	all := make([]core.Selection, 0)
	sorted := make([]core.Selection, 0)
	bpos := make([]int, 0)

	for _, reg := range vhl.Surrounds() {
		all = append(all, reg)
		rbpos, _ := reg.Pos()
		bpos = append(bpos, rbpos)
	}

	if vhl.Active() && vhl.IsVisual() {
		all = append(all, vhl)
		vbpos, _ := vhl.Pos()
		bpos = append(bpos, vbpos)
	}

	sort.Ints(bpos)

	prevIsMatcher := false
	prevPos := 0

	for _, pos := range bpos {
		for _, reg := range all {
			bpos, _ := reg.Pos()
			isMatcher := reg.Type == "matcher"

			if bpos != pos || !reg.Active() || !reg.IsVisual() {
				continue
			}

			// If we have both a matcher and a visual selection
			// starting at the same position, then we might have
			// just added the matcher, and we must "overwrite" it
			// with the visual selection, so skip until we find it.
			if prevIsMatcher && isMatcher && prevPos == pos {
				continue
			}

			// Else the region is good to be used in that order.
			sorted = append(sorted, reg)
			prevIsMatcher = reg.Type == "matcher"
			prevPos = bpos

			break
		}
	}

	return sorted
}

func (e *Engine) getHighlights(line []rune, sorted []core.Selection) map[int][]rune {
	highlights := make(map[int][]rune)

	// Find any highlighting already applied on the line,
	// and keep the indexes so that we can skip those.
	var colors [][]int

	colorMatch := regexp.MustCompile(`\x1b\[[0-9;]+m`)
	colors = colorMatch.FindAllStringIndex(string(line), -1)

	// marks that started highlighting, but not done yet.
	regions := make([]core.Selection, 0)
	pos := -1
	skip := 0

	// Build the string.
	for rawIndex := range line {
		var posHl []rune
		var newHl core.Selection

		// While in a color escape, keep reading runes.
		if skip > 0 {
			skip--
			continue
		}

		// If starting a color escape code, add offset and read.
		if len(colors) > 0 && colors[0][0] == rawIndex {
			skip += colors[0][1] - colors[0][0] - 1
			colors = colors[1:]

			continue
		}

		// Or we are reading a printed rune.
		pos++

		// First check if we have a new highlighter to apply
		for _, hl := range sorted {
			bpos, _ := hl.Pos()

			if bpos == pos {
				newHl = hl
				regions = append(regions, hl)
			}
		}

		// Add new colors if any, and reset if some are done.
		regions, posHl = e.hlReset(regions, posHl, pos)
		posHl = e.hlAdd(regions, newHl, posHl)

		// Add to the line, with the raw index since
		// we must take into account embedded colors.
		if len(posHl) > 0 {
			highlights[rawIndex] = posHl
		}
	}

	return highlights
}

func (e *Engine) hlAdd(regions []core.Selection, newHl core.Selection, line []rune) []rune {
	var (
		fg, bg  string
		matcher bool
		hl      core.Selection
	)

	if newHl.Active() {
		hl = newHl
	} else if len(regions) > 0 {
		hl = regions[len(regions)-1]
	}

	fg, bg = hl.Highlights()
	matcher = hl.Type == "matcher"

	// Update the highlighting with inputrc settings if any.
	if bg != "" && !matcher {
		background := color.UnquoteRC("active-region-start-color")
		if bg, _ = strconv.Unquote(background); bg == "" {
			bg = color.Reverse
		}
	}

	// Add highlightings
	line = append(line, []rune(bg)...)
	line = append(line, []rune(fg)...)

	return line
}

func (e *Engine) hlReset(regions []core.Selection, line []rune, pos int) ([]core.Selection, []rune) {
	for i, reg := range regions {
		_, epos := reg.Pos()
		foreground, background := reg.Highlights()
		matcher := reg.Type == "matcher"

		if epos != pos {
			continue
		}

		if i < len(regions)-1 {
			regions = append(regions[:i], regions[i+1:]...)
		} else {
			regions = regions[:i]
		}

		if foreground != "" {
			line = append(line, []rune(color.FgDefault)...)
		}

		if background != "" {
			background, _ := strconv.Unquote(e.opts.GetString("active-region-end-color"))
			foreground := e.opts.GetString("active-region-start-color")

			if background == "" && foreground == "" && !matcher {
				line = append(line, []rune(color.ReverseReset)...)
			} else {
				line = append(line, []rune(color.BgDefault)...)
			}
		}
	}

	return regions, line
}
