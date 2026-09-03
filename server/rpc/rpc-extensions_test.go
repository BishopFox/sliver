package rpc

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
	uuid "uuid"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRequireCallExtensionBOFCapability(t *testing.T) {
	sessionLookup := capabilityMap(map[string]uint64{
		"capable-session": sliverpb.CapabilityBOFV1,
		"old-session":     0,
	})
	beaconLookup := capabilityMap(map[string]uint64{
		"capable-beacon": sliverpb.CapabilityBOFV1,
		"old-beacon":     0,
	})

	tests := []struct {
		name        string
		req         *sliverpb.CallExtensionReq
		wantErr     error
		wantCode    codes.Code
		wantMessage string
	}{
		{
			name: "legacy extension bypasses capability gate",
			req:  &sliverpb.CallExtensionReq{},
		},
		{
			name:     "missing request",
			req:      &sliverpb.CallExtensionReq{IsBOF: true},
			wantErr:  ErrMissingRequestField,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "capable session",
			req: &sliverpb.CallExtensionReq{IsBOF: true, Request: &commonpb.Request{
				SessionID: "capable-session",
			}},
		},
		{
			name: "old session fails closed",
			req: &sliverpb.CallExtensionReq{IsBOF: true, Request: &commonpb.Request{
				SessionID: "old-session",
			}},
			wantCode:    codes.FailedPrecondition,
			wantMessage: "target session does not support built-in BOF execution",
		},
		{
			name: "unknown session",
			req: &sliverpb.CallExtensionReq{IsBOF: true, Request: &commonpb.Request{
				SessionID: "missing-session",
			}},
			wantErr:  ErrInvalidSessionID,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "capable beacon",
			req: &sliverpb.CallExtensionReq{IsBOF: true, Request: &commonpb.Request{
				Async:    true,
				BeaconID: "capable-beacon",
			}},
		},
		{
			name: "old beacon fails closed",
			req: &sliverpb.CallExtensionReq{IsBOF: true, Request: &commonpb.Request{
				Async:    true,
				BeaconID: "old-beacon",
			}},
			wantCode:    codes.FailedPrecondition,
			wantMessage: "target beacon does not support built-in BOF execution",
		},
		{
			name: "unknown beacon",
			req: &sliverpb.CallExtensionReq{IsBOF: true, Request: &commonpb.Request{
				Async:    true,
				BeaconID: "missing-beacon",
			}},
			wantErr:  ErrInvalidBeaconID,
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := requireCallExtensionBOFCapabilityWithLookups(tt.req, sessionLookup, beaconLookup)
			if tt.wantErr == nil && tt.wantCode == codes.OK {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if got := status.Code(err); got != tt.wantCode {
				t.Fatalf("expected code %v, got %v (%v)", tt.wantCode, got, err)
			}
			if tt.wantMessage != "" && !strings.Contains(err.Error(), tt.wantMessage) {
				t.Fatalf("expected error containing %q, got %v", tt.wantMessage, err)
			}
		})
	}
}

