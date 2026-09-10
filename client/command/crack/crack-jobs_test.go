package crack

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestCrackJobCommandsAreRegistered(t *testing.T) {
	root := Commands(nil)[0]
	jobs, _, err := root.Find([]string{"jobs"})
	if err != nil || jobs == root {
		t.Fatalf("find crack jobs: command=%v error=%v", jobs, err)
	}
	job, _, err := root.Find([]string{"job", "job-id"})
	if err != nil || job == root {
		t.Fatalf("find crack job: command=%v error=%v", job, err)
	}
	if job.Flags().Lookup("watch") == nil || job.Flags().Lookup("poll-interval") == nil {
		t.Fatal("crack job is missing watch flags")
	}
	if err := job.Args(job, nil); err == nil {
		t.Fatal("crack job accepted a missing job ID")
	}
	if err := job.Args(job, []string{"job-id"}); err != nil {
		t.Fatalf("crack job rejected one job ID: %v", err)
	}
}

func TestParseCrackStatusJSONUsesNewestStatus(t *testing.T) {
	raw := []byte("{\"status\":1,\"progress\":[1,10]}\n" +
		"{\"session\":\"distributed-job\",\"status\":2,\"target\":\"hashes\",\"progress\":[5,10]," +
		"\"restore_point\":4,\"recovered_hashes\":[1,3],\"recovered_salts\":[1,2],\"rejected\":7," +
		"\"time_start\":1700000000,\"estimated_stop\":1700000100," +
		"\"guess\":{\"guess_mode\":3,\"guess_base\":\"?d?d\",\"guess_base_offset\":1,\"guess_base_count\":10," +
		"\"guess_base_percent\":25.5,\"guess_mask_length\":2,\"guess_mod\":\"rules\",\"guess_mod_offset\":2,\"guess_mod_count\":4,\"guess_mod_percent\":50},\"devices\":[" +
		"{\"device_id\":1,\"device_name\":\"GPU 0\",\"device_type\":\"GPU\",\"speed\":1500,\"temp\":65,\"util\":98,\"corespeed\":2100,\"memoryspeed\":9500,\"buslanes\":16}," +
		"{\"id\":2,\"name\":\"GPU 1\",\"type\":\"GPU\",\"speed\":500,\"temperature\":70,\"utilization\":97}]}\n")

	status, err := parseCrackStatusJSON(raw)
	if err != nil {
		t.Fatalf("parseCrackStatusJSON: %v", err)
	}
	if status.Status != "2" || status.ProgressCurrent != "5" || status.ProgressTotal != "10" {
		t.Fatalf("unexpected status: %#v", status)
	}
	if got := status.progressText(); got != "5/10 (50.0%)" {
		t.Fatalf("progressText = %q", got)
	}
	if got := status.speedText(); got != "2.00 kH/s" {
		t.Fatalf("speedText = %q", got)
	}
	if got := status.temperatureText(); got != "70 C" {
		t.Fatalf("temperatureText = %q", got)
	}
	if len(status.Devices) != 2 || status.Devices[1].Name != "GPU 1" {
		t.Fatalf("devices = %#v", status.Devices)
	}
	if status.Session != "distributed-job" || status.Target != "hashes" || status.RestorePoint != "4" ||
		status.RecoveredHashesCurrent != "1" || status.RecoveredHashesTotal != "3" ||
		status.RecoveredSaltsCurrent != "1" || status.RecoveredSaltsTotal != "2" || status.Rejected != "7" ||
		status.TimeStart != "1700000000" || status.EstimatedStop != "1700000100" {
		t.Fatalf("full status fields were not decoded: %#v", status)
	}
	if status.Guess.Mode != "3" || status.Guess.Base != "?d?d" || status.Guess.BasePercent != "25.5" || status.Guess.MaskLength != "2" || status.Guess.Mod != "rules" || status.Guess.ModPercent != "50" {
		t.Fatalf("guess = %#v", status.Guess)
	}
	if got := status.guessText(); !strings.Contains(got, "base=?d?d [1/10] (25.5%)") || !strings.Contains(got, "mask-length=2") || !strings.Contains(got, "mod=rules [2/4] (50%)") {
		t.Fatalf("guessText = %q", got)
	}
	if status.Devices[0].CoreSpeed != "2100" || status.Devices[0].MemorySpeed != "9500" || status.Devices[0].BusLanes != "16" {
		t.Fatalf("extended device telemetry = %#v", status.Devices[0])
	}
}

