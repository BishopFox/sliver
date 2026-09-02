package shell

import (
	"context"
	"testing"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/grpc"
)

type closeTunnelTestRPC struct {
	rpcpb.SliverRPCClient
	deadline time.Time
}

func (r *closeTunnelTestRPC) CloseTunnel(ctx context.Context, _ *sliverpb.Tunnel, _ ...grpc.CallOption) (*commonpb.Empty, error) {
	deadline, ok := ctx.Deadline()
	if ok {
		r.deadline = deadline
	}
	return &commonpb.Empty{}, nil
}

func TestCloseShellTunnelUsesBoundedContext(t *testing.T) {
	rpc := &closeTunnelTestRPC{}
	con := &console.SliverClient{Rpc: rpc}

	if err := closeShellTunnel(con, 99, "session"); err != nil {
		t.Fatalf("close shell tunnel: %v", err)
	}
	if rpc.deadline.IsZero() {
		t.Fatal("CloseTunnel RPC context has no deadline")
	}
	if remaining := time.Until(rpc.deadline); remaining <= 0 || remaining > shellCloseTimeout {
		t.Fatalf("CloseTunnel deadline is %v away, want within (0, %v]", remaining, shellCloseTimeout)
	}
}
