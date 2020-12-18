package xproto

/*
	Tests for XGB.

	These tests only test the core X protocol at the moment. It isn't even
	close to complete coverage (and probably never will be), but it does test
	a number of different corners: requests with no replies, requests without
	replies, checked (i.e., synchronous) errors, unchecked (i.e., asynchronous)
	errors, and sequence number wrapping.

	There are also a couple of benchmarks that show the difference between
	correctly issuing lots of requests and gathering replies and
	incorrectly doing the same. (This particular difference is one of the
	claimed advantages of the XCB, and therefore XGB, family.)

	In sum, these tests are more focused on testing the core xgb package itself,
	rather than whether xproto has properly implemented the core X client
	protocol.
*/

import (
	"fmt"
	"log"
	"math/rand"
	"testing"
	"time"

	"github.com/BurntSushi/xgb"
)

// The X connection used throughout testing.
var X *xgb.Conn

// init initializes the X connection, seeds the RNG and starts waiting
// for events.
func init() {
	var err error

	X, err = xgb.NewConn()
	if err != nil {
		log.Fatal(err)
	}

	rand.Seed(time.Now().UnixNano())

	go grabEvents()
}

/******************************************************************************/
// Tests
/******************************************************************************/

// TestSynchronousError purposefully causes a BadWindow error in a
// MapWindow request, and checks it synchronously.
func TestSynchronousError(t *testing.T) {
	err := MapWindowChecked(X, 0).Check() // resource 0 is always invalid
	if err == nil {
		t.Fatalf("MapWindow: A MapWindow request that should return an " +
			"error has returned a nil error.")
	}
	verifyMapWindowError(t, err)
}

// TestAsynchronousError does the same thing as TestSynchronousError, but
// grabs the error asynchronously instead.
func TestAsynchronousError(t *testing.T) {
	MapWindow(X, 0) // resource id 0 is always invalid

	evOrErr := waitForEvent(t, 5)
	if evOrErr.ev != nil {
		t.Fatalf("After issuing an erroneous MapWindow request, we have "+
			"received an event rather than an error: %s", evOrErr.ev)
	}
	verifyMapWindowError(t, evOrErr.err)
}

// TestCookieBuffer issues (2^16) + n requets *without* replies to guarantee
// that the sequence number wraps and that the cookie buffer will have to
// flush itself (since there are no replies coming in to flush it).
// And just like TestSequenceWrap, we issue another request with a reply
// at the end to make sure XGB is still working properly.
func TestCookieBuffer(t *testing.T) {
	n := (1 << 16) + 10
	for i := 0; i < n; i++ {
		NoOperation(X)
	}
	TestProperty(t)
}

// TestSequenceWrap issues (2^16) + n requests w/ replies to guarantee that the
// sequence number (which is a 16 bit integer) will wrap. It then issues one
// final request to ensure things still work properly.
func TestSequenceWrap(t *testing.T) {
	n := (1 << 16) + 10
	for i := 0; i < n; i++ {
		_, err := InternAtom(X, false, 5, "RANDO").Reply()
		if err != nil {
			t.Fatalf("InternAtom: %s", err)
		}
	}
	TestProperty(t)
}

// TestProperty tests whether a random value can be set and read.
func TestProperty(t *testing.T) {
	propName := randString(20) // whatevs
	writeVal := randString(20)
	readVal, err := changeAndGetProp(propName, writeVal)
	if err != nil {
		t.Error(err)
	}

	if readVal != writeVal {
		t.Errorf("The value written, '%s', is not the same as the "+
			"value read '%s'.", writeVal, readVal)
	}
}

