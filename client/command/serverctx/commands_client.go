//go:build client

package serverctx

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	clienttheme "github.com/bishopfox/sliver/client/theme"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/connectivity"
)

// Commands returns client-only server connection context commands.
func Commands(con *console.SliverClient) []*cobra.Command {
	root := &cobra.Command{
		Use:   "server",
		Short: "Show and manage the current server connection",
		Run: func(cmd *cobra.Command, args []string) {
			serverInfo(cmd, con)
		},
	}

	switchCmd := &cobra.Command{
		Use:   "switch",
		Short: "Switch to another server/operator profile",
		Run: func(cmd *cobra.Command, args []string) {
			serverSwitch(cmd, con)
		},
	}
	root.AddCommand(switchCmd)

	return []*cobra.Command{root}
}

func serverInfo(_ *cobra.Command, con *console.SliverClient) {
	details, state, ok := con.CurrentConnection()
	if !ok || con.Rpc == nil {
		con.PrintErrorf("Not connected\n")
		return
	}

	var hostPort string
	operator := ""
	fingerprint := ""
	if details != nil && details.Config != nil {
		hostPort = fmt.Sprintf("%s:%d", details.Config.LHost, details.Config.LPort)
		operator = operatorFromConfig(details.Config)
		fingerprint = shortCertFingerprint(details.Config)
	}
	if operator == "" {
		operator = "<unknown>"
	}
	if hostPort == "" {
		hostPort = "<unknown>"
	}

	ver, err := con.Rpc.GetVersion(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("GetVersion failed: %s\n", err)
		return
	}

	const keyWidth = 8 // Align on ": " (e.g., "Operator", "Version")

	label := func(key string) string {
		// Bold "default/white" label keys, padded so colons align.
		return console.StyleBoldGray.Render(fmt.Sprintf("%-*s:", keyWidth, key))
	}
	valueSecondary := func(s string) string { return console.StylePurple.Render(s) }
	valueDim := func(s string) string { return console.StyleGray.Render(s) }

	stateStyle := console.StyleGray
	switch state {
	case connectivity.Ready:
		stateStyle = console.StyleGreen
	case connectivity.Connecting:
		stateStyle = console.StyleOrange
	case connectivity.TransientFailure, connectivity.Shutdown:
		stateStyle = console.StyleRed
	case connectivity.Idle:
		stateStyle = console.StyleOrange
	}

	// Server/operator values should be "default" terminal text (not themed colorized).
	con.Printf("%s %s\n", label("Server"), hostPort)
	if details != nil && details.ConfigKey != "" {
		con.Printf("%s %s\n", label("Profile"), valueSecondary(details.ConfigKey))
	}
	con.Printf("%s %s\n", label("Operator"), operator)
	if fingerprint != "" {
		con.Printf("%s %s\n", label("Cert"), valueDim(fingerprint))
	}
	con.Printf("%s %s\n", label("gRPC"), stateStyle.Render(state.String()))
	con.Printf("%s %s %s\n",
		label("Version"),
		console.StyleBold.Render(fmt.Sprintf("%d.%d.%d", ver.Major, ver.Minor, ver.Patch)),
		valueDim("("+ver.Commit+")"),
	)
}

