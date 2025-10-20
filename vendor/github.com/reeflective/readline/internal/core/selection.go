package core

import (
	"regexp"
	"unicode"

	"github.com/reeflective/readline/inputrc"
	"github.com/reeflective/readline/internal/color"
	"github.com/reeflective/readline/internal/strutil"
)

// Selection contains all regions of an input line that are currently selected/marked
// with either a begin and/or end position. The main selection is the visual one, used
// with the default cursor mark and position, and contains a list of additional surround
// selections used to change/select multiple parts of the line at once.
type Selection struct {
	Type       string // Can be a normal one, surrounding (pairs), (cursor) matchers, etc.
	active     bool   // The selection is running.
	visual     bool   // The selection is highlighted.
	visualLine bool   // The selection should span entire lines.
	bpos       int    // Beginning index position
	epos       int    // End index position (can be +1 in visual mode, to encompass cursor pos)
	kpos       int    // Keyword regexp matchers cycling counter.
	kmpos      int    // Keyword regexp matcher subgroups counter.

	// Display
	fg        string      // Foreground color of the highlighted selection.
	bg        string      // Background color.
	surrounds []Selection // Surrounds are usually pairs of characters matching each other (quotes/brackets, etc.)

	// Core
	line   *Line
	cursor *Cursor
}

// NewSelection is a required constructor to use for initializing
// a selection, as some numeric values must be negative by default.
func NewSelection(line *Line, cursor *Cursor) *Selection {
	return &Selection{
		bpos:   -1,
		epos:   -1,
		line:   line,
		cursor: cursor,
	}
}

// Mark starts a pending selection at the specified position in the line.
// If the position is out of the line bounds, no selection is started.
// If this function is called on a surround selection, nothing happens.
func (s *Selection) Mark(pos int) {
	if pos < 0 || pos > s.line.Len() {
		return
	}

	s.MarkRange(pos, -1)
}

// MarkRange starts a selection as a range in the input line. If either of
// begin/end pos are negative, it is replaced with the current cursor position.
// Any out of range positive value is replaced by the length of the line.
func (s *Selection) MarkRange(bpos, epos int) {
	bpos, epos, valid := s.checkRange(bpos, epos)
	if !valid {
		return
	}

	s.Type = "visual"
	s.active = true
	s.bpos = bpos
	s.epos = epos
	s.bg = color.BgBlue
}

// MarkSurround creates two distinct selections each containing one rune.
// The first area starts at bpos, and the second one at epos. If either bpos
// is negative or epos is > line.Len()-1, no selection is created.
func (s *Selection) MarkSurround(bpos, epos int) {
	if bpos < 0 || epos > s.line.Len()-1 {
		return
	}

	s.active = true

	for _, pos := range []int{bpos, epos} {
		s.surrounds = append(s.surrounds, Selection{
			Type:   "surround",
			active: true,
			visual: true,
			bpos:   pos,
			epos:   pos,
			bg:     color.BgRed,
			line:   s.line,
			cursor: s.cursor,
		})
	}
}

// Active return true if the selection is active.
// When created, all selections are marked active,
// so that visual modes in Vim can work properly.
func (s *Selection) Active() bool {
	return s.active
}

// Visual sets the selection as a visual one (highlighted),
// which is commonly seen in Vim edition mode.
// If line is true, the visual is extended to entire lines.
func (s *Selection) Visual(line bool) {
	s.visual = true
	s.visualLine = line
}

// IsVisual indicates whether the selection should be highlighted.
func (s *Selection) IsVisual() bool {
	return s.visual
}

// Pos returns the begin and end positions of the selection.
// If any of these is not set, it is set to the cursor position.
// This is generally the case with "pending" visual selections.
func (s *Selection) Pos() (bpos, epos int) {
	if s.line.Len() == 0 || !s.active {
		return -1, -1
	}

	bpos, epos, valid := s.checkRange(s.bpos, s.epos)
	if !valid {
		return
	}

	// Use currently set values, or update if one is pending.
	s.bpos, s.epos = bpos, epos

	if epos == -1 {
		bpos, epos = s.selectToCursor(bpos)
	}

	if s.visual {
		epos++
	}

	// Always check that neither the initial values nor the ones
	// that we might have updated are wrong. It's very rare that
	// the adjusted values would be invalid as a result of this
	// call (unfixable values), but better being too safe than not.
	bpos, epos, valid = s.checkRange(bpos, epos)
	if !valid {
		return -1, -1
	}

	return bpos, epos
}

