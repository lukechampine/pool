// Package pool is a collection of pool-related utilities.
package pool

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// noopLocker implements the sync.Locker interface with no-ops. It exists
// solely to speed up the methods of sync.Cond.
type noopLocker struct{}

func (noopLocker) Lock()   {}
func (noopLocker) Unlock() {}

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
