package socks5

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	"github.com/things-go/go-socks5/statute"
)

// AddressRewriter is used to rewrite a destination transparently
type AddressRewriter interface {
	Rewrite(ctx context.Context, request *Request) (context.Context, *statute.AddrSpec)
}

// A Request represents request received by a server
type Request struct {
	statute.Request
	// AuthContext provided during negotiation
	AuthContext *AuthContext
	// LocalAddr of the the network server listen
	LocalAddr net.Addr
	// RemoteAddr of the the network that sent the request
	RemoteAddr net.Addr
	// DestAddr of the actual destination (might be affected by rewrite)
	DestAddr *statute.AddrSpec
	// Reader connect of request
	Reader io.Reader
	// RawDestAddr of the desired destination
	RawDestAddr *statute.AddrSpec
}

// ParseRequest creates a new Request from the tcp connection
func ParseRequest(bufConn io.Reader) (*Request, error) {
	hd, err := statute.ParseRequest(bufConn)
	if err != nil {
		return nil, err
	}
	return &Request{
		Request:     hd,
		RawDestAddr: &hd.DstAddr,
		Reader:      bufConn,
	}, nil
}

// handleRequest is used for request processing after authentication
func (sf *Server) handleRequest(write io.Writer, req *Request) error {
	var err error

	ctx := context.Background()
	// Resolve the address if we have a FQDN
	dest := req.RawDestAddr
	if dest.FQDN != "" {
		ctx, dest.IP, err = sf.resolver.Resolve(ctx, dest.FQDN)
		if err != nil {
			if err := SendReply(write, statute.RepHostUnreachable, nil); err != nil {
				return fmt.Errorf("failed to send reply, %v", err)
			}
			return fmt.Errorf("failed to resolve destination[%v], %v", dest.FQDN, err)
		}
	}

	// Apply any address rewrites
	req.DestAddr = req.RawDestAddr
	if sf.rewriter != nil {
		ctx, req.DestAddr = sf.rewriter.Rewrite(ctx, req)
	}

	// Check if this is allowed
	var ok bool
	ctx, ok = sf.rules.Allow(ctx, req)
	if !ok {
		if err := SendReply(write, statute.RepRuleFailure, nil); err != nil {
			return fmt.Errorf("failed to send reply, %v", err)
		}
		return fmt.Errorf("bind to %v blocked by rules", req.RawDestAddr)
	}

	// Switch on the command
	switch req.Command {
	case statute.CommandConnect:
		if sf.userConnectHandle != nil {
			return sf.userConnectHandle(ctx, write, req)
		}
		return sf.handleConnect(ctx, write, req)
	case statute.CommandBind:
		if sf.userBindHandle != nil {
			return sf.userBindHandle(ctx, write, req)
		}
		return sf.handleBind(ctx, write, req)
	case statute.CommandAssociate:
		if sf.userAssociateHandle != nil {
			return sf.userAssociateHandle(ctx, write, req)
		}
		return sf.handleAssociate(ctx, write, req)
	default:
		if err := SendReply(write, statute.RepCommandNotSupported, nil); err != nil {
			return fmt.Errorf("failed to send reply, %v", err)
		}
		return fmt.Errorf("unsupported command[%v]", req.Command)
	}
}

// handleConnect is used to handle a connect command
func (sf *Server) handleConnect(ctx context.Context, writer io.Writer, request *Request) error {
	// Attempt to connect
	dial := sf.dial
	if dial == nil {
		dial = func(ctx context.Context, net_, addr string) (net.Conn, error) {
			return net.Dial(net_, addr)
		}
	}
	target, err := dial(ctx, "tcp", request.DestAddr.String())
	if err != nil {
		msg := err.Error()
		resp := statute.RepHostUnreachable
		if strings.Contains(msg, "refused") {
			resp = statute.RepConnectionRefused
		} else if strings.Contains(msg, "network is unreachable") {
			resp = statute.RepNetworkUnreachable
		}
		if err := SendReply(writer, resp, nil); err != nil {
			return fmt.Errorf("failed to send reply, %v", err)
		}
		return fmt.Errorf("connect to %v failed, %v", request.RawDestAddr, err)
	}
	defer target.Close()

	// Send success
	if err := SendReply(writer, statute.RepSuccess, target.LocalAddr()); err != nil {
		return fmt.Errorf("failed to send reply, %v", err)
	}

	// Start proxying
	errCh := make(chan error, 2)
	sf.goFunc(func() { errCh <- sf.Proxy(target, request.Reader) })
	sf.goFunc(func() { errCh <- sf.Proxy(writer, target) })
	// Wait
	for i := 0; i < 2; i++ {
		e := <-errCh
		if e != nil {
			// return from this function closes target (and conn).
			return e
		}
	}
	return nil
}

// handleBind is used to handle a connect command
func (sf *Server) handleBind(_ context.Context, writer io.Writer, _ *Request) error {
	// TODO: Support bind
	if err := SendReply(writer, statute.RepCommandNotSupported, nil); err != nil {
		return fmt.Errorf("failed to send reply: %v", err)
	}
	return nil
}

