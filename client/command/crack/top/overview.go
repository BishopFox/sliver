package top

import (
	"fmt"
	"image/color"
	"math"
	"math/big"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	clienttheme "github.com/bishopfox/sliver/client/theme"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/charmbracelet/x/ansi"
)

const (
	crackTopOverviewTopBarHeight   = 1
	crackTopOverviewSummaryHeight  = 6
	crackTopOverviewMetersHeight   = 4
	crackTopOverviewJournalHeight  = 5
	crackTopOverviewChartMinHeight = 6
	crackTopOverviewChartMaxHeight = 10
	crackTopJournalLimit           = 12
)

type crackTopJournalEntry struct {
	at      time.Time
	level   string
	label   string
	message string
}

type crackTopChartPoint struct {
	value      float64
	known      bool
	at         time.Time
	lowerBound bool
}

func (m *crackTopModel) authoritativeSnapshotPending() bool {
	return m != nil && (len(m.jobStatusOverrides) > 0 || len(m.deletedJobIDs) > 0)
}

//nolint:gocyclo // Observation recording intentionally evaluates all job, worker, recovery, and telemetry transitions together.
func (m *crackTopModel) recordDashboardObservation(previous crackTopDashboard, hadSnapshot bool, observedAt time.Time) {
	if observedAt.IsZero() {
		observedAt = time.Now()
	}
	if m.authoritativeSnapshotPending() {
		// An older snapshot can arrive after a lifecycle RPC. Its job status is
		// overridden locally, but its task/device payload is still pre-action.
		// Keep the entire observation unknown until a matching snapshot arrives.
		m.history = appendCrackTopHistory(m.history, crackTopHistorySample{At: observedAt})
	} else {
		m.history = appendCrackTopHistory(m.history, crackTopHistorySampleFromDashboard(m.dashboard, observedAt))
	}
	if !hadSnapshot {
		rate := "rate awaiting telemetry"
		if m.dashboard.ClusterRateKnown {
			rate = humanizeHashRate(m.dashboard.ClusterRate)
			if !m.dashboard.ClusterComplete {
				rate = "≥ " + rate
			}
		}
		m.appendJournal(crackTopJournalEntry{
			at:      observedAt,
			level:   "success",
			label:   "LIVE",
			message: fmt.Sprintf("snapshot ready • %d active • %d/%d workers • %s", m.dashboard.ActiveJobs, m.dashboard.OnlineWorkers, crackTopWorkerTotal(m.dashboard), rate),
		})
		return
	}

	previousJobs := make(map[string]crackTopJobRow, len(previous.Jobs))
	for _, job := range previous.Jobs {
		previousJobs[job.ID] = job
	}
	currentJobs := make(map[string]struct{}, len(m.dashboard.Jobs))
	for _, job := range m.dashboard.Jobs {
		currentJobs[job.ID] = struct{}{}
		old, existed := previousJobs[job.ID]
		switch {
		case !existed:
			m.appendJournal(crackTopJournalEntry{at: observedAt, level: "primary", label: "JOB", message: fmt.Sprintf("%s added • %s", crackTopShortID(job.ID), job.Status)})
		case old.Status != job.Status:
			m.appendJournal(crackTopJournalEntry{at: observedAt, level: crackTopJobJournalLevel(job.Status.String()), label: "JOB", message: fmt.Sprintf("%s %s", crackTopShortID(job.ID), job.Status)})
		}
	}
	for jobID, old := range previousJobs {
		if _, exists := currentJobs[jobID]; exists {
			continue
		}
		if old.Status == clientpb.CrackJobStatus_IN_PROGRESS || old.Status == clientpb.CrackJobStatus_PAUSED {
			m.appendJournal(crackTopJournalEntry{at: observedAt, level: "warning", label: "JOB", message: fmt.Sprintf("%s removed from live set", crackTopShortID(jobID))})
		}
	}

	previousWorkers := make(map[string]crackTopWorkerRow, len(previous.Workers))
	for _, worker := range previous.Workers {
		previousWorkers[worker.ID] = worker
	}
	currentWorkers := make(map[string]crackTopWorkerRow, len(m.dashboard.Workers))
	for _, worker := range m.dashboard.Workers {
		currentWorkers[worker.ID] = worker
		old, existed := previousWorkers[worker.ID]
		if worker.Online && (!existed || !old.Online) {
			m.appendJournal(crackTopJournalEntry{at: observedAt, level: "success", label: "WORKER", message: worker.Name + " online"})
		} else if !worker.Online && existed && old.Online {
			m.appendJournal(crackTopJournalEntry{at: observedAt, level: "warning", label: "WORKER", message: worker.Name + " offline"})
		}
	}
	for workerID, old := range previousWorkers {
		if _, exists := currentWorkers[workerID]; !exists && old.Online {
			m.appendJournal(crackTopJournalEntry{at: observedAt, level: "warning", label: "WORKER", message: old.Name + " disconnected"})
		}
	}

	if m.dashboard.Recovered > previous.Recovered {
		m.appendJournal(crackTopJournalEntry{
			at:      observedAt,
			level:   "success",
			label:   "RECOVERED",
			message: fmt.Sprintf("+%d across loaded jobs", m.dashboard.Recovered-previous.Recovered),
		})
	}
	if m.dashboard.ReportingWorkers != previous.ReportingWorkers || m.dashboard.ExpectedWorkers != previous.ExpectedWorkers {
		m.appendJournal(crackTopJournalEntry{
			at:      observedAt,
			level:   crackTopCompletenessJournalLevel(m.dashboard),
			label:   "TELEMETRY",
			message: fmt.Sprintf("%d/%d expected workers reporting", m.dashboard.ReportingWorkers, m.dashboard.ExpectedWorkers),
		})
	}
}

