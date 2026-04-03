package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ink-yht-code/gint/logger"
)

func init() {
	gin.SetMode(gin.TestMode)
	_ = logger.Init(logger.DefaultConfig())
}

func TestWithShutdownTimeoutOption(t *testing.T) {
	s := New(":0", gin.New(), WithShutdownTimeout(5*time.Second))
	if s.shutdownTS != 5*time.Second {
		t.Fatalf("shutdownTS = %v, want %v", s.shutdownTS, 5*time.Second)
	}
}

func TestStartReturnsNil(t *testing.T) {
	s := New(":0", gin.New())
	if err := s.Start(); err != nil {
		t.Fatalf("Start() error = %v, want nil", err)
	}
}

func TestHookCloseFunctionsInvokeCloseFunc(t *testing.T) {
	expectedErr := errors.New("close failed")
	var called int
	closeFn := func() error {
		called++
		return expectedErr
	}

	hooks := []ShutdownHook{
		HookCloseDB(closeFn),
		HookCloseRedis(closeFn),
		HookCloseMQ(closeFn),
	}

	for _, hook := range hooks {
		err := hook(context.Background())
		if !errors.Is(err, expectedErr) {
			t.Fatalf("hook error = %v, want %v", err, expectedErr)
		}
	}
	if called != 3 {
		t.Fatalf("close function called %d times, want 3", called)
	}
}
