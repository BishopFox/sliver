package crack

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/termio"
	clienttheme "github.com/bishopfox/sliver/client/theme"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	crackTopDefaultWidth    = 110
	crackTopDefaultHeight   = 30
	crackTopMinWidth        = 64
	crackTopMinHeight       = 18
	crackTopWideWidth       = 104
	crackTopHeaderHeight    = 5
	crackTopFooterHeight    = 1
	crackTopMinimumInterval = 250 * time.Millisecond
	crackTopEventDebounce   = 500 * time.Millisecond
	crackTopToastDuration   = 5 * time.Second
	crackTopWindowPoll      = 100 * time.Millisecond
)

var (
	openCrackTopTTY = tea.OpenTTY
	runCrackTopTUI  = func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
		return tea.NewProgram(model, options...).Run()
	}
)

type crackTopFetchFunc func(context.Context) (*crackTopSnapshot, error)
type crackTopJobActionFunc func(context.Context, string) (*clientpb.CrackJob, error)

type crackTopJobAction string

const (
	crackTopJobActionDelete crackTopJobAction = "delete"
	crackTopJobActionPause  crackTopJobAction = "pause"
	crackTopJobActionResume crackTopJobAction = "resume"
	crackTopJobActionCancel crackTopJobAction = "cancel"
)

type crackTopPendingJobAction struct {
	action crackTopJobAction
	jobID  string
}

type crackTopSnapshotMsg struct {
	snapshot *crackTopSnapshot
	err      error
}

type crackTopPollMsg struct {
	generation uint64
}

type crackTopEventMsg struct{}
type crackTopListenerClosedMsg struct{}

type crackTopEventRefreshMsg struct {
	generation uint64
}

type crackTopToastMsg struct {
	level   string
	message string
}

type crackTopToastExpiredMsg struct {
	generation uint64
}

type crackTopJobActionCompletedMsg struct {
	action    crackTopJobAction
	jobID     string
	status    clientpb.CrackJobStatus
	updatedAt int64
	hasStatus bool
	err       error
}

type crackTopJobStatusOverride struct {
	status                       clientpb.CrackJobStatus
	updatedAt                    int64
	equalVersionDisagreementSeen bool
}

type crackTopWindowPollMsg struct {
	width  int
	height int
	ok     bool
}

type crackTopFilter string

type crackTopFocus int

const (
	crackTopFocusJobs crackTopFocus = iota
	crackTopFocusWorkers
)

const (
	crackTopFilterAll       crackTopFilter = "all"
	crackTopFilterActive    crackTopFilter = "active"
	crackTopFilterPaused    crackTopFilter = "paused"
	crackTopFilterCompleted crackTopFilter = "completed"
	crackTopFilterFailed    crackTopFilter = "failed"
)

type crackTopStyles struct {
	title      lipgloss.Style
	heading    lipgloss.Style
	primary    lipgloss.Style
	secondary  lipgloss.Style
	normal     lipgloss.Style
	muted      lipgloss.Style
	success    lipgloss.Style
	warning    lipgloss.Style
	danger     lipgloss.Style
	selected   lipgloss.Style
	filterCard lipgloss.Style
	actionCard lipgloss.Style
}

type crackTopModel struct {
	ctx                 context.Context
	fetch               crackTopFetchFunc
	listener            <-chan *clientpb.Event
	pollInterval        time.Duration
	width               int
	height              int
	refreshing          bool
	refreshQueued       bool
	pollGeneration      uint64
	pollCancel          context.CancelFunc
	eventRefreshPending bool
	eventRefreshQueued  bool
	eventGeneration     uint64
	eventCancel         context.CancelFunc
	snapshot            *crackTopSnapshot
	dashboard           crackTopDashboard
	lastError           string
	eventStreamClosed   bool
	selectedJobID       string
	jobCursor           int
	selectedWorkerID    string
	workerCursor        int
	focus               crackTopFocus
	filter              crackTopFilter
	filterDraft         crackTopFilter
	filterForm          *huh.Form
	jobActions          map[crackTopJobAction]crackTopJobActionFunc
	pendingJobAction    crackTopPendingJobAction
	runningJobAction    crackTopPendingJobAction
	deletedJobIDs       map[string]struct{}
	jobStatusOverrides  map[string]crackTopJobStatusOverride
	toast               string
	toastLevel          string
	toastGeneration     uint64
	toastCancel         context.CancelFunc
	spinner             spinner.Model
	styles              crackTopStyles
}

// CrackTopCmd launches the full-screen real-time crack cluster monitor.
//
//nolint:revive // Keep the established exported command-handler name for compatibility.
func CrackTopCmd(cmd *cobra.Command, con *console.SliverClient, _ []string) {
	if con == nil || con.Rpc == nil {
		if con != nil {
			con.PrintErrorf("Connect to a server before using `crack top`.\n")
		}
		return
	}
	pollInterval, _ := cmd.Flags().GetDuration("poll-interval")
	if pollInterval < crackTopMinimumInterval {
		con.PrintErrorf("--poll-interval must be at least %s\n", crackTopMinimumInterval)
		return
	}

	timeoutSeconds, timeoutErr := cmd.Flags().GetInt64("timeout")
	if timeoutErr != nil {
		timeoutSeconds, _ = cmd.InheritedFlags().GetInt64("timeout")
	}
	perCallTimeout := time.Duration(0)
	if timeoutSeconds > 0 {
		perCallTimeout = time.Duration(timeoutSeconds) * time.Second
	}

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()
	listenerID, listener := con.CreateEventListener()
	defer con.RemoveEventListener(listenerID)
	restoreNotifications := con.SuppressEventNotifications()
	defer restoreNotifications()

	fetch := func(fetchCtx context.Context) (*crackTopSnapshot, error) {
		return loadCrackTopSnapshot(fetchCtx, con.Rpc, perCallTimeout, time.Time{})
	}
	model := newCrackTopModel(ctx, fetch, listener, pollInterval)
	model.jobActions = map[crackTopJobAction]crackTopJobActionFunc{
		crackTopJobActionDelete: func(actionCtx context.Context, jobID string) (*clientpb.CrackJob, error) {
			actionCtx, actionCancel := crackTopCallContext(actionCtx, perCallTimeout)
			defer actionCancel()
			_, err := con.Rpc.CrackJobDelete(actionCtx, &clientpb.CrackJob{ID: jobID})
			return nil, err
		},
		crackTopJobActionPause: func(actionCtx context.Context, jobID string) (*clientpb.CrackJob, error) {
			actionCtx, actionCancel := crackTopCallContext(actionCtx, perCallTimeout)
			defer actionCancel()
			return con.Rpc.CrackJobPause(actionCtx, &clientpb.CrackJob{ID: jobID})
		},
		crackTopJobActionResume: func(actionCtx context.Context, jobID string) (*clientpb.CrackJob, error) {
			actionCtx, actionCancel := crackTopCallContext(actionCtx, perCallTimeout)
			defer actionCancel()
			return con.Rpc.CrackJobResume(actionCtx, &clientpb.CrackJob{ID: jobID})
		},
		crackTopJobActionCancel: func(actionCtx context.Context, jobID string) (*clientpb.CrackJob, error) {
			actionCtx, actionCancel := crackTopCallContext(actionCtx, perCallTimeout)
			defer actionCancel()
			return con.Rpc.CrackJobCancel(actionCtx, &clientpb.CrackJob{ID: jobID})
		},
	}
	width, height := crackTopTerminalSize()
	options := []tea.ProgramOption{
		tea.WithContext(ctx),
		tea.WithWindowSize(width, height),
		tea.WithColorProfile(colorprofile.TrueColor),
	}
	ttyOptions, cleanup := configureCrackTopProgramTTY()
	if cleanup != nil {
		defer cleanup()
	}
	options = append(options, ttyOptions...)

	if _, err := runCrackTopTUI(model, options...); err != nil &&
		!errors.Is(err, context.Canceled) &&
		!errors.Is(err, tea.ErrInterrupted) &&
		!errors.Is(err, tea.ErrProgramKilled) {
		con.PrintErrorf("Crack top TUI error: %s\n", err)
	}
}

func crackTopCallContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout > 0 {
		return context.WithTimeout(ctx, timeout)
	}
	return context.WithCancel(ctx)
}

func newCrackTopModel(ctx context.Context, fetch crackTopFetchFunc, listener <-chan *clientpb.Event, pollInterval time.Duration) *crackTopModel {
	if ctx == nil {
		ctx = context.Background()
	}
	if pollInterval < crackTopMinimumInterval {
		pollInterval = time.Second
	}
	activity := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(clienttheme.Secondary())),
	)
	return &crackTopModel{
		ctx:          ctx,
		fetch:        fetch,
		listener:     listener,
		pollInterval: pollInterval,
		width:        crackTopDefaultWidth,
		height:       crackTopDefaultHeight,
		refreshing:   true,
		filter:       crackTopFilterAll,
		spinner:      activity,
		styles:       newCrackTopStyles(),
	}
}

func newCrackTopStyles() crackTopStyles {
	return crackTopStyles{
		title:     lipgloss.NewStyle().Bold(true).Foreground(clienttheme.Primary()),
		heading:   lipgloss.NewStyle().Bold(true).Foreground(clienttheme.DefaultMod(900)),
		primary:   lipgloss.NewStyle().Foreground(clienttheme.Primary()),
		secondary: lipgloss.NewStyle().Foreground(clienttheme.Secondary()),
		normal:    lipgloss.NewStyle().Foreground(clienttheme.DefaultMod(800)),
		muted:     lipgloss.NewStyle().Foreground(clienttheme.DefaultMod(500)),
		success:   lipgloss.NewStyle().Foreground(clienttheme.Success()).Bold(true),
		warning:   lipgloss.NewStyle().Foreground(clienttheme.Warning()).Bold(true),
		danger:    lipgloss.NewStyle().Foreground(clienttheme.Danger()).Bold(true),
		selected:  lipgloss.NewStyle().Foreground(clienttheme.Primary()).Bold(true),
		filterCard: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(clienttheme.Primary()).
			Padding(1, 2),
		actionCard: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2),
	}
}

func (m *crackTopModel) Init() tea.Cmd {
	commands := []tea.Cmd{m.spinner.Tick, m.fetchCmd()}
	if m.listener != nil {
		commands = append(commands, waitForCrackTopEventCmd(m.listener))
	}
	if runtime.GOOS == "windows" {
		commands = append(commands, crackTopWindowPollCmd(m.ctx))
	}
	return tea.Batch(commands...)
}

//nolint:gocyclo // Async refresh, modal, and navigation messages share one Bubble Tea update loop.
func (m *crackTopModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width = max(1, msg.Width)
		m.height = max(1, msg.Height)
		if (m.width < crackTopMinWidth || m.height < crackTopMinHeight) && m.pendingJobAction.jobID != "" {
			// The compact-size warning replaces the entire dashboard, including
			// confirmation modals. Never leave a hidden destructive prompt
			// actionable while the operator cannot see it.
			m.pendingJobAction = crackTopPendingJobAction{}
		}
		if m.filterForm != nil {
			m.resizeFilterForm()
			updated, command := m.filterForm.Update(msg)
			if form, ok := updated.(*huh.Form); ok {
				m.filterForm = form
			}
			return m, command
		}
		return m, nil

	case crackTopWindowPollMsg:
		commands := []tea.Cmd{crackTopWindowPollCmd(m.ctx)}
		if msg.ok && (msg.width != m.width || msg.height != m.height) {
			width, height := msg.width, msg.height
			commands = append(commands, func() tea.Msg {
				return tea.WindowSizeMsg{Width: width, Height: height}
			})
		}
		return m, tea.Batch(commands...)

	case crackTopSnapshotMsg:
		m.refreshing = false
		if msg.snapshot != nil {
			m.suppressDeletedJobs(msg.snapshot)
			m.applyJobStatusOverrides(msg.snapshot)
			m.snapshot = mergeCrackTopSnapshot(m.snapshot, msg.snapshot)
			m.dashboard = buildCrackTopDashboard(m.snapshot)
			m.normalizeJobSelection()
			m.normalizeWorkerSelection()
		}
		if msg.err != nil {
			m.lastError = crackTopWarningText(msg.err.Error())
		} else {
			m.lastError = ""
		}
		if m.refreshQueued {
			m.refreshQueued = false
			m.eventRefreshQueued = false
			return m, m.startRefresh()
		}
		if m.eventRefreshQueued {
			m.eventRefreshQueued = false
			return m, m.scheduleEventRefresh()
		}
		return m, m.schedulePoll()

	case crackTopPollMsg:
		if msg.generation != m.pollGeneration {
			return m, nil
		}
		if m.pollCancel != nil {
			m.pollCancel()
		}
		m.pollCancel = nil
		return m, m.requestRefresh()

	case crackTopEventMsg:
		return m, tea.Batch(waitForCrackTopEventCmd(m.listener), m.scheduleEventRefresh())

	case crackTopEventRefreshMsg:
		if !m.eventRefreshPending || msg.generation != m.eventGeneration {
			return m, nil
		}
		if m.eventCancel != nil {
			m.eventCancel()
		}
		m.eventCancel = nil
		m.eventRefreshPending = false
		if m.refreshing {
			m.eventRefreshQueued = true
			return m, nil
		}
		return m, m.startRefresh()

	case crackTopToastMsg:
		return m, tea.Batch(
			waitForCrackTopEventCmd(m.listener),
			m.showToast(msg.level, msg.message),
		)

	case crackTopJobActionCompletedMsg:
		if msg.action != m.runningJobAction.action || msg.jobID != m.runningJobAction.jobID {
			return m, nil
		}
		m.runningJobAction = crackTopPendingJobAction{}
		if msg.err != nil {
			return m, m.showToast("error", fmt.Sprintf("Failed to %s crack job %s: %s", msg.action, crackTopShortID(msg.jobID), msg.err))
		}
		if msg.action == crackTopJobActionDelete {
			m.removeJob(msg.jobID)
		} else if msg.hasStatus {
			m.overrideJobStatus(msg.jobID, msg.status, msg.updatedAt)
		} else {
			return m, m.showToast("error", fmt.Sprintf("Failed to %s crack job %s: server returned no job state", msg.action, crackTopShortID(msg.jobID)))
		}
		return m, tea.Batch(
			m.showToast("success", fmt.Sprintf("%s crack job %s", crackTopJobActionPastTense(msg.action), crackTopShortID(msg.jobID))),
			m.requestRefresh(),
		)

	case crackTopToastExpiredMsg:
		if msg.generation == m.toastGeneration {
			if m.toastCancel != nil {
				m.toastCancel()
			}
			m.toastCancel = nil
			m.toast = ""
			m.toastLevel = ""
		}
		return m, nil

	case crackTopListenerClosedMsg:
		m.listener = nil
		m.eventStreamClosed = true
		return m, nil

	case spinner.TickMsg:
		if !m.refreshing {
			return m, nil
		}
		var command tea.Cmd
		m.spinner, command = m.spinner.Update(msg)
		return m, command

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.pendingJobAction.jobID != "" {
			switch msg.String() {
			case "y", "Y":
				return m, m.confirmJobAction()
			case "n", "N", "esc":
				m.pendingJobAction = crackTopPendingJobAction{}
			}
			return m, nil
		}
		if m.filterForm != nil && msg.String() == "esc" {
			m.filterForm = nil
			return m, nil
		}
	}

	if m.filterForm != nil {
		updated, command := m.filterForm.Update(message)
		if form, ok := updated.(*huh.Form); ok {
			m.filterForm = form
		}
		switch m.filterForm.State {
		case huh.StateCompleted:
			m.filter = m.filterDraft
			m.filterForm = nil
			m.normalizeJobSelection()
		case huh.StateAborted:
			m.filterForm = nil
		}
		return m, command
	}

	key, ok := message.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "q", "esc":
		return m, tea.Quit
	case "tab", "shift+tab":
		if m.focus == crackTopFocusJobs {
			m.focus = crackTopFocusWorkers
		} else {
			m.focus = crackTopFocusJobs
		}
	case "up", "k":
		m.moveSelection(-1)
	case "down", "j":
		m.moveSelection(1)
	case "home", "g":
		m.moveSelectionTo(0)
	case "end", "G":
		if m.focus == crackTopFocusWorkers {
			m.moveSelectionTo(len(m.dashboard.Workers) - 1)
		} else {
			m.moveSelectionTo(len(m.filteredJobs()) - 1)
		}
	case "f":
		return m, m.openFilter()
	case "d", "delete":
		return m, m.openJobAction(crackTopJobActionDelete, true)
	case "p":
		return m, m.openJobAction(crackTopJobActionPause, true)
	case "u":
		return m, m.openJobAction(crackTopJobActionResume, false)
	case "c":
		return m, m.openJobAction(crackTopJobActionCancel, true)
	case "r":
		return m, m.requestRefresh()
	}
	return m, nil
}