func (m *crackTopModel) recordRefreshFailure(observedAt time.Time, journal bool) {
	if observedAt.IsZero() {
		observedAt = time.Now()
	}
	m.history = appendCrackTopHistory(m.history, crackTopHistorySample{At: observedAt})
	if !journal {
		return
	}
	message := "live snapshot refresh unavailable"
	if m.snapshot != nil {
		message += " • retaining last snapshot"
	}
	m.appendJournal(crackTopJournalEntry{at: observedAt, level: "warning", label: "DATA", message: message})
}

func (m *crackTopModel) recordJobAction(message crackTopJobActionCompletedMsg) {
	observedAt := time.Now()
	if message.updatedAt > 0 {
		observedAt = time.Unix(message.updatedAt, 0)
	}
	// The lifecycle result is authoritative for job state, but its compact
	// response does not contain the post-action queue and device telemetry.
	// Leave an explicit gap until the next complete CrackTop snapshot arrives.
	m.history = appendCrackTopHistory(m.history, crackTopHistorySample{At: observedAt})
	status := "DELETED"
	level := "warning"
	if message.action != crackTopJobActionDelete {
		status = message.status.String()
		level = crackTopJobJournalLevel(status)
	}
	m.appendJournal(crackTopJournalEntry{
		at:      observedAt,
		level:   level,
		label:   "JOB",
		message: fmt.Sprintf("%s %s", crackTopShortID(message.jobID), status),
	})
}

func (m *crackTopModel) appendJournal(entry crackTopJournalEntry) {
	entry.label = crackTopSafeCell(entry.label)
	entry.message = crackTopSafeCell(entry.message)
	if len(m.journal) >= crackTopJournalLimit {
		start := len(m.journal) - (crackTopJournalLimit - 1)
		retained := copy(m.journal, m.journal[start:])
		m.journal = m.journal[:retained]
	}
	m.journal = append(m.journal, entry)
}

func crackTopJobJournalLevel(status string) string {
	switch strings.ToUpper(status) {
	case "COMPLETED":
		return "success"
	case "FAILED", "CANCELLED":
		return "danger"
	default:
		return "warning"
	}
}

func crackTopCompletenessJournalLevel(dashboard crackTopDashboard) string {
	if dashboard.ReportingWorkers == dashboard.ExpectedWorkers {
		return "success"
	}
	return "warning"
}

func (m *crackTopModel) renderOverview(width, height int) string {
	remaining := height - crackTopOverviewTopBarHeight - crackTopOverviewSummaryHeight -
		crackTopOverviewMetersHeight - crackTopOverviewJournalHeight - crackTopFooterHeight
	chartHeight := max(crackTopOverviewChartMinHeight, min(crackTopOverviewChartMaxHeight, (remaining+1)*2/5))
	detailsHeight := remaining - chartHeight
	if detailsHeight < 7 {
		detailsHeight = 7
		chartHeight = remaining - detailsHeight
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderOverviewTopBar(width),
		m.renderOverviewSummary(width, crackTopOverviewSummaryHeight),
		m.renderOverviewMeters(width, crackTopOverviewMetersHeight),
		m.renderOverviewCharts(width, chartHeight),
		m.renderOverviewDetails(width, detailsHeight),
		m.renderOverviewJournal(width, crackTopOverviewJournalHeight),
		m.renderOverviewFooter(width),
	)
}

func (m *crackTopModel) renderOverviewTopBar(width int) string {
	activeTab := lipgloss.NewStyle().
		Bold(true).
		Foreground(clienttheme.DefaultMod(50)).
		Background(clienttheme.Primary()).
		Render(" OVERVIEW ")
	focus := "JOBS FOCUS"
	if m.focus == crackTopFocusWorkers {
		focus = "WORKERS FOCUS"
	}
	left := m.styles.title.Render("CRACK TOP") + "  " + activeTab + " " + m.styles.primary.Render(focus)
	right := m.styles.muted.Render("tab focus   p/u/c/d job   f filter   r refresh   q quit")
	return crackTopPadLine(crackTopJoinEdges(left, right, width), width)
}

