// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

package ipnlocal

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/netip"
	"net/url"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"tailscale.com/ipn"
	"tailscale.com/logtail/backoff"
	"tailscale.com/net/netutil"
	"tailscale.com/syncs"
	"tailscale.com/tailcfg"
	"tailscale.com/types/logger"
	"tailscale.com/util/mak"
	"tailscale.com/version"
)

// ErrETagMismatch signals that the given
// If-Match header does not match with the
// current etag of a resource.
var ErrETagMismatch = errors.New("etag mismatch")

// serveHTTPContextKey is the context.Value key for a *serveHTTPContext.
type serveHTTPContextKey struct{}

type serveHTTPContext struct {
	SrcAddr  netip.AddrPort
	DestPort uint16
}

// serveListener is the state of host-level net.Listen for a specific (Tailscale IP, serve port)
// combination. If there are two TailscaleIPs (v4 and v6) and three ports being served,
// then there will be six of these active and looping in their Run method.
//
// This is not used in userspace-networking mode.
//
// Most serve traffic is intercepted by netstack. This exists purely for connections
// from the machine itself, as that goes via the kernel, so we need to be in the
// kernel's listening/routing tables.
type serveListener struct {
	b      *LocalBackend
	ap     netip.AddrPort
	ctx    context.Context    // valid while listener is desired
	cancel context.CancelFunc // for ctx, to close listener
	logf   logger.Logf
	bo     *backoff.Backoff // for retrying failed Listen calls

	closeListener syncs.AtomicValue[func() error] // Listener's Close method, if any
}

func (b *LocalBackend) newServeListener(ctx context.Context, ap netip.AddrPort, logf logger.Logf) *serveListener {
	ctx, cancel := context.WithCancel(ctx)
	return &serveListener{
		b:      b,
		ap:     ap,
		ctx:    ctx,
		cancel: cancel,
		logf:   logf,

		bo: backoff.NewBackoff("serve-listener", logf, 30*time.Second),
	}
}

// Close cancels the context and closes the listener, if any.
func (s *serveListener) Close() error {
	s.cancel()
	if close, ok := s.closeListener.LoadOk(); ok {
		s.closeListener.Store(nil)
		close()
	}
	return nil
}

// Run starts a net.Listen for the serveListener's address and port.
// If unable to listen, it retries with exponential backoff.
// Listen is retried until the context is canceled.
func (s *serveListener) Run() {
	for {
		ip := s.ap.Addr()
		ipStr := ip.String()

		var lc net.ListenConfig
		if initListenConfig != nil {
			// On macOS, this sets the lc.Control hook to
			// setsockopt the interface index to bind to. This is
			// required by the network sandbox to allow binding to
			// a specific interface. Without this hook, the system
			// chooses a default interface to bind to.
			if err := initListenConfig(&lc, ip, s.b.prevIfState, s.b.dialer.TUNName()); err != nil {
				s.logf("serve failed to init listen config %v, backing off: %v", s.ap, err)
				s.bo.BackOff(s.ctx, err)
				continue
			}
			// On macOS (AppStore or macsys) and if we're binding to a privileged port,
			if version.IsSandboxedMacOS() && s.ap.Port() < 1024 {
				// On macOS, we need to bind to ""/all-interfaces due to
				// the network sandbox. Ideally we would only bind to the
				// Tailscale interface, but macOS errors out if we try to
				// to listen on privileged ports binding only to a specific
				// interface. (#6364)
				ipStr = ""
			}
		}

		tcp4or6 := "tcp4"
		if ip.Is6() {
			tcp4or6 = "tcp6"
		}

		ln, err := lc.Listen(s.ctx, tcp4or6, net.JoinHostPort(ipStr, fmt.Sprint(s.ap.Port())))
		if err != nil {
			if s.shouldWarnAboutListenError(err) {
				s.logf("serve failed to listen on %v, backing off: %v", s.ap, err)
			}
			s.bo.BackOff(s.ctx, err)
			continue
		}
		s.closeListener.Store(ln.Close)

		s.logf("serve listening on %v", s.ap)
		err = s.handleServeListenersAccept(ln)
		if s.ctx.Err() != nil {
			// context canceled, we're done
			return
		}
		if err != nil {
			s.logf("serve listener accept error, retrying: %v", err)
		}
	}
}

func (s *serveListener) shouldWarnAboutListenError(err error) bool {
	if !s.b.sys.NetMon.Get().InterfaceState().HasIP(s.ap.Addr()) {
		// Machine likely doesn't have IPv6 enabled (or the IP is still being
		// assigned). No need to warn. Notably, WSL2 (Issue 6303).
		return false
	}
	// TODO(bradfitz): check errors.Is(err, syscall.EADDRNOTAVAIL) etc? Let's
	// see what happens in practice.
	return true
}

