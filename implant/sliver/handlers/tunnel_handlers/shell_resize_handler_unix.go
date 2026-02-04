//go:build darwin || linux || freebsd || openbsd || dragonfly

package tunnel_handlers

import (
	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"os"

	"github.com/bishopfox/sliver/implant/sliver/shell/pty"
	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

type ptyResizer interface {
	Resize(rows, cols uint32) error
}

func ShellResizeReqHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	req := &sliverpb.ShellResizeReq{}
	err := proto.Unmarshal(envelope.Data, req)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[shell] Failed to unmarshal protobuf %s", err)
		// {{end}}
	} else if tun := connection.Tunnel(req.TunnelID); tun != nil {
		rows := req.GetRows()
		cols := req.GetCols()
		if rows != 0 && cols != 0 {
			if resizer, ok := tun.Writer.(ptyResizer); ok {
				err := resizer.Resize(rows, cols)
				if err != nil {
					// {{if .Config.Debug}}
					log.Printf("[shell] Failed to resize PTY: %s", err)
					// {{end}}
				}
			} else if f, ok := tun.Writer.(*os.File); ok {
				if rows > 0xffff {
					rows = 0xffff
				}
				if cols > 0xffff {
					cols = 0xffff
				}
				err := pty.Setsize(f, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
				if err != nil {
					// {{if .Config.Debug}}
					log.Printf("[shell] Failed to resize PTY: %s", err)
					// {{end}}
				}
			}
		}
	}

	resp, _ := proto.Marshal(&commonpb.Empty{})
	connection.Send <- &sliverpb.Envelope{
		ID:   envelope.ID,
		Data: resp,
	}
}
