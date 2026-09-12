package top

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/charmbracelet/x/ansi"
	"google.golang.org/grpc"
)

const crackTopWarningRunes = 256

// crackTopRPC is the complete wire-level dependency of the crack top client.
// Keeping it narrow also makes it difficult for the dashboard to accidentally
// fall back to RPCs that expose crack targets or recovered material.
type crackTopRPC interface {
	CrackTop(context.Context, *commonpb.Empty, ...grpc.CallOption) (*clientpb.CrackTopSnapshot, error)
}

type crackTopTaskTelemetry struct {
	status        *crackStatusView
	parseErr      bool
	progressExact bool
}

// crackTopSnapshot is the client-side representation consumed by the existing
// dashboard telemetry code. Jobs and stations contain only fields supplied by
// the narrow CrackTop RPC; parsed status is kept separately so no sensitive raw
// status JSON has to be reconstructed.
type crackTopSnapshot struct {
	Jobs          []*clientpb.CrackJob
	Stations      []*clientpb.Crackstation
	RefreshedAt   time.Time
	taskTelemetry map[string]crackTopTaskTelemetry
}

// CrackTop snapshots are atomic. A successful refresh replaces all previously
// displayed server state, while a failed refresh returns no snapshot and leaves
// the model's existing value untouched.
func mergeCrackTopSnapshot(_ *crackTopSnapshot, incoming *crackTopSnapshot) *crackTopSnapshot {
	return incoming
}

func loadCrackTopSnapshot(ctx context.Context, rpc crackTopRPC, perCallTimeout time.Duration, completionTime time.Time) (*crackTopSnapshot, error) {
	if rpc == nil {
		return nil, crackTopContextError("load crack top snapshot", errors.New("rpc client is unavailable"))
	}
	if ctx == nil {
		ctx = context.Background()
	}

	var cancel context.CancelFunc
	if perCallTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, perCallTimeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	wire, err := rpc.CrackTop(ctx, &commonpb.Empty{})
	if completionTime.IsZero() {
		completionTime = time.Now()
	}
	if err != nil {
		return nil, crackTopContextError("load crack top snapshot", err)
	}
	if wire == nil {
		return nil, crackTopContextError("load crack top snapshot", errors.New("server returned an empty response"))
	}

	refreshedAt := completionTime
	if wire.GetObservedAt() != 0 {
		refreshedAt = time.Unix(wire.GetObservedAt(), 0)
	}
	snapshot := &crackTopSnapshot{
		Jobs:          make([]*clientpb.CrackJob, len(wire.GetJobs())),
		Stations:      make([]*clientpb.Crackstation, len(wire.GetCrackstations())),
		RefreshedAt:   refreshedAt,
		taskTelemetry: map[string]crackTopTaskTelemetry{},
	}
	for index, job := range wire.GetJobs() {
		snapshot.Jobs[index] = crackTopJobFromWire(job, snapshot.taskTelemetry)
	}
	for index, station := range wire.GetCrackstations() {
		snapshot.Stations[index] = crackTopStationFromWire(station)
	}
	return snapshot, nil
}

func crackTopJobFromWire(wire *clientpb.CrackTopJob, telemetry map[string]crackTopTaskTelemetry) *clientpb.CrackJob {
	if wire == nil {
		return nil
	}

	command := &clientpb.CrackCommand{
		AttackMode: wire.GetAttackMode(),
		HashType:   wire.GetHashType(),
	}
	if wire.HashMode != nil {
		hashMode := wire.GetHashMode()
		command.HashMode = &hashMode
	}
	job := &clientpb.CrackJob{
		ID:          wire.GetID(),
		CreatedAt:   wire.GetCreatedAt(),
		CompletedAt: wire.GetCompletedAt(),
		Status:      wire.GetStatus(),
		Command:     command,
		UpdatedAt:   wire.GetUpdatedAt(),
		ResultCount: wire.GetResultCount(),
		Tasks:       make([]*clientpb.CrackTask, len(wire.GetTasks())),
	}
	for index, task := range wire.GetTasks() {
		job.Tasks[index] = crackTopTaskFromWire(task, telemetry)
	}
	return job
}

