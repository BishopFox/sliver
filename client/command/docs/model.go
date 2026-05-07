package docs

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"charm.land/bubbles/v2/paginator"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	glamourstyles "charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"
	clienttheme "github.com/bishopfox/sliver/client/theme"
	embeddeddocs "github.com/bishopfox/sliver/docs"
	"github.com/charmbracelet/x/ansi"
	"golang.org/x/term"
)

const (
	docsMinWidth            = 72
	docsMinHeight           = 18
	docsPaneGap             = 1
	docsViewerHeaderLines   = 1
	docsNarrowWidth         = 96
	docsBrowserMinWidth     = 32
	docsBrowserMaxWidth     = 46
	docsWindowPollInterval  = 100 * time.Millisecond
	docsBrowserItemHeight   = 3
	docsBrowserHeaderGap    = 1
	docsBrowserFooterGap    = 1
	docsBrowserFooterLines  = 1
	docsBrowserFullHelpRows = 3
)

var (
	markdownTrimChars      = regexp.MustCompile(`^[#>\-\*\+\d\.\)\s` + "`" + `]+`)
	yamlFrontmatterBound   = regexp.MustCompile(`(?m)^---\r?\n(\s*\r?\n)?`)
	markdownLinkPattern    = regexp.MustCompile(`!?\[([^\]]+)\]\([^)]+\)`)
	markdownInlineReplacer = strings.NewReplacer(
		"**", "",
		"__", "",
		"~~", "",
		"`", "",
		"*", "",
		"_", "",
	)
)

type docsFocus int

const (
	docsFocusBrowser docsFocus = iota
	docsFocusViewer
)

type docsFilterState int

const (
	docsFilterStateUnfiltered docsFilterState = iota
	docsFilterStateFiltering
	docsFilterStateApplied
)

type docEntry struct {
	Name        string
	Content     string
	Description string
}

type docsHelpEntry struct {
	key    string
	action string
}

type docsStyles struct {
	app                 lipgloss.Style
	pane                lipgloss.Style
	paneFocused         lipgloss.Style
	header              lipgloss.Style
	headerMeta          lipgloss.Style
	empty               lipgloss.Style
	error               lipgloss.Style
	minSize             lipgloss.Style
	browserCount        lipgloss.Style
	browserCountActive  lipgloss.Style
	browserTitle        lipgloss.Style
	browserTitleMuted   lipgloss.Style
	browserTitleActive  lipgloss.Style
	browserMeta         lipgloss.Style
	browserMetaActive   lipgloss.Style
	browserHelpKey      lipgloss.Style
	browserHelpText     lipgloss.Style
	browserHelpDivider  lipgloss.Style
	browserHelpOverflow lipgloss.Style
}

type docsWindowPollMsg struct {
	width  int
	height int
}

type docsModel struct {
	width                 int
	height                int
	focus                 docsFocus
	filterState           docsFilterState
	browserShowFullHelp   bool
	browserWidth          int
	browserHeight         int
	browserCursor         int
	browserFilter         textinput.Model
	browserFilterSnapshot string
	browserPaginator      paginator.Model
	filteredEntries       []docEntry
	viewer                viewport.Model
	entries               []docEntry
	entriesByName         map[string]docEntry
	currentDocName        string
	renderCache           map[string]string
	styles                docsStyles
}

