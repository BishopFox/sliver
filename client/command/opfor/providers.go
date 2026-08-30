//go:build client

package opfor

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	opforengine "github.com/sliverarmory/opfor"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

const (
	beaconTaskPollInterval   = 250 * time.Millisecond
	bofRequestFallback       = 59 * time.Second
	bofRequestDeadlineMargin = time.Second

	bofOutputDefault = int32(0x00)
	bofOutputError   = int32(0x0d)
)

type opforRPC interface {
	GetSessions(context.Context, *commonpb.Empty, ...grpc.CallOption) (*clientpb.Sessions, error)
	GetBeacons(context.Context, *commonpb.Empty, ...grpc.CallOption) (*clientpb.Beacons, error)
	GetBeaconTaskContent(context.Context, *clientpb.BeaconTask, ...grpc.CallOption) (*clientpb.BeaconTask, error)
	CallExtension(context.Context, *sliverpb.CallExtensionReq, ...grpc.CallOption) (*sliverpb.CallExtension, error)
}

var _ opforRPC = (rpcpb.SliverRPCClient)(nil)

type resolvedTarget struct {
	session *clientpb.Session
	beacon  *clientpb.Beacon
}

func (target resolvedTarget) id() string {
	if target.session != nil {
		return target.session.ID
	}
	if target.beacon != nil {
		return target.beacon.ID
	}
	return ""
}

func (target resolvedTarget) capabilities() uint64 {
	if target.session != nil {
		return target.session.Capabilities
	}
	if target.beacon != nil {
		return target.beacon.Capabilities
	}
	return 0
}

func (target resolvedTarget) arch() string {
	if target.session != nil {
		return target.session.Arch
	}
	if target.beacon != nil {
		return target.beacon.Arch
	}
	return ""
}

func (manager *Manager) rpc() (opforRPC, error) {
	if manager == nil || manager.client == nil || manager.client.Rpc == nil {
		return nil, errors.New("opfor: Sliver RPC client is unavailable")
	}
	return manager.client.Rpc, nil
}

func (manager *Manager) resolveTarget(ctx context.Context, id string) (resolvedTarget, error) {
	if strings.TrimSpace(id) == "" {
		return resolvedTarget{}, errors.New("opfor: target ID is empty")
	}
	rpc, err := manager.rpc()
	if err != nil {
		return resolvedTarget{}, err
	}
	sessions, err := rpc.GetSessions(ctx, &commonpb.Empty{})
	if err != nil {
		return resolvedTarget{}, fmt.Errorf("opfor: list sessions: %w", err)
	}
	for _, session := range sessions.GetSessions() {
		if session != nil && session.ID == id {
			return resolvedTarget{session: session}, nil
		}
	}
	beacons, err := rpc.GetBeacons(ctx, &commonpb.Empty{})
	if err != nil {
		return resolvedTarget{}, fmt.Errorf("opfor: list beacons: %w", err)
	}
	for _, beacon := range beacons.GetBeacons() {
		if beacon != nil && beacon.ID == id {
			return resolvedTarget{beacon: beacon}, nil
		}
	}
	return resolvedTarget{}, fmt.Errorf("opfor: target %q was not found", id)
}

func (manager *Manager) executeBeacon(
	ctx context.Context,
	request opforengine.AggressorBeaconExecutionRequest,
) (opforengine.Value, error) {
	targetID, call, err := manager.prepareBeaconExecution(ctx, request)
	if err != nil {
		return opforengine.Null(), err
	}

	rpc, err := manager.rpc()
	if err != nil {
		return opforengine.Null(), err
	}
	response, err := rpc.CallExtension(ctx, call)
	if err != nil {
		executionErr := fmt.Errorf("opfor: execute BOF on %s: %w", targetID, err)
		return opforengine.Null(), notifyBOFLifecycleError(ctx, request, "", executionErr)
	}
	taskID := response.GetResponse().GetTaskID()
	response, err = manager.resolveCallExtensionResponse(ctx, rpc, response)
	if err != nil {
		return opforengine.Null(), notifyBOFLifecycleError(ctx, request, taskID, err)
	}
	if taskID == "" {
		taskID = response.GetResponse().GetTaskID()
	}

	return manager.completeBeaconExecution(ctx, request, taskID, response)
}

