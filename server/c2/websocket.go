package c2

import (
	"encoding/base64"
	"fmt"
	"net/http"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/cryptography"
	"github.com/bishopfox/sliver/server/db"
	serverHandlers "github.com/bishopfox/sliver/server/handlers"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/util/encoders"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

var (
	wssLog = log.NamedLogger("c2", consts.WsStr)
)
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (s *SliverHTTPC2) acceptWebSocketConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		wssLog.Errorf("Accept failed: %v", err)
		return
	}
	cipherCtx, err := GetsKey(conn)
	if err != nil {
		conn.Close()
		wssLog.Errorf("Accept failed: %v", err)
		return
	}
	wssLog.Infof("GetsKey is ok ")
	go handleWebSocketConnection(conn, cipherCtx)
}

func GetsKey(conn *websocket.Conn) (*cryptography.CipherContext, error) {
	_, originMessage, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	dataLengthBuf := originMessage[:4]
	nonce := encoders.BytesToInt(dataLengthBuf)
	_, encoder, err := encoders.EncoderFromNonce(nonce)
	if err != nil {
		return nil, err
	}
	message, err := encoder.Decode(originMessage[4:])
	if err != nil {
		return nil, err
	}

	var publicKeyDigest [32]byte
	copy(publicKeyDigest[:], message[:32])
	implantConfig, err := db.ImplantConfigByECCPublicKeyDigest(publicKeyDigest)
	if err != nil || implantConfig == nil {
		httpLog.Warn("Unknown public key")
		return nil, err
	}
	publicKey, err := base64.RawStdEncoding.DecodeString(implantConfig.ECCPublicKey)
	if err != nil || len(publicKey) != 32 {
		wssLog.Warn("Failed to decode public key")
		return nil, err
	}
	var senderPublicKey [32]byte
	copy(senderPublicKey[:], publicKey)

	serverKeyPair := cryptography.ECCServerKeyPair()
	sessionInitData, err := cryptography.ECCDecrypt(&senderPublicKey, serverKeyPair.Private, message[32:])
	if err != nil {
		wssLog.Error("ECC decryption failed")
		wssLog.Error(fmt.Sprintf("senderPublicKey %#v Private: %#v data: %#v", senderPublicKey, serverKeyPair.Private, message[32:]))
		return nil, err
	}
	sessionInit := &sliverpb.HTTPSessionInit{}
	err = proto.Unmarshal(sessionInitData, sessionInit)
	if err != nil {
		wssLog.Error("Failed to decode session init")
		return nil, err
	}

	sKey, err := cryptography.KeyFromBytes(sessionInit.Key)
	if err != nil {
		wssLog.Error("Failed to convert bytes to session key")
		return nil, err
	}

	err = conn.WriteMessage(websocket.TextMessage, originMessage)
	if err != nil {
		return nil, err
	}
	return cryptography.NewCipherContext(sKey), nil
}

func handleWebSocketConnection(conn *websocket.Conn, cipherCtx *cryptography.CipherContext) {
	wssLog.Infof("Accepted incoming connection: %s", conn.RemoteAddr())
	implantConn := core.NewImplantConnection(consts.WsStr, conn.RemoteAddr().String())

	defer func() {
		wssLog.Debugf("websocket connection closing")
		conn.Close()
		implantConn.Cleanup()
	}()

	done := make(chan bool)
	go func() {
		defer func() {
			done <- true
		}()
		handlers := serverHandlers.GetHandlers()
		for {
			envelope, err := socketWSSReadEnvelope(conn, cipherCtx)
			if err != nil {
				wssLog.Errorf("Socket read error %v", err)
				return
			}
			implantConn.UpdateLastMessage()
			if envelope.ID != 0 {
				wssLog.Debugf("Resp %s new websocket message type %d, ID: %d", conn.RemoteAddr(), envelope.Type, envelope.ID)
				implantConn.RespMutex.RLock() // mutex
				if resp, ok := implantConn.Resp[envelope.ID]; ok {
					resp <- envelope // Could deadlock, maybe want to investigate better solutions
				}
				wssLog.Debugf("Resp %s new websocket message type %d, ID: %d is ok", conn.RemoteAddr(), envelope.Type, len(envelope.Data))
				implantConn.RespMutex.RUnlock()
			} else if handler, ok := handlers[envelope.Type]; ok {
				wssLog.Debugf("Received %s new websocket message type %d, data: %d", conn.RemoteAddr(), envelope.Type, len(envelope.Data))
				go func() {
					respEnvelope := handler(implantConn, envelope.Data)
					if respEnvelope != nil {
						implantConn.Send <- respEnvelope
					}
				}()
			}
		}
	}()

Loop:
	for {
		select {
		case envelope := <-implantConn.Send:
			err := socketWSSWriteEnvelope(conn, envelope, cipherCtx)
			if err != nil {
				wssLog.Errorf("Socket write failed %v", err)
				break Loop
			}
		case <-done:
			break Loop
		}
	}
	wssLog.Debugf("Closing implant connection %s", implantConn.ID)
}

// socketWSSWriteEnvelope - Writes a message to the TLS socket using length prefix framing
// which is a fancy way of saying we write the length of the message then the message
// e.g. [uint32 length|message] so the receiver can delimit messages properly
func socketWSSWriteEnvelope(connection *websocket.Conn, envelope *sliverpb.Envelope, cipherCtx *cryptography.CipherContext) error {
	data, err := proto.Marshal(envelope)
	if err != nil {
		wssLog.Errorf("Envelope marshaling error: %v", err)
		return err
	}
	// 加密
	data, err = cipherCtx.Encrypt(data)
	if err != nil {
		return err
	}

	// 编码
	nonce, encoder := encoders.RandomEncoder()
	payload := encoder.Encode(data)
	encoderFlag := encoders.IntToBytes(nonce)
	payload = append(encoderFlag, payload...)

	connection.WriteMessage(websocket.TextMessage, payload)
	return nil
}

// socketWSSReadEnvelope - Reads a message from the TLS connection using length prefix framing
// returns messageType, message, and error
func socketWSSReadEnvelope(connection *websocket.Conn, cipherCtx *cryptography.CipherContext) (*sliverpb.Envelope, error) {
	// Read the first four bytes to determine data length
	_, message, err := connection.ReadMessage()
	if err != nil {
		return nil, err
	}
	dataLengthBuf := message[:4]
	nonce := encoders.BytesToInt(dataLengthBuf)
	_, encoder, err := encoders.EncoderFromNonce(nonce)
	if err != nil {
		return nil, err
	}
	message, err = encoder.Decode(message[4:])
	if err != nil {
		return nil, err
	}

	// decrypt data
	data, err := cipherCtx.Decrypt(message)
	if err != nil {
		return nil, err
	}

	// Unmarshal the protobuf envelope
	wssLog.Infof("message: %d", len(message))
	envelope := &sliverpb.Envelope{}
	err = proto.Unmarshal(data, envelope)
	if err != nil {
		wssLog.Errorf("Un-marshaling envelope error: %v", err)
		return nil, err
	}
	return envelope, nil
}