func loadDocEntries() ([]docEntry, error) {
	loaded, err := embeddeddocs.All()
	if err != nil {
		return nil, err
	}

	entries := make([]docEntry, 0, len(loaded.Docs))
	for _, doc := range loaded.Docs {
		entries = append(entries, docEntry{
			Name:        doc.Name,
			Content:     doc.Content,
			Description: summarizeMarkdown(doc.Content),
		})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	return entries, nil
}

func newDocsModel(entries []docEntry) *docsModel {
	filter := textinput.New()
	filter.Prompt = "Find: "
	filter.Placeholder = "Search docs"
	filterStyles := textinput.DefaultStyles(true)
	filterStyles.Focused.Prompt = filterStyles.Focused.Prompt.Foreground(clienttheme.Primary()).Bold(true)
	filterStyles.Blurred.Prompt = filterStyles.Blurred.Prompt.Foreground(clienttheme.DefaultMod(500))
	filterStyles.Focused.Text = filterStyles.Focused.Text.Foreground(clienttheme.DefaultMod(900))
	filterStyles.Blurred.Text = filterStyles.Blurred.Text.Foreground(clienttheme.DefaultMod(700))
	filterStyles.Focused.Placeholder = filterStyles.Focused.Placeholder.Foreground(clienttheme.DefaultMod(400))
	filterStyles.Blurred.Placeholder = filterStyles.Blurred.Placeholder.Foreground(clienttheme.DefaultMod(400))
	filter.SetStyles(filterStyles)
	filter.Blur()

	browserPager := paginator.New()
	browserPager.Type = paginator.Dots
	browserPager.ActiveDot = lipgloss.NewStyle().Foreground(clienttheme.Primary()).Render("•")
	browserPager.InactiveDot = lipgloss.NewStyle().Foreground(clienttheme.DefaultMod(300)).Render("•")

	viewer := viewport.New()
	viewer.MouseWheelEnabled = true
	viewer.SoftWrap = false

	model := &docsModel{
		focus:            docsFocusBrowser,
		filterState:      docsFilterStateUnfiltered,
		browserFilter:    filter,
		browserPaginator: browserPager,
		viewer:           viewer,
		entries:          entries,
		entriesByName:    make(map[string]docEntry, len(entries)),
		renderCache:      make(map[string]string),
		styles:           newDocsStyles(),
	}

	for _, entry := range entries {
		model.entriesByName[entry.Name] = entry
	}

	if idx := preferredDocIndex(entries); idx >= 0 {
		model.selectVisibleIndex(idx)
		model.setCurrentDoc(entries[idx].Name, false)
	}

	return model
}

func newDocsStyles() docsStyles {
	return docsStyles{
		app: lipgloss.NewStyle(),
		pane: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(clienttheme.DefaultMod(300)).
			Padding(0, 1),
		paneFocused: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(clienttheme.Primary()).
			Padding(0, 1),
		header: lipgloss.NewStyle().
			Bold(true).
			Foreground(clienttheme.DefaultMod(900)),
		headerMeta: lipgloss.NewStyle().
			Foreground(clienttheme.DefaultMod(500)),
		empty: lipgloss.NewStyle().
			Foreground(clienttheme.DefaultMod(500)),
		error: lipgloss.NewStyle().
			Foreground(clienttheme.Danger()),
		minSize: lipgloss.NewStyle().
			Foreground(clienttheme.Warning()).
			Bold(true),
		browserCount: lipgloss.NewStyle().
			Foreground(clienttheme.DefaultMod(500)),
		browserCountActive: lipgloss.NewStyle().
			Foreground(clienttheme.DefaultMod(700)),
		browserTitle: lipgloss.NewStyle().
			Foreground(clienttheme.DefaultMod(900)),
		browserTitleMuted: lipgloss.NewStyle().
			Foreground(clienttheme.DefaultMod(400)),
		browserTitleActive: lipgloss.NewStyle().
			Foreground(clienttheme.Primary()),
		browserMeta: lipgloss.NewStyle().
			Foreground(clienttheme.DefaultMod(500)),
		browserMetaActive: lipgloss.NewStyle().
			Foreground(clienttheme.DefaultMod(700)),
		browserHelpKey: lipgloss.NewStyle().
			Foreground(clienttheme.DefaultMod(700)),
		browserHelpText: lipgloss.NewStyle().
			Foreground(clienttheme.DefaultMod(500)),
		browserHelpDivider: lipgloss.NewStyle().
			Foreground(clienttheme.DefaultMod(300)),
		browserHelpOverflow: lipgloss.NewStyle().
			Foreground(clienttheme.DefaultMod(400)),
	}
}

func preferredDocIndex(entries []docEntry) int {
	for i, entry := range entries {
		if strings.EqualFold(entry.Name, "Getting Started") {
			return i
		}
	}
	if len(entries) == 0 {
		return -1
	}
	return 0
}

func summarizeMarkdown(markdown string) string {
	text := string(removeFrontmatter([]byte(markdown)))
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "```") {
			continue
		}
		line = markdownTrimChars.ReplaceAllString(line, "")
		line = sanitizeBrowserSummary(line)
		if line != "" {
			return line
		}
	}
	return "Embedded Sliver documentation"
}

func sanitizeBrowserSummary(value string) string {
	value = markdownLinkPattern.ReplaceAllString(value, "$1")
	value = markdownInlineReplacer.Replace(value)

	var builder strings.Builder
	lastWasSpace := true
	for _, r := range value {
		switch {
		case unicode.IsSpace(r):
			if !lastWasSpace {
				builder.WriteByte(' ')
				lastWasSpace = true
			}
		case r > unicode.MaxASCII:
			continue
		case !unicode.IsPrint(r):
			continue
		default:
			builder.WriteRune(r)
			lastWasSpace = false
		}
	}

	return strings.TrimSpace(builder.String())
}