func (manager *Manager) prepareBeaconExecution(
	ctx context.Context,
	request opforengine.AggressorBeaconExecutionRequest,
) (string, *sliverpb.CallExtensionReq, error) {
	if request.Kind != opforengine.AggressorBeaconInlineExecute {
		return "", nil, fmt.Errorf("opfor: unsupported Beacon execution function %q", request.Name)
	}
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}
	targetID := request.BeaconID.String()
	target, err := manager.resolveTarget(ctx, targetID)
	if err != nil {
		return "", nil, err
	}
	if _, err := beaconArchitecture(target.arch()); err != nil {
		return "", nil, err
	}
	if target.capabilities()&sliverpb.CapabilityBOFV1 == 0 {
		return "", nil, fmt.Errorf("opfor: target %q does not advertise the bof_v1 capability", targetID)
	}

	bofData, ok := request.Content.Bytes()
	if !ok {
		return "", nil, errors.New("opfor: beacon_inline_execute BOF content is not a byte string")
	}
	packed, ok := request.PackedArguments.Bytes()
	if !ok {
		packed = []byte(request.PackedArguments.String())
	}
	arguments, err := prefixBOFArguments(packed)
	if err != nil {
		return "", nil, err
	}
	digest := sha256.Sum256(bofData)
	return targetID, &sliverpb.CallExtensionReq{
		Name:    hex.EncodeToString(digest[:]),
		Args:    arguments,
		Export:  request.EntryPoint.String(),
		BOFData: bofData,
		IsBOF:   true,
		// Older implants ignore this additive request bit and return legacy
		// flattened Output, which bofOutputRecords continues to accept.
		WantBOFOutputs: true,
		Request:        targetRequest(ctx, target),
	}, nil
}

func (manager *Manager) completeBeaconExecution(
	ctx context.Context,
	request opforengine.AggressorBeaconExecutionRequest,
	taskID string,
	response *sliverpb.CallExtension,
) (opforengine.Value, error) {
	records := bofOutputRecords(response)
	var executionErr error
	if response.GetResponse().GetErr() != "" {
		executionErr = fmt.Errorf("opfor: BOF execution failed: %s", response.GetResponse().GetErr())
	}
	if request.Callback == nil {
		manager.renderBOFOutputs(records)
		if executionErr != nil {
			return opforengine.Null(), executionErr
		}
		return opforengine.Null(), nil
	}

	for index, record := range records {
		chunkNumber := index + 1
		isFinal := index == len(records)-1
		if err := invokeBOFOutputCallback(ctx, request, record, chunkNumber, isFinal); err != nil {
			callbackErr := fmt.Errorf("opfor: BOF result callback chunk %d: %w", chunkNumber, err)
			if executionErr != nil {
				return opforengine.Null(), errors.Join(executionErr, callbackErr)
			}
			return opforengine.Null(), callbackErr
		}
	}
	if executionErr != nil {
		return opforengine.Null(), notifyBOFLifecycleError(ctx, request, taskID, executionErr)
	}
	if taskID != "" {
		if err := invokeBOFLifecycleCallback(ctx, request, "task_completed", taskID, nil); err != nil {
			return opforengine.Null(), fmt.Errorf("opfor: BOF task_completed callback: %w", err)
		}
	} else if len(records) == 0 {
		if err := invokeBOFLifecycleCallback(ctx, request, "success", "", nil); err != nil {
			return opforengine.Null(), fmt.Errorf("opfor: BOF success callback: %w", err)
		}
	}
	return opforengine.Null(), nil
}