// handleServeListenersAccept accepts connections for the Listener.
// Calls incoming handler in a new goroutine for each accepted connection.
func (s *serveListener) handleServeListenersAccept(ln net.Listener) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		srcAddr := conn.RemoteAddr().(*net.TCPAddr).AddrPort()
		handler := s.b.tcpHandlerForServe(s.ap.Port(), srcAddr)
		if handler == nil {
			s.b.logf("serve RST for %v", srcAddr)
			conn.Close()
			continue
		}
		go handler(conn)
	}
}

// updateServeTCPPortNetMapAddrListenersLocked starts a net.Listen for configured
// Serve ports on all the node's addresses.
// Existing Listeners are closed if port no longer in incoming ports list.
//
// b.mu must be held.
func (b *LocalBackend) updateServeTCPPortNetMapAddrListenersLocked(ports []uint16) {
	// close existing listeners where port
	// is no longer in incoming ports list
	for ap, sl := range b.serveListeners {
		if !slices.Contains(ports, ap.Port()) {
			b.logf("closing listener %v", ap)
			sl.Close()
			delete(b.serveListeners, ap)
		}
	}

	nm := b.netMap
	if nm == nil {
		b.logf("netMap is nil")
		return
	}
	if !nm.SelfNode.Valid() {
		b.logf("netMap SelfNode is nil")
		return
	}

	addrs := nm.GetAddresses()
	for i := range addrs.LenIter() {
		a := addrs.At(i)
		for _, p := range ports {
			addrPort := netip.AddrPortFrom(a.Addr(), p)
			if _, ok := b.serveListeners[addrPort]; ok {
				continue // already listening
			}

			sl := b.newServeListener(context.Background(), addrPort, b.logf)
			mak.Set(&b.serveListeners, addrPort, sl)

			go sl.Run()
		}
	}
}

// SetServeConfig establishes or replaces the current serve config.
// ETag is an optional parameter to enforce Optimistic Concurrency Control.
// If it is an empty string, then the config will be overwritten.
func (b *LocalBackend) SetServeConfig(config *ipn.ServeConfig, etag string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.setServeConfigLocked(config, etag)
}

func (b *LocalBackend) setServeConfigLocked(config *ipn.ServeConfig, etag string) error {
	prefs := b.pm.CurrentPrefs()
	if config.IsFunnelOn() && prefs.ShieldsUp() {
		return errors.New("Unable to turn on Funnel while shields-up is enabled")
	}

	nm := b.netMap
	if nm == nil {
		return errors.New("netMap is nil")
	}
	if !nm.SelfNode.Valid() {
		return errors.New("netMap SelfNode is nil")
	}

	// If etag is present, check that it has
	// not changed from the last config.
	if etag != "" {
		// Note that we marshal b.serveConfig
		// and not use b.lastServeConfJSON as that might
		// be a Go nil value, which produces a different
		// checksum from a JSON "null" value.
		previousCfg, err := json.Marshal(b.serveConfig)
		if err != nil {
			return fmt.Errorf("error encoding previous config: %w", err)
		}
		sum := sha256.Sum256(previousCfg)
		previousEtag := hex.EncodeToString(sum[:])
		if etag != previousEtag {
			return ErrETagMismatch
		}
	}

	var bs []byte
	if config != nil {
		j, err := json.Marshal(config)
		if err != nil {
			return fmt.Errorf("encoding serve config: %w", err)
		}
		bs = j
	}

	profileID := b.pm.CurrentProfile().ID
	confKey := ipn.ServeConfigKey(profileID)
	if err := b.store.WriteState(confKey, bs); err != nil {
		return fmt.Errorf("writing ServeConfig to StateStore: %w", err)
	}

	b.setTCPPortsInterceptedFromNetmapAndPrefsLocked(b.pm.CurrentPrefs())
	return nil
}

// ServeConfig provides a view of the current serve mappings.
// If serving is not configured, the returned view is not Valid.
func (b *LocalBackend) ServeConfig() ipn.ServeConfigView {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.serveConfig
}

// DeleteForegroundSession deletes a ServeConfig's foreground session
// in the LocalBackend if it exists. It also ensures check, delete, and
// set operations happen within the same mutex lock to avoid any races.
func (b *LocalBackend) DeleteForegroundSession(sessionID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.serveConfig.Valid() || !b.serveConfig.Foreground().Has(sessionID) {
		return nil
	}
	sc := b.serveConfig.AsStruct()
	delete(sc.Foreground, sessionID)
	return b.setServeConfigLocked(sc, "")
}

