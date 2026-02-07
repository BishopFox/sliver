package hexedit

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/util"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

type editorMode int

const (
	modeNormal editorMode = iota
	modeHexEdit
	modeASCIIEdit
	modeCommand
)

type exitAction int

const (
	actionNone exitAction = iota
	actionQuit
	actionSaveQuit
)

const bytesPerLine = 16

type editorResult struct {
	Data     []byte
	Modified bool
	Action   exitAction
	Force    bool
}

type editorModel struct {
	data          []byte
	original      []byte
	modified      []bool
	modifiedCount int
	cursor        int
	top           int
	width         int
	height        int
	mode          editorMode
	command       string
	commandPrefix rune
	pending       rune
	dirty         bool
	forceQuit     bool
	action        exitAction
	filename      string
	message       string
	clearMessage  bool
	nibbleSet     bool
	nibbleValue   byte
}

var (
	cursorStyle = lipgloss.NewStyle().Reverse(true)
	statusStyle = lipgloss.NewStyle().Reverse(true)
	lineStyle   = lipgloss.NewStyle()
)

func modifiedStyle() lipgloss.Style {
	// Read from the global console theme so theme.yaml changes are reflected.
	return console.StyleWarning
}

func modifiedCursorStyle() lipgloss.Style {
	// Read from the global console theme so theme.yaml changes are reflected.
	return console.StyleWarning.Copy().Reverse(true)
}

func newEditorModel(data []byte, filename string, offset int) *editorModel {
	copyData := append([]byte(nil), data...)
	model := &editorModel{
		data:          copyData,
		original:      append([]byte(nil), data...),
		modified:      make([]bool, len(copyData)),
		mode:          modeNormal,
		filename:      filename,
		commandPrefix: ':',
	}
	if offset > 0 && offset < len(copyData) {
		model.cursor = offset
	}
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
	case tea.KeyMsg:
		if m.clearMessage {
			m.message = ""
			m.clearMessage = false
		}
		if msg.Type == tea.KeyCtrlC {
			m.action = actionQuit
			return m, tea.Quit
		}
		switch m.mode {
		case modeCommand:
			return m.handleCommand(msg)
		case modeHexEdit:
			return m.handleHexEdit(msg)
		case modeASCIIEdit:
			return m.handleASCIIEdit(msg)
		default:
			return m.handleNormal(msg)
		}
	}
	return m, nil
}

func (m *editorModel) View() string {
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
		return b.String()
	}

	b.WriteByte('\n')
	b.WriteString(statusStyle.Width(m.width).Render(m.statusLine()))

	if m.height < 3 {
		return b.String()
	}

	b.WriteByte('\n')
	b.WriteString(lineStyle.Width(m.width).Render(m.commandLine()))
	return b.String()
}

func (m *editorModel) result() editorResult {
	data := append([]byte(nil), m.data...)
	return editorResult{
		Data:     data,
		Modified: m.dirty,
		Action:   m.action,
		Force:    m.forceQuit,
	}
}

func (m *editorModel) handleCommand(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = modeNormal
		m.command = ""
		m.commandPrefix = ':'
	case tea.KeyEnter:
		m.executeCommand(strings.TrimSpace(m.command))
		m.command = ""
		m.mode = modeNormal
		m.commandPrefix = ':'
		if m.action != actionNone {
			return m, tea.Quit
		}
	case tea.KeyBackspace:
		m.command = popRune(m.command)
	case tea.KeyRunes:
		m.command += string(msg.Runes)
	}
	return m, nil
}

func (m *editorModel) handleHexEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = modeNormal
		m.nibbleSet = false
	case tea.KeyBackspace:
		if m.nibbleSet {
			m.nibbleSet = false
			return m, nil
		}
		m.moveLeft()
	case tea.KeyLeft:
		m.moveLeft()
		m.nibbleSet = false
	case tea.KeyRight:
		m.moveRight()
		m.nibbleSet = false
	case tea.KeyUp:
		m.moveUp()
		m.nibbleSet = false
	case tea.KeyDown:
		m.moveDown()
		m.nibbleSet = false
	case tea.KeyRunes:
		for _, r := range msg.Runes {
			val, ok := hexNibble(r)
			if !ok {
				continue
			}
			if !m.nibbleSet {
				m.nibbleValue = val << 4
				m.nibbleSet = true
				continue
			}
			m.setByte(m.nibbleValue | val)
			m.nibbleSet = false
			m.moveRight()
		}
	}
	m.ensureCursorVisible()
	return m, nil
}

