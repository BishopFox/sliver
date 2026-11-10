package sqlite3_wrap

import (
	"io"

	sqlite3_wasm "github.com/ncruces/go-sqlite3-wasm/v6"
	"github.com/ncruces/go-sqlite3/internal/errutil"
)

type Wrapper struct {
	*sqlite3_wasm.Module
	*Memory
	DB       any
	SysError error

	handles []any
	deleted int
}

func (w *Wrapper) Close() (err error) {
	for _, h := range w.handles {
		if c, ok := h.(io.Closer); ok {
			if e := c.Close(); err == nil {
				err = e
			}
		}
	}
	if e := w.Memory.Close(); e != nil {
		err = e
	}
	*w = Wrapper{}
	return err
}

func (w *Wrapper) GetHandle(id Ptr_t) any {
	if id == 0 {
		return nil
	}
	return w.handles[^id]
}

func (w *Wrapper) DelHandle(id Ptr_t) error {
	if id == 0 {
		return nil
	}
	a := w.handles[^id]
	w.handles[^id] = nil
	if l := Ptr_t(len(w.handles)); l == ^id {
		w.handles = w.handles[:l-1]
	} else {
		w.deleted++
	}
	if c, ok := a.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func (w *Wrapper) AddHandle(a any) Ptr_t {
	if a == nil {
		panic(errutil.NilErr)
	}

	// Find an empty slot.
	if w.deleted > cap(w.handles)-len(w.handles) {
		for id, h := range w.handles {
			if h == nil {
				w.deleted--
				w.handles[id] = a
				return ^Ptr_t(id)
			}
		}
	}

	// Add a new slot.
	w.handles = append(w.handles, a)
	return -Ptr_t(len(w.handles))
}

func (w *Wrapper) Xmemory() sqlite3_wasm.Memory { return w.Memory }

func (w *Wrapper) Xgo_destroy(pApp int32) { w.DelHandle(Ptr_t(pApp)) }
