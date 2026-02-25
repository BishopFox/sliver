package edit

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/x/ansi"
	"golang.org/x/term"
)

type editorMode int

const (
	modeNormal editorMode = iota
	modeInsert
	modeCommand
)

type exitAction int

const (
	actionNone exitAction = iota
	actionQuit
	actionSaveQuit
)

type editorResult struct {
	Content  string
	Modified bool
	Action   exitAction
	Force    bool
}

type editorModel struct {
	lines             [][]rune
	row               int
	col               int
	top               int
	left              int
	width             int
	height            int
	mode              editorMode
	command           string
	commandPrefix     rune
	pending           rune
	dirty             bool
	filename          string
	showLineNumbers   bool
	syntaxName        string
	lexer             chroma.Lexer
	formatter         chroma.Formatter
	style             *chroma.Style
	highlighted       []string
	highlightOn       bool
	highlightDirty    bool
	lastSearch        string
	lastSearchForward bool
	action            exitAction
	forceQuit         bool
	message           string
	clearMessage      bool
}

var (
	lineStyle = lipgloss.NewStyle()
)

const (
	ansiReverseOn  = "\x1b[7m"
	ansiReverseOff = "\x1b[0m"
)

func newEditorModel(content, filename string, lexer chroma.Lexer, syntaxName string, showLineNumbers bool) *editorModel {
	lines := splitLines(content)
	model := &editorModel{
		lines:           lines,
		mode:            modeNormal,
		filename:        filename,
		showLineNumbers: showLineNumbers,
		commandPrefix:   ':',
	}
	model.setSyntax(lexer, syntaxName)
	return model
}

func (m *editorModel) Init() tea.Cmd {
	if m.width == 0 || m.height == 0 {
		if w, h, err := term.GetSize(0); err == nil && w > 0 && h > 0 {
			m.width = w
			m.height = h
		} else {
			m.width = 80
			m.height = 24
		}
		m.ensureCursorVisible()
	}
	return nil
}

func (m *editorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureCursorVisible()
		return m, nil
	case tea.KeyPressMsg:
		if m.clearMessage {
			m.message = ""
			m.clearMessage = false
		}
		key := msg.Key()
		if key.Mod.Contains(tea.ModCtrl) && key.Code == 'c' {
			m.action = actionQuit
			return m, tea.Quit
		}
		switch m.mode {
		case modeCommand:
			return m.handleCommand(key)
		case modeInsert:
			return m.handleInsert(key)
		default:
			return m.handleNormal(key)
		}
	}
	return m, nil
}

func (m *editorModel) View() tea.View {
	if m.width == 0 || m.height == 0 {
		if w, h, err := term.GetSize(0); err == nil && w > 0 && h > 0 {
			m.width = w
			m.height = h
		} else {
			m.width = 80
			m.height = 24
		}
		m.ensureCursorVisible()
	}

	textHeight := m.textHeight()
	var b strings.Builder
	for i := 0; i < textHeight; i++ {
		lineIndex := m.top + i
		b.WriteString(m.renderLine(lineIndex))
		if i < textHeight-1 {
			b.WriteByte('\n')
		}
	}

	if m.height < 2 {
		return editorView(b.String())
	}

	b.WriteByte('\n')
	b.WriteString(renderStatusBar(m.width, m.statusLine()))

	if m.height < 3 {
		return editorView(b.String())
	}

	b.WriteByte('\n')
	b.WriteString(lineStyle.Width(m.width).Render(m.commandLine()))
	return editorView(b.String())
}

func renderStatusBar(width int, text string) string {
	return ansiReverseOn + lineStyle.Width(width).Render(text) + ansiReverseOff
}

func renderCursorCell(cell string) string {
	return ansiReverseOn + cell + ansiReverseOff
}

func editorView(content string) tea.View {
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m *editorModel) result() editorResult {
	return editorResult{
		Content:  m.content(),
		Modified: m.dirty,
		Action:   m.action,
		Force:    m.forceQuit,
	}
}

