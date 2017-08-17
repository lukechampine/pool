package mem

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"unsafe"
)

// An IndexPool is a pool the holds a set of indices in the range [0, n).
// These indices are not very useful on their own; typically the IndexPool is
// used within another struct to create a type-specific resource pool.
//
// IndexPools are safe for concurrent use.
type IndexPool struct {
	indices []int32
	cond    *sync.Cond
}

// Get returns an index from the pool, blocking if necessary until one becomes
// available.
func (p *IndexPool) Get() int {
	for {
		// search for an available index
		for i := range p.indices {
			// try to mark the index as unavailable
			if atomic.CompareAndSwapInt32(&p.indices[i], 0, 1) {
				return i
			}
		}
		// no indices are available, so block until woken up by a call to Put
		p.cond.Wait()
	}
}

// Put returns index i to the pool. Put panics if i was already returned to
// the pool, or if i is larger than the number of indices in the pool.
func (p *IndexPool) Put(i int) {
	if i < 0 || i >= len(p.indices) {
		panic(fmt.Sprintf("index %v does not belong to the pool [0,%v)", i, len(p.indices)))
	} else if atomic.LoadInt32(&p.indices[i]) == 0 {
		panic(fmt.Sprintf("index %v was already returned to pool", i))
	} else if i > 0 && i < len(p.indices) {
		// mark the index as available
		atomic.StoreInt32(&p.indices[i], 0)
	}
	// if there are blocked Get calls, wake one up
	p.cond.Signal()
}

// NewIndexPool creates a new IndexPool that contains indices in the range
// [0,n). n must be positive.
func NewIndexPool(n int) *IndexPool {
	if n <= 0 {
		panic("cannot create empty IndexPool")
	}
	return &IndexPool{
		indices: make([]int32, n),
		cond:    sync.NewCond(noopLocker{}),
	}
}

// A BufferPool is a pool of fixed-size []byte buffers. BufferPools are safe
// for concurrent use.
type BufferPool struct {
	// NoClear controls whether buffers are cleared before being returned to
	// the pool. Enabling NoClear improves performance, but requires the
	// caller to deal with potentially dirty buffers.
	NoClear bool

	bufs    [][]byte
	indices *IndexPool
}

// Get returns one of the buffers in the pool, blocking if necessary until one
// becomes available.
func (p BufferPool) Get() []byte {
	return p.bufs[p.indices.Get()]
}

// Put returns a buffer to the pool. b must be a buffer that was returned by
// Get; otherwise, Put panics. However, buffer may be modified or resliced.
// All that matters is that b point to the same memory location as the
// original slice returned by Get. For example, this is legal:
//
//    b := pool.Get()
//    b = append(b[:0], 1) // b's data pointer is unchanged
//    pool.Put(b)
//
// But this is not:
//
//    b := pool.Get()
//    b = append(b, 1) // causes b to be reallocated
//    pool.Put(b)
//
// Callers must not modify the contents of a buffer after returning it to the
// pool with Put.
func (p BufferPool) Put(b []byte) {
	// look for the buffer whose pointer matches b
	for i := range p.bufs {
		// NOTE: this must be a single expression; otherwise the GC can
		// relocate the Data pointers
		if (*reflect.SliceHeader)(unsafe.Pointer(&p.bufs[i])).Data == (*reflect.SliceHeader)(unsafe.Pointer(&b)).Data {
			if !p.NoClear {
				for j := range p.bufs[i] {
					p.bufs[i][j] = 0
				}
			}
			p.indices.Put(i)
			return
		}
	}
	panic("Put []byte did not originate in pool")
}

// NewBufferPool creates a new BufferPool that contains n buffers of the
// specified size. Both arguments must be non-zero.
func NewBufferPool(n, bufSize int) BufferPool {
	if n <= 0 || bufSize <= 0 {
		panic("cannot create empty BufferPool")
	}
	buf := make([]byte, n*bufSize)
	bufs := make([][]byte, n)
	for i := range bufs {
		bufs[i] = buf[i*bufSize : (i+1)*bufSize : (i+1)*bufSize]
	}
	return BufferPool{
		bufs:    bufs,
		indices: NewIndexPool(n),
	}
}