func serverSwitch(_ *cobra.Command, con *console.SliverClient) {
	configs := assets.GetConfigs()
	if len(configs) == 0 {
		con.PrintErrorf("No config files found at %s\n", assets.GetConfigDir())
		return
	}

	keys := make([]string, 0, len(configs))
	for k := range configs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	mgr := getConsoleManager(con)
	mgr.syncActiveFromConsole(con)

	instances, activeID := mgr.snapshot()
	result, err := runSwitchTUI(instances, activeID, keys, mgr.configInstanceCountsLocked(instances))
	if err != nil {
		con.PrintErrorf("TUI error: %s\n", err)
		return
	}
	if result.Action == actionNone {
		return
	}
	switch result.Action {
	case actionSwitchExisting:
		inst := mgr.findInstance(result.ExistingID)
		if inst == nil || inst.Config == nil {
			con.PrintErrorf("Invalid selection\n")
			return
		}
		if inst.ID == activeID {
			con.PrintInfof("Already on %s\n", inst.DisplayName())
			return
		}

		rpc, conn, err := transport.MTLSConnect(inst.Config)
		if err != nil {
			con.PrintErrorf("Connection failed: %s\n", err)
			return
		}

		if err := con.SetConnection(rpc, conn, &console.ConnectionDetails{ConfigKey: inst.ConfigKey, Config: inst.Config}); err != nil {
			_ = conn.Close()
			con.PrintErrorf("Switch failed: %s\n", err)
			return
		}

		mgr.setActive(inst.ID)
		con.PrintSuccessf("Switched to %s:%d (%s)\n", inst.Config.LHost, inst.Config.LPort, inst.DisplayName())

	case actionNewConsole:
		cfg := configs[result.ConfigKey]
		if cfg == nil {
			con.PrintErrorf("Invalid selection\n")
			return
		}

		rpc, conn, err := transport.MTLSConnect(cfg)
		if err != nil {
			con.PrintErrorf("Connection failed: %s\n", err)
			return
		}

		if err := con.SetConnection(rpc, conn, &console.ConnectionDetails{ConfigKey: result.ConfigKey, Config: cfg}); err != nil {
			_ = conn.Close()
			con.PrintErrorf("Switch failed: %s\n", err)
			return
		}

		inst := mgr.addInstance(result.ConfigKey, cfg)
		con.PrintSuccessf("Connected to %s:%d (%s)\n", cfg.LHost, cfg.LPort, inst.DisplayName())
	}
}

var consoleManagers sync.Map // *console.SliverClient -> *consoleManager

type consoleManager struct {
	mu       sync.Mutex
	nextID   int
	activeID string
	insts    []*consoleInstance
}

type consoleInstance struct {
	ID        string
	ConfigKey string
	Config    *assets.ClientConfig
	CreatedAt time.Time
	LastUsed  time.Time
}

func (i *consoleInstance) DisplayName() string {
	if i == nil {
		return ""
	}
	return fmt.Sprintf("console-%s", i.ID)
}

func getConsoleManager(con *console.SliverClient) *consoleManager {
	if con == nil {
		return &consoleManager{}
	}
	if v, ok := consoleManagers.Load(con); ok {
		return v.(*consoleManager)
	}
	m := &consoleManager{}
	actual, _ := consoleManagers.LoadOrStore(con, m)
	return actual.(*consoleManager)
}

func (m *consoleManager) syncActiveFromConsole(con *console.SliverClient) {
	if m == nil || con == nil {
		return
	}
	details, _, ok := con.CurrentConnection()
	if !ok || details == nil || details.Config == nil || details.ConfigKey == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// If we already have an active instance, keep it. Otherwise create one from the current connection.
	if m.activeID != "" {
		return
	}

	m.nextID++
	id := fmt.Sprintf("%d", m.nextID)
	now := time.Now()
	m.activeID = id
	m.insts = append(m.insts, &consoleInstance{
		ID:        id,
		ConfigKey: details.ConfigKey,
		Config:    details.Config,
		CreatedAt: now,
		LastUsed:  now,
	})
}

func (m *consoleManager) snapshot() ([]*consoleInstance, string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]*consoleInstance, 0, len(m.insts))
	out = append(out, m.insts...)
	return out, m.activeID
}

func (m *consoleManager) findInstance(id string) *consoleInstance {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, inst := range m.insts {
		if inst != nil && inst.ID == id {
			return inst
		}
	}
	return nil
}

func (m *consoleManager) setActive(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeID = id
	for _, inst := range m.insts {
		if inst != nil && inst.ID == id {
			inst.LastUsed = time.Now()
			break
		}
	}
}

func (m *consoleManager) addInstance(configKey string, cfg *assets.ClientConfig) *consoleInstance {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nextID++
	id := fmt.Sprintf("%d", m.nextID)
	now := time.Now()
	inst := &consoleInstance{
		ID:        id,
		ConfigKey: configKey,
		Config:    cfg,
		CreatedAt: now,
		LastUsed:  now,
	}
	m.insts = append(m.insts, inst)
	m.activeID = id
	return inst
}

func (m *consoleManager) configInstanceCountsLocked(insts []*consoleInstance) map[string]int {
	// Caller provides insts snapshot; we just count.
	counts := map[string]int{}
	for _, inst := range insts {
		if inst == nil {
			continue
		}
		counts[inst.ConfigKey]++
	}
	return counts
}

