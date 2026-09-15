package crack

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/charmbracelet/x/ansi"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestCrackJobCommandsAreRegistered(t *testing.T) {
	root := Commands(nil)[0]
	jobs, _, err := root.Find([]string{"jobs"})
	if err != nil || jobs == root {
		t.Fatalf("find crack jobs: command=%v error=%v", jobs, err)
	}
	top, _, err := root.Find([]string{"top"})
	if err != nil || top == root {
		t.Fatalf("find crack top: command=%v error=%v", top, err)
	}
	if top.Use != "top" {
		t.Fatalf("crack top use = %q, want top", top.Use)
	}
	if err := top.Args(top, nil); err != nil {
		t.Fatalf("crack top rejected no arguments: %v", err)
	}
	if err := top.Args(top, []string{"unexpected"}); err == nil {
		t.Fatal("crack top accepted a positional argument")
	}
	pollInterval := top.Flags().Lookup("poll-interval")
	if pollInterval == nil {
		t.Fatal("crack top is missing --poll-interval")
	}
	if got, err := top.Flags().GetDuration("poll-interval"); err != nil || got != time.Second {
		t.Fatalf("crack top --poll-interval = %s, %v; want %s", got, err, time.Second)
	}
	if top.PersistentFlags().Lookup("poll-interval") != nil || root.PersistentFlags().Lookup("poll-interval") != nil {
		t.Fatal("--poll-interval must remain local to crack top")
	}
	if top.InheritedFlags().Lookup("timeout") == nil {
		t.Fatal("crack top must inherit --timeout")
	}
	job, _, err := root.Find([]string{"job", "job-id"})
	if err != nil || job == root {
		t.Fatalf("find crack job: command=%v error=%v", job, err)
	}
	if job.Flags().Lookup("watch") == nil || job.Flags().Lookup("poll-interval") == nil {
		t.Fatal("crack job is missing watch flags")
	}
	if job.Use != "job [id]" {
		t.Fatalf("crack job use = %q, want %q", job.Use, "job [id]")
	}
	if job.ValidArgsFunction == nil {
		t.Fatal("crack job is missing ID completion")
	}
	if err := job.Args(job, nil); err != nil {
		t.Fatalf("crack job rejected interactive selection: %v", err)
	}
	if err := job.Args(job, []string{"job-id"}); err != nil {
		t.Fatalf("crack job rejected one job ID: %v", err)
	}
	if err := job.Args(job, []string{"first", "second"}); err == nil {
		t.Fatal("crack job accepted more than one job ID")
	}
}

func TestResolveCrackJobIDUsesExplicitIDWithoutListing(t *testing.T) {
	listed := false
	selected := false
	jobID, err := resolveCrackJobID([]string{"  full-job-id  "}, func() (*clientpb.CrackJobs, error) {
		listed = true
		return nil, nil
	}, func([]*clientpb.CrackJob) (string, error) {
		selected = true
		return "", nil
	})
	if err != nil {
		t.Fatalf("resolveCrackJobID: %v", err)
	}
	if jobID != "full-job-id" {
		t.Fatalf("job ID = %q, want full-job-id", jobID)
	}
	if listed || selected {
		t.Fatalf("explicit ID unexpectedly listed=%v selected=%v", listed, selected)
	}
}

func TestResolveCrackJobIDSelectsFromListedJobs(t *testing.T) {
	wantJobs := []*clientpb.CrackJob{{ID: "first"}, {ID: "selected-full-id"}}
	listCalls := 0
	selectCalls := 0
	jobID, err := resolveCrackJobID(nil, func() (*clientpb.CrackJobs, error) {
		listCalls++
		return &clientpb.CrackJobs{Jobs: wantJobs}, nil
	}, func(jobs []*clientpb.CrackJob) (string, error) {
		selectCalls++
		if len(jobs) != len(wantJobs) || jobs[1] != wantJobs[1] {
			t.Fatalf("selector jobs = %#v, want %#v", jobs, wantJobs)
		}
		return "selected-full-id", nil
	})
	if err != nil {
		t.Fatalf("resolveCrackJobID: %v", err)
	}
	if jobID != "selected-full-id" {
		t.Fatalf("job ID = %q, want selected-full-id", jobID)
	}
	if listCalls != 1 || selectCalls != 1 {
		t.Fatalf("list calls = %d, select calls = %d; want 1 each", listCalls, selectCalls)
	}
}

func TestResolveCrackJobIDHandlesEmptyAndInvalidSelections(t *testing.T) {
	selectorCalled := false
	for _, listedJobs := range [][]*clientpb.CrackJob{nil, {nil, {ID: "   "}}} {
		_, err := resolveCrackJobID(nil, func() (*clientpb.CrackJobs, error) {
			return &clientpb.CrackJobs{Jobs: listedJobs}, nil
		}, func([]*clientpb.CrackJob) (string, error) {
			selectorCalled = true
			return "unused", nil
		})
		if !errors.Is(err, errNoCrackJobs) {
			t.Fatalf("empty job error = %v, want %v", err, errNoCrackJobs)
		}
	}
	if selectorCalled {
		t.Fatal("selector called with no jobs")
	}

	_, err := resolveCrackJobID(nil, func() (*clientpb.CrackJobs, error) {
		return &clientpb.CrackJobs{Jobs: []*clientpb.CrackJob{{ID: "job-id"}}}, nil
	}, func([]*clientpb.CrackJob) (string, error) {
		return "   ", nil
	})
	if err == nil || !strings.Contains(err.Error(), "no crack job selected") {
		t.Fatalf("blank selection error = %v", err)
	}

	_, err = resolveCrackJobID([]string{"   "}, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "job ID cannot be empty") {
		t.Fatalf("blank explicit ID error = %v", err)
	}
}

func TestResolveCrackJobIDPropagatesUserAbort(t *testing.T) {
	_, err := resolveCrackJobID(nil, func() (*clientpb.CrackJobs, error) {
		return &clientpb.CrackJobs{Jobs: []*clientpb.CrackJob{{ID: "job-id"}}}, nil
	}, func([]*clientpb.CrackJob) (string, error) {
		return "", forms.ErrUserAborted
	})
	if !errors.Is(err, forms.ErrUserAborted) {
		t.Fatalf("selection error = %v, want user abort", err)
	}
}

//nolint:gocyclo // One fixture verifies the complete newest-status telemetry contract and all presentation helpers.
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

	rawOutput := renderCrackJob(job, settings.SliverDefault)
	output := ansi.Strip(rawOutput)
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
	for _, styled := range []string{
		console.StyleBoldPrimary.Render("Crack Job"),
		console.StyleBoldWarning.Render(clientpb.CrackJobStatus_IN_PROGRESS.String()),
		console.StyleBoldPrimary.Render("Tasks"),
		console.StyleBoldWarning.Render(clientpb.CrackTaskState_CRACK_TASK_RUNNING.String()),
		console.StyleBoldPrimary.Render("Hashcat Statistics"),
		console.StyleBoldPrimary.Render("Devices"),
		console.StyleBoldSuccess.Render("Recovered Credentials"),
	} {
		if !strings.Contains(rawOutput, styled) {
			t.Fatalf("rendered job is missing Lip Gloss styling %q:\n%s", styled, rawOutput)
		}
	}
	if strings.Contains(rawOutput, console.StylePrimary.Render("task-id")) {
		t.Fatalf("rendered task ID should use the default foreground color:\n%s", rawOutput)
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
	rawOutput := renderCrackJobs([]*clientpb.CrackJob{job}, settings.SliverDefault)
	output := ansi.Strip(rawOutput)
	for _, expected := range []string{"job-id", "IN_PROGRESS", "1000", "1"} {
		if !strings.Contains(output, expected) {
			t.Errorf("rendered jobs do not contain %q:\n%s", expected, output)
		}
	}
	if !strings.Contains(rawOutput, console.StyleBoldPrimary.Render("job-id")) ||
		!strings.Contains(rawOutput, console.StyleBoldWarning.Render(clientpb.CrackJobStatus_IN_PROGRESS.String())) {
		t.Fatalf("rendered jobs are missing Lip Gloss ID or status styling:\n%s", rawOutput)
	}
}

func TestCrackJobStatusStyleUsesSemanticPalette(t *testing.T) {
	tests := []struct {
		status string
		want   console.TextStyle
	}{
		{status: "IN_PROGRESS", want: console.StyleBoldWarning},
		{status: "CRACK_TASK_RUNNING", want: console.StyleBoldWarning},
		{status: "CRACK_TASK_QUEUED", want: console.StyleBold},
		{status: "CRACK_TASK_LEASED", want: console.StyleBoldWarning},
		{status: "COMPLETED", want: console.StyleBoldSuccess},
		{status: "CRACK_TASK_COMPLETED", want: console.StyleBoldSuccess},
		{status: "CRACK_TASK_FAILED", want: console.StyleBoldDanger},
		{status: "CANCELLED", want: console.StyleBoldGray},
		{status: "CANCELED", want: console.StyleBoldGray},
		{status: "PAUSED", want: console.StyleBoldGray},
		{status: "CRACK_TASK_CANCELLED", want: console.StyleBoldGray},
		{status: "UNKNOWN", want: console.StyleBold},
	}

	for _, test := range tests {
		t.Run(test.status, func(t *testing.T) {
			got := crackJobStatusStyle(test.status)
			if got.GetBold() != test.want.GetBold() {
				t.Fatalf("bold = %v, want %v", got.GetBold(), test.want.GetBold())
			}
			gotColor := got.GetForeground()
			wantColor := test.want.GetForeground()
			if (gotColor == nil) != (wantColor == nil) {
				t.Fatalf("foreground = %v, want %v", gotColor, wantColor)
			}
			if gotColor == nil {
				return
			}
			gotR, gotG, gotB, gotA := gotColor.RGBA()
			wantR, wantG, wantB, wantA := wantColor.RGBA()
			if gotR != wantR || gotG != wantG || gotB != wantB || gotA != wantA {
				t.Fatalf("foreground RGBA = (%d,%d,%d,%d), want (%d,%d,%d,%d)", gotR, gotG, gotB, gotA, wantR, wantG, wantB, wantA)
			}
		})
	}
}

func TestRenderCrackJobStylesErrors(t *testing.T) {
	rawOutput := renderCrackJob(&clientpb.CrackJob{
		ID:     "job-id",
		Status: clientpb.CrackJobStatus_FAILED,
		Err:    "worker failed",
	}, settings.SliverDefault)
	for _, styled := range []string{
		console.StyleBoldDanger.Render(clientpb.CrackJobStatus_FAILED.String()),
		console.StyleBoldDanger.Render("Error"),
		console.StyleDanger.Render("worker failed"),
	} {
		if !strings.Contains(rawOutput, styled) {
			t.Fatalf("rendered error is missing Lip Gloss styling %q:\n%s", styled, rawOutput)
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
