package pool

import (
	"runtime"
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
				x := p.Get()
				runtime.Gosched()
				p.Put(x)
			}
		}()
	}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p.Put(p.Get())
	}
}
