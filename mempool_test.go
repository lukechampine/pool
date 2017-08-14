package mempool

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

func TestPool(t *testing.T) {
	s := New(10, 1000)

	// Get all available buffers
	var got []byte
	for i := 0; i < 10; i++ {
		got = s.Get()
	}

	// 11th Get should block until we call Put
	done := make(chan struct{})
	go func() {
		_ = s.Get()
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("Get should have blocked")
	case <-time.After(10 * time.Millisecond):
	}

	// Modify and return the buffer we got
	got = append(got[:0], 1)
	s.Put(got)

	// 11th Get should now return
	select {
	case <-done:
	case <-time.After(10 * time.Millisecond):
		t.Fatal("Get should have returned")
	}

	// Putting a buffer not owned by the pool should cause a panic
	shouldPanic(t, func() { s.Put(make([]byte, 1000)) })
}

func TestConcurrent(t *testing.T) {
	s := New(10, 1000)

	getAndPut := func(n int) {
		for i := 0; i < n; i++ {
			b := s.Get()
			s.Put(b)
			runtime.Gosched()
		}
	}
	for i := 0; i < 10; i++ {
		go getAndPut(1000)
	}
	getAndPut(1000)
}

func TestEmptyPool(t *testing.T) {
	// Empty pools should panic
	shouldPanic(t, func() { New(0, 1) })
	shouldPanic(t, func() { New(1, 0) })
	shouldPanic(t, func() { New(0, 0) })
}

func BenchmarkPool(b *testing.B) {
	s := New(1000, 1000)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b := s.Get()
		s.Put(b)
	}
}

func BenchmarkPoolContention(b *testing.B) {
	s := New(1000, 1000)
	for i := 0; i < 10; i++ {
		go func() {
			for {
				b := s.Get()
				s.Put(b)
				runtime.Gosched()
			}
		}()
	}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b := s.Get()
		s.Put(b)
	}
}
