package extension

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type testExtension struct {
	id string
}

func (e *testExtension) Load() error {
	return nil
}

func (e *testExtension) Call(_ string, arguments []byte, callback func([]byte)) error {
	callback(arguments)
	return nil
}

func (e *testExtension) GetID() string {
	return e.id
}

func (e *testExtension) GetArch() string {
	return "test"
}

type registeringTestExtension struct {
	testExtension
	loadCalls *atomic.Int32
	loadErr   error
	started   chan struct{}
	release   chan struct{}
}

func (e *registeringTestExtension) Load() error {
	e.loadCalls.Add(1)
	if e.started != nil {
		close(e.started)
	}
	if e.release != nil {
		<-e.release
	}
	return e.loadErr
}

func resetExtensionRegistry(t *testing.T) {
	t.Helper()
	extensionsMu.Lock()
	previousExtensions := extensions
	previousLoads := extensionLoads
	extensions = make(map[string]Extension)
	extensionLoads = make(map[string]*extensionLoad)
	extensionsMu.Unlock()
	t.Cleanup(func() {
		extensionsMu.Lock()
		extensions = previousExtensions
		extensionLoads = previousLoads
		extensionsMu.Unlock()
	})
}

func waitForExtensionWaiters(t *testing.T, extensionID string, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		extensionsMu.RLock()
		loading := extensionLoads[extensionID]
		got := 0
		if loading != nil {
			got = loading.waiters
		}
		extensionsMu.RUnlock()
		if got == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("registration waiters = %d, want %d", got, want)
		}
		runtime.Gosched()
	}
}

func TestExtensionRegistryConcurrentAccess(t *testing.T) {
	resetExtensionRegistry(t)

	const count = 64
	errCh := make(chan error, count)
	var waitGroup sync.WaitGroup
	for index := 0; index < count; index++ {
		index := index
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			id := fmt.Sprintf("extension-%d", index)
			Add(&testExtension{id: id})
			argument := []byte(id)
			if err := Run(id, "run", argument, func(output []byte) {
				if string(output) != id {
					errCh <- fmt.Errorf("callback output = %q, want %q", output, id)
				}
			}); err != nil {
				errCh <- err
			}
			_ = List()
		}()
	}
	waitGroup.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}

	if got := len(List()); got != count {
		t.Fatalf("List returned %d extensions, want %d", got, count)
	}
}

func TestRegisterLoadsExtensionOnlyOnce(t *testing.T) {
	resetExtensionRegistry(t)

	const extensionID = "duplicate-extension"
	var loadCalls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	first := &registeringTestExtension{
		testExtension: testExtension{id: extensionID},
		loadCalls:     &loadCalls,
		started:       started,
		release:       release,
	}

	const registrations = 16
	errorsCh := make(chan error, registrations)
	go func() {
		errorsCh <- Register(first)
	}()
	<-started
	for index := 1; index < registrations; index++ {
		go func() {
			errorsCh <- Register(&registeringTestExtension{
				testExtension: testExtension{id: extensionID},
				loadCalls:     &loadCalls,
			})
		}()
	}
	waitForExtensionWaiters(t, extensionID, registrations-1)
	select {
	case err := <-errorsCh:
		t.Fatalf("Register returned before the in-flight load completed: %v", err)
	default:
	}
	close(release)
	for index := 0; index < registrations; index++ {
		if err := <-errorsCh; err != nil {
			t.Fatalf("Register returned an error: %v", err)
		}
	}

	if got := loadCalls.Load(); got != 1 {
		t.Fatalf("Load called %d times, want 1", got)
	}
	if got := List(); len(got) != 1 || got[0] != extensionID {
		t.Fatalf("List returned %q, want [%q]", got, extensionID)
	}

	if err := Register(&registeringTestExtension{
		testExtension: testExtension{id: extensionID},
		loadCalls:     &loadCalls,
	}); err != nil {
		t.Fatalf("repeat Register returned an error: %v", err)
	}
	if got := loadCalls.Load(); got != 1 {
		t.Fatalf("repeat Register called Load; total calls = %d, want 1", got)
	}
}

func TestRegisterRetriesAfterFailedLoad(t *testing.T) {
	resetExtensionRegistry(t)

	const extensionID = "retry-extension"
	var loadCalls atomic.Int32
	wantErr := errors.New("load failed")
	started := make(chan struct{})
	release := make(chan struct{})
	const registrations = 8
	errorsCh := make(chan error, registrations)
	go func() {
		errorsCh <- Register(&registeringTestExtension{
			testExtension: testExtension{id: extensionID},
			loadCalls:     &loadCalls,
			loadErr:       wantErr,
			started:       started,
			release:       release,
		})
	}()
	<-started
	for index := 1; index < registrations; index++ {
		go func() {
			errorsCh <- Register(&registeringTestExtension{
				testExtension: testExtension{id: extensionID},
				loadCalls:     &loadCalls,
				loadErr:       errors.New("unexpected follower load"),
			})
		}()
	}
	waitForExtensionWaiters(t, extensionID, registrations-1)
	close(release)
	for index := 0; index < registrations; index++ {
		if err := <-errorsCh; !errors.Is(err, wantErr) {
			t.Fatalf("Register error = %v, want shared error %v", err, wantErr)
		}
	}
	if got := loadCalls.Load(); got != 1 {
		t.Fatalf("failed Load called %d times, want 1", got)
	}
	if got := List(); len(got) != 0 {
		t.Fatalf("failed extension was published: %q", got)
	}

	if err := Register(&registeringTestExtension{
		testExtension: testExtension{id: extensionID},
		loadCalls:     &loadCalls,
	}); err != nil {
		t.Fatalf("retry Register returned an error: %v", err)
	}
	if got := loadCalls.Load(); got != 2 {
		t.Fatalf("Load called %d times, want 2", got)
	}
}