func (m *editorModel) handleCommand(key tea.Key) (tea.Model, tea.Cmd) {
	switch key.Code {
	case tea.KeyEsc:
		m.mode = modeNormal
		m.command = ""
		m.commandPrefix = ':'
	case tea.KeyEnter:
		if m.commandPrefix == '/' {
			m.executeSearch(strings.TrimSpace(m.command))
		} else {
			m.executeCommand(strings.TrimSpace(m.command))
		}
		m.command = ""
		m.mode = modeNormal
		m.commandPrefix = ':'
		if m.action != actionNone {
			return m, tea.Quit
		}
	case tea.KeyBackspace:
		m.command = popRune(m.command)
	default:
		if key.Text != "" {
			m.command += key.Text
		}
	}
	return m, nil
}

func (m *editorModel) handleInsert(key tea.Key) (tea.Model, tea.Cmd) {
	switch key.Code {
	case tea.KeyEsc:
		m.mode = modeNormal
	case tea.KeyEnter:
		m.insertNewline()
	case tea.KeyBackspace:
		m.backspace()
	case tea.KeyDelete:
		m.deleteChar()
	case tea.KeyLeft:
		m.moveLeft()
	case tea.KeyRight:
		m.moveRight()
	case tea.KeyUp:
		m.moveUp()
	case tea.KeyDown:
		m.moveDown()
	case tea.KeyTab:
		m.insertRune('\t')
	default:
		for _, r := range key.Text {
			m.insertRune(r)
		}
	}
	m.ensureCursorVisible()
	return m, nil
}

func (m *editorModel) handleNormal(key tea.Key) (tea.Model, tea.Cmd) {
	switch key.Code {
	case tea.KeyEsc:
		m.pending = 0
	case tea.KeyLeft:
		m.moveLeft()
	case tea.KeyRight:
		m.moveRight()
	case tea.KeyUp:
		m.moveUp()
	case tea.KeyDown:
		m.moveDown()
	default:
		for _, r := range key.Text {
			if m.handleNormalRune(r) {
				break
			}
		}
	}
	m.ensureCursorVisible()
	return m, nil
}

func (m *editorModel) handleNormalRune(r rune) bool {
	if m.pending != 0 {
		handled := true
		switch m.pending {
		case 'd':
			if r == 'd' {
				m.deleteLine()
			} else {
				handled = false
			}
		case 'g':
			if r == 'g' {
				m.gotoTop()
			} else {
				handled = false
			}
		default:
			handled = false
		}
		m.pending = 0
		if handled {
			return true
		}
	}

	switch r {
	case 'h':
		m.moveLeft()
	case 'j':
		m.moveDown()
	case 'k':
		m.moveUp()
	case 'l':
		m.moveRight()
	case 'w':
		m.moveWordForward()
	case 'b':
		m.moveWordBackward()
	case 'e':
		m.moveWordEnd()
	case '^':
		m.moveLineFirstNonWhitespace()
	case '0':
		m.moveLineStart()
	case '$':
		m.moveLineEnd()
	case 'i':
		m.mode = modeInsert
	case 'a':
		m.moveRightNoWrap()
		m.mode = modeInsert
	case 'A':
		m.moveLineEnd()
		m.mode = modeInsert
	case 'I':
		m.moveLineStart()
		m.mode = modeInsert
	case 'o':
		m.openLineBelow()
	case 'O':
		m.openLineAbove()
	case 'x':
		m.deleteCharNormal()
	case 'd':
		m.pending = 'd'
	case 'g':
		m.pending = 'g'
	case 'G':
		m.gotoBottom()
	case 'n':
		m.searchNext(true)
	case 'N':
		m.searchNext(false)
	case ':':
		m.mode = modeCommand
		m.command = ""
		m.commandPrefix = ':'
	case '/':
		m.mode = modeCommand
		m.command = ""
		m.commandPrefix = '/'
	}

	return false
}

func (m *editorModel) executeCommand(cmd string) {
	if isDigits(cmd) {
		target, err := strconv.Atoi(cmd)
		if err == nil {
			m.gotoLine(target)
			return
		}
	}
	switch strings.ToLower(cmd) {
	case "":
		return
	case "q":
		m.action = actionQuit
	case "q!":
		m.action = actionQuit
		m.forceQuit = true
	case "wq", "x":
		m.action = actionSaveQuit
	case "w":
		m.message = "Use :wq to upload and exit"
		m.clearMessage = true
	case "n":
		m.showLineNumbers = !m.showLineNumbers
		m.ensureCursorVisible()
	default:
		m.message = fmt.Sprintf("Unknown command: %s", cmd)
		m.clearMessage = true
	}
}