// bofOutputRecords prefers the typed wire representation. Older servers and
// implants populate only Output, which remains one default-channel record.
func bofOutputRecords(response *sliverpb.CallExtension) []*sliverpb.BOFOutput {
	if response == nil {
		return nil
	}
	if records := response.GetBOFOutputs(); len(records) != 0 {
		return records
	}
	if len(response.GetOutput()) == 0 {
		return nil
	}
	return []*sliverpb.BOFOutput{{
		Type: bofOutputDefault,
		Data: response.GetOutput(),
	}}
}

// The callback is deliberately multi-shot. Data records retain their exact
// byte strings and raw Cobalt output channel. chunk_num is one-based and
// is_final marks the final data record, independently of task lifecycle events.
func invokeBOFOutputCallback(
	ctx context.Context,
	request opforengine.AggressorBeaconExecutionRequest,
	record *sliverpb.BOFOutput,
	chunkNumber int,
	isFinal bool,
) error {
	if request.Callback == nil {
		return nil
	}
	if record == nil {
		record = &sliverpb.BOFOutput{}
	}
	kind := "output"
	if record.GetType() == bofOutputError {
		kind = "error"
	}
	information := opforengine.NewOrderedHash()
	information.Set("type", opforengine.String(kind))
	information.Set("type_id", opforengine.Int(record.GetType()))
	information.Set("chunk_num", opforengine.Int(int32(chunkNumber)))
	information.Set("is_final", opforengine.Bool(isFinal))
	_, err := request.Callback.Invoke(
		ctx,
		request.BeaconID,
		opforengine.BinaryString(record.GetData()),
		opforengine.HashValue(information),
	)
	return err
}

func invokeBOFLifecycleCallback(
	ctx context.Context,
	request opforengine.AggressorBeaconExecutionRequest,
	kind string,
	taskID string,
	result []byte,
) error {
	if request.Callback == nil {
		return nil
	}
	information := opforengine.NewOrderedHash()
	information.Set("type", opforengine.String(kind))
	if taskID != "" {
		information.Set("taskid", opforengine.String(taskID))
	}
	_, err := request.Callback.Invoke(
		ctx,
		request.BeaconID,
		opforengine.BinaryString(result),
		opforengine.HashValue(information),
	)
	return err
}

func notifyBOFLifecycleError(
	ctx context.Context,
	request opforengine.AggressorBeaconExecutionRequest,
	taskID string,
	executionErr error,
) error {
	if request.Callback == nil {
		return executionErr
	}
	if ctx != nil && ctx.Err() != nil {
		return executionErr
	}
	callbackErr := invokeBOFLifecycleCallback(
		ctx, request, "error", taskID, []byte(executionErr.Error()),
	)
	if callbackErr != nil {
		return errors.Join(executionErr, fmt.Errorf("opfor: BOF error callback: %w", callbackErr))
	}
	return executionErr
}

func (manager *Manager) renderBOFOutputs(records []*sliverpb.BOFOutput) {
	wrote := false
	endedWithNewline := true
	for _, record := range records {
		if record == nil || len(record.GetData()) == 0 {
			continue
		}
		text := safeBOFText(record.GetData())
		if text == "" {
			continue
		}
		wrote = true
		endedWithNewline = strings.HasSuffix(text, "\n")
		if record.GetType() == bofOutputError {
			manager.output.PrintErrorf("%s", text)
		} else {
			manager.output.Printf("%s", text)
		}
	}
	if wrote && !endedWithNewline {
		manager.output.Printf("\n")
	}
}

// safeBOFText prevents untrusted native output from injecting terminal control
// sequences. Valid printable UTF-8 plus tab/newline/carriage return is retained;
// invalid bytes and other control bytes are rendered as hexadecimal escapes.
func safeBOFText(data []byte) string {
	var result strings.Builder
	result.Grow(len(data))
	for len(data) != 0 {
		r, size := utf8.DecodeRune(data)
		if r == utf8.RuneError && size == 1 {
			fmt.Fprintf(&result, `\x%02x`, data[0])
			data = data[1:]
			continue
		}
		if r == '\n' || r == '\r' || r == '\t' || unicode.IsPrint(r) {
			result.WriteRune(r)
		} else {
			for _, value := range data[:size] {
				fmt.Fprintf(&result, `\x%02x`, value)
			}
		}
		data = data[size:]
	}
	return result.String()
}

