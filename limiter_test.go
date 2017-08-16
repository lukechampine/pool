package mem

import (
	"runtime"
	"testing"
	"time"
)

func TestLimiter(t *testing.T) {
	l := NewLimiter(10)

	// Get full limit
	l.Get(10)

	// Next Get should block until we call Put
	done := make(chan struct{})
	go func() {
		l.Get(1)
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("Get should have blocked")
	case <-time.After(10 * time.Millisecond):
	}

	// Return 1
	l.Put(1)

	// Get should now return
	select {
	case <-done:
	case <-time.After(10 * time.Millisecond):
		t.Fatal("Get should have returned")
	}
}

func TestMemLimiter(t *testing.T) {
	l := NewMemLimiter(10)

	// Get full limit
	got := l.Get(10)

	// Next Get should block until we call Put
	done := make(chan struct{})
	go func() {
		_ = l.Get(1)
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("Get should have blocked")
	case <-time.After(10 * time.Millisecond):
	}

	// Return 1
	l.Put(got[:1])

	// Get should now return
	select {
	case <-done:
	case <-time.After(10 * time.Millisecond):
		t.Fatal("Get should have returned")
	}
}

func TestLimiterConcurrent(t *testing.T) {
	l := NewLimiter(10)

	getAndPut := func(n int) {
		for i := 0; i < n; i++ {
			l.Get(1)
			l.Put(1)
			runtime.Gosched()
		}
	}
	for i := 0; i < 10; i++ {
		go getAndPut(1000)
	}
	getAndPut(1000)
}

func TestLimiterPanics(t *testing.T) {
	shouldPanic(t, func() { NewLimiter(-1) })
	shouldPanic(t, func() { NewLimiter(1).Get(-1) })
	shouldPanic(t, func() { NewLimiter(1).Get(2) })
	shouldPanic(t, func() { NewLimiter(1).Put(-1) })
	shouldPanic(t, func() { NewLimiter(1).Put(2) })
}

func BenchmarkLimiter(b *testing.B) {
	l := NewLimiter(10)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		l.Get(1)
		l.Put(1)
	}
}

func BenchmarkLimiterContention(b *testing.B) {
	l := NewLimiter(10)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < b.N*2; j++ {
				l.Get(1)
				l.Put(1)
				runtime.Gosched()
			}
		}()
	}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		l.Get(1)
		l.Put(1)
	}
}
