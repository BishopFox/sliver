package crack

import (
	"errors"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/charmbracelet/x/ansi"
)

const crackTopMaxProgressTextSize = 148

type crackTopDashboard struct {
	Jobs             []crackTopJobRow
	Workers          []crackTopWorkerRow
	ActiveJobs       int
	OnlineWorkers    int
	Unavailable      int
	Recovered        uint64
	ClusterRate      uint64
	ClusterRateKnown bool
	ClusterComplete  bool
	ExpectedWorkers  int
	ReportingWorkers int
	Progress         float64
	ProgressKnown    bool
	ProgressJobs     int
}

type crackTopJobRow struct {
	ID               string
	Status           clientpb.CrackJobStatus
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Progress         float64
	ProgressKnown    bool
	ProgressComplete bool
	Rate             uint64
	RateKnown        bool
	Workers          int
	Reporting        int
	Results          uint64
	Mode             string
	TelemetryErrors  int
}

type crackTopWorkerRow struct {
	ID            string
	Name          string
	State         string
	Online        bool
	Syncing       bool
	JobID         string
	Rate          uint64
	RateKnown     bool
	Devices       int
	Temperature   float64
	HasTemp       bool
	Utilization   float64
	HasUtil       bool
	LastSeen      time.Time
	TelemetryLive bool
}

type crackTopTaskSample struct {
	jobID       string
	task        *clientpb.CrackTask
	status      *crackStatusView
	statusErr   error
	rate        uint64
	rateKnown   bool
	devices     int
	temperature float64
	hasTemp     bool
	utilization float64
	hasUtil     bool
	lastSeen    time.Time
}

type crackTopProgressAccumulator struct {
	completed *big.Rat
	total     *big.Rat
}

func newCrackTopProgressAccumulator() *crackTopProgressAccumulator {
	return &crackTopProgressAccumulator{completed: new(big.Rat), total: new(big.Rat)}
}

func (a *crackTopProgressAccumulator) add(fraction, weight *big.Rat) {
	if a == nil || fraction == nil || weight == nil || weight.Sign() <= 0 {
		return
	}
	clamped := new(big.Rat).Set(fraction)
	if clamped.Sign() < 0 {
		clamped.SetInt64(0)
	}
	if clamped.Cmp(big.NewRat(1, 1)) > 0 {
		clamped.SetInt64(1)
	}
	a.completed.Add(a.completed, new(big.Rat).Mul(clamped, weight))
	a.total.Add(a.total, weight)
}

func (a *crackTopProgressAccumulator) value() (float64, bool) {
	if a == nil || a.total.Sign() <= 0 {
		return 0, false
	}
	fraction, _ := new(big.Rat).Quo(a.completed, a.total).Float64()
	if math.IsNaN(fraction) || math.IsInf(fraction, 0) {
		return 0, false
	}
	return math.Max(0, math.Min(1, fraction)), true
}

