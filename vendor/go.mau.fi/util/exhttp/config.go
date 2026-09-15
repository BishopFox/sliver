package exhttp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

type DialerFunc func(ctx context.Context, network, addr string) (net.Conn, error)

func (df DialerFunc) Dial(network, addr string) (net.Conn, error) {
	return df(context.Background(), network, addr)
}

func (df DialerFunc) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return df(ctx, network, addr)
}

type ClientSettings struct {
	DialTimeout  time.Duration
	Dial         DialerFunc
	innerDialer  DialerFunc
	HTTPProxy    func(*http.Request) (*url.URL, error)
	ProxyAddress string

	GlobalTimeout         time.Duration
	TLSHandshakeTimeout   time.Duration
	ResponseHeaderTimeout time.Duration
	IdleConnTimeout       time.Duration

	TLSConfig    *tls.Config
	InsecureTLS  bool
	DisableHTTP2 bool

	TransportOverride func(ClientSettings) http.RoundTripper
}

var SensibleClientSettings = ClientSettings{
	GlobalTimeout:         5 * time.Minute,
	TLSHandshakeTimeout:   10 * time.Second,
	ResponseHeaderTimeout: 10 * time.Second,
	IdleConnTimeout:       90 * time.Second,
}.WithDialTimeout(10 * time.Second)

// WithDial sets a custom dialer function for the HTTP client settings.
func (cs ClientSettings) WithDial(dial DialerFunc) ClientSettings {
	cs.Dial = dial
	cs.innerDialer = dial
	cs, _ = cs.WithProxy(cs.ProxyAddress)
	return cs
}

// WithDialTimeout sets a TCP dial timeout for the HTTP client. Resets any custom dialer.
func (cs ClientSettings) WithDialTimeout(timeout time.Duration) ClientSettings {
	cs.DialTimeout = timeout
	return cs.WithDial((&net.Dialer{Timeout: timeout}).DialContext)
}

func (cs ClientSettings) WithTLSHandshakeTimeout(timeout time.Duration) ClientSettings {
	cs.TLSHandshakeTimeout = timeout
	return cs
}

func (cs ClientSettings) WithResponseHeaderTimeout(timeout time.Duration) ClientSettings {
	cs.ResponseHeaderTimeout = timeout
	return cs
}

func (cs ClientSettings) WithIdleConnTimeout(timeout time.Duration) ClientSettings {
	cs.IdleConnTimeout = timeout
	return cs
}

func (cs ClientSettings) WithGlobalTimeout(timeout time.Duration) ClientSettings {
	cs.GlobalTimeout = timeout
	return cs
}

func (cs ClientSettings) WithoutHTTP2() ClientSettings {
	cs.DisableHTTP2 = true
	return cs
}

func (cs ClientSettings) WithTLSConfig(tlsConfig *tls.Config) ClientSettings {
	cs.TLSConfig = tlsConfig
	return cs
}

// WithProxy sets a proxy for the HTTP client. This will preserve any custom dialer set previously.
func (cs ClientSettings) WithProxy(addr string) (ClientSettings, error) {
	if addr == "" {
		cs.ProxyAddress = addr
		cs.HTTPProxy = nil
		cs.Dial = cs.innerDialer
		return cs, nil
	}
	parsedAddr, err := url.Parse(addr)
	if err != nil {
		return cs, err
	}
	switch parsedAddr.Scheme {
	case "http", "https":
		cs.HTTPProxy = http.ProxyURL(parsedAddr)
		cs.Dial = cs.innerDialer
		cs.ProxyAddress = addr
	case "socks5", "socks5h":
		socksProxy, err := proxy.FromURL(parsedAddr, cs.innerDialer)
		if err != nil {
			return cs, err
		}
		ctxDialer := socksProxy.(proxy.ContextDialer)
		cs.Dial = ctxDialer.DialContext
		cs.HTTPProxy = nil
		cs.ProxyAddress = addr
	default:
		return cs, fmt.Errorf("unsupported proxy scheme %q", parsedAddr.Scheme)
	}
	return cs, nil
}

// Configure configures the given HTTP transport with the settings in this struct.
// Note that the GlobalTimeout field is not applied on the transport level and needs to be set in the http.Client.
func (cs ClientSettings) Configure(transport *http.Transport) *http.Transport {
	if cs.Dial != nil {
		transport.DialContext = cs.Dial
	}
	if cs.HTTPProxy != nil {
		transport.Proxy = cs.HTTPProxy
	}
	if cs.TLSHandshakeTimeout != 0 {
		transport.TLSHandshakeTimeout = cs.TLSHandshakeTimeout
	}
	if cs.ResponseHeaderTimeout != 0 {
		transport.ResponseHeaderTimeout = cs.ResponseHeaderTimeout
	}
	if cs.IdleConnTimeout != 0 {
		transport.IdleConnTimeout = cs.IdleConnTimeout
	}
	if !cs.DisableHTTP2 {
		transport.ForceAttemptHTTP2 = true
	}
	if cs.TLSConfig != nil {
		transport.TLSClientConfig = cs.TLSConfig
	}
	if cs.InsecureTLS {
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{}
		}
		transport.TLSClientConfig.InsecureSkipVerify = true
	}
	return transport
}

func (cs ClientSettings) Compile() (c *http.Client) {
	c = &http.Client{
		Timeout: cs.GlobalTimeout,
	}
	if cs.TransportOverride != nil {
		c.Transport = cs.TransportOverride(cs)
	} else {
		c.Transport = cs.Configure(&http.Transport{})
	}
	return
}