func renderMarkdownWithGlow(width int, markdown string) (string, error) {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(glamourstyles.DarkStyle),
		glamour.WithWordWrap(maxInt(12, width)),
		glamour.WithPreservedNewLines(),
	)
	if err != nil {
		return "", err
	}

	rendered, err := renderer.Render(string(removeFrontmatter([]byte(markdown))))
	if err != nil {
		return "", err
	}

	lines := strings.Split(rendered, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n"), nil
}

func removeFrontmatter(content []byte) []byte {
	if frontmatterBoundaries := detectFrontmatter(content); frontmatterBoundaries[0] == 0 {
		return content[frontmatterBoundaries[1]:]
	}
	return content
}

func detectFrontmatter(content []byte) []int {
	if matches := yamlFrontmatterBound.FindAllIndex(content, 2); len(matches) > 1 {
		return []int{matches[0][0], matches[1][1]}
	}
	return []int{-1, -1}
}

func (m *docsModel) Init() tea.Cmd {
	return docsWindowPollCmd()
}

func (m *docsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.applyWindowSize(msg.Width, msg.Height)
		return m, nil

	case docsWindowPollMsg:
		return m, tea.Batch(m.windowSizeCmd(msg.width, msg.height), docsWindowPollCmd())

	case tea.KeyPressMsg:
		return m, m.handleKey(msg)
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd

	if m.filterState == docsFilterStateFiltering {
		before := m.browserFilter.Value()
		m.browserFilter, cmd = m.browserFilter.Update(msg)
		cmds = append(cmds, cmd)
		if before != m.browserFilter.Value() {
			m.updateFilteredEntries(true)
		}
	}

	m.viewer, cmd = m.viewer.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *docsModel) handleKey(msg tea.KeyPressMsg) tea.Cmd {
	if msg.String() == "ctrl+c" {
		return tea.Quit
	}

	if m.filterState == docsFilterStateFiltering {
		switch msg.Code {
		case tea.KeyEnter:
			m.finishFiltering(true)
			return nil
		case tea.KeyEsc:
			m.cancelFiltering()
			return nil
		case tea.KeyTab:
			m.finishFiltering(false)
			m.focus = docsFocusViewer
			return nil
		}

		switch msg.String() {
		case "k", "ctrl+k", "up", "j", "ctrl+j", "down", "home", "g", "end", "G", "h", "left", "l", "right":
			m.handleBrowserKey(msg)
			return nil
		}

		before := m.browserFilter.Value()
		updated, cmd := m.browserFilter.Update(msg)
		m.browserFilter = updated
		if before != m.browserFilter.Value() {
			m.updateFilteredEntries(true)
		}
		return cmd
	}

	switch msg.Code {
	case tea.KeyTab:
		if m.focus == docsFocusBrowser {
			m.focus = docsFocusViewer
		} else {
			m.focus = docsFocusBrowser
		}
		return nil
	case tea.KeyEsc:
		if m.focus == docsFocusViewer {
			m.focus = docsFocusBrowser
			return nil
		}
		if m.filterState == docsFilterStateApplied {
			m.clearFilter()
			return nil
		}
	}

	switch msg.String() {
	case "q":
		return tea.Quit
	case "?":
		m.browserShowFullHelp = !m.browserShowFullHelp
		m.updateBrowserPagination()
		return nil
	case "/":
		return m.startFiltering()
	case "enter":
		if m.focus == docsFocusBrowser && m.selectedEntry() != nil {
			m.focus = docsFocusViewer
		}
		return nil
	}

	if m.focus == docsFocusBrowser {
		m.handleBrowserKey(msg)
		return nil
	}

	updated, cmd := m.viewer.Update(msg)
	m.viewer = updated
	return cmd
}

func (m *docsModel) handleBrowserKey(msg tea.KeyPressMsg) {
	switch msg.String() {
	case "k", "ctrl+k", "up":
		m.moveBrowserCursorUp()
	case "j", "ctrl+j", "down":
		m.moveBrowserCursorDown()
	case "home", "g":
		m.moveBrowserToStart()
	case "end", "G":
		m.moveBrowserToEnd()
	case "h", "left":
		m.browserPrevPage()
	case "l", "right":
		m.browserNextPage()
	default:
		return
	}
	m.syncSelectionFromBrowser()
}

func (m *docsModel) startFiltering() tea.Cmd {
	m.focus = docsFocusBrowser
	m.filterState = docsFilterStateFiltering
	m.browserFilterSnapshot = m.browserFilter.Value()
	m.browserFilter.CursorEnd()
	return m.browserFilter.Focus()
}

func (m *docsModel) cancelFiltering() {
	m.browserFilter.SetValue(m.browserFilterSnapshot)
	m.browserFilter.Blur()
	if strings.TrimSpace(m.browserFilter.Value()) == "" {
		m.clearFilter()
		return
	}
	m.filterState = docsFilterStateApplied
	m.updateFilteredEntries(true)
}

