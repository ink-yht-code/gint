// Copyright 2025 ink-yht-code
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "json encoding",
			cfg: Config{
				Level:    "info",
				Encoding: "json",
				Output:   "stdout",
			},
			wantErr: false,
		},
		{
			name: "console encoding",
			cfg: Config{
				Level:    "debug",
				Encoding: "console",
				Output:   "stdout",
			},
			wantErr: false,
		},
		{
			name: "empty encoding defaults to json",
			cfg: Config{
				Level:    "info",
				Encoding: "",
				Output:   "stdout",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Init(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestL(t *testing.T) {
	// Before init, should return nop logger
	defaultLogger = nil
	logger := L()
	assert.NotNil(t, logger)

	// After init
	_ = Init(Config{Level: "info", Encoding: "json", Output: "stdout"})
	logger = L()
	assert.NotNil(t, logger)
}

func TestS(t *testing.T) {
	sugarLogger = nil
	s := S()
	assert.NotNil(t, s)
}

func TestGetLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"debug", "debug"},
		{"info", "info"},
		{"warn", "warn"},
		{"error", "error"},
		{"unknown", "info"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level := getLevel(tt.input)
			assert.Equal(t, tt.expected, level.String())
		})
	}
}

func TestWithContext(t *testing.T) {
	_ = Init(Config{Level: "info", Encoding: "json", Output: "stdout"})

	ctx := context.Background()
	logger := L().With(zap.String("test", "value"))
	ctx = WithContext(ctx, logger)

	retrieved := Ctx(ctx)
	assert.NotNil(t, retrieved)
}

func TestGetRequestID(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "nil context",
			ctx:      nil,
			expected: "",
		},
		{
			name:     "no request id",
			ctx:      context.Background(),
			expected: "",
		},
		{
			name:     "with request_id",
			ctx:      context.WithValue(context.Background(), "request_id", "test-123"),
			expected: "test-123",
		},
		{
			name:     "with X-Request-Id",
			ctx:      context.WithValue(context.Background(), "X-Request-Id", "test-456"),
			expected: "test-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := GetRequestID(tt.ctx)
			assert.Equal(t, tt.expected, id)
		})
	}
}

func TestLogFunctions(t *testing.T) {
	_ = Init(Config{Level: "debug", Encoding: "json", Output: "stdout"})

	// These should not panic
	assert.NotPanics(t, func() { Debug("test debug") })
	assert.NotPanics(t, func() { Info("test info") })
	assert.NotPanics(t, func() { Warn("test warn") })
	assert.NotPanics(t, func() { Error("test error") })
}

func TestLogCtxFunctions(t *testing.T) {
	_ = Init(Config{Level: "debug", Encoding: "json", Output: "stdout"})
	ctx := context.Background()

	// These should not panic
	assert.NotPanics(t, func() { DebugCtx(ctx, "test debug") })
	assert.NotPanics(t, func() { InfoCtx(ctx, "test info") })
	assert.NotPanics(t, func() { WarnCtx(ctx, "test warn") })
	assert.NotPanics(t, func() { ErrorCtx(ctx, "test error") })
}

func TestSync(t *testing.T) {
	_ = Init(Config{Level: "info", Encoding: "json", Output: "stdout"})

	err := Sync()
	assert.NoError(t, err)
}
