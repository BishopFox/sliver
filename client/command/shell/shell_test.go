package shell

import (
	"context"
	"testing"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/grpc"
)

type closeTunnelTestRPC struct {
	rpcpb.SliverRPCClient
	deadline time.Time
	calls    int
}

type closeShellTestStream struct {
	grpc.BidiStreamingClient[sliverpb.TunnelData, sliverpb.TunnelData]
	sent chan *sliverpb.TunnelData
}

func (stream *closeShellTestStream) Send(frame *sliverpb.TunnelData) error {
	stream.sent <- frame
	return nil
}

func (r *closeTunnelTestRPC) CloseTunnel(ctx context.Context, _ *sliverpb.Tunnel, _ ...grpc.CallOption) (*commonpb.Empty, error) {
	r.calls++
	deadline, ok := ctx.Deadline()
	if ok {
		r.deadline = deadline
	}
	return &commonpb.Empty{}, nil
}

func TestCloseShellTunnelUsesBoundedContext(t *testing.T) {
	tunnels := core.GetTunnels()
	tunnels.Reset()
	t.Cleanup(tunnels.Reset)
	stream := &closeShellTestStream{sent: make(chan *sliverpb.TunnelData, 1)}
	tunnels.SetStream(stream)
	tunnel := tunnels.Start(99, "session")
	waitForShellBindFrame(t, stream.sent)

	rpc := &closeTunnelTestRPC{}
	con := &console.SliverClient{Rpc: rpc}

	if err := closeShellTunnel(con, tunnel); err != nil {
		t.Fatalf("close shell tunnel: %v", err)
	}
	if rpc.calls != 1 {
		t.Fatalf("CloseTunnel RPC calls = %d, want 1", rpc.calls)
	}
	if rpc.deadline.IsZero() {
		t.Fatal("CloseTunnel RPC context has no deadline")
	}
	if remaining := time.Until(rpc.deadline); remaining <= 0 || remaining > shellCloseTimeout {
		t.Fatalf("CloseTunnel deadline is %v away, want within (0, %v]", remaining, shellCloseTimeout)
	}
}

func TestCloseShellTunnelDoesNotCloseReplacementGeneration(t *testing.T) {
	tunnels := core.GetTunnels()
	tunnels.Reset()
	t.Cleanup(tunnels.Reset)
	stream := &closeShellTestStream{sent: make(chan *sliverpb.TunnelData, 2)}
	tunnels.SetStream(stream)

	old := tunnels.Start(100, "old-session")
	waitForShellBindFrame(t, stream.sent)
	replacement := tunnels.Start(100, "new-session")
	waitForShellBindFrame(t, stream.sent)
	rpc := &closeTunnelTestRPC{}
	con := &console.SliverClient{Rpc: rpc}

	if err := closeShellTunnel(con, old); err != nil {
		t.Fatalf("close stale shell tunnel: %v", err)
	}
	if rpc.calls != 0 {
		t.Fatalf("stale shell cleanup sent %d numeric CloseTunnel RPCs, want 0", rpc.calls)
	}
	if current := tunnels.Get(100); current != replacement {
		t.Fatal("stale shell cleanup removed the replacement tunnel")
	}
	select {
	case <-replacement.Done():
		t.Fatal("stale shell cleanup closed the replacement tunnel")
	default:
	}
}

func waitForShellBindFrame(t *testing.T, frames <-chan *sliverpb.TunnelData) {
	t.Helper()
	select {
	case <-frames:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for shell tunnel bind frame")
	}
}