// Cursor returns what should be the cursor position if the active
// selection is to be deleted, but also works for yank operations.
func (s *Selection) Cursor() int {
	bpos, epos := s.Pos()
	if bpos == -1 && epos == -1 {
		return s.cursor.Pos()
	}

	cpos := bpos

	if !s.visual || !s.visualLine {
		return cpos
	}

	var indent int
	pos := s.cursor.Pos()

	// Get the indent of the cursor line.
	for cpos = pos - 1; cpos >= 0; cpos-- {
		if (*s.line)[cpos] == '\n' {
			break
		}
	}

	indent = pos - cpos - 1

	// If the selection includes the last line,
	// the cursor will move up the above line.
	var hpos, rpos int

	if epos < s.line.Len() {
		hpos = epos + 1
		rpos = bpos
	} else {
		for hpos = bpos - 2; hpos >= 0; hpos-- {
			if (*s.line)[hpos] == '\n' {
				break
			}
		}
		if hpos < -1 {
			hpos = -1
		}
		hpos++
		rpos = hpos
	}

	// Now calculate the cursor position, the indent
	// must be less than the line characters.
	for cpos = hpos; cpos < s.line.Len(); cpos++ {
		if (*s.line)[cpos] == '\n' {
			break
		}

		if hpos+indent <= cpos {
			break
		}
	}

	// That cursor position might be bigger than the line itself:
	// it should be controlled when the line is redisplayed.
	cpos = rpos + cpos - hpos

	return cpos
}

// Len returns the length of the current selection. If any
// of begin/end pos is not set, the cursor position is used.
func (s *Selection) Len() int {
	if s.line.Len() == 0 || (s.bpos == s.epos) {
		return 0
	}

	bpos, epos := s.Pos()
	buf := (*s.line)[bpos:epos]

	return len(buf)
}

// Text returns the current selection as a string, but does not reset it.
func (s *Selection) Text() string {
	if s.line.Len() == 0 {
		return ""
	}

	bpos, epos := s.Pos()
	if bpos == -1 || epos == -1 {
		return ""
	}

	return string((*s.line)[bpos:epos])
}

// Pop returns the contents of the current selection as a string,
// as well as its begin and end position in the line, and the cursor
// position as given by the Cursor() method. Then, the selection is reset.
func (s *Selection) Pop() (buf string, bpos, epos, cpos int) {
	if s.line.Len() == 0 {
		return "", -1, -1, 0
	}

	defer s.Reset()

	bpos, epos = s.Pos()
	if bpos == -1 || epos == -1 {
		return "", -1, -1, 0
	}

	cpos = s.Cursor()
	buf = string((*s.line)[bpos:epos])

	return buf, bpos, epos, cpos
}

// InsertAt insert the contents of the selection into the line, between the
// begin and end position, effectively deleting everything in between those.
//
// If either or these positions is equal to -1, the selection content
// is inserted at the other position. If both are negative, nothing is done.
// This is equivalent to selection.Pop(), and line.InsertAt() combined.
//
// After insertion, the selection is reset.
func (s *Selection) InsertAt(bpos, epos int) {
	bpos, epos, valid := s.checkRange(bpos, epos)
	if !valid {
		return
	}

	// Get and reset the selection.
	defer s.Reset()

	buf := s.Text()

	switch {
	case epos == -1, bpos == epos:
		s.line.Insert(bpos, []rune(buf)...)
	default:
		s.line.InsertBetween(bpos, epos, []rune(buf)...)
	}
}