func (m *docsModel) finishFiltering(openViewer bool) {
	m.browserFilter.Blur()
	if strings.TrimSpace(m.browserFilter.Value()) == "" {
		m.clearFilter()
		return
	}
	m.filterState = docsFilterStateApplied
	m.updateFilteredEntries(true)
	if openViewer && m.selectedEntry() != nil {
		m.focus = docsFocusViewer
	}
}

func (m *docsModel) clearFilter() {
	m.browserFilter.Reset()
	m.browserFilter.Blur()
	m.browserFilterSnapshot = ""
	m.filterState = docsFilterStateUnfiltered
	m.filteredEntries = nil
	m.updateBrowserPagination()
	if !m.selectDocInVisible(m.currentDocName) {
		m.selectVisibleIndex(0)
	}
	m.syncSelectionFromBrowser()
}

func (m *docsModel) updateFilteredEntries(preferCurrent bool) {
	query := strings.TrimSpace(strings.ToLower(m.browserFilter.Value()))
	if query == "" {
		m.filteredEntries = nil
		m.updateBrowserPagination()
		if !m.selectDocInVisible(m.currentDocName) {
			m.selectVisibleIndex(0)
		}
		m.syncSelectionFromBrowser()
		return
	}

	filtered := make([]docEntry, 0, len(m.entries))
	for _, entry := range m.entries {
		if matchesDocQuery(entry, query) {
			filtered = append(filtered, entry)
		}
	}
	m.filteredEntries = filtered
	m.updateBrowserPagination()

	if preferCurrent && m.selectDocInVisible(m.currentDocName) {
		m.syncSelectionFromBrowser()
		return
	}
	m.selectVisibleIndex(0)
	m.syncSelectionFromBrowser()
}

func matchesDocQuery(entry docEntry, query string) bool {
	if query == "" {
		return true
	}
	haystack := strings.ToLower(entry.Name + ".md\n" + entry.Description)
	return strings.Contains(haystack, query)
}

func (m *docsModel) moveBrowserCursorUp() {
	if len(m.visibleEntries()) == 0 {
		return
	}

	m.browserCursor--
	if m.browserCursor >= 0 {
		return
	}
	if !m.browserPaginator.OnFirstPage() {
		m.browserPaginator.PrevPage()
		m.browserCursor = maxInt(0, m.browserPaginator.ItemsOnPage(len(m.visibleEntries()))-1)
		return
	}
	m.browserCursor = 0
}

func (m *docsModel) moveBrowserCursorDown() {
	if len(m.visibleEntries()) == 0 {
		return
	}

	m.browserCursor++
	itemsOnPage := m.browserPaginator.ItemsOnPage(len(m.visibleEntries()))
	if m.browserCursor < itemsOnPage {
		return
	}
	if !m.browserPaginator.OnLastPage() {
		m.browserPaginator.NextPage()
		m.browserCursor = 0
		return
	}
	m.browserCursor = maxInt(0, itemsOnPage-1)
}

func (m *docsModel) moveBrowserToStart() {
	if len(m.visibleEntries()) == 0 {
		return
	}
	m.browserPaginator.Page = 0
	m.browserCursor = 0
}

func (m *docsModel) moveBrowserToEnd() {
	visible := m.visibleEntries()
	if len(visible) == 0 {
		return
	}
	m.selectVisibleIndex(len(visible) - 1)
}

func (m *docsModel) browserPrevPage() {
	if m.browserPaginator.OnFirstPage() {
		return
	}
	m.browserPaginator.PrevPage()
	m.clampBrowserSelection()
}

func (m *docsModel) browserNextPage() {
	if m.browserPaginator.OnLastPage() {
		return
	}
	m.browserPaginator.NextPage()
	m.clampBrowserSelection()
}

func (m *docsModel) visibleEntries() []docEntry {
	if strings.TrimSpace(m.browserFilter.Value()) == "" {
		return m.entries
	}
	return m.filteredEntries
}

func (m *docsModel) currentPageEntries() []docEntry {
	visible := m.visibleEntries()
	if len(visible) == 0 {
		return nil
	}
	start, end := m.browserPaginator.GetSliceBounds(len(visible))
	if start < 0 || start >= len(visible) || start >= end {
		return nil
	}
	return visible[start:end]
}

func (m *docsModel) selectedEntry() *docEntry {
	visible := m.visibleEntries()
	if len(visible) == 0 {
		return nil
	}
	index := m.browserPaginator.Page*maxInt(1, m.browserPaginator.PerPage) + m.browserCursor
	if index < 0 || index >= len(visible) {
		return nil
	}
	return &visible[index]
}

