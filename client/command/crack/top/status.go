package top

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/command/crack/internal/hashcatdisplay"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

// These views intentionally contain only the status fields used by Crack Top.
// The production loader fills them from the lean CrackTop RPC; JSON decoding is
// retained for compatibility with locally constructed snapshots and fixtures.
type crackStatusView struct {
	ProgressCurrent string
	ProgressTotal   string
	Devices         []crackDeviceView
}

type crackDeviceView struct {
	ID          string
	Name        string
	Type        string
	Speed       string
	Temperature string
	Utilization string
}

func hashcatHashTypeName(mode int32) (string, bool) {
	return hashcatdisplay.TypeName(mode)
}

func humanizeHashRate(rate uint64) string {
	return hashcatdisplay.HumanizeRate(rate)
}

func crackJobSortTime(job *clientpb.CrackJob) time.Time {
	if job == nil {
		return time.Time{}
	}
	if updatedAt := job.GetUpdatedAt(); updatedAt > 0 {
		return time.Unix(updatedAt, 0)
	}
	createdAt := strings.TrimSpace(job.GetCreatedAt())
	if unix, err := strconv.ParseInt(createdAt, 10, 64); err == nil && unix > 0 {
		return time.Unix(unix, 0)
	}
	parsed, _ := time.Parse(time.RFC3339, createdAt)
	return parsed
}

func crackJobTasks(job *clientpb.CrackJob) []*clientpb.CrackTask {
	if job == nil {
		return nil
	}
	return job.GetTasks()
}

func crackTaskStatus(task *clientpb.CrackTask) (*crackStatusView, error) {
	if task == nil || len(bytes.TrimSpace(task.GetLatestStatusJSON())) == 0 {
		return nil, nil
	}
	return parseCrackStatusJSON(task.GetLatestStatusJSON())
}

func parseCrackStatusJSON(raw []byte) (*crackStatusView, error) {
	root, err := decodeLastCrackTopJSONObject(raw)
	if err != nil {
		return nil, err
	}
	status := &crackStatusView{}
	status.ProgressCurrent, status.ProgressTotal = crackTopJSONPair(root["progress"])
	if devices, ok := root["devices"].([]any); ok {
		for _, item := range devices {
			values, ok := item.(map[string]any)
			if !ok {
				continue
			}
			status.Devices = append(status.Devices, crackDeviceView{
				ID:          crackTopJSONText(firstCrackTopJSONValue(values, "device_id", "id")),
				Name:        crackTopSafeCell(crackTopJSONText(firstCrackTopJSONValue(values, "device_name", "name"))),
				Type:        crackTopSafeCell(crackTopJSONText(firstCrackTopJSONValue(values, "device_type", "type"))),
				Speed:       crackTopJSONText(firstCrackTopJSONValue(values, "speed")),
				Temperature: crackTopJSONText(firstCrackTopJSONValue(values, "temp", "temperature")),
				Utilization: crackTopJSONText(firstCrackTopJSONValue(values, "util", "utilization")),
			})
		}
	}
	return status, nil
}

func decodeLastCrackTopJSONObject(raw []byte) (map[string]any, error) {
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

func firstCrackTopJSONValue(values map[string]any, names ...string) any {
	for _, name := range names {
		if value, ok := values[name]; ok {
			return value
		}
	}
	return nil
}

func crackTopJSONPair(value any) (string, string) {
	items, ok := value.([]any)
	if !ok || len(items) < 2 {
		return "", ""
	}
	return crackTopJSONText(items[0]), crackTopJSONText(items[1])
}

func crackTopJSONText(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return crackTopSafeCell(typed)
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(typed)
	default:
		return crackTopSafeCell(fmt.Sprint(typed))
	}
}

func valueOrDash(value string) string {
	value = crackTopSafeCell(value)
	if value == "" {
		return "-"
	}
	return value
}