type switchAction int

const (
	actionNone switchAction = iota
	actionSwitchExisting
	actionNewConsole
)

type switchResult struct {
	Action     switchAction
	ExistingID string
	ConfigKey  string
}

type switchTab int

const (
	tabExisting switchTab = iota
	tabNew
)

type switchTUIModel struct {
	tab      switchTab
	existing *huh.Form
	newConn  *huh.Form
	result   switchResult
	width    int
	height   int
	theme    *huh.Theme
	styles   switchStyles
	activeID string
	counts   map[string]int
	initOnce bool
	keyMap   *huh.KeyMap
}

type switchStyles struct {
	frame       lipgloss.Style
	header      lipgloss.Style
	tabActive   lipgloss.Style
	tabInactive lipgloss.Style
	footer      lipgloss.Style
	footerDim   lipgloss.Style
	divider     lipgloss.Style
}

func runSwitchTUI(instances []*consoleInstance, activeID string, configKeys []string, counts map[string]int) (switchResult, error) {
	m := newSwitchTUIModel(instances, activeID, configKeys, counts)
	prog := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := prog.Run()
	if err != nil {
		return switchResult{}, err
	}
	out, ok := finalModel.(switchTUIModel)
	if !ok {
		return switchResult{}, fmt.Errorf("unexpected TUI state")
	}
	return out.result, nil
}

func newSwitchTUIModel(instances []*consoleInstance, activeID string, configKeys []string, counts map[string]int) switchTUIModel {
	keyMap := huh.NewDefaultKeyMap()
	// Free up tab for switching between views; use enter to select/advance.
	keyMap.Select.Next = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select"))
	keyMap.Select.Submit = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit"))

	huhTheme := clienttheme.HuhTheme()

	styles := switchStyles{
		frame: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(clienttheme.DefaultMod(300)).
			Padding(0, 1),
		header: lipgloss.NewStyle().MarginBottom(1),
		tabActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(clienttheme.DefaultMod(900)).
			Background(clienttheme.Primary()).
			Padding(0, 1),
		tabInactive: lipgloss.NewStyle().
			Foreground(clienttheme.DefaultMod(500)).
			Background(clienttheme.DefaultMod(50)).
			Padding(0, 1),
		footer:    lipgloss.NewStyle().MarginTop(1),
		footerDim: lipgloss.NewStyle().Foreground(clienttheme.DefaultMod(400)),
		divider:   lipgloss.NewStyle().Foreground(clienttheme.DefaultMod(300)),
	}

	existingID := activeID
	existingOptions := make([]huh.Option[string], 0, len(instances))
	for _, inst := range instances {
		if inst == nil {
			continue
		}
		label := fmt.Sprintf("%s %s", inst.DisplayName(), inst.ConfigKey)
		if inst.ID == activeID && activeID != "" {
			label = fmt.Sprintf("* %s", label)
		}
		existingOptions = append(existingOptions, huh.NewOption(label, inst.ID))
	}

	if len(existingOptions) == 0 {
		// Always provide at least one option so Huh doesn't error.
		existingOptions = append(existingOptions, huh.NewOption("(no existing consoles)", ""))
	}

	existingSelect := huh.NewSelect[string]().
		Title("Existing Consoles").
		Description("Switch to an already-created console context. Use / to filter.").
		Options(existingOptions...).
		Key("existing").
		Value(&existingID)

	existingForm := huh.NewForm(huh.NewGroup(existingSelect)).
		WithTheme(huhTheme).
		WithKeyMap(keyMap).
		WithShowHelp(false)

	newKey := ""
	configOptions := make([]huh.Option[string], 0, len(configKeys))
	for _, k := range configKeys {
		label := k
		if n := counts[k]; n > 0 {
			label = fmt.Sprintf("%s (existing: %d)", k, n)
		}
		configOptions = append(configOptions, huh.NewOption(label, k))
		if newKey == "" {
			newKey = k
		}
	}
	newSelect := huh.NewSelect[string]().
		Title("New Connection / Console").
		Description("Create a new console context from a profile in ~/.sliver-client/configs. Use / to filter.").
		Options(configOptions...).
		Key("config").
		Value(&newKey)

	newForm := huh.NewForm(huh.NewGroup(newSelect)).
		WithTheme(huhTheme).
		WithKeyMap(keyMap).
		WithShowHelp(false)

	return switchTUIModel{
		tab:      tabExisting,
		existing: existingForm,
		newConn:  newForm,
		theme:    huhTheme,
		styles:   styles,
		activeID: activeID,
		counts:   counts,
		keyMap:   keyMap,
	}
}

