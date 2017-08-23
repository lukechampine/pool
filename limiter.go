package pool

import (
	"sync/atomic"

	"sync"
)

// A Limiter limits access to a resource.
type Limiter struct {
	inuse int64
	limit int64
	cond  *sync.Cond
}

// Get blocks until n units are available, and then claims them. n must be
// non-negative and less than the Limiter's limit.
func (l *Limiter) Get(n int) {
	if n < 0 {
		panic("cannot Get a negative number")
	} else if int64(n) > l.limit {
		panic("cannot Get more than the limit")
	}
	for {
		inuse := atomic.LoadInt64(&l.inuse)
		new := inuse + int64(n)
		if new <= l.limit && atomic.CompareAndSwapInt64(&l.inuse, inuse, new) {
			return
		}
		l.cond.Wait()
	}
}

// Put returns n units to the limiter. n must be non-negative and must not
// exceed the total number of units currently in use.
func (l *Limiter) Put(n int) {
	if n < 0 {
		panic("cannot Put a negative number")
	}
	if atomic.AddInt64(&l.inuse, int64(-n)) < 0 {
		panic("inuse cannot be negative")
	}
	l.cond.Broadcast()
}

// NewLimiter returns a Limiter with the supplied limit, which must be non-
// negative.
func NewLimiter(limit int64) *Limiter {
	if limit < 0 {
		panic("limit must be non-negative")
	}
	return &Limiter{
		limit: limit,
		cond:  sync.NewCond(noopLocker{}),
	}
}

// A MemLimiter limits access to memory allocations.
type MemLimiter struct {
	l Limiter
}

// Get blocks until n bytes are available, then allocates and returns a []byte
// with length and capacity n. If n is greater than the limit, Get panics.
func (m *MemLimiter) Get(n int) []byte {
	m.l.Get(n)
	return make([]byte, n)
}

// Put returns len(b) bytes to the limiter.
func (m *MemLimiter) Put(b []byte) {
	m.l.Put(len(b))
}

// NewMemLimiter returns a MemLimiter that allows up to limit bytes to be
// allocated at any given time.
func NewMemLimiter(limit int64) *MemLimiter {
	return &MemLimiter{l: *NewLimiter(limit)}
}