func TestCallExtensionRejectsUnsupportedSessionBeforeDispatch(t *testing.T) {
	session := core.NewSession(core.NewImplantConnection("test", "n/a"))
	core.Sessions.Add(session)
	t.Cleanup(func() {
		core.Sessions.Remove(session.ID)
	})

	_, err := (&Server{}).CallExtension(context.Background(), &sliverpb.CallExtensionReq{
		IsBOF: true,
		Request: &commonpb.Request{
			SessionID: session.ID,
		},
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", err)
	}
}

func TestCallExtensionRejectsUnsupportedBeaconBeforeTasking(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "call-extension-capabilities.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := testDB.AutoMigrate(&models.Beacon{}, &models.BeaconTask{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	originalDB := db.Client
	db.Client = testDB
	t.Cleanup(func() {
		db.Client = originalDB
		sqlDB, dbErr := testDB.DB()
		if dbErr == nil {
			_ = sqlDB.Close()
		}
	})

	beaconID := uuid.NewV4()
	if err := testDB.Create(&models.Beacon{ID: models.UUIDFrom(beaconID)}).Error; err != nil {
		t.Fatalf("create beacon: %v", err)
	}

	_, err = (&Server{}).CallExtension(context.Background(), &sliverpb.CallExtensionReq{
		IsBOF: true,
		Request: &commonpb.Request{
			Async:    true,
			BeaconID: beaconID.String(),
		},
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", err)
	}

	var taskCount int64
	if err := testDB.Model(&models.BeaconTask{}).Count(&taskCount).Error; err != nil {
		t.Fatalf("count beacon tasks: %v", err)
	}
	if taskCount != 0 {
		t.Fatalf("expected no beacon tasks, got %d", taskCount)
	}
}

func TestCallExtensionPreservesTypedBOFPartialResponse(t *testing.T) {
	connection := core.NewImplantConnection("test", "n/a")
	session := core.NewSession(connection)
	session.Capabilities = sliverpb.CapabilityBOFV1
	core.Sessions.Add(session)
	t.Cleanup(func() {
		core.Sessions.Remove(session.ID)
	})

	want := &sliverpb.CallExtension{
		Output: []byte("partial legacy output"),
		BOFOutputs: []*sliverpb.BOFOutput{
			{Type: 0, Data: []byte("before failure")},
			{Type: 1, Data: []byte{0x00, 0xff, 0x01}},
		},
		Response: &commonpb.Response{Err: "BOF failed after producing output"},
	}
	responseDone := respondToCallExtension(t, connection, want)

	got, err := (&Server{}).CallExtension(context.Background(), &sliverpb.CallExtensionReq{
		IsBOF:          true,
		WantBOFOutputs: true,
		Request: &commonpb.Request{
			SessionID: session.ID,
			Timeout:   int64(2 * time.Second),
		},
	})
	if responseErr := <-responseDone; responseErr != nil {
		t.Fatal(responseErr)
	}
	if err != nil {
		t.Fatalf("unexpected gRPC error: %v", err)
	}
	if !proto.Equal(got, want) {
		t.Fatalf("typed BOF response mismatch:\n got: %v\nwant: %v", got, want)
	}
}

func TestCallExtensionPreservesLegacyErrorSemantics(t *testing.T) {
	tests := []struct {
		name           string
		isBOF          bool
		wantBOFOutputs bool
	}{
		{
			name:  "legacy BOF response",
			isBOF: true,
		},
		{
			name:           "non-BOF response",
			wantBOFOutputs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connection := core.NewImplantConnection("test", "n/a")
			session := core.NewSession(connection)
			session.Capabilities = sliverpb.CapabilityBOFV1
			core.Sessions.Add(session)
			t.Cleanup(func() {
				core.Sessions.Remove(session.ID)
			})

			responseDone := respondToCallExtension(t, connection, &sliverpb.CallExtension{
				Output:   []byte("partial output"),
				Response: &commonpb.Response{Err: "extension failed"},
			})
			got, err := (&Server{}).CallExtension(context.Background(), &sliverpb.CallExtensionReq{
				IsBOF:          tt.isBOF,
				WantBOFOutputs: tt.wantBOFOutputs,
				Request: &commonpb.Request{
					SessionID: session.ID,
					Timeout:   int64(2 * time.Second),
				},
			})
			if responseErr := <-responseDone; responseErr != nil {
				t.Fatal(responseErr)
			}
			if got != nil {
				t.Fatalf("expected response to be discarded, got %v", got)
			}
			if status.Code(err) != codes.FailedPrecondition {
				t.Fatalf("expected FailedPrecondition, got %v", err)
			}
		})
	}
}

func respondToCallExtension(t *testing.T, connection *core.ImplantConnection, response *sliverpb.CallExtension) <-chan error {
	t.Helper()

	responseData, err := proto.Marshal(response)
	if err != nil {
		t.Fatalf("marshal call extension response: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		request := <-connection.Send
		connection.RespMutex.RLock()
		responseChannel, ok := connection.Resp[request.ID]
		connection.RespMutex.RUnlock()
		if !ok {
			done <- fmt.Errorf("missing response channel for request %d", request.ID)
			return
		}
		responseChannel <- &sliverpb.Envelope{ID: request.ID, Data: responseData}
		done <- nil
	}()
	return done
}

func capabilityMap(capabilities map[string]uint64) capabilityLookup {
	return func(id string) (uint64, bool) {
		value, ok := capabilities[id]
		return value, ok
	}
}