func (m *crackTopModel) renderOverviewFooter(width int) string {
	if m.toast != "" || m.runningJobAction.jobID != "" {
		return m.renderFooter(width)
	}
	if m.lastError != "" {
		return m.renderFooter(width)
	}
	focus := "JOBS"
	if m.focus == crackTopFocusWorkers {
		focus = "CRACKSTATIONS"
	}
	message := fmt.Sprintf("%s focus  •  ↑/k ↓/j select  •  tab switch focus", focus)
	return crackTopPadLine(m.styles.muted.Render(message), width)
}

func (m *crackTopModel) renderOverviewSummary(width, height int) string {
	widths := crackTopSplitWidths(width, 3, 1)
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.renderOverviewStateCard(widths[0], height),
		" ",
		m.renderOverviewThroughputCard(widths[1], height),
		" ",
		m.renderOverviewDiagnosticsCard(widths[2], height),
	)
}

func (m *crackTopModel) renderOverviewStateCard(width, height int) string {
	innerWidth := max(1, width-4)
	if m.snapshot == nil {
		state := m.styles.warning.Render("WAITING") + "   " + m.styles.muted.Render("awaiting first live snapshot")
		border := clienttheme.Primary()
		if m.width < 120 {
			state = m.styles.warning.Render("WAITING") + "   " + m.styles.muted.Render("first live snapshot")
		}
		if m.lastError != "" {
			state = m.styles.danger.Render("NO SNAPSHOT") + "   " + m.styles.warning.Render("refresh failed")
			border = clienttheme.Warning()
		}
		lines := []string{
			m.styles.warning.Render("! CLUSTER / STATE"),
			state,
			"ACTIVE —   JOBS —",
			"WORKERS —   DEVICES —",
		}
		return crackTopOverviewPanel(width, height, border, crackTopFitLines(lines, innerWidth))
	}

	state := "IDLE"
	stateStyle := m.styles.success
	if m.dashboard.ActiveJobs > 0 {
		state = "ACTIVE"
		stateStyle = m.styles.warning
	} else if crackTopPausedJobCount(m.dashboard) > 0 {
		state = "PAUSED"
		stateStyle = m.styles.secondary
	}
	health := m.styles.success.Render("DATA OK")
	if m.lastError != "" || crackTopLiveDataIssueCount(m.dashboard) > 0 {
		health = m.styles.warning.Render("DATA GAPS")
	}
	freshness := crackTopFreshness(m.snapshot.RefreshedAt)
	if m.lastError != "" {
		freshness = "LAST " + strings.TrimPrefix(freshness, "LIVE ")
	} else if m.authoritativeSnapshotPending() {
		health = m.styles.warning.Render("SYNCING")
		freshness = "TASK DATA LAST"
	}
	jobLine := fmt.Sprintf("ACTIVE %d   JOBS %d loaded", m.dashboard.ActiveJobs, len(m.dashboard.Jobs))
	if m.dashboard.ActiveJobs == 0 && crackTopPausedJobCount(m.dashboard) > 0 {
		jobLine = fmt.Sprintf("PAUSED %d   JOBS %d loaded", crackTopPausedJobCount(m.dashboard), len(m.dashboard.Jobs))
	}
	workerLine := fmt.Sprintf("WORKERS %d/%d online   DEVICES %d live", m.dashboard.OnlineWorkers, crackTopWorkerTotal(m.dashboard), crackTopDeviceCount(m.dashboard))
	if m.width < 120 {
		workerLine = fmt.Sprintf("%d/%d workers   %d devices", m.dashboard.OnlineWorkers, crackTopWorkerTotal(m.dashboard), crackTopDeviceCount(m.dashboard))
	}
	lines := []string{
		m.styles.warning.Render("! CLUSTER / STATE"),
		stateStyle.Render(state) + "   " + health + "   " + m.styles.muted.Render(freshness),
		jobLine,
		workerLine,
	}
	return crackTopOverviewPanel(width, height, clienttheme.Primary(), crackTopFitLines(lines, innerWidth))
}