// Surround surrounds the selection with a begin and end character,
// effectively inserting those characters into the current input line.
// If epos is greater than the line length, the line length is used.
// After insertion, the selection is reset.
func (s *Selection) Surround(bchar, echar rune) {
	if s.line.Len() == 0 || s.Len() == 0 {
		return
	}

	defer s.Reset()

	var buf []rune
	buf = append(buf, bchar)
	buf = append(buf, []rune(s.Text())...)
	buf = append(buf, echar)

	// The begin and end positions of the selection
	// is where we insert our new buffer.
	bpos, epos := s.Pos()
	if bpos == -1 || epos == -1 {
		return
	}

	s.line.InsertBetween(bpos, epos, buf...)
}

// SelectAWord selects a word around the current cursor position,
// selecting leading or trailing spaces depending on where the cursor
// is: if on a blank space, in a word, or at the end of the line.
func (s *Selection) SelectAWord() (bpos, epos int) {
	if s.line.Len() == 0 {
		return
	}

	bpos = s.cursor.Pos()
	cpos := bpos

	spaceBefore, spaceUnder := s.spacesAroundWord(bpos)

	bpos, epos = s.line.SelectWord(cpos)
	s.cursor.Set(epos)
	cpos = s.cursor.Pos()

	spaceAfter := cpos < s.line.Len()-1 && isSpace((*s.line)[cpos+1])

	// And only select spaces after it if the word selected is not preceded
	// by spaces as well, or if we started the selection within this word.
	bpos, _ = s.adjustWordSelection(spaceBefore, spaceUnder, spaceAfter, bpos)

	if !s.Active() || bpos < cpos {
		s.Mark(bpos)
	}

	return bpos, epos
}

// SelectABlankWord selects a bigword around the current cursor position,
// selecting leading or trailing spaces depending on where the cursor is:
// if on a blank space, in a word, or at the end of the line.
func (s *Selection) SelectABlankWord() (bpos, epos int) {
	if s.line.Len() == 0 {
		return
	}

	bpos = s.cursor.Pos()
	spaceBefore, spaceUnder := s.spacesAroundWord(bpos)

	// If we are out of a word or in the middle of one, find its beginning.
	if !spaceUnder && !spaceBefore {
		s.cursor.Inc()
		s.cursor.Move(s.line.Backward(s.line.TokenizeSpace, s.cursor.Pos()))
		bpos = s.cursor.Pos()
	} else {
		s.cursor.ToFirstNonSpace(true)
	}

	// Then go to the end of the blank word
	s.cursor.Move(s.line.ForwardEnd(s.line.TokenizeSpace, s.cursor.Pos()))
	spaceAfter := s.cursor.Pos() < s.line.Len()-1 && isSpace((*s.line)[s.cursor.Pos()+1])

	// And only select spaces after it if the word selected is not preceded
	// by spaces as well, or if we started the selection within this word.
	bpos, _ = s.adjustWordSelection(spaceBefore, spaceUnder, spaceAfter, bpos)

	if !s.Active() || bpos < s.cursor.Pos() {
		s.Mark(bpos)
	}

	return bpos, s.cursor.Pos()
}

// SelectAShellWord selects a shell word around the cursor position,
// selecting leading or trailing spaces depending on where the cursor
// is: if on a blank space, in a word, or at the end of the line.
func (s *Selection) SelectAShellWord() (bpos, epos int) {
	if s.line.Len() == 0 {
		return
	}

	s.cursor.CheckCommand()
	s.cursor.ToFirstNonSpace(true)

	sBpos, sEpos := s.line.SurroundQuotes(true, s.cursor.Pos())
	dBpos, dEpos := s.line.SurroundQuotes(false, s.cursor.Pos())

	mark, cpos := strutil.AdjustSurroundQuotes(dBpos, dEpos, sBpos, sEpos)

	// If none of the quotes matched, use blank word
	if mark == -1 && cpos == -1 {
		mark, cpos = s.line.SelectBlankWord(s.cursor.Pos())
	}

	s.cursor.Set(mark)

	// The quotes might be followed by non-blank characters,
	// in which case we must keep expanding our selection.
	for {
		spaceBefore := mark > 0 && isSpace((*s.line)[mark-1])
		if spaceBefore {
			s.cursor.Dec()
			s.cursor.ToFirstNonSpace(false)
			s.cursor.Inc()

			break
		} else if mark == 0 {
			break
		}

		s.cursor.Move(s.line.Backward(s.line.TokenizeSpace, s.cursor.Pos()))
		mark = s.cursor.Pos()
	}

	bpos = s.cursor.Pos()
	s.cursor.Set(cpos)

	// Adjust if no spaces after.
	for {
		spaceAfter := cpos < s.line.Len()-1 && isSpace((*s.line)[cpos+1])
		if spaceAfter || cpos == s.line.Len()-1 {
			break
		}

		s.cursor.Move(s.line.ForwardEnd(s.line.TokenizeSpace, cpos))
		cpos = s.cursor.Pos()
	}

	// Else set the region inside those quotes
	if !s.Active() || bpos < s.cursor.Pos() {
		s.Mark(bpos)
	}

	return bpos, cpos
}

