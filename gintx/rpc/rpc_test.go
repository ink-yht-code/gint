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

package rpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		cfg := Config{Enabled: false, Addr: ":9090"}
		s := NewServer(cfg)
		assert.Nil(t, s)
	})

	t.Run("enabled", func(t *testing.T) {
		cfg := Config{Enabled: true, Addr: ":0"}
		s := NewServer(cfg)
		assert.NotNil(t, s)
		assert.NotNil(t, s.Server)
	})
}

func TestServer_Shutdown(t *testing.T) {
	cfg := Config{Enabled: true, Addr: ":0"}
	s := NewServer(cfg)

	ctx := context.Background()
	err := s.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestUnaryInterceptor(t *testing.T) {
	// Test that the interceptor function exists
	assert.NotNil(t, unaryInterceptor)
}

func TestStreamInterceptor(t *testing.T) {
	// Test that the interceptor function exists
	assert.NotNil(t, streamInterceptor)
}
