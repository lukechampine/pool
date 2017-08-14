package mempool

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

// A MemPool is a pool of fixed-size []byte buffers. MemPools are safe for
// concurrent use.
type MemPool struct {
	bufs [][]byte
	cond *sync.Cond
}

// uintptrSliceHeader represents the memory layout of a slice. It is identical
// to reflect.SliceHeader, except that Len and Cap are uintptrs instead of
// ints. This allows atomic operations on those fields. Unfortunately, it also
// means that this package may break on architectures where sizeof(int) !=
// sizeof(uintptr).
type uintptrSliceHeader struct {
	Data, Len, Cap uintptr
}

// Get returns one of the buffers in the pool. If no buffers are available,
// Get blocks. Buffers are not zeroed before being returned.
func (p *MemPool) Get() []byte {
	// search for a buf with len > 0 (i.e. available)
	for {
		for i, s := range p.bufs {
			iHdr := (*uintptrSliceHeader)(unsafe.Pointer(&p.bufs[i]))
			// try to mark the buffer as unavailable
			if atomic.CompareAndSwapUintptr(&iHdr.Len, iHdr.Cap, 0) {
				return s
			}
		}
		// no bufs are available, so block until woken up by a call to Put
		p.cond.L.Lock()
		p.cond.Wait()
		p.cond.L.Unlock()
	}
}

// Put returns a buffer to the pool. b must be a buffer that was returned by
// Get; otherwise, Put panics. However, the caller may modify the contents of
// the buffer or change its length or capacity before returning it. All that
// matters is that b point to the same memory location as the original slice
// returned by Get. As an example, this is legal:
//
//    b := pool.Get()
//    b = append(b[:0], 1) // reuses existing capacity
//    pool.Put(b)
//
// But this is not:
//
//    b := pool.Get()
//    b = append(b, 1) // causes b to be reallocated
//    pool.Put(b)
func (p *MemPool) Put(b []byte) {
	// look for the buffer whose pointer matches b
	bHdr := (*uintptrSliceHeader)(unsafe.Pointer(&b))
	for i := range p.bufs {
		iHdr := (*uintptrSliceHeader)(unsafe.Pointer(&p.bufs[i]))
		if iHdr.Data == bHdr.Data {
			// mark the buf as available
			atomic.StoreUintptr(&iHdr.Len, iHdr.Cap)
			// if there are blocked Get calls, wake one up
			p.cond.Signal()
			return
		}
	}
	panic("Put []byte did not originate in pool")
}

// New creates a new MemPool that contains n buffers of the specified size.
// Both arguments must be non-zero.
func New(n, bufSize int) *MemPool {
	if n <= 0 || bufSize <= 0 {
		panic("cannot create empty MemPool")
	}
	bufs := make([][]byte, n)
	for i := range bufs {
		bufs[i] = make([]byte, bufSize)
	}
	return &MemPool{
		bufs: bufs,
		cond: sync.NewCond(new(sync.Mutex)),
	}
}