// handleAssociate is used to handle a connect command
func (sf *Server) handleAssociate(ctx context.Context, writer io.Writer, request *Request) error {
	// Attempt to connect
	dial := sf.dial
	if dial == nil {
		dial = func(ctx context.Context, net_, addr string) (net.Conn, error) {
			return net.Dial(net_, addr)
		}
	}

	target, err := dial(ctx, "udp", request.DestAddr.String())
	if err != nil {
		msg := err.Error()
		resp := statute.RepHostUnreachable
		if strings.Contains(msg, "refused") {
			resp = statute.RepConnectionRefused
		} else if strings.Contains(msg, "network is unreachable") {
			resp = statute.RepNetworkUnreachable
		}
		if err := SendReply(writer, resp, nil); err != nil {
			return fmt.Errorf("failed to send reply, %v", err)
		}
		return fmt.Errorf("connect to %v failed, %v", request.RawDestAddr, err)
	}
	defer target.Close()

	bindLn, err := net.ListenUDP("udp", nil)
	if err != nil {
		if err := SendReply(writer, statute.RepServerFailure, nil); err != nil {
			return fmt.Errorf("failed to send reply, %v", err)
		}
		return fmt.Errorf("listen udp failed, %v", err)
	}
	defer bindLn.Close()

	sf.logger.Errorf("target addr %v, listen addr: %s", target.RemoteAddr(), bindLn.LocalAddr())
	// send BND.ADDR and BND.PORT, client used
	if err = SendReply(writer, statute.RepSuccess, bindLn.LocalAddr()); err != nil {
		return fmt.Errorf("failed to send reply, %v", err)
	}

	sf.goFunc(func() {
		// read from client and write to remote server
		conns := sync.Map{}
		bufPool := sf.bufferPool.Get()
		defer func() {
			target.Close()
			bindLn.Close()
			sf.bufferPool.Put(bufPool)
		}()
		for {
			n, srcAddr, err := bindLn.ReadFrom(bufPool[:cap(bufPool)])
			if err != nil {
				if err == io.EOF {
					return
				}
				if strings.Contains(err.Error(), "use of closed network connection") {
					sf.logger.Errorf("read data from bind listen address %s failed, %v", bindLn.LocalAddr(), err)
					return
				}
				continue
			}

			pk, err := statute.ParseDatagram(bufPool[:n])
			if err != nil {
				continue
			}

			if _, ok := conns.LoadOrStore(srcAddr.String(), struct{}{}); !ok {
				sf.goFunc(func() {
					// read from remote server and write to client
					bufPool := sf.bufferPool.Get()
					defer func() {
						target.Close()
						bindLn.Close()
						sf.bufferPool.Put(bufPool)
					}()

					for {
						buf := bufPool[:cap(bufPool)]
						n, err := target.Read(buf)
						if err != nil {
							if err == io.EOF {
								return
							}
							sf.logger.Errorf("read data from remote %s failed, %v", target.RemoteAddr().String(), err)
							return
						}

						pkb, err := statute.NewDatagram(target.RemoteAddr().String(), buf[:n])
						if err != nil {
							continue
						}
						tmpBufPool := sf.bufferPool.Get()
						proBuf := tmpBufPool
						proBuf = append(proBuf, pkb.Header()...)
						proBuf = append(proBuf, pkb.Data...)
						if _, err := bindLn.WriteTo(proBuf, srcAddr); err != nil {
							sf.bufferPool.Put(tmpBufPool)
							sf.logger.Errorf("write data to client %s failed, %v", bindLn.LocalAddr(), err)
							return
						}
						sf.bufferPool.Put(tmpBufPool)
					}
				})
			}

			// 把消息写给remote sever
			if _, err := target.Write(pk.Data); err != nil {
				sf.logger.Errorf("write data to remote %s failed, %v", target.RemoteAddr().String(), err)
				return
			}
		}
	})

	buf := sf.bufferPool.Get()
	defer sf.bufferPool.Put(buf)

	for {
		_, err := request.Reader.Read(buf[:cap(buf)])
		// sf.logger.Errorf("read data from client %s, %d bytesm, err is %+v", request.RemoteAddr.String(), num, err)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			if strings.Contains(err.Error(), "use of closed network connection") {
				return err
			}
		}
	}
}

// SendReply is used to send a reply message
// rep: reply status see statute's statute file
func SendReply(w io.Writer, rep uint8, bindAddr net.Addr) error {
	rsp := statute.Reply{
		Version:  statute.VersionSocks5,
		Response: rep,
		BndAddr: statute.AddrSpec{
			AddrType: statute.ATYPIPv4,
			IP:       net.IPv4zero,
			Port:     0,
		},
	}

	if rsp.Response == statute.RepSuccess {
		if tcpAddr, ok := bindAddr.(*net.TCPAddr); ok && tcpAddr != nil {
			rsp.BndAddr.IP = tcpAddr.IP
			rsp.BndAddr.Port = tcpAddr.Port
		} else if udpAddr, ok := bindAddr.(*net.UDPAddr); ok && udpAddr != nil {
			rsp.BndAddr.IP = udpAddr.IP
			rsp.BndAddr.Port = udpAddr.Port
		} else {
			rsp.Response = statute.RepAddrTypeNotSupported
		}

		if rsp.BndAddr.IP.To4() != nil {
			rsp.BndAddr.AddrType = statute.ATYPIPv4
		} else if rsp.BndAddr.IP.To16() != nil {
			rsp.BndAddr.AddrType = statute.ATYPIPv6
		}
	}
	// Send the message
	_, err := w.Write(rsp.Bytes())
	return err
}

type closeWriter interface {
	CloseWrite() error
}

// Proxy is used to suffle data from src to destination, and sends errors
// down a dedicated channel
func (sf *Server) Proxy(dst io.Writer, src io.Reader) error {
	buf := sf.bufferPool.Get()
	defer sf.bufferPool.Put(buf)
	_, err := io.CopyBuffer(dst, src, buf[:cap(buf)])
	if tcpConn, ok := dst.(closeWriter); ok {
		tcpConn.CloseWrite() //nolint: errcheck
	}
	return err
}
