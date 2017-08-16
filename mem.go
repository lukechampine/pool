// Package mem is a collection of memory-management utilities.
package mem

// noopLocker implements the sync.Locker interface with no-ops. It exists
// solely to speed up the methods of sync.Cond.
type noopLocker struct{}

func (noopLocker) Lock()   {}
func (noopLocker) Unlock() {}

// uintptrSliceHeader represents the memory layout of a slice. It is identical
// to reflect.SliceHeader, except that Len and Cap are uintptrs instead of
// ints. This allows atomic operations on those fields. Unfortunately, it also
// means that this package may break on architectures where sizeof(int) !=
// sizeof(uintptr).
type uintptrSliceHeader struct {
	Data, Len, Cap uintptr
}