func TestParseCrackStatusJSONRejectsMalformedOutput(t *testing.T) {
	if _, err := parseCrackStatusJSON([]byte("hashcat is still starting")); err == nil {
		t.Fatal("malformed status output was accepted")
	}
}

func TestUnavailableDeviceTelemetryRendersAsUnknown(t *testing.T) {
	status := &crackStatusView{Devices: []crackDeviceView{{Temperature: "-1"}}}
	if got := status.temperatureText(); got != "-" {
		t.Fatalf("temperatureText = %q, want -", got)
	}
	if got := unitValue("-1", " C"); got != "-" {
		t.Fatalf("unitValue = %q, want -", got)
	}
}

func TestRenderCrackJobIncludesTasksProgressDevicesAndResults(t *testing.T) {
	status := []byte(`{"session":"job-session","status":2,"target":"hash-file","progress":[25,100],"restore_point":24,"recovered_hashes":[1,2],"recovered_salts":[1,1],"rejected":3,"time_start":1700000000,"estimated_stop":1700000100,"guess":{"guess_mode":3,"guess_base":"?d?d","guess_base_offset":0,"guess_base_count":1},"devices":[{"device_id":1,"device_name":"GPU 0","device_type":"GPU","speed":2500000,"temp":61,"util":99,"fanspeed":45,"corespeed":2100,"memoryspeed":9000,"buslanes":16,"power":125000}]}`)
	recovered := []byte(`[{"hash":"hash-one","plaintext":"cGxhaW4tb25l"}]`)
	task := &clientpb.CrackTask{
		ID:               "task-id",
		HostUUID:         "station-id",
		Kind:             clientpb.CrackTaskKind_CRACK_TASK_CRACK,
		State:            clientpb.CrackTaskState_CRACK_TASK_RUNNING,
		UpdatedAt:        1700000002,
		LatestStatusJSON: status,
		RecoveredJSON:    recovered,
		ShardSkip:        100,
		ShardLimit:       50,
	}
	job := &clientpb.CrackJob{
		ID:        "job-id",
		CreatedAt: "2026-09-09T00:00:00Z",
		UpdatedAt: 1700000003,
		Status:    clientpb.CrackJobStatus_IN_PROGRESS,
		Keyspace:  "1000000",
		Tasks:     []*clientpb.CrackTask{task},
		Results: []*clientpb.CrackResult{{
			Hash:      "hash-one",
			Plaintext: []byte("plain-one"),
		}, {
			CredentialID: "credential-id",
			Hash:         "hash-two",
			Plaintext:    []byte("plain-two"),
		}},
	}

	output := renderCrackJob(job, settings.SliverDefault)
	for _, expected := range []string{
		"job-id", "1000000", "task-id", "station-id", "100+50", "25/100 (25.0%)",
		"2.50 MH/s", "61 C", "GPU 0", "credential-id", "hash-one", "plain-one", "hash-two", "plain-two",
		"Hashcat Statistics", "job-session", "hash-file", "mode=3", "base=?d?d", "1/2", "1/1",
		"2100 MHz", "9000 MHz", "16 lanes", "125000 mW", "2023-11-14T22:13:20Z", "2023-11-14T22:15:00Z",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("rendered job does not contain %q:\n%s", expected, output)
		}
	}
}

func TestRenderCrackJobsSummarizesExtendedFields(t *testing.T) {
	job := &clientpb.CrackJob{
		ID:        "job-id",
		CreatedAt: "2026-09-09T00:00:00Z",
		UpdatedAt: 1700000003,
		Status:    clientpb.CrackJobStatus_IN_PROGRESS,
		Keyspace:  "1000",
		Tasks:     []*clientpb.CrackTask{{ID: "task-id"}},
		Results:   []*clientpb.CrackResult{{Hash: "hash", Plaintext: []byte("plain")}},
	}
	output := renderCrackJobs([]*clientpb.CrackJob{job}, settings.SliverDefault)
	for _, expected := range []string{"job-id", "IN_PROGRESS", "1000", "1"} {
		if !strings.Contains(output, expected) {
			t.Errorf("rendered jobs do not contain %q:\n%s", expected, output)
		}
	}
}