func (m *crackTopModel) View() tea.View {
	width := max(1, m.width)
	height := max(1, m.height)
	if width < crackTopMinWidth || height < crackTopMinHeight {
		message := fmt.Sprintf("crack top needs at least %dx%d; terminal is %dx%d", crackTopMinWidth, crackTopMinHeight, width, height)
		message = crackTopFitLine(m.styles.warning.Render(message), width)
		content := lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, message)
		return crackTopView(content)
	}
	header := m.renderHeader(width)
	bodyHeight := height - crackTopHeaderHeight - crackTopFooterHeight
	var body string
	if width >= crackTopWideWidth {
		const gap = 1
		jobsWidth := (width - gap) * 3 / 5
		workersWidth := width - gap - jobsWidth
		body = lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.renderJobsPane(jobsWidth, bodyHeight),
			strings.Repeat(" ", gap),
			m.renderWorkersPane(workersWidth, bodyHeight),
		)
	} else {
		jobsHeight := max(5, bodyHeight*3/5)
		workersHeight := bodyHeight - jobsHeight
		if workersHeight < 4 {
			workersHeight = 4
			jobsHeight = bodyHeight - workersHeight
		}
		body = lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderJobsPane(width, jobsHeight),
			m.renderWorkersPane(width, workersHeight),
		)
	}
	footer := m.renderFooter(width)
	content := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
	if m.filterForm != nil {
		content = m.renderFilterModal(content, width, height)
	} else if m.pendingJobAction.jobID != "" {
		content = m.renderJobActionModal(content, width, height)
	}
	return crackTopView(content)
}

func crackTopView(content string) tea.View {
	view := tea.NewView(content)
	view.AltScreen = true
	return view
}

func (m *crackTopModel) fetchCmd() tea.Cmd {
	fetch := m.fetch
	ctx := m.ctx
	return func() tea.Msg {
		if fetch == nil {
			return crackTopSnapshotMsg{err: errors.New("crack top data loader is unavailable")}
		}
		snapshot, err := fetch(ctx)
		return crackTopSnapshotMsg{snapshot: snapshot, err: err}
	}
}

func (m *crackTopModel) requestRefresh() tea.Cmd {
	if m.refreshing {
		m.refreshQueued = true
		return nil
	}
	return m.startRefresh()
}

func (m *crackTopModel) startRefresh() tea.Cmd {
	m.refreshing = true
	if m.pollCancel != nil {
		m.pollCancel()
		m.pollCancel = nil
	}
	m.pollGeneration++ // Invalidate the outstanding periodic timer.
	if m.eventRefreshPending {
		if m.eventCancel != nil {
			m.eventCancel()
			m.eventCancel = nil
		}
		m.eventRefreshPending = false
		m.eventGeneration++ // This refresh satisfies the pending event invalidation.
	}
	return tea.Batch(m.fetchCmd(), m.spinner.Tick)
}

func (m *crackTopModel) schedulePoll() tea.Cmd {
	if m.pollCancel != nil {
		m.pollCancel()
	}
	m.pollGeneration++
	generation := m.pollGeneration
	timerCtx, cancel := context.WithCancel(m.ctx)
	m.pollCancel = cancel
	return crackTopTimerCmd(timerCtx, m.pollInterval, crackTopPollMsg{generation: generation})
}

func (m *crackTopModel) scheduleEventRefresh() tea.Cmd {
	if m.eventCancel != nil {
		m.eventCancel()
	}
	m.eventRefreshPending = true
	m.eventGeneration++
	generation := m.eventGeneration
	delay := min(m.pollInterval, crackTopEventDebounce)
	timerCtx, cancel := context.WithCancel(m.ctx)
	m.eventCancel = cancel
	return crackTopTimerCmd(timerCtx, delay, crackTopEventRefreshMsg{generation: generation})
}

func (m *crackTopModel) showToast(level, message string) tea.Cmd {
	m.toast = crackTopWarningText(message)
	m.toastLevel = strings.ToLower(strings.TrimSpace(level))
	m.toastGeneration++
	generation := m.toastGeneration
	if m.toastCancel != nil {
		m.toastCancel()
	}
	toastCtx, cancel := context.WithCancel(m.ctx)
	m.toastCancel = cancel
	return crackTopTimerCmd(toastCtx, crackTopToastDuration, crackTopToastExpiredMsg{generation: generation})
}

