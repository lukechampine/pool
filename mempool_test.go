package pool

import (
	"bytes"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestMemPool(t *testing.T) {
	p := NewMemPool(10, 1000)

	// Get all available buffers
	var got []byte
	for i := 0; i < 10; i++ {
		got = p.Get()
	}

	// 11th Get should block until we call Put
	done := make(chan struct{})
	go func() {
		_ = p.Get()
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("Get should have blocked")
	case <-time.After(10 * time.Millisecond):
	}

	// Modify and return the buffer we got
	got = append(got[:0], 1)
	p.Put(got)

	// 11th Get should now return
	select {
	case <-done:
	case <-time.After(10 * time.Millisecond):
		t.Fatal("Get should have returned")
	}

	// Putting a buffer not owned by the pool should cause a panic
	shouldPanic(t, func() { p.Put(make([]byte, 1000)) })
}

func TestMemPoolConcurrent(t *testing.T) {
	p := NewMemPool(10, 1000)

	getAndPut := func(n int) {
		for i := 0; i < n; i++ {
			b := p.Get()
			runtime.Gosched()
			p.Put(b)
		}
	}
	for i := 0; i < 10; i++ {
		go getAndPut(1000)
	}
	getAndPut(1000)
}

func TestMemPoolEmpty(t *testing.T) {
	// Empty pools should panic
	shouldPanic(t, func() { NewMemPool(0, 1) })
	shouldPanic(t, func() { NewMemPool(1, 0) })
	shouldPanic(t, func() { NewMemPool(0, 0) })
}

func BenchmarkMemPool(b *testing.B) {
	p := NewMemPool(1000, 1000)
	p.NoClear = true
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p.Put(p.Get())
	}
}

func BenchmarkMemPoolContention(b *testing.B) {
	p := NewMemPool(1000, 1000)
	p.NoClear = true
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < b.N*2; j++ {
				b := p.Get()
				runtime.Gosched()
				p.Put(b)
			}
		}()
	}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p.Put(p.Get())
	}
}

func BenchmarkSyncPool(b *testing.B) {
	p := sync.Pool{
		New: func() interface{} {
			return make([]byte, 1000)
		},
	}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p.Put(p.Get().([]byte))
	}
}

func BenchmarkSyncPoolContention(b *testing.B) {
	p := sync.Pool{
		New: func() interface{} {
			return make([]byte, 1000)
		},
	}
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < b.N*2; j++ {
				b := p.Get().([]byte)
				runtime.Gosched()
				p.Put(b)
			}
		}()
	}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p.Put(p.Get().([]byte))
	}
}

func TestBufferPool(t *testing.T) {
	p := NewBufferPool(10, 1000)

	// Get all available buffers
	var got *bytes.Buffer
	for i := 0; i < 10; i++ {
		got = p.Get()
	}

	// 11th Get should block until we call Put
	done := make(chan struct{})
	go func() {
		_ = p.Get()
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("Get should have blocked")
	case <-time.After(10 * time.Millisecond):
	}

	// Modify and return the buffer we got
	got.WriteString("foo")
	p.Put(got)

	// 11th Get should now return
	select {
	case <-done:
	case <-time.After(10 * time.Millisecond):
		t.Fatal("Get should have returned")
	}

	// Putting a buffer not owned by the pool should cause a panic
	shouldPanic(t, func() { p.Put(new(bytes.Buffer)) })
}

func TestBufferPoolConcurrent(t *testing.T) {
	p := NewBufferPool(10, 1000)

	getAndPut := func(n int) {
		for i := 0; i < n; i++ {
			b := p.Get()
			runtime.Gosched()
			p.Put(b)
		}
	}
	for i := 0; i < 10; i++ {
		go getAndPut(1000)
	}
	getAndPut(1000)
}

func TestBufferPoolEmpty(t *testing.T) {
	// Empty pools should panic
	shouldPanic(t, func() { NewBufferPool(0, 0) })
	shouldPanic(t, func() { NewBufferPool(0, 1) })
	// Unlike NewMemPool, second arg may be 0
	_ = NewBufferPool(1, 0)
}
