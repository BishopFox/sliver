package buildscheduler

import (
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/bishopfox/sliver/server/buildplan"
)

var testReference = buildplan.Reference{
	Name: "builtin/sliver", Version: "1.0.0",
	Digest: "sha256:91dec1eb4394dbc34ab9e0594d7297424cff0d5523e6ab3b3f6d4ba4f90a1af0",
}

func testRequirement() Requirement {
	return Requirement{
		Spec: testReference, Target: buildplan.Target{OS: "linux", Arch: "amd64"},
		Mode: "session", Protocols: []string{"mtls"},
		Output:    buildplan.Artifact{Kind: "executable", Container: "elf", Target: buildplan.Target{OS: "linux", Arch: "amd64"}, GoBuildMode: "default"},
		Toolchain: Toolchain{Driver: "go", Requirements: []string{"go"}},
	}
}

func testRegistration(name string, requirement Requirement) Registration {
	return Registration{Name: name, Specs: []SpecSupport{{Reference: requirement.Spec, Capabilities: []Capability{{
		Target: requirement.Target, Mode: requirement.Mode, Protocols: []string{"mtls", "http"},
		Input: requirement.Input, Output: requirement.Output, Toolchain: requirement.Toolchain,
	}}}}}
}

func testStatus(cpu int, memory uint64) Status {
	return Status{EffectiveCPUCores: cpu, AvailableMemoryBytes: memory, Toolchains: []ToolchainStatus{{
		Target: buildplan.Target{OS: "linux", Arch: "amd64"}, Driver: "go", Requirements: []string{"go"}, Identity: "go-test", Available: true,
	}}}
}

func registerTestBuilder(t *testing.T, scheduler *Scheduler, name string, cpu int, memory uint64, now time.Time) {
	t.Helper()
	if err := scheduler.Register(testRegistration(name, testRequirement()), testStatus(cpu, memory), now); err != nil {
		t.Fatal(err)
	}
}

func TestSelectMemoryCPUNameAndPin(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	scheduler := New(DefaultStatusTTL)
	registerTestBuilder(t, scheduler, "alpha", 4, 1_499_999, now)
	registerTestBuilder(t, scheduler, "bravo", 2, 1_500_000, now)
	registerTestBuilder(t, scheduler, "charlie", 8, 1_499_999, now)
	requirement := testRequirement()
	order := scheduler.Candidates(requirement, "", now)
	if got := []string{order[0].Name, order[1].Name, order[2].Name}; !slices.Equal(got, []string{"bravo", "charlie", "alpha"}) {
		t.Fatalf("rounded MB then CPU ordering: %v", got)
	}
	if order[0].RoundedAvailableMB != 2 || order[1].RoundedAvailableMB != 1 {
		t.Fatalf("decimal MB rounding: %+v", order)
	}
	if err := scheduler.UpdateStatus("alpha", testStatus(8, 1_499_999), now); err != nil {
		t.Fatal(err)
	}
	order = scheduler.Candidates(requirement, "", now)
	if order[1].Name != "alpha" || order[2].Name != "charlie" {
		t.Fatalf("stable name tie-break: %+v", order)
	}
	pinned, err := scheduler.SelectAndReserve(requirement, "charlie", 1_000_000, now)
	if err != nil || pinned.Builder != "charlie" || pinned.Decision.Pin != "charlie" {
		t.Fatalf("pinned selection: %+v, %v", pinned, err)
	}
	if !scheduler.Release(pinned.ID) || scheduler.Release(pinned.ID) {
		t.Fatal("release must remove a reservation once")
	}
}

