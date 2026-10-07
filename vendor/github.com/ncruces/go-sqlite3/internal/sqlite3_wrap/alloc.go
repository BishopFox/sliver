package sqlite3_wrap

import "github.com/ncruces/go-sqlite3/internal/errutil"

func (w *Wrapper) Free(ptr Ptr_t) {
	if ptr == 0 {
		return
	}
	w.Xsqlite3_free(int32(ptr))
}

func (w *Wrapper) New(size int64) Ptr_t {
	ptr := Ptr_t(w.Xsqlite3_malloc64(size))
	if ptr == 0 && size != 0 {
		panic(errutil.OOMErr)
	}
	return ptr
}

func (w *Wrapper) Realloc(ptr Ptr_t, size int64) Ptr_t {
	ptr = Ptr_t(w.Xsqlite3_realloc64(int32(ptr), size))
	if ptr == 0 && size != 0 {
		panic(errutil.OOMErr)
	}
	return ptr
}

func (w *Wrapper) NewBytes(b []byte) Ptr_t {
	if len(b) == 0 {
		return 0
	}
	ptr := w.New(int64(len(b)))
	w.WriteBytes(ptr, b)
	return ptr
}

func (w *Wrapper) NewString(s string) Ptr_t {
	ptr := w.New(int64(len(s)) + 1)
	w.WriteString(ptr, s)
	return ptr
}

func (w *Wrapper) StackMark() (reset func()) {
	ptr := w.X__stack_pointer()
	old := *ptr
	return func() { *ptr = old }
}

func (w *Wrapper) StackAlloc(size int64) Ptr_t {
	ptr := w.X__stack_pointer()
	new := int64(*ptr) - size
	if size < 0 || new <= 0 {
		panic(errutil.OOMErr)
	}
	*ptr = int32(new)
	return Ptr_t(new)
}

const arenaSize = 4096

func (w *Wrapper) NewArena() Arena {
	return Arena{
		sqlt: w,
		base: w.New(arenaSize),
	}
}

type Arena struct {
	sqlt *Wrapper
	ptrs []Ptr_t
	base Ptr_t
	next int32
}

func (a *Arena) Free() {
	if a.sqlt == nil {
		return
	}
	for _, ptr := range a.ptrs {
		a.sqlt.Free(ptr)
	}
	a.sqlt.Free(a.base)
	a.sqlt = nil
}

func (a *Arena) Mark() (reset func()) {
	ptrs := len(a.ptrs)
	next := a.next
	return func() {
		for _, ptr := range a.ptrs[ptrs:] {
			a.sqlt.Free(ptr)
		}
		a.ptrs = a.ptrs[:ptrs]
		a.next = next
	}
}

func (a *Arena) New(size int64) Ptr_t {
	// Align the next address, to 4 or 8 bytes.
	if size&7 != 0 {
		a.next = (a.next + 3) &^ 3
	} else {
		a.next = (a.next + 7) &^ 7
	}
	if size <= arenaSize-int64(a.next) {
		ptr := a.base + Ptr_t(a.next)
		a.next += int32(size)
		return Ptr_t(ptr)
	}
	ptr := a.sqlt.New(size)
	a.ptrs = append(a.ptrs, ptr)
	return Ptr_t(ptr)
}

func (a *Arena) Bytes(b []byte) Ptr_t {
	if len(b) == 0 {
		return 0
	}
	ptr := a.New(int64(len(b)))
	a.sqlt.WriteBytes(ptr, b)
	return ptr
}

func (a *Arena) String(s string) Ptr_t {
	ptr := a.New(int64(len(s)) + 1)
	a.sqlt.WriteString(ptr, s)
	return ptr
}
