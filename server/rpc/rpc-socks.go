package rpc

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"sync"
	"sync/atomic"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"google.golang.org/protobuf/proto"
)

var (
	// SessionID->Tunnels[TunnelID]->Tunnel->Cache map[uint64]*sliverpb.SocksData{}
	toImplantCacheSocks = socksDataCache{mutex: &sync.RWMutex{}, cache: map[uint64]map[uint64]*sliverpb.SocksData{}}

	// SessionID->Tunnels[TunnelID]->Tunnel->Cache
	fromImplantCacheSocks = socksDataCache{mutex: &sync.RWMutex{}, cache: map[uint64]map[uint64]*sliverpb.SocksData{}}
)

type socksDataCache struct {
	mutex *sync.RWMutex
	cache map[uint64]map[uint64]*sliverpb.SocksData
}

func (c *socksDataCache) Add(tunnelID uint64, sequence uint64, tunnelData *sliverpb.SocksData) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, ok := c.cache[tunnelID]; !ok {
		c.cache[tunnelID] = map[uint64]*sliverpb.SocksData{}
	}

	c.cache[tunnelID][sequence] = tunnelData
}

func (c *socksDataCache) Get(tunnelID uint64, sequence uint64) (*sliverpb.SocksData, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if _, ok := c.cache[tunnelID]; !ok {
		return nil, false
	}

	val, ok := c.cache[tunnelID][sequence]

	return val, ok
}

func (c *socksDataCache) DeleteTun(tunnelID uint64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.cache, tunnelID)
}

func (c *socksDataCache) DeleteSeq(tunnelID uint64, sequence uint64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, ok := c.cache[tunnelID]; !ok {
		return
	}

	delete(c.cache[tunnelID], sequence)
}

// Socks - Open an in-band port forward

const (
	writeTimeout = 5 * time.Second
	batchSize    = 100 // Maximum number of sequences to batch
)