func (m *crackTopModel) openJobAction(action crackTopJobAction, confirm bool) tea.Cmd {
	if m.width < crackTopMinWidth || m.height < crackTopMinHeight {
		// View renders only the compact-size warning at these dimensions, so a
		// confirmation could not be shown safely.
		return nil
	}
	if m.runningJobAction.jobID != "" {
		return m.showToast("warning", fmt.Sprintf(
			"Cannot %s another crack job while a %s operation is in progress for job %s",
			action,
			m.runningJobAction.action,
			crackTopShortID(m.runningJobAction.jobID),
		))
	}
	if m.focus != crackTopFocusJobs {
		return m.showToast("warning", fmt.Sprintf("Select the jobs pane before trying to %s a crack job", action))
	}
	job, ok := m.selectedJob()
	if !ok {
		return m.showToast("warning", "No crack job is selected")
	}
	if warning := crackTopJobActionEligibilityWarning(action, job.Status); warning != "" {
		return m.showToast("warning", warning)
	}
	if m.jobActions[action] == nil {
		return m.showToast("error", fmt.Sprintf("Crack job %s is unavailable", action))
	}
	pending := crackTopPendingJobAction{action: action, jobID: job.ID}
	if confirm {
		m.pendingJobAction = pending
		return nil
	}
	return m.startJobAction(pending)
}

func (m *crackTopModel) selectedJob() (crackTopJobRow, bool) {
	for _, job := range m.filteredJobs() {
		if job.ID == m.selectedJobID {
			return job, true
		}
	}
	return crackTopJobRow{}, false
}

func crackTopJobStatusDeletable(status clientpb.CrackJobStatus) bool {
	switch status {
	case clientpb.CrackJobStatus_COMPLETED, clientpb.CrackJobStatus_FAILED, clientpb.CrackJobStatus_CANCELLED:
		return true
	default:
		return false
	}
}

func crackTopJobActionEligibilityWarning(action crackTopJobAction, status clientpb.CrackJobStatus) string {
	switch action {
	case crackTopJobActionDelete:
		if !crackTopJobStatusDeletable(status) {
			return "Only completed, failed, or cancelled crack jobs can be deleted"
		}
	case crackTopJobActionPause:
		if status != clientpb.CrackJobStatus_IN_PROGRESS {
			return "Only in-progress crack jobs can be paused"
		}
	case crackTopJobActionResume:
		if status != clientpb.CrackJobStatus_PAUSED {
			return "Only paused crack jobs can be resumed"
		}
	case crackTopJobActionCancel:
		if status != clientpb.CrackJobStatus_IN_PROGRESS && status != clientpb.CrackJobStatus_PAUSED {
			return "Only in-progress or paused crack jobs can be cancelled"
		}
	default:
		return "Unknown crack job action"
	}
	return ""
}

func (m *crackTopModel) confirmJobAction() tea.Cmd {
	pending := m.pendingJobAction
	m.pendingJobAction = crackTopPendingJobAction{}
	return m.startJobAction(pending)
}

func (m *crackTopModel) startJobAction(pending crackTopPendingJobAction) tea.Cmd {
	actionJob := m.jobActions[pending.action]
	if pending.jobID == "" || actionJob == nil {
		return nil
	}
	m.runningJobAction = pending
	if m.toastCancel != nil {
		m.toastCancel()
		m.toastCancel = nil
	}
	m.toast = ""
	m.toastLevel = ""
	ctx := m.ctx
	return func() tea.Msg {
		job, err := actionJob(ctx, pending.jobID)
		if err == nil {
			err = crackTopValidateJobActionResult(pending, job)
		}
		message := crackTopJobActionCompletedMsg{action: pending.action, jobID: pending.jobID, err: err}
		if err == nil && job != nil {
			message.status = job.GetStatus()
			message.updatedAt = job.GetUpdatedAt()
			message.hasStatus = true
		}
		return message
	}
}

func crackTopValidateJobActionResult(pending crackTopPendingJobAction, job *clientpb.CrackJob) error {
	if pending.action == crackTopJobActionDelete {
		return nil
	}
	if job == nil {
		return errors.New("server returned no job state")
	}
	if job.GetID() != pending.jobID {
		return fmt.Errorf("server returned state for unexpected job %q", crackTopWarningText(job.GetID()))
	}
	if job.GetUpdatedAt() <= 0 {
		return errors.New("server returned job state without an update timestamp")
	}
	expected := clientpb.CrackJobStatus_IN_PROGRESS
	switch pending.action {
	case crackTopJobActionPause:
		expected = clientpb.CrackJobStatus_PAUSED
	case crackTopJobActionCancel:
		expected = clientpb.CrackJobStatus_CANCELLED
	case crackTopJobActionResume:
		expected = clientpb.CrackJobStatus_IN_PROGRESS
	default:
		return fmt.Errorf("unknown crack job action %q", pending.action)
	}
	if job.GetStatus() != expected {
		return fmt.Errorf("server returned unexpected job status %s (want %s)", job.GetStatus(), expected)
	}
	return nil
}

func crackTopJobActionPastTense(action crackTopJobAction) string {
	switch action {
	case crackTopJobActionDelete:
		return "Deleted"
	case crackTopJobActionPause:
		return "Paused"
	case crackTopJobActionResume:
		return "Resumed"
	case crackTopJobActionCancel:
		return "Cancelled"
	default:
		return "Updated"
	}
}

func crackTopJobActionPresentParticiple(action crackTopJobAction) string {
	switch action {
	case crackTopJobActionDelete:
		return "deleting"
	case crackTopJobActionPause:
		return "pausing"
	case crackTopJobActionResume:
		return "resuming"
	case crackTopJobActionCancel:
		return "cancelling"
	default:
		return "updating"
	}
}

func (m *crackTopModel) removeJob(jobID string) {
	if m.deletedJobIDs == nil {
		m.deletedJobIDs = map[string]struct{}{}
	}
	m.deletedJobIDs[jobID] = struct{}{}
	if m.snapshot == nil {
		return
	}
	removeCrackTopJobFromSnapshot(m.snapshot, jobID)
	m.dashboard = buildCrackTopDashboard(m.snapshot)
	m.normalizeJobSelection()
	m.normalizeWorkerSelection()
}

func (m *crackTopModel) suppressDeletedJobs(snapshot *crackTopSnapshot) {
	if snapshot == nil || len(m.deletedJobIDs) == 0 {
		return
	}
	for jobID := range m.deletedJobIDs {
		found := false
		for _, job := range snapshot.Jobs {
			if job != nil && job.GetID() == jobID {
				found = true
				break
			}
		}
		if found {
			removeCrackTopJobFromSnapshot(snapshot, jobID)
		} else {
			delete(m.deletedJobIDs, jobID)
		}
	}
}

