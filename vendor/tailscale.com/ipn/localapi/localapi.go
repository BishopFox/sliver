// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

// Package localapi contains the HTTP server handlers for tailscaled's API server.
package localapi

import (
	"bytes"
	"context"
	"crypto/sha256"
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
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"tailscale.com/client/tailscale/apitype"
	"tailscale.com/envknob"
	"tailscale.com/health"
	"tailscale.com/hostinfo"
	"tailscale.com/ipn"
	"tailscale.com/ipn/ipnlocal"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/logtail"
	"tailscale.com/net/netmon"
	"tailscale.com/net/netutil"
	"tailscale.com/net/portmapper"
	"tailscale.com/net/tstun"
	"tailscale.com/tailcfg"
	"tailscale.com/tka"
	"tailscale.com/tstime"
	"tailscale.com/types/key"
	"tailscale.com/types/logger"
	"tailscale.com/types/logid"
	"tailscale.com/types/ptr"
	"tailscale.com/types/tkatype"
	"tailscale.com/util/clientmetric"
	"tailscale.com/util/httpm"
	"tailscale.com/util/mak"
	"tailscale.com/util/osdiag"
	"tailscale.com/util/rands"
	"tailscale.com/version"
)

type localAPIHandler func(*Handler, http.ResponseWriter, *http.Request)

// handler is the set of LocalAPI handlers, keyed by the part of the
// Request.URL.Path after "/localapi/v0/". If the key ends with a trailing slash
// then it's a prefix match.
var handler = map[string]localAPIHandler{
	// The prefix match handlers end with a slash:
	"cert/":     (*Handler).serveCert,
	"file-put/": (*Handler).serveFilePut,
	"files/":    (*Handler).serveFiles,
	"profiles/": (*Handler).serveProfiles,

	// The other /localapi/v0/NAME handlers are exact matches and contain only NAME
	// without a trailing slash:
	"bugreport":                   (*Handler).serveBugReport,
	"check-ip-forwarding":         (*Handler).serveCheckIPForwarding,
	"check-prefs":                 (*Handler).serveCheckPrefs,
	"component-debug-logging":     (*Handler).serveComponentDebugLogging,
	"debug":                       (*Handler).serveDebug,
	"debug-derp-region":           (*Handler).serveDebugDERPRegion,
	"debug-packet-filter-matches": (*Handler).serveDebugPacketFilterMatches,
	"debug-packet-filter-rules":   (*Handler).serveDebugPacketFilterRules,
	"debug-portmap":               (*Handler).serveDebugPortmap,
	"debug-peer-endpoint-changes": (*Handler).serveDebugPeerEndpointChanges,
	"debug-capture":               (*Handler).serveDebugCapture,
	"debug-log":                   (*Handler).serveDebugLog,
	"derpmap":                     (*Handler).serveDERPMap,
	"dev-set-state-store":         (*Handler).serveDevSetStateStore,
	"set-push-device-token":       (*Handler).serveSetPushDeviceToken,
	"dial":                        (*Handler).serveDial,
	"file-targets":                (*Handler).serveFileTargets,
	"goroutines":                  (*Handler).serveGoroutines,
	"id-token":                    (*Handler).serveIDToken,
	"login-interactive":           (*Handler).serveLoginInteractive,
	"logout":                      (*Handler).serveLogout,
	"logtap":                      (*Handler).serveLogTap,
	"metrics":                     (*Handler).serveMetrics,
	"ping":                        (*Handler).servePing,
	"prefs":                       (*Handler).servePrefs,
	"pprof":                       (*Handler).servePprof,
	"reset-auth":                  (*Handler).serveResetAuth,
	"serve-config":                (*Handler).serveServeConfig,
	"set-dns":                     (*Handler).serveSetDNS,
	"set-expiry-sooner":           (*Handler).serveSetExpirySooner,
	"start":                       (*Handler).serveStart,
	"status":                      (*Handler).serveStatus,
	"tka/init":                    (*Handler).serveTKAInit,
	"tka/log":                     (*Handler).serveTKALog,
	"tka/modify":                  (*Handler).serveTKAModify,
	"tka/sign":                    (*Handler).serveTKASign,
	"tka/status":                  (*Handler).serveTKAStatus,
	"tka/disable":                 (*Handler).serveTKADisable,
	"tka/force-local-disable":     (*Handler).serveTKALocalDisable,
	"tka/affected-sigs":           (*Handler).serveTKAAffectedSigs,
	"tka/wrap-preauth-key":        (*Handler).serveTKAWrapPreauthKey,
	"tka/verify-deeplink":         (*Handler).serveTKAVerifySigningDeeplink,
	"tka/generate-recovery-aum":   (*Handler).serveTKAGenerateRecoveryAUM,
	"tka/cosign-recovery-aum":     (*Handler).serveTKACosignRecoveryAUM,
	"tka/submit-recovery-aum":     (*Handler).serveTKASubmitRecoveryAUM,
	"upload-client-metrics":       (*Handler).serveUploadClientMetrics,
	"watch-ipn-bus":               (*Handler).serveWatchIPNBus,
	"whois":                       (*Handler).serveWhoIs,
	"query-feature":               (*Handler).serveQueryFeature,
}

var (
	// The clientmetrics package is stateful, but we want to expose a simple
	// imperative API to local clients, so we need to keep track of
	// clientmetric.Metric instances that we've created for them. These need to
	// be globals because we end up creating many Handler instances for the
	// lifetime of a client.
	metricsMu sync.Mutex
	metrics   = map[string]*clientmetric.Metric{}
)

// NewHandler creates a new LocalAPI HTTP handler. All parameters except netMon
// are required (if non-nil it's used to do faster interface lookups).
func NewHandler(b *ipnlocal.LocalBackend, logf logger.Logf, netMon *netmon.Monitor, logID logid.PublicID) *Handler {
	return &Handler{b: b, logf: logf, netMon: netMon, backendLogID: logID, clock: tstime.StdClock{}}
}

type Handler struct {
	// RequiredPassword, if non-empty, forces all HTTP
	// requests to have HTTP basic auth with this password.
	// It's used by the sandboxed macOS sameuserproof GUI auth mechanism.
	RequiredPassword string

	// PermitRead is whether read-only HTTP handlers are allowed.
	PermitRead bool

	// PermitWrite is whether mutating HTTP handlers are allowed.
	// If PermitWrite is true, everything is allowed.
	// It effectively means that the user is root or the admin
	// (operator user).
	PermitWrite bool

	// PermitCert is whether the client is additionally granted
	// cert fetching access.
	PermitCert bool

	b            *ipnlocal.LocalBackend
	logf         logger.Logf
	netMon       *netmon.Monitor // optional; nil means interfaces will be looked up on-demand
	backendLogID logid.PublicID
	clock        tstime.Clock
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.b == nil {
		http.Error(w, "server has no local backend", http.StatusInternalServerError)
		return
	}
	if r.Referer() != "" || r.Header.Get("Origin") != "" || !h.validHost(r.Host) {
		metricInvalidRequests.Add(1)
		http.Error(w, "invalid localapi request", http.StatusForbidden)
		return
	}
	w.Header().Set("Tailscale-Version", version.Long())
	w.Header().Set("Tailscale-Cap", strconv.Itoa(int(tailcfg.CurrentCapabilityVersion)))
	w.Header().Set("Content-Security-Policy", `default-src 'none'; frame-ancestors 'none'; script-src 'none'; script-src-elem 'none'; script-src-attr 'none'`)
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if h.RequiredPassword != "" {
		_, pass, ok := r.BasicAuth()
		if !ok {
			metricInvalidRequests.Add(1)
			http.Error(w, "auth required", http.StatusUnauthorized)
			return
		}
		if pass != h.RequiredPassword {
			metricInvalidRequests.Add(1)
			http.Error(w, "bad password", http.StatusForbidden)
			return
		}
	}
	if fn, ok := handlerForPath(r.URL.Path); ok {
		fn(h, w, r)
	} else {
		http.NotFound(w, r)
	}
}

