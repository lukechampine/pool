// Package mem is a collection of memory-management utilities.
package mem

// noopLocker implements the sync.Locker interface with no-ops. It exists
// solely to speed up the methods of sync.Cond.
type noopLocker struct{}

func (noopLocker) Lock()   {}
func (noopLocker) Unlock() {}