func TestStaleOfflineStatusAndImmutableSupport(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	scheduler := New(30 * time.Second)
	requirement := testRequirement()
	registerTestBuilder(t, scheduler, "alpha", 4, 2_000_000_000, now)
	registerTestBuilder(t, scheduler, "bravo", 4, 1_000_000_000, now)
	if err := scheduler.UpdateStatus("bravo", testStatus(4, 3_000_000_000), now.Add(31*time.Second)); err != nil {
		t.Fatal(err)
	}
	if scheduler.CanRun("alpha", requirement, now.Add(31*time.Second)) {
		t.Fatal("stale builder was treated as available")
	}
	selected, err := scheduler.SelectAndReserve(requirement, "", 1_000_000, now.Add(31*time.Second))
	if err != nil || selected.Builder != "bravo" {
		t.Fatalf("fresh builder selection: %+v, %v", selected, err)
	}
	scheduler.Disconnect("bravo")
	if scheduler.CanRun("bravo", requirement, now.Add(31*time.Second)) {
		t.Fatal("offline builder was treated as available")
	}
	if _, err := scheduler.SelectAndReserve(requirement, "bravo", 1_000_000, now.Add(31*time.Second)); !errors.Is(err, ErrNoBuilder) {
		t.Fatalf("offline pinned builder: %v", err)
	}
	entries := scheduler.Inventory(now.Add(31 * time.Second))
	if entries[0].Fresh || entries[1].Connected || entries[1].Specs[0].Reference != testReference {
		t.Fatalf("inventory after stale/offline: %+v", entries)
	}
	changed := testRequirement()
	changed.Spec.Digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := scheduler.UpdateStatus("alpha", testStatus(10, 8_000_000_000), now.Add(32*time.Second)); err != nil {
		t.Fatal(err)
	}
	if scheduler.CanRun("alpha", changed, now.Add(32*time.Second)) {
		t.Fatal("status refresh mutated the advertised spec digest")
	}
}

func TestConcurrentReservationsDoNotReuseReportedMemory(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	scheduler := New(DefaultStatusTTL)
	registerTestBuilder(t, scheduler, "alpha", 4, 1_000_000_000, now)
	registerTestBuilder(t, scheduler, "bravo", 4, 1_000_000_000, now)
	var group sync.WaitGroup
	results := make(chan Reservation, 3)
	errs := make(chan error, 3)
	for range 3 {
		group.Add(1)
		go func() {
			defer group.Done()
			reservation, err := scheduler.SelectAndReserve(testRequirement(), "", 600_000_000, now)
			if err != nil {
				errs <- err
				return
			}
			results <- reservation
		}()
	}
	group.Wait()
	close(results)
	close(errs)
	selected := make(map[string]bool)
	for reservation := range results {
		if selected[reservation.Builder] || reservation.Decision.Candidates[0].Name != reservation.Builder {
			t.Fatalf("duplicate or inconsistent capacity selection: %+v", reservation)
		}
		selected[reservation.Builder] = true
	}
	if len(selected) != 2 || len(errs) != 1 {
		t.Fatalf("expected two distinct reservations and one capacity rejection: selected=%v errors=%d", selected, len(errs))
	}
	for err := range errs {
		if !errors.Is(err, ErrNoBuilder) {
			t.Fatal(err)
		}
	}
	entries := scheduler.Inventory(now)
	for _, entry := range entries {
		if entry.ReservedMemoryBytes != 600_000_000 || entry.AvailableMemoryBytes != 400_000_000 {
			t.Fatalf("uncommitted reported memory: %+v", entry)
		}
	}
}