func (s *Server) SocksProxy(stream rpcpb.SliverRPC_SocksProxyServer) error {
	errChan := make(chan error, 2)
	defer close(errChan)
	defer func() {
		if r := recover(); r != nil {
			rpcLog.Errorf("Recovered from panic in SocksProxy: %v", r)
		}
	}()

	for {
		fromClient, err := stream.Recv()
		if err == io.EOF {
			break
		}
		//fmt.Println("Send Agent 1 ",fromClient.TunnelID,len(fromClient.Data))
		if err != nil {
			rpcLog.Warnf("Error on stream recv %s", err)
			return err
		}
		tunnelLog.Debugf("Tunnel %d: From client %d byte(s)",
			fromClient.TunnelID, len(fromClient.Data))
		socks := core.SocksTunnels.Get(fromClient.TunnelID)
		if socks == nil {
			return nil
		}
		if socks.Client == nil {
			socks.Client = stream // Bind client to tunnel
			// Send Client
			go func() {
				defer func() {
					if r := recover(); r != nil {
						errChan <- fmt.Errorf("client sender panic: %v", r)
						rpcLog.Errorf("Recovered from panic in client sender: %v", r)
					}
				}()

				pendingData := make(map[uint64]*sliverpb.SocksData)
				ticker := time.NewTicker(1 * time.Millisecond) // 1ms ticker - data coming back from implant is usually larger response data
				defer ticker.Stop()

				for {
					select {
					case tunnelData, ok := <-socks.FromImplant:
						if !ok {
							rpcLog.Debug("FromImplant channel closed")
							return
						}
						sequence := tunnelData.Sequence
						fromImplantCacheSocks.Add(fromClient.TunnelID, sequence, tunnelData)
						pendingData[sequence] = tunnelData

					case <-ticker.C:
						if len(pendingData) == 0 {
							continue
						}

						expectedSequence := atomic.LoadUint64(&socks.FromImplantSequence)
						processed := 0

						// Perform Batching
						for i := 0; i < batchSize && processed < len(pendingData); i++ {
							data, exists := pendingData[expectedSequence]
							if !exists {
								break // Stop batching if we don't have the next expected sequence
							}

							func() {
								defer func() {
									if r := recover(); r != nil {
										errChan <- fmt.Errorf("client sender panic: %v", r)
										rpcLog.Errorf("Recovered from panic in client sender: %v", r)
									}
								}()

								err := stream.Send(&sliverpb.SocksData{
									CloseConn: data.CloseConn,
									TunnelID:  data.TunnelID,
									Data:      data.Data,
								})

								if err != nil {
									rpcLog.Errorf("Send error: %v", err)
									return
								}

								delete(pendingData, expectedSequence)
								fromImplantCacheSocks.DeleteSeq(fromClient.TunnelID, expectedSequence)
								atomic.AddUint64(&socks.FromImplantSequence, 1)
								expectedSequence++
								processed++
							}()
						}
					}
				}
			}()

			// Send Agent
			go func() {
				defer func() {
					if r := recover(); r != nil {
						errChan <- fmt.Errorf("agent sender panic: %v", r)
						rpcLog.Errorf("Recovered from panic in agent sender: %v", r)
					}
				}()

				pendingData := make(map[uint64]*sliverpb.SocksData)
				ticker := time.NewTicker(10 * time.Millisecond) // 10ms ticker - data going towards implact is usually smaller request data
				defer ticker.Stop()

				for {
					select {
					case <-ticker.C:
						sequence := atomic.LoadUint64(&socks.ToImplantSequence)

						func() {
							defer func() {
								if r := recover(); r != nil {
									rpcLog.Errorf("Recovered from processing panic: %v", r)
								}
							}()

							for {
								recv, ok := toImplantCacheSocks.Get(fromClient.TunnelID, sequence)
								if !ok {
									break
								}

								session := core.Sessions.Get(fromClient.Request.SessionID)
								if session == nil {
									rpcLog.Error("Session not found")
									break
								}

								data, err := proto.Marshal(recv)
								if err != nil {
									rpcLog.Errorf("Failed to marshal data: %v", err)
									continue
								}

								select {
								case session.Connection.Send <- &sliverpb.Envelope{
									Type: sliverpb.MsgSocksData,
									Data: data,
								}:
									toImplantCacheSocks.DeleteSeq(fromClient.TunnelID, sequence)
									atomic.AddUint64(&socks.ToImplantSequence, 1)
									sequence++
								case <-time.After(writeTimeout):
									rpcLog.Error("Write timeout to implant")
									pendingData[sequence] = recv
									break
								}
							}
						}()
					}
				}
			}()
		}

		toImplantCacheSocks.Add(fromClient.TunnelID, fromClient.Sequence, fromClient)
	}

	select {
	case err := <-errChan:
		rpcLog.Errorf("SocksProxy Goroutine error: %v", err)
	default:
	}

	return nil
}

// CreateSocks5 - Create requests we close a Socks
func (s *Server) CreateSocks(ctx context.Context, req *sliverpb.Socks) (*sliverpb.Socks, error) {
	session := core.Sessions.Get(req.SessionID)
	if session == nil {
		return nil, ErrInvalidSessionID
	}
	tunnel := core.SocksTunnels.Create(session.ID)
	if tunnel == nil {
		return nil, ErrTunnelInitFailure
	}

	return &sliverpb.Socks{
		SessionID: session.ID,
		TunnelID:  tunnel.ID,
	}, nil
}

// CloseSocks - Client requests we close a Socks
func (s *Server) CloseSocks(ctx context.Context, req *sliverpb.Socks) (*commonpb.Empty, error) {
	err := core.SocksTunnels.Close(req.TunnelID)
	toImplantCacheSocks.DeleteTun(req.TunnelID)
	fromImplantCacheSocks.DeleteTun(req.TunnelID)
	if err != nil {
		return nil, err
	}
	return &commonpb.Empty{}, nil
}
