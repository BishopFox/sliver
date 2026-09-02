package e2e

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	opforengine "github.com/sliverarmory/opfor"
)

func TestOPFORCallbackFixtureLoadsAndValidatesContract(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("fixtures", "opfor_callback.cna"))
	if err != nil {
		t.Fatalf("read OPFOR callback fixture: %v", err)
	}

	fixtureDirectory := t.TempDir()
	scriptPath := filepath.Join(fixtureDirectory, "opfor_callback.cna")
	if err := os.WriteFile(scriptPath, fixture, 0o600); err != nil {
		t.Fatalf("write OPFOR callback fixture: %v", err)
	}
	objectData := []byte{0x4c, 0x01, 0x00, 0x00}
	if err := os.WriteFile(filepath.Join(fixtureDirectory, "opfor_bof_fixture.x64.o"), objectData, 0o600); err != nil {
		t.Fatalf("write OPFOR object fixture: %v", err)
	}

	var observedModes []int32
	var transcript []string
	runtime, err := opforengine.New(
		opforengine.WithBOFPackByteOrder(opforengine.BOFPackLittleEndian),
		opforengine.WithAggressorBeaconTranscriptSink(
			opforengine.AggressorBeaconTranscriptSinkFunc(func(_ context.Context, record opforengine.AggressorBeaconTranscriptRecord) error {
				if record.Kind != opforengine.AggressorBeaconTranscriptLog || record.BeaconID.String() != "fixture-session" {
					return fmt.Errorf("unexpected OPFOR fixture transcript: %s %q", record.Kind, record.BeaconID.String())
				}
				transcript = append(transcript, record.Text.String())
				return nil
			}),
		),
		opforengine.WithAggressorSessionQueryProvider(
			opforengine.AggressorSessionQueryProviderFunc(func(_ context.Context, query opforengine.AggressorSessionQuery) (opforengine.Value, error) {
				if query.Kind != opforengine.AggressorSessionQueryBeaconArchitecture || query.SessionID.String() != "fixture-session" {
					return opforengine.Null(), fmt.Errorf("unexpected OPFOR fixture session query: %s %q", query.Kind, query.SessionID.String())
				}
				return opforengine.String("x64"), nil
			}),
		),
		opforengine.WithAggressorBeaconExecutionProvider(
			opforengine.AggressorBeaconExecutionProviderFunc(func(ctx context.Context, request opforengine.AggressorBeaconExecutionRequest) (opforengine.Value, error) {
				if request.Kind != opforengine.AggressorBeaconInlineExecute || request.BeaconID.String() != "fixture-session" {
					return opforengine.Null(), fmt.Errorf("unexpected OPFOR fixture execution request: %s %q", request.Kind, request.BeaconID.String())
				}
				content, ok := request.Content.Bytes()
				if !ok || !bytes.Equal(content, objectData) {
					return opforengine.Null(), fmt.Errorf("OPFOR fixture object = %x/binary:%v", content, ok)
				}
				if request.EntryPoint.String() != "go" || request.Callback == nil {
					return opforengine.Null(), fmt.Errorf("OPFOR fixture entry point/callback = %q/%v", request.EntryPoint.String(), request.Callback != nil)
				}
				packed, ok := request.PackedArguments.Bytes()
				if !ok || len(packed) != 8 {
					return opforengine.Null(), fmt.Errorf("OPFOR fixture packed arguments = %x/binary:%v", packed, ok)
				}
				mode := int32(binary.LittleEndian.Uint32(packed[:4]))
				sleepMilliseconds := int32(binary.LittleEndian.Uint32(packed[4:]))
				observedModes = append(observedModes, mode)

				switch mode {
				case 0:
					if sleepMilliseconds != 0 {
						return opforengine.Null(), fmt.Errorf("output sleep milliseconds = %d", sleepMilliseconds)
					}
					records := []struct {
						typeID int32
						data   []byte
					}{
						{typeID: 0x00, data: []byte("alpha")},
						{typeID: 0x0d, data: []byte("beta")},
						{typeID: 0x1e, data: []byte{0x41, 0x00, 0xff, 0x42}},
						{typeID: 0x20, data: []byte{'s', 'n', 'o', 'w', ':', 0xe2, 0x98, 0x83}},
						{typeID: 0x7f, data: nil},
					}
					for index, record := range records {
						if err := invokeOPFORFixtureDataCallback(ctx, request, record.typeID, record.data, index+1, index == len(records)-1); err != nil {
							return opforengine.Null(), err
						}
					}
					if err := invokeOPFORFixtureLifecycleCallback(ctx, request, "task_completed", nil); err != nil {
						return opforengine.Null(), err
					}
				case 1:
					if sleepMilliseconds != 0 {
						return opforengine.Null(), fmt.Errorf("partial sleep milliseconds = %d", sleepMilliseconds)
					}
					if err := invokeOPFORFixtureDataCallback(ctx, request, 0, []byte("before-error"), 1, true); err != nil {
						return opforengine.Null(), err
					}
					if err := invokeOPFORFixtureLifecycleCallback(ctx, request, "error", []byte("bofloader: execute BOF: BeaconOutput: invalid data 0x0 or length 1")); err != nil {
						return opforengine.Null(), err
					}
				case 2:
					if sleepMilliseconds != 750 {
						return opforengine.Null(), fmt.Errorf("timeout sleep milliseconds = %d", sleepMilliseconds)
					}
					if err := invokeOPFORFixtureLifecycleCallback(ctx, request, "error", []byte("context deadline exceeded")); err != nil {
						return opforengine.Null(), err
					}
				default:
					return opforengine.Null(), fmt.Errorf("unexpected OPFOR fixture mode %d", mode)
				}
				return opforengine.Null(), nil
			}),
		),
	)
	if err != nil {
		t.Fatalf("create OPFOR fixture runtime: %v", err)
	}
	t.Cleanup(func() {
		if err := runtime.Close(context.Background()); err != nil {
			t.Errorf("close OPFOR fixture runtime: %v", err)
		}
	})

	program, err := runtime.Compile(opforengine.NewSource(scriptPath, fixture))
	if err != nil {
		t.Fatalf("compile OPFOR callback fixture: %v", err)
	}
	if _, err := runtime.Load(context.Background(), program); err != nil {
		t.Fatalf("load OPFOR callback fixture: %v", err)
	}
	catalog, err := runtime.SnapshotAggressorCommandCatalog(opforengine.AggressorCommandBeacon)
	if err != nil {
		t.Fatalf("snapshot OPFOR fixture command catalog: %v", err)
	}
	commandNames := make([]string, len(catalog.Commands))
	for index, command := range catalog.Commands {
		commandNames[index] = command.Name
	}
	if want := []string{"opfor-e2e-output", "opfor-e2e-partial", "opfor-e2e-timeout"}; !reflect.DeepEqual(commandNames, want) {
		t.Fatalf("OPFOR fixture command catalog = %v, want %v", commandNames, want)
	}

	aliases := []struct {
		name      string
		arguments []opforengine.Value
	}{
		{name: "opfor-e2e-output", arguments: []opforengine.Value{opforengine.String("fixture-session")}},
		{name: "opfor-e2e-partial", arguments: []opforengine.Value{opforengine.String("fixture-session")}},
		{name: "opfor-e2e-timeout", arguments: []opforengine.Value{opforengine.String("fixture-session"), opforengine.Int(750)}},
	}
	for _, test := range aliases {
		bindings := runtime.Bindings(opforengine.BindingAlias, test.name)
		if len(bindings) != 1 || bindings[0].Callback == nil {
			t.Fatalf("OPFOR fixture alias %q bindings = %#v", test.name, bindings)
		}
		if _, err := bindings[0].Callback.Invoke(context.Background(), test.arguments...); err != nil {
			t.Fatalf("invoke OPFOR fixture alias %q: %v", test.name, err)
		}
	}
	if want := []int32{0, 1, 2}; !reflect.DeepEqual(observedModes, want) {
		t.Fatalf("OPFOR fixture modes = %v, want %v", observedModes, want)
	}
	if want := []string{"OPFOR_E2E_TYPED_CALLBACK_OK", "OPFOR_E2E_PARTIAL_CALLBACK_OK"}; !reflect.DeepEqual(transcript, want) {
		t.Fatalf("OPFOR fixture transcript = %v, want %v", transcript, want)
	}
}

func invokeOPFORFixtureDataCallback(
	ctx context.Context,
	request opforengine.AggressorBeaconExecutionRequest,
	typeID int32,
	data []byte,
	chunkNumber int,
	isFinal bool,
) error {
	information := opforengine.NewOrderedHash()
	kind := "output"
	if typeID == 0x0d {
		kind = "error"
	}
	information.Set("type", opforengine.String(kind))
	information.Set("type_id", opforengine.Int(typeID))
	information.Set("chunk_num", opforengine.Int(int32(chunkNumber)))
	information.Set("is_final", opforengine.Bool(isFinal))
	_, err := request.Callback.Invoke(
		ctx,
		request.BeaconID,
		opforengine.BinaryString(data),
		opforengine.HashValue(information),
	)
	return err
}

func invokeOPFORFixtureLifecycleCallback(
	ctx context.Context,
	request opforengine.AggressorBeaconExecutionRequest,
	kind string,
	result []byte,
) error {
	information := opforengine.NewOrderedHash()
	information.Set("type", opforengine.String(kind))
	_, err := request.Callback.Invoke(
		ctx,
		request.BeaconID,
		opforengine.BinaryString(result),
		opforengine.HashValue(information),
	)
	return err
}