//nolint:gocyclo // Jobs, task telemetry, and connected workers are normalized in one immutable snapshot pass.
func buildCrackTopDashboard(snapshot *crackTopSnapshot) crackTopDashboard {
	dashboard := crackTopDashboard{}
	if snapshot == nil {
		return dashboard
	}

	stations := make(map[string]*clientpb.Crackstation, len(snapshot.Stations))
	for _, station := range snapshot.Stations {
		if station == nil {
			continue
		}
		hostID := crackTopStationID(station)
		if hostID == "" {
			continue
		}
		if _, exists := stations[hostID]; !exists {
			stations[hostID] = station
		}
	}
	dashboard.OnlineWorkers = len(stations)

	jobs := make([]*clientpb.CrackJob, 0, len(snapshot.Jobs))
	for _, job := range snapshot.Jobs {
		if job != nil && strings.TrimSpace(job.GetID()) != "" {
			jobs = append(jobs, job)
		}
	}
	sort.SliceStable(jobs, func(i, j int) bool {
		left := crackJobSortTime(jobs[i])
		right := crackJobSortTime(jobs[j])
		if left.Equal(right) {
			return jobs[i].GetID() < jobs[j].GetID()
		}
		return left.After(right)
	})

	globalProgress := newCrackTopProgressAccumulator()
	workerSamples := map[string]crackTopTaskSample{}
	for _, job := range jobs {
		row, samples, progress := crackTopJobTelemetry(job, stations, snapshot.RefreshedAt, snapshot.taskTelemetry)
		dashboard.Jobs = append(dashboard.Jobs, row)
		dashboard.Recovered = saturatingUint64Add(dashboard.Recovered, row.Results)
		if job.GetStatus() == clientpb.CrackJobStatus_IN_PROGRESS {
			dashboard.ActiveJobs++
			if row.ProgressComplete {
				dashboard.ProgressJobs++
			}
			if progress != nil {
				globalProgress.completed.Add(globalProgress.completed, progress.completed)
				globalProgress.total.Add(globalProgress.total, progress.total)
			}
			for hostID, sample := range samples {
				if hostID == "" {
					continue
				}
				if current, ok := workerSamples[hostID]; !ok || crackTopSamplePreferred(sample, current, stations[hostID], snapshot.RefreshedAt) {
					workerSamples[hostID] = sample
				}
			}
		}
	}

	dashboard.Progress, dashboard.ProgressKnown = globalProgress.value()
	dashboard.ProgressKnown = dashboard.ProgressKnown && dashboard.ProgressJobs == dashboard.ActiveJobs
	if dashboard.ActiveJobs == 0 {
		dashboard.Progress = 0
		dashboard.ProgressKnown = false
	}

	workerIDs := make(map[string]struct{}, len(stations)+len(workerSamples))
	for hostID := range stations {
		workerIDs[hostID] = struct{}{}
	}
	for hostID := range workerSamples {
		workerIDs[hostID] = struct{}{}
	}
	for hostID := range workerIDs {
		station := stations[hostID]
		sample, hasSample := workerSamples[hostID]
		worker := crackTopWorkerRow{ID: hostID, Online: station != nil}
		if station != nil {
			worker.Name = crackTopSafeCell(station.GetName())
			if worker.Name == "" && station.GetStatus() != nil {
				worker.Name = crackTopSafeCell(station.GetStatus().GetName())
			}
			if status := station.GetStatus(); status != nil {
				worker.State = status.GetState().String()
				worker.Syncing = status.GetIsSyncing()
				worker.JobID = strings.TrimSpace(status.GetCurrentCrackJobID())
			}
		}
		if worker.Name == "" {
			worker.Name = crackTopShortID(hostID)
		}
		if worker.State == "" {
			if worker.Online {
				worker.State = "ONLINE"
			} else {
				worker.State = "OFFLINE"
			}
		}
		if crackTopStationExpectsTelemetry(station) {
			dashboard.ExpectedWorkers++
		}
		if hasSample {
			if worker.JobID == "" && !worker.Online {
				worker.JobID = sample.jobID
			}
			worker.LastSeen = sample.lastSeen
			if crackTopTaskSampleIsLive(sample, station, snapshot.RefreshedAt) {
				worker.TelemetryLive = true
				worker.Rate = sample.rate
				worker.RateKnown = sample.rateKnown
				worker.Devices = sample.devices
				worker.Temperature = sample.temperature
				worker.HasTemp = sample.hasTemp
				worker.Utilization = sample.utilization
				worker.HasUtil = sample.hasUtil
			}
		}
		if !worker.Online {
			dashboard.Unavailable++
		}
		if worker.RateKnown {
			dashboard.ClusterRate = saturatingUint64Add(dashboard.ClusterRate, worker.Rate)
			dashboard.ClusterRateKnown = true
			dashboard.ReportingWorkers++
		}
		dashboard.Workers = append(dashboard.Workers, worker)
	}
	dashboard.ClusterComplete = dashboard.ReportingWorkers == dashboard.ExpectedWorkers
	dashboard.ClusterRateKnown = dashboard.ReportingWorkers > 0 || dashboard.ClusterComplete

	sort.SliceStable(dashboard.Workers, func(i, j int) bool {
		left, right := dashboard.Workers[i], dashboard.Workers[j]
		if left.Online != right.Online {
			return left.Online
		}
		if left.RateKnown != right.RateKnown {
			return left.RateKnown
		}
		if left.Rate != right.Rate {
			return left.Rate > right.Rate
		}
		leftName := strings.ToLower(left.Name)
		rightName := strings.ToLower(right.Name)
		if leftName != rightName {
			return leftName < rightName
		}
		return left.ID < right.ID
	})
	return dashboard
}