func (m *crackTopModel) overrideJobStatus(jobID string, status clientpb.CrackJobStatus, updatedAt int64) {
	if m.jobStatusOverrides == nil {
		m.jobStatusOverrides = map[string]crackTopJobStatusOverride{}
	}
	m.jobStatusOverrides[jobID] = crackTopJobStatusOverride{status: status, updatedAt: updatedAt}
	if m.snapshot == nil {
		return
	}
	for _, job := range m.snapshot.Jobs {
		if job == nil || job.GetID() != jobID {
			continue
		}
		job.Status = status
		if updatedAt > job.GetUpdatedAt() {
			job.UpdatedAt = updatedAt
		}
		break
	}
	m.dashboard = buildCrackTopDashboard(m.snapshot)
	m.normalizeJobSelection()
	m.normalizeWorkerSelection()
}

// An action can complete while a snapshot requested before the action is still
// in flight. Keep the authoritative RPC result visible until a snapshot either
// observes that state or advances beyond the action that produced it.
func (m *crackTopModel) applyJobStatusOverrides(snapshot *crackTopSnapshot) {
	if snapshot == nil || len(m.jobStatusOverrides) == 0 {
		return
	}
	for jobID, override := range m.jobStatusOverrides {
		found := false
		for _, job := range snapshot.Jobs {
			if job == nil || job.GetID() != jobID {
				continue
			}
			found = true
			if job.GetStatus() == override.status || job.GetUpdatedAt() > override.updatedAt {
				delete(m.jobStatusOverrides, jobID)
				break
			}
			// UpdatedAt is second-resolution on the wire. One equal-version
			// disagreement can be the snapshot that was already in flight when the
			// action completed; a repeated disagreement is authoritative evidence
			// of another lifecycle action in that same second.
			if job.GetUpdatedAt() == override.updatedAt {
				if override.equalVersionDisagreementSeen {
					delete(m.jobStatusOverrides, jobID)
					break
				}
				override.equalVersionDisagreementSeen = true
				m.jobStatusOverrides[jobID] = override
			}
			job.Status = override.status
			if override.updatedAt > job.GetUpdatedAt() {
				job.UpdatedAt = override.updatedAt
			}
			break
		}
		if !found {
			delete(m.jobStatusOverrides, jobID)
		}
	}
}

func removeCrackTopJobFromSnapshot(snapshot *crackTopSnapshot, jobID string) {
	if snapshot == nil {
		return
	}
	jobs := make([]*clientpb.CrackJob, 0, len(snapshot.Jobs))
	for _, job := range snapshot.Jobs {
		if job == nil || job.GetID() != jobID {
			jobs = append(jobs, job)
			continue
		}
		for _, task := range job.GetTasks() {
			if task != nil {
				delete(snapshot.taskTelemetry, task.GetID())
			}
		}
	}
	snapshot.Jobs = jobs
}

func crackTopTimerCmd(ctx context.Context, delay time.Duration, message tea.Msg) tea.Cmd {
	return func() tea.Msg {
		timer := time.NewTimer(max(time.Millisecond, delay))
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
			return message
		}
	}
}

func crackTopWindowPollCmd(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		timer := time.NewTimer(crackTopWindowPoll)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
		}
		width, height, ok := crackTopCurrentTerminalSize()
		return crackTopWindowPollMsg{width: width, height: height, ok: ok}
	}
}

func waitForCrackTopEventCmd(listener <-chan *clientpb.Event) tea.Cmd {
	return func() tea.Msg {
		if listener == nil {
			return crackTopListenerClosedMsg{}
		}
		for {
			event, ok := <-listener
			if !ok {
				return crackTopListenerClosedMsg{}
			}
			if event == nil {
				continue
			}
			switch event.GetEventType() {
			case consts.ClientToastEvent:
				return crackTopToastMsg{
					level:   event.GetErr(),
					message: string(event.GetData()),
				}
			case consts.CrackTaskStatus,
				consts.CrackJobCreated,
				consts.CrackJobUpdated,
				consts.CredentialCrackedEvent,
				consts.CrackstationConnected,
				consts.CrackstationDisconnected,
				consts.CrackStatusEvent:
				return crackTopEventMsg{}
			}
		}
	}
}

func (m *crackTopModel) filteredJobs() []crackTopJobRow {
	if m == nil {
		return nil
	}
	rows := make([]crackTopJobRow, 0, len(m.dashboard.Jobs))
	for _, row := range m.dashboard.Jobs {
		matches := false
		switch m.filter {
		case crackTopFilterActive:
			matches = row.Status == clientpb.CrackJobStatus_IN_PROGRESS || row.Status == clientpb.CrackJobStatus_PAUSED
		case crackTopFilterPaused:
			matches = row.Status == clientpb.CrackJobStatus_PAUSED
		case crackTopFilterCompleted:
			matches = row.Status == clientpb.CrackJobStatus_COMPLETED
		case crackTopFilterFailed:
			matches = row.Status == clientpb.CrackJobStatus_FAILED || row.Status == clientpb.CrackJobStatus_CANCELLED
		default:
			matches = true
		}
		if matches {
			rows = append(rows, row)
		}
	}
	return rows
}

func (m *crackTopModel) normalizeJobSelection() {
	jobs := m.filteredJobs()
	if len(jobs) == 0 {
		m.jobCursor = 0
		m.selectedJobID = ""
		return
	}
	if m.selectedJobID != "" {
		for index, job := range jobs {
			if job.ID == m.selectedJobID {
				m.jobCursor = index
				return
			}
		}
	}
	if m.selectedJobID == "" {
		for index, job := range jobs {
			if job.Status == clientpb.CrackJobStatus_IN_PROGRESS {
				m.jobCursor = index
				m.selectedJobID = job.ID
				return
			}
		}
	}
	m.jobCursor = max(0, min(m.jobCursor, len(jobs)-1))
	m.selectedJobID = jobs[m.jobCursor].ID
}

func (m *crackTopModel) moveJobSelection(delta int) {
	jobs := m.filteredJobs()
	if len(jobs) == 0 {
		return
	}
	m.jobCursor = max(0, min(m.jobCursor+delta, len(jobs)-1))
	m.selectedJobID = jobs[m.jobCursor].ID
}

func (m *crackTopModel) moveJobSelectionTo(index int) {
	jobs := m.filteredJobs()
	if len(jobs) == 0 {
		return
	}
	m.jobCursor = max(0, min(index, len(jobs)-1))
	m.selectedJobID = jobs[m.jobCursor].ID
}

func (m *crackTopModel) normalizeWorkerSelection() {
	workers := m.dashboard.Workers
	if len(workers) == 0 {
		m.workerCursor = 0
		m.selectedWorkerID = ""
		return
	}
	if m.selectedWorkerID != "" {
		for index, worker := range workers {
			if worker.ID == m.selectedWorkerID {
				m.workerCursor = index
				return
			}
		}
	}
	m.workerCursor = max(0, min(m.workerCursor, len(workers)-1))
	m.selectedWorkerID = workers[m.workerCursor].ID
}