// validLocalHostForTesting allows loopback handlers without RequiredPassword for testing.
var validLocalHostForTesting = false

// validHost reports whether h is a valid Host header value for a LocalAPI request.
func (h *Handler) validHost(hostname string) bool {
	// The client code sends a hostname of "local-tailscaled.sock".
	switch hostname {
	case "", apitype.LocalAPIHost:
		return true
	}
	if !validLocalHostForTesting && h.RequiredPassword == "" {
		return false // only allow localhost with basic auth or in tests
	}
	host, _, err := net.SplitHostPort(hostname)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}
	return addr.IsLoopback()
}

// handlerForPath returns the LocalAPI handler for the provided Request.URI.Path.
// (the path doesn't include any query parameters)
func handlerForPath(urlPath string) (h localAPIHandler, ok bool) {
	if urlPath == "/" {
		return (*Handler).serveLocalAPIRoot, true
	}
	suff, ok := strings.CutPrefix(urlPath, "/localapi/v0/")
	if !ok {
		// Currently all LocalAPI methods start with "/localapi/v0/" to signal
		// to people that they're not necessarily stable APIs. In practice we'll
		// probably need to keep them pretty stable anyway, but for now treat
		// them as an internal implementation detail.
		return nil, false
	}
	if fn, ok := handler[suff]; ok {
		// Here we match exact handler suffixes like "status" or ones with a
		// slash already in their name, like "tka/status".
		return fn, true
	}
	// Otherwise, it might be a prefix match like "files/*" which we look up
	// by the prefix including first trailing slash.
	if i := strings.IndexByte(suff, '/'); i != -1 {
		suff = suff[:i+1]
		if fn, ok := handler[suff]; ok {
			return fn, true
		}
	}
	return nil, false
}

func (*Handler) serveLocalAPIRoot(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "tailscaled\n")
}

// serveIDToken handles requests to get an OIDC ID token.
func (h *Handler) serveIDToken(w http.ResponseWriter, r *http.Request) {
	if !h.PermitWrite {
		http.Error(w, "id-token access denied", http.StatusForbidden)
		return
	}
	nm := h.b.NetMap()
	if nm == nil {
		http.Error(w, "no netmap", http.StatusServiceUnavailable)
		return
	}
	aud := strings.TrimSpace(r.FormValue("aud"))
	if len(aud) == 0 {
		http.Error(w, "no audience requested", http.StatusBadRequest)
		return
	}
	req := &tailcfg.TokenRequest{
		CapVersion: tailcfg.CurrentCapabilityVersion,
		Audience:   aud,
		NodeKey:    nm.NodeKey,
	}
	b, err := json.Marshal(req)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	httpReq, err := http.NewRequest("POST", "https://unused/machine/id-token", bytes.NewReader(b))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	resp, err := h.b.DoNoiseRequest(httpReq)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func (h *Handler) serveBugReport(w http.ResponseWriter, r *http.Request) {
	if !h.PermitRead {
		http.Error(w, "bugreport access denied", http.StatusForbidden)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}
	defer h.b.TryFlushLogs() // kick off upload after bugreport's done logging

	logMarker := func() string {
		return fmt.Sprintf("BUG-%v-%v-%v", h.backendLogID, h.clock.Now().UTC().Format("20060102150405Z"), rands.HexString(16))
	}
	if envknob.NoLogsNoSupport() {
		logMarker = func() string { return "BUG-NO-LOGS-NO-SUPPORT-this-node-has-had-its-logging-disabled" }
	}

	startMarker := logMarker()
	h.logf("user bugreport: %s", startMarker)
	if note := r.URL.Query().Get("note"); len(note) > 0 {
		h.logf("user bugreport note: %s", note)
	}
	hi, _ := json.Marshal(hostinfo.New())
	h.logf("user bugreport hostinfo: %s", hi)
	if err := health.OverallError(); err != nil {
		h.logf("user bugreport health: %s", err.Error())
	} else {
		h.logf("user bugreport health: ok")
	}

	// Information about the current node from the netmap
	if nm := h.b.NetMap(); nm != nil {
		if self := nm.SelfNode; self.Valid() {
			h.logf("user bugreport node info: nodeid=%q stableid=%q expiry=%q", self.ID(), self.StableID(), self.KeyExpiry().Format(time.RFC3339))
		}
		h.logf("user bugreport public keys: machine=%q node=%q", nm.MachineKey, nm.NodeKey)
	} else {
		h.logf("user bugreport netmap: no active netmap")
	}

	// Print all envknobs; we otherwise only print these on startup, and
	// printing them here ensures we don't have to go spelunking through
	// logs for them.
	envknob.LogCurrent(logger.WithPrefix(h.logf, "user bugreport: "))

	// OS-specific details
	osdiag.LogSupportInfo(logger.WithPrefix(h.logf, "user bugreport OS: "), osdiag.LogSupportInfoReasonBugReport)

	if defBool(r.URL.Query().Get("diagnose"), false) {
		h.b.Doctor(r.Context(), logger.WithPrefix(h.logf, "diag: "))
	}
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintln(w, startMarker)

	// Nothing else to do if we're not in record mode; we wrote the marker
	// above, so we can just finish our response now.
	if !defBool(r.URL.Query().Get("record"), false) {
		return
	}

	until := h.clock.Now().Add(12 * time.Hour)

	var changed map[string]bool
	for _, component := range []string{"magicsock"} {
		if h.b.GetComponentDebugLogging(component).IsZero() {
			if err := h.b.SetComponentDebugLogging(component, until); err != nil {
				h.logf("bugreport: error setting component %q logging: %v", component, err)
				continue
			}

			mak.Set(&changed, component, true)
		}
	}
	defer func() {
		for component := range changed {
			h.b.SetComponentDebugLogging(component, time.Time{})
		}
	}()

	// NOTE(andrew): if we have anything else we want to do while recording
	// a bugreport, we can add it here.

	// Read from the client; this will also return when the client closes
	// the connection.
	var buf [1]byte
	_, err := r.Body.Read(buf[:])

	switch {
	case err == nil:
		// good
	case errors.Is(err, io.EOF):
		// good
	case errors.Is(err, io.ErrUnexpectedEOF):
		// this happens when Ctrl-C'ing the tailscale client; don't
		// bother logging an error
	default:
		// Log but continue anyway.
		h.logf("user bugreport: error reading body: %v", err)
	}

	// Generate another log marker and return it to the client.
	endMarker := logMarker()
	h.logf("user bugreport end: %s", endMarker)
	fmt.Fprintln(w, endMarker)
}

func (h *Handler) serveWhoIs(w http.ResponseWriter, r *http.Request) {
	if !h.PermitRead {
		http.Error(w, "whois access denied", http.StatusForbidden)
		return
	}
	b := h.b
	var ipp netip.AddrPort
	if v := r.FormValue("addr"); v != "" {
		var err error
		ipp, err = netip.ParseAddrPort(v)
		if err != nil {
			http.Error(w, "invalid 'addr' parameter", 400)
			return
		}
	} else {
		http.Error(w, "missing 'addr' parameter", 400)
		return
	}
	n, u, ok := b.WhoIs(ipp)
	if !ok {
		http.Error(w, "no match for IP:port", 404)
		return
	}
	res := &apitype.WhoIsResponse{
		Node:        n.AsStruct(), // always non-nil per WhoIsResponse contract
		UserProfile: &u,           // always non-nil per WhoIsResponse contract
		CapMap:      b.PeerCaps(ipp.Addr()),
	}
	j, err := json.MarshalIndent(res, "", "\t")
	if err != nil {
		http.Error(w, "JSON encoding error", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(j)
}

func (h *Handler) serveGoroutines(w http.ResponseWriter, r *http.Request) {
	// Require write access out of paranoia that the goroutine dump
	// (at least its arguments) might contain something sensitive.
	if !h.PermitWrite {
		http.Error(w, "goroutine dump access denied", http.StatusForbidden)
		return
	}
	buf := make([]byte, 2<<20)
	buf = buf[:runtime.Stack(buf, true)]
	w.Header().Set("Content-Type", "text/plain")
	w.Write(buf)
}

// serveLogTap taps into the tailscaled/logtail server output and streams
// it to the client.
func (h *Handler) serveLogTap(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Require write access (~root) as the logs could contain something
	// sensitive.
	if !h.PermitWrite {
		http.Error(w, "logtap access denied", http.StatusForbidden)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "GET required", http.StatusMethodNotAllowed)
		return
	}
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	io.WriteString(w, `{"text":"[logtap connected]\n"}`+"\n")
	f.Flush()

	msgc := make(chan string, 16)
	unreg := logtail.RegisterLogTap(msgc)
	defer unreg()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-msgc:
			io.WriteString(w, msg)
			f.Flush()
		}
	}
}