func (b *LocalBackend) HandleIngressTCPConn(ingressPeer tailcfg.NodeView, target ipn.HostPort, srcAddr netip.AddrPort, getConnOrReset func() (net.Conn, bool), sendRST func()) {
	b.mu.Lock()
	sc := b.serveConfig
	b.mu.Unlock()

	if !sc.Valid() {
		b.logf("localbackend: got ingress conn w/o serveConfig; rejecting")
		sendRST()
		return
	}

	if !sc.HasFunnelForTarget(target) {
		b.logf("localbackend: got ingress conn for unconfigured %q; rejecting", target)
		sendRST()
		return
	}

	_, port, err := net.SplitHostPort(string(target))
	if err != nil {
		b.logf("localbackend: got ingress conn for bad target %q; rejecting", target)
		sendRST()
		return
	}
	port16, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		b.logf("localbackend: got ingress conn for bad target %q; rejecting", target)
		sendRST()
		return
	}
	dport := uint16(port16)
	if b.getTCPHandlerForFunnelFlow != nil {
		handler := b.getTCPHandlerForFunnelFlow(srcAddr, dport)
		if handler != nil {
			c, ok := getConnOrReset()
			if !ok {
				b.logf("localbackend: getConn didn't complete from %v to port %v", srcAddr, dport)
				return
			}
			handler(c)
			return
		}
	}
	// TODO(bradfitz): pass ingressPeer etc in context to tcpHandlerForServe,
	// extend serveHTTPContext or similar.
	handler := b.tcpHandlerForServe(dport, srcAddr)
	if handler == nil {
		sendRST()
		return
	}
	c, ok := getConnOrReset()
	if !ok {
		b.logf("localbackend: getConn didn't complete from %v to port %v", srcAddr, dport)
		return
	}
	handler(c)
}

// tcpHandlerForServe returns a handler for a TCP connection to be served via
// the ipn.ServeConfig.
func (b *LocalBackend) tcpHandlerForServe(dport uint16, srcAddr netip.AddrPort) (handler func(net.Conn) error) {
	b.mu.Lock()
	sc := b.serveConfig
	b.mu.Unlock()

	if !sc.Valid() {
		b.logf("[unexpected] localbackend: got TCP conn w/o serveConfig; from %v to port %v", srcAddr, dport)
		return nil
	}

	tcph, ok := sc.FindTCP(dport)
	if !ok {
		b.logf("[unexpected] localbackend: got TCP conn without TCP config for port %v; from %v", dport, srcAddr)
		return nil
	}

	if tcph.HTTPS() || tcph.HTTP() {
		hs := &http.Server{
			Handler: http.HandlerFunc(b.serveWebHandler),
			BaseContext: func(_ net.Listener) context.Context {
				return context.WithValue(context.Background(), serveHTTPContextKey{}, &serveHTTPContext{
					SrcAddr:  srcAddr,
					DestPort: dport,
				})
			},
		}
		if tcph.HTTPS() {
			hs.TLSConfig = &tls.Config{
				GetCertificate: b.getTLSServeCertForPort(dport),
			}
			return func(c net.Conn) error {
				return hs.ServeTLS(netutil.NewOneConnListener(c, nil), "", "")
			}
		}

		return func(c net.Conn) error {
			return hs.Serve(netutil.NewOneConnListener(c, nil))
		}
	}

	if backDst := tcph.TCPForward(); backDst != "" {
		return func(conn net.Conn) error {
			defer conn.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			backConn, err := b.dialer.SystemDial(ctx, "tcp", backDst)
			cancel()
			if err != nil {
				b.logf("localbackend: failed to TCP proxy port %v (from %v) to %s: %v", dport, srcAddr, backDst, err)
				return nil
			}
			defer backConn.Close()
			if sni := tcph.TerminateTLS(); sni != "" {
				conn = tls.Server(conn, &tls.Config{
					GetCertificate: func(hi *tls.ClientHelloInfo) (*tls.Certificate, error) {
						ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
						defer cancel()
						pair, err := b.GetCertPEM(ctx, sni, false)
						if err != nil {
							return nil, err
						}
						cert, err := tls.X509KeyPair(pair.CertPEM, pair.KeyPEM)
						if err != nil {
							return nil, err
						}
						return &cert, nil
					},
				})
			}

			// TODO(bradfitz): do the RegisterIPPortIdentity and
			// UnregisterIPPortIdentity stuff that netstack does
			errc := make(chan error, 1)
			go func() {
				_, err := io.Copy(backConn, conn)
				errc <- err
			}()
			go func() {
				_, err := io.Copy(conn, backConn)
				errc <- err
			}()
			return <-errc
		}
	}

	b.logf("closing TCP conn to port %v (from %v) with actionless TCPPortHandler", dport, srcAddr)
	return nil
}

