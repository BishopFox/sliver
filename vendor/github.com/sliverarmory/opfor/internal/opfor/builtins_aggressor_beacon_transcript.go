package opfor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func (r *Runtime) berror(ctx context.Context, invocation Invocation) (Value, error) {
	return r.publishAggressorBeaconTranscript(ctx, invocation, AggressorBeaconTranscriptError)
}

func (r *Runtime) blog(ctx context.Context, invocation Invocation) (Value, error) {
	return r.publishAggressorBeaconTranscript(ctx, invocation, AggressorBeaconTranscriptLog)
}

func (r *Runtime) blog2(ctx context.Context, invocation Invocation) (Value, error) {
	return r.publishAggressorBeaconTranscript(ctx, invocation, AggressorBeaconTranscriptLog2)
}

func (r *Runtime) binput(ctx context.Context, invocation Invocation) (Value, error) {
	return r.publishAggressorBeaconTranscript(ctx, invocation, AggressorBeaconTranscriptInput)
}

func (r *Runtime) btask(ctx context.Context, invocation Invocation) (Value, error) {
	return r.publishAggressorBeaconTranscript(ctx, invocation, AggressorBeaconTranscriptTask)
}

func (r *Runtime) btaskcompleted(ctx context.Context, invocation Invocation) (Value, error) {
	return r.publishAggressorBeaconTranscript(ctx, invocation, AggressorBeaconTranscriptTaskCompleted)
}

func (r *Runtime) bjoblog(ctx context.Context, invocation Invocation) (Value, error) {
	return r.publishAggressorBeaconTranscript(ctx, invocation, AggressorBeaconTranscriptJobLog)
}

func (r *Runtime) bjoberror(ctx context.Context, invocation Invocation) (Value, error) {
	return r.publishAggressorBeaconTranscript(ctx, invocation, AggressorBeaconTranscriptJobError)
}