func (m *crackTopModel) renderOverviewThroughputCard(width, height int) string {
	innerWidth := max(1, width-4)
	history := m.overviewHistory()
	average, peak, known, complete := crackTopRateStats(history)
	if m.snapshot == nil {
		lines := []string{
			m.styles.secondary.Render("⚡ THROUGHPUT"),
			m.styles.heading.Render("CLUSTER ") + m.styles.muted.Render("— awaiting telemetry"),
			"AVG —   PEAK —",
			"RECOVERED — loaded   VIEW —",
		}
		return crackTopOverviewPanel(width, height, clienttheme.Secondary(), crackTopFitLines(lines, innerWidth))
	}
	rate := "—"
	if m.dashboard.ClusterRateKnown {
		rate = humanizeHashRate(m.dashboard.ClusterRate)
		if !m.dashboard.ClusterComplete {
			rate = "≥ " + rate
		}
		if m.lastError != "" || m.authoritativeSnapshotPending() {
			rate = "last " + rate
		}
	}
	averageText, peakText := "—", "—"
	if known {
		averageText = humanizeHashRate(average)
		peakText = humanizeHashRate(peak)
		if !complete {
			averageText = "≥ " + averageText
			peakText = "≥ " + peakText
		}
	}
	recoveredDelta, recoveredDeltaKnown := crackTopRecoveredDelta(history)
	averageLine := fmt.Sprintf("SAMPLE AVG %s   PEAK %s", averageText, peakText)
	recoveredLine := fmt.Sprintf("RECOVERED %d loaded   VIEW —", m.dashboard.Recovered)
	if recoveredDeltaKnown {
		recoveredLine = fmt.Sprintf("RECOVERED %d loaded   VIEW +%d", m.dashboard.Recovered, recoveredDelta)
	}
	if m.lastError != "" || m.authoritativeSnapshotPending() {
		recoveredLine = fmt.Sprintf("RECOVERED %d last snapshot   VIEW —", m.dashboard.Recovered)
	}
	if m.width < 120 {
		averageLine = fmt.Sprintf("SAVG %s   PK %s", averageText, peakText)
		recoveredLine = fmt.Sprintf("%d recovered   view —", m.dashboard.Recovered)
		if recoveredDeltaKnown {
			recoveredLine = fmt.Sprintf("%d recovered   +%d view", m.dashboard.Recovered, recoveredDelta)
		}
		if m.lastError != "" || m.authoritativeSnapshotPending() {
			recoveredLine = fmt.Sprintf("%d recovered last   view —", m.dashboard.Recovered)
		}
	}
	lines := []string{
		m.styles.secondary.Render("⚡ THROUGHPUT"),
		m.styles.heading.Render("CLUSTER ") + m.styles.secondary.Render(rate),
		averageLine,
		recoveredLine,
	}
	return crackTopOverviewPanel(width, height, clienttheme.Secondary(), crackTopFitLines(lines, innerWidth))
}

func (m *crackTopModel) renderOverviewDiagnosticsCard(width, height int) string {
	innerWidth := max(1, width-4)
	if m.snapshot == nil {
		status := m.styles.warning.Render("WAIT") + "   awaiting live telemetry"
		border := clienttheme.Primary()
		if m.lastError != "" {
			status = m.styles.danger.Render("ERROR") + "   no live snapshot"
			border = clienttheme.Warning()
		}
		lines := []string{
			m.styles.success.Render("◆ DIAGNOSTICS"),
			status,
			"QUEUE — run   — leased   — waiting",
			"SOURCE " + crackTopOverviewSource(m),
		}
		return crackTopOverviewPanel(width, height, border, crackTopFitLines(lines, innerWidth))
	}
	issues := crackTopLiveDataIssueCount(m.dashboard)
	if m.lastError != "" {
		issues++
	}
	status := m.styles.success.Render("OK") + "   LIVE DATA COMPLETE"
	border := clienttheme.Success()
	syncing := m.authoritativeSnapshotPending()
	if syncing {
		status = m.styles.warning.Render("SYNC") + "   action confirmed • task data pending"
		border = clienttheme.Warning()
	} else if issues > 0 {
		status = m.styles.warning.Render("WARN") + fmt.Sprintf("   %d live data issue(s)", issues)
		border = clienttheme.Warning()
	}
	queueLine := fmt.Sprintf("QUEUE %d run   %d leased   %d waiting", m.dashboard.RunningTasks, m.dashboard.LeasedTasks, m.dashboard.QueuedTasks)
	queueStale := syncing || m.lastError != ""
	if queueStale {
		queueLine = fmt.Sprintf("QUEUE last %d run   %d leased   %d waiting", m.dashboard.RunningTasks, m.dashboard.LeasedTasks, m.dashboard.QueuedTasks)
	}
	if m.width < 120 {
		queueLine = fmt.Sprintf("%d run   %d leased   %d queued", m.dashboard.RunningTasks, m.dashboard.LeasedTasks, m.dashboard.QueuedTasks)
		if queueStale {
			queueLine = fmt.Sprintf("last %d run   %d lease   %d queue", m.dashboard.RunningTasks, m.dashboard.LeasedTasks, m.dashboard.QueuedTasks)
		}
	}
	lines := []string{
		m.styles.success.Render("◆ DIAGNOSTICS"),
		status,
		queueLine,
		"SOURCE " + crackTopOverviewSource(m),
	}
	return crackTopOverviewPanel(width, height, border, crackTopFitLines(lines, innerWidth))
}

