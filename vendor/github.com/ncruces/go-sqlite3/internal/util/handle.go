package util

import (
	"context"
	"io"
)

type handleKey struct{}
type handleState struct {
	handles []any
	empty   int
}

func NewContext(ctx context.Context) (context.Context, io.Closer) {
	state := new(handleState)
	return context.WithValue(ctx, handleKey{}, state), state
}

func (s *handleState) Close() (err error) {
	for _, h := range s.handles {
		if c, ok := h.(io.Closer); ok {
			if e := c.Close(); err == nil {
				err = e
			}
		}
	}
	s.handles = nil
	s.empty = 0
	return err
}

func GetHandle(ctx context.Context, id uint32) any {
	if id == 0 {
		return nil
	}
	s := ctx.Value(handleKey{}).(*handleState)
	return s.handles[^id]
}

func DelHandle(ctx context.Context, id uint32) error {
	if id == 0 {
		return nil
	}
	s := ctx.Value(handleKey{}).(*handleState)
	a := s.handles[^id]
	s.handles[^id] = nil
	s.empty++
	if c, ok := a.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func AddHandle(ctx context.Context, a any) (id uint32) {
	if a == nil {
		panic(NilErr)
	}
	s := ctx.Value(handleKey{}).(*handleState)

	// Find an empty slot.
	if s.empty > cap(s.handles)-len(s.handles) {
		for id, h := range s.handles {
			if h == nil {
				s.empty--
				s.handles[id] = a
				return ^uint32(id)
			}
		}
	}

	// Add a new slot.
	s.handles = append(s.handles, a)
	return -uint32(len(s.handles))
}
