package mempool

import (
	"reflect"
	"sync"
	"unsafe"
)

// A MemPool is a pool of fixed-size []byte buffers. MemPools are safe for
// concurrent use.
type MemPool struct {
	bufs [][]byte
	mu   sync.Mutex
	cond *sync.Cond
}

// Get returns one of the buffers in the pool. If no buffers are available,
// Get blocks. Buffers are not zeroed before being returned.
func (p *MemPool) Get() []byte {
	// search for a buf with len > 0 (i.e. available)
	p.mu.Lock()
	for {
		for i, s := range p.bufs {
			if len(s) != 0 {
				// mark the buf as unavailable
				(*reflect.SliceHeader)(unsafe.Pointer(&p.bufs[i])).Len = 0
				p.mu.Unlock()
				return s
			}
		}
		// no bufs are available, so block until woken up by a call to Put
		p.cond.Wait()
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
	bHdr := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	for i := range p.bufs {
		iHdr := (*reflect.SliceHeader)(unsafe.Pointer(&p.bufs[i]))
		if iHdr.Data == bHdr.Data {
			// mark the buf as available
			p.mu.Lock()
			iHdr.Len = iHdr.Cap
			p.mu.Unlock()
			// if there are blocked Get calls, wake one up
			p.cond.Signal()
			return
		}
	}
	panic("Put []byte did not originate in pool")
}

// New creates a new MemPool. Both arguments must be non-zero.
func New(bufs, bufSize int) *MemPool {
	if bufs <= 0 || bufSize <= 0 {
		panic("cannot create empty MemPool")
	}
	s := &MemPool{bufs: make([][]byte, bufs)}
	for i := range s.bufs {
		s.bufs[i] = make([]byte, bufSize)
	}
	s.cond = sync.NewCond(&s.mu)
	return s
}