//nolint:gocyclo // The three meters intentionally share one snapshot and freshness decision path.
func (m *crackTopModel) renderOverviewMeters(width, height int) string {
	widths := crackTopSplitWidths(width, 3, 1)
	history := m.overviewHistory()
	current := crackTopHistorySample{}
	if len(history) > 0 {
		current = history[len(history)-1]
	}
	syncing := m.authoritativeSnapshotPending()

	progressLabel := "awaiting keyspace"
	if current.ProgressKnown {
		progressLabel = fmt.Sprintf("%5.1f%%", current.Progress*100)
	} else if current.CountsKnown && current.ActiveJobs == 0 && m.snapshot != nil {
		progressLabel = "no active jobs"
	} else if m.lastError != "" && m.snapshot != nil {
		progressLabel = "current data unavailable"
	}
	if syncing {
		progressLabel = "last snapshot • syncing"
	}
	progressPanel := m.renderOverviewMeter(
		widths[0], height, "◇ GLOBAL KEYSPACE / PROGRESS", progressLabel,
		current.Progress, current.ProgressKnown, clienttheme.Secondary(),
	)

	workerKnown := current.CountsKnown && current.TotalWorkers > 0
	workerFraction := 0.0
	workerLabel := "awaiting workers"
	if workerKnown {
		workerFraction = float64(current.OnlineWorkers) / float64(current.TotalWorkers)
		workerLabel = fmt.Sprintf("%d/%d online", current.OnlineWorkers, current.TotalWorkers)
	} else if current.CountsKnown && m.snapshot != nil {
		workerLabel = "no observed workers"
	} else if m.lastError != "" && m.snapshot != nil {
		workerLabel = "current data unavailable"
	}
	if syncing {
		workerLabel = "last snapshot • syncing"
	}
	workerPanel := m.renderOverviewMeter(
		widths[1], height, "● OBSERVED WORKERS", workerLabel,
		workerFraction, workerKnown, clienttheme.Success(),
	)

	deviceLabel := "awaiting device telemetry"
	switch {
	case current.UtilizationKnown && current.TemperatureKnown:
		deviceLabel = fmt.Sprintf("%.0f%% avg worker max • %.0f°C max", current.Utilization, current.Temperature)
	case current.UtilizationKnown:
		deviceLabel = fmt.Sprintf("%.0f%% avg worker max", current.Utilization)
	case current.TemperatureKnown:
		deviceLabel = fmt.Sprintf("%.0f°C max • util unavailable", current.Temperature)
	case m.lastError != "" && m.snapshot != nil:
		deviceLabel = "current data unavailable"
	}
	if syncing {
		deviceLabel = "last snapshot • syncing"
	}
	devicePanel := m.renderOverviewMeter(
		widths[2], height, "◆ DEVICE LOAD", deviceLabel,
		current.Utilization/100, current.UtilizationKnown, clienttheme.Warning(),
	)

	return lipgloss.JoinHorizontal(lipgloss.Top, progressPanel, " ", workerPanel, " ", devicePanel)
}

func (m *crackTopModel) renderOverviewMeter(width, height int, title, label string, fraction float64, known bool, fill color.Color) string {
	innerWidth := max(1, width-4)
	minBarWidth := min(6, max(1, innerWidth/4))
	label = crackTopFitLine(label, max(1, innerWidth-minBarWidth-1))
	barWidth := max(1, innerWidth-ansi.StringWidth(label)-1)
	bar := m.styles.muted.Render(strings.Repeat("·", barWidth))
	if known {
		fraction = max(0, min(1, fraction))
		filled := int(math.Round(fraction * float64(barWidth)))
		bar = lipgloss.NewStyle().Foreground(fill).Render(strings.Repeat("█", filled)) +
			m.styles.muted.Render(strings.Repeat("░", barWidth-filled))
	}
	content := []string{
		m.styles.heading.Render(title),
		crackTopFitLine(label+" "+bar, innerWidth),
	}
	return crackTopOverviewPanel(width, height, fill, strings.Join(content, "\n"))
}

func (m *crackTopModel) renderOverviewCharts(width, height int) string {
	widths := crackTopSplitWidths(width, 3, 1)
	history := m.overviewHistory()
	ratePoints := make([]crackTopChartPoint, 0, len(history))
	progressPoints := make([]crackTopChartPoint, 0, len(history))
	utilizationPoints := make([]crackTopChartPoint, 0, len(history))
	for _, sample := range history {
		ratePoints = append(ratePoints, crackTopChartPoint{value: float64(sample.ClusterRate), known: sample.ClusterRateKnown, at: sample.At, lowerBound: !sample.ClusterComplete})
		progressPoints = append(progressPoints, crackTopChartPoint{value: sample.Progress * 100, known: sample.ProgressKnown, at: sample.At})
		utilizationPoints = append(utilizationPoints, crackTopChartPoint{value: sample.Utilization, known: sample.UtilizationKnown, at: sample.At})
	}
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.renderOverviewChart(widths[0], height, "rate", ratePoints, 0, formatCrackTopChartRate, clienttheme.Secondary()),
		" ",
		m.renderOverviewChart(widths[1], height, "progress", progressPoints, 100, formatCrackTopChartPercent, clienttheme.Primary()),
		" ",
		m.renderOverviewChart(widths[2], height, "worker max util", utilizationPoints, 100, formatCrackTopChartPercent, clienttheme.Success()),
	)
}