func (m *docsModel) selectVisibleIndex(index int) {
	visible := m.visibleEntries()
	if len(visible) == 0 {
		m.browserPaginator.Page = 0
		m.browserCursor = 0
		return
	}

	index = clampInt(index, 0, len(visible)-1)
	perPage := maxInt(1, m.browserPaginator.PerPage)
	m.browserPaginator.Page = index / perPage
	m.browserCursor = index % perPage
	m.clampBrowserSelection()
}

func (m *docsModel) selectDocInVisible(name string) bool {
	if name == "" {
		return false
	}
	for i, entry := range m.visibleEntries() {
		if strings.EqualFold(entry.Name, name) {
			m.selectVisibleIndex(i)
			return true
		}
	}
	return false
}

func (m *docsModel) clampBrowserSelection() {
	visible := m.visibleEntries()
	total := len(visible)
	if total == 0 {
		m.browserPaginator.Page = 0
		m.browserCursor = 0
		return
	}

	if m.browserPaginator.TotalPages <= 0 {
		m.browserPaginator.SetTotalPages(total)
	}
	if m.browserPaginator.Page > maxInt(0, m.browserPaginator.TotalPages-1) {
		m.browserPaginator.Page = maxInt(0, m.browserPaginator.TotalPages-1)
	}
	if m.browserPaginator.Page < 0 {
		m.browserPaginator.Page = 0
	}

	itemsOnPage := m.browserPaginator.ItemsOnPage(total)
	if itemsOnPage <= 0 {
		m.browserCursor = 0
		return
	}
	if m.browserCursor > itemsOnPage-1 {
		m.browserCursor = itemsOnPage - 1
	}
	if m.browserCursor < 0 {
		m.browserCursor = 0
	}
}

func (m *docsModel) syncSelectionFromBrowser() {
	selected := m.selectedEntry()
	if selected == nil || selected.Name == m.currentDocName {
		return
	}
	m.setCurrentDoc(selected.Name, false)
}

func (m *docsModel) View() tea.View {
	if m.width < docsMinWidth || m.height < docsMinHeight {
		view := tea.NewView(m.styles.minSize.Render(
			fmt.Sprintf("Docs viewer needs at least %dx%d. Current size: %dx%d", docsMinWidth, docsMinHeight, m.width, m.height),
		))
		view.AltScreen = true
		return view
	}

	browserPane, viewerPane := m.renderPanes(m.height)
	layout := lipgloss.JoinHorizontal(lipgloss.Top, browserPane, strings.Repeat(" ", docsPaneGap), viewerPane)
	if m.isNarrow() {
		layout = lipgloss.JoinVertical(lipgloss.Left, browserPane, viewerPane)
	}

	view := tea.NewView(m.styles.app.Render(layout))
	view.AltScreen = true
	return view
}

func (m *docsModel) renderPanes(bodyHeight int) (string, string) {
	if m.isNarrow() {
		browserHeight := clampInt(bodyHeight/3, 8, 13)
		viewerHeight := maxInt(5, bodyHeight-browserHeight-docsPaneGap)
		browserPane := m.renderBrowserPane(m.width, browserHeight)
		viewerPane := m.renderViewerPane(m.width, viewerHeight)
		return browserPane, viewerPane
	}

	browserWidth := clampInt(m.width/3, docsBrowserMinWidth, docsBrowserMaxWidth)
	viewerWidth := maxInt(docsMinWidth-browserWidth, m.width-browserWidth-docsPaneGap)
	return m.renderBrowserPane(browserWidth, bodyHeight), m.renderViewerPane(viewerWidth, bodyHeight)
}

func (m *docsModel) renderBrowserPane(width, height int) string {
	style := m.styles.pane
	if m.focus == docsFocusBrowser {
		style = m.styles.paneFocused
	}

	innerWidth := maxInt(1, width-style.GetHorizontalFrameSize())
	innerHeight := maxInt(1, height-style.GetVerticalFrameSize())
	body := m.renderBrowserBody(innerWidth, innerHeight)
	return style.Width(width).Height(height).Render(body)
}