//nolint:gocyclo // Job telemetry combines task state, liveness, progress, and device metrics in one aggregation pass.
func crackTopJobTelemetry(
	job *clientpb.CrackJob,
	stations map[string]*clientpb.Crackstation,
	now time.Time,
	taskTelemetry map[string]crackTopTaskTelemetry,
) (crackTopJobRow, map[string]crackTopTaskSample, *crackTopProgressAccumulator) {
	row := crackTopJobRow{
		ID:        strings.TrimSpace(job.GetID()),
		Status:    job.GetStatus(),
		CreatedAt: crackTopParseTime(job.GetCreatedAt()),
		UpdatedAt: crackTopParseUnix(job.GetUpdatedAt()),
		Results:   job.GetResultCount(),
		Mode:      crackTopJobMode(job),
	}
	progress := newCrackTopProgressAccumulator()
	samples := map[string]crackTopTaskSample{}
	workers := map[string]struct{}{}
	workTasks := 0
	knownProgressTasks := 0
	allProgressWeightsKnown := true

	for _, task := range crackJobTasks(job) {
		if !crackTopCrackTask(task, taskTelemetry) {
			continue
		}
		workTasks++
		hostID := strings.TrimSpace(task.GetHostUUID())
		if hostID != "" {
			workers[hostID] = struct{}{}
		}
		status, statusErr := crackTaskStatus(task)
		progressExact := task.GetShardSkip() == 0
		if metadata, ok := taskTelemetry[strings.TrimSpace(task.GetID())]; ok {
			status = metadata.status
			statusErr = nil
			progressExact = metadata.progressExact || task.GetShardSkip() == 0
			if metadata.parseErr {
				statusErr = errors.New("crack status telemetry could not be parsed")
			}
		}
		if statusErr != nil {
			row.TelemetryErrors++
		}
		fraction, weight, known := crackTopTaskProgress(task, status, progressExact)
		if known {
			progress.add(fraction, weight)
			knownProgressTasks++
		} else if weight := crackTopAssignedShardWeight(task); weight != nil {
			// Unknown work may already be complete, so zero is only a lower
			// bound. Including its weight prevents known small shards from
			// overstating the job's completion percentage.
			progress.add(new(big.Rat), weight)
		} else {
			allProgressWeightsKnown = false
		}
		sample := crackTopTaskSample{
			jobID:     row.ID,
			task:      task,
			status:    status,
			statusErr: statusErr,
			lastSeen:  crackTopTaskTime(task),
		}
		if status != nil {
			sample.rate, sample.rateKnown = crackTopStatusRate(status)
			sample.devices = len(status.Devices)
			sample.temperature, sample.hasTemp = crackTopStatusMaximum(status, func(device crackDeviceView) string { return device.Temperature })
			sample.utilization, sample.hasUtil = crackTopStatusMaximum(status, func(device crackDeviceView) string { return device.Utilization })
		}
		if hostID != "" {
			if current, ok := samples[hostID]; !ok || crackTopSamplePreferred(sample, current, stations[hostID], now) {
				samples[hostID] = sample
			}
		}
	}

	row.Workers = len(workers)
	row.Progress, row.ProgressKnown = progress.value()
	row.ProgressKnown = row.ProgressKnown && workTasks > 0 && allProgressWeightsKnown
	row.ProgressComplete = row.ProgressKnown && knownProgressTasks == workTasks
	if job.GetStatus() == clientpb.CrackJobStatus_COMPLETED {
		row.Progress = 1
		row.ProgressKnown = true
		row.ProgressComplete = true
	}
	if job.GetStatus() == clientpb.CrackJobStatus_IN_PROGRESS {
		for hostID, sample := range samples {
			if crackTopTaskSampleIsLive(sample, stations[hostID], now) && sample.rateKnown {
				row.Rate = saturatingUint64Add(row.Rate, sample.rate)
				row.RateKnown = true
				row.Reporting++
			}
		}
	}
	return row, samples, progress
}

