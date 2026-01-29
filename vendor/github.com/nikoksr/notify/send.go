package notify

import (
	"context"
	"errors"

	"golang.org/x/sync/errgroup"
)

// send calls the underlying notification services to send the given subject and message to their respective endpoints.
func (n *Notify) send(ctx context.Context, subject, message string) error {
	if n.Disabled {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	var eg errgroup.Group
	for _, service := range n.notifiers {
		if service == nil {
			continue
		}

		eg.Go(func() error {
			return service.Send(ctx, subject, message)
		})
	}

	err := eg.Wait()
	if err != nil {
		err = errors.Join(ErrSendNotification, err)
	}

	return err
}

// Send calls the underlying notification services to send the given subject and message to their respective endpoints.
func (n *Notify) Send(ctx context.Context, subject, message string) error {
	return n.send(ctx, subject, message)
}

// Send calls the underlying notification services to send the given subject and message to their respective endpoints.
func Send(ctx context.Context, subject, message string) error {
	return std.Send(ctx, subject, message)
}