func TestCrackJobResultCountUsesSummaryField(t *testing.T) {
	job := &clientpb.CrackJob{Results: []*clientpb.CrackResult{{Plaintext: []byte("not-a-summary-count")}}}
	field := findProtoField(job.ProtoReflect(), "ResultCount")
	want := "-"
	if field != nil {
		job.ProtoReflect().Set(field, protoreflect.ValueOfUint64(7))
		want = "7"
	}
	if got := crackJobResultCount(job); got != want {
		t.Fatalf("crackJobResultCount() = %q, want %q", got, want)
	}
}

func TestCrackJobResultsRenderAuthoritativePlaintextBytes(t *testing.T) {
	job := &clientpb.CrackJob{Results: []*clientpb.CrackResult{
		{Hash: "valid", Plaintext: []byte("line\ntext\x1b[31m")},
		{Hash: "binary", Plaintext: []byte{0xff, 0x00, 0x1b}},
		{Hash: "nul", Plaintext: []byte{'a', 0, 'b'}},
	}}
	results := crackJobResults(job)
	if len(results) != 3 {
		t.Fatalf("crackJobResults() = %#v", results)
	}
	if results[0].Plaintext != "linetext[31m" {
		t.Fatalf("valid UTF-8 plaintext = %q, want sanitized text", results[0].Plaintext)
	}
	if results[1].Plaintext != "0xff001b" {
		t.Fatalf("binary plaintext = %q, want 0xff001b", results[1].Plaintext)
	}
	if results[2].Plaintext != "0x610062" {
		t.Fatalf("NUL plaintext = %q, want 0x610062", results[2].Plaintext)
	}
}

func TestWatchCrackJobPrintsChangesAndStopsAtTerminalState(t *testing.T) {
	inProgress := &clientpb.CrackJob{ID: "job-id", Status: clientpb.CrackJobStatus_IN_PROGRESS}
	completed := &clientpb.CrackJob{ID: "job-id", Status: clientpb.CrackJobStatus_COMPLETED}
	responses := []*clientpb.CrackJob{inProgress, inProgress, completed}
	requests := 0
	fetch := func(_ context.Context, id string) (*clientpb.CrackJob, error) {
		if id != "job-id" {
			t.Fatalf("job ID = %q", id)
		}
		response := responses[requests]
		requests++
		return response, nil
	}
	updates := make([]clientpb.CrackJobStatus, 0)
	err := watchCrackJob(context.Background(), "job-id", time.Millisecond, fetch, func(job *clientpb.CrackJob) {
		updates = append(updates, job.Status)
	})
	if err != nil {
		t.Fatalf("watchCrackJob: %v", err)
	}
	if requests != 3 {
		t.Fatalf("requests = %d, want 3", requests)
	}
	if len(updates) != 2 || updates[0] != clientpb.CrackJobStatus_IN_PROGRESS || updates[1] != clientpb.CrackJobStatus_COMPLETED {
		t.Fatalf("updates = %#v", updates)
	}
}

func TestWatchCrackJobCancellationAndValidation(t *testing.T) {
	fetch := func(_ context.Context, _ string) (*clientpb.CrackJob, error) {
		return &clientpb.CrackJob{ID: "job-id", Status: clientpb.CrackJobStatus_IN_PROGRESS}, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := watchCrackJob(ctx, "job-id", time.Hour, fetch, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("watchCrackJob cancellation error = %v", err)
	}
	if err := watchCrackJob(context.Background(), "job-id", 0, fetch, nil); err == nil {
		t.Fatal("watchCrackJob accepted zero poll interval")
	}
}

func TestRecoveredResultsStripTerminalControlCharacters(t *testing.T) {
	results := parseRecoveredResults([]byte(`[{"hash":"hash\u001b[31m","plaintext":"cGxhaW4KdGV4dA=="}]`))
	if len(results) != 1 {
		t.Fatalf("results = %#v", results)
	}
	if strings.ContainsAny(results[0].Hash, "\x1b\n\r") || strings.ContainsAny(results[0].Plaintext, "\x1b\n\r") {
		t.Fatalf("control characters were not stripped: %#v", results[0])
	}
}

func TestRecoveredResultsRenderNULPlaintextAsHex(t *testing.T) {
	results := parseRecoveredResults([]byte(`[{"hash":"hash","plaintext":"YQBi"}]`))
	if len(results) != 1 {
		t.Fatalf("results = %#v", results)
	}
	if results[0].Plaintext != "0x610062" {
		t.Fatalf("plaintext = %q, want 0x610062", results[0].Plaintext)
	}
}
