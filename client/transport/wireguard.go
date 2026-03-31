package transport

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"google.golang.org/grpc"
)

const (
	multiplayerWireGuardDefaultServerIP = "100.65.0.1"
	multiplayerWireGuardMTU             = 1420
	multiplayerWireGuardKeepalive       = 25
	multiplayerWireGuardDialTimeout     = 30 * time.Second
	multiplayerWireGuardRetryDelay      = 250 * time.Millisecond
)

var (
	ErrMissingWireGuardConfig    = errors.New("operator config has no wg block")
	ErrIncompleteWireGuardConfig = errors.New("operator config has incomplete wg block")
)

type wireGuardTunnel struct {
	dev *device.Device
	net *transportNet

	closeOnce sync.Once
}

func (t *wireGuardTunnel) Close() error {
	if t == nil {
		return nil
	}
	t.closeOnce.Do(func() {
		if t.dev != nil {
			t.dev.Close()
		}
	})
	return nil
}

func (t *wireGuardTunnel) DialContext(ctx context.Context, address string) (net.Conn, error) {
	if t == nil || t.net == nil {
		return nil, errors.New("wireguard tunnel is not initialized")
	}
	return t.net.DialContext(ctx, "tcp", address)
}

func wireGuardMTLSConnect(config *assets.ClientConfig, statusFn ConnectStatusFn) (rpcpb.SliverRPCClient, *grpc.ClientConn, error) {
	notifyConnectStatus(statusFn, connectStatusWireGuard)
	tunnel, target, err := newWireGuardTunnel(config)
	if err != nil {
		return nil, nil, err
	}

	options, err := newMTLSDialOptions(config)
	if err != nil {
		_ = tunnel.Close()
		return nil, nil, err
	}
	options = append(options, grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		return tunnel.DialContext(ctx, addr)
	}))

	notifyConnectStatus(statusFn, connectStatusGRPCMTLSOverWireGuard)
	rpcClient, conn, err := dialWireGuardRPCClient(target, options, tunnel)
	if err != nil {
		_ = tunnel.Close()
		return nil, nil, err
	}
	return rpcClient, conn, nil
}

func dialWireGuardRPCClient(target string, options []grpc.DialOption, tunnel connectionCloser) (rpcpb.SliverRPCClient, *grpc.ClientConn, error) {
	deadline := time.Now().Add(multiplayerWireGuardDialTimeout)
	var lastErr error
	attempts := 0

	for {
		attempts++

		rpcClient, conn, err := dialRPCClient(target, options, nil)
		if err == nil {
			registerConnCloser(conn, tunnel)
			return rpcClient, conn, nil
		}

		lastErr = err
		if !errors.Is(err, context.DeadlineExceeded) || !time.Now().Before(deadline) {
			break
		}
		time.Sleep(multiplayerWireGuardRetryDelay)
	}

	if attempts > 1 {
		return nil, nil, fmt.Errorf("wireguard multiplayer connect failed after %d attempts: %w", attempts, lastErr)
	}
	return nil, nil, lastErr
}

func newWireGuardTunnel(config *assets.ClientConfig) (*wireGuardTunnel, string, error) {
	if err := validateWireGuardConfig(config); err != nil {
		return nil, "", err
	}
	if config.LPort <= 0 || 65535 < config.LPort {
		return nil, "", fmt.Errorf("invalid multiplayer port %d", config.LPort)
	}

	clientIP, err := netip.ParseAddr(strings.TrimSpace(config.WG.ClientIP))
	if err != nil {
		return nil, "", fmt.Errorf("invalid wireguard client IP %q: %w", config.WG.ClientIP, err)
	}

	serverIPValue := strings.TrimSpace(config.WG.ServerIP)
	if serverIPValue == "" {
		serverIPValue = multiplayerWireGuardDefaultServerIP
	}
	serverIP, err := netip.ParseAddr(serverIPValue)
	if err != nil {
		return nil, "", fmt.Errorf("invalid wireguard server IP %q: %w", serverIPValue, err)
	}

	endpoint, err := resolveWireGuardEndpoint(config.LHost, config.LPort)
	if err != nil {
		return nil, "", err
	}

	tun, tNet, err := createTransportNetTUN([]netip.Addr{clientIP}, multiplayerWireGuardMTU)
	if err != nil {
		return nil, "", err
	}

	dev := device.NewDevice(tun, conn.NewDefaultBind(), device.NewLogger(device.LogLevelSilent, "[client/wg] "))
	wgConf := bytes.NewBuffer(nil)
	fmt.Fprintf(wgConf, "private_key=%s\n", strings.TrimSpace(config.WG.ClientPrivateKey))
	fmt.Fprintf(wgConf, "public_key=%s\n", strings.TrimSpace(config.WG.ServerPubKey))
	fmt.Fprintf(wgConf, "endpoint=%s\n", endpoint.String())
	fmt.Fprintf(wgConf, "allowed_ip=%s\n", multiplayerWireGuardAllowedIP(serverIP))
	fmt.Fprintf(wgConf, "persistent_keepalive_interval=%d\n", multiplayerWireGuardKeepalive)

	if err := dev.IpcSetOperation(bufio.NewReader(wgConf)); err != nil {
		dev.Close()
		return nil, "", err
	}
	if err := dev.Up(); err != nil {
		dev.Close()
		return nil, "", err
	}

	target := net.JoinHostPort(serverIP.String(), strconv.Itoa(config.LPort))
	return &wireGuardTunnel{dev: dev, net: tNet}, target, nil
}

func validateWireGuardConfig(config *assets.ClientConfig) error {
	if config == nil {
		return errors.New("client config is required")
	}
	if config.WG == nil {
		return ErrMissingWireGuardConfig
	}

	missing := make([]string, 0, 3)
	if strings.TrimSpace(config.WG.ServerPubKey) == "" {
		missing = append(missing, "server_pub_key")
	}
	if strings.TrimSpace(config.WG.ClientPrivateKey) == "" {
		missing = append(missing, "client_private_key")
	}
	if strings.TrimSpace(config.WG.ClientIP) == "" {
		missing = append(missing, "client_ip")
	}
	if len(missing) != 0 {
		return fmt.Errorf("%w: missing %s", ErrIncompleteWireGuardConfig, strings.Join(missing, ", "))
	}
	return nil
}

func resolveWireGuardEndpoint(host string, port int) (netip.AddrPort, error) {
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if host == "" {
		return netip.AddrPort{}, errors.New("wireguard endpoint host is required")
	}

	if addr, err := netip.ParseAddr(host); err == nil {
		return netip.AddrPortFrom(addr, uint16(port)), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return netip.AddrPort{}, fmt.Errorf("failed to resolve wireguard endpoint %q: %w", host, err)
	}
	for _, ip := range ips {
		if ip.IsValid() {
			return netip.AddrPortFrom(ip, uint16(port)), nil
		}
	}
	return netip.AddrPort{}, fmt.Errorf("wireguard endpoint %q did not resolve to a usable IP", host)
}

func multiplayerWireGuardAllowedIP(addr netip.Addr) string {
	if addr.Is6() {
		return fmt.Sprintf("%s/128", addr)
	}
	return fmt.Sprintf("%s/32", addr)
}
