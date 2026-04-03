package config

import (
	"errors"
	"testing"

	"github.com/ink-yht-code/gint/logger"
)

type testSource struct {
	name string
	err  error
	data map[string]any
}

func (s testSource) Load() (map[string]any, error) { return s.data, s.err }
func (s testSource) Name() string                  { return s.name }

func init() {
	_ = logger.Init(logger.DefaultConfig())
}

func TestLoadReturnsJoinedErrorsAndMergesSuccessData(t *testing.T) {
	errA := errors.New("source a failed")
	errB := errors.New("source b failed")
	c := New()
	c.sources = []Source{
		testSource{name: "a", err: errA},
		testSource{name: "ok", data: map[string]any{"app": map[string]any{"name": "gint"}}},
		testSource{name: "b", err: errB},
	}

	err := c.Load()
	if err == nil {
		t.Fatal("Load() error = nil, want joined error")
	}
	if !errors.Is(err, errA) || !errors.Is(err, errB) {
		t.Fatalf("Load() error = %v, want contains both source errors", err)
	}
	if got := c.GetString("app.name"); got != "gint" {
		t.Fatalf("GetString(app.name) = %s, want gint", got)
	}
}
