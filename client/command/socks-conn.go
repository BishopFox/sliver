package command

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

const (
	socksServerVersion = 5
	tcpChunkSize = 4096

	noAuthMethodType = 0x00

	connectCommand = 0x01
	bindCommand = 0x02
	udpCommand = 0x03

	ipv4AddressType = 0x01
	domainNameAddressType = 0x03
	ipv6AddressType = 0x04

	successfulConnectReply = 0x00
	failedConnectReply = 0x01

	reservedValue = 0x00
)

type SocksConn struct {
	ClientConn 			*net.TCPConn
	RemoteHost 			string
	RemotePort			uint32
	AuthMethod 			byte
	SocksCommand 		byte
	AddressType 		byte
}

func (conn *SocksConn) Close() {
	err := conn.ClientConn.Close()
	if err != nil {
		fmt.Printf("%s\n", err)
	}
}

func (conn *SocksConn) HandleAuthRequest() error {
	// --- Socks Auth Negotiation ---
	initRequest := make([]byte, tcpChunkSize)
	bytesRead, err := conn.ClientConn.Read(initRequest)

	if err != nil {
		return err
	}

	initRequest = initRequest[:bytesRead]

	socksVersion := initRequest[0]
	authMethodsCount := initRequest[1]
	authMethods := initRequest[2:2+authMethodsCount]

	if socksVersion != socksServerVersion {
		return errors.New(fmt.Sprintf("This socks version is not support (%d) \n", socksVersion))
	}

	if !contains(authMethods, noAuthMethodType) {
		return errors.New(fmt.Sprintf("No auth is not supported by the client - %s", authMethods))
	}

	initResponse := make([]byte, 2)
	initResponse[0] = socksVersion
	initResponse[1] = noAuthMethodType
	bytesWritten, err := conn.ClientConn.Write(initResponse)
	if err != nil {
		return err
	}
	if bytesWritten != len(initResponse) {
		return errors.New("Written bytes count does not match message size")
	}
	return nil
}

func (conn *SocksConn) HandleConnectRequest() error {
	// --- Connect request
	connectRequest := make([]byte, tcpChunkSize)
	bytesRead, err := conn.ClientConn.Read(connectRequest)
	if err != nil {
		return err
	}

	connectRequest = connectRequest[:bytesRead]

	socksCmd := connectRequest[1]
	addressType := connectRequest[3]

	if socksCmd != connectCommand {
		return errors.New(fmt.Sprintf("This socks command is not supported (%d) \n", socksCmd))
	}

	if addressType == ipv4AddressType {
		remoteAddressStringArray := make([]string, 0)
		for _, element := range connectRequest[4:8] {
			remoteAddressStringArray = append(remoteAddressStringArray, strconv.Itoa(int(element)))
		}
		conn.RemoteHost = strings.Join(remoteAddressStringArray, ".")
		conn.RemotePort = uint32(binary.BigEndian.Uint16(connectRequest[8:10]))
		return nil
	} else if addressType == domainNameAddressType {
		domainNameLength := connectRequest[4]

		conn.RemoteHost = string(connectRequest[5:5+domainNameLength])
		conn.RemotePort = uint32(binary.BigEndian.Uint16(connectRequest[5+domainNameLength:5+domainNameLength+2]))
		return nil
	} else {
		return errors.New(fmt.Sprintf("This address type is not supported (%d) \n", addressType))
	}
}

func (conn *SocksConn) ReturnSuccessConnectMessage() error {
	connectReply := make([]byte, 4)
	connectReply[0] = socksServerVersion
	connectReply[1] = successfulConnectReply
	connectReply[2] = reservedValue
	connectReply[3] = ipv4AddressType

	localAddressArray := []byte{127,0,0,1}
	connectReply = append(connectReply, localAddressArray...)

	localPortArray := []byte{0,0}
	connectReply = append(connectReply, localPortArray...)

	bytesWritten, err := conn.ClientConn.Write(connectReply)
	if err != nil {
		return err
	}
	if bytesWritten != len(connectReply) {
		return errors.New("Bytes written does not equal to the size of the message")
	}

	return nil
}

func (conn *SocksConn) ReturnFailureConnectMessage() error {
	connectReply := make([]byte, 4)
	connectReply[0] = socksServerVersion
	connectReply[1] = failedConnectReply
	connectReply[2] = reservedValue
	connectReply[3] = ipv4AddressType

	localAddressArray := []byte{127, 0, 0, 1}
	connectReply = append(connectReply, localAddressArray...)

	localPortArray := []byte{0,0}
	connectReply = append(connectReply, localPortArray...)

	bytesWritten, err := conn.ClientConn.Write(connectReply)
	if err != nil {
		return err
	}
	if bytesWritten != len(connectReply) {
		return errors.New("Bytes written does not equal to the size of the message")
	}

	return nil
}

func contains(arr []byte, b byte) bool {
	for _, a := range arr {
		if a == b {
			return true
		}
	}
	return false
}