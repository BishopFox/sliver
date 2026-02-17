//go:build server && go_sqlite && sliver_e2e

package c2_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	implantHandlers "github.com/bishopfox/sliver/implant/sliver/handlers"
	implantMTLS "github.com/bishopfox/sliver/implant/sliver/transports/mtls"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/c2"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/transport"
	"github.com/google/uuid"
	"github.com/hashicorp/yamux"
	"google.golang.org/protobuf/proto"
)

func TestMTLSYamux_Beacon_EndToEndAsyncPingRPC(t *testing.T) {
	// NOTE: If you run this test in a restricted environment where writes to
	// `~/.sliver` are blocked, set `SLIVER_ROOT_DIR` to a writable temp dir.

	grpcServer, grpcListener, err := transport.LocalListener()
	if err != nil {
		t.Fatalf("start local grpc listener: %v", err)
	}
	t.Cleanup(func() {
		grpcServer.Stop()
		_ = grpcListener.Close()
	})

	serverConn, implantConn := net.Pipe()
	t.Cleanup(func() {
		_ = serverConn.Close()
		_ = implantConn.Close()
	})
	go c2.HandleSliverConnectionForTest(serverConn)

	beaconID := uuid.NewString()
	beacon := startTestBeacon(t, implantConn, beaconID)
	t.Cleanup(beacon.Stop)

	rpcConn, err := dialBufConn(context.Background(), grpcListener)
	if err != nil {
		t.Fatalf("dial grpc/bufconn: %v", err)
	}
	t.Cleanup(func() { _ = rpcConn.Close() })
	rpcClient := rpcpb.NewSliverRPCClient(rpcConn)

	waitForBeaconRegistration(t, rpcClient, beaconID, 10*time.Second)

	const tasksCount = 8
	type taskInfo struct {
		id    string
		nonce int32
	}
	taskInfos := make([]taskInfo, 0, tasksCount)

	for i := 0; i < tasksCount; i++ {
		callCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		nonce := int32(1000 + i)
		resp, err := rpcClient.Ping(callCtx, &sliverpb.Ping{
			Nonce: nonce,
			Request: &commonpb.Request{
				Async:    true,
				BeaconID: beaconID,
				Timeout:  int64(5 * time.Second),
			},
		})
		cancel()
		if err != nil {
			t.Fatalf("rpc ping (async): %v", err)
		}
		if resp.GetResponse() == nil || resp.GetResponse().TaskID == "" {
			t.Fatalf("expected ping task id in async response, got=%v", resp.GetResponse())
		}
		taskInfos = append(taskInfos, taskInfo{id: resp.GetResponse().TaskID, nonce: nonce})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pendingTasks, err := beacon.FetchPendingTasks(ctx)
	if err != nil {
		t.Fatalf("fetch pending beacon tasks: %v", err)
	}
	if len(pendingTasks) < tasksCount {
		t.Fatalf("unexpected pending tasks count: got=%d want>=%d", len(pendingTasks), tasksCount)
	}

	results := executeBeaconTasks(t, pendingTasks)
	if err := beacon.SendResults(results); err != nil {
		t.Fatalf("send beacon results: %v", err)
	}

	for _, info := range taskInfos {
		waitForBeaconTaskCompleted(t, rpcClient, info.id, info.nonce, 10*time.Second)
	}
}

func TestMTLSYamux_Beacon_EndToEndAsyncMixedRPCs(t *testing.T) {
	// NOTE: If you run this test in a restricted environment where writes to
	// `~/.sliver` are blocked, set `SLIVER_ROOT_DIR` to a writable temp dir.

	grpcServer, grpcListener, err := transport.LocalListener()
	if err != nil {
		t.Fatalf("start local grpc listener: %v", err)
	}
	t.Cleanup(func() {
		grpcServer.Stop()
		_ = grpcListener.Close()
	})

	serverConn, implantConn := net.Pipe()
	t.Cleanup(func() {
		_ = serverConn.Close()
		_ = implantConn.Close()
	})
	go c2.HandleSliverConnectionForTest(serverConn)

	beaconID := uuid.NewString()
	beacon := startTestBeacon(t, implantConn, beaconID)
	t.Cleanup(beacon.Stop)

	rpcConn, err := dialBufConn(context.Background(), grpcListener)
	if err != nil {
		t.Fatalf("dial grpc/bufconn: %v", err)
	}
	t.Cleanup(func() { _ = rpcConn.Close() })
	rpcClient := rpcpb.NewSliverRPCClient(rpcConn)

	waitForBeaconRegistration(t, rpcClient, beaconID, 10*time.Second)

	testDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(testDir, "alpha.txt"), []byte("alpha"), 0600); err != nil {
		t.Fatalf("write alpha.txt: %v", err)
	}
	if err := os.Mkdir(filepath.Join(testDir, "subdir"), 0700); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}

	reqTimeout := int64(5 * time.Second)

	type taskCheck struct {
		id    string
		check func(*testing.T, []byte)
	}
	checks := make([]taskCheck, 0, 8)

	addTask := func(callCtx context.Context, taskID string, check func(*testing.T, []byte)) {
		if taskID == "" {
			t.Fatalf("expected task id in async response")
		}
		checks = append(checks, taskCheck{id: taskID, check: check})
	}

	for i := 0; i < 4; i++ {
		callCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		nonce := int32(2000 + i)
		resp, err := rpcClient.Ping(callCtx, &sliverpb.Ping{
			Nonce: nonce,
			Request: &commonpb.Request{
				Async:    true,
				BeaconID: beaconID,
				Timeout:  reqTimeout,
			},
		})
		cancel()
		if err != nil {
			t.Fatalf("rpc ping (async): %v", err)
		}

		taskID := resp.GetResponse().GetTaskID()
		wantNonce := nonce
		addTask(callCtx, taskID, func(t *testing.T, data []byte) {
			t.Helper()
			ping := &sliverpb.Ping{}
			if err := proto.Unmarshal(data, ping); err != nil {
				t.Fatalf("unmarshal ping response: %v", err)
			}
			if ping.Nonce != wantNonce {
				t.Fatalf("unexpected ping nonce: got=%d want=%d", ping.Nonce, wantNonce)
			}
		})
	}

	{
		callCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		resp, err := rpcClient.Pwd(callCtx, &sliverpb.PwdReq{
			Request: &commonpb.Request{
				Async:    true,
				BeaconID: beaconID,
				Timeout:  reqTimeout,
			},
		})
		cancel()
		if err != nil {
			t.Fatalf("rpc pwd (async): %v", err)
		}
		taskID := resp.GetResponse().GetTaskID()
		addTask(callCtx, taskID, func(t *testing.T, data []byte) {
			t.Helper()
			pwd := &sliverpb.Pwd{}
			if err := proto.Unmarshal(data, pwd); err != nil {
				t.Fatalf("unmarshal pwd response: %v", err)
			}
			if pwd.Path == "" {
				t.Fatalf("unexpected empty pwd path")
			}
		})
	}

	{
		callCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		resp, err := rpcClient.GetEnv(callCtx, &sliverpb.EnvReq{
			Name: "PATH",
			Request: &commonpb.Request{
				Async:    true,
				BeaconID: beaconID,
				Timeout:  reqTimeout,
			},
		})
		cancel()
		if err != nil {
			t.Fatalf("rpc getenv (async): %v", err)
		}
		taskID := resp.GetResponse().GetTaskID()
		addTask(callCtx, taskID, func(t *testing.T, data []byte) {
			t.Helper()
			env := &sliverpb.EnvInfo{}
			if err := proto.Unmarshal(data, env); err != nil {
				t.Fatalf("unmarshal env response: %v", err)
			}
			if len(env.Variables) != 1 || env.Variables[0].Key != "PATH" {
				t.Fatalf("unexpected env response: %+v", env.Variables)
			}
		})
	}

	{
		callCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		resp, err := rpcClient.Ls(callCtx, &sliverpb.LsReq{
			Path: testDir,
			Request: &commonpb.Request{
				Async:    true,
				BeaconID: beaconID,
				Timeout:  reqTimeout,
			},
		})
		cancel()
		if err != nil {
			t.Fatalf("rpc ls (async): %v", err)
		}
		taskID := resp.GetResponse().GetTaskID()
		addTask(callCtx, taskID, func(t *testing.T, data []byte) {
			t.Helper()
			ls := &sliverpb.Ls{}
			if err := proto.Unmarshal(data, ls); err != nil {
				t.Fatalf("unmarshal ls response: %v", err)
			}
			if !ls.Exists {
				t.Fatalf("unexpected ls response (Exists=false): %v", ls.GetResponse())
			}
			found := false
			for _, file := range ls.Files {
				if file.Name == "alpha.txt" {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("ls missing expected alpha.txt entry")
			}
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pendingTasks, err := beacon.FetchPendingTasks(ctx)
	if err != nil {
		t.Fatalf("fetch pending beacon tasks: %v", err)
	}
	if len(pendingTasks) < len(checks) {
		t.Fatalf("unexpected pending tasks count: got=%d want>=%d", len(pendingTasks), len(checks))
	}

	results := executeBeaconTasks(t, pendingTasks)
	if err := beacon.SendResults(results); err != nil {
		t.Fatalf("send beacon results: %v", err)
	}

	for _, tc := range checks {
		data := waitForBeaconTaskResponseBytes(t, rpcClient, tc.id, 10*time.Second)
		tc.check(t, data)
	}
}

type testBeacon struct {
	id       string
	conn     net.Conn
	session  *yamux.Session
	incoming chan *sliverpb.Envelope
	done     chan struct{}
}

func startTestBeacon(t *testing.T, conn net.Conn, beaconID string) *testBeacon {
	t.Helper()

	if _, err := conn.Write([]byte(implantMTLS.YamuxPreface)); err != nil {
		t.Fatalf("write yamux preface: %v", err)
	}

	cfg := yamux.DefaultConfig()
	cfg.LogOutput = io.Discard
	muxSession, err = yamux.Client(conn, cfg)
	if err != nil {
		t.Fatalf("start yamux client session: %v", err)
	}

	beacon := &testBeacon{
		id:       beaconID,
		conn:     conn,
		session:  muxSession,
		incoming: make(chan *sliverpb.Envelope, 64),
		done:     make(chan struct{}),
	}
	go beacon.recvLoop()

	register := &sliverpb.Register{
		Name:              "e2e-beacon",
		Hostname:          "localhost",
		Uuid:              uuid.NewString(),
		Username:          "unit-test",
		Os:                runtime.GOOS,
		Arch:              runtime.GOARCH,
		Pid:               int32(os.Getpid()),
		Filename:          "sliver-e2e",
		ActiveC2:          "mtls://e2e",
		Version:           "e2e",
		ReconnectInterval: 0,
		ProxyURL:          "",
		Locale:            "en_US",
	}
	regData, err := proto.Marshal(&sliverpb.BeaconRegister{
		ID:          beaconID,
		Interval:    int64(1),
		Jitter:      int64(0),
		Register:    register,
		NextCheckin: int64(1),
	})
	if err != nil {
		t.Fatalf("marshal beacon register: %v", err)
	}
	if err := sendYamuxEnvelope(muxSession, &sliverpb.Envelope{Type: sliverpb.MsgBeaconRegister, Data: regData}); err != nil {
		t.Fatalf("send beacon register: %v", err)
	}

	return beacon
}

func (b *testBeacon) Stop() {
	select {
	case <-b.done:
	default:
		close(b.done)
	}
	_ = b.session.Close()
	_ = b.conn.Close()
}

func (b *testBeacon) recvLoop() {
	defer close(b.incoming)
	for {
		stream, err := b.session.Accept()
		if err != nil {
			return
		}

		go func(stream net.Conn) {
			defer stream.Close()
			envelope, err := implantMTLS.ReadEnvelope(stream)
			if err != nil || envelope == nil {
				return
			}
			select {
			case b.incoming <- envelope:
			case <-b.done:
			}
		}(stream)
	}
}

func (b *testBeacon) FetchPendingTasks(ctx context.Context) ([]*sliverpb.Envelope, error) {
	reqData, err := proto.Marshal(&sliverpb.BeaconTasks{
		ID:          b.id,
		NextCheckin: int64(1),
	})
	if err != nil {
		return nil, err
	}
	if err := sendYamuxEnvelope(b.session, &sliverpb.Envelope{Type: sliverpb.MsgBeaconTasks, Data: reqData}); err != nil {
		return nil, err
	}

	for {
		select {
		case envelope, ok := <-b.incoming:
			if !ok {
				return nil, fmt.Errorf("beacon connection closed")
			}
			if envelope.Type != sliverpb.MsgBeaconTasks {
				continue
			}
			resp := &sliverpb.BeaconTasks{}
			if err := proto.Unmarshal(envelope.Data, resp); err != nil {
				return nil, err
			}
			return resp.Tasks, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (b *testBeacon) SendResults(results []*sliverpb.Envelope) error {
	reqData, err := proto.Marshal(&sliverpb.BeaconTasks{
		ID:          b.id,
		NextCheckin: int64(1),
		Tasks:       results,
	})
	if err != nil {
		return err
	}
	return sendYamuxEnvelope(b.session, &sliverpb.Envelope{Type: sliverpb.MsgBeaconTasks, Data: reqData})
}

func executeBeaconTasks(t *testing.T, tasks []*sliverpb.Envelope) []*sliverpb.Envelope {
	t.Helper()

	sysHandlers := implantHandlers.GetSystemHandlers()
	results := make([]*sliverpb.Envelope, 0, len(tasks))

	for _, task := range tasks {
		if task == nil || task.ID == 0 {
			continue
		}

		handler, ok := sysHandlers[task.Type]
		if !ok {
			results = append(results, &sliverpb.Envelope{ID: task.ID, UnknownMessageType: true})
			continue
		}

		done := make(chan struct{})
		var (
			respData []byte
			respErr  error
		)
		var doneOnce sync.Once
		handler(task.Data, func(data []byte, err error) {
			respData = data
			respErr = err
			doneOnce.Do(func() {
				close(done)
			})
		})

		select {
		case <-done:
			_ = respErr
			results = append(results, &sliverpb.Envelope{ID: task.ID, Data: respData})
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for handler response (task=%d type=%d)", task.ID, task.Type)
		}
	}
	return results
}

func waitForBeaconRegistration(t *testing.T, rpcClient rpcpb.SliverRPCClient, beaconID string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		_, err := rpcClient.GetBeacon(ctx, &clientpb.Beacon{ID: beaconID})
		cancel()
		if err == nil {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for beacon registration")
}

func waitForBeaconTaskResponseBytes(t *testing.T, rpcClient rpcpb.SliverRPCClient, taskID string, timeout time.Duration) []byte {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		task, err := rpcClient.GetBeaconTaskContent(ctx, &clientpb.BeaconTask{ID: taskID})
		cancel()
		if err != nil {
			time.Sleep(25 * time.Millisecond)
			continue
		}
		if task.State != models.COMPLETED || len(task.Response) == 0 {
			time.Sleep(25 * time.Millisecond)
			continue
		}
		return task.Response
	}
	t.Fatalf("timed out waiting for beacon task completion (task=%s)", taskID)
	return nil
}

func waitForBeaconTaskCompleted(t *testing.T, rpcClient rpcpb.SliverRPCClient, taskID string, wantNonce int32, timeout time.Duration) {
	t.Helper()

	data := waitForBeaconTaskResponseBytes(t, rpcClient, taskID, timeout)
	ping := &sliverpb.Ping{}
	if err := proto.Unmarshal(data, ping); err != nil {
		t.Fatalf("unmarshal ping response: %v", err)
	}
	if ping.Nonce != wantNonce {
		t.Fatalf("unexpected ping nonce: got=%d want=%d", ping.Nonce, wantNonce)
	}
}
