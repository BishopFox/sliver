package console

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// ConnectionDetails provides (optional) metadata about the active connection.
// ConnectionDetails 提供有关活动 connection. 的（可选）元数据
// In the sliver-client binary this is typically sourced from ~/.sliver-client/configs/*.
// In sliver__PH0__ 二进制文件通常源自 ~/.sliver__PH1__/configs/*。
type ConnectionDetails struct {
	ConfigKey string
	Config    *assets.ClientConfig
}

// CurrentConnection returns the current connection metadata, if any.
// CurrentConnection 返回当前连接元数据，如果 any.
func (con *SliverClient) CurrentConnection() (*ConnectionDetails, connectivity.State, bool) {
	con.connMu.Lock()
	defer con.connMu.Unlock()

	if con.grpcConn == nil {
		return con.connDetails, connectivity.Idle, false
	}
	return con.connDetails, con.grpcConn.GetState(), true
}

// SetConnection swaps the active RPC connection, restarting streams (events/tunnels/logs).
// SetConnection 交换活动的 RPC 连接，重新启动流 (events/tunnels/logs)。
// It is safe to call multiple times (e.g., on server switch).
// It 可以安全地多次调用（e.g.，在服务器交换机上）。
func (con *SliverClient) SetConnection(rpc rpcpb.SliverRPCClient, grpcConn *grpc.ClientConn, details *ConnectionDetails) error {
	con.connMu.Lock()
	defer con.connMu.Unlock()

	con.detachConnectionLocked()

	con.Rpc = rpc
	con.grpcConn = grpcConn
	con.connDetails = details

	if con.Rpc == nil || con.grpcConn == nil {
		return nil
	}

	// Clear per-server state that should not be carried across connections.
	// Clear per__PH0__ 声明不应跨过 connections.
	con.EventListeners = &sync.Map{}
	con.BeaconTaskCallbacksMutex.Lock()
	con.BeaconTaskCallbacks = map[string]BeaconTaskCallback{}
	con.BeaconTaskCallbacksMutex.Unlock()
	con.ActiveTarget.Set(nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	con.connCancel = cancel

	wg := &sync.WaitGroup{}
	con.connWg = wg

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		con.startEventLoop(ctx)
	}(wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		if err := core.TunnelLoop(ctx, con.Rpc); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("TunnelLoop error: %v", err)
		}
	}(wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup, conn *grpc.ClientConn) {
		defer wg.Done()
		con.monitorConnectionLost(ctx, conn)
	}(wg, con.grpcConn)

	con.refreshRemoteLogStreamsLocked()

	return nil
}

// CloseConnection stops background loops and closes the current gRPC connection.
// CloseConnection 停止后台循环并关闭当前的 gRPC connection.
func (con *SliverClient) CloseConnection() error {
	con.connMu.Lock()
	defer con.connMu.Unlock()

	con.detachConnectionLocked()
	return nil
}

func (con *SliverClient) detachConnectionLocked() {
	// Stop sending remote logs first so background io.Copy loops can't break.
	// Stop 首先发送远程日志，因此后台 io.Copy 循环不能 break.
	con.setRemoteLogStreamsLocked(nil, nil)

	cancel := con.connCancel
	con.connCancel = nil
	wg := con.connWg
	con.connWg = &sync.WaitGroup{}

	if cancel != nil {
		cancel()
	}
	if wg != nil {
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			// Best-effort; we don't want to deadlock a switch/exit.
			// Best__PH0__；我们不想让 switch/exit. 陷入僵局
		}
	}

	if con.grpcConn != nil {
		_ = con.grpcConn.Close()
		con.grpcConn = nil
	}
	con.Rpc = nil
	con.connDetails = nil

	// Tear down any singleton network tooling that was tied to the previous server.
	// Tear 关闭与之前的 server. 相关的任何单例网络工具
	core.ResetClientState()
}

func (con *SliverClient) monitorConnectionLost(ctx context.Context, conn *grpc.ClientConn) {
	if conn == nil {
		return
	}

	for {
		state := conn.GetState()
		if state == connectivity.Idle {
			// Mirror the previous behavior from cli/console.go, but only when not canceled.
			// Mirror 之前来自 cli/console.go 的行为，但仅当不是 canceled. 时
			select {
			case <-ctx.Done():
				return
			default:
			}
			fmt.Println("\nLost connection to server. Exiting now.")
			con.FlushOutput()
			//nolint:forbidigo // Explicitly exits to match legacy behavior.
			os.Exit(1)
		}

		if !conn.WaitForStateChange(ctx, state) {
			return
		}
	}
}
