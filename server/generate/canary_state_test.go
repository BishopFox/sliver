package generate

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"gorm.io/gorm"
)

func createCanaryStateFixture(t *testing.T) *models.DNSCanary {
	t.Helper()
	canary := &models.DNSCanary{
		ImplantName: "canary_state_" + models.NewUUID().String(),
		Domain:      models.NewUUID().String() + ".canary.invalid.",
	}
	if err := db.Session().Create(canary).Error; err != nil {
		t.Fatalf("create canary: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Session().Delete(canary).Error; err != nil {
			t.Errorf("delete canary: %v", err)
		}
	})
	return canary
}

func readCanaryState(t *testing.T, id models.UUID) models.DNSCanary {
	t.Helper()
	var canary models.DNSCanary
	if err := db.Session().First(&canary, "id = ?", id).Error; err != nil {
		t.Fatalf("read canary: %v", err)
	}
	return canary
}

func assertCanaryTriggerCount(t *testing.T, canary models.DNSCanary, count uint32) {
	t.Helper()
	if !canary.Triggered || canary.Count != count {
		t.Errorf("triggered/count = %v/%d, want true/%d", canary.Triggered, canary.Count, count)
	}
	if canary.FirstTrigger.IsZero() || canary.LatestTrigger.Before(canary.FirstTrigger) {
		t.Errorf("invalid trigger times: first=%v latest=%v", canary.FirstTrigger, canary.LatestTrigger)
	}
}

func TestTriggerCanaryPersistsRepeatedQueries(t *testing.T) {
	canary := createCanaryStateFixture(t)
	// Reuse a stale caller snapshot; persistence must use the database state.
	snapshot := &clientpb.DNSCanary{ID: canary.ID.String(), Count: 99}
	first, err := TriggerCanary(snapshot)
	if err != nil || !first {
		t.Fatalf("first trigger = (%v, %v), want (true, nil)", first, err)
	}
	initial := readCanaryState(t, canary.ID)
	assertCanaryTriggerCount(t, initial, 1)
	first, err = TriggerCanary(snapshot)
	if err != nil || first {
		t.Fatalf("repeat trigger = (%v, %v), want (false, nil)", first, err)
	}
	latest := readCanaryState(t, canary.ID)
	assertCanaryTriggerCount(t, latest, 2)
	if !latest.FirstTrigger.Equal(initial.FirstTrigger) || latest.LatestTrigger.Before(initial.LatestTrigger) {
		t.Error("repeat query changed first trigger time or moved latest trigger backwards")
	}
	if latest.Domain != canary.Domain || latest.ImplantName != canary.ImplantName {
		t.Error("trigger overwrote canary identity from the caller snapshot")
	}
}

func TestTriggerCanaryConcurrentQueries(t *testing.T) {
	const queries = 12
	canary := createCanaryStateFixture(t)
	snapshot := &clientpb.DNSCanary{ID: canary.ID.String()}
	type result struct {
		first bool
		err   error
	}
	results := make(chan result, queries)
	start := make(chan struct{})
	for range queries {
		go func() {
			<-start
			first, err := TriggerCanary(snapshot)
			results <- result{first: first, err: err}
		}()
	}
	close(start)
	firstTriggers := 0
	for range queries {
		got := <-results
		if got.err != nil {
			t.Errorf("concurrent trigger: %v", got.err)
		}
		if got.first {
			firstTriggers++
		}
	}
	if firstTriggers != 1 {
		t.Errorf("first triggers = %d, want 1", firstTriggers)
	}
	assertCanaryTriggerCount(t, readCanaryState(t, canary.ID), queries)
}

func TestTriggerCanaryRejectsMissingIdentity(t *testing.T) {
	canary := createCanaryStateFixture(t)
	var before int64
	if err := db.Session().Model(&models.DNSCanary{}).Count(&before).Error; err != nil {
		t.Fatalf("count canaries: %v", err)
	}
	for _, snapshot := range []*clientpb.DNSCanary{
		nil,
		{},
		{ID: "not-a-uuid"},
		{ID: models.NilUUID().String()},
		{ID: models.NewUUID().String()},
	} {
		first, err := TriggerCanary(snapshot)
		if err == nil || first {
			t.Errorf("trigger for %v = (%v, %v), want (false, error)", snapshot, first, err)
		}
	}
	var after int64
	if err := db.Session().Model(&models.DNSCanary{}).Count(&after).Error; err != nil {
		t.Fatalf("count canaries: %v", err)
	}
	if after != before {
		t.Errorf("invalid identity inserted canaries: count changed from %d to %d", before, after)
	}
	unchanged := readCanaryState(t, canary.ID)
	if unchanged.Triggered || unchanged.Count != 0 {
		t.Error("invalid identity changed an existing canary")
	}
}

func TestTriggerCanaryRollsBackWriteFailure(t *testing.T) {
	canary := createCanaryStateFixture(t)
	injectedErr := errors.New("injected canary timestamp write failure")
	callbackName := "test:canary-trigger-write-" + canary.ID.String()
	var rejectWrite atomic.Bool
	rejectWrite.Store(true)
	if err := db.Client.Callback().Update().Before("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		update, ok := tx.Statement.Dest.(models.DNSCanary)
		if rejectWrite.Load() && ok && !update.LatestTrigger.IsZero() {
			_ = tx.AddError(injectedErr)
		}
	}); err != nil {
		t.Fatalf("register write failure: %v", err)
	}
	t.Cleanup(func() {
		rejectWrite.Store(false)
		if err := db.Client.Callback().Update().Remove(callbackName); err != nil {
			t.Errorf("remove write failure callback: %v", err)
		}
	})
	first, err := TriggerCanary(&clientpb.DNSCanary{ID: canary.ID.String()})
	if first || !errors.Is(err, injectedErr) {
		t.Fatalf("failed trigger = (%v, %v), want (false, injected error)", first, err)
	}
	unchanged := readCanaryState(t, canary.ID)
	if unchanged.Triggered || unchanged.Count != 0 || !unchanged.FirstTrigger.IsZero() || !unchanged.LatestTrigger.IsZero() {
		t.Error("failed trigger did not roll back count and trigger timestamps")
	}
	rejectWrite.Store(false)
	first, err = TriggerCanary(&clientpb.DNSCanary{ID: canary.ID.String()})
	if err != nil || !first {
		t.Fatalf("trigger after rollback = (%v, %v), want (true, nil)", first, err)
	}
	assertCanaryTriggerCount(t, readCanaryState(t, canary.ID), 1)
}

func TestGenerateCanaryNormalizesParent(t *testing.T) {
	for _, parent := range []string{"MiXeD.Example", "MIXED.EXAMPLE.", " \t.MiXeD.Example \t"} {
		t.Run(parent, func(t *testing.T) {
			name := "canary_normalization_" + models.NewUUID().String()
			t.Cleanup(func() {
				if err := db.Session().Where("implant_name = ?", name).Delete(&models.DNSCanary{}).Error; err != nil {
					t.Errorf("delete generated canary: %v", err)
				}
			})
			generator := CanaryGenerator{ImplantName: name, ParentDomains: []string{parent}}
			domain := generator.GenerateCanary()
			if !strings.HasSuffix(domain, ".mixed.example.") || domain != strings.ToLower(domain) {
				t.Fatalf("generated domain %q was not normalized", domain)
			}
			canary, err := db.CanaryByDomain(domain)
			if err != nil {
				t.Fatalf("look up normalized canary: %v", err)
			}
			if canary.Domain != domain || canary.ImplantName != name {
				t.Errorf("stored canary = %v, want domain %s and implant %s", canary, domain, name)
			}
		})
	}
}
