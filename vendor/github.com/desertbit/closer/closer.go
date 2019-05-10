/*
 *  Closer - A simple thread-safe closer
 *  Copyright (C) 2016  Roland Singer <roland.singer[at]desertbit.com>
 *
 *  This program is free software: you can redistribute it and/or modify
 *  it under the terms of the GNU General Public License as published by
 *  the Free Software Foundation, either version 3 of the License, or
 *  (at your option) any later version.
 *
 *  This program is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *  GNU General Public License for more details.
 *
 *  You should have received a copy of the GNU General Public License
 *  along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

// Package closer offers a simple thread-safe closer.
package closer

import (
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
	// Close in a thread-safe manner. implements the io.Closer interface.
	// This method returns always the close error, regardless of how often
	// this method is called. Close blocks until all close functions are done,
	// no matter which goroutine called this method.
	// Returns a hashicorp multierror.
	Close() error

	// CloseChan channel which is closed as
	// soon as the closer is closed.
	CloseChan() <-chan struct{}

	// CloseWaitGroup returns the wait group for this closer.
	// Use this wait group to wait before calling the close functions.
	CloseWaitGroup() *sync.WaitGroup

	// IsClosed returns a boolean indicating
	// if this instance was closed.
	IsClosed() bool

	// Calls the close function on close.
	// Errors are appended to the Close() multi error.
	// Close functions are called in LIFO order.
	OnClose(f CloseFunc)
}

//######################//
//### Implementation ###//
//######################//

type closer struct {
	mutex     sync.Mutex
	wg        *sync.WaitGroup
	closeChan chan struct{}
	closeErr  error
	funcs     []CloseFunc
}

// New creates a new closer.
// Optional pass functions which are called only once during close.
// Close function are called in LIFO order.
func New(f ...CloseFunc) Closer {
	return &closer{
		closeChan: make(chan struct{}),
		funcs:     f,
	}
}

func (c *closer) OnClose(f CloseFunc) {
	c.mutex.Lock()
	c.funcs = append(c.funcs, f)
	c.mutex.Unlock()
}

func (c *closer) IsClosed() bool {
	select {
	case <-c.closeChan:
		return true
	default:
		return false
	}
}

func (c *closer) CloseChan() <-chan struct{} {
	return c.closeChan
}

func (c *closer) CloseWaitGroup() *sync.WaitGroup {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.wg == nil {
		c.wg = &sync.WaitGroup{}
	}

	return c.wg
}

func (c *closer) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Close the close channel if not already closed.
	if c.IsClosed() {
		return c.closeErr
	}
	close(c.closeChan)

	// Block if the wait group is defined.
	if c.wg != nil {
		c.wg.Wait()
	}

	var mErr *multierror.Error

	// Call in LIFO order. Append the errors.
	for i := len(c.funcs) - 1; i >= 0; i-- {
		if err := c.funcs[i](); err != nil {
			mErr = multierror.Append(mErr, err)
		}
	}
	c.funcs = nil

	if mErr != nil {
		// The default multi error formatting uses too much space.
		mErr.ErrorFormat = func(errors []error) string {
			str := fmt.Sprintf("%v close errors occurred:", len(errors))
			for _, err := range errors {
				str += "\n- " + err.Error()
			}
			return str
		}
		c.closeErr = mErr
	}

	return c.closeErr
}