func (m *editorModel) handleASCIIEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = modeNormal
	case tea.KeyLeft:
		m.moveLeft()
	case tea.KeyRight:
		m.moveRight()
	case tea.KeyUp:
		m.moveUp()
	case tea.KeyDown:
		m.moveDown()
	case tea.KeyRunes:
		for _, r := range msg.Runes {
			if r < 0 || r > 0xFF {
				continue
			}
			m.setByte(byte(r))
			m.moveRight()
		}
	}
	m.ensureCursorVisible()
	return m, nil
}

func (m *editorModel) handleNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
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
	case tea.KeyRunes:
		for _, r := range msg.Runes {
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
	case '0':
		m.moveLineStart()
	case '$':
		m.moveLineEnd()
	case 'i':
		m.mode = modeHexEdit
		m.nibbleSet = false
	case 'a':
		m.mode = modeASCIIEdit
	case 'x':
		m.setByte(0x00)
	case 'g':
		m.pending = 'g'
	case 'G':
		m.gotoBottom()
	case ':':
		m.mode = modeCommand
		m.command = ""
		m.commandPrefix = ':'
	}

	return false
}

func (m *editorModel) executeCommand(cmd string) {
	if cmd == "" {
		return
	}

	if isNumericOffset(cmd) {
		if value, err := strconv.ParseInt(cmd, 0, 64); err == nil {
			m.gotoOffset(int(value))
			return
		}
	}

	switch strings.ToLower(cmd) {
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
	default:
		m.message = fmt.Sprintf("Unknown command: %s", cmd)
		m.clearMessage = true
	}
}

func (m *editorModel) renderLine(lineIndex int) string {
	totalWidth := m.textWidth()
	if totalWidth <= 0 {
		return ""
	}

	if lineIndex >= m.totalLines() {
		return lineStyle.Width(totalWidth).Render("~")
	}

	lineStart := lineIndex * bytesPerLine
	hexCells := make([]string, bytesPerLine)
	asciiCells := make([]string, bytesPerLine)

	for i := 0; i < bytesPerLine; i++ {
		idx := lineStart + i
		if idx >= len(m.data) {
			hexCells[i] = "  "
			asciiCells[i] = " "
			continue
		}
		byteVal := m.data[idx]
		hexRaw := fmt.Sprintf("%02x", byteVal)
		if idx == m.cursor && m.mode == modeHexEdit && m.nibbleSet {
			hexRaw = string([]byte{hexChar(m.nibbleValue >> 4), '_'})
		}
		asciiRaw := string(printableByte(byteVal))
		isModified := m.isModified(idx)
		if idx == m.cursor {
			if isModified {
				hexCells[i] = modifiedCursorStyle().Render(hexRaw)
				asciiCells[i] = modifiedCursorStyle().Render(asciiRaw)
			} else {
				hexCells[i] = cursorStyle.Render(hexRaw)
				asciiCells[i] = cursorStyle.Render(asciiRaw)
			}
			continue
		}
		if isModified {
			hexCells[i] = modifiedStyle().Render(hexRaw)
			asciiCells[i] = modifiedStyle().Render(asciiRaw)
			continue
		}
		hexCells[i] = hexRaw
		asciiCells[i] = asciiRaw
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("%08x", lineStart))
	b.WriteString(": ")
	for i := 0; i < bytesPerLine; i++ {
		if i == 8 {
			b.WriteByte(' ')
		}
		b.WriteString(hexCells[i])
		if i < bytesPerLine-1 {
			b.WriteByte(' ')
		}
	}
	b.WriteString("  |")
	for i := 0; i < bytesPerLine; i++ {
		b.WriteString(asciiCells[i])
	}
	b.WriteString("|")

	return lineStyle.Width(totalWidth).Render(b.String())
}