// TestWindowEvents creates a window, maps it, listens for configure notify
// events, issues a configure request, and checks for the appropriate
// configure notify event.
// This probably violates the notion of "test one thing and test it well,"
// but testing X stuff is unique since it involves so much state.
// Each request is checked to make sure there are no errors returned. If there
// is an error, the test is failed.
// You may see a window appear quickly and then disappear. Do not be alarmed :P
// It's possible that this test will yield a false negative because we cannot
// control our environment. That is, the window manager could override the
// placement set. However, we set override redirect on the window, so the
// window manager *shouldn't* touch our window if it is well-behaved.
func TestWindowEvents(t *testing.T) {
	// The geometry to set the window.
	gx, gy, gw, gh := 200, 400, 1000, 300

	wid, err := NewWindowId(X)
	if err != nil {
		t.Fatalf("NewId: %s", err)
	}

	screen := Setup(X).DefaultScreen(X) // alias
	err = CreateWindowChecked(X, screen.RootDepth, wid, screen.Root,
		0, 0, 500, 500, 0,
		WindowClassInputOutput, screen.RootVisual,
		CwBackPixel|CwOverrideRedirect, []uint32{0xffffffff, 1}).Check()
	if err != nil {
		t.Fatalf("CreateWindow: %s", err)
	}

	err = MapWindowChecked(X, wid).Check()
	if err != nil {
		t.Fatalf("MapWindow: %s", err)
	}

	// We don't listen in the CreateWindow request so that we don't get
	// a MapNotify event.
	err = ChangeWindowAttributesChecked(X, wid,
		CwEventMask, []uint32{EventMaskStructureNotify}).Check()
	if err != nil {
		t.Fatalf("ChangeWindowAttributes: %s", err)
	}

	err = ConfigureWindowChecked(X, wid,
		ConfigWindowX|ConfigWindowY|
			ConfigWindowWidth|ConfigWindowHeight,
		[]uint32{uint32(gx), uint32(gy), uint32(gw), uint32(gh)}).Check()
	if err != nil {
		t.Fatalf("ConfigureWindow: %s", err)
	}

	evOrErr := waitForEvent(t, 5)
	switch event := evOrErr.ev.(type) {
	case ConfigureNotifyEvent:
		if event.X != int16(gx) {
			t.Fatalf("x was set to %d but ConfigureNotify reports %d",
				gx, event.X)
		}
		if event.Y != int16(gy) {
			t.Fatalf("y was set to %d but ConfigureNotify reports %d",
				gy, event.Y)
		}
		if event.Width != uint16(gw) {
			t.Fatalf("width was set to %d but ConfigureNotify reports %d",
				gw, event.Width)
		}
		if event.Height != uint16(gh) {
			t.Fatalf("height was set to %d but ConfigureNotify reports %d",
				gh, event.Height)
		}
	default:
		t.Fatalf("Expected a ConfigureNotifyEvent but got %T instead.", event)
	}

	// Okay, clean up!
	err = ChangeWindowAttributesChecked(X, wid,
		CwEventMask, []uint32{0}).Check()
	if err != nil {
		t.Fatalf("ChangeWindowAttributes: %s", err)
	}

	err = DestroyWindowChecked(X, wid).Check()
	if err != nil {
		t.Fatalf("DestroyWindow: %s", err)
	}
}

// Calls GetFontPath function, Issue #12
func TestGetFontPath(t *testing.T) {
	fontPathReply, err := GetFontPath(X).Reply()
	if err != nil {
		t.Fatalf("GetFontPath: %v", err)
	}
	_ = fontPathReply
}

func TestListFonts(t *testing.T) {
	listFontsReply, err := ListFonts(X, 10, 1, "*").Reply()
	if err != nil {
		t.Fatalf("ListFonts: %v", err)
	}
	_ = listFontsReply
}

/******************************************************************************/
// Benchmarks
/******************************************************************************/

// BenchmarkInternAtomsGood shows how many requests with replies
// *should* be sent and gathered from the server. Namely, send as many
// requests as you can at once, then go back and gather up all the replies.
// More importantly, this approach can exploit parallelism when
// GOMAXPROCS > 1.
// Run with `go test -run 'nomatch' -bench '.*' -cpu 1,2,6` if you have
// multiple cores to see the improvement that parallelism brings.
func BenchmarkInternAtomsGood(b *testing.B) {
	b.StopTimer()
	names := seqNames(b.N)

	b.StartTimer()
	cookies := make([]InternAtomCookie, b.N)
	for i := 0; i < b.N; i++ {
		cookies[i] = InternAtom(X, false, uint16(len(names[i])), names[i])
	}
	for _, cookie := range cookies {
		cookie.Reply()
	}
}