func crackTopTaskFromWire(wire *clientpb.CrackTopTask, telemetry map[string]crackTopTaskTelemetry) *clientpb.CrackTask {
	if wire == nil {
		return nil
	}

	task := &clientpb.CrackTask{
		ID:              wire.GetID(),
		HostUUID:        wire.GetHostUUID(),
		CreatedAt:       wire.GetCreatedAt(),
		StartedAt:       wire.GetStartedAt(),
		CompletedAt:     wire.GetCompletedAt(),
		Kind:            wire.GetKind(),
		State:           wire.GetState(),
		Attempt:         wire.GetAttempt(),
		LeaseExpiresAt:  wire.GetLeaseExpiresAt(),
		UpdatedAt:       wire.GetUpdatedAt(),
		LastHeartbeatAt: wire.GetLastHeartbeatAt(),
		ShardSkip:       wire.GetShardSkip(),
		ShardLimit:      wire.GetShardLimit(),
	}

	taskID := strings.TrimSpace(task.GetID())
	if telemetry == nil || taskID == "" {
		return task
	}
	metadata := crackTopTaskTelemetry{
		parseErr:      wire.GetStatusParseError(),
		progressExact: wire.GetProgressExact(),
	}
	if wire.GetStatusAvailable() && !metadata.parseErr {
		metadata.status = &crackStatusView{Devices: crackTopDevicesFromWire(wire.GetDevices())}
		if metadata.progressExact || wire.GetShardSkip() == 0 {
			metadata.status.ProgressCurrent = wire.GetProgressCurrent()
			metadata.status.ProgressTotal = wire.GetProgressTotal()
		}
	}
	if metadata.status != nil || metadata.parseErr {
		telemetry[taskID] = metadata
	}
	return task
}

func crackTopDevicesFromWire(devices []*clientpb.CrackTopDevice) []crackDeviceView {
	converted := make([]crackDeviceView, 0, len(devices))
	for _, device := range devices {
		if device == nil {
			continue
		}
		view := crackDeviceView{ID: device.GetID()}
		if device.GetSpeedAvailable() {
			view.Speed = strconv.FormatUint(device.GetSpeed(), 10)
		}
		if device.GetTemperatureAvailable() {
			view.Temperature = strconv.FormatFloat(device.GetTemperature(), 'f', -1, 64)
		}
		if device.GetUtilizationAvailable() {
			view.Utilization = strconv.FormatFloat(device.GetUtilization(), 'f', -1, 64)
		}
		converted = append(converted, view)
	}
	return converted
}

func crackTopStationFromWire(wire *clientpb.CrackTopStation) *clientpb.Crackstation {
	if wire == nil {
		return nil
	}
	station := &clientpb.Crackstation{
		ID:       wire.GetID(),
		Name:     wire.GetName(),
		HostUUID: wire.GetHostUUID(),
	}
	if wire.GetStatusAvailable() {
		station.Status = &clientpb.CrackstationStatus{
			Name:              wire.GetName(),
			HostUUID:          wire.GetHostUUID(),
			State:             wire.GetState(),
			CurrentCrackJobID: wire.GetCurrentCrackJobID(),
			IsSyncing:         wire.GetIsSyncing(),
		}
	}
	return station
}

func crackTopContextError(scope string, err error) error {
	message := crackTopSanitizeError(err)
	if strings.TrimSpace(scope) == "" {
		return errors.New(message)
	}
	return fmt.Errorf("%s: %s", strings.TrimSpace(scope), message)
}

func crackTopWarningText(message string) string {
	return crackTopSanitizeError(errors.New(message))
}

func crackTopSanitizeError(err error) string {
	if err == nil {
		return "unknown error"
	}
	message := ansi.Strip(err.Error())
	message = strings.Map(func(character rune) rune {
		if unicode.IsSpace(character) {
			return ' '
		}
		if !unicode.IsPrint(character) {
			return -1
		}
		return character
	}, message)
	message = strings.Join(strings.Fields(message), " ")
	if message == "" {
		return "unknown error"
	}
	runes := []rune(message)
	if len(runes) > crackTopWarningRunes {
		message = string(runes[:crackTopWarningRunes-1]) + "…"
	}
	return message
}