func (m *editorModel) statusLine() string {
	mode := "NORMAL"
	switch m.mode {
	case modeHexEdit:
		mode = "HEX"
	case modeASCIIEdit:
		mode = "ASCII"
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
	cursor := m.cursor
	return fmt.Sprintf(" %s  %s  Off 0x%08x (%d)  Size %s%s", mode, fileName, cursor, cursor, util.ByteCountBinary(int64(len(m.data))), modified)
}

func (m *editorModel) commandLine() string {
	if m.mode == modeCommand {
		return ":" + m.command
	}
	if m.message != "" {
		return m.message
	}
	switch m.mode {
	case modeHexEdit:
		return "ESC: normal  hex: 0-9/a-f  :wq save+quit  :q quit"
	case modeASCIIEdit:
		return "ESC: normal  type to edit bytes  :wq save+quit  :q quit"
	default:
		return "i: hex edit  a: ascii edit  h/j/k/l move  x zero  :wq save+quit  :q quit"
	}
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

func (m *editorModel) totalLines() int {
	if len(m.data) == 0 {
		return 1
	}
	return (len(m.data) + bytesPerLine - 1) / bytesPerLine
}

func (m *editorModel) ensureCursorVisible() {
	if len(m.data) == 0 {
		m.cursor = 0
		m.top = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.data) {
		m.cursor = len(m.data) - 1
	}

	line := m.cursor / bytesPerLine
	height := m.textHeight()
	if line < m.top {
		m.top = line
	} else if line >= m.top+height {
		m.top = line - height + 1
	}
	if m.top < 0 {
		m.top = 0
	}
	maxTop := m.totalLines() - height
	if maxTop < 0 {
		maxTop = 0
	}
	if m.top > maxTop {
		m.top = maxTop
	}
}

func (m *editorModel) moveLeft() {
	if len(m.data) == 0 {
		return
	}
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *editorModel) moveRight() {
	if len(m.data) == 0 {
		return
	}
	if m.cursor < len(m.data)-1 {
		m.cursor++
	}
}

func (m *editorModel) moveUp() {
	if len(m.data) == 0 {
		return
	}
	if m.cursor >= bytesPerLine {
		m.cursor -= bytesPerLine
	}
}

func (m *editorModel) moveDown() {
	if len(m.data) == 0 {
		return
	}
	if m.cursor+bytesPerLine < len(m.data) {
		m.cursor += bytesPerLine
	}
}

func (m *editorModel) moveLineStart() {
	if len(m.data) == 0 {
		return
	}
	line := m.cursor / bytesPerLine
	m.cursor = line * bytesPerLine
}

func (m *editorModel) moveLineEnd() {
	if len(m.data) == 0 {
		return
	}
	line := m.cursor / bytesPerLine
	end := (line+1)*bytesPerLine - 1
	if end >= len(m.data) {
		end = len(m.data) - 1
	}
	m.cursor = end
}

func (m *editorModel) gotoTop() {
	if len(m.data) == 0 {
		m.cursor = 0
		return
	}
	m.cursor = 0
}

func (m *editorModel) gotoBottom() {
	if len(m.data) == 0 {
		m.cursor = 0
		return
	}
	m.cursor = len(m.data) - 1
}

func (m *editorModel) gotoOffset(offset int) {
	if len(m.data) == 0 {
		m.message = "Empty file"
		m.clearMessage = true
		return
	}
	if offset < 0 || offset >= len(m.data) {
		m.message = fmt.Sprintf("Offset out of range: %d", offset)
		m.clearMessage = true
		return
	}
	m.cursor = offset
	m.ensureCursorVisible()
}

func (m *editorModel) setByte(value byte) {
	if len(m.data) == 0 {
		m.message = "Empty file"
		m.clearMessage = true
		return
	}
	if m.cursor >= len(m.data) || m.cursor >= len(m.original) {
		return
	}
	wasModified := m.modified[m.cursor]
	orig := m.original[m.cursor]
	m.data[m.cursor] = value
	if value == orig {
		if wasModified {
			m.modified[m.cursor] = false
			if m.modifiedCount > 0 {
				m.modifiedCount--
			}
		}
	} else if !wasModified {
		m.modified[m.cursor] = true
		m.modifiedCount++
	}
	m.dirty = m.modifiedCount > 0
}

func printableByte(b byte) byte {
	if b >= 32 && b <= 126 {
		return b
	}
	return '.'
}

func (m *editorModel) isModified(idx int) bool {
	if idx < 0 || idx >= len(m.modified) {
		return false
	}
	return m.modified[idx]
}

func hexChar(n byte) byte {
	if n < 10 {
		return '0' + n
	}
	return 'a' + (n - 10)
}

func hexNibble(r rune) (byte, bool) {
	switch {
	case r >= '0' && r <= '9':
		return byte(r - '0'), true
	case r >= 'a' && r <= 'f':
		return byte(r-'a') + 10, true
	case r >= 'A' && r <= 'F':
		return byte(r-'A') + 10, true
	}
	return 0, false
}

func isNumericOffset(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	if strings.HasPrefix(strings.ToLower(value), "0x") {
		_, err := strconv.ParseInt(value, 0, 64)
		return err == nil
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func popRune(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return value
	}
	return string(runes[:len(runes)-1])
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