func (m *crackTopModel) renderOverviewChart(
	width, height int,
	title string,
	points []crackTopChartPoint,
	fixedMaximum float64,
	format func(float64) string,
	lineColor color.Color,
) string {
	innerWidth := max(1, width-4)
	innerHeight := max(1, height-2)
	points = crackTopVisibleChartPoints(points, innerWidth)
	latest, average, peak, latestKnown, historyKnown, latestLowerBound, historyLowerBound := crackTopChartStats(points)
	header := title + "  waiting for samples"
	if historyKnown {
		latestText := "—"
		if latestKnown {
			latestText = format(latest)
			if latestLowerBound {
				latestText = "≥ " + latestText
			}
		}
		averageText := format(average)
		peakText := format(peak)
		if historyLowerBound {
			averageText = "≥ " + averageText
			peakText = "≥ " + peakText
		}
		span := crackTopChartSpan(points)
		candidates := []string{
			fmt.Sprintf("%s  %s · savg %s · pk %s · %s", title, latestText, averageText, peakText, span),
			fmt.Sprintf("%s  %s · savg %s · %s", title, latestText, averageText, span),
			fmt.Sprintf("%s  %s · %s", title, latestText, span),
		}
		header = candidates[len(candidates)-1]
		for _, candidate := range candidates {
			if ansi.StringWidth(candidate) <= innerWidth {
				header = candidate
				break
			}
		}
	}
	plotHeight := max(1, innerHeight-1)
	plot := crackTopRenderPlot(points, innerWidth, plotHeight, fixedMaximum)
	content := m.styles.heading.Render(crackTopFitLine(header, innerWidth))
	if plotHeight > 0 {
		content += "\n" + lipgloss.NewStyle().Foreground(lineColor).Render(plot)
	}
	return crackTopOverviewPanel(width, height, lineColor, content)
}

func (m *crackTopModel) renderOverviewDetails(width, height int) string {
	const gap = 1
	jobsWidth := (width - gap) * 3 / 5
	workersWidth := width - gap - jobsWidth
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.renderJobsPane(jobsWidth, height),
		" ",
		m.renderWorkersPane(workersWidth, height),
	)
}

func (m *crackTopModel) renderOverviewJournal(width, height int) string {
	innerWidth := max(1, width-4)
	capacity := max(1, height-3)
	entries := m.journal
	if len(entries) == 0 {
		entry := crackTopJournalEntry{at: time.Now(), level: "muted", label: "WAITING", message: "waiting for the first live snapshot"}
		if m.snapshot != nil {
			entry = crackTopJournalEntry{at: m.snapshot.RefreshedAt, level: "success", label: "LIVE", message: fmt.Sprintf("snapshot loaded • %d active • %d/%d workers", m.dashboard.ActiveJobs, m.dashboard.OnlineWorkers, crackTopWorkerTotal(m.dashboard))}
		}
		entries = []crackTopJournalEntry{entry}
	}
	start := max(0, len(entries)-capacity)
	lines := []string{m.styles.heading.Render("RECENT JOURNAL")}
	for _, entry := range entries[start:] {
		labelStyle := m.styles.primary
		switch entry.level {
		case "success":
			labelStyle = m.styles.success
		case "warning":
			labelStyle = m.styles.warning
		case "danger", "error":
			labelStyle = m.styles.danger
		case "muted":
			labelStyle = m.styles.muted
		}
		stamp := "--:--:--"
		if !entry.at.IsZero() {
			stamp = entry.at.Local().Format("15:04:05")
		}
		line := m.styles.muted.Render(stamp) + "  " + labelStyle.Render(entry.label) + "  " + m.styles.normal.Render(entry.message)
		lines = append(lines, crackTopFitLine(line, innerWidth))
	}
	return crackTopOverviewPanel(width, height, clienttheme.DefaultMod(300), strings.Join(lines, "\n"))
}

func (m *crackTopModel) overviewHistory() []crackTopHistorySample {
	if len(m.history) > 0 {
		return m.history
	}
	if m.snapshot == nil {
		return nil
	}
	return []crackTopHistorySample{crackTopHistorySampleFromDashboard(m.dashboard, m.snapshot.RefreshedAt)}
}

func crackTopOverviewPanel(width, height int, border color.Color, content string) string {
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
		Border(lipgloss.NormalBorder()).
		BorderForeground(border).
		Render(strings.Join(lines, "\n"))
}

func crackTopSplitWidths(width, count, gap int) []int {
	if count <= 0 {
		return nil
	}
	available := max(count*4, width-gap*(count-1))
	result := make([]int, count)
	for index := range result {
		result[index] = available / count
		if index < available%count {
			result[index]++
		}
	}
	return result
}

func crackTopFitLines(lines []string, width int) string {
	for index := range lines {
		lines[index] = crackTopFitLine(lines[index], width)
	}
	return strings.Join(lines, "\n")
}

func crackTopWorkerTotal(dashboard crackTopDashboard) int {
	return dashboard.OnlineWorkers + dashboard.Unavailable
}

func crackTopTelemetryErrorCount(dashboard crackTopDashboard) int {
	total := 0
	for _, job := range dashboard.Jobs {
		total += job.TelemetryErrors
	}
	return total
}