func TestUnknownEstimateReservesBuilderExclusivelyAcrossStatusUpdates(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	scheduler := New(DefaultStatusTTL)
	registerTestBuilder(t, scheduler, "alpha", 4, 1_000_000_000, now)
	requirement := testRequirement()
	reservation, err := scheduler.SelectAndReserve(requirement, "", 0, now)
	if err != nil || reservation.Builder != "alpha" || reservation.Decision.PeakMemoryBytes != 1_000_000_000 {
		t.Fatalf("exclusive fallback reservation: %+v, %v", reservation, err)
	}
	if len(scheduler.Eligible(requirement, "", 0, now)) != 0 {
		t.Fatal("builder accepted another step during exclusive reservation")
	}
	if err := scheduler.UpdateStatus("alpha", testStatus(4, 2_000_000_000), now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if len(scheduler.Eligible(requirement, "", 100_000_000, now.Add(time.Second))) != 0 {
		t.Fatal("fresh capacity report released exclusive reservation")
	}
	if scheduler.CanRun("alpha", requirement, now.Add(time.Second)) || !scheduler.Inventory(now.Add(time.Second))[0].Exclusive {
		t.Fatal("exclusive reservation appeared available in scheduler inventory")
	}
	if !scheduler.Release(reservation.ID) {
		t.Fatal("exclusive reservation did not release")
	}
	if got := scheduler.Eligible(requirement, "", 100_000_000, now.Add(time.Second)); len(got) != 1 || got[0].AvailableMemoryBytes != 2_000_000_000 {
		t.Fatalf("released builder did not use latest status: %+v", got)
	}
}

func TestImpossibleEffectiveCPUCoresRejected(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	scheduler := New(DefaultStatusTTL)
	status := testStatus(MaxEffectiveCPUCores+1, 1_000_000_000)
	if err := scheduler.Register(testRegistration("alpha", testRequirement()), status, now); err == nil {
		t.Fatal("accepted impossible CPU count at registration")
	}
	registerTestBuilder(t, scheduler, "alpha", 4, 1_000_000_000, now)
	if err := scheduler.UpdateStatus("alpha", status, now.Add(time.Second)); err == nil {
		t.Fatal("accepted impossible CPU count in status update")
	}
}

func TestStepBuilderNameMustRouteExactly(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	for _, name := range []string{" worker", "worker ", "worker\nname", "server"} {
		registration := testRegistration(name, testRequirement())
		if err := New(DefaultStatusTTL).Register(registration, testStatus(4, 1_000_000_000), now); err == nil {
			t.Fatalf("accepted unroutable builder name %q", name)
		}
	}
}

func TestExactTupleAndToolchainRequired(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	scheduler := New(DefaultStatusTTL)
	registerTestBuilder(t, scheduler, "alpha", 4, 4_000_000_000, now)
	requirement := testRequirement()
	changes := []func(*Requirement){
		func(r *Requirement) { r.Spec.Name = "other/core" },
		func(r *Requirement) {
			r.Spec.Digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		},
		func(r *Requirement) { r.Target.Arch = "arm64" },
		func(r *Requirement) { r.Mode = "beacon" },
		func(r *Requirement) { r.Protocols = []string{"dns"} },
		func(r *Requirement) { r.Output.Kind = "shellcode" },
		func(r *Requirement) { r.Output.Container = "raw" },
		func(r *Requirement) { r.Toolchain.Requirements = []string{"go", "cgo"} },
		func(r *Requirement) { input := r.Output; r.Input = &input },
	}
	for i, change := range changes {
		wrong := requirement
		change(&wrong)
		if scheduler.CanRun("alpha", wrong, now) {
			t.Fatalf("change %d unexpectedly matches immutable tuple", i)
		}
	}
	resolved := requirement
	resolved.Output.EntrySymbols = []string{"StartW"}
	if !scheduler.CanRun("alpha", resolved, now) {
		t.Fatal("validated entry symbols resolved from options should fit an unrestricted spec tuple")
	}
	if err := scheduler.UpdateStatus("alpha", Status{EffectiveCPUCores: 4, AvailableMemoryBytes: 4_000_000_000,
		Toolchains: []ToolchainStatus{{Target: requirement.Target, Driver: "go", Requirements: []string{"go"}, Identity: "go-test", Available: false}}}, now); err != nil {
		t.Fatal(err)
	}
	if scheduler.CanRun("alpha", requirement, now) {
		t.Fatal("unavailable toolchain was accepted")
	}
	wrongTargetStatus := testStatus(4, 4_000_000_000)
	wrongTargetStatus.Toolchains[0].Target.Arch = "arm64"
	if err := scheduler.UpdateStatus("alpha", wrongTargetStatus, now); err != nil {
		t.Fatal(err)
	}
	if scheduler.CanRun("alpha", requirement, now) {
		t.Fatal("toolchain availability for a different target was accepted")
	}
}

func TestRegistrationAndInventoryAreCopies(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	scheduler := New(DefaultStatusTTL)
	registration := testRegistration("alpha", testRequirement())
	status := testStatus(4, 2_000_000_000)
	if err := scheduler.Register(registration, status, now); err != nil {
		t.Fatal(err)
	}
	registration.Specs[0].Capabilities[0].Protocols[0] = "dns"
	status.Toolchains[0].Available = false
	if !scheduler.CanRun("alpha", testRequirement(), now) {
		t.Fatal("caller mutated registered support or status")
	}
	entries := scheduler.Inventory(now)
	entries[0].Specs[0].Capabilities[0].Protocols[0] = "dns"
	entries[0].Toolchains[0].Available = false
	if !scheduler.CanRun("alpha", testRequirement(), now) {
		t.Fatal("inventory exposed mutable scheduler state")
	}
}
