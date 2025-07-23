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
	toImplantCacheSocks = socksDataCache{mutex: &sync.RWMutex{}, cache: map[uint64]map[uint64]*sliverpb.SocksData{}, lastActivity: map[uint64]time.Time{}}

	// SessionID->Tunnels[TunnelID]->Tunnel->Cache
	fromImplantCacheSocks = socksDataCache{mutex: &sync.RWMutex{}, cache: map[uint64]map[uint64]*sliverpb.SocksData{}, lastActivity: map[uint64]time.Time{}}
)

type socksDataCache struct {
	mutex        *sync.RWMutex
	cache        map[uint64]map[uint64]*sliverpb.SocksData
	lastActivity map[uint64]time.Time
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
	delete(c.lastActivity, tunnelID)
}

func (c *socksDataCache) DeleteSeq(tunnelID uint64, sequence uint64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, ok := c.cache[tunnelID]; !ok {
		return
	}

	delete(c.cache[tunnelID], sequence)
}

func (c *socksDataCache) recordActivity(tunnelID uint64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.lastActivity[tunnelID] = time.Now()
}

// Socks - Open an in-band port forward

const (
	writeTimeout            = 5 * time.Second
	batchSize               = 100 // Maximum number of sequences to batch
	inactivityCheckInterval = 4 * time.Second
	inactivityTimeout       = 15 * time.Second
	ToImplantTickerInterval = 10 * time.Millisecond // data going towards implant is usually smaller request data
	ToClientTickerInterval  = 5 * time.Millisecond  // data coming back from implant is usually larger response data
)

