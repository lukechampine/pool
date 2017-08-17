package mem

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

func shouldPanic(t *testing.T, fn func()) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	fn()
}

func TestIndexPool(t *testing.T) {
	p := NewIndexPool(10)

	// Get all available indices
	var got int
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

	// Return the index we got
	p.Put(got)

	// 11th Get should now return
	select {
	case <-done:
	case <-time.After(10 * time.Millisecond):
		t.Fatal("Get should have returned")
	}
}

func TestIndexPoolConcurrent(t *testing.T) {
	p := NewIndexPool(10)

	getAndPut := func(n int) {
		for i := 0; i < n; i++ {
			p.Put(p.Get())
			runtime.Gosched()
		}
	}
	for i := 0; i < 10; i++ {
		go getAndPut(1000)
	}
	getAndPut(1000)
}

func TestIndexPoolPanics(t *testing.T) {
	// empty pool
	shouldPanic(t, func() { NewIndexPool(0) })
	// negative pool
	shouldPanic(t, func() { NewIndexPool(-1) })
	// out-of-bounds Put
	shouldPanic(t, func() { NewIndexPool(1).Put(-1) })
	shouldPanic(t, func() { NewIndexPool(1).Put(2) })
	// Put without Get
	shouldPanic(t, func() { NewIndexPool(1).Put(0) })
}

func BenchmarkIndexPool(b *testing.B) {
	p := NewIndexPool(1000)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p.Put(p.Get())
	}
}

func BenchmarkIndexPoolContention(b *testing.B) {
	p := NewIndexPool(1000)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < b.N*2; j++ {
				p.Put(p.Get())
				runtime.Gosched()
			}
		}()
	}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p.Put(p.Get())
	}
}

func TestBufferPool(t *testing.T) {
	p := NewBufferPool(10, 1000)

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

func TestBufferPoolConcurrent(t *testing.T) {
	p := NewBufferPool(10, 1000)

	getAndPut := func(n int) {
		for i := 0; i < n; i++ {
			p.Put(p.Get())
			runtime.Gosched()
		}
	}
	for i := 0; i < 10; i++ {
		go getAndPut(1000)
	}
	getAndPut(1000)
}

func TestBufferPoolEmpty(t *testing.T) {
	// Empty pools should panic
	shouldPanic(t, func() { NewBufferPool(0, 1) })
	shouldPanic(t, func() { NewBufferPool(1, 0) })
	shouldPanic(t, func() { NewBufferPool(0, 0) })
}

func BenchmarkBufferPool(b *testing.B) {
	p := NewBufferPool(1000, 1000)
	p.NoClear = true
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p.Put(p.Get())
	}
}

func BenchmarkBufferPoolContention(b *testing.B) {
	p := NewBufferPool(1000, 1000)
	p.NoClear = true
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < b.N*2; j++ {
				b := p.Get()
				p.Put(b)
				runtime.Gosched()
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
				p.Put(p.Get().([]byte))
				runtime.Gosched()
			}
		}()
	}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p.Put(p.Get().([]byte))
	}
}
