package transport

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"sync"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/core"
	servernetstack "github.com/bishopfox/sliver/server/netstack"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"google.golang.org/grpc"
)

const (
	multiplayerWireGuardMTU = 1420
)

type wireGuardWrappedClientListener struct {
	net.Listener
	dev    *device.Device
	events chan core.Event
	once   sync.Once
}

func (l *wireGuardWrappedClientListener) Close() error {
	var err error
	l.once.Do(func() {
		if l.events != nil {
			core.EventBroker.Unsubscribe(l.events)
			l.events = nil
		}
		if l.Listener != nil {
			err = l.Listener.Close()
		}
		if l.dev != nil {
			l.dev.Close()
		}
	})
	return err
}

func (l *wireGuardWrappedClientListener) processPeerEvents() {
	if l == nil || l.events == nil || l.dev == nil {
		return
	}

	for event := range l.events {
		switch event.EventType {
		case consts.MultiplayerWireGuardNewPeer, consts.MultiplayerWireGuardRemoved:
			if len(event.Data) == 0 {
				continue
			}
			if err := l.dev.IpcSetOperation(bufio.NewReader(bytes.NewReader(event.Data))); err != nil {
				mtlsLog.Errorf("Failed to update multiplayer wireguard config: %v", err)
			}
		}
	}
}

// StartWGWrappedMtlsClientListener exposes the multiplayer mTLS listener only
// inside a WireGuard tunnel while preserving the existing gRPC/mTLS auth stack.
func StartWGWrappedMtlsClientListener(host string, port uint16) (*grpc.Server, net.Listener, error) {
	mtlsLog.Infof("Starting gRPC/mtls listener on %s:%d inside WireGuard", host, port)

	tunAddr := netip.MustParseAddr(certs.MultiplayerWireGuardServerIP)
	tun, tNet, err := servernetstack.CreateNetTUN([]netip.Addr{tunAddr}, nil, multiplayerWireGuardMTU)
	if err != nil {
		return nil, nil, err
	}

	if err := tNet.AllowTCPPort(tunAddr, port); err != nil {
		_ = tun.Close()
		return nil, nil, err
	}

	privateKey, _, err := certs.GetMultiplayerWGServerKeys()
	if err == certs.ErrMultiplayerWGServerKeysDoNotExist {
		certs.SetupMultiplayerWGKeys()
		privateKey, _, err = certs.GetMultiplayerWGServerKeys()
	}
	if err != nil {
		_ = tun.Close()
		return nil, nil, err
	}

	dev := device.NewDevice(tun, conn.NewDefaultBind(), device.NewLogger(device.LogLevelSilent, "[transport/multiplayer-wg] "))
	wgConf := bytes.NewBuffer(nil)
	fmt.Fprintf(wgConf, "private_key=%s\n", privateKey)
	fmt.Fprintf(wgConf, "listen_port=%d\n", port)

	if err := appendOperatorWGPeers(wgConf); err != nil {
		dev.Close()
		return nil, nil, err
	}
	if err := dev.IpcSetOperation(bufio.NewReader(wgConf)); err != nil {
		dev.Close()
		return nil, nil, err
	}
	if err := dev.Up(); err != nil {
		dev.Close()
		return nil, nil, err
	}

	innerListener, err := tNet.ListenTCP(&net.TCPAddr{IP: net.ParseIP(certs.MultiplayerWireGuardServerIP), Port: int(port)})
	if err != nil {
		dev.Close()
		return nil, nil, err
	}

	wrappedListener := &wireGuardWrappedClientListener{
		Listener: innerListener,
		dev:      dev,
		events:   core.EventBroker.Subscribe(),
	}
	go wrappedListener.processPeerEvents()

	grpcServer, err := StartMtlsClientServer(wrappedListener)
	if err != nil {
		_ = wrappedListener.Close()
		return nil, nil, err
	}
	return grpcServer, wrappedListener, nil
}

func appendOperatorWGPeers(dst *bytes.Buffer) error {
	peers, err := certs.GetOperatorWGPeers()
	if err != nil && err != certs.ErrWGPeerDoesNotExist {
		return err
	}
	for publicKey, tunIP := range peers {
		if strings.TrimSpace(publicKey) == "" || strings.TrimSpace(tunIP) == "" {
			continue
		}
		if _, err := netip.ParseAddr(tunIP); err != nil {
			mtlsLog.Warnf("Skipping multiplayer wireguard peer %q with invalid tunnel IP %q: %v", publicKey, tunIP, err)
			continue
		}
		fmt.Fprintf(dst, "public_key=%s\n", publicKey)
		fmt.Fprintf(dst, "allowed_ip=%s/32\n", tunIP)
	}
	return nil
}
