package c2

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/dnspb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/miekg/dns"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

func TestDNSCanarySameParentTriggersOnce(t *testing.T) {
	const parent = "canary-parent.invalid."
	// Legacy records can contain mixed-case parent names, and DNS queries are
	// case-insensitive independently of how the registered name was spelled.
	canary := createDNSCanaryRecord(t, models.NewUUID().String()+".CaNaRy-PaReNt.InVaLiD.")
	events := subscribeDNSCanaryEvents(t)
	lookups := countDNSCanaryLookups(t)
	server := newTestDNSServer()

	response := queryDNSCanary(t, server, parent, strings.ToUpper(canary.Domain), true)
	assertDNSCanaryResponse(t, response)
	if got := lookups.Load(); got != 1 {
		t.Fatalf("first query performed %d canary lookups, want 1", got)
	}
	first := assertDNSCanaryRecord(t, canary, 1)
	assertDNSCanaryEvents(t, collectDNSCanaryEvents(t, events, false), canary, 1)

	lookups.Store(0)
	response = queryDNSCanary(t, server, parent, strings.ToLower(canary.Domain), true)
	assertDNSCanaryResponse(t, response)
	if got := lookups.Load(); got != 1 {
		t.Fatalf("repeat query performed %d canary lookups, want 1", got)
	}
	second := assertDNSCanaryRecord(t, canary, 2)
	if !second.FirstTrigger.Equal(first.FirstTrigger) || second.LatestTrigger.Before(first.LatestTrigger) {
		t.Error("repeat query changed the first trigger or moved the latest trigger backwards")
	}
	assertDNSCanaryEvents(t, collectDNSCanaryEvents(t, events, false), canary, 0)
}

func TestDNSCanaryC2Routing(t *testing.T) {
	tests := []struct {
		name       string
		registered bool
		enabled    bool
	}{
		{name: "unknown_name", enabled: true},
		{name: "disabled_detection", registered: true},
		{name: "registered_name_precedes_c2", registered: true, enabled: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, parent, domain, checksum := newDNSCanaryNOP(t)
			var canary *models.DNSCanary
			if tt.registered {
				canary = createDNSCanaryRecord(t, domain)
			}
			events := subscribeDNSCanaryEvents(t)
			lookups := countDNSCanaryLookups(t)
			response := queryDNSCanary(t, server, parent, domain, tt.enabled)
			wantLookups := int64(0)
			if tt.enabled {
				wantLookups = 1
			}
			if got := lookups.Load(); got != wantLookups {
				t.Fatalf("performed %d canary lookups, want %d", got, wantLookups)
			}
			if tt.registered && tt.enabled {
				assertDNSCanaryResponse(t, response)
				assertDNSCanaryRecord(t, canary, 1)
				assertDNSCanaryEvents(t, collectDNSCanaryEvents(t, events, false), canary, 1)
				return
			}
			assertDNSCanaryNOP(t, response, checksum)
			assertDNSCanaryEvents(t, collectDNSCanaryEvents(t, events, false), canary, 0)
			if tt.registered {
				assertDNSCanaryRecord(t, canary, 0)
			}
		})
	}
}

func TestDNSCanaryFailedTriggerDoesNotPublish(t *testing.T) {
	const parent = "canary-failure.invalid."
	canary := createDNSCanaryRecord(t, models.NewUUID().String()+"."+parent)
	events := subscribeDNSCanaryEvents(t)
	reject := rejectDNSCanaryUpdates(t)
	server := newTestDNSServer()

	assertDNSCanaryResponse(t, queryDNSCanary(t, server, parent, canary.Domain, true))
	assertDNSCanaryRecord(t, canary, 0)
	assertDNSCanaryEvents(t, collectDNSCanaryEvents(t, events, false), canary, 0)

	reject.Store(false)
	assertDNSCanaryResponse(t, queryDNSCanary(t, server, parent, canary.Domain, true))
	assertDNSCanaryRecord(t, canary, 1)
	assertDNSCanaryEvents(t, collectDNSCanaryEvents(t, events, false), canary, 1)
}

func createDNSCanaryRecord(t *testing.T, domain string) *models.DNSCanary {
	t.Helper()
	canary := &models.DNSCanary{ImplantName: "dns-canary-" + models.NewUUID().String(), Domain: domain}
	if err := db.Session().Create(canary).Error; err != nil {
		t.Fatalf("create canary: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Session().Where("id = ?", canary.ID).Delete(&models.DNSCanary{}).Error; err != nil {
			t.Errorf("delete canary: %v", err)
		}
	})
	return canary
}

func assertDNSCanaryRecord(t *testing.T, canary *models.DNSCanary, count uint32) models.DNSCanary {
	t.Helper()
	var stored models.DNSCanary
	if err := db.Session().Where("id = ?", canary.ID).First(&stored).Error; err != nil {
		t.Fatalf("reload canary: %v", err)
	}
	if stored.Count != count || stored.Triggered != (count > 0) {
		t.Fatalf("stored count=%d triggered=%v, want count=%d triggered=%v", stored.Count, stored.Triggered, count, count > 0)
	}
	if stored.FirstTrigger.IsZero() != (count == 0) || stored.LatestTrigger.IsZero() != (count == 0) {
		t.Error("trigger timestamps do not match the persisted count")
	}
	return stored
}

func countDNSCanaryLookups(t *testing.T) *atomic.Int64 {
	t.Helper()
	var count atomic.Int64
	name := "test:count-dns-canary-lookups-" + models.NewUUID().String()
	if err := db.Client.Callback().Query().Before("gorm:query").Register(name, func(tx *gorm.DB) {
		if tx.Statement.Table == "dns_canaries" {
			count.Add(1)
		}
	}); err != nil {
		t.Fatalf("register canary query counter: %v", err)
	}
	t.Cleanup(func() { _ = db.Client.Callback().Query().Remove(name) })
	return &count
}

