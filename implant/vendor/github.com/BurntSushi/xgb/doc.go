/*
Package XGB provides the X Go Binding, which is a low-level API to communicate
with the core X protocol and many of the X extensions.

It is *very* closely modeled on XCB, so that experience with XCB (or xpyb) is
easily translatable to XGB. That is, it uses the same cookie/reply model
and is thread safe. There are otherwise no major differences (in the API).

Most uses of XGB typically fall under the realm of window manager and GUI kit
development, but other applications (like pagers, panels, tilers, etc.) may
also require XGB. Moreover, it is a near certainty that if you need to work
with X, xgbutil will be of great use to you as well:
https://github.com/BurntSushi/xgbutil

Example

This is an extremely terse example that demonstrates how to connect to X,
create a window, listen to StructureNotify events and Key{Press,Release}
events, map the window, and print out all events received. An example with
accompanying documentation can be found in examples/create-window.

	package main

	import (
		"fmt"
		"github.com/BurntSushi/xgb"
		"github.com/BurntSushi/xgb/xproto"
	)

	func main() {
		X, err := xgb.NewConn()
		if err != nil {
			fmt.Println(err)
			return
		}

		wid, _ := xproto.NewWindowId(X)
		screen := xproto.Setup(X).DefaultScreen(X)
		xproto.CreateWindow(X, screen.RootDepth, wid, screen.Root,
			0, 0, 500, 500, 0,
			xproto.WindowClassInputOutput, screen.RootVisual,
			xproto.CwBackPixel | xproto.CwEventMask,
			[]uint32{ // values must be in the order defined by the protocol
				0xffffffff,
				xproto.EventMaskStructureNotify |
				xproto.EventMaskKeyPress |
				xproto.EventMaskKeyRelease})

		xproto.MapWindow(X, wid)
		for {
			ev, xerr := X.WaitForEvent()
			if ev == nil && xerr == nil {
				fmt.Println("Both event and error are nil. Exiting...")
				return
			}

			if ev != nil {
				fmt.Printf("Event: %s\n", ev)
			}
			if xerr != nil {
				fmt.Printf("Error: %s\n", xerr)
			}
		}
	}

Xinerama Example

This is another small example that shows how to query Xinerama for geometry
information of each active head. Accompanying documentation for this example
can be found in examples/xinerama.

	package main

	import (
		"fmt"
		"log"
		"github.com/BurntSushi/xgb"
		"github.com/BurntSushi/xgb/xinerama"
	)

	func main() {
		X, err := xgb.NewConn()
		if err != nil {
			log.Fatal(err)
		}

		// Initialize the Xinerama extension.
		// The appropriate 'Init' function must be run for *every*
		// extension before any of its requests can be used.
		err = xinerama.Init(X)
		if err != nil {
			log.Fatal(err)
		}

		reply, err := xinerama.QueryScreens(X).Reply()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Number of heads: %d\n", reply.Number)
		for i, screen := range reply.ScreenInfo {
			fmt.Printf("%d :: X: %d, Y: %d, Width: %d, Height: %d\n",
				i, screen.XOrg, screen.YOrg, screen.Width, screen.Height)
		}
	}

Parallelism

XGB can benefit greatly from parallelism due to its concurrent design. For
evidence of this claim, please see the benchmarks in xproto/xproto_test.go.

Tests

xproto/xproto_test.go contains a number of contrived tests that stress
particular corners of XGB that I presume could be problem areas. Namely:
requests with no replies, requests with replies, checked errors, unchecked
errors, sequence number wrapping, cookie buffer flushing (i.e., forcing a round
trip every N requests made that don't have a reply), getting/setting properties
and creating a window and listening to StructureNotify events.

Code Generator

Both XCB and xpyb use the same Python module (xcbgen) for a code generator. XGB
(before this fork) used the same code generator as well, but in my attempt to
add support for more extensions, I found the code generator extremely difficult
to work with. Therefore, I re-wrote the code generator in Go. It can be found
in its own sub-package, xgbgen, of xgb. My design of xgbgen includes a rough
consideration that it could be used for other languages.

What works

I am reasonably confident that the core X protocol is in full working form. I've
also tested the Xinerama and RandR extensions sparingly. Many of the other
existing extensions have Go source generated (and are compilable) and are
included in this package, but I am currently unsure of their status. They
*should* work.

What does not work

XKB is the only extension that intentionally does not work, although I suspect
that GLX also does not work (however, there is Go source code for GLX that
compiles, unlike XKB). I don't currently have any intention of getting XKB
working, due to its complexity and my current mental incapacity to test it.

*/
package xgb
