package wazevoapi

const poolPageSize = 128

// Pool is a pool of T that can be allocated and reset.
// This is useful to avoid unnecessary allocations.
type Pool[T any] struct {
	pages            []*[poolPageSize]T
	resetFn          func(*T)
	allocated, index int
}

// NewPool returns a new Pool.
// resetFn is called when a new T is allocated in Pool.Allocate.
func NewPool[T any](resetFn func(*T)) Pool[T] {
	var ret Pool[T]
	ret.resetFn = resetFn
	ret.Reset()
	return ret
}

// Allocated returns the number of allocated T currently in the pool.
func (p *Pool[T]) Allocated() int {
	return p.allocated
}

// Allocate allocates a new T from the pool.
func (p *Pool[T]) Allocate() *T {
	if p.index == poolPageSize {
		if len(p.pages) == cap(p.pages) {
			p.pages = append(p.pages, new([poolPageSize]T))
		} else {
			i := len(p.pages)
			p.pages = p.pages[:i+1]
			if p.pages[i] == nil {
				p.pages[i] = new([poolPageSize]T)
			}
		}
		p.index = 0
	}
	ret := &p.pages[len(p.pages)-1][p.index]
	p.resetFn(ret)
	p.index++
	p.allocated++
	return ret
}

// View returns the pointer to i-th item from the pool.
func (p *Pool[T]) View(i int) *T {
	page, index := i/poolPageSize, i%poolPageSize
	return &p.pages[page][index]
}

// Reset resets the pool.
func (p *Pool[T]) Reset() {
	p.pages = p.pages[:0]
	p.index = poolPageSize
	p.allocated = 0
}
