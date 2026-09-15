//go:build !windows

package opfor

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

// duplicatedSleepSocketAcceptor preserves the Unix implementation's
// descriptor-per-waiter deadlines. Closing or timing out one cached accept
// therefore cannot disturb another waiter on the same listening socket.
type duplicatedSleepSocketAcceptor struct {
	listener *net.TCPListener

	mu        sync.Mutex
	closed    bool
	acceptors map[*net.TCPListener]struct{}
}

func newSleepSocketAcceptor(listener *net.TCPListener) sleepSocketAcceptor {
	return &duplicatedSleepSocketAcceptor{
		listener:  listener,
		acceptors: make(map[*net.TCPListener]struct{}),
	}
}

func (listener *duplicatedSleepSocketAcceptor) accept(ctx context.Context, timeout int32) (net.Conn, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	acceptor, err := listener.newAcceptor()
	if err != nil {
		return nil, err
	}
	defer listener.releaseAcceptor(acceptor)

	deadline := time.Time{}
	if timeout > 0 {
		deadline = time.Now().Add(time.Duration(timeout) * time.Millisecond)
	}
	if err := acceptor.SetDeadline(deadline); err != nil {
		return nil, err
	}

	// File and FileListener give every waiter a duplicate descriptor and its
	// own deadline, avoiding cross-cancellation when an owning script unloads.
	stop := make(chan struct{})
	wakeDone := make(chan struct{})
	go func() {
		defer close(wakeDone)
		select {
		case <-ctx.Done():
			_ = acceptor.SetDeadline(time.Now())
		case <-stop:
		}
	}()
	conn, err := acceptor.AcceptTCP()
	close(stop)
	<-wakeDone
	if contextErr := ctx.Err(); contextErr != nil {
		if conn != nil {
			_ = conn.Close()
		}
		return nil, contextErr
	}
	return conn, err
}

func (listener *duplicatedSleepSocketAcceptor) newAcceptor() (*net.TCPListener, error) {
	file, err := listener.listener.File()
	if err != nil {
		return nil, err
	}
	duplicate, duplicateErr := net.FileListener(file)
	fileErr := file.Close()
	if duplicateErr != nil {
		return nil, errors.Join(duplicateErr, fileErr)
	}
	acceptor, ok := duplicate.(*net.TCPListener)
	if !ok {
		_ = duplicate.Close()
		return nil, fmt.Errorf("socket listener duplicate has type %T", duplicate)
	}
	if fileErr != nil {
		_ = acceptor.Close()
		return nil, fileErr
	}

	listener.mu.Lock()
	if listener.closed {
		listener.mu.Unlock()
		_ = acceptor.Close()
		return nil, net.ErrClosed
	}
	listener.acceptors[acceptor] = struct{}{}
	listener.mu.Unlock()
	return acceptor, nil
}

func (listener *duplicatedSleepSocketAcceptor) releaseAcceptor(acceptor *net.TCPListener) {
	if listener == nil || acceptor == nil {
		return
	}
	listener.mu.Lock()
	delete(listener.acceptors, acceptor)
	listener.mu.Unlock()
	_ = acceptor.Close()
}

func (listener *duplicatedSleepSocketAcceptor) close() error {
	if listener == nil {
		return nil
	}
	listener.mu.Lock()
	if listener.closed {
		listener.mu.Unlock()
		return nil
	}
	listener.closed = true
	acceptors := make([]*net.TCPListener, 0, len(listener.acceptors))
	for acceptor := range listener.acceptors {
		acceptors = append(acceptors, acceptor)
	}
	listener.acceptors = make(map[*net.TCPListener]struct{})
	listener.mu.Unlock()

	result := listener.listener.Close()
	for _, acceptor := range acceptors {
		result = errors.Join(result, acceptor.Close())
	}
	return result
}