func (r *Runtime) publishAggressorBeaconTranscript(
	ctx context.Context,
	invocation Invocation,
	kind AggressorBeaconTranscriptKind,
) (Value, error) {
	if err := requireAggressorBeaconTranscriptArguments(invocation, kind); err != nil {
		return Null(), err
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	if ctx == nil {
		ctx = context.Background()
	}

	// Resolve every live argument exactly once. The record contains Value
	// copies, never pass-by-name cells or the Invocation itself.
	values := invocation.Values()
	record := AggressorBeaconTranscriptRecord{
		Kind:      kind,
		RuntimeID: r.ID(),
		Script:    invocation.Script,
		Span:      invocation.Span,
		BeaconID:  values[0],
	}
	switch kind {
	case AggressorBeaconTranscriptError,
		AggressorBeaconTranscriptLog,
		AggressorBeaconTranscriptLog2,
		AggressorBeaconTranscriptInput:
		record.Text = values[1]
	case AggressorBeaconTranscriptTask:
		record.Text = values[1]
		if len(values) == 3 {
			record.HasMITREIDs = true
			record.RawMITREIDs = values[2].String()
		}
	case AggressorBeaconTranscriptTaskCompleted:
		record.TaskID = values[1]
	case AggressorBeaconTranscriptJobLog, AggressorBeaconTranscriptJobError:
		record.JobID = values[1]
		record.Text = values[2]
	}

	if !isNilInterface(r.aggressorBeaconTranscriptSink) {
		if err := r.aggressorBeaconTranscriptSink.PublishAggressorBeaconTranscript(ctx, record); err != nil {
			return Null(), preserveNativeBoundaryError(ctx, err)
		}
	} else {
		if r.stdout == nil {
			return Null(), errors.New("opfor: stdout writer is nil")
		}
		line := formatAggressorBeaconTranscriptRecord(record)
		written, err := io.WriteString(r.stdout, line)
		if err != nil {
			return Null(), preserveNativeBoundaryError(ctx, err)
		}
		if written != len(line) {
			return Null(), io.ErrShortWrite
		}
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	return Null(), nil
}

func requireAggressorBeaconTranscriptArguments(
	invocation Invocation,
	kind AggressorBeaconTranscriptKind,
) error {
	expected := 2
	if kind == AggressorBeaconTranscriptTask {
		if len(invocation.Arguments) == 2 || len(invocation.Arguments) == 3 {
			return nil
		}
		return fmt.Errorf("&%s: expected exactly 2 or 3 arguments, received %d",
			builtinName(invocation.Name), len(invocation.Arguments))
	}
	if kind == AggressorBeaconTranscriptJobLog || kind == AggressorBeaconTranscriptJobError {
		expected = 3
	}
	return requireExactAggressorClientArguments(invocation, expected)
}

// formatAggressorBeaconTranscriptRecord is a stable headless presentation,
// not a Sleep or Cobalt Strike serialization. Every string is Go-quoted so
// control characters, newlines, and invalid UTF-8 bytes cannot create a second
// terminal record and can be recovered without replacement.
func formatAggressorBeaconTranscriptRecord(record AggressorBeaconTranscriptRecord) string {
	var line strings.Builder
	line.WriteString("opfor.aggressor.beacon_transcript")
	appendAggressorBeaconTranscriptString(&line, "kind", string(record.Kind))
	appendAggressorBeaconTranscriptUint(&line, "runtime_id", uint64(record.RuntimeID))
	appendAggressorBeaconTranscriptUint(&line, "script", uint64(record.Script))
	appendAggressorBeaconTranscriptString(&line, "source", record.Span.Source)
	appendAggressorBeaconTranscriptInt(&line, "start_offset", record.Span.Start.Offset)
	appendAggressorBeaconTranscriptInt(&line, "start_line", record.Span.Start.Line)
	appendAggressorBeaconTranscriptInt(&line, "start_column", record.Span.Start.Column)
	appendAggressorBeaconTranscriptInt(&line, "end_offset", record.Span.End.Offset)
	appendAggressorBeaconTranscriptInt(&line, "end_line", record.Span.End.Line)
	appendAggressorBeaconTranscriptInt(&line, "end_column", record.Span.End.Column)
	appendAggressorBeaconTranscriptValue(&line, "beacon_id", record.BeaconID)

	switch record.Kind {
	case AggressorBeaconTranscriptError,
		AggressorBeaconTranscriptLog,
		AggressorBeaconTranscriptLog2,
		AggressorBeaconTranscriptInput:
		appendAggressorBeaconTranscriptValue(&line, "text", record.Text)
	case AggressorBeaconTranscriptTask:
		appendAggressorBeaconTranscriptValue(&line, "text", record.Text)
		appendAggressorBeaconTranscriptBool(&line, "mitre_ids_present", record.HasMITREIDs)
		appendAggressorBeaconTranscriptString(&line, "raw_mitre_ids", record.RawMITREIDs)
	case AggressorBeaconTranscriptTaskCompleted:
		appendAggressorBeaconTranscriptValue(&line, "task_id", record.TaskID)
	case AggressorBeaconTranscriptJobLog, AggressorBeaconTranscriptJobError:
		appendAggressorBeaconTranscriptValue(&line, "job_id", record.JobID)
		appendAggressorBeaconTranscriptValue(&line, "text", record.Text)
	}
	line.WriteByte('\n')
	return line.String()
}

func appendAggressorBeaconTranscriptValue(line *strings.Builder, name string, value Value) {
	appendAggressorBeaconTranscriptString(line, name+".kind", value.Kind().String())
	appendAggressorBeaconTranscriptBool(line, name+".binary", value.IsBinaryString())
	appendAggressorBeaconTranscriptBool(line, name+".tainted", value.IsTainted())
	appendAggressorBeaconTranscriptString(line, name, value.String())
}

func appendAggressorBeaconTranscriptString(line *strings.Builder, name, value string) {
	line.WriteByte(' ')
	line.WriteString(name)
	line.WriteByte('=')
	line.WriteString(strconv.Quote(value))
}

func appendAggressorBeaconTranscriptBool(line *strings.Builder, name string, value bool) {
	line.WriteByte(' ')
	line.WriteString(name)
	line.WriteByte('=')
	line.WriteString(strconv.FormatBool(value))
}

func appendAggressorBeaconTranscriptInt(line *strings.Builder, name string, value int) {
	line.WriteByte(' ')
	line.WriteString(name)
	line.WriteByte('=')
	line.WriteString(strconv.Itoa(value))
}

func appendAggressorBeaconTranscriptUint(line *strings.Builder, name string, value uint64) {
	line.WriteByte(' ')
	line.WriteString(name)
	line.WriteByte('=')
	line.WriteString(strconv.FormatUint(value, 10))
}
