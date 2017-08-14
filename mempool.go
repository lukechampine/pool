package mempool

import (
	"reflect"
	"sync"
	"unsafe"
)

type MemPool struct {
	bufs [][]byte
	mu   sync.Mutex
	cond *sync.Cond
}

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

func New(bufs, bufSize int) *MemPool {
	if bufs == 0 || bufSize == 0 {
		panic("cannot create empty MemPool")
	}
	s := &MemPool{bufs: make([][]byte, bufs)}
	for i := range s.bufs {
		s.bufs[i] = make([]byte, bufSize)
	}
	s.cond = sync.NewCond(&s.mu)
	return s
}
