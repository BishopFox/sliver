package crack

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type crackJobFetcher func(context.Context, string) (*clientpb.CrackJob, error)

var errNoCrackJobs = errors.New("no crack jobs")

// CrackJobsCmd lists all durable cracking jobs known to the server.
//
//nolint:revive // Keep the established exported command-handler name for compatibility.
func CrackJobsCmd(cmd *cobra.Command, con *console.SliverClient, _ []string) {
	ctx, cancel := crackCommandContext(cmd.Context(), cmd)
	defer cancel()

	jobs, err := con.Rpc.CrackJobs(ctx, &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if jobs == nil || len(jobs.Jobs) == 0 {
		con.PrintInfof("No crack jobs\n")
		return
	}
	con.Printf("%s\n", renderCrackJobs(jobs.Jobs, settings.GetTableStyle(con)))
}

// CrackJobCmd displays one cracking job, optionally polling until completion.
//
//nolint:revive // Keep the established exported command-handler name for compatibility.
func CrackJobCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	pollInterval, _ := cmd.Flags().GetDuration("poll-interval")
	if pollInterval <= 0 {
		con.PrintErrorf("--poll-interval must be greater than zero\n")
		return
	}
	jobID, err := resolveCrackJobID(args, func() (*clientpb.CrackJobs, error) {
		ctx, cancel := crackCommandContext(cmd.Context(), cmd)
		defer cancel()
		return con.Rpc.CrackJobs(ctx, &commonpb.Empty{})
	}, forms.CrackJobSelectForm)
	if errors.Is(err, errNoCrackJobs) {
		con.PrintInfof("No crack jobs\n")
		return
	}
	if errors.Is(err, forms.ErrUserAborted) {
		return
	}
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	timeoutSeconds, _ := cmd.Flags().GetInt64("timeout")
	fetch := func(parent context.Context, id string) (*clientpb.CrackJob, error) {
		ctx := parent
		cancel := func() {}
		if timeoutSeconds > 0 {
			ctx, cancel = context.WithTimeout(parent, time.Duration(timeoutSeconds)*time.Second)
		}
		defer cancel()
		return con.Rpc.CrackJobByID(ctx, &clientpb.CrackJob{ID: id})
	}

	watch, _ := cmd.Flags().GetBool("watch")
	if !watch {
		job, err := fetch(cmd.Context(), jobID)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		con.Printf("%s\n", renderCrackJob(job, settings.GetTableStyle(con)))
		return
	}

	first := true
	err = watchCrackJob(cmd.Context(), jobID, pollInterval, fetch, func(job *clientpb.CrackJob) {
		if !first {
			con.Println()
		}
		first = false
		con.Printf("%s\n", renderCrackJob(job, settings.GetTableStyle(con)))
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		con.PrintErrorf("%s\n", err)
	}
}

func resolveCrackJobID(
	args []string,
	listJobs func() (*clientpb.CrackJobs, error),
	selectJob func([]*clientpb.CrackJob) (string, error),
) (string, error) {
	if len(args) > 1 {
		return "", errors.New("only one crack job ID may be specified")
	}
	if len(args) == 1 {
		jobID := strings.TrimSpace(args[0])
		if jobID == "" {
			return "", errors.New("job ID cannot be empty")
		}
		return jobID, nil
	}
	if listJobs == nil {
		return "", errors.New("crack job lister is nil")
	}

	jobs, err := listJobs()
	if err != nil {
		return "", err
	}
	if jobs == nil || !hasSelectableCrackJob(jobs.GetJobs()) {
		return "", errNoCrackJobs
	}
	if selectJob == nil {
		return "", errors.New("crack job selector is nil")
	}
	jobID, err := selectJob(jobs.GetJobs())
	if err != nil {
		return "", err
	}
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return "", errors.New("no crack job selected")
	}
	return jobID, nil
}

