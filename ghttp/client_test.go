package ghttp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ink-yht-code/gint/logger"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func init() {
	_ = logger.Init(logger.DefaultConfig())
}

func TestDoRetriesOn5xx(t *testing.T) {
	var attempts int32
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("server error")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})

	c := NewClient(
		WithTransport(transport),
		WithRetry(2, time.Millisecond),
	)
	req := c.NewRequest(http.MethodGet, "http://example.com").WithContext(context.Background())

	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v, want nil", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Fatalf("attempts = %d, want 3", got)
	}
}

func TestParseResponseHandlesHTTPError(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader("bad request")),
	}

	data, err := ParseResponse[map[string]any](resp, nil)
	if data != nil {
		t.Fatalf("data = %v, want nil", data)
	}
	if err == nil || err.Error() != "bad request" {
		t.Fatalf("err = %v, want bad request", err)
	}
}

func TestParseResponseReturnsInputError(t *testing.T) {
	inputErr := errors.New("network down")
	data, err := ParseResponse[map[string]any](nil, inputErr)
	if data != nil {
		t.Fatalf("data = %v, want nil", data)
	}
	if !errors.Is(err, inputErr) {
		t.Fatalf("err = %v, want %v", err, inputErr)
	}
}