// SelectKeyword attempts to find a pattern in the current blank word
// around the current cursor position, using various regular expressions.
// Repeatedly calling this function will cycle through all regex matches,
// or if a matcher captured multiple subgroups, through each of those groups.
//
// Those are, in the order in which they are tried:
// URI / URL / Domain|IPv4|IPv6 / URL path component / URL parameters.
//
// The returned positions are the beginning and end positions of the match
// on the line (absolute position, not relative to cursor), or if no matcher
// succeeds, the bpos and epos parameters are returned unchanged.
// If found is true, it means a match occurred, otherwise false is returned.
func (s *Selection) SelectKeyword(bpos, epos int, next bool) (kbpos, kepos int, match bool) {
	if s.line.Len() == 0 {
		return bpos, epos, false
	}

	selection := (*s.line)[bpos:epos]

	_, match, kbpos, kepos = s.matchKeyword(selection, bpos, next)
	if !match {
		return bpos, epos, false
	}

	// Always check the validity of the selection
	kbpos, kepos, match = s.checkRange(kbpos, kepos)
	if !match {
		return bpos, epos, false
	}

	// Mark the selection at its beginning
	// and move the cursor to its end.
	s.Mark(kbpos)

	return kbpos, kepos, true
}

// ReplaceWith replaces all characters of the line within the current
// selection range by applying to each rune the provided replacer function.
// After replacement, the selection is reset.
func (s *Selection) ReplaceWith(replacer func(r rune) rune) {
	if s.line.Len() == 0 || s.Len() == 0 {
		return
	}

	defer s.Reset()

	bpos, epos := s.Pos()
	if bpos == -1 || epos == -1 {
		return
	}

	for pos := bpos; pos < epos; pos++ {
		char := (*s.line)[pos]
		char = replacer(char)
		(*s.line)[pos] = char
	}
}

// Cut deletes the current selection from the line, updates the cursor position
// and returns the deleted content, which can then be passed to the shell registers.
// After deletion, the selection is reset.
func (s *Selection) Cut() (buf string) {
	if s.line.Len() == 0 {
		return
	}

	defer s.Reset()

	switch {
	case len(s.surrounds) > 0:
		offset := 0

		for _, surround := range s.surrounds {
			s.line.CutRune(surround.bpos - offset)
			offset++
		}

	default:
		bpos, epos := s.Pos()
		if bpos == -1 || epos == -1 {
			return
		}

		buf = s.Text()

		s.line.Cut(bpos, epos)
	}

	return
}

// Surrounds returns all surround-selected regions contained by the selection.
func (s *Selection) Surrounds() []Selection {
	return s.surrounds
}

// Highlights returns the highlighting sequences for the selection.
func (s *Selection) Highlights() (fg, bg string) {
	return s.fg, s.bg
}

// HighlightMatchers adds highlighting to matching
// parens when the cursor is on one of them.
func HighlightMatchers(sel *Selection) {
	cpos := sel.cursor.Pos()

	if sel.line.Len() == 0 || cpos == sel.line.Len() {
		return
	}

	if strutil.IsBracket(sel.cursor.Char()) {
		var adjust, ppos int

		split, index, pos := sel.line.TokenizeBlock(cpos)

		switch {
		case len(split) == 0:
			return
		case pos == 0 && len(split) > index:
			adjust = len(split[index])
		default:
			adjust = pos * -1
		}

		ppos = cpos + adjust

		sel.surrounds = append(sel.surrounds, Selection{
			Type:   "matcher",
			active: true,
			visual: true,
			bpos:   ppos,
			epos:   ppos,
			bg:     color.Fmt("240"),
			line:   sel.line,
			cursor: sel.cursor,
		})
	}
}

