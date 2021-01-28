package c2

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

// import (
//         "context"
//         "fmt"
//         "net"
//
//         "github.com/bishopfox/sliver/server/certs"
//         "github.com/bishopfox/sliver/server/comm"
//         "github.com/lucas-clemente/quic-go"
// )

// DialSliverQUIC - Get a C2 implant session over QUIC + Mutual TLS by dialing a UDP destination.
// func DialSliverQUIC(bindIface string, port uint16) (err error) {
//         host := bindIface
//         if host == "" {
//                 host = defaultServerCert
//         }
//
//         // Certificates should already be available because we compiled the implant, but in case...
//         _, _, err = certs.GetCertificate(certs.C2ServerCA, certs.ECCKey, host)
//         if err != nil {
//                 certs.C2ServerGenerateECCCertificate(host)
//         }
//
//         // Load appropriate client MTLS configuration.
//         creds := newCredentialsTLS()
//         tlsConfig := creds.ClientConfig(host)
//
//         // Dial QUIC, which gives a session that might create multiple independent streams/connections.
//         session, err := quic.DialAddr(fmt.Sprintf("%s:%d", host, port), tlsConfig, nil)
//         if err != nil {
//                 return err
//         }
//
//         // Get a single QUIC stream from the session.
//         stream, err := session.OpenStreamSync(context.Background())
//         if err != nil {
//                 return nil
//         }
//
//         // Wrap the QUIC stream into a QUIC conn custom (for satisfying net.Conn interface)
//         conn := newConnQUIC(stream, session.LocalAddr(), session.RemoteAddr())
//
//         // The QUIC stream satisfies the net.Conn interface, so we handle it
//         // by simply adding the read/write RPC loop around this connection, like MTLS
//         go handleSliverConnection(conn)
//
//         return
// }
//
// // StartQUICListenerComm - Start a QUIC listener working with the Comm system.
// // The undelying transport protocol is UDP, so any implant may yield us a UDP connection.
// func StartQUICListenerComm(bindIface string, port uint16) (err error) {
//         StartPivotListener()
//
//         host := bindIface
//         if host == "" {
//                 host = defaultServerCert
//         }
//
//         // Certificates should already be available because we compiled the implant, but in case...
//         _, _, err = certs.GetCertificate(certs.C2ServerCA, certs.ECCKey, host)
//         if err != nil {
//                 certs.C2ServerGenerateECCCertificate(host)
//         }
//
//         // Load the TLS configuration for a server position.
//         tlsConfig := newCredentialsTLS().ServerConfig(host)
//
//         mtlsLog.Infof("Starting routed Raw UDP/QUIC listener on %s:%d", bindIface, port)
//
//         // Get a UDP listener from the comm system. The UDP connection  might be
//         // initiated from one of the implants, which then route back the UDP traffic.
//         packetConn, err := comm.ListenUDP("udp", fmt.Sprintf("%s:%d", host, port))
//         if err != nil {
//                 return err
//         }
//
//         // "Wrap" a QUIC listener around the packet conn, waiting for QUIC client connections.
//         ln, err := quic.Listen(packetConn, tlsConfig, nil)
//         if err != nil {
//                 return err
//         }
//
//         // Handle incoming QUIC connections in the background.
//         go acceptSliverConnectionsQUIC(ln)
//
//         return
// }
//
// func acceptSliverConnectionsQUIC(ln quic.Listener) {
//         for {
//                 // Accept a QUIC client connection on the packet conn QUIC listener
//                 session, err := ln.Accept(context.Background())
//                 if err != nil {
//                         // Normally errors should satisfy the net.Error interface, per QUIC documentation.
//                         if errType, ok := err.(*net.OpError); ok && errType.Op == "accept" {
//                                 mtlsLog.Errorf("QUIC Accept failed: %v", err)
//                                 break
//                         }
//                         continue
//                 }
//
//                 // Accept a single stream (one per Sliver implant)
//                 stream, err := session.AcceptStream(context.Background())
//                 if err != nil {
//                         mtlsLog.Errorf("QUIC Session Accept failed: %v", err)
//                         break
//                 }
//
//                 // Wrap the QUIC stream into a QUIC conn custom (for satisfying net.Conn interface)
//                 conn := newConnQUIC(stream, session.LocalAddr(), session.RemoteAddr())
//
//                 // Handle the stream like any other net.Conn, adding a
//                 // read/write RPC loop layer around the connection.
//                 go handleSliverConnection(conn)
//         }
// }
//
// type connQUIC struct {
//         quic.Stream
//         lAddr net.Addr
//         rAddr net.Addr
// }
//
// func newConnQUIC(stream quic.Stream, lAddr, rAddr net.Addr) *connQUIC {
//         qc := &connQUIC{
//                 stream,
//                 lAddr,
//                 rAddr,
//         }
//         return qc
// }
//
// func (qc *connQUIC) LocalAddr() net.Addr {
//         return qc.lAddr
// }
//
// func (qc *connQUIC) RemoteAddr() net.Addr {
//         return qc.rAddr
// }
