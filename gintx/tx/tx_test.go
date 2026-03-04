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

package tx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestNewManager(t *testing.T) {
	// Test that NewManager returns a non-nil Manager
	m := NewManager(nil)
	assert.NotNil(t, m)
}

func TestFromContext(t *testing.T) {
	tests := []struct {
		name      string
		ctx       context.Context
		expectNil bool
	}{
		{
			name:      "nil context returns default",
			ctx:       nil,
			expectNil: true,
		},
		{
			name:      "empty context returns default",
			ctx:       context.Background(),
			expectNil: true,
		},
		{
			name:      "context with tx value",
			ctx:       context.WithValue(context.Background(), ctxKey{}, &gorm.DB{}),
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FromContext(tt.ctx, nil)
			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
			}
		})
	}
}

func TestGetDB(t *testing.T) {
	t.Run("returns from context when available", func(t *testing.T) {
		tx := &gorm.DB{}
		ctx := context.WithValue(context.Background(), ctxKey{}, tx)
		result := GetDB(ctx, nil)
		assert.Equal(t, tx, result)
	})

	t.Run("returns default when not in context", func(t *testing.T) {
		ctx := context.Background()
		result := GetDB(ctx, nil)
		assert.Nil(t, result)
	})
}

func TestCtxKey(t *testing.T) {
	// Verify ctxKey is a valid context key type
	key := ctxKey{}
	_ = key
}