// BenchmarkInternAtomsBad shows how *not* to issue a lot of requests with
// replies. Namely, each subsequent request isn't issued *until* the last
// reply is made. This implies a round trip to the X server for every
// iteration.
func BenchmarkInternAtomsPoor(b *testing.B) {
	b.StopTimer()
	names := seqNames(b.N)

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		InternAtom(X, false, uint16(len(names[i])), names[i]).Reply()
	}
}

/******************************************************************************/
// Helper functions
/******************************************************************************/

// changeAndGetProp sets property 'prop' with value 'val'.
// It then gets the value of that property and returns it.
// (It's used to check that the 'val' going in is the same 'val' going out.)
// It tests both requests with and without replies (GetProperty and
// ChangeProperty respectively.)
func changeAndGetProp(prop, val string) (string, error) {
	setup := Setup(X)
	root := setup.DefaultScreen(X).Root

	propAtom, err := InternAtom(X, false, uint16(len(prop)), prop).Reply()
	if err != nil {
		return "", fmt.Errorf("InternAtom: %s", err)
	}

	typName := "UTF8_STRING"
	typAtom, err := InternAtom(X, false, uint16(len(typName)), typName).Reply()
	if err != nil {
		return "", fmt.Errorf("InternAtom: %s", err)
	}

	err = ChangePropertyChecked(X, PropModeReplace, root, propAtom.Atom,
		typAtom.Atom, 8, uint32(len(val)), []byte(val)).Check()
	if err != nil {
		return "", fmt.Errorf("ChangeProperty: %s", err)
	}

	reply, err := GetProperty(X, false, root, propAtom.Atom,
		GetPropertyTypeAny, 0, (1<<32)-1).Reply()
	if err != nil {
		return "", fmt.Errorf("GetProperty: %s", err)
	}
	if reply.Format != 8 {
		return "", fmt.Errorf("Property reply format is %d but it should be 8.",
			reply.Format)
	}

	return string(reply.Value), nil
}

// verifyMapWindowError takes an error that is returned with an invalid
// MapWindow request with a window Id of 0 and makes sure the error is the
// right type and contains the correct values.
func verifyMapWindowError(t *testing.T, err error) {
	switch e := err.(type) {
	case WindowError:
		if e.BadValue != 0 {
			t.Fatalf("WindowError should report a bad value of 0 but "+
				"it reports %d instead.", e.BadValue)
		}
		if e.MajorOpcode != 8 {
			t.Fatalf("WindowError should report a major opcode of 8 "+
				"(which is a MapWindow request), but it reports %d instead.",
				e.MajorOpcode)
		}
	default:
		t.Fatalf("Expected a WindowError but got %T instead.", e)
	}
}

// randString generates a random string of length n.
func randString(n int) string {
	byts := make([]byte, n)
	for i := 0; i < n; i++ {
		rando := rand.Intn(53)
		switch {
		case rando <= 25:
			byts[i] = byte(65 + rando)
		case rando <= 51:
			byts[i] = byte(97 + rando - 26)
		default:
			byts[i] = ' '
		}
	}
	return string(byts)
}

// seqNames creates a slice of NAME0, NAME1, ..., NAMEN.
func seqNames(n int) []string {
	names := make([]string, n)
	for i := range names {
		names[i] = fmt.Sprintf("NAME%d", i)
	}
	return names
}

// evErr represents a value that is either an event or an error.
type evErr struct {
	ev  xgb.Event
	err xgb.Error
}

// channel used to pass evErrs.
var evOrErrChan = make(chan evErr, 0)

// grabEvents is a goroutine that reads events off the wire.
// We used this instead of WaitForEvent directly in our tests so that
// we can timeout and fail a test.
func grabEvents() {
	for {
		ev, err := X.WaitForEvent()
		evOrErrChan <- evErr{ev, err}
	}
}

// waitForEvent asks the evOrErrChan channel for an event.
// If it doesn't get an event in 'n' seconds, the current test is failed.
func waitForEvent(t *testing.T, n int) evErr {
	var evOrErr evErr

	select {
	case evOrErr = <-evOrErrChan:
	case <-time.After(time.Second * 5):
		t.Fatalf("After waiting 5 seconds for an event or an error, " +
			"we have timed out.")
	}

	return evOrErr
}