func (m *docsModel) renderBrowserBody(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	headerLines := m.browserHeaderLines(width)
	helpLines := m.browserHelpLines(width)
	footerLines := []string{
		"",
		m.browserPaginationLine(width),
	}
	footerLines = append(footerLines, helpLines...)

	itemHeight := maxInt(0, height-len(headerLines)-len(footerLines))
	itemLines := m.renderBrowserItems(width, itemHeight)

	lines := append([]string{}, headerLines...)
	lines = append(lines, itemLines...)
	lines = append(lines, footerLines...)

	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

func (m *docsModel) browserHeaderLines(width int) []string {
	countStyle := m.styles.browserCount
	if strings.TrimSpace(m.browserFilter.Value()) != "" {
		countStyle = m.styles.browserCountActive
	}

	lines := []string{
		truncateText(countStyle.Render(m.browserCountText()), width),
	}
	if m.filterState == docsFilterStateFiltering || strings.TrimSpace(m.browserFilter.Value()) != "" {
		lines = append(lines, truncateText(m.browserFilter.View(), width))
	}
	lines = append(lines, "")
	return lines
}

func (m *docsModel) browserCountText() string {
	total := len(m.entries)
	visible := len(m.visibleEntries())
	if strings.TrimSpace(m.browserFilter.Value()) == "" {
		if total == 1 {
			return "1 document"
		}
		return fmt.Sprintf("%d documents", total)
	}
	if visible == 1 {
		return fmt.Sprintf("1 of %d documents", total)
	}
	return fmt.Sprintf("%d of %d documents", visible, total)
}

func (m *docsModel) browserPaginationLine(width int) string {
	if m.browserPaginator.TotalPages <= 1 {
		return ""
	}
	return truncateText(m.browserPaginator.View(), width)
}

func (m *docsModel) browserHelpLines(width int) []string {
	groups := m.browserHelpGroups()
	if !m.browserShowFullHelp {
		return []string{m.renderHelpLine(width, m.browserMiniHelpEntries())}
	}

	lines := make([]string, 0, len(groups))
	for _, group := range groups {
		if len(group) == 0 {
			continue
		}
		lines = append(lines, m.renderHelpLine(width, group))
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func (m *docsModel) browserMiniHelpEntries() []docsHelpEntry {
	if m.filterState == docsFilterStateFiltering {
		action := "apply"
		if len(m.visibleEntries()) > 0 {
			action = "open"
		}
		return []docsHelpEntry{
			{key: "enter", action: action},
			{key: "esc", action: "cancel"},
		}
	}

	entries := make([]docsHelpEntry, 0, 5)
	if m.filterState != docsFilterStateApplied && m.browserPaginator.TotalPages > 1 {
		entries = append(entries, docsHelpEntry{key: "h/l", action: "page"})
	}
	if m.filterState == docsFilterStateApplied {
		entries = append(entries,
			docsHelpEntry{key: "/", action: "edit"},
			docsHelpEntry{key: "esc", action: "clear"},
		)
	} else {
		entries = append(entries, docsHelpEntry{key: "/", action: "find"})
	}
	entries = append(entries,
		docsHelpEntry{key: "q", action: "quit"},
		docsHelpEntry{key: "?", action: ternaryHelpLabel(m.browserShowFullHelp)},
	)
	return entries
}

func (m *docsModel) browserHelpGroups() [][]docsHelpEntry {
	switchTarget := "viewer"
	if m.focus == docsFocusViewer {
		switchTarget = "browser"
	}

	if m.filterState == docsFilterStateFiltering {
		var nav []docsHelpEntry
		if len(m.visibleEntries()) > 1 {
			nav = append(nav, docsHelpEntry{key: "j/k", action: "move"})
		}
		if m.browserPaginator.TotalPages > 1 {
			nav = append(nav, docsHelpEntry{key: "h/l", action: "page"})
		}

		action := "apply"
		if len(m.visibleEntries()) > 0 {
			action = "open"
		}

		return [][]docsHelpEntry{
			nav,
			{
				{key: "enter", action: action},
				{key: "esc", action: "cancel"},
			},
			{
				{key: "tab", action: switchTarget},
				{key: "ctrl+c", action: "quit"},
			},
		}
	}

	var nav []docsHelpEntry
	if len(m.visibleEntries()) > 0 {
		nav = append(nav, docsHelpEntry{key: "j/k", action: "move"})
	}
	if m.browserPaginator.TotalPages > 1 {
		nav = append(nav, docsHelpEntry{key: "h/l", action: "page"})
	}

	filter := []docsHelpEntry{{key: "/", action: "find"}}
	if m.filterState == docsFilterStateApplied {
		filter = []docsHelpEntry{
			{key: "/", action: "edit"},
			{key: "esc", action: "clear"},
		}
	}
	if len(m.visibleEntries()) > 0 {
		filter = append(filter, docsHelpEntry{key: "enter", action: "open"})
	}

	return [][]docsHelpEntry{
		nav,
		filter,
		{
			{key: "tab", action: switchTarget},
			{key: "q", action: "quit"},
			{key: "?", action: ternaryHelpLabel(m.browserShowFullHelp)},
		},
	}
}

func ternaryHelpLabel(showFull bool) string {
	if showFull {
		return "less"
	}
	return "more"
}

func (m *docsModel) renderHelpLine(width int, entries []docsHelpEntry) string {
	if width <= 0 || len(entries) == 0 {
		return ""
	}

	var builder strings.Builder
	overflow := m.styles.browserHelpOverflow.Render("…")

	for i, entry := range entries {
		segment := m.styles.browserHelpKey.Render(entry.key) + " " + m.styles.browserHelpText.Render(entry.action)
		if i < len(entries)-1 {
			segment += m.styles.browserHelpDivider.Render(" • ")
		}
		if lipgloss.Width(builder.String())+lipgloss.Width(segment) > width {
			if builder.Len() == 0 {
				return truncateText(segment, width)
			}
			builder.WriteString(overflow)
			break
		}
		builder.WriteString(segment)
	}

	return builder.String()
}

func (m *docsModel) renderBrowserItems(width, height int) []string {
	if height <= 0 {
		return nil
	}

	pageEntries := m.currentPageEntries()
	lines := make([]string, 0, height)
	if len(pageEntries) == 0 {
		lines = append(lines, truncateText(m.styles.empty.Render("No matching documents."), width))
		for len(lines) < height {
			lines = append(lines, "")
		}
		return lines
	}

	for i, entry := range pageEntries {
		lines = append(lines, m.browserItemLines(entry, width, i == m.browserCursor)...)
	}

	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return lines
}

func (m *docsModel) browserItemLines(entry docEntry, width int, selected bool) []string {
	contentWidth := maxInt(1, width-2)
	title := truncatePlain(entry.Name+".md", contentWidth)
	meta := truncatePlain(entry.Description, contentWidth)
	gutter := " "

	titleStyle := m.styles.browserTitle
	metaStyle := m.styles.browserMeta
	if selected {
		gutter = lipgloss.NewStyle().Foreground(clienttheme.Primary()).Render("│")
		titleStyle = m.styles.browserTitleActive
		metaStyle = m.styles.browserMetaActive
	} else if m.filterState == docsFilterStateFiltering && strings.TrimSpace(m.browserFilter.Value()) == "" {
		titleStyle = m.styles.browserTitleMuted
	}

	return []string{
		truncateText(gutter+" "+titleStyle.Render(title), width),
		truncateText(gutter+" "+metaStyle.Render(meta), width),
		"",
	}
}

func (m *docsModel) renderViewerPane(width, height int) string {
	style := m.styles.pane
	if m.focus == docsFocusViewer {
		style = m.styles.paneFocused
	}

	innerWidth := maxInt(1, width-style.GetHorizontalFrameSize())
	header := truncateText(
		m.styles.header.Render(m.viewerTitle())+" "+m.styles.headerMeta.Render(m.viewerMeta()),
		innerWidth,
	)

	body := m.viewer.View()
	if strings.TrimSpace(body) == "" {
		body = m.styles.empty.Render("Select a document from the browser to start reading.")
	}
	content := lipgloss.JoinVertical(lipgloss.Left, header, body)
	return style.Width(width).Height(height).Render(content)
}

func (m *docsModel) applyWindowSize(width, height int) {
	m.width = maxInt(0, width)
	m.height = maxInt(0, height)

	if m.width < docsMinWidth || m.height < docsMinHeight {
		return
	}

	if m.isNarrow() {
		browserHeight := clampInt(m.height/3, 8, 13)
		viewerHeight := maxInt(5, m.height-browserHeight-docsPaneGap)
		m.setBrowserSize(m.width, browserHeight)
		m.setViewerSize(m.width, viewerHeight)
	} else {
		browserWidth := clampInt(m.width/3, docsBrowserMinWidth, docsBrowserMaxWidth)
		viewerWidth := maxInt(docsMinWidth-browserWidth, m.width-browserWidth-docsPaneGap)
		m.setBrowserSize(browserWidth, m.height)
		m.setViewerSize(viewerWidth, m.height)
	}

	if m.currentDocName != "" {
		m.selectDocInVisible(m.currentDocName)
	}
	m.renderCurrentDoc(true)
}

func (m *docsModel) windowSizeCmd(width, height int) tea.Cmd {
	if width <= 0 || height <= 0 {
		return nil
	}
	if m.width == width && m.height == height {
		return nil
	}
	return func() tea.Msg {
		return tea.WindowSizeMsg{Width: width, Height: height}
	}
}

func (m *docsModel) setBrowserSize(width, height int) {
	style := m.styles.pane
	innerWidth := maxInt(1, width-style.GetHorizontalFrameSize())
	innerHeight := maxInt(6, height-style.GetVerticalFrameSize())
	m.browserWidth = innerWidth
	m.browserHeight = innerHeight
	m.browserFilter.SetWidth(maxInt(8, innerWidth-lipgloss.Width(m.browserFilter.Prompt)-1))
	m.updateBrowserPagination()
}

func (m *docsModel) browserHeaderHeight() int {
	height := 1 + docsBrowserHeaderGap
	if m.filterState == docsFilterStateFiltering || strings.TrimSpace(m.browserFilter.Value()) != "" {
		height++
	}
	return height
}

func (m *docsModel) browserHelpHeight() int {
	if !m.browserShowFullHelp {
		return 1
	}

	rows := 0
	for _, group := range m.browserHelpGroups() {
		if len(group) > 0 {
			rows++
		}
	}
	if rows == 0 {
		return 1
	}
	return minInt(rows, docsBrowserFullHelpRows)
}

func (m *docsModel) updateBrowserPagination() {
	availableHeight := m.browserHeight -
		m.browserHeaderHeight() -
		docsBrowserFooterGap -
		docsBrowserFooterLines -
		m.browserHelpHeight()

	m.browserPaginator.PerPage = maxInt(1, availableHeight/docsBrowserItemHeight)
	m.browserPaginator.SetTotalPages(len(m.visibleEntries()))
	m.clampBrowserSelection()
}

func (m *docsModel) setViewerSize(width, height int) {
	style := m.styles.pane
	innerWidth := maxInt(12, width-style.GetHorizontalFrameSize())
	innerHeight := maxInt(3, height-style.GetVerticalFrameSize()-docsViewerHeaderLines)
	m.viewer.SetWidth(innerWidth)
	m.viewer.SetHeight(innerHeight)
}

func (m *docsModel) setCurrentDoc(name string, preserveScroll bool) {
	if _, ok := m.entriesByName[name]; !ok {
		return
	}
	m.currentDocName = name
	m.renderCurrentDoc(preserveScroll)
}

func (m *docsModel) renderCurrentDoc(preserveScroll bool) {
	if m.currentDocName == "" || m.viewer.Width() <= 0 {
		return
	}

	entry, ok := m.entriesByName[m.currentDocName]
	if !ok {
		return
	}

	cacheKey := fmt.Sprintf("%s:%d", entry.Name, m.viewer.Width())
	rendered, ok := m.renderCache[cacheKey]
	if !ok {
		var err error
		rendered, err = renderMarkdownWithGlow(m.viewer.Width(), entry.Content)
		if err != nil {
			rendered = m.styles.error.Render(fmt.Sprintf("Failed to render %s\n\n%s", entry.Name, err))
		}
		m.renderCache[cacheKey] = rendered
	}

	scrollPercent := m.viewer.ScrollPercent()
	m.viewer.SetContent(rendered)
	if preserveScroll {
		m.restoreViewerScroll(scrollPercent)
		return
	}
	m.viewer.GotoTop()
}

func (m *docsModel) restoreViewerScroll(scrollPercent float64) {
	maxOffset := maxInt(0, m.viewer.TotalLineCount()-m.viewer.VisibleLineCount())
	m.viewer.SetYOffset(int(math.Round(scrollPercent * float64(maxOffset))))
}

func (m *docsModel) isNarrow() bool {
	return m.width < docsNarrowWidth
}

func (m *docsModel) viewerTitle() string {
	if m.currentDocName == "" {
		return "Document"
	}
	return m.currentDocName
}

func (m *docsModel) viewerMeta() string {
	total := m.viewer.TotalLineCount()
	if total == 0 {
		return "embedded markdown via Glow"
	}
	line := minInt(total, m.viewer.YOffset()+1)
	return fmt.Sprintf("line %d/%d", line, total)
}

func truncateText(value string, width int) string {
	if width <= 0 || lipgloss.Width(value) <= width {
		return value
	}
	return lipgloss.NewStyle().MaxWidth(width).Render(value)
}

func truncatePlain(value string, width int) string {
	if width <= 0 {
		return ""
	}
	value = strings.TrimSpace(value)
	if ansi.StringWidth(value) <= width {
		return value
	}
	return ansi.Truncate(value, width, "…")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampInt(value, low, high int) int {
	return minInt(high, maxInt(low, value))
}

func docsWindowPollCmd() tea.Cmd {
	return tea.Tick(docsWindowPollInterval, func(time.Time) tea.Msg {
		width, height, ok := currentTerminalSize()
		if !ok {
			return docsWindowPollMsg{}
		}
		return docsWindowPollMsg{width: width, height: height}
	})
}

func currentTerminalSize() (int, int, bool) {
	for _, fd := range []int{1, 0, 2} {
		width, height, err := term.GetSize(fd)
		if err == nil && width > 0 && height > 0 {
			return width, height, true
		}
	}
	return 0, 0, false
}
