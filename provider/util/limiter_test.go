package util

import (
	"testing"
	"time"
)

func TestLimiter(t *testing.T) {
	limiter := NewLimiter(2)

	ok := make(chan bool)

	go func() {
		limiter.Acquire()
		limiter.Acquire()
		limiter.Release()
		limiter.Release()

		ok <- true
	}()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	select {
	case <-ok:
	case <-ticker.C:
		t.Errorf("Expected Acquire and Release to succeed")
	}
}

func TestLimiter_AcquireWhenFull(t *testing.T) {
	limiter := NewLimiter(2)
	limiter.Acquire()
	limiter.Acquire()
	err := make(chan error)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	go func() {
		limiter.Acquire()
		err <- nil
	}()

	select {
	case <-err:
		t.Errorf("Expected Acquire to block when limiter is full")
	case <-ticker.C:
	}
}

func TestLimiter_ReleaseWhenEmpty(t *testing.T) {
	limiter := NewLimiter(2)
	err := make(chan error)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	go func() {
		limiter.Release()
		err <- nil
	}()

	select {
	case <-err:
		t.Errorf("Expected Release to block when limiter is empty")
	case <-ticker.C:
	}
}

func TestLimiter_Noop(t *testing.T) {
	limiter := NewLimiter(0)

	ok := make(chan bool)

	go func() {
		limiter.Acquire()
		limiter.Acquire()
		limiter.Acquire()
		limiter.Release()
		limiter.Release()
		limiter.Release()
		limiter.Release()
		ok <- true
	}()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	select {
	case <-ok:
	case <-ticker.C:
		t.Errorf("Expected Acquire and Release to succeed")
	}
}