func crackTopLiveDataIssueCount(dashboard crackTopDashboard) int {
	issues := crackTopTelemetryErrorCount(dashboard) + dashboard.Unavailable
	if dashboard.ExpectedWorkers > dashboard.ReportingWorkers {
		issues += dashboard.ExpectedWorkers - dashboard.ReportingWorkers
	} else if dashboard.ClusterRateKnown && !dashboard.ClusterComplete {
		issues++
	}
	return issues
}

func crackTopPausedJobCount(dashboard crackTopDashboard) int {
	total := 0
	for _, job := range dashboard.Jobs {
		if job.Status == clientpb.CrackJobStatus_PAUSED {
			total++
		}
	}
	return total
}

func crackTopDeviceCount(dashboard crackTopDashboard) int {
	total := 0
	for _, worker := range dashboard.Workers {
		if worker.TelemetryLive {
			total += worker.Devices
		}
	}
	return total
}

func crackTopFreshness(at time.Time) string {
	if at.IsZero() {
		return "time unknown"
	}
	age := time.Since(at)
	if age < 0 {
		age = 0
	}
	switch {
	case age < time.Second:
		return "LIVE now"
	case age < time.Minute:
		return fmt.Sprintf("LIVE %ds old", int(age.Seconds()))
	case age < time.Hour:
		return fmt.Sprintf("LIVE %dm old", int(age.Minutes()))
	case age < 24*time.Hour:
		return fmt.Sprintf("LIVE %dh old", int(age.Hours()))
	default:
		return fmt.Sprintf("LIVE %dd old", int(age.Hours()/24))
	}
}

func crackTopOverviewSource(m *crackTopModel) string {
	if m == nil {
		return "unavailable"
	}
	if m.eventStreamClosed {
		return fmt.Sprintf("poll only • %s", crackTopDurationLabel(m.pollInterval))
	}
	return fmt.Sprintf("events + %s poll", crackTopDurationLabel(m.pollInterval))
}

func crackTopDurationLabel(duration time.Duration) string {
	if duration <= 0 {
		return "manual"
	}
	if duration < time.Second {
		return fmt.Sprintf("%dms", duration.Milliseconds())
	}
	return duration.Round(time.Millisecond).String()
}

func crackTopRateStats(history []crackTopHistorySample) (uint64, uint64, bool, bool) {
	var peak uint64
	total := new(big.Int)
	value := new(big.Int)
	count := uint64(0)
	complete := true
	for _, sample := range history {
		if !sample.ClusterRateKnown {
			continue
		}
		complete = complete && sample.ClusterComplete
		peak = max(peak, sample.ClusterRate)
		total.Add(total, value.SetUint64(sample.ClusterRate))
		count++
	}
	if count == 0 {
		return 0, 0, false, false
	}
	average := new(big.Int).Quo(total, value.SetUint64(count)).Uint64()
	return average, peak, true, complete
}

func crackTopRecoveredDelta(history []crackTopHistorySample) (uint64, bool) {
	first := -1
	last := -1
	for index, sample := range history {
		if !sample.CountsKnown {
			continue
		}
		if first == -1 {
			first = index
		}
		last = index
	}
	if first == -1 || last == -1 || first == last || history[last].Recovered < history[first].Recovered {
		return 0, false
	}
	return history[last].Recovered - history[first].Recovered, true
}

func crackTopChartStats(points []crackTopChartPoint) (float64, float64, float64, bool, bool, bool, bool) {
	latest := 0.0
	latestKnown := false
	latestLowerBound := false
	if len(points) > 0 {
		point := points[len(points)-1]
		if point.known && !math.IsNaN(point.value) && !math.IsInf(point.value, 0) {
			latest = max(0, point.value)
			latestKnown = true
			latestLowerBound = point.lowerBound
		}
	}
	peak := 0.0
	total := 0.0
	count := 0
	historyLowerBound := false
	for _, point := range points {
		if !point.known || math.IsNaN(point.value) || math.IsInf(point.value, 0) {
			continue
		}
		value := max(0, point.value)
		historyLowerBound = historyLowerBound || point.lowerBound
		peak = max(peak, value)
		total += value
		count++
	}
	if count == 0 {
		return 0, 0, 0, false, false, false, false
	}
	return latest, total / float64(count), peak, latestKnown, true, latestLowerBound, historyLowerBound
}

func crackTopVisibleChartPoints(points []crackTopChartPoint, width int) []crackTopChartPoint {
	width = max(1, width)
	if len(points) <= width {
		return points
	}
	return points[len(points)-width:]
}

func crackTopChartSpan(points []crackTopChartPoint) string {
	if len(points) <= 1 {
		return fmt.Sprintf("%d sample", len(points))
	}
	var first, last time.Time
	for _, point := range points {
		if point.at.IsZero() {
			continue
		}
		if first.IsZero() {
			first = point.at
		}
		last = point.at
	}
	if first.IsZero() || !last.After(first) {
		return fmt.Sprintf("%d samples", len(points))
	}
	return fmt.Sprintf("%s/%d samples", crackTopDurationLabel(last.Sub(first)), len(points))
}