// ResetMatchers is used by the display engine
// to reset matching parens highlighting regions.
func ResetMatchers(sel *Selection) {
	var surrounds []Selection

	for _, surround := range sel.surrounds {
		if surround.Type == "matcher" {
			continue
		}

		surrounds = append(surrounds, surround)
	}

	sel.surrounds = surrounds
}

// Reset makes the current selection inactive, resetting all of its values.
func (s *Selection) Reset() {
	s.Type = ""
	s.active = false
	s.visual = false
	s.visualLine = false
	s.bpos = -1
	s.epos = -1
	s.kpos = 0
	s.fg = ""
	s.bg = ""

	// Get rid of all surround regions but matcher ones.
	surrounds := make([]Selection, 0)

	for _, surround := range s.surrounds {
		if surround.Type != "matcher" {
			continue
		}

		surrounds = append(surrounds, surround)
	}

	s.surrounds = surrounds
}

func (s *Selection) checkRange(bpos, epos int) (int, int, bool) {
	// Return on some on unfixable cases.
	switch {
	case s.line.Len() == 0:
		return -1, -1, false
	case bpos < 0 && epos < 0:
		return -1, -1, false
	case bpos > s.line.Len() && epos > s.line.Len():
		return -1, -1, false
	}

	// Adjust positive out-of-range values
	if bpos > s.line.Len() {
		bpos = s.line.Len()
	}

	if epos > s.line.Len() {
		epos = s.line.Len()
	}

	// Adjust negative values when pending.
	if bpos < 0 {
		bpos, epos = epos, -1
	} else if epos < 0 {
		epos = -1
	}

	// And reorder if inversed.
	if bpos > epos && epos != -1 {
		bpos, epos = epos, bpos
	}

	return bpos, epos, true
}

func (s *Selection) selectToCursor(bpos int) (int, int) {
	var epos int

	// The cursor might be now before its original mark,
	// in which case we invert before doing any move.
	if s.cursor.Pos() < bpos {
		bpos, epos = s.cursor.Pos(), bpos
	} else {
		epos = s.cursor.Pos()
	}

	if s.visualLine {
		// To beginning of line
		for bpos--; bpos >= 0; bpos-- {
			if (*s.line)[bpos] == '\n' {
				bpos++
				break
			}
		}

		if bpos == -1 {
			bpos = 0
		}

		// To end of line
		for ; epos < s.line.Len(); epos++ {
			if epos == -1 {
				epos = 0
			}

			if (*s.line)[epos] == '\n' {
				break
			}
		}
	}

	// Check again in case the visual line inverted both.
	if bpos > epos {
		bpos, epos = epos, bpos
	}

	return bpos, epos
}

func (s *Selection) spacesAroundWord(cpos int) (before, under bool) {
	under = isSpace(s.cursor.Char())
	before = cpos > 0 && isSpace((*s.line)[cpos-1])

	return
}

func isSpace(char rune) bool {
	return unicode.IsSpace(char) && char != inputrc.Newline
}

// adjustWordSelection adjust the beginning and end of a word (blank or not) selection, depending
// on whether it's surrounded by spaces, and if selection started from a whitespace or within word.
func (s *Selection) adjustWordSelection(_, under, after bool, bpos int) (int, int) {
	var epos int

	if after && !under {
		s.cursor.Inc()
		s.cursor.ToFirstNonSpace(true)
		s.cursor.Dec()
	} else if !after {
		epos = s.cursor.Pos()
		s.cursor.Set(bpos - 1)
		s.cursor.ToFirstNonSpace(false)
		s.cursor.Inc()
		bpos = s.cursor.Pos()
		s.cursor.Set(epos)
	}

	epos = s.cursor.Pos()

	return bpos, epos
}