func crackTopTaskProgress(task *clientpb.CrackTask, status *crackStatusView, progressExact bool) (*big.Rat, *big.Rat, bool) {
	if task == nil {
		return nil, nil, false
	}
	weight := crackTopAssignedShardWeight(task)
	if weight == nil {
		weight = new(big.Rat)
	}
	state := task.GetState()
	if state == clientpb.CrackTaskState_CRACK_TASK_COMPLETED {
		if weight.Sign() <= 0 && status != nil {
			_, total, ok := crackTopProgressPair(status)
			if ok {
				weight.Set(total)
			}
		}
		if weight.Sign() > 0 {
			return big.NewRat(1, 1), weight, true
		}
	}

	current, total, ok := crackTopProgressPair(status)
	if ok && (task.GetShardSkip() == 0 || progressExact) {
		fraction := new(big.Rat).Quo(current, total)
		if weight.Sign() <= 0 {
			weight.Set(total)
		}
		return fraction, weight, weight.Sign() > 0
	}
	if weight.Sign() > 0 && (state == clientpb.CrackTaskState_CRACK_TASK_QUEUED || state == clientpb.CrackTaskState_CRACK_TASK_LEASED) {
		return new(big.Rat), weight, true
	}
	return nil, nil, false
}

func crackTopAssignedShardWeight(task *clientpb.CrackTask) *big.Rat {
	if task == nil || task.GetShardLimit() == 0 {
		return nil
	}
	return new(big.Rat).SetInt(new(big.Int).SetUint64(task.GetShardLimit()))
}

func crackTopProgressPair(status *crackStatusView) (*big.Rat, *big.Rat, bool) {
	if status == nil {
		return nil, nil, false
	}
	current, currentOK := crackTopRat(status.ProgressCurrent)
	total, totalOK := crackTopRat(status.ProgressTotal)
	if !currentOK || !totalOK || total.Sign() <= 0 {
		return nil, nil, false
	}
	return current, total, true
}

func crackTopRat(value string) (*big.Rat, bool) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > crackTopMaxProgressTextSize {
		return nil, false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return nil, false
		}
	}
	result := new(big.Rat)
	if _, ok := result.SetString(value); !ok {
		return nil, false
	}
	return result, true
}

func crackTopStatusRate(status *crackStatusView) (uint64, bool) {
	if status == nil {
		return 0, false
	}
	var total uint64
	found := false
	for _, device := range status.Devices {
		rate, err := strconv.ParseUint(strings.TrimSpace(device.Speed), 10, 64)
		if err != nil {
			continue
		}
		found = true
		total = saturatingUint64Add(total, rate)
	}
	return total, found
}

func crackTopStatusMaximum(status *crackStatusView, value func(crackDeviceView) string) (float64, bool) {
	if status == nil || value == nil {
		return 0, false
	}
	maximum := 0.0
	found := false
	for _, device := range status.Devices {
		number, err := strconv.ParseFloat(strings.TrimSpace(value(device)), 64)
		if err != nil || number < 0 || math.IsNaN(number) || math.IsInf(number, 0) {
			continue
		}
		if !found || number > maximum {
			maximum = number
			found = true
		}
	}
	return maximum, found
}

func crackTopTaskSampleIsLive(sample crackTopTaskSample, station *clientpb.Crackstation, now time.Time) bool {
	if station == nil || sample.task == nil || sample.statusErr != nil || sample.task.GetState() != clientpb.CrackTaskState_CRACK_TASK_RUNNING {
		return false
	}
	leaseExpiresAt := sample.task.GetLeaseExpiresAt()
	if now.IsZero() || leaseExpiresAt <= now.Unix() {
		return false
	}
	status := station.GetStatus()
	if status == nil {
		return false
	}
	if status.GetIsSyncing() || status.GetState() != clientpb.States_CRACKING {
		return false
	}
	currentJobID := strings.TrimSpace(status.GetCurrentCrackJobID())
	return currentJobID != "" && currentJobID == sample.jobID
}

func crackTopStationExpectsTelemetry(station *clientpb.Crackstation) bool {
	if station == nil {
		return false
	}
	status := station.GetStatus()
	if status == nil {
		// The station is connected, but without status we cannot prove it is
		// idle. Keep the cluster total unknown instead of reporting a false zero.
		return true
	}
	if status.GetIsSyncing() {
		return false
	}
	return status.GetState() == clientpb.States_CRACKING
}