func getServeHTTPContext(r *http.Request) (c *serveHTTPContext, ok bool) {
	c, ok = r.Context().Value(serveHTTPContextKey{}).(*serveHTTPContext)
	return c, ok
}

func (b *LocalBackend) getServeHandler(r *http.Request) (_ ipn.HTTPHandlerView, at string, ok bool) {
	var z ipn.HTTPHandlerView // zero value

	hostname := r.Host
	if r.TLS == nil {
		tcd := "." + b.Status().CurrentTailnet.MagicDNSSuffix
		if host, _, err := net.SplitHostPort(hostname); err == nil {
			hostname = host
		}
		if !strings.HasSuffix(hostname, tcd) {
			hostname += tcd
		}
	} else {
		hostname = r.TLS.ServerName
	}

	sctx, ok := getServeHTTPContext(r)
	if !ok {
		b.logf("[unexpected] localbackend: no serveHTTPContext in request")
		return z, "", false
	}
	wsc, ok := b.webServerConfig(hostname, sctx.DestPort)
	if !ok {
		return z, "", false
	}

	if h, ok := wsc.Handlers().GetOk(r.URL.Path); ok {
		return h, r.URL.Path, true
	}
	pth := path.Clean(r.URL.Path)
	for {
		withSlash := pth + "/"
		if h, ok := wsc.Handlers().GetOk(withSlash); ok {
			return h, withSlash, true
		}
		if h, ok := wsc.Handlers().GetOk(pth); ok {
			return h, pth, true
		}
		if pth == "/" {
			return z, "", false
		}
		pth = path.Dir(pth)
	}
}

// proxyHandlerForBackend creates a new HTTP reverse proxy for a particular backend that
// we serve requests for. `backend` is a HTTPHandler.Proxy string (url, hostport or just port).
func (b *LocalBackend) proxyHandlerForBackend(backend string) (*httputil.ReverseProxy, error) {
	targetURL, insecure := expandProxyArg(backend)
	u, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid url %s: %w", targetURL, err)
	}
	rp := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(u)
			r.Out.Host = r.In.Host
			addProxyForwardedHeaders(r)
			b.addTailscaleIdentityHeaders(r)
		},
		Transport: &http.Transport{
			DialContext: b.dialer.SystemDial,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecure,
			},
			// Values for the following parameters have been copied from http.DefaultTransport.
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	return rp, nil
}

func addProxyForwardedHeaders(r *httputil.ProxyRequest) {
	r.Out.Header.Set("X-Forwarded-Host", r.In.Host)
	if r.In.TLS != nil {
		r.Out.Header.Set("X-Forwarded-Proto", "https")
	}
	if c, ok := getServeHTTPContext(r.Out); ok {
		r.Out.Header.Set("X-Forwarded-For", c.SrcAddr.Addr().String())
	}
}

func (b *LocalBackend) addTailscaleIdentityHeaders(r *httputil.ProxyRequest) {
	// Clear any incoming values squatting in the headers.
	r.Out.Header.Del("Tailscale-User-Login")
	r.Out.Header.Del("Tailscale-User-Name")
	r.Out.Header.Del("Tailscale-User-Profile-Pic")
	r.Out.Header.Del("Tailscale-Headers-Info")

	c, ok := getServeHTTPContext(r.Out)
	if !ok {
		return
	}
	node, user, ok := b.WhoIs(c.SrcAddr)
	if !ok {
		return // traffic from outside of Tailnet (funneled)
	}
	if node.IsTagged() {
		// 2023-06-14: Not setting identity headers for tagged nodes.
		// Only currently set for nodes with user identities.
		return
	}
	r.Out.Header.Set("Tailscale-User-Login", user.LoginName)
	r.Out.Header.Set("Tailscale-User-Name", user.DisplayName)
	r.Out.Header.Set("Tailscale-User-Profile-Pic", user.ProfilePicURL)
	r.Out.Header.Set("Tailscale-Headers-Info", "https://tailscale.com/s/serve-headers")
}