func (m *crackTopModel) moveSelection(delta int) {
	if m.focus == crackTopFocusWorkers {
		workers := m.dashboard.Workers
		if len(workers) == 0 {
			return
		}
		m.workerCursor = max(0, min(m.workerCursor+delta, len(workers)-1))
		m.selectedWorkerID = workers[m.workerCursor].ID
		return
	}
	m.moveJobSelection(delta)
}

func (m *crackTopModel) moveSelectionTo(index int) {
	if m.focus == crackTopFocusWorkers {
		workers := m.dashboard.Workers
		if len(workers) == 0 {
			return
		}
		m.workerCursor = max(0, min(index, len(workers)-1))
		m.selectedWorkerID = workers[m.workerCursor].ID
		return
	}
	m.moveJobSelectionTo(index)
}

func (m *crackTopModel) openFilter() tea.Cmd {
	m.filterDraft = m.filter
	field := huh.NewSelect[crackTopFilter]().
		Title("Job scope").
		Description("Choose which loaded crack jobs appear. Use / to search.").
		Options(
			huh.NewOption("All loaded jobs", crackTopFilterAll),
			huh.NewOption("Active or paused", crackTopFilterActive),
			huh.NewOption("Paused", crackTopFilterPaused),
			huh.NewOption("Completed", crackTopFilterCompleted),
			huh.NewOption("Failed or cancelled", crackTopFilterFailed),
		).
		Key("scope").
		Value(&m.filterDraft).
		Height(5)
	m.filterForm = huh.NewForm(huh.NewGroup(field)).
		WithTheme(clienttheme.HuhTheme()).
		WithShowErrors(false).
		WithShowHelp(true)
	m.resizeFilterForm()
	return m.filterForm.Init()
}

func (m *crackTopModel) resizeFilterForm() {
	if m == nil || m.filterForm == nil {
		return
	}
	// The card uses two cells of border, four of horizontal padding, and two
	// lines each for its title/footer and vertical padding.
	formWidth := max(18, min(52, m.width-10))
	formHeight := max(6, min(10, m.height-6))
	m.filterForm.WithWidth(formWidth).WithHeight(formHeight)
}

func (m *crackTopModel) renderFilterModal(background string, width, height int) string {
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.title.Render("Filter crack jobs"),
		m.filterForm.View(),
		m.styles.muted.Render("enter apply  •  esc cancel  •  / search"),
	)
	card := m.styles.filterCard.Render(content)
	card = crackTopFitBlock(card, width, height)
	left := max(0, (width-lipgloss.Width(card))/2)
	top := max(0, (height-lipgloss.Height(card))/2)
	compositor := lipgloss.NewCompositor(
		lipgloss.NewLayer(background).Z(0),
		lipgloss.NewLayer(card).X(left).Y(top).Z(1),
	)
	return lipgloss.NewCanvas(width, height).Compose(compositor).Render()
}

func (m *crackTopModel) renderJobActionModal(background string, width, height int) string {
	prompt := m.pendingJobAction
	contentWidth := max(18, min(68, width-8))
	title := "Update crack job?"
	description := "Apply this action to the selected crack job?"
	warning := ""
	confirm := "y confirm"
	titleStyle := m.styles.warning
	border := clienttheme.Warning()
	switch prompt.action {
	case crackTopJobActionDelete:
		title = "Delete crack job?"
		description = "Permanently delete this terminal job and all stored tasks and results?"
		warning = "This cannot be undone."
		confirm = "y delete"
		titleStyle = m.styles.danger
		border = clienttheme.Danger()
	case crackTopJobActionPause:
		title = "Pause crack job?"
		description = "Pause this job and release its current crackstation work?"
		warning = "An in-flight task attempt may restart when the job is resumed."
		confirm = "y pause"
	case crackTopJobActionCancel:
		title = "Cancel crack job?"
		description = "Stop this job and release all assigned crackstation work?"
		warning = "Cancelled jobs cannot be resumed."
		confirm = "y cancel"
		titleStyle = m.styles.danger
		border = clienttheme.Danger()
	}
	lines := []string{
		titleStyle.Render(title),
		m.styles.normal.Width(contentWidth).Render(description),
		m.styles.primary.Render(prompt.jobID),
	}
	if warning != "" {
		lines = append(lines, m.styles.warning.Render(warning))
	}
	lines = append(lines, m.styles.muted.Render(confirm+"  •  n/esc cancel"))
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		lines...,
	)
	card := m.styles.actionCard.BorderForeground(border).Width(contentWidth).Render(content)
	card = crackTopFitBlock(card, width, height)
	left := max(0, (width-lipgloss.Width(card))/2)
	top := max(0, (height-lipgloss.Height(card))/2)
	compositor := lipgloss.NewCompositor(
		lipgloss.NewLayer(background).Z(0),
		lipgloss.NewLayer(card).X(left).Y(top).Z(1),
	)
	return lipgloss.NewCanvas(width, height).Compose(compositor).Render()
}

func (m *crackTopModel) renderHeader(width int) string {
	innerWidth := max(1, width-4)
	activity := ""
	if m.refreshing {
		activity = m.spinner.View() + " refreshing"
	} else if m.snapshot != nil && !m.snapshot.RefreshedAt.IsZero() {
		activity = "updated " + m.snapshot.RefreshedAt.Local().Format("15:04:05")
	} else {
		activity = "waiting for data"
	}
	if m.eventStreamClosed {
		activity += " • polling only"
	}
	lineOne := crackTopJoinEdges(m.styles.title.Render("CRACK TOP"), m.styles.muted.Render(activity), innerWidth)

	totalWorkers := m.dashboard.OnlineWorkers + m.dashboard.Unavailable
	clusterRate := "—"
	if m.dashboard.ClusterRateKnown {
		clusterRate = humanizeHashRate(m.dashboard.ClusterRate)
		if !m.dashboard.ClusterComplete {
			clusterRate = "≥ " + clusterRate
		}
	}
	if !m.dashboard.ClusterComplete && m.dashboard.ExpectedWorkers > 0 {
		clusterRate += fmt.Sprintf(" (%d/%d live)", m.dashboard.ReportingWorkers, m.dashboard.ExpectedWorkers)
	}
	metrics := fmt.Sprintf(
		"ACTIVE %d   JOBS %d loaded   WORKERS %d/%d online   CLUSTER %s   RECOVERED %d",
		m.dashboard.ActiveJobs,
		len(m.dashboard.Jobs),
		m.dashboard.OnlineWorkers,
		totalWorkers,
		clusterRate,
		m.dashboard.Recovered,
	)
	lineTwo := crackTopFitLine(m.styles.normal.Render(metrics), innerWidth)

	progressLabel := "GLOBAL "
	progressText := " awaiting keyspace"
	if m.dashboard.ProgressKnown {
		progressText = fmt.Sprintf(" %5.1f%%", m.dashboard.Progress*100)
	} else if m.dashboard.ActiveJobs == 0 {
		progressText = " no active jobs"
	} else {
		if m.dashboard.ProgressJobs > 0 {
			progressText = fmt.Sprintf(" %d/%d jobs ready", m.dashboard.ProgressJobs, m.dashboard.ActiveJobs)
		}
	}
	progressWidth := max(6, innerWidth-ansi.StringWidth(progressLabel)-ansi.StringWidth(progressText))
	bar := m.renderUnknownProgress(progressWidth)
	if m.dashboard.ProgressKnown {
		bar = m.renderProgressBar(progressWidth, m.dashboard.Progress, clientpb.CrackJobStatus_IN_PROGRESS)
	}
	lineThree := crackTopFitLine(m.styles.heading.Render(progressLabel)+bar+m.styles.muted.Render(progressText), innerWidth)
	border := clienttheme.Primary()
	if m.lastError != "" {
		border = clienttheme.Warning()
	}
	return crackTopPanel(width, crackTopHeaderHeight, border, strings.Join([]string{lineOne, lineTwo, lineThree}, "\n"))
}

