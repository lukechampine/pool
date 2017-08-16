package mem

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

func TestFixedPool(t *testing.T) {
	p := NewFixedPool(10, 1000)

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

func TestFixedPoolConcurrent(t *testing.T) {
	p := NewFixedPool(10, 1000)

	getAndPut := func(n int) {
		for i := 0; i < n; i++ {
			b := p.Get()
			p.Put(b)
			runtime.Gosched()
		}
	}
	for i := 0; i < 10; i++ {
		go getAndPut(1000)
	}
	getAndPut(1000)
}

func TestFixedPoolEmpty(t *testing.T) {
	// Empty pools should panic
	shouldPanic(t, func() { NewFixedPool(0, 1) })
	shouldPanic(t, func() { NewFixedPool(1, 0) })
	shouldPanic(t, func() { NewFixedPool(0, 0) })
}

func BenchmarkFixedPool(b *testing.B) {
	p := NewFixedPool(1000, 1000)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b := p.Get()
		p.Put(b)
	}
}

func BenchmarkFixedPoolContention(b *testing.B) {
	p := NewFixedPool(1000, 1000)
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
		b := p.Get()
		p.Put(b)
	}
}
