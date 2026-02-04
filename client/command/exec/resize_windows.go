//go:build windows

package exec

import (
	"context"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"os"
	"sync"
	"time"
)

func startPtyResizeWatcher(con *console.SliverClient, cmd *cobra.Command, tunnelID uint64) func() {
	if con == nil || con.Rpc == nil {
		return func() {}
	}

	stdoutFd := int(os.Stdout.Fd())
	stdinFd := int(os.Stdin.Fd())
	if !term.IsTerminal(stdoutFd) && !term.IsTerminal(stdinFd) {
		return func() {}
	}

	stopCh := make(chan struct{})
	doneCh := make(chan struct{})

	go func() {
		defer close(doneCh)

		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()

		lastCols, lastRows := -1, -1
		send := func() bool {
			cols, rows, err := term.GetSize(stdoutFd)
			if err != nil || cols <= 0 || rows <= 0 {
				cols, rows, err = term.GetSize(stdinFd)
			}
			if err != nil || cols <= 0 || rows <= 0 {
				return true
			}
			if cols == lastCols && rows == lastRows {
				return true
			}
			lastCols, lastRows = cols, rows

			req := con.ActiveTarget.Request(cmd)
			if req == nil {
				return false
			}
			req.Timeout = int64(2 * time.Second)

			_, err = con.Rpc.ShellResize(context.Background(), &sliverpb.ShellResizeReq{
				Request:  req,
				TunnelID: tunnelID,
				Rows:     uint32(rows),
				Cols:     uint32(cols),
			})
			if status.Code(err) == codes.Unimplemented {
				return false
			}
			return true
		}

		if !send() {
			return
		}

		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				if !send() {
					return
				}
			}
		}
	}()

	var once sync.Once
	return func() {
		once.Do(func() {
			close(stopCh)
			<-doneCh
		})
	}
}