func (m switchTUIModel) Init() tea.Cmd {
	// Initialize both forms so they're ready when switching tabs.
	return tea.Batch(m.existing.Init(), m.newConn.Init())
}

func (m switchTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Forward to both forms so they size correctly.
		modelA, cmdA := m.existing.Update(msg)
		modelB, cmdB := m.newConn.Update(msg)
		m.existing = modelA.(*huh.Form)
		m.newConn = modelB.(*huh.Form)
		return m, tea.Batch(cmdA, cmdB)

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			if m.tab == tabExisting {
				m.tab = tabNew
			} else {
				m.tab = tabExisting
			}
			return m, nil
		case "shift+tab":
			if m.tab == tabExisting {
				m.tab = tabNew
			} else {
				m.tab = tabExisting
			}
			return m, nil
		}
	}

	// Delegate to active tab's form.
	switch m.tab {
	case tabExisting:
		model, cmd := m.existing.Update(msg)
		m.existing = model.(*huh.Form)
		if m.existing.State == huh.StateCompleted {
			id := m.existing.GetString("existing")
			if id != "" {
				m.result = switchResult{Action: actionSwitchExisting, ExistingID: id}
			}
			return m, tea.Quit
		}
		if m.existing.State == huh.StateAborted {
			m.result = switchResult{Action: actionNone}
			return m, tea.Quit
		}
		return m, cmd

	case tabNew:
		model, cmd := m.newConn.Update(msg)
		m.newConn = model.(*huh.Form)
		if m.newConn.State == huh.StateCompleted {
			key := m.newConn.GetString("config")
			if key != "" {
				m.result = switchResult{Action: actionNewConsole, ConfigKey: key}
			}
			return m, tea.Quit
		}
		if m.newConn.State == huh.StateAborted {
			m.result = switchResult{Action: actionNone}
			return m, tea.Quit
		}
		return m, cmd
	}

	return m, nil
}

func (m switchTUIModel) View() string {
	header := m.renderHeader()

	body := ""
	switch m.tab {
	case tabExisting:
		body = m.existing.View()
	case tabNew:
		body = m.newConn.View()
	}

	footer := m.renderFooter()

	content := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
	return m.styles.frame.Render(content)
}

func (m switchTUIModel) renderHeader() string {
	left := m.styles.tabInactive.Render("Existing")
	right := m.styles.tabInactive.Render("New")
	if m.tab == tabExisting {
		left = m.styles.tabActive.Render("Existing")
	} else {
		right = m.styles.tabActive.Render("New")
	}
	line := lipgloss.JoinHorizontal(lipgloss.Left, left, m.styles.divider.Render(" "), right)
	return m.styles.header.Render(line)
}

func (m switchTUIModel) renderFooter() string {
	keys := "tab: switch view  enter: select  /: filter  ctrl+c: cancel"
	return m.styles.footerDim.Render(keys)
}

func operatorFromConfig(cfg *assets.ClientConfig) string {
	if cfg == nil {
		return ""
	}
	// Prefer the certificate CN, since config.Operator is mostly informational.
	block, _ := pem.Decode([]byte(cfg.Certificate))
	if block != nil {
		cert, err := x509.ParseCertificate(block.Bytes)
		if err == nil && cert.Subject.CommonName != "" {
			return cert.Subject.CommonName
		}
	}
	if strings.TrimSpace(cfg.Operator) != "" {
		return strings.TrimSpace(cfg.Operator)
	}
	return ""
}

func shortCertFingerprint(cfg *assets.ClientConfig) string {
	if cfg == nil {
		return ""
	}
	digest := sha256.Sum256([]byte(cfg.Certificate))
	return fmt.Sprintf("%x", digest[:8])
}