func (s *Selection) matchKeyword(buf []rune, bbpos int, next bool) (name string, found bool, bpos, epos int) {
	matchersNames := []string{
		"URI",
	}

	matchers := map[string]*regexp.Regexp{
		"URI": regexp.MustCompile(`https?:\/\/(?:www\.)?([-a-zA-Z0-9@:%._\+~#=]{2,256}\.[a-z]{2,6}\b)*(\/[\/\d\w\.-]*)*(?:[\?])*(.+)*`),
	}

	// The matcher name and all the capturing subgroups it found.
	var matcherName string
	var groups []int

	// Always preload the current/last target matcher used.
	if s.kpos > 0 && s.kpos <= len(matchersNames) {
		matcherName = matchersNames[s.kpos-1]
		matcher := matchers[matcherName]

		groups = s.runMatcher(buf, matcher)
	}

	// Either increment/decrement the counter (switch the matcher altogether),
	// or cycle through the match groups for the current matcher.
	if sbpos, sepos, cycled := s.cycleSubgroup(groups, bbpos, next); cycled {
		return matcherName, true, sbpos, sepos
	}

	// We did not cycle through subgroups, so cycle the matchers.
	prevPos := s.kpos
	s.kmpos = 1

	if next {
		s.kpos++
	} else {
		s.kpos--
	}

	if s.kpos > len(matchersNames) {
		s.kpos = 1
	}

	if s.kpos < 1 {
		s.kpos = len(matchersNames)
	}

	// Prepare the cycling functions, increments and counters.
	var (
		kpos = s.kpos
		done func(i int) bool
		move func(inc int) int
	)

	if next {
		done = func(i int) bool { return i <= len(matchersNames) }
		move = func(inc int) int { return kpos + 1 }
	} else {
		done = func(i int) bool { return i > 0 }
		move = func(inc int) int { return kpos - 1 }
	}

	// Try the different matchers until one succeeds, and select the first/last capturing group.
	for done(kpos) {
		matcherName = matchersNames[kpos-1]
		matcher := matchers[matcherName]

		groups = s.runMatcher(buf, matcher)

		if len(groups) <= 1 {
			kpos = move(kpos)
			continue
		}

		if !next && prevPos == 1 {
			s.kmpos = len(groups)

			return matcherName, true, bbpos + groups[len(groups)-2], bbpos + groups[len(groups)-1]
		}

		return matcherName, true, bbpos + groups[0], bbpos + groups[1]
	}

	return "", false, -1, -1
}

func (s *Selection) cycleSubgroup(groups []int, bbpos int, next bool) (bpos, epos int, cycled bool) {
	canCycleSubgroup := (next && s.kmpos < len(groups)-1) || (!next && s.kmpos > 1)

	if !canCycleSubgroup {
		return
	}

	if next {
		s.kmpos++
	} else {
		s.kmpos--
	}

	return bbpos + groups[s.kmpos-1], bbpos + groups[s.kmpos], true
}

func (s *Selection) runMatcher(buf []rune, matcher *regexp.Regexp) (groups []int) {
	if matcher == nil {
		return
	}

	matches := matcher.FindAllStringSubmatchIndex(string(buf), -1)
	if len(matches) == 0 {
		return
	}

	return matches[0]
}

// Tested URL / IP regexp matchers, not working as well as the current ones
//
// "URL": regexp.MustCompile(`([\w+]+\:\/\/)?([\w\d-]+\.)*[\w-]+[\.\:]\w+([\/\?\=\&\#.]?[\w-]+)*\/?`),

// "Domain|IP": regexp.MustCompile(ipv4_regex + `|` + ipv6_regex + `|` + domain_regex),
// (http(s?):\/\/)?(((www\.)?+[a-zA-Z0-9\.\-\_]+(\.[a-zA-Z]{2,3})+)|(\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b))(\/[a-zA-Z0-9\_\-\s\.\/\?\%\#\&\=]*)?
// "Domain|IP": regexp.MustCompile(`((www\.)?[a-zA-Z0-9\.\-\_]+(\.[a-zA-Z]{2,3})+)|(\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b)`),

// "URL path": ,
// "URL parameters": regexp.MustCompile(`[(\?|\&)]([^=]+)\=([^&#]+)`),

// ipv6Regex   = `^(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))`
// ipv4Regex   = `^(((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.|$)){4})`
// domainRegex = `^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]`
