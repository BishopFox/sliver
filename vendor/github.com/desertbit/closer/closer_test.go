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

package closer

import (
	"fmt"
	"sync"
	"testing"
	"time"

	multierror "github.com/hashicorp/go-multierror"
)

func TestCloser(t *testing.T) {
	c := New()
	if c.IsClosed() {
		t.Error()
	}
	select {
	case <-c.CloseChan():
		t.Error()
	default:
	}

	err := c.Close()
	if !c.IsClosed() {
		t.Error()
	} else if err != nil {
		t.Error()
	}
	select {
	case <-c.CloseChan():
	default:
		t.Error()
	}

	err = c.Close()
	if !c.IsClosed() {
		t.Error()
	} else if err != nil {
		t.Error()
	}
}

func TestCloserError(t *testing.T) {
	c := New(func() error {
		return fmt.Errorf("error")
	})

	err := c.Close()
	if merr := err.(*multierror.Error); err == nil || merr.Errors[0].Error() != "error" {
		t.Error(err)
	} else if len(err.Error()) == 0 { // Trigger th ErrorFormat function.
		t.Error(err)
	}
}

func TestCloserWithFunc(t *testing.T) {
	c := New(func() error {
		return fmt.Errorf("error")
	})
	if c.IsClosed() {
		t.Error()
	}

	for i := 0; i < 3; i++ {
		err := c.Close()
		if merr := err.(*multierror.Error); err == nil || merr.Errors[0].Error() != "error" {
			t.Error(err)
		}
	}
}

func TestCloserWithFuncs(t *testing.T) {
	c := New(func() error {
		return fmt.Errorf("error")
	})
	if c.IsClosed() {
		t.Error()
	}

	for i := 0; i < 3; i++ {
		c.OnClose(func() error {
			return fmt.Errorf("error")
		})
	}

	for i := 0; i < 3; i++ {
		err := c.Close()
		merr := err.(*multierror.Error)

		for i := 0; i < 4; i++ {
			if merr.Errors[i].Error() != "error" {
				t.Fatal(merr.Errors[i])
			}
		}
	}
}

func TestCloseFuncsLIFO(t *testing.T) {
	orderChan := make(chan int, 4)

	c := New(func() error {
		orderChan <- 0
		return nil
	})
	c.OnClose(func() error {
		orderChan <- 1
		return nil
	})
	c.OnClose(func() error {
		orderChan <- 2
		return nil
	})
	c.OnClose(func() error {
		orderChan <- 3
		return nil
	})

	err := c.Close()
	if err != nil {
		t.Fatal(err)
	}

	for i := 3; i >= 0; i-- {
		v := <-orderChan
		if i != v {
			t.Fatal(i, v)
		}
	}
}

func TestCloserWaitGroup(t *testing.T) {
	c := New()
	wg := c.CloseWaitGroup()
	doneChan := make(chan struct{}, 4)

	for i := 0; i < 4; i++ {
		c.OnClose(func() error {
			if len(doneChan) != 4 {
				return fmt.Errorf("error")
			}
			return nil
		})
	}

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(200 * time.Millisecond)
			doneChan <- struct{}{}
		}()
	}

	err := c.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentCloser(t *testing.T) {
	c := New()
	wg := sync.WaitGroup{}

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			c.Close()
			wg.Done()
		}()
	}

	wg.Wait()

	if !c.IsClosed() {
		t.Error()
	}

	select {
	case <-c.CloseChan():
	default:
		t.Error()
	}
}