func (m *editorModel) renderLine(index int) string {
	totalWidth := m.textWidth()
	if totalWidth <= 0 {
		return ""
	}
	prefixWidth := m.lineNumberWidth()
	contentWidth := totalWidth - prefixWidth
	if contentWidth < 0 {
		contentWidth = 0
	}
	prefix := m.linePrefix(index, prefixWidth)

	if index >= len(m.lines) {
		return lineStyle.Width(totalWidth).Render(prefix + "~")
	}

	line := m.lines[index]
	start := clamp(m.left, 0, len(line))
	end := clamp(start+contentWidth, 0, len(line))
	if m.highlightOn {
		m.ensureHighlighted()
		highlighted := m.highlightedLine(index)
		if highlighted != "" {
			lineText := ansi.Cut(highlighted, start, end)
			if index == m.row {
				lineText = m.applyCursor(lineText, highlighted, line, start, end)
			}
			return lineStyle.Width(totalWidth).Render(prefix + lineText)
		}
	}

	visible := line[start:end]
	lineText := string(visible)

	if index == m.row {
		cursor := clamp(m.col, 0, len(line))
		if cursor >= start && cursor <= start+contentWidth {
			rel := cursor - start
			if rel < len(visible) {
				lineText = string(visible[:rel]) + renderCursorCell(string(visible[rel])) + string(visible[rel+1:])
			} else {
				lineText = string(visible) + renderCursorCell(" ")
			}
		}
	}

	return lineStyle.Width(totalWidth).Render(prefix + lineText)
}

func (m *editorModel) statusLine() string {
	mode := "NORMAL"
	switch m.mode {
	case modeInsert:
		mode = "INSERT"
	case modeCommand:
		mode = "COMMAND"
	}
	fileName := baseName(m.filename)
	if fileName == "" {
		fileName = "[No Name]"
	}
	modified := ""
	if m.dirty {
		modified = " [+]"
	}
	syntax := ""
	if m.syntaxName != "" {
		syntax = "  " + m.syntaxName
	}
	return fmt.Sprintf(" %s  %s  Ln %d, Col %d%s%s", mode, fileName, m.row+1, m.col+1, modified, syntax)
}

func (m *editorModel) commandLine() string {
	if m.mode == modeCommand {
		prefix := ":"
		if m.commandPrefix == '/' {
			prefix = "/"
		}
		return prefix + m.command
	}
	if m.message != "" {
		return m.message
	}
	if m.mode == modeInsert {
		return "ESC: normal  / search  :wq save+quit  :q quit  :n line numbers  :<num> goto line"
	}
	return "i: insert  h/j/k/l: move  / search  :wq save+quit  :q quit  :n line numbers  :<num> goto line"
}