func (m *crackTopModel) renderJobsPane(width, height int) string {
	innerWidth := max(1, width-4)
	innerHeight := max(1, height-2)
	jobs := m.filteredJobs()
	capacity := max(1, (innerHeight-1)/2)
	start, end := crackTopVisibleRange(len(jobs), m.jobCursor, capacity)
	header := fmt.Sprintf("JOBS  %d shown  •  all active + recent history  •  filter: %s", len(jobs), m.filter)
	if len(jobs) > capacity {
		header += fmt.Sprintf("  •  %d-%d", start+1, end)
	}
	lines := []string{crackTopFitLine(m.styles.heading.Render(header), innerWidth)}
	if len(jobs) == 0 {
		lines = append(lines, m.styles.muted.Render("No jobs match this filter."))
	}
	for index := start; index < end; index++ {
		row := jobs[index]
		selected := row.ID == m.selectedJobID
		cursor := "  "
		idStyle := m.styles.primary
		if selected {
			cursor = m.styles.selected.Render("› ")
			idStyle = m.styles.selected
		}
		rate := "—"
		if row.RateKnown {
			rate = humanizeHashRate(row.Rate)
		}
		workers := strconv.Itoa(row.Workers) + " workers"
		if row.Workers == 1 {
			workers = "1 worker"
		}
		if row.Reporting != row.Workers {
			workers = fmt.Sprintf("%d/%d live", row.Reporting, row.Workers)
		}
		warning := ""
		if row.TelemetryErrors > 0 {
			warning = fmt.Sprintf("  %s", m.styles.warning.Render("telemetry !"))
		}
		first := cursor + idStyle.Render(crackTopShortID(row.ID)) + "  " +
			m.renderJobStatus(row.Status) + "  " + m.styles.secondary.Render(rate) + "  " +
			m.styles.muted.Render(workers+" • "+strconv.FormatUint(row.Results, 10)+" found") + warning
		lines = append(lines, crackTopFitLine(first, innerWidth))

		percent := "  --.-%"
		if row.ProgressKnown {
			percent = fmt.Sprintf("  %5.1f%%", row.Progress*100)
			if !row.ProgressComplete {
				percent = fmt.Sprintf(" ≥%5.1f%%", row.Progress*100)
			}
		}
		mode := ""
		if row.Mode != "" && innerWidth >= 54 {
			mode = "  " + m.styles.muted.Render(row.Mode)
		}
		barWidth := max(4, innerWidth-2-ansi.StringWidth(percent)-ansi.StringWidth(mode))
		bar := m.renderUnknownProgress(barWidth)
		if row.ProgressKnown {
			bar = m.renderProgressBar(barWidth, row.Progress, row.Status)
		}
		second := "  " + bar + m.styles.muted.Render(percent) + mode
		lines = append(lines, crackTopFitLine(second, innerWidth))
	}
	border := clienttheme.DefaultMod(300)
	if m.focus == crackTopFocusJobs {
		border = clienttheme.Primary()
	}
	return crackTopPanel(width, height, border, strings.Join(lines, "\n"))
}

//nolint:gocyclo // Worker rendering keeps responsive layout and telemetry-state formatting together.
func (m *crackTopModel) renderWorkersPane(width, height int) string {
	innerWidth := max(1, width-4)
	innerHeight := max(1, height-2)
	workers := m.dashboard.Workers
	capacity := max(1, (innerHeight-1)/2)
	start, end := crackTopVisibleRange(len(workers), m.workerCursor, capacity)
	header := fmt.Sprintf("CRACKSTATIONS  %d online", m.dashboard.OnlineWorkers)
	if len(workers) > capacity {
		header += fmt.Sprintf("  •  %d-%d/%d", start+1, end, len(workers))
	}
	lines := []string{crackTopFitLine(m.styles.heading.Render(header), innerWidth)}
	if len(workers) == 0 {
		lines = append(lines, m.styles.muted.Render("No connected or assigned crackstations."))
	}
	for index := start; index < end; index++ {
		worker := workers[index]
		cursor := "  "
		identityStyle := m.styles.primary
		if worker.ID == m.selectedWorkerID {
			cursor = m.styles.selected.Render("› ")
			identityStyle = m.styles.selected
		}
		dot := m.styles.danger.Render("●")
		if worker.Online {
			dot = m.styles.success.Render("●")
		}
		state := worker.State
		if worker.Syncing {
			state = "SYNCING"
		}
		rate := "—"
		if worker.RateKnown {
			rate = humanizeHashRate(worker.Rate)
		} else if worker.Online && strings.EqualFold(state, "IDLE") {
			rate = humanizeHashRate(0)
		}
		identity := worker.Name + " [" + crackTopShortID(worker.ID) + "]"
		first := cursor + dot + " " + identityStyle.Render(identity) + "  " +
			m.renderWorkerState(state, worker.Online) + "  " + m.styles.secondary.Render(rate)
		lines = append(lines, crackTopFitLine(first, innerWidth))

		details := make([]string, 0, 5)
		if worker.JobID != "" {
			details = append(details, "job "+crackTopShortID(worker.JobID))
		} else if worker.Online && strings.EqualFold(state, "IDLE") {
			details = append(details, "idle")
		} else if worker.Online && strings.EqualFold(state, "ONLINE") {
			details = append(details, "status unavailable")
		}
		if worker.Devices > 0 {
			details = append(details, fmt.Sprintf("%d devices", worker.Devices))
		}
		if worker.HasTemp {
			details = append(details, fmt.Sprintf("%.0f C", worker.Temperature))
		}
		if worker.HasUtil {
			details = append(details, fmt.Sprintf("%.0f%% util", worker.Utilization))
		}
		if !worker.LastSeen.IsZero() {
			seen := "last telemetry "
			if worker.TelemetryLive {
				seen = "seen "
			}
			details = append(details, seen+worker.LastSeen.Local().Format("15:04:05"))
		}
		if len(details) == 0 {
			details = append(details, "no live device telemetry")
		}
		lines = append(lines, crackTopFitLine("    "+m.styles.muted.Render(strings.Join(details, " • ")), innerWidth))
	}
	border := clienttheme.DefaultMod(300)
	if m.focus == crackTopFocusWorkers {
		border = clienttheme.Primary()
	}
	return crackTopPanel(width, height, border, strings.Join(lines, "\n"))
}

