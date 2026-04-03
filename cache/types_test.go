package cache

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultCacheNotInitialized(t *testing.T) {
	original := defaultCache
	defaultCache = nil
	t.Cleanup(func() {
		defaultCache = original
	})

	_, err := Get(context.Background(), "k")
	if !errors.Is(err, ErrCacheNotInitialized) {
		t.Fatalf("Get() error = %v, want %v", err, ErrCacheNotInitialized)
	}

	err = Set(context.Background(), "k", []byte("v"), time.Second)
	if !errors.Is(err, ErrCacheNotInitialized) {
		t.Fatalf("Set() error = %v, want %v", err, ErrCacheNotInitialized)
	}

	err = Delete(context.Background(), "k")
	if !errors.Is(err, ErrCacheNotInitialized) {
		t.Fatalf("Delete() error = %v, want %v", err, ErrCacheNotInitialized)
	}
}
