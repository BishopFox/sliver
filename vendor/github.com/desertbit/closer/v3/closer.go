/*
 * closer - A simple, thread-safe closer
 *
 * The MIT License (MIT)
 *
 * Copyright (c) 2019 Roland Singer <roland.singer[at]desertbit.com>
 * Copyright (c) 2019 Sebastian Borchers <sebastian[at]desertbit.com>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

// Package closer offers a simple, thread-safe closer.
//
// It allows to build up a tree of closing relationships, where you typically
// start with a root closer that branches into different children and
// children's children. When a parent closer spawns a child closer, the child
// either has a one-way or two-way connection to its parent. One-way children
// are closed when their parent closes. In addition, two-way children also close
// their parent, if they are closed themselves.
//
// A closer is also useful to ensure that certain dependencies, such as network
// connections, are reliably taken down, once the closer closes.
// In addition, a closer can be concurrently closed many times, without closing
// more than once, but still returning the errors to every caller.
//
// This allows to represent complex closing relationships and helps avoiding
// leaking goroutines, gracefully shutting down, etc.
package closer

import (
	"context"
	"fmt"
	"sync"

	multierror "github.com/hashicorp/go-multierror"
)

//#############//
//### Types ###//
//#############//

// CloseFunc defines the general close function.
type CloseFunc func() error

//#################//
//### Interface ###//
//#################//

// A Closer is a thread-safe helper for common close actions.
type Closer interface {
	// Close closes this closer in a thread-safe manner.
	//
	// Implements the io.Closer interface.
	//
	// This method always returns the close error,
	// regardless of how often it gets called.
	//
	// The closing order looks like this:
	// 1: the closing chan is closed.
	// 2: the OnClosing funcs are executed.
	// 3: each of the closer's children is closed.
	// 4: it waits for the wait group.
	// 5: the OnClose funcs are executed.
	// 6: the closed chan is closed.
	// 7: the parent is closed, if it has one.
	//
	// Close blocks, until all steps of the closing order
	// have been done.
	// No matter which goroutine called this method.
	// Returns a hashicorp multierror.
	Close() error

	// Close_ is a convenience version of Close(), for use in defer
	// where the error is not of interest.
	Close_()

	// CloseAndDone performs the same operation as Close(), but decrements
	// the closer's wait group by one beforehand.
	// Attention: Calling this without first adding to the WaitGroup by
	// calling AddWaitGroup() results in a panic.
	CloseAndDone() error

	// CloseAndDone_ is a convenience version of CloseAndDone(), for use in
	// defer where the error is not of interest.
	CloseAndDone_()

	// ClosedChan returns a channel, which is closed as
	// soon as the closer is completely closed.
	// See Close() for the position in the closing order.
	ClosedChan() <-chan struct{}

	// CloserAddWait adds the given delta to the closer's
	// wait group. Useful to wait for routines associated
	// with this closer to gracefully shutdown.
	// See Close() for the position in the closing order.
	CloserAddWait(delta int)

	// CloserDone decrements the closer's wait group by one.
	// Attention: Calling this without first adding to the WaitGroup by
	// calling AddWaitGroup() results in a panic.
	CloserDone()

	// CloserOneWay creates a new child closer that has a one-way relationship
	// with the current closer. This means that the child is closed whenever
	// the parent closes, but not vice versa.
	// See Close() for the position in the closing order.
	CloserOneWay() Closer

	// CloserTwoWay creates a new child closer that has a two-way relationship
	// with the current closer. This means that the child is closed whenever
	// the parent closes and vice versa.
	// See Close() for the position in the closing order.
	CloserTwoWay() Closer

	// ClosingChan returns a channel, which is closed as
	// soon as the closer is about to close.
	// Remains closed, once ClosedChan() has also been closed.
	// See Close() for the position in the closing order.
	ClosingChan() <-chan struct{}

	// Context returns a context.Context, which is cancelled
	// as soon as the closer is closing.
	// The returned cancel func should be called as soon as the
	// context is no longer needed, to free resources.
	Context() (context.Context, context.CancelFunc)

	// IsClosed returns a boolean indicating
	// whether this instance has been closed completely.
	IsClosed() bool

	// IsClosing returns a boolean indicating
	// whether this instance is about to close.
	// Also returns true, if IsClosed() returns true.
	IsClosing() bool

	// OnClose adds the given CloseFuncs to the closer.
	// Their errors are appended to the Close() multi error.
	// Close functions are called in LIFO order.
	// See Close() for their position in the closing order.
	OnClose(f ...CloseFunc)

	// OnClosing adds the given CloseFuncs to the closer.
	// Their errors are appended to the Close() multi error.
	// Closing functions are called in LIFO order.
	// It is guaranteed that all closing funcs are executed before
	// any close funcs.
	// See Close() for their position in the closing order.
	OnClosing(f ...CloseFunc)
}

//######################//
//### Implementation ###//
//######################//

const (
	minChildrenCap = 100
)

// The closer type is this package's implementation of the Closer interface.
type closer struct {
	// An unbuffered channel that expresses whether the
	// closer is about to close.
	// The channel itself gets closed to represent the closing
	// of the closer, which leads to reads off of it to succeed.
	closingChan chan struct{}
	// An unbuffered channel that expresses whether the
	// closer has been completely closed.
	// The channel itself gets closed to represent the closing
	// of the closer, which leads to reads off of it to succeed.
	closedChan chan struct{}
	// The error collected by executing the Close() func
	// and combining all encountered errors from the close funcs.
	closeErr error

	// Synchronises the access to the following properties.
	mx sync.Mutex
	// The close funcs that are executed when this closer closes.
	closeFuncs []CloseFunc
	// The closing funcs that are executed when this closer closes.
	closingFuncs []CloseFunc
	// The parent of this closer. May be nil.
	parent *closer
	// The closer children that this closer spawned.
	children []*closer
	// Used to wait for external dependencies of the closer
	// before the Close() method actually returns.
	wg sync.WaitGroup

	// A flag that indicates whether this closer is a two-way closer.
	// In comparison to a standard one-way closer, which closes when
	// its parent closes, a two-way closer closes also its parent, when
	// it itself gets closed.
	twoWay bool

	// The index of this closer in its parent's children slice.
	// Needed to efficiently remove the closer from its parent.
	parentIndex int
}

// New creates a new closer.
func New() Closer {
	return newCloser()
}

// Implements the Closer interface.
func (c *closer) Close() error {
	// Mutex is not unlocked on defer! Therefore, be cautious when adding
	// new control flow statements like return.
	c.mx.Lock()

	// If the closer is already closing, just return the error.
	if c.IsClosing() {
		c.mx.Unlock()
		return c.closeErr
	}

	// Close the closing channel to signal that this closer is about to close now.
	close(c.closingChan)

	// Execute all closing funcs of this closer.
	c.closeErr = c.execCloseFuncs(c.closingFuncs)
	// Delete them, to free resources.
	c.closingFuncs = nil

	// Close all children.
	for _, child := range c.children {
		child.Close_()
	}

	// Wait, until all dependencies of this closer have closed.
	c.wg.Wait()

	// Execute all close funcs of this closer.
	c.closeErr = c.execCloseFuncs(c.closeFuncs)
	// Delete them, to free resources.
	c.closeFuncs = nil

	// Close the closed channel to signal that this closer is closed now.
	close(c.closedChan)

	c.mx.Unlock()

	// Close the parent now as well, if this is a two way closer.
	// Otherwise, the closer must remove its reference from its parent's children
	// to prevent a leak.
	// Only perform these actions, if the parent is not closing already!
	if c.parent != nil && !c.parent.IsClosing() {
		if c.twoWay {
			c.parent.Close_()
		} else {
			c.parent.removeChild(c)
		}
	}

	return c.closeErr
}

// Implements the Closer interface.
func (c *closer) Close_() {
	_ = c.Close()
}

// Implements the Closer interface.
func (c *closer) CloseAndDone() error {
	c.wg.Done()
	return c.Close()
}

// Implements the Closer interface.
func (c *closer) CloseAndDone_() {
	_ = c.CloseAndDone()
}

// Implements the Closer interface.
func (c *closer) ClosedChan() <-chan struct{} {
	return c.closedChan
}

// Implements the Closer interface.
func (c *closer) CloserAddWait(delta int) {
	c.wg.Add(delta)
}

// Implements the Closer interface.
func (c *closer) CloserDone() {
	c.wg.Done()
}

// Implements the Closer interface.
func (c *closer) CloserOneWay() Closer {
	return c.addChild(false)
}

// Implements the Closer interface.
func (c *closer) CloserTwoWay() Closer {
	return c.addChild(true)
}

// Implements the Closer interface.
func (c *closer) ClosingChan() <-chan struct{} {
	return c.closingChan
}

// Implements the Closer interface.
func (c *closer) Context() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		select {
		case <-c.closingChan:
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx, cancel
}

// Implements the Closer interface.
func (c *closer) IsClosed() bool {
	select {
	case <-c.closedChan:
		return true
	default:
		return false
	}
}

// Implements the Closer interface.
func (c *closer) IsClosing() bool {
	select {
	case <-c.closingChan:
		return true
	default:
		return false
	}
}

// Implements the Closer interface.
func (c *closer) OnClose(f ...CloseFunc) {
	c.mx.Lock()
	c.closeFuncs = append(c.closeFuncs, f...)
	c.mx.Unlock()
}

// Implements the Closer interface.
func (c *closer) OnClosing(f ...CloseFunc) {
	c.mx.Lock()
	c.closingFuncs = append(c.closingFuncs, f...)
	c.mx.Unlock()
}

//###############//
//### Private ###//
//###############//

// newCloser creates a new closer with the given close funcs.
func newCloser() *closer {
	return &closer{
		closingChan: make(chan struct{}),
		closedChan:  make(chan struct{}),
	}
}

// addChild creates a new closer and adds it as either
// a one-way or two-way child to this closer.
func (c *closer) addChild(twoWay bool) *closer {
	// Create a new closer and set the current closer as its parent.
	// Also set the twoWay flag.
	child := newCloser()
	child.parent = c
	child.twoWay = twoWay

	// Add the closer to the current closer's children.
	c.mx.Lock()
	child.parentIndex = len(c.children)
	c.children = append(c.children, child)
	c.mx.Unlock()

	return child
}

// removeChild removes the given child from this closer's children.
// If the child can not be found, this is a no-op.
func (c *closer) removeChild(child *closer) {
	c.mx.Lock()
	defer c.mx.Unlock()

	last := len(c.children) - 1
	c.children[last].parentIndex = child.parentIndex
	c.children[child.parentIndex] = c.children[last]
	c.children[last] = nil
	c.children = c.children[:last]

	// Prevent endless growth.
	// If the capacity is bigger than our min value and
	// four times larger than the length, shrink it by half.
	cp := cap(c.children)
	le := len(c.children)
	if cp > minChildrenCap && cp > 4*le {
		children := make([]*closer, le, le*2)
		copy(children, c.children)
		c.children = children
	}
}

// execCloseFuncs executes the given close funcs and appends them
// to the closer's closeErr, which is a hashicorp.multiError.
// The error is then returned.
func (c *closer) execCloseFuncs(f []CloseFunc) error {
	// Batch errors together.
	var mErr *multierror.Error

	// If an error is already set, append the next errors to it.
	if c.closeErr != nil {
		mErr = multierror.Append(mErr, c.closeErr)
	}

	// Call in LIFO order. Append the errors.
	for i := len(f) - 1; i >= 0; i-- {
		if err := f[i](); err != nil {
			mErr = multierror.Append(mErr, err)
		}
	}

	// If no error is available, return.
	if mErr == nil {
		return nil
	}

	// The default multiCloser error formatting uses too much space.
	mErr.ErrorFormat = func(errors []error) string {
		str := fmt.Sprintf("%v close errors occurred:", len(errors))
		for _, err := range errors {
			str += "\n- " + err.Error()
		}
		return str
	}

	return mErr
}