func hasSelectableCrackJob(jobs []*clientpb.CrackJob) bool {
	for _, job := range jobs {
		if job != nil && strings.TrimSpace(job.GetID()) != "" {
			return true
		}
	}
	return false
}

func crackCommandContext(parent context.Context, cmd *cobra.Command) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	timeoutSeconds, _ := cmd.Flags().GetInt64("timeout")
	if timeoutSeconds <= 0 {
		return parent, func() {}
	}
	return context.WithTimeout(parent, time.Duration(timeoutSeconds)*time.Second)
}

func watchCrackJob(ctx context.Context, jobID string, pollInterval time.Duration, fetch crackJobFetcher, onUpdate func(*clientpb.CrackJob)) error {
	if pollInterval <= 0 {
		return errors.New("poll interval must be greater than zero")
	}
	if fetch == nil {
		return errors.New("job fetcher is nil")
	}

	var previous []byte
	for {
		job, err := fetch(ctx, jobID)
		if err != nil {
			return err
		}
		if job == nil {
			return errors.New("server returned an empty crack job")
		}
		serialized, err := proto.MarshalOptions{Deterministic: true}.Marshal(job)
		if err != nil {
			return fmt.Errorf("encode crack job: %w", err)
		}
		if !bytes.Equal(serialized, previous) {
			if onUpdate != nil {
				onUpdate(job)
			}
			previous = serialized
		}
		if crackJobTerminal(job) {
			return nil
		}

		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func crackJobTerminal(job *clientpb.CrackJob) bool {
	if job == nil {
		return false
	}
	status := strings.ToUpper(job.Status.String())
	return strings.HasSuffix(status, "COMPLETED") || strings.HasSuffix(status, "FAILED") || strings.HasSuffix(status, "CANCELLED")
}

func crackJobStatusStyle(status string) console.TextStyle {
	status = strings.ToUpper(strings.TrimSpace(status))
	switch {
	case strings.HasSuffix(status, "COMPLETED"):
		return console.StyleBoldSuccess
	case strings.HasSuffix(status, "FAILED"):
		return console.StyleBoldDanger
	case strings.HasSuffix(status, "CANCELLED"), strings.HasSuffix(status, "CANCELED"):
		return console.StyleBoldGray
	case status == "IN_PROGRESS", strings.HasSuffix(status, "RUNNING"), strings.HasSuffix(status, "LEASED"):
		return console.StyleBoldWarning
	case strings.HasSuffix(status, "QUEUED"), strings.HasSuffix(status, "PENDING"):
		return console.StyleBold
	default:
		return console.StyleBold
	}
}

func renderCrackJobStatus(status string) string {
	status = safeCrackCell(status)
	return crackJobStatusStyle(status).Render(status)
}

func renderCrackJobTitle(title string) string {
	return console.StyleBoldPrimary.Render(title)
}

func renderCrackJobs(jobs []*clientpb.CrackJob, style table.Style) string {
	ordered := append([]*clientpb.CrackJob(nil), jobs...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return crackJobSortTime(ordered[i]).After(crackJobSortTime(ordered[j]))
	})

	tw := table.NewWriter()
	tw.SetStyle(style)
	tw.AppendHeader(table.Row{"ID", "Status", "Created", "Updated", "Keyspace", "Tasks", "Results"})
	for _, job := range ordered {
		if job == nil {
			continue
		}
		tw.AppendRow(table.Row{
			console.StyleBoldPrimary.Render(safeCrackCell(job.ID)),
			renderCrackJobStatus(job.Status.String()),
			formatCrackTime(job.CreatedAt),
			formatCrackTime(protoFieldText(job.ProtoReflect(), "UpdatedAt")),
			valueOrDash(protoFieldText(job.ProtoReflect(), "Keyspace")),
			strconv.Itoa(len(crackJobTasks(job))),
			crackJobResultCount(job),
		})
	}
	return tw.Render()
}

//nolint:gocyclo // Keep the summary, task, device, and result sections in one deterministic job rendering pass.
func renderCrackJob(job *clientpb.CrackJob, style table.Style) string {
	if job == nil {
		return "Empty crack job"
	}

	sections := make([]string, 0, 4)
	summary := table.NewWriter()
	summary.SetStyle(style)
	summary.SetTitle(renderCrackJobTitle("Crack Job"))
	summary.AppendRow(table.Row{console.StyleBold.Render("ID"), console.StyleBoldPrimary.Render(safeCrackCell(job.ID))})
	summary.AppendRow(table.Row{console.StyleBold.Render("Status"), renderCrackJobStatus(job.Status.String())})
	summary.AppendRow(table.Row{console.StyleBold.Render("Created"), formatCrackTime(job.CreatedAt)})
	summary.AppendRow(table.Row{console.StyleBold.Render("Updated"), formatCrackTime(protoFieldText(job.ProtoReflect(), "UpdatedAt"))})
	summary.AppendRow(table.Row{console.StyleBold.Render("Keyspace"), valueOrDash(protoFieldText(job.ProtoReflect(), "Keyspace"))})
	if job.Err != "" {
		summary.AppendRow(table.Row{console.StyleBoldDanger.Render("Error"), console.StyleDanger.Render(safeCrackCell(job.Err))})
	}
	sections = append(sections, summary.Render())

	tasks := crackJobTasks(job)
	if len(tasks) > 0 {
		taskTable := table.NewWriter()
		taskTable.SetStyle(style)
		taskTable.SetTitle(renderCrackJobTitle("Tasks"))
		taskTable.AppendHeader(table.Row{"ID", "Kind", "State", "Station", "Range", "Progress", "Speed", "Temp", "Updated"})
		devices := make([]crackDeviceView, 0)
		for _, task := range tasks {
			status, statusErr := crackTaskStatus(task)
			progress, speed, temperature := "-", "-", "-"
			if statusErr != nil {
				progress = console.StyleDanger.Render("invalid status JSON")
			} else if status != nil {
				progress = status.progressText()
				speed = status.speedText()
				temperature = status.temperatureText()
				for _, device := range status.Devices {
					device.TaskID = task.ID
					devices = append(devices, device)
				}
			}
			taskTable.AppendRow(table.Row{
				safeCrackCell(task.ID),
				valueOrDash(protoFieldText(task.ProtoReflect(), "Kind")),
				renderCrackJobStatus(crackTaskState(task)),
				valueOrDash(task.HostUUID),
				crackTaskRange(task),
				progress,
				speed,
				temperature,
				formatCrackTime(protoFieldText(task.ProtoReflect(), "UpdatedAt", "LastHeartbeatAt")),
			})
		}
		sections = append(sections, taskTable.Render())

		statisticsTable := table.NewWriter()
		statisticsTable.SetStyle(style)
		statisticsTable.SetTitle(renderCrackJobTitle("Hashcat Statistics"))
		statisticsTable.AppendHeader(table.Row{
			"Task", "Session", "Status", "Target", "Guess", "Progress", "Rejected", "Restore",
			"Recovered Hashes", "Recovered Salts", "Started", "Estimated Stop",
		})
		hasStatistics := false
		for _, task := range tasks {
			status, err := crackTaskStatus(task)
			if err != nil || status == nil {
				continue
			}
			hasStatistics = true
			statisticsTable.AppendRow(table.Row{
				safeCrackCell(task.ID),
				valueOrDash(status.Session),
				valueOrDash(status.Status),
				valueOrDash(status.Target),
				status.guessText(),
				status.progressText(),
				valueOrDash(status.Rejected),
				valueOrDash(status.RestorePoint),
				pairText(status.RecoveredHashesCurrent, status.RecoveredHashesTotal),
				pairText(status.RecoveredSaltsCurrent, status.RecoveredSaltsTotal),
				formatCrackTime(status.TimeStart),
				formatCrackTime(status.EstimatedStop),
			})
		}
		if hasStatistics {
			sections = append(sections, statisticsTable.Render())
		}

		if len(devices) > 0 {
			deviceTable := table.NewWriter()
			deviceTable.SetStyle(style)
			deviceTable.SetTitle(renderCrackJobTitle("Devices"))
			deviceTable.AppendHeader(table.Row{"Task", "ID", "Name", "Type", "Speed", "Temp", "Util", "Fan", "Core", "Memory", "Bus", "Power"})
			for _, device := range devices {
				deviceTable.AppendRow(table.Row{
					safeCrackCell(device.TaskID),
					valueOrDash(device.ID),
					valueOrDash(device.Name),
					valueOrDash(device.Type),
					device.speedText(),
					unitValue(device.Temperature, " C"),
					unitValue(device.Utilization, "%"),
					unitValue(device.FanSpeed, "%"),
					unitValue(device.CoreSpeed, " MHz"),
					unitValue(device.MemorySpeed, " MHz"),
					unitValue(device.BusLanes, " lanes"),
					unitValue(device.Power, " mW"),
				})
			}
			sections = append(sections, deviceTable.Render())
		}
	}

	results := crackJobResults(job)
	if len(results) > 0 {
		resultTable := table.NewWriter()
		resultTable.SetStyle(style)
		resultTable.SetTitle(console.StyleBoldSuccess.Render("Recovered Credentials"))
		resultTable.AppendHeader(table.Row{"Credential", "Hash", "Plaintext"})
		for _, result := range results {
			resultTable.AppendRow(table.Row{
				valueOrDash(result.CredentialID),
				valueOrDash(result.Hash),
				valueOrDash(result.Plaintext),
			})
		}
		sections = append(sections, resultTable.Render())
	}

	return strings.Join(sections, "\n\n")
}

func crackJobSortTime(job *clientpb.CrackJob) time.Time {
	if job == nil {
		return time.Time{}
	}
	for _, value := range []string{protoFieldText(job.ProtoReflect(), "UpdatedAt"), job.CreatedAt} {
		if unix, err := strconv.ParseInt(value, 10, 64); err == nil && unix > 0 {
			return time.Unix(unix, 0)
		}
		if parsed, err := time.Parse(time.RFC3339, value); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func crackJobTasks(job *clientpb.CrackJob) []*clientpb.CrackTask {
	if job == nil {
		return nil
	}
	list, ok := protoRepeatedMessages(job.ProtoReflect(), "Tasks")
	if !ok {
		return nil
	}
	tasks := make([]*clientpb.CrackTask, 0, list.Len())
	for index := 0; index < list.Len(); index++ {
		if task, ok := list.Get(index).Message().Interface().(*clientpb.CrackTask); ok {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

func crackTaskState(task *clientpb.CrackTask) string {
	if state := protoFieldText(task.ProtoReflect(), "State"); state != "" {
		return safeCrackCell(state)
	}
	if task.Err != "" {
		return "FAILED"
	}
	if task.CompletedAt != 0 {
		return "COMPLETED"
	}
	if task.StartedAt != 0 {
		return "RUNNING"
	}
	return "QUEUED"
}

func crackTaskRange(task *clientpb.CrackTask) string {
	skip := protoFieldText(task.ProtoReflect(), "ShardSkip")
	limit := protoFieldText(task.ProtoReflect(), "ShardLimit")
	if skip == "" && task.Command != nil {
		skip = strconv.FormatUint(task.Command.Skip, 10)
	}
	if limit == "" && task.Command != nil {
		limit = strconv.FormatUint(task.Command.Limit, 10)
	}
	if skip == "" {
		skip = "0"
	}
	if limit == "" || limit == "0" {
		return skip + "+"
	}
	return skip + "+" + limit
}

func crackTaskStatus(task *clientpb.CrackTask) (*crackStatusView, error) {
	raw := protoFieldBytes(task.ProtoReflect(), "LatestStatusJSON", "RawStatusJSON", "StatusJSON")
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	return parseCrackStatusJSON(raw)
}

type crackStatusView struct {
	Session                string
	Status                 string
	Target                 string
	ProgressCurrent        string
	ProgressTotal          string
	RestorePoint           string
	RecoveredHashesCurrent string
	RecoveredHashesTotal   string
	RecoveredSaltsCurrent  string
	RecoveredSaltsTotal    string
	Rejected               string
	TimeStart              string
	EstimatedStop          string
	Guess                  crackGuessView
	Devices                []crackDeviceView
}

type crackGuessView struct {
	Mode        string
	Base        string
	BaseOffset  string
	BaseCount   string
	BasePercent string
	MaskLength  string
	Mod         string
	ModOffset   string
	ModCount    string
	ModPercent  string
}

func (status *crackStatusView) progressText() string {
	if status == nil || status.ProgressCurrent == "" || status.ProgressTotal == "" {
		return "-"
	}
	current, currentErr := strconv.ParseFloat(status.ProgressCurrent, 64)
	total, totalErr := strconv.ParseFloat(status.ProgressTotal, 64)
	if currentErr == nil && totalErr == nil && total > 0 {
		return fmt.Sprintf("%s/%s (%.1f%%)", status.ProgressCurrent, status.ProgressTotal, math.Min(100, current/total*100))
	}
	return status.ProgressCurrent + "/" + status.ProgressTotal
}

func (status *crackStatusView) speedText() string {
	if status == nil {
		return "-"
	}
	var total uint64
	for _, device := range status.Devices {
		speed, err := strconv.ParseUint(device.Speed, 10, 64)
		if err == nil {
			total += speed
		}
	}
	if total == 0 {
		return "-"
	}
	return humanizeHashRate(total)
}

func (status *crackStatusView) temperatureText() string {
	if status == nil {
		return "-"
	}
	var maximum float64
	found := false
	for _, device := range status.Devices {
		temperature, err := strconv.ParseFloat(device.Temperature, 64)
		if err == nil && temperature >= 0 && (!found || temperature > maximum) {
			maximum = temperature
			found = true
		}
	}
	if !found {
		return "-"
	}
	return fmt.Sprintf("%g C", maximum)
}

func (status *crackStatusView) guessText() string {
	if status == nil {
		return "-"
	}
	parts := make([]string, 0, 3)
	if status.Guess.Mode != "" {
		parts = append(parts, "mode="+status.Guess.Mode)
	}
	if status.Guess.Base != "" {
		base := status.Guess.Base
		if status.Guess.BaseOffset != "" || status.Guess.BaseCount != "" {
			base += " [" + pairText(status.Guess.BaseOffset, status.Guess.BaseCount) + "]"
		}
		if status.Guess.BasePercent != "" {
			base += " (" + status.Guess.BasePercent + "%)"
		}
		parts = append(parts, "base="+base)
	}
	if status.Guess.MaskLength != "" {
		parts = append(parts, "mask-length="+status.Guess.MaskLength)
	}
	if status.Guess.Mod != "" {
		mod := status.Guess.Mod
		if status.Guess.ModOffset != "" || status.Guess.ModCount != "" {
			mod += " [" + pairText(status.Guess.ModOffset, status.Guess.ModCount) + "]"
		}
		if status.Guess.ModPercent != "" {
			mod += " (" + status.Guess.ModPercent + "%)"
		}
		parts = append(parts, "mod="+mod)
	}
	if len(parts) == 0 {
		return "-"
	}
	return safeCrackCell(strings.Join(parts, "; "))
}

type crackDeviceView struct {
	TaskID      string
	ID          string
	Name        string
	Type        string
	Speed       string
	Temperature string
	Utilization string
	FanSpeed    string
	CoreSpeed   string
	MemorySpeed string
	BusLanes    string
	Power       string
}

func (device crackDeviceView) speedText() string {
	speed, err := strconv.ParseUint(device.Speed, 10, 64)
	if err != nil || speed == 0 {
		return valueOrDash(device.Speed)
	}
	return humanizeHashRate(speed)
}

func parseCrackStatusJSON(raw []byte) (*crackStatusView, error) {
	root, err := decodeLastJSONObject(raw)
	if err != nil {
		return nil, err
	}
	status := &crackStatusView{
		Session:       jsonText(firstJSONValue(root, "session")),
		Status:        jsonText(firstJSONValue(root, "status_string", "status")),
		Target:        jsonText(firstJSONValue(root, "target")),
		RestorePoint:  jsonText(firstJSONValue(root, "restore_point")),
		Rejected:      jsonText(firstJSONValue(root, "rejected")),
		TimeStart:     jsonText(firstJSONValue(root, "time_start")),
		EstimatedStop: jsonText(firstJSONValue(root, "estimated_stop")),
	}
	status.ProgressCurrent, status.ProgressTotal = jsonPair(firstJSONValue(root, "progress"))
	status.RecoveredHashesCurrent, status.RecoveredHashesTotal = jsonPair(firstJSONValue(root, "recovered_hashes"))
	status.RecoveredSaltsCurrent, status.RecoveredSaltsTotal = jsonPair(firstJSONValue(root, "recovered_salts"))
	if guess, ok := firstJSONValue(root, "guess").(map[string]any); ok {
		status.Guess = crackGuessView{
			Mode:        jsonText(firstJSONValue(guess, "guess_mode", "mode")),
			Base:        jsonText(firstJSONValue(guess, "guess_base", "base")),
			BaseOffset:  jsonText(firstJSONValue(guess, "guess_base_offset", "base_offset")),
			BaseCount:   jsonText(firstJSONValue(guess, "guess_base_count", "base_count")),
			BasePercent: jsonText(firstJSONValue(guess, "guess_base_percent", "base_percent")),
			MaskLength:  jsonText(firstJSONValue(guess, "guess_mask_length", "mask_length")),
			Mod:         jsonText(firstJSONValue(guess, "guess_mod", "mod")),
			ModOffset:   jsonText(firstJSONValue(guess, "guess_mod_offset", "mod_offset")),
			ModCount:    jsonText(firstJSONValue(guess, "guess_mod_count", "mod_count")),
			ModPercent:  jsonText(firstJSONValue(guess, "guess_mod_percent", "mod_percent")),
		}
	}
	if devices, ok := firstJSONValue(root, "devices").([]any); ok {
		for _, item := range devices {
			values, ok := item.(map[string]any)
			if !ok {
				continue
			}
			status.Devices = append(status.Devices, crackDeviceView{
				ID:          jsonText(firstJSONValue(values, "device_id", "id")),
				Name:        safeCrackCell(jsonText(firstJSONValue(values, "device_name", "name"))),
				Type:        safeCrackCell(jsonText(firstJSONValue(values, "device_type", "type"))),
				Speed:       jsonText(firstJSONValue(values, "speed")),
				Temperature: jsonText(firstJSONValue(values, "temp", "temperature")),
				Utilization: jsonText(firstJSONValue(values, "util", "utilization")),
				FanSpeed:    jsonText(firstJSONValue(values, "fanspeed", "fan_speed")),
				CoreSpeed:   jsonText(firstJSONValue(values, "corespeed", "core_speed")),
				MemorySpeed: jsonText(firstJSONValue(values, "memoryspeed", "memory_speed")),
				BusLanes:    jsonText(firstJSONValue(values, "buslanes", "bus_lanes")),
				Power:       jsonText(firstJSONValue(values, "power")),
			})
		}
	}
	return status, nil
}

func decodeLastJSONObject(raw []byte) (map[string]any, error) {
	decode := func(candidate []byte) (map[string]any, error) {
		decoder := json.NewDecoder(bytes.NewReader(candidate))
		decoder.UseNumber()
		var last map[string]any
		for {
			value := map[string]any{}
			if err := decoder.Decode(&value); err != nil {
				break
			}
			last = value
		}
		if last == nil {
			return nil, errors.New("no JSON object")
		}
		return last, nil
	}
	trimmed := bytes.TrimSpace(raw)
	if value, err := decode(trimmed); err == nil {
		return value, nil
	}
	lines := bytes.Split(trimmed, []byte{'\n'})
	for index := len(lines) - 1; index >= 0; index-- {
		candidate := bytes.TrimSpace(lines[index])
		if len(candidate) == 0 {
			continue
		}
		if value, err := decode(candidate); err == nil {
			return value, nil
		}
	}
	return nil, errors.New("hashcat status did not contain a JSON object")
}

type crackResultView struct {
	CredentialID string
	Hash         string
	Plaintext    string
}

func crackJobResults(job *clientpb.CrackJob) []crackResultView {
	if job == nil {
		return nil
	}
	results := make([]crackResultView, 0)
	if list, ok := protoRepeatedMessages(job.ProtoReflect(), "Results"); ok {
		for index := 0; index < list.Len(); index++ {
			message := list.Get(index).Message()
			results = append(results, crackResultView{
				CredentialID: safeCrackCell(protoFieldText(message, "CredentialID")),
				Hash:         safeCrackCell(protoFieldText(message, "Hash")),
				Plaintext:    displayCrackPlaintext(protoFieldBytes(message, "Plaintext")),
			})
		}
	}
	// The job-level result rows are authoritative once the server has
	// materialized them. Fall back to worker JSON only while those rows have not
	// appeared yet, avoiding duplicate plaintexts in a watched job.
	if len(results) == 0 {
		for _, task := range crackJobTasks(job) {
			results = append(results, parseRecoveredResults(protoFieldBytes(task.ProtoReflect(), "RecoveredJSON"))...)
		}
	}

	deduplicated := make([]crackResultView, 0, len(results))
	seen := map[string]struct{}{}
	for _, result := range results {
		key := result.CredentialID + "\x00" + result.Hash + "\x00" + result.Plaintext
		if _, ok := seen[key]; ok || (result.Hash == "" && result.Plaintext == "" && result.CredentialID == "") {
			continue
		}
		seen[key] = struct{}{}
		deduplicated = append(deduplicated, result)
	}
	return deduplicated
}

func crackJobResultCount(job *clientpb.CrackJob) string {
	if job == nil {
		return "-"
	}
	return valueOrDash(protoFieldText(job.ProtoReflect(), "ResultCount"))
}

func displayCrackPlaintext(plaintext []byte) string {
	if !utf8.Valid(plaintext) || bytes.IndexByte(plaintext, 0) >= 0 {
		return "0x" + hex.EncodeToString(plaintext)
	}
	return safeCrackCell(string(plaintext))
}

func parseRecoveredResults(raw []byte) []crackResultView {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var items []map[string]any
	if err := decoder.Decode(&items); err != nil {
		return nil
	}
	results := make([]crackResultView, 0, len(items))
	for _, item := range items {
		results = append(results, crackResultView{
			CredentialID: safeCrackCell(jsonText(firstJSONValue(item, "credential_id", "credentialID"))),
			Hash:         safeCrackCell(jsonText(firstJSONValue(item, "hash"))),
			Plaintext:    decodeRecoveredPlaintext(jsonText(firstJSONValue(item, "plaintext"))),
		})
	}
	return results
}

func decodeRecoveredPlaintext(value string) string {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		// Older crackstations encoded plaintext as a JSON string. Preserve that
		// representation when it is not valid protobuf bytes JSON.
		return safeCrackCell(value)
	}
	if !utf8.Valid(decoded) || bytes.IndexByte(decoded, 0) >= 0 {
		return "0x" + hex.EncodeToString(decoded)
	}
	return safeCrackCell(string(decoded))
}

func protoRepeatedMessages(message protoreflect.Message, names ...string) (protoreflect.List, bool) {
	field := findProtoField(message, names...)
	if field == nil || !field.IsList() || field.Kind() != protoreflect.MessageKind {
		return nil, false
	}
	return message.Get(field).List(), true
}

func protoFieldBytes(message protoreflect.Message, names ...string) []byte {
	field := findProtoField(message, names...)
	if field == nil {
		return nil
	}
	switch field.Kind() {
	case protoreflect.BytesKind:
		return append([]byte(nil), message.Get(field).Bytes()...)
	case protoreflect.StringKind:
		return []byte(message.Get(field).String())
	default:
		return nil
	}
}

func protoFieldText(message protoreflect.Message, names ...string) string {
	field := findProtoField(message, names...)
	if field == nil {
		return ""
	}
	value := message.Get(field)
	switch field.Kind() {
	case protoreflect.StringKind:
		return safeCrackCell(value.String())
	case protoreflect.BytesKind:
		return safeCrackCell(string(value.Bytes()))
	case protoreflect.EnumKind:
		if descriptor := field.Enum().Values().ByNumber(value.Enum()); descriptor != nil {
			return string(descriptor.Name())
		}
		return strconv.FormatInt(int64(value.Enum()), 10)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return strconv.FormatInt(value.Int(), 10)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind, protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return strconv.FormatUint(value.Uint(), 10)
	case protoreflect.FloatKind:
		return strconv.FormatFloat(value.Float(), 'f', -1, 32)
	case protoreflect.DoubleKind:
		return strconv.FormatFloat(value.Float(), 'f', -1, 64)
	case protoreflect.BoolKind:
		return strconv.FormatBool(value.Bool())
	default:
		return ""
	}
}

func findProtoField(message protoreflect.Message, names ...string) protoreflect.FieldDescriptor {
	if !message.IsValid() {
		return nil
	}
	wanted := map[string]struct{}{}
	for _, name := range names {
		wanted[normalizedProtoName(name)] = struct{}{}
	}
	fields := message.Descriptor().Fields()
	for index := 0; index < fields.Len(); index++ {
		field := fields.Get(index)
		if _, ok := wanted[normalizedProtoName(string(field.Name()))]; ok {
			return field
		}
		if _, ok := wanted[normalizedProtoName(field.JSONName())]; ok {
			return field
		}
	}
	return nil
}

func normalizedProtoName(value string) string {
	value = strings.ToLower(value)
	return strings.NewReplacer("_", "", "-", "").Replace(value)
}

func firstJSONValue(values map[string]any, names ...string) any {
	for _, name := range names {
		if value, ok := values[name]; ok {
			return value
		}
	}
	return nil
}

func jsonPair(value any) (string, string) {
	items, ok := value.([]any)
	if !ok || len(items) < 2 {
		return "", ""
	}
	return jsonText(items[0]), jsonText(items[1])
}

func pairText(first, second string) string {
	if first == "" && second == "" {
		return "-"
	}
	return valueOrDash(first) + "/" + valueOrDash(second)
}

func jsonText(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return safeCrackCell(typed)
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(typed)
	default:
		return safeCrackCell(fmt.Sprint(typed))
	}
}

func formatCrackTime(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "0" {
		return "-"
	}
	if unix, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.Unix(unix, 0).UTC().Format(time.RFC3339)
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC().Format(time.RFC3339)
	}
	return safeCrackCell(value)
}

func unitValue(value, suffix string) string {
	if value == "" {
		return "-"
	}
	if numeric, err := strconv.ParseFloat(value, 64); err == nil && numeric < 0 {
		return "-"
	}
	return safeCrackCell(value) + suffix
}

func valueOrDash(value string) string {
	value = safeCrackCell(value)
	if value == "" {
		return "-"
	}
	return value
}

func safeCrackCell(value string) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
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
