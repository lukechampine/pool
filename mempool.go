package pool

import (
	"bytes"
	"reflect"
	"unsafe"
)

// A MemPool is a pool of fixed-size []byte buffers. MemPools are safe
// for concurrent use.
type MemPool struct {
	// NoClear controls whether buffers are cleared before being returned to
	// the pool. Enabling NoClear improves performance, but requires the
	// caller to deal with potentially dirty buffers.
	NoClear bool

	bufs    [][]byte
	indices *IndexPool
}

// Get returns one of the buffers in the pool, blocking if necessary until one
// becomes available.
func (p MemPool) Get() []byte {
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
func (p MemPool) Put(b []byte) {
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

// NewMemPool creates a new MemPool that contains n buffers of the
// specified size. Both arguments must be non-zero.
func NewMemPool(n, size int) MemPool {
	if n <= 0 || size <= 0 {
		panic("cannot create empty MemPool")
	}
	buf := make([]byte, n*size)
	bufs := make([][]byte, n)
	for i := range bufs {
		bufs[i] = buf[i*size : (i+1)*size : (i+1)*size]
	}
	return MemPool{
		bufs:    bufs,
		indices: NewIndexPool(n),
	}
}

// A BufferPool is a pool of bytes.Buffers. BufferPools are safe for
// concurrent use. Note that unlike a MemPool, writing to the buffers may
// cause them to grow beyond their original capacity.
type BufferPool struct {
	bufs    []*bytes.Buffer
	indices *IndexPool
}

// Get returns one of the buffers in the pool, blocking if necessary until one
// becomes available. The buffer will have a length of 0.
func (p BufferPool) Get() *bytes.Buffer {
	return p.bufs[p.indices.Get()]
}

// Put returns a buffer to the pool. b must be a buffer that was returned by
// Get; otherwise, Put panics. Callers must not modify the contents of a
// buffer after returning it to the pool with Put.
func (p BufferPool) Put(b *bytes.Buffer) {
	for i := range p.bufs {
		if p.bufs[i] == b {
			p.bufs[i].Reset()
			p.indices.Put(i)
			return
		}
	}
	panic("Put buffer did not originate in pool")
}

// NewBufferPool creates a new BufferPool that contains n buffers of the
// specified capacity. n must be non-zero.
func NewBufferPool(n, capacity int) BufferPool {
	if n <= 0 {
		panic("cannot create empty BufferPool")
	}
	bufs := make([]*bytes.Buffer, n)
	for i := range bufs {
		bufs[i] = new(bytes.Buffer)
		bufs[i].Grow(capacity)
	}
	return BufferPool{
		bufs:    bufs,
		indices: NewIndexPool(n),
	}
}