func crackTopSamplePreferred(left, right crackTopTaskSample, station *clientpb.Crackstation, now time.Time) bool {
	leftLive := crackTopTaskSampleIsLive(left, station, now)
	rightLive := crackTopTaskSampleIsLive(right, station, now)
	if leftLive != rightLive {
		return leftLive
	}
	return crackTopSampleNewer(left, right)
}

func crackTopSampleNewer(left, right crackTopTaskSample) bool {
	if !left.lastSeen.Equal(right.lastSeen) {
		return left.lastSeen.After(right.lastSeen)
	}
	leftRunning := left.task != nil && left.task.GetState() == clientpb.CrackTaskState_CRACK_TASK_RUNNING
	rightRunning := right.task != nil && right.task.GetState() == clientpb.CrackTaskState_CRACK_TASK_RUNNING
	if leftRunning != rightRunning {
		return leftRunning
	}
	if left.task == nil || right.task == nil {
		return left.task != nil
	}
	return left.task.GetAttempt() > right.task.GetAttempt()
}

func crackTopCrackTask(task *clientpb.CrackTask, taskTelemetry map[string]crackTopTaskTelemetry) bool {
	if task == nil {
		return false
	}
	switch task.GetKind() {
	case clientpb.CrackTaskKind_CRACK_TASK_CRACK:
		return true
	case clientpb.CrackTaskKind_CRACK_TASK_UNSPECIFIED:
		if task.GetShardLimit() != 0 || len(task.GetLatestStatusJSON()) != 0 {
			return true
		}
		metadata, ok := taskTelemetry[strings.TrimSpace(task.GetID())]
		return ok && (metadata.status != nil || metadata.parseErr)
	default:
		return false
	}
}

func crackTopStationID(station *clientpb.Crackstation) string {
	if station == nil {
		return ""
	}
	if hostID := strings.TrimSpace(station.GetHostUUID()); hostID != "" {
		return hostID
	}
	if status := station.GetStatus(); status != nil {
		if hostID := strings.TrimSpace(status.GetHostUUID()); hostID != "" {
			return hostID
		}
	}
	return strings.TrimSpace(station.GetID())
}

func crackTopTaskTime(task *clientpb.CrackTask) time.Time {
	if task == nil {
		return time.Time{}
	}
	for _, value := range []int64{task.GetLastHeartbeatAt(), task.GetUpdatedAt(), task.GetStartedAt(), task.GetCreatedAt()} {
		if value > 0 {
			return time.Unix(value, 0)
		}
	}
	return time.Time{}
}

func crackTopParseTime(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339, strings.TrimSpace(value))
	return parsed
}

func crackTopParseUnix(value int64) time.Time {
	if value <= 0 {
		return time.Time{}
	}
	return time.Unix(value, 0)
}

func crackTopJobMode(job *clientpb.CrackJob) string {
	if job == nil || job.GetCommand() == nil {
		return ""
	}
	mode := int32(job.GetCommand().GetHashType())
	if job.GetCommand().HashMode != nil {
		mode = int32(job.GetCommand().GetHashMode())
	}
	if name, ok := hashcatHashTypeName(mode); ok {
		return name
	}
	return "mode " + strconv.FormatInt(int64(mode), 10)
}

func crackTopShortID(value string) string {
	value = crackTopSafeCell(value)
	if value == "" {
		return "unknown"
	}
	if index := strings.IndexByte(value, '-'); index > 0 {
		value = value[:index]
	}
	runes := []rune(value)
	if len(runes) > 8 {
		value = string(runes[:8])
	}
	return value
}

func crackTopSafeCell(value string) string {
	value = ansi.Strip(value)
	value = strings.Map(func(r rune) rune {
		if !unicode.IsPrint(r) {
			return -1
		}
		if unicode.IsSpace(r) && r != ' ' {
			return ' '
		}
		return r
	}, value)
	value = strings.TrimSpace(value)
	const maximumRunes = 96
	runes := []rune(value)
	if len(runes) > maximumRunes {
		value = string(runes[:maximumRunes-1]) + "…"
	}
	return value
}

func saturatingUint64Add(left, right uint64) uint64 {
	if math.MaxUint64-left < right {
		return math.MaxUint64
	}
	return left + right
}