func prefixBOFArguments(packed []byte) ([]byte, error) {
	if uint64(len(packed)) > math.MaxUint32 {
		return nil, errors.New("opfor: BOF argument buffer exceeds uint32 length")
	}
	arguments := make([]byte, 4+len(packed))
	binary.LittleEndian.PutUint32(arguments, uint32(len(packed)))
	copy(arguments[4:], packed)
	return arguments, nil
}

func targetRequest(ctx context.Context, target resolvedTarget) *commonpb.Request {
	request := &commonpb.Request{Timeout: int64(bofRequestTimeout(ctx))}
	if target.session != nil {
		request.SessionID = target.session.ID
		request.Async = false
	} else if target.beacon != nil {
		request.BeaconID = target.beacon.ID
		request.Async = true
	}
	return request
}

func bofRequestTimeout(ctx context.Context) time.Duration {
	if ctx == nil {
		return bofRequestFallback
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return bofRequestFallback
	}
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return time.Nanosecond
	}
	if remaining > bofRequestDeadlineMargin {
		remaining -= bofRequestDeadlineMargin
	}
	return remaining
}

func (manager *Manager) resolveCallExtensionResponse(
	ctx context.Context,
	rpc opforRPC,
	response *sliverpb.CallExtension,
) (*sliverpb.CallExtension, error) {
	if response == nil {
		return nil, errors.New("opfor: BOF execution returned an empty response")
	}
	if response.GetResponse().GetErr() != "" || !response.GetResponse().GetAsync() {
		return response, nil
	}
	taskID := response.GetResponse().GetTaskID()
	if taskID == "" {
		return nil, errors.New("opfor: asynchronous BOF execution returned no task ID")
	}

	ticker := time.NewTicker(beaconTaskPollInterval)
	defer ticker.Stop()
	for {
		task, err := rpc.GetBeaconTaskContent(ctx, &clientpb.BeaconTask{ID: taskID})
		if err != nil {
			return nil, fmt.Errorf("opfor: fetch beacon task %s: %w", taskID, err)
		}
		state := strings.ToLower(task.GetState())
		switch state {
		case "completed":
			if len(task.Response) == 0 {
				return nil, fmt.Errorf("opfor: beacon task %s returned an empty response", taskID)
			}
			completed := &sliverpb.CallExtension{}
			if err := proto.Unmarshal(task.Response, completed); err != nil {
				return nil, fmt.Errorf("opfor: decode beacon task %s response: %w", taskID, err)
			}
			if completed.Response == nil {
				completed.Response = &commonpb.Response{}
			}
			if completed.Response.TaskID == "" {
				completed.Response.TaskID = taskID
			}
			return completed, nil
		case "failed", "canceled", "cancelled":
			return nil, fmt.Errorf("opfor: beacon task %s %s", taskID, state)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (manager *Manager) publishTranscript(
	_ context.Context,
	record opforengine.AggressorBeaconTranscriptRecord,
) error {
	target := record.BeaconID.String()
	text := record.Text.String()
	prefix := ""
	if target != "" {
		prefix = "[" + target + "] "
	}
	switch record.Kind {
	case opforengine.AggressorBeaconTranscriptError,
		opforengine.AggressorBeaconTranscriptJobError:
		manager.output.PrintErrorf("%s%s\n", prefix, text)
	case opforengine.AggressorBeaconTranscriptTask,
		opforengine.AggressorBeaconTranscriptTaskCompleted:
		manager.output.PrintInfof("%s%s\n", prefix, text)
	default:
		manager.output.Printf("%s%s\n", prefix, text)
	}
	return nil
}