func rejectDNSCanaryUpdates(t *testing.T) *atomic.Bool {
	t.Helper()
	var reject atomic.Bool
	reject.Store(true)
	name := "test:reject-dns-canary-updates-" + models.NewUUID().String()
	if err := db.Client.Callback().Update().Before("gorm:update").Register(name, func(tx *gorm.DB) {
		if reject.Load() && tx.Statement.Table == "dns_canaries" {
			_ = tx.AddError(errors.New("injected DNS canary update failure"))
		}
	}); err != nil {
		t.Fatalf("register canary update failure: %v", err)
	}
	t.Cleanup(func() { _ = db.Client.Callback().Update().Remove(name) })
	return &reject
}

func subscribeDNSCanaryEvents(t *testing.T) chan core.Event {
	t.Helper()
	events := core.EventBroker.Subscribe()
	t.Cleanup(func() { core.EventBroker.Unsubscribe(events) })
	// Subscription registration is asynchronous. Complete a broker round trip
	// before sending requests so the first canary event cannot be missed.
	collectDNSCanaryEvents(t, events, true)
	return events
}

func collectDNSCanaryEvents(t *testing.T, events <-chan core.Event, subscribing bool) []core.Event {
	t.Helper()
	marker := core.Event{EventType: "dns-canary-test-barrier", Data: []byte(models.NewUUID().String())}
	core.EventBroker.Publish(marker)
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	var canaries []core.Event
	for {
		select {
		case event := <-events:
			if event.EventType == marker.EventType && string(event.Data) == string(marker.Data) {
				return canaries
			}
			if event.EventType == consts.CanaryEvent {
				canaries = append(canaries, event)
			}
		case <-ticker.C:
			if subscribing {
				marker.Data = []byte(models.NewUUID().String())
				core.EventBroker.Publish(marker)
			}
		case <-timer.C:
			t.Fatal("timed out waiting for DNS canary event barrier")
		}
	}
}

func assertDNSCanaryEvents(t *testing.T, events []core.Event, canary *models.DNSCanary, count int) {
	t.Helper()
	if len(events) != count {
		t.Fatalf("received %d canary events, want %d", len(events), count)
	}
	for _, event := range events {
		if event.Session == nil || event.Session.Name != canary.ImplantName || string(event.Data) != canary.Domain {
			t.Errorf("unexpected canary event: %+v", event)
		}
	}
}

func newDNSCanaryNOP(t *testing.T) (*SliverDNSServer, string, string, uint32) {
	t.Helper()
	server := newTestDNSServer()
	const sessionID = uint32(0x123456)
	server.sessions.Store(sessionID, newTestDNSSession(sessionID))
	parent := models.NewUUID().String() + ".canary-c2.invalid."
	msg := &dnspb.DNSMessage{Type: dnspb.DNSMessageType_NOP, ID: (1 << 24) | sessionID, Data: []byte("canary-routing")}
	raw, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal DNS NOP: %v", err)
	}
	domain, err := encodeMessageDomain("."+parent, msg)
	if err != nil {
		t.Fatalf("encode DNS NOP: %v", err)
	}
	return server, parent, domain, crc32.ChecksumIEEE(raw)
}

func queryDNSCanary(t *testing.T, server *SliverDNSServer, parent, domain string, enabled bool) *dns.Msg {
	t.Helper()
	req := new(dns.Msg)
	req.SetQuestion(domain, dns.TypeA)
	writer := &canaryDNSResponseWriter{}
	server.HandleDNSRequest([]string{parent}, enabled, writer, req)
	if writer.response == nil {
		t.Fatal("DNS handler did not write a response")
	}
	return writer.response
}

func assertDNSCanaryResponse(t *testing.T, response *dns.Msg) {
	t.Helper()
	if response.Rcode != dns.RcodeNameError || !response.Authoritative {
		t.Fatalf("canary response rcode=%d authoritative=%v, want authoritative NXDOMAIN", response.Rcode, response.Authoritative)
	}
}

func assertDNSCanaryNOP(t *testing.T, response *dns.Msg, checksum uint32) {
	t.Helper()
	if response.Rcode != dns.RcodeSuccess || len(response.Answer) != 1 {
		t.Fatalf("NOP response rcode=%d answers=%d, want success with one answer", response.Rcode, len(response.Answer))
	}
	answer, ok := response.Answer[0].(*dns.A)
	if !ok || len(answer.A) != 4 {
		t.Fatalf("unexpected DNS NOP answer: %v", response.Answer[0])
	}
	if got := binary.LittleEndian.Uint32(answer.A); got != checksum {
		t.Fatalf("NOP checksum=%d, want %d", got, checksum)
	}
}

type canaryDNSResponseWriter struct {
	response *dns.Msg
}

func (w *canaryDNSResponseWriter) LocalAddr() net.Addr  { return &net.UDPAddr{} }
func (w *canaryDNSResponseWriter) RemoteAddr() net.Addr { return &net.UDPAddr{} }
func (w *canaryDNSResponseWriter) Close() error         { return nil }
func (w *canaryDNSResponseWriter) TsigStatus() error    { return nil }
func (w *canaryDNSResponseWriter) TsigTimersOnly(bool)  {}
func (w *canaryDNSResponseWriter) Hijack()              {}

func (w *canaryDNSResponseWriter) WriteMsg(msg *dns.Msg) error {
	w.response = msg.Copy()
	return nil
}

func (w *canaryDNSResponseWriter) Write(data []byte) (int, error) {
	msg := new(dns.Msg)
	if err := msg.Unpack(data); err != nil {
		return 0, err
	}
	w.response = msg
	return len(data), nil
}