type crackTopPlotEdge uint8

const (
	crackTopPlotUp crackTopPlotEdge = 1 << iota
	crackTopPlotDown
	crackTopPlotLeft
	crackTopPlotRight
)

//nolint:gocyclo // Plot rasterization keeps scaling, gaps, and connected edge composition in one bounded pass.
func crackTopRenderPlot(points []crackTopChartPoint, width, height int, fixedMaximum float64) string {
	width = max(1, width)
	height = max(1, height)
	grid := make([][]rune, height)
	edges := make([][]crackTopPlotEdge, height)
	samples := make([][]bool, height)
	for row := range grid {
		grid[row] = []rune(strings.Repeat(" ", width))
		edges[row] = make([]crackTopPlotEdge, width)
		samples[row] = make([]bool, width)
	}
	if height > 2 {
		for column := range grid[height/2] {
			grid[height/2][column] = '·'
		}
	}
	for column := range grid[height-1] {
		grid[height-1][column] = '·'
	}
	if len(points) == 0 {
		return crackTopRuneGrid(grid)
	}
	if len(points) > width {
		points = points[len(points)-width:]
	}
	maximum := fixedMaximum
	if maximum <= 0 {
		for _, point := range points {
			if point.known && !math.IsNaN(point.value) && !math.IsInf(point.value, 0) {
				maximum = max(maximum, point.value)
			}
		}
	}
	if maximum <= 0 {
		maximum = 1
	}
	offset := width - len(points)
	previousX, previousY := 0, 0
	previousKnown := false
	for index, point := range points {
		if !point.known || math.IsNaN(point.value) || math.IsInf(point.value, 0) {
			previousKnown = false
			continue
		}
		x := offset + index
		fraction := max(0, min(1, point.value/maximum))
		y := height - 1 - int(math.Round(fraction*float64(height-1)))
		samples[y][x] = true
		if previousKnown && x == previousX+1 {
			edges[previousY][previousX] |= crackTopPlotRight
			edges[previousY][x] |= crackTopPlotLeft
			if y < previousY {
				edges[previousY][x] |= crackTopPlotUp
				for row := previousY - 1; row > y; row-- {
					edges[row][x] |= crackTopPlotUp | crackTopPlotDown
				}
				edges[y][x] |= crackTopPlotDown
			} else if y > previousY {
				edges[previousY][x] |= crackTopPlotDown
				for row := previousY + 1; row < y; row++ {
					edges[row][x] |= crackTopPlotUp | crackTopPlotDown
				}
				edges[y][x] |= crackTopPlotUp
			}
		}
		previousX, previousY, previousKnown = x, y, true
	}
	for row := range grid {
		for column := range grid[row] {
			if edges[row][column] != 0 {
				grid[row][column] = crackTopPlotEdgeGlyph(edges[row][column])
			} else if samples[row][column] {
				grid[row][column] = '●'
			}
		}
	}
	return crackTopRuneGrid(grid)
}

//nolint:gocyclo // The exhaustive switch names every nonzero four-direction edge combination.
func crackTopPlotEdgeGlyph(edge crackTopPlotEdge) rune {
	switch edge {
	case crackTopPlotUp:
		return '╵'
	case crackTopPlotDown:
		return '╷'
	case crackTopPlotLeft:
		return '╴'
	case crackTopPlotRight:
		return '╶'
	case crackTopPlotUp | crackTopPlotDown:
		return '│'
	case crackTopPlotLeft | crackTopPlotRight:
		return '─'
	case crackTopPlotUp | crackTopPlotRight:
		return '└'
	case crackTopPlotUp | crackTopPlotLeft:
		return '┘'
	case crackTopPlotDown | crackTopPlotRight:
		return '┌'
	case crackTopPlotDown | crackTopPlotLeft:
		return '┐'
	case crackTopPlotUp | crackTopPlotDown | crackTopPlotRight:
		return '├'
	case crackTopPlotUp | crackTopPlotDown | crackTopPlotLeft:
		return '┤'
	case crackTopPlotDown | crackTopPlotLeft | crackTopPlotRight:
		return '┬'
	case crackTopPlotUp | crackTopPlotLeft | crackTopPlotRight:
		return '┴'
	case crackTopPlotUp | crackTopPlotDown | crackTopPlotLeft | crackTopPlotRight:
		return '┼'
	default:
		return '●'
	}
}

func crackTopRuneGrid(grid [][]rune) string {
	lines := make([]string, len(grid))
	for index := range grid {
		lines[index] = string(grid[index])
	}
	return strings.Join(lines, "\n")
}

func formatCrackTopChartRate(value float64) string {
	if value <= 0 {
		return humanizeHashRate(0)
	}
	if value >= float64(math.MaxUint64) {
		return humanizeHashRate(math.MaxUint64)
	}
	return humanizeHashRate(uint64(math.Round(value)))
}

func formatCrackTopChartPercent(value float64) string {
	return fmt.Sprintf("%.1f%%", value)
}
