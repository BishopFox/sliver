//go:build !windows

package exec

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)

	stopCh := make(chan struct{})
	doneCh := make(chan struct{})

	go func() {
		defer close(doneCh)

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
			case <-sigCh:
				if !send() {
					return
				}
			}
		}
	}()

	var once sync.Once
	return func() {
		once.Do(func() {
			signal.Stop(sigCh)
			close(stopCh)
			<-doneCh
		})
	}
}