func (s *Server) SocksProxy(stream rpcpb.SliverRPC_SocksProxyServer) error {
	errChan := make(chan error, 2)
	defer close(errChan)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connDone := make(chan struct{})
	defer close(connDone)

	// Track all goroutines spawned for this session
	var wg sync.WaitGroup
	defer wg.Wait()

	// Track all tunnels created for this session
	activeTunnels := make(map[uint64]bool)
	var tunnelMutex sync.Mutex

	// Cleanup all tunnels on SocksProxy closure
	defer func() {
		tunnelMutex.Lock()
		for tunnelID := range activeTunnels {
			if tunnel := core.SocksTunnels.Get(tunnelID); tunnel != nil {
				rpcLog.Infof("Cleaning up tunnel %d on proxy closure", tunnelID)
				close(tunnel.FromImplant)
				tunnel.Client = nil
				s.CloseSocks(context.Background(), &sliverpb.Socks{TunnelID: tunnelID})
			}
		}
		tunnelMutex.Unlock()
	}()

	for {
		select {
		case err := <-errChan:
			rpcLog.Errorf("SocksProxy error: %v", err)
			return err
		default:
		}

		fromClient, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			rpcLog.Warnf("Error on stream recv %s", err)
			return err
		}

		tunnelMutex.Lock()
		activeTunnels[fromClient.TunnelID] = true // Mark this as an active tunnel
		tunnelMutex.Unlock()

		tunnel := core.SocksTunnels.Get(fromClient.TunnelID)
		if tunnel == nil {
			continue
		}

		if tunnel.Client == nil {
			tunnel.Client = stream
			tunnel.FromImplant = make(chan *sliverpb.SocksData, 100) // Buffered channel for 100 messages

			// Monitor tunnel goroutines for inactivity and cleanup
			wg.Add(1)
			go func(tunnelID uint64) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						rpcLog.Errorf("Recovered from panic in monitor: %v", r)
						errChan <- fmt.Errorf("monitor goroutine panic: %v", r)
						cancel() // Cancel context in case of a panic
					}
				}()

				ticker := time.NewTicker(inactivityCheckInterval)
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						return
					case <-connDone:
						return
					case <-ticker.C:
						tunnel := core.SocksTunnels.Get(tunnelID)
						if tunnel == nil || tunnel.Client == nil {
							return
						}
						session := core.Sessions.Get(tunnel.SessionID)

						// Check both caches for activity
						toImplantCacheSocks.mutex.RLock()
						fromImplantCacheSocks.mutex.RLock()
						toLastActivity := toImplantCacheSocks.lastActivity[tunnelID]
						fromLastActivity := fromImplantCacheSocks.lastActivity[tunnelID]
						toImplantCacheSocks.mutex.RUnlock()
						fromImplantCacheSocks.mutex.RUnlock()

						// Clean up goroutine if both directions have hit the idle threshold
						if time.Since(toLastActivity) > inactivityTimeout && time.Since(fromLastActivity) > inactivityTimeout {
							s.CloseSocks(context.Background(), &sliverpb.Socks{TunnelID: tunnelID})
							return
						}

						// Clean up goroutine if the client has disconnected early
						if tunnel.Client == nil || session == nil {
							s.CloseSocks(context.Background(), &sliverpb.Socks{TunnelID: tunnelID})
							return
						}

					}
				}
			}(fromClient.TunnelID)

			// Send Client
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						rpcLog.Errorf("Recovered from panic in client sender: %v", r)
						errChan <- fmt.Errorf("client sender panic: %v", r)
						cancel() // Cancel context in case of a panic
					}
				}()

				pendingData := make(map[uint64]*sliverpb.SocksData)
				ticker := time.NewTicker(ToClientTickerInterval)
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						return
					case <-connDone:
						return
					case tunnelData, ok := <-tunnel.FromImplant:
						if !ok {
							return
						}

						// Check if implant is requesting to close the tunnel
						if tunnelData.CloseConn {
							// Clean up the tunnel
							s.CloseSocks(context.Background(), &sliverpb.Socks{TunnelID: fromClient.TunnelID})
							return
						}

						sequence := tunnelData.Sequence
						fromImplantCacheSocks.Add(fromClient.TunnelID, sequence, tunnelData)
						pendingData[sequence] = tunnelData
						fromImplantCacheSocks.recordActivity(fromClient.TunnelID)

					case <-ticker.C:
						if tunnel.Client == nil {
							return
						}
						if len(pendingData) == 0 {
							continue
						}

						expectedSequence := atomic.LoadUint64(&tunnel.FromImplantSequence)
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
										errChan <- fmt.Errorf("Client sender panic: %v", r)
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
								atomic.AddUint64(&tunnel.FromImplantSequence, 1)
								expectedSequence++
								processed++
							}()
						}

					}
				}
			}()

			// Send Agent
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						rpcLog.Errorf("Recovered from panic in agent sender: %v", r)
						errChan <- fmt.Errorf("agent sender panic: %v", r)
						cancel() // Cancel context in case of a panic
					}
				}()

				pendingData := make(map[uint64]*sliverpb.SocksData)
				ticker := time.NewTicker(ToImplantTickerInterval)
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						return
					case <-connDone:
						return
					case <-ticker.C:
						if tunnel.Client == nil {
							return
						}
						sequence := atomic.LoadUint64(&tunnel.ToImplantSequence)

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
									atomic.AddUint64(&tunnel.ToImplantSequence, 1)
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
	defer func() {
		if r := recover(); r != nil {
			rpcLog.Errorf("Recovered from panic in CloseSocks for tunnel %d: %v", req.TunnelID, r)
		}
	}()

	tunnel := core.SocksTunnels.Get(req.TunnelID)
	if tunnel != nil {
		// Signal close to implant first
		if session := core.Sessions.Get(tunnel.SessionID); session != nil {
			data, _ := proto.Marshal(&sliverpb.SocksData{
				TunnelID:  req.TunnelID,
				CloseConn: true,
			})
			session.Connection.Send <- &sliverpb.Envelope{
				Type: sliverpb.MsgSocksData,
				Data: data,
			}
		}
		time.Sleep(100 * time.Millisecond) // Delay to allow close message to be sent
		tunnel.Client = nil                // Cleanup the tunnel
		if tunnel.FromImplant != nil {
			select {
			case _, ok := <-tunnel.FromImplant:
				if ok {
					close(tunnel.FromImplant)
				}
			default:
				close(tunnel.FromImplant)
			}
			tunnel.FromImplant = nil
		}
	}

	// Clean up caches
	toImplantCacheSocks.DeleteTun(req.TunnelID)
	fromImplantCacheSocks.DeleteTun(req.TunnelID)

	// Remove from core tunnels last
	if err := core.SocksTunnels.Close(req.TunnelID); err != nil {
		rpcLog.Errorf("Error closing tunnel %d: %v", req.TunnelID, err)
	}

	return &commonpb.Empty{}, nil
}