// serveWebHandler is an http.HandlerFunc that maps incoming requests to the
// correct *http.
func (b *LocalBackend) serveWebHandler(w http.ResponseWriter, r *http.Request) {
	h, mountPoint, ok := b.getServeHandler(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if s := h.Text(); s != "" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(w, s)
		return
	}
	if v := h.Path(); v != "" {
		b.serveFileOrDirectory(w, r, v, mountPoint)
		return
	}
	if v := h.Proxy(); v != "" {
		p, ok := b.serveProxyHandlers.Load(v)
		if !ok {
			http.Error(w, "unknown proxy destination", http.StatusInternalServerError)
			return
		}
		h := p.(http.Handler)
		// Trim the mount point from the URL path before proxying. (#6571)
		if r.URL.Path != "/" {
			h = http.StripPrefix(strings.TrimSuffix(mountPoint, "/"), h)
		}
		h.ServeHTTP(w, r)
		return
	}

	http.Error(w, "empty handler", 500)
}

func (b *LocalBackend) serveFileOrDirectory(w http.ResponseWriter, r *http.Request, fileOrDir, mountPoint string) {
	fi, err := os.Stat(fileOrDir)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), 500)
		return
	}
	if fi.Mode().IsRegular() {
		if mountPoint != r.URL.Path {
			http.NotFound(w, r)
			return
		}
		f, err := os.Open(fileOrDir)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer f.Close()
		http.ServeContent(w, r, path.Base(mountPoint), fi.ModTime(), f)
		return
	}
	if !fi.IsDir() {
		http.Error(w, "not a file or directory", 500)
		return
	}
	if len(r.URL.Path) < len(mountPoint) && r.URL.Path+"/" == mountPoint {
		http.Redirect(w, r, mountPoint, http.StatusFound)
		return
	}

	var fs http.Handler = http.FileServer(http.Dir(fileOrDir))
	if mountPoint != "/" {
		fs = http.StripPrefix(strings.TrimSuffix(mountPoint, "/"), fs)
	}
	fs.ServeHTTP(&fixLocationHeaderResponseWriter{
		ResponseWriter: w,
		mountPoint:     mountPoint,
	}, r)
}

// fixLocationHeaderResponseWriter is an http.ResponseWriter wrapper that, upon
// flushing HTTP headers, prefixes any Location header with the mount point.
type fixLocationHeaderResponseWriter struct {
	http.ResponseWriter
	mountPoint string
	fixOnce    sync.Once // guards call to fix
}

func (w *fixLocationHeaderResponseWriter) fix() {
	h := w.ResponseWriter.Header()
	if v := h.Get("Location"); v != "" {
		h.Set("Location", w.mountPoint+v)
	}
}

func (w *fixLocationHeaderResponseWriter) WriteHeader(code int) {
	w.fixOnce.Do(w.fix)
	w.ResponseWriter.WriteHeader(code)
}

func (w *fixLocationHeaderResponseWriter) Write(p []byte) (int, error) {
	w.fixOnce.Do(w.fix)
	return w.ResponseWriter.Write(p)
}

// expandProxyArg returns a URL from s, where s can be of form:
//
// * port number ("8080")
// * host:port ("localhost:8080")
// * full URL ("http://localhost:8080", in which case it's returned unchanged)
// * insecure TLS ("https+insecure://127.0.0.1:4430")
func expandProxyArg(s string) (targetURL string, insecureSkipVerify bool) {
	if s == "" {
		return "", false
	}
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return s, false
	}
	if rest, ok := strings.CutPrefix(s, "https+insecure://"); ok {
		return "https://" + rest, true
	}
	if allNumeric(s) {
		return "http://127.0.0.1:" + s, false
	}
	return "http://" + s, false
}

func allNumeric(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return s != ""
}

func (b *LocalBackend) webServerConfig(hostname string, port uint16) (c ipn.WebServerConfigView, ok bool) {
	key := ipn.HostPort(fmt.Sprintf("%s:%v", hostname, port))

	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.serveConfig.Valid() {
		return c, false
	}
	return b.serveConfig.FindWeb(key)
}

func (b *LocalBackend) getTLSServeCertForPort(port uint16) func(hi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(hi *tls.ClientHelloInfo) (*tls.Certificate, error) {
		if hi == nil || hi.ServerName == "" {
			return nil, errors.New("no SNI ServerName")
		}
		_, ok := b.webServerConfig(hi.ServerName, port)
		if !ok {
			return nil, errors.New("no webserver configured for name/port")
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		pair, err := b.GetCertPEM(ctx, hi.ServerName, false)
		if err != nil {
			return nil, err
		}
		cert, err := tls.X509KeyPair(pair.CertPEM, pair.KeyPEM)
		if err != nil {
			return nil, err
		}
		return &cert, nil
	}
}