func (m *crackTopModel) renderFooter(width int) string {
	message := "tab pane  •  ↑/k ↓/j  •  p pause  •  u resume  •  c cancel  •  d delete  •  f filter  •  r refresh  •  q quit"
	style := m.styles.muted
	if m.toast != "" {
		message = m.toast
		if m.lastError != "" {
			message = "DEGRADED  •  " + message
		}
		switch m.toastLevel {
		case "error", "danger":
			style = m.styles.danger
		case "warn", "warning":
			style = m.styles.warning
		case "success":
			style = m.styles.success
		default:
			style = m.styles.primary
		}
	} else if m.runningJobAction.jobID != "" {
		progress := crackTopJobActionPresentParticiple(m.runningJobAction.action)
		message = fmt.Sprintf(
			"%s crack job %s…",
			strings.ToUpper(progress[:1])+progress[1:],
			crackTopShortID(m.runningJobAction.jobID),
		)
		style = m.styles.warning
	} else if m.lastError != "" {
		retained := "no snapshot available"
		if m.snapshot != nil {
			retained = "showing last snapshot"
		}
		message = "DEGRADED: " + m.lastError + "  •  " + retained + "  •  r retry  •  q quit"
		style = m.styles.warning
	}
	return crackTopPadLine(style.Render(message), width)
}

func (m *crackTopModel) renderJobStatus(status clientpb.CrackJobStatus) string {
	text := status.String()
	switch status {
	case clientpb.CrackJobStatus_COMPLETED:
		return m.styles.success.Render(text)
	case clientpb.CrackJobStatus_FAILED:
		return m.styles.danger.Render(text)
	case clientpb.CrackJobStatus_CANCELLED:
		return m.styles.muted.Render(text)
	case clientpb.CrackJobStatus_PAUSED:
		return m.styles.secondary.Render(text)
	default:
		return m.styles.warning.Render(text)
	}
}

func (m *crackTopModel) renderWorkerState(state string, online bool) string {
	state = valueOrDash(state)
	if !online {
		return m.styles.danger.Render("OFFLINE")
	}
	switch strings.ToUpper(state) {
	case "CRACKING":
		return m.styles.warning.Render(state)
	case "IDLE":
		return m.styles.success.Render(state)
	case "INITIALIZING", "SYNCING":
		return m.styles.secondary.Render(state)
	default:
		return m.styles.normal.Render(state)
	}
}

func (m *crackTopModel) renderProgressBar(width int, fraction float64, status clientpb.CrackJobStatus) string {
	fill := clienttheme.Primary()
	switch status {
	case clientpb.CrackJobStatus_COMPLETED:
		fill = clienttheme.Success()
	case clientpb.CrackJobStatus_FAILED:
		fill = clienttheme.Danger()
	case clientpb.CrackJobStatus_CANCELLED:
		fill = clienttheme.DefaultMod(500)
	case clientpb.CrackJobStatus_PAUSED:
		fill = clienttheme.Secondary()
	}
	bar := progress.New(
		progress.WithWidth(max(1, width)),
		progress.WithoutPercentage(),
		progress.WithColors(fill, clienttheme.DefaultMod(100)),
		progress.WithFillCharacters('━', '─'),
	)
	return bar.ViewAs(max(0, min(1, fraction)))
}

func (m *crackTopModel) renderUnknownProgress(width int) string {
	return m.styles.muted.Render(strings.Repeat("·", max(1, width)))
}

func crackTopPanel(width, height int, border color.Color, content string) string {
	width = max(4, width)
	height = max(3, height)
	innerWidth := max(1, width-4)
	innerHeight := max(1, height-2)
	lines := strings.Split(content, "\n")
	if len(lines) > innerHeight {
		lines = lines[:innerHeight]
	}
	for index := range lines {
		lines[index] = crackTopFitLine(lines[index], innerWidth)
	}
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Render(strings.Join(lines, "\n"))
}

func crackTopVisibleRange(total, cursor, capacity int) (int, int) {
	if total <= 0 || capacity <= 0 {
		return 0, 0
	}
	cursor = max(0, min(cursor, total-1))
	start := cursor - capacity/2
	start = max(0, min(start, total-capacity))
	return start, min(total, start+capacity)
}

func crackTopJoinEdges(left, right string, width int) string {
	width = max(1, width)
	right = ansi.Truncate(right, width, "…")
	rightWidth := ansi.StringWidth(right)
	leftLimit := max(0, width-rightWidth-1)
	left = ansi.Truncate(left, leftLimit, "…")
	gap := max(1, width-ansi.StringWidth(left)-rightWidth)
	return crackTopFitLine(left+strings.Repeat(" ", gap)+right, width)
}

func crackTopFitLine(value string, width int) string {
	if width <= 0 {
		return ""
	}
	return ansi.Truncate(value, width, "…")
}

func crackTopFitBlock(value string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	lines := strings.Split(value, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for index := range lines {
		lines[index] = crackTopFitLine(lines[index], width)
	}
	return strings.Join(lines, "\n")
}

func crackTopPadLine(value string, width int) string {
	value = crackTopFitLine(value, width)
	return value + strings.Repeat(" ", max(0, width-ansi.StringWidth(value)))
}

func crackTopTerminalSize() (int, int) {
	if width, height, ok := crackTopCurrentTerminalSize(); ok {
		return width, height
	}
	return crackTopDefaultWidth, crackTopDefaultHeight
}

func crackTopCurrentTerminalSize() (int, int, bool) {
	files := []*os.File{os.Stdin, termio.InteractiveInput(), os.Stdout, termio.InteractiveOutput()}
	for _, file := range files {
		if file == nil || !term.IsTerminal(int(file.Fd())) {
			continue
		}
		if width, height, err := term.GetSize(int(file.Fd())); err == nil && width > 0 && height > 0 {
			return width, height, true
		}
	}
	return 0, 0, false
}

// Console logging can redirect stdout/stderr to pipes. Bind the dashboard back
// to the original interactive terminal so alternate-screen rendering remains
// isolated from session logs.
func configureCrackTopProgramTTY() ([]tea.ProgramOption, func()) {
	stdinTTY := crackTopIsTTY(os.Stdin)
	stdoutTTY := crackTopIsTTY(os.Stdout)
	if !stdinTTY || stdoutTTY {
		return nil, nil
	}
	if crackTopIsTTY(termio.InteractiveInput()) && crackTopIsTTY(termio.InteractiveOutput()) {
		return []tea.ProgramOption{
			tea.WithInput(termio.InteractiveInput()),
			tea.WithOutput(termio.InteractiveOutput()),
		}, nil
	}
	inTTY, outTTY, err := openCrackTopTTY()
	if err != nil {
		return nil, nil
	}
	return []tea.ProgramOption{
		tea.WithInput(inTTY),
		tea.WithOutput(outTTY),
	}, func() {
		_ = inTTY.Close()
		if outTTY != inTTY {
			_ = outTTY.Close()
		}
	}
}

func crackTopIsTTY(file *os.File) bool {
	return file != nil && term.IsTerminal(int(file.Fd()))
}