func (m *editorModel) content() string {
	var b strings.Builder
	for i, line := range m.lines {
		b.WriteString(string(line))
		if i < len(m.lines)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (m *editorModel) textHeight() int {
	if m.height <= 2 {
		return 1
	}
	return m.height - 2
}

func (m *editorModel) textWidth() int {
	if m.width <= 0 {
		return 1
	}
	return m.width
}

func (m *editorModel) lineNumberWidth() int {
	if !m.showLineNumbers {
		return 0
	}
	total := len(m.lines)
	if total < 1 {
		total = 1
	}
	return len(fmt.Sprintf("%d", total)) + 1
}

func (m *editorModel) linePrefix(index, width int) string {
	if width == 0 {
		return ""
	}
	if index >= len(m.lines) {
		return fmt.Sprintf("%*s ", width-1, "")
	}
	return fmt.Sprintf("%*d ", width-1, index+1)
}

func (m *editorModel) ensureCursorVisible() {
	if m.row < 0 {
		m.row = 0
	}
	if m.row >= len(m.lines) {
		m.row = len(m.lines) - 1
	}
	if m.row < 0 {
		m.row = 0
	}
	if len(m.lines) == 0 {
		m.lines = [][]rune{{}}
		m.row = 0
		m.col = 0
	}

	lineLen := len(m.lines[m.row])
	if m.col < 0 {
		m.col = 0
	}
	if m.col > lineLen {
		m.col = lineLen
	}

	height := m.textHeight()
	if m.row < m.top {
		m.top = m.row
	} else if m.row >= m.top+height {
		m.top = m.row - height + 1
	}
	if m.top < 0 {
		m.top = 0
	}

	width := m.textWidth() - m.lineNumberWidth()
	if width < 1 {
		width = 1
	}
	if m.col < m.left {
		m.left = m.col
	} else if m.col >= m.left+width {
		m.left = m.col - width + 1
	}
	if m.left < 0 {
		m.left = 0
	}
}

func (m *editorModel) moveLeft() {
	if m.col > 0 {
		m.col--
	}
}

func (m *editorModel) moveRight() {
	lineLen := len(m.lines[m.row])
	if m.col < lineLen {
		m.col++
	}
}

func (m *editorModel) moveRightNoWrap() {
	lineLen := len(m.lines[m.row])
	if m.col < lineLen {
		m.col++
	}
}

func (m *editorModel) moveUp() {
	if m.row > 0 {
		m.row--
		m.clampCol()
	}
}

func (m *editorModel) moveDown() {
	if m.row < len(m.lines)-1 {
		m.row++
		m.clampCol()
	}
}

func (m *editorModel) moveLineStart() {
	m.col = 0
}

func (m *editorModel) moveLineFirstNonWhitespace() {
	line := m.lines[m.row]
	col := 0
	for col < len(line) {
		if line[col] != ' ' && line[col] != '\t' {
			break
		}
		col++
	}
	m.col = col
}

func (m *editorModel) moveLineEnd() {
	m.col = len(m.lines[m.row])
}

func (m *editorModel) gotoTop() {
	m.row = 0
	m.clampCol()
}

func (m *editorModel) gotoBottom() {
	if len(m.lines) == 0 {
		m.row = 0
		m.col = 0
		return
	}
	m.row = len(m.lines) - 1
	m.clampCol()
}

func (m *editorModel) gotoLine(lineNumber int) {
	if lineNumber < 1 {
		lineNumber = 1
	}
	if len(m.lines) == 0 {
		m.row = 0
		m.col = 0
		return
	}
	if lineNumber > len(m.lines) {
		lineNumber = len(m.lines)
	}
	m.row = lineNumber - 1
	m.clampCol()
	m.ensureCursorVisible()
}

func (m *editorModel) moveWordForward() {
	row := m.row
	col := m.col
	for {
		if row >= len(m.lines) {
			return
		}
		line := m.lines[row]
		if col >= len(line) {
			row++
			col = 0
			continue
		}
		if isWordChar(line[col]) {
			for col < len(line) && isWordChar(line[col]) {
				col++
			}
		}
		for col < len(line) && !isWordChar(line[col]) {
			col++
		}
		if col < len(line) {
			m.row = row
			m.col = col
			return
		}
		row++
		col = 0
	}
}

func (m *editorModel) moveWordBackward() {
	row := m.row
	col := m.col - 1
	for {
		if row < 0 {
			return
		}
		line := m.lines[row]
		if col >= len(line) {
			col = len(line) - 1
		}
		for col >= 0 && (col >= len(line) || !isWordChar(line[col])) {
			col--
		}
		for col >= 0 && isWordChar(line[col]) {
			col--
		}
		if col+1 >= 0 && col+1 < len(line) {
			m.row = row
			m.col = col + 1
			return
		}
		row--
		if row >= 0 {
			col = len(m.lines[row]) - 1
		}
	}
}

func (m *editorModel) moveWordEnd() {
	row := m.row
	col := m.col
	for {
		if row >= len(m.lines) {
			return
		}
		line := m.lines[row]
		if col >= len(line) {
			row++
			col = 0
			continue
		}
		if isWordChar(line[col]) {
			for col+1 < len(line) && isWordChar(line[col+1]) {
				col++
			}
			m.row = row
			m.col = col
			return
		}
		for col < len(line) && !isWordChar(line[col]) {
			col++
		}
		if col < len(line) {
			continue
		}
		row++
		col = 0
	}
}

func (m *editorModel) clampCol() {
	lineLen := len(m.lines[m.row])
	if m.col > lineLen {
		m.col = lineLen
	}
}

func (m *editorModel) insertRune(r rune) {
	line := m.lines[m.row]
	if m.col > len(line) {
		m.col = len(line)
	}
	line = append(line[:m.col], append([]rune{r}, line[m.col:]...)...)
	m.lines[m.row] = line
	m.col++
	m.dirty = true
	m.highlightDirty = true
}

func (m *editorModel) insertNewline() {
	line := m.lines[m.row]
	if m.col > len(line) {
		m.col = len(line)
	}
	left := append([]rune(nil), line[:m.col]...)
	right := append([]rune(nil), line[m.col:]...)
	m.lines[m.row] = left
	m.lines = append(m.lines[:m.row+1], append([][]rune{right}, m.lines[m.row+1:]...)...)
	m.row++
	m.col = 0
	m.dirty = true
	m.highlightDirty = true
}

func (m *editorModel) backspace() {
	if m.col > 0 {
		line := m.lines[m.row]
		line = append(line[:m.col-1], line[m.col:]...)
		m.lines[m.row] = line
		m.col--
		m.dirty = true
		m.highlightDirty = true
		return
	}
	if m.row == 0 {
		return
	}
	prev := m.lines[m.row-1]
	line := m.lines[m.row]
	m.col = len(prev)
	m.lines[m.row-1] = append(prev, line...)
	m.lines = append(m.lines[:m.row], m.lines[m.row+1:]...)
	m.row--
	m.dirty = true
	m.highlightDirty = true
}

func (m *editorModel) deleteChar() {
	line := m.lines[m.row]
	if m.col < len(line) {
		line = append(line[:m.col], line[m.col+1:]...)
		m.lines[m.row] = line
		m.dirty = true
		m.highlightDirty = true
		return
	}
	if m.row < len(m.lines)-1 {
		next := m.lines[m.row+1]
		m.lines[m.row] = append(line, next...)
		m.lines = append(m.lines[:m.row+1], m.lines[m.row+2:]...)
		m.dirty = true
		m.highlightDirty = true
	}
}

func (m *editorModel) deleteCharNormal() {
	line := m.lines[m.row]
	if m.col < len(line) {
		line = append(line[:m.col], line[m.col+1:]...)
		m.lines[m.row] = line
		m.dirty = true
		m.highlightDirty = true
	}
}

func (m *editorModel) deleteLine() {
	if len(m.lines) == 0 {
		return
	}
	if len(m.lines) == 1 {
		m.lines[0] = []rune{}
		m.col = 0
		m.dirty = true
		m.highlightDirty = true
		return
	}
	m.lines = append(m.lines[:m.row], m.lines[m.row+1:]...)
	if m.row >= len(m.lines) {
		m.row = len(m.lines) - 1
	}
	m.clampCol()
	m.dirty = true
	m.highlightDirty = true
}

func (m *editorModel) openLineBelow() {
	m.lines = append(m.lines[:m.row+1], append([][]rune{{}}, m.lines[m.row+1:]...)...)
	m.row++
	m.col = 0
	m.mode = modeInsert
	m.dirty = true
	m.highlightDirty = true
}

func (m *editorModel) openLineAbove() {
	m.lines = append(m.lines[:m.row], append([][]rune{{}}, m.lines[m.row:]...)...)
	m.col = 0
	m.mode = modeInsert
	m.dirty = true
	m.highlightDirty = true
}

func (m *editorModel) setSyntax(lexer chroma.Lexer, syntaxName string) {
	if lexer == nil {
		m.lexer = nil
		m.syntaxName = "none"
		m.highlightOn = false
		m.highlighted = nil
		m.highlightDirty = false
		return
	}
	m.lexer = lexer
	m.syntaxName = syntaxName
	m.formatter = formatters.Get("terminal16m")
	if m.formatter == nil {
		m.formatter = formatters.Fallback
	}
	m.style = styles.Get("monokai")
	if m.style == nil {
		m.style = styles.Fallback
	}
	m.highlightOn = true
	m.highlightDirty = true
}

func (m *editorModel) ensureHighlighted() {
	if !m.highlightOn || !m.highlightDirty {
		return
	}
	iterator, err := m.lexer.Tokenise(nil, m.content())
	if err != nil {
		m.highlighted = nil
		m.highlightDirty = false
		return
	}
	var b strings.Builder
	if err := m.formatter.Format(&b, m.style, iterator); err != nil {
		m.highlighted = nil
		m.highlightDirty = false
		return
	}
	lines := strings.Split(b.String(), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for len(lines) < len(m.lines) {
		lines = append(lines, "")
	}
	m.highlighted = lines
	m.highlightDirty = false
}

func (m *editorModel) highlightedLine(index int) string {
	if index < 0 || index >= len(m.highlighted) {
		return ""
	}
	return m.highlighted[index]
}

func (m *editorModel) applyCursor(rendered, highlighted string, line []rune, start, end int) string {
	cursor := clamp(m.col, 0, len(line))
	if cursor < start || cursor > end {
		return rendered
	}
	if cursor == len(line) {
		left := ansi.Cut(highlighted, start, cursor)
		return left + renderCursorCell(" ")
	}
	left := ansi.Cut(highlighted, start, cursor)
	right := ansi.Cut(highlighted, cursor+1, end)
	return left + renderCursorCell(string(line[cursor])) + right
}

func (m *editorModel) executeSearch(pattern string) {
	if pattern == "" {
		if m.lastSearch == "" {
			return
		}
		pattern = m.lastSearch
	}
	m.lastSearch = pattern
	m.lastSearchForward = true
	if !m.searchNext(true) {
		m.message = fmt.Sprintf("Pattern not found: %s", pattern)
		m.clearMessage = true
	}
}

func (m *editorModel) searchNext(forward bool) bool {
	pattern := m.lastSearch
	if pattern == "" {
		m.message = "No previous search"
		m.clearMessage = true
		return false
	}
	row, col, ok := m.findMatch(pattern, forward)
	if !ok {
		m.message = fmt.Sprintf("Pattern not found: %s", pattern)
		m.clearMessage = true
		return false
	}
	m.row = row
	m.col = col
	m.lastSearchForward = forward
	m.ensureCursorVisible()
	return true
}

func (m *editorModel) findMatch(pattern string, forward bool) (int, int, bool) {
	if pattern == "" {
		return 0, 0, false
	}
	pat := []rune(pattern)
	if forward {
		return m.findForward(pat)
	}
	return m.findBackward(pat)
}

func (m *editorModel) findForward(pattern []rune) (int, int, bool) {
	row := m.row
	col := m.col + 1
	for r := row; r < len(m.lines); r++ {
		line := m.lines[r]
		start := 0
		if r == row {
			start = col
		}
		if idx := indexOfRunes(line, pattern, start); idx >= 0 {
			return r, idx, true
		}
	}
	for r := 0; r <= row && r < len(m.lines); r++ {
		line := m.lines[r]
		end := len(line)
		if r == row {
			end = min(end, col)
		}
		if idx := indexOfRunes(line, pattern, 0); idx >= 0 && idx < end {
			return r, idx, true
		}
	}
	return 0, 0, false
}

func (m *editorModel) findBackward(pattern []rune) (int, int, bool) {
	row := m.row
	col := m.col - 1
	for r := row; r >= 0; r-- {
		line := m.lines[r]
		end := len(line)
		if r == row {
			end = col + 1
		}
		if idx := lastIndexOfRunes(line, pattern, end); idx >= 0 {
			return r, idx, true
		}
	}
	for r := len(m.lines) - 1; r >= row && r >= 0; r-- {
		line := m.lines[r]
		end := len(line)
		if r == row {
			end = len(line)
		}
		if idx := lastIndexOfRunes(line, pattern, end); idx >= 0 {
			return r, idx, true
		}
	}
	return 0, 0, false
}

func splitLines(content string) [][]rune {
	parts := strings.Split(content, "\n")
	if len(parts) == 0 {
		return [][]rune{{}}
	}
	lines := make([][]rune, len(parts))
	for i, line := range parts {
		lines[i] = []rune(line)
	}
	return lines
}

func popRune(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return value
	}
	return string(runes[:len(runes)-1])
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isWordChar(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func indexOfRunes(haystack, needle []rune, start int) int {
	if len(needle) == 0 {
		return -1
	}
	if start < 0 {
		start = 0
	}
	if start >= len(haystack) {
		return -1
	}
	for i := start; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func lastIndexOfRunes(haystack, needle []rune, end int) int {
	if len(needle) == 0 {
		return -1
	}
	if end > len(haystack) {
		end = len(haystack)
	}
	for i := end - len(needle); i >= 0; i-- {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func baseName(path string) string {
	if path == "" {
		return ""
	}
	clean := strings.ReplaceAll(path, "\\", "/")
	if idx := strings.LastIndex(clean, "/"); idx != -1 {
		return clean[idx+1:]
	}
	return path
}

func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