func (h *Handler) serveMetrics(w http.ResponseWriter, r *http.Request) {
	// Require write access out of paranoia that the metrics
	// might contain something sensitive.
	if !h.PermitWrite {
		http.Error(w, "metric access denied", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	clientmetric.WritePrometheusExpositionFormat(w)
}

func (h *Handler) serveDebug(w http.ResponseWriter, r *http.Request) {
	if !h.PermitWrite {
		http.Error(w, "debug access denied", http.StatusForbidden)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	// The action is normally in a POST form parameter, but
	// some actions (like "notify") want a full JSON body, so
	// permit some to have their action in a header.
	var action string
	switch v := r.Header.Get("Debug-Action"); v {
	case "notify":
		action = v
	default:
		action = r.FormValue("action")
	}
	var err error
	switch action {
	case "rebind":
		err = h.b.DebugRebind()
	case "restun":
		err = h.b.DebugReSTUN()
	case "enginestatus":
		// serveRequestEngineStatus kicks off a call to RequestEngineStatus (via
		// LocalBackend => UserspaceEngine => LocalBackend =>
		// ipn.Notify{Engine}).
		//
		// This is a temporary (2022-11-25) measure for the Windows client's
		// move to the LocalAPI HTTP interface. It was polling this over the IPN
		// bus before every 2 seconds which is wasteful. We should add a bit to
		// WatchIPNMask instead to let an IPN bus watcher say that it's
		// interested in that info and then only send it on demand, not via
		// polling. But for now we keep this interface because that's what the
		// client already did. A future change will remove this, so don't depend
		// on it.
		h.b.RequestEngineStatus()
	case "notify":
		var n ipn.Notify
		err = json.NewDecoder(r.Body).Decode(&n)
		if err != nil {
			break
		}
		h.b.DebugNotify(n)
	case "break-tcp-conns":
		err = h.b.DebugBreakTCPConns()
	case "break-derp-conns":
		err = h.b.DebugBreakDERPConns()
	case "force-netmap-update":
		h.b.DebugForceNetmapUpdate()
	case "control-knobs":
		k := h.b.ControlKnobs()
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(k.AsDebugJSON())
		if err == nil {
			return
		}
	case "":
		err = fmt.Errorf("missing parameter 'action'")
	default:
		err = fmt.Errorf("unknown action %q", action)
	}
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	io.WriteString(w, "done\n")
}

func (h *Handler) serveDevSetStateStore(w http.ResponseWriter, r *http.Request) {
	if !h.PermitWrite {
		http.Error(w, "debug access denied", http.StatusForbidden)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	if err := h.b.SetDevStateStore(r.FormValue("key"), r.FormValue("value")); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	io.WriteString(w, "done\n")
}

func (h *Handler) serveDebugPacketFilterRules(w http.ResponseWriter, r *http.Request) {
	if !h.PermitWrite {
		http.Error(w, "debug access denied", http.StatusForbidden)
		return
	}
	nm := h.b.NetMap()
	if nm == nil {
		http.Error(w, "no netmap", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")
	enc.Encode(nm.PacketFilterRules)
}

func (h *Handler) serveDebugPacketFilterMatches(w http.ResponseWriter, r *http.Request) {
	if !h.PermitWrite {
		http.Error(w, "debug access denied", http.StatusForbidden)
		return
	}
	nm := h.b.NetMap()
	if nm == nil {
		http.Error(w, "no netmap", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")
	enc.Encode(nm.PacketFilter)
}

func (h *Handler) serveDebugPortmap(w http.ResponseWriter, r *http.Request) {
	if !h.PermitWrite {
		http.Error(w, "debug access denied", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "text/plain")

	dur, err := time.ParseDuration(r.FormValue("duration"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	gwSelf := r.FormValue("gateway_and_self")

	// Update portmapper debug flags
	debugKnobs := &portmapper.DebugKnobs{VerboseLogs: true}
	switch r.FormValue("type") {
	case "":
	case "pmp":
		debugKnobs.DisablePCP = true
		debugKnobs.DisableUPnP = true
	case "pcp":
		debugKnobs.DisablePMP = true
		debugKnobs.DisableUPnP = true
	case "upnp":
		debugKnobs.DisablePCP = true
		debugKnobs.DisablePMP = true
	default:
		http.Error(w, "unknown portmap debug type", http.StatusBadRequest)
		return
	}

	if defBool(r.FormValue("log_http"), false) {
		debugKnobs.LogHTTP = true
	}

	var (
		logLock     sync.Mutex
		handlerDone bool
	)
	logf := func(format string, args ...any) {
		if !strings.HasSuffix(format, "\n") {
			format = format + "\n"
		}

		logLock.Lock()
		defer logLock.Unlock()

		// The portmapper can call this log function after the HTTP
		// handler returns, which is not allowed and can cause a panic.
		// If this happens, ignore the log lines since this typically
		// occurs due to a client disconnect.
		if handlerDone {
			return
		}

		// Write and flush each line to the client so that output is streamed
		fmt.Fprintf(w, format, args...)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
	defer func() {
		logLock.Lock()
		handlerDone = true
		logLock.Unlock()
	}()

	ctx, cancel := context.WithTimeout(r.Context(), dur)
	defer cancel()

	done := make(chan bool, 1)

	var c *portmapper.Client
	c = portmapper.NewClient(logger.WithPrefix(logf, "portmapper: "), h.netMon, debugKnobs, h.b.ControlKnobs(), func() {
		logf("portmapping changed.")
		logf("have mapping: %v", c.HaveMapping())

		if ext, ok := c.GetCachedMappingOrStartCreatingOne(); ok {
			logf("cb: mapping: %v", ext)
			select {
			case done <- true:
			default:
			}
			return
		}
		logf("cb: no mapping")
	})
	defer c.Close()

	netMon, err := netmon.New(logger.WithPrefix(logf, "monitor: "))
	if err != nil {
		logf("error creating monitor: %v", err)
		return
	}

	gatewayAndSelfIP := func() (gw, self netip.Addr, ok bool) {
		if a, b, ok := strings.Cut(gwSelf, "/"); ok {
			gw = netip.MustParseAddr(a)
			self = netip.MustParseAddr(b)
			return gw, self, true
		}
		return netMon.GatewayAndSelfIP()
	}

	c.SetGatewayLookupFunc(gatewayAndSelfIP)

	gw, selfIP, ok := gatewayAndSelfIP()
	if !ok {
		logf("no gateway or self IP; %v", netMon.InterfaceState())
		return
	}
	logf("gw=%v; self=%v", gw, selfIP)

	uc, err := net.ListenPacket("udp", "0.0.0.0:0")
	if err != nil {
		return
	}
	defer uc.Close()
	c.SetLocalPort(uint16(uc.LocalAddr().(*net.UDPAddr).Port))

	res, err := c.Probe(ctx)
	if err != nil {
		logf("error in Probe: %v", err)
		return
	}
	logf("Probe: %+v", res)

	if !res.PCP && !res.PMP && !res.UPnP {
		logf("no portmapping services available")
		return
	}

	if ext, ok := c.GetCachedMappingOrStartCreatingOne(); ok {
		logf("mapping: %v", ext)
	} else {
		logf("no mapping")
	}

	select {
	case <-done:
	case <-ctx.Done():
		if r.Context().Err() == nil {
			logf("serveDebugPortmap: context done: %v", ctx.Err())
		} else {
			h.logf("serveDebugPortmap: context done: %v", ctx.Err())
		}
	}
}

func (h *Handler) serveComponentDebugLogging(w http.ResponseWriter, r *http.Request) {
	if !h.PermitWrite {
		http.Error(w, "debug access denied", http.StatusForbidden)
		return
	}
	component := r.FormValue("component")
	secs, _ := strconv.Atoi(r.FormValue("secs"))
	err := h.b.SetComponentDebugLogging(component, h.clock.Now().Add(time.Duration(secs)*time.Second))
	var res struct {
		Error string
	}
	if err != nil {
		res.Error = err.Error()
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// servePprofFunc is the implementation of Handler.servePprof, after auth,
// for platforms where we want to link it in.
var servePprofFunc func(http.ResponseWriter, *http.Request)

func (h *Handler) servePprof(w http.ResponseWriter, r *http.Request) {
	// Require write access out of paranoia that the profile dump
	// might contain something sensitive.
	if !h.PermitWrite {
		http.Error(w, "profile access denied", http.StatusForbidden)
		return
	}
	if servePprofFunc == nil {
		http.Error(w, "not implemented on this platform", http.StatusServiceUnavailable)
		return
	}
	servePprofFunc(w, r)
}

func (h *Handler) serveResetAuth(w http.ResponseWriter, r *http.Request) {
	if !h.PermitWrite {
		http.Error(w, "reset-auth modify access denied", http.StatusForbidden)
		return
	}
	if r.Method != httpm.POST {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}

	if err := h.b.ResetAuth(); err != nil {
		http.Error(w, "reset-auth failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) serveServeConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		if !h.PermitRead {
			http.Error(w, "serve config denied", http.StatusForbidden)
			return
		}
		config := h.b.ServeConfig()
		bts, err := json.Marshal(config)
		if err != nil {
			http.Error(w, "error encoding config: "+err.Error(), http.StatusInternalServerError)
			return
		}
		sum := sha256.Sum256(bts)
		etag := hex.EncodeToString(sum[:])
		w.Header().Set("Etag", etag)
		w.Header().Set("Content-Type", "application/json")
		w.Write(bts)
	case "POST":
		if !h.PermitWrite {
			http.Error(w, "serve config denied", http.StatusForbidden)
			return
		}
		configIn := new(ipn.ServeConfig)
		if err := json.NewDecoder(r.Body).Decode(configIn); err != nil {
			writeErrorJSON(w, fmt.Errorf("decoding config: %w", err))
			return
		}
		etag := r.Header.Get("If-Match")
		if err := h.b.SetServeConfig(configIn, etag); err != nil {
			if errors.Is(err, ipnlocal.ErrETagMismatch) {
				http.Error(w, err.Error(), http.StatusPreconditionFailed)
				return
			}
			writeErrorJSON(w, fmt.Errorf("updating config: %w", err))
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) serveCheckIPForwarding(w http.ResponseWriter, r *http.Request) {
	if !h.PermitRead {
		http.Error(w, "IP forwarding check access denied", http.StatusForbidden)
		return
	}
	var warning string
	if err := h.b.CheckIPForwarding(); err != nil {
		warning = err.Error()
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Warning string
	}{
		Warning: warning,
	})
}

func (h *Handler) serveStatus(w http.ResponseWriter, r *http.Request) {
	if !h.PermitRead {
		http.Error(w, "status access denied", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	var st *ipnstate.Status
	if defBool(r.FormValue("peers"), true) {
		st = h.b.Status()
	} else {
		st = h.b.StatusWithoutPeers()
	}
	e := json.NewEncoder(w)
	e.SetIndent("", "\t")
	e.Encode(st)
}

func (h *Handler) serveDebugPeerEndpointChanges(w http.ResponseWriter, r *http.Request) {
	if !h.PermitRead {
		http.Error(w, "status access denied", http.StatusForbidden)
		return
	}

	ipStr := r.FormValue("ip")
	if ipStr == "" {
		http.Error(w, "missing 'ip' parameter", 400)
		return
	}
	ip, err := netip.ParseAddr(ipStr)
	if err != nil {
		http.Error(w, "invalid IP", 400)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	chs, err := h.b.GetPeerEndpointChanges(r.Context(), ip)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	e := json.NewEncoder(w)
	e.SetIndent("", "\t")
	e.Encode(chs)
}

// InUseOtherUserIPNStream reports whether r is a request for the watch-ipn-bus
// handler. If so, it writes an ipn.Notify InUseOtherUser message to the user
// and returns true. Otherwise it returns false, in which case it doesn't write
// to w.
//
// Unlike the regular watch-ipn-bus handler, this one doesn't block. The caller
// (in ipnserver.Server) provides the blocking until the connection is no longer
// in use.
func InUseOtherUserIPNStream(w http.ResponseWriter, r *http.Request, err error) (handled bool) {
	if r.Method != "GET" || r.URL.Path != "/localapi/v0/watch-ipn-bus" {
		return false
	}
	js, err := json.Marshal(&ipn.Notify{
		Version:    version.Long(),
		State:      ptr.To(ipn.InUseOtherUser),
		ErrMessage: ptr.To(err.Error()),
	})
	if err != nil {
		return false
	}
	js = append(js, '\n')
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
	return true
}

func (h *Handler) serveWatchIPNBus(w http.ResponseWriter, r *http.Request) {
	if !h.PermitRead {
		http.Error(w, "watch ipn bus access denied", http.StatusForbidden)
		return
	}
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "not a flusher", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	var mask ipn.NotifyWatchOpt
	if s := r.FormValue("mask"); s != "" {
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			http.Error(w, "bad mask", http.StatusBadRequest)
			return
		}
		mask = ipn.NotifyWatchOpt(v)
	}
	ctx := r.Context()
	h.b.WatchNotifications(ctx, mask, f.Flush, func(roNotify *ipn.Notify) (keepGoing bool) {
		js, err := json.Marshal(roNotify)
		if err != nil {
			h.logf("json.Marshal: %v", err)
			return false
		}
		if _, err := fmt.Fprintf(w, "%s\n", js); err != nil {
			return false
		}
		f.Flush()
		return true
	})
}

func (h *Handler) serveLoginInteractive(w http.ResponseWriter, r *http.Request) {
	if !h.PermitWrite {
		http.Error(w, "login access denied", http.StatusForbidden)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "want POST", 400)
		return
	}
	h.b.StartLoginInteractive()
	w.WriteHeader(http.StatusNoContent)
	return
}

func (h *Handler) serveStart(w http.ResponseWriter, r *http.Request) {
	if !h.PermitWrite {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "want POST", 400)
		return
	}
	var o ipn.Options
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err := h.b.Start(o)
	if err != nil {
		// TODO(bradfitz): map error to a good HTTP error
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) serveLogout(w http.ResponseWriter, r *http.Request) {
	if !h.PermitWrite {
		http.Error(w, "logout access denied", http.StatusForbidden)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "want POST", 400)
		return
	}
	err := h.b.Logout(r.Context())
	if err == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Error(w, err.Error(), 500)
}

func (h *Handler) servePrefs(w http.ResponseWriter, r *http.Request) {
	if !h.PermitRead {
		http.Error(w, "prefs access denied", http.StatusForbidden)
		return
	}
	var prefs ipn.PrefsView
	switch r.Method {
	case "PATCH":
		if !h.PermitWrite {
			http.Error(w, "prefs write access denied", http.StatusForbidden)
			return
		}
		mp := new(ipn.MaskedPrefs)
		if err := json.NewDecoder(r.Body).Decode(mp); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		var err error
		prefs, err = h.b.EditPrefs(mp)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(resJSON{Error: err.Error()})
			return
		}
	case "GET", "HEAD":
		prefs = h.b.Prefs()
	default:
		http.Error(w, "unsupported method", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	e := json.NewEncoder(w)
	e.SetIndent("", "\t")
	e.Encode(prefs)
}

type resJSON struct {
	Error string `json:",omitempty"`
}

func (h *Handler) serveCheckPrefs(w http.ResponseWriter, r *http.Request) {
	if !h.PermitWrite {
		http.Error(w, "checkprefs access denied", http.StatusForbidden)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "unsupported method", http.StatusMethodNotAllowed)
		return
	}
	p := new(ipn.Prefs)
	if err := json.NewDecoder(r.Body).Decode(p); err != nil {
		http.Error(w, "invalid JSON body", 400)
		return
	}
	err := h.b.CheckPrefs(p)
	var res resJSON
	if err != nil {
		res.Error = err.Error()
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) serveFiles(w http.ResponseWriter, r *http.Request) {
	if !h.PermitWrite {
		http.Error(w, "file access denied", http.StatusForbidden)
		return
	}
	suffix, ok := strings.CutPrefix(r.URL.EscapedPath(), "/localapi/v0/files/")
	if !ok {
		http.Error(w, "misconfigured", http.StatusInternalServerError)
		return
	}
	if suffix == "" {
		if r.Method != "GET" {
			http.Error(w, "want GET to list files", 400)
			return
		}
		ctx := r.Context()
		if s := r.FormValue("waitsec"); s != "" && s != "0" {
			d, err := strconv.Atoi(s)
			if err != nil {
				http.Error(w, "invalid waitsec", http.StatusBadRequest)
				return
			}
			deadline := time.Now().Add(time.Duration(d) * time.Second)
			var cancel context.CancelFunc
			ctx, cancel = context.WithDeadline(ctx, deadline)
			defer cancel()
		}
		wfs, err := h.b.AwaitWaitingFiles(ctx)
		if err != nil && ctx.Err() == nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wfs)
		return
	}
	name, err := url.PathUnescape(suffix)
	if err != nil {
		http.Error(w, "bad filename", 400)
		return
	}
	if r.Method == "DELETE" {
		if err := h.b.DeleteFile(name); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	rc, size, err := h.b.OpenFile(name)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Length", fmt.Sprint(size))
	w.Header().Set("Content-Type", "application/octet-stream")
	io.Copy(w, rc)
}

func writeErrorJSON(w http.ResponseWriter, err error) {
	if err == nil {
		err = errors.New("unexpected nil error")
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(500)
	type E struct {
		Error string `json:"error"`
	}
	json.NewEncoder(w).Encode(E{err.Error()})
}

func (h *Handler) serveFileTargets(w http.ResponseWriter, r *http.Request) {
	if !h.PermitRead {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "want GET to list targets", 400)
		return
	}
	fts, err := h.b.FileTargets()
	if err != nil {
		writeErrorJSON(w, err)
		return
	}
	mak.NonNilSliceForJSON(&fts)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fts)
}

// serveFilePut sends a file to another node.
//
// It's sometimes possible for clients to do this themselves, without
// tailscaled, except in the case of tailscaled running in
// userspace-networking ("netstack") mode, in which case tailscaled
// needs to a do a netstack dial out.
//
// Instead, the CLI also goes through tailscaled so it doesn't need to be
// aware of the network mode in use.
//
// macOS/iOS have always used this localapi method to simplify the GUI
// clients.
//
// The Windows client currently (2021-11-30) uses the peerapi (/v0/put/)
// directly, as the Windows GUI always runs in tun mode anyway.
//
// URL format:
//
//   - PUT /localapi/v0/file-put/:stableID/:escaped-filename
func (h *Handler) serveFilePut(w http.ResponseWriter, r *http.Request) {
	metricFilePutCalls.Add(1)

	if !h.PermitWrite {
		http.Error(w, "file access denied", http.StatusForbidden)
		return
	}
	if r.Method != "PUT" {
		http.Error(w, "want PUT to put file", 400)
		return
	}
	fts, err := h.b.FileTargets()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	upath, ok := strings.CutPrefix(r.URL.EscapedPath(), "/localapi/v0/file-put/")
	if !ok {
		http.Error(w, "misconfigured", http.StatusInternalServerError)
		return
	}
	stableIDStr, filenameEscaped, ok := strings.Cut(upath, "/")
	if !ok {
		http.Error(w, "bogus URL", 400)
		return
	}
	stableID := tailcfg.StableNodeID(stableIDStr)

	var ft *apitype.FileTarget
	for _, x := range fts {
		if x.Node.StableID == stableID {
			ft = x
			break
		}
	}
	if ft == nil {
		http.Error(w, "node not found", 404)
		return
	}
	dstURL, err := url.Parse(ft.PeerAPIURL)
	if err != nil {
		http.Error(w, "bogus peer URL", 500)
		return
	}
	outReq, err := http.NewRequestWithContext(r.Context(), "PUT", "http://peer/v0/put/"+filenameEscaped, r.Body)
	if err != nil {
		http.Error(w, "bogus outreq", 500)
		return
	}
	outReq.ContentLength = r.ContentLength

	rp := httputil.NewSingleHostReverseProxy(dstURL)
	rp.Transport = h.b.Dialer().PeerAPITransport()
	rp.ServeHTTP(w, outReq)
}

func (h *Handler) serveSetDNS(w http.ResponseWriter, r *http.Request) {
	if !h.PermitWrite {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "want POST", 400)
		return
	}
	ctx := r.Context()
	err := h.b.SetDNS(ctx, r.FormValue("name"), r.FormValue("value"))
	if err != nil {
		writeErrorJSON(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct{}{})
}

func (h *Handler) serveDERPMap(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "want GET", 400)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	e := json.NewEncoder(w)
	e.SetIndent("", "\t")
	e.Encode(h.b.DERPMap())
}

// serveSetExpirySooner sets the expiry date on the current machine, specified
// by an `expiry` unix timestamp as POST or query param.
func (h *Handler) serveSetExpirySooner(w http.ResponseWriter, r *http.Request) {
	if !h.PermitWrite {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var expiryTime time.Time
	if v := r.FormValue("expiry"); v != "" {
		expiryInt, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			http.Error(w, "can't parse expiry time, expects a unix timestamp", http.StatusBadRequest)
			return
		}
		expiryTime = time.Unix(expiryInt, 0)
	} else {
		http.Error(w, "missing 'expiry' parameter, a unix timestamp", http.StatusBadRequest)
		return
	}
	err := h.b.SetExpirySooner(r.Context(), expiryTime)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	io.WriteString(w, "done\n")
}

func (h *Handler) servePing(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method != "POST" {
		http.Error(w, "want POST", 400)
		return
	}
	ipStr := r.FormValue("ip")
	if ipStr == "" {
		http.Error(w, "missing 'ip' parameter", 400)
		return
	}
	ip, err := netip.ParseAddr(ipStr)
	if err != nil {
		http.Error(w, "invalid IP", 400)
		return
	}
	pingTypeStr := r.FormValue("type")
	if pingTypeStr == "" {
		http.Error(w, "missing 'type' parameter", 400)
		return
	}
	size := 0
	sizeStr := r.FormValue("size")
	if sizeStr != "" {
		size, err = strconv.Atoi(sizeStr)
		if err != nil {
			http.Error(w, "invalid 'size' parameter", 400)
			return
		}
		if size != 0 && tailcfg.PingType(pingTypeStr) != tailcfg.PingDisco {
			http.Error(w, "'size' parameter is only supported with disco pings", 400)
			return
		}
		if size > int(tstun.DefaultMTU()) {
			http.Error(w, fmt.Sprintf("maximum value for 'size' is %v", tstun.DefaultMTU()), 400)
			return
		}
	}
	res, err := h.b.Ping(ctx, ip, tailcfg.PingType(pingTypeStr), size)
	if err != nil {
		writeErrorJSON(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) serveDial(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	const upgradeProto = "ts-dial"
	if !strings.Contains(r.Header.Get("Connection"), "upgrade") ||
		r.Header.Get("Upgrade") != upgradeProto {
		http.Error(w, "bad ts-dial upgrade", http.StatusBadRequest)
		return
	}
	hostStr, portStr := r.Header.Get("Dial-Host"), r.Header.Get("Dial-Port")
	if hostStr == "" || portStr == "" {
		http.Error(w, "missing Dial-Host or Dial-Port header", http.StatusBadRequest)
		return
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "make request over HTTP/1", http.StatusBadRequest)
		return
	}

	addr := net.JoinHostPort(hostStr, portStr)
	outConn, err := h.b.Dialer().UserDial(r.Context(), "tcp", addr)
	if err != nil {
		http.Error(w, "dial failure: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer outConn.Close()

	w.Header().Set("Upgrade", upgradeProto)
	w.Header().Set("Connection", "upgrade")
	w.WriteHeader(http.StatusSwitchingProtocols)

	reqConn, brw, err := hijacker.Hijack()
	if err != nil {
		h.logf("localapi dial Hijack error: %v", err)
		return
	}
	defer reqConn.Close()
	if err := brw.Flush(); err != nil {
		return
	}
	reqConn = netutil.NewDrainBufConn(reqConn, brw.Reader)

	errc := make(chan error, 1)
	go func() {
		_, err := io.Copy(reqConn, outConn)
		errc <- err
	}()
	go func() {
		_, err := io.Copy(outConn, reqConn)
		errc <- err
	}()
	<-errc
}

func (h *Handler) serveSetPushDeviceToken(w http.ResponseWriter, r *http.Request) {
	if !h.PermitWrite {
		http.Error(w, "set push device token access denied", http.StatusForbidden)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "unsupported method", http.StatusMethodNotAllowed)
		return
	}
	var params apitype.SetPushDeviceTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "invalid JSON body", 400)
		return
	}
	hostinfo.SetPushDeviceToken(params.PushDeviceToken)
	h.b.ResendHostinfoIfNeeded()
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) serveUploadClientMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "unsupported method", http.StatusMethodNotAllowed)
		return
	}
	type clientMetricJSON struct {
		Name  string `json:"name"`
		Type  string `json:"type"`  // one of "counter" or "gauge"
		Value int    `json:"value"` // amount to increment metric by
	}

	var clientMetrics []clientMetricJSON
	if err := json.NewDecoder(r.Body).Decode(&clientMetrics); err != nil {
		http.Error(w, "invalid JSON body", 400)
		return
	}

	metricsMu.Lock()
	defer metricsMu.Unlock()

	for _, m := range clientMetrics {
		if metric, ok := metrics[m.Name]; ok {
			metric.Add(int64(m.Value))
		} else {
			if clientmetric.HasPublished(m.Name) {
				http.Error(w, "Already have a metric named "+m.Name, 400)
				return
			}
			var metric *clientmetric.Metric
			switch m.Type {
			case "counter":
				metric = clientmetric.NewCounter(m.Name)
			case "gauge":
				metric = clientmetric.NewGauge(m.Name)
			default:
				http.Error(w, "Unknown metric type "+m.Type, 400)
				return
			}
			metrics[m.Name] = metric
			metric.Add(int64(m.Value))
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct{}{})
}

func (h *Handler) serveTKAStatus(w http.ResponseWriter, r *http.Request) {
	if !h.PermitRead {
		http.Error(w, "lock status access denied", http.StatusForbidden)
		return
	}
	if r.Method != httpm.GET {
		http.Error(w, "use GET", http.StatusMethodNotAllowed)
		return
	}

	j, err := json.MarshalIndent(h.b.NetworkLockStatus(), "", "\t")
	if err != nil {
		http.Error(w, "JSON encoding error", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(j)
}

func (h *Handler) serveTKASign(w http.ResponseWriter, r *http.Request) {
	if !h.PermitRead {
		http.Error(w, "lock status access denied", http.StatusForbidden)
		return
	}
	if r.Method != httpm.POST {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}

	type signRequest struct {
		NodeKey        key.NodePublic
		RotationPublic []byte
	}
	var req signRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if err := h.b.NetworkLockSign(req.NodeKey, req.RotationPublic); err != nil {
		http.Error(w, "signing failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) serveTKAInit(w http.ResponseWriter, r *http.Request) {
	if !h.PermitWrite {
		http.Error(w, "lock init access denied", http.StatusForbidden)
		return
	}
	if r.Method != httpm.POST {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}

	type initRequest struct {
		Keys               []tka.Key
		DisablementValues  [][]byte
		SupportDisablement []byte
	}
	var req initRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", 400)
		return
	}

	if err := h.b.NetworkLockInit(req.Keys, req.DisablementValues, req.SupportDisablement); err != nil {
		http.Error(w, "initialization failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	j, err := json.MarshalIndent(h.b.NetworkLockStatus(), "", "\t")
	if err != nil {
		http.Error(w, "JSON encoding error", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(j)
}

func (h *Handler) serveTKAModify(w http.ResponseWriter, r *http.Request) {
	if !h.PermitWrite {
		http.Error(w, "network-lock modify access denied", http.StatusForbidden)
		return
	}
	if r.Method != httpm.POST {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}

	type modifyRequest struct {
		AddKeys    []tka.Key
		RemoveKeys []tka.Key
	}
	var req modifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", 400)
		return
	}

	if err := h.b.NetworkLockModify(req.AddKeys, req.RemoveKeys); err != nil {
		http.Error(w, "network-lock modify failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(204)
}

func (h *Handler) serveTKAWrapPreauthKey(w http.ResponseWriter, r *http.Request) {
	if !h.PermitWrite {
		http.Error(w, "network-lock modify access denied", http.StatusForbidden)
		return
	}
	if r.Method != httpm.POST {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}

	type wrapRequest struct {
		TSKey  string
		TKAKey string // key.NLPrivate.MarshalText
	}
	var req wrapRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 12*1024)).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	var priv key.NLPrivate
	if err := priv.UnmarshalText([]byte(req.TKAKey)); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	wrappedKey, err := h.b.NetworkLockWrapPreauthKey(req.TSKey, priv)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(200)
	w.Write([]byte(wrappedKey))
}

func (h *Handler) serveTKAVerifySigningDeeplink(w http.ResponseWriter, r *http.Request) {
	if !h.PermitRead {
		http.Error(w, "signing deeplink verification access denied", http.StatusForbidden)
		return
	}
	if r.Method != httpm.POST {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}

	type verifyRequest struct {
		URL string
	}
	var req verifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON for verifyRequest body", 400)
		return
	}

	res := h.b.NetworkLockVerifySigningDeeplink(req.URL)
	j, err := json.MarshalIndent(res, "", "\t")
	if err != nil {
		http.Error(w, "JSON encoding error", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(j)
}

func (h *Handler) serveTKADisable(w http.ResponseWriter, r *http.Request) {
	if !h.PermitWrite {
		http.Error(w, "network-lock modify access denied", http.StatusForbidden)
		return
	}
	if r.Method != httpm.POST {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}

	body := io.LimitReader(r.Body, 1024*1024)
	secret, err := io.ReadAll(body)
	if err != nil {
		http.Error(w, "reading secret", 400)
		return
	}

	if err := h.b.NetworkLockDisable(secret); err != nil {
		http.Error(w, "network-lock disable failed: "+err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(200)
}

func (h *Handler) serveTKALocalDisable(w http.ResponseWriter, r *http.Request) {
	if !h.PermitWrite {
		http.Error(w, "network-lock modify access denied", http.StatusForbidden)
		return
	}
	if r.Method != httpm.POST {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}

	// Require a JSON stanza for the body as an additional CSRF protection.
	var req struct{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", 400)
		return
	}

	if err := h.b.NetworkLockForceLocalDisable(); err != nil {
		http.Error(w, "network-lock local disable failed: "+err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(200)
}

func (h *Handler) serveTKALog(w http.ResponseWriter, r *http.Request) {
	if r.Method != httpm.GET {
		http.Error(w, "use GET", http.StatusMethodNotAllowed)
		return
	}

	limit := 50
	if limitStr := r.FormValue("limit"); limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, "parsing 'limit' parameter: "+err.Error(), http.StatusBadRequest)
			return
		}
		limit = int(l)
	}

	updates, err := h.b.NetworkLockLog(limit)
	if err != nil {
		http.Error(w, "reading log failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	j, err := json.MarshalIndent(updates, "", "\t")
	if err != nil {
		http.Error(w, "JSON encoding error", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(j)
}

func (h *Handler) serveTKAAffectedSigs(w http.ResponseWriter, r *http.Request) {
	if r.Method != httpm.POST {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}
	keyID, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 2048))
	if err != nil {
		http.Error(w, "reading body", http.StatusBadRequest)
		return
	}

	sigs, err := h.b.NetworkLockAffectedSigs(keyID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j, err := json.MarshalIndent(sigs, "", "\t")
	if err != nil {
		http.Error(w, "JSON encoding error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(j)
}

func (h *Handler) serveTKAGenerateRecoveryAUM(w http.ResponseWriter, r *http.Request) {
	if !h.PermitWrite {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}
	if r.Method != httpm.POST {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}

	type verifyRequest struct {
		Keys     []tkatype.KeyID
		ForkFrom string
	}
	var req verifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON for verifyRequest body", http.StatusBadRequest)
		return
	}

	var forkFrom tka.AUMHash
	if req.ForkFrom != "" {
		if err := forkFrom.UnmarshalText([]byte(req.ForkFrom)); err != nil {
			http.Error(w, "decoding fork-from: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	res, err := h.b.NetworkLockGenerateRecoveryAUM(req.Keys, forkFrom)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(res.Serialize())
}

func (h *Handler) serveTKACosignRecoveryAUM(w http.ResponseWriter, r *http.Request) {
	if !h.PermitWrite {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}
	if r.Method != httpm.POST {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}

	body := io.LimitReader(r.Body, 1024*1024)
	aumBytes, err := io.ReadAll(body)
	if err != nil {
		http.Error(w, "reading AUM", http.StatusBadRequest)
		return
	}
	var aum tka.AUM
	if err := aum.Unserialize(aumBytes); err != nil {
		http.Error(w, "decoding AUM", http.StatusBadRequest)
		return
	}

	res, err := h.b.NetworkLockCosignRecoveryAUM(&aum)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(res.Serialize())
}

func (h *Handler) serveTKASubmitRecoveryAUM(w http.ResponseWriter, r *http.Request) {
	if !h.PermitWrite {
		http.Error(w, "access denied", http.StatusForbidden)
		return
	}
	if r.Method != httpm.POST {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}

	body := io.LimitReader(r.Body, 1024*1024)
	aumBytes, err := io.ReadAll(body)
	if err != nil {
		http.Error(w, "reading AUM", http.StatusBadRequest)
		return
	}
	var aum tka.AUM
	if err := aum.Unserialize(aumBytes); err != nil {
		http.Error(w, "decoding AUM", http.StatusBadRequest)
		return
	}

	if err := h.b.NetworkLockSubmitRecoveryAUM(&aum); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// serveProfiles serves profile switching-related endpoints. Supported methods
// and paths are:
//   - GET /profiles/: list all profiles (JSON-encoded array of ipn.LoginProfiles)
//   - PUT /profiles/: add new profile (no response). A separate
//     StartLoginInteractive() is needed to populate and persist the new profile.
//   - GET /profiles/current: current profile (JSON-ecoded ipn.LoginProfile)
//   - GET /profiles/<id>: output profile (JSON-ecoded ipn.LoginProfile)
//   - POST /profiles/<id>: switch to profile (no response)
//   - DELETE /profiles/<id>: delete profile (no response)
func (h *Handler) serveProfiles(w http.ResponseWriter, r *http.Request) {
	if !h.PermitWrite {
		http.Error(w, "profiles access denied", http.StatusForbidden)
		return
	}
	suffix, ok := strings.CutPrefix(r.URL.EscapedPath(), "/localapi/v0/profiles/")
	if !ok {
		http.Error(w, "misconfigured", http.StatusInternalServerError)
		return
	}
	if suffix == "" {
		switch r.Method {
		case httpm.GET:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(h.b.ListProfiles())
		case httpm.PUT:
			err := h.b.NewProfile()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
		default:
			http.Error(w, "use GET or PUT", http.StatusMethodNotAllowed)
		}
		return
	}
	suffix, err := url.PathUnescape(suffix)
	if err != nil {
		http.Error(w, "bad profile ID", http.StatusBadRequest)
		return
	}
	if suffix == "current" {
		switch r.Method {
		case httpm.GET:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(h.b.CurrentProfile())
		default:
			http.Error(w, "use GET", http.StatusMethodNotAllowed)
		}
		return
	}

	profileID := ipn.ProfileID(suffix)
	switch r.Method {
	case httpm.GET:
		profiles := h.b.ListProfiles()
		profileIndex := slices.IndexFunc(profiles, func(p ipn.LoginProfile) bool {
			return p.ID == profileID
		})
		if profileIndex == -1 {
			http.Error(w, "Profile not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profiles[profileIndex])
	case httpm.POST:
		err := h.b.SwitchProfile(profileID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case httpm.DELETE:
		err := h.b.DeleteProfile(profileID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "use POST or DELETE", http.StatusMethodNotAllowed)
	}
}

// serveQueryFeature makes a request to the "/machine/feature/query"
// Noise endpoint to get instructions on how to enable a feature, such as
// Funnel, for the node's tailnet.
//
// This request itself does not directly enable the feature on behalf of
// the node, but rather returns information that can be presented to the
// acting user about where/how to enable the feature. If relevant, this
// includes a control URL the user can visit to explicitly consent to
// using the feature.
//
// See tailcfg.QueryFeatureResponse for full response structure.
func (h *Handler) serveQueryFeature(w http.ResponseWriter, r *http.Request) {
	feature := r.FormValue("feature")
	switch {
	case !h.PermitRead:
		http.Error(w, "access denied", http.StatusForbidden)
		return
	case r.Method != httpm.POST:
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	case feature == "":
		http.Error(w, "missing feature", http.StatusInternalServerError)
		return
	}
	nm := h.b.NetMap()
	if nm == nil {
		http.Error(w, "no netmap", http.StatusServiceUnavailable)
		return
	}

	b, err := json.Marshal(&tailcfg.QueryFeatureRequest{
		NodeKey: nm.NodeKey,
		Feature: feature,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(),
		"POST", "https://unused/machine/feature/query", bytes.NewReader(b))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := h.b.DoNoiseRequest(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func defBool(a string, def bool) bool {
	if a == "" {
		return def
	}
	v, err := strconv.ParseBool(a)
	if err != nil {
		return def
	}
	return v
}

func (h *Handler) serveDebugCapture(w http.ResponseWriter, r *http.Request) {
	if !h.PermitWrite {
		http.Error(w, "debug access denied", http.StatusForbidden)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(200)
	w.(http.Flusher).Flush()
	h.b.StreamDebugCapture(r.Context(), w)
}

func (h *Handler) serveDebugLog(w http.ResponseWriter, r *http.Request) {
	if !h.PermitRead {
		http.Error(w, "debug-log access denied", http.StatusForbidden)
		return
	}
	if r.Method != httpm.POST {
		http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}
	defer h.b.TryFlushLogs() // kick off upload after we're done logging

	type logRequestJSON struct {
		Lines  []string
		Prefix string
	}

	var logRequest logRequestJSON
	if err := json.NewDecoder(r.Body).Decode(&logRequest); err != nil {
		http.Error(w, "invalid JSON body", 400)
		return
	}

	prefix := logRequest.Prefix
	if prefix == "" {
		prefix = "debug-log"
	}
	logf := logger.WithPrefix(h.logf, prefix+": ")

	// We can write logs too fast for logtail to handle, even when
	// opting-out of rate limits. Limit ourselves to at most one message
	// per 20ms and a burst of 60 log lines, which should be fast enough to
	// not block for too long but slow enough that we can upload all lines.
	logf = logger.SlowLoggerWithClock(r.Context(), logf, 20*time.Millisecond, 60, h.clock.Now)

	for _, line := range logRequest.Lines {
		logf("%s", line)
	}

	w.WriteHeader(http.StatusNoContent)
}

var (
	metricInvalidRequests = clientmetric.NewCounter("localapi_invalid_requests")

	// User-visible LocalAPI endpoints.
	metricFilePutCalls = clientmetric.NewCounter("localapi_file_put")
)
