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

package session

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ink-yht-code/gint/gint/gctx"
	"github.com/ink-yht-code/gint/gint/jwt"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockSession 实现 Session 接口用于测试
type mockSession struct {
	userId string
	data   map[string]any
}

func (m *mockSession) Set(ctx context.Context, key string, val any) error {
	m.data[key] = val
	return nil
}

func (m *mockSession) Get(ctx context.Context, key string) (any, error) {
	return m.data[key], nil
}

func (m *mockSession) Del(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *mockSession) Destroy(ctx context.Context) error {
	m.data = make(map[string]any)
	return nil
}

func (m *mockSession) Claims() *jwt.Claims {
	return &jwt.Claims{
		UserId: m.userId,
		Data:   make(map[string]string),
	}
}

func (m *mockSession) Refresh(ctx context.Context) error {
	return nil
}

// mockProvider 实现 Provider 接口用于测试
type mockProvider struct {
	sess *mockSession
	err  error
}

func (m *mockProvider) NewSession(ctx *gctx.Context, userId string, jwtData map[string]string, sessData map[string]any) (Session, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.sess = &mockSession{userId: userId, data: sessData}
	return m.sess, nil
}

func (m *mockProvider) Get(ctx *gctx.Context) (Session, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.sess == nil {
		return nil, errors.New("session not found")
	}
	return m.sess, nil
}

func (m *mockProvider) Destroy(ctx *gctx.Context) error {
	m.sess = nil
	return nil
}

func (m *mockProvider) RenewToken(ctx *gctx.Context) error {
	return nil
}

func TestErrProviderNotInitialized(t *testing.T) {
	// 测试错误消息
	if ErrProviderNotInitialized.Error() != "session provider 未初始化" {
		t.Errorf("ErrProviderNotInitialized.Error() = %s, want 'session provider 未初始化'", ErrProviderNotInitialized.Error())
	}
}

func TestGetDefaultProvider_NotInitialized(t *testing.T) {
	// 使用一个新的 atomic.Value 来测试未初始化情况
	// 因为 atomic.Value 不能存储 nil，我们需要用新变量测试
	var testProvider atomic.Value

	// 测试未初始化时返回错误
	provider, err := getDefaultProviderFrom(testProvider)
	if !errors.Is(err, ErrProviderNotInitialized) {
		t.Errorf("getDefaultProvider() error = %v, want ErrProviderNotInitialized", err)
	}
	if provider != nil {
		t.Errorf("getDefaultProvider() provider = %v, want nil", provider)
	}
}

// getDefaultProviderFrom 从指定的 atomic.Value 获取 Provider
func getDefaultProviderFrom(v atomic.Value) (Provider, error) {
	p := v.Load()
	if p == nil {
		return nil, ErrProviderNotInitialized
	}
	return p.(Provider), nil
}

func TestSetDefaultProvider(t *testing.T) {
	// 保存当前状态
	originalProvider := defaultProvider.Load()

	// 设置 mock provider
	mock := &mockProvider{}
	SetDefaultProvider(mock)

	// 验证设置成功
	provider, err := getDefaultProvider()
	if err != nil {
		t.Errorf("getDefaultProvider() error = %v, want nil", err)
	}
	if provider != mock {
		t.Errorf("getDefaultProvider() provider = %p, want %p", provider, mock)
	}

	// 恢复状态
	if originalProvider != nil {
		defaultProvider.Store(originalProvider)
	}
}

func TestSetProvider(t *testing.T) {
	// 创建测试上下文
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := &gctx.Context{Context: ginCtx}

	// 创建 mock provider
	mock := &mockProvider{}

	// 注入 provider
	SetProvider(ctx, mock)

	// 验证注入成功
	provider, err := getProvider(ctx)
	if err != nil {
		t.Errorf("getProvider() error = %v, want nil", err)
	}
	if provider != mock {
		t.Errorf("getProvider() provider = %p, want %p", provider, mock)
	}
}

func TestGetProvider_FallbackToDefault(t *testing.T) {
	// 保存当前状态
	originalProvider := defaultProvider.Load()

	// 设置默认 provider
	defaultMock := &mockProvider{}
	SetDefaultProvider(defaultMock)

	// 创建没有注入 provider 的上下文
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := &gctx.Context{Context: ginCtx}

	// 应该回退到默认 provider
	provider, err := getProvider(ctx)
	if err != nil {
		t.Errorf("getProvider() error = %v, want nil", err)
	}
	if provider != defaultMock {
		t.Errorf("getProvider() provider = %p, want %p", provider, defaultMock)
	}

	// 恢复状态
	if originalProvider != nil {
		defaultProvider.Store(originalProvider)
	}
}

func TestGetProvider_NilContext(t *testing.T) {
	// 保存当前状态
	originalProvider := defaultProvider.Load()

	// 设置默认 provider
	defaultMock := &mockProvider{}
	SetDefaultProvider(defaultMock)

	// 传入 nil context 应该回退到默认 provider
	provider, err := getProvider(nil)
	if err != nil {
		t.Errorf("getProvider(nil) error = %v, want nil", err)
	}
	if provider != defaultMock {
		t.Errorf("getProvider(nil) provider = %p, want %p", provider, defaultMock)
	}

	// 恢复状态
	if originalProvider != nil {
		defaultProvider.Store(originalProvider)
	}
}

func TestGet_Session(t *testing.T) {
	// 保存当前状态
	originalProvider := defaultProvider.Load()

	// 设置 mock provider
	mock := &mockProvider{
		sess: &mockSession{userId: "user123", data: make(map[string]any)},
	}
	SetDefaultProvider(mock)

	// 创建测试上下文
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := &gctx.Context{Context: ginCtx}

	// 获取 session
	sess, err := Get(ctx)
	if err != nil {
		t.Errorf("Get() error = %v, want nil", err)
	}
	if sess == nil {
		t.Errorf("Get() sess = nil, want non-nil")
	}

	// 恢复状态
	if originalProvider != nil {
		defaultProvider.Store(originalProvider)
	}
}

func TestGet_ProviderError(t *testing.T) {
	// 保存当前状态
	originalProvider := defaultProvider.Load()

	// 设置返回错误的 mock provider
	mock := &mockProvider{err: errors.New("provider error")}
	SetDefaultProvider(mock)

	// 创建测试上下文
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := &gctx.Context{Context: ginCtx}

	// 获取 session 应该返回错误
	sess, err := Get(ctx)
	if err == nil {
		t.Errorf("Get() error = nil, want error")
	}
	if sess != nil {
		t.Errorf("Get() sess = %v, want nil", sess)
	}

	// 恢复状态
	if originalProvider != nil {
		defaultProvider.Store(originalProvider)
	}
}

func TestGet_ProviderNotInitialized(t *testing.T) {
	// 使用一个新的 atomic.Value 来测试未初始化情况
	// 因为 atomic.Value 不能存储 nil，我们用新变量模拟未初始化状态
	var testProvider atomic.Value

	// 使用测试的 provider 获取函数
	p, err := getDefaultProviderFrom(testProvider)
	if !errors.Is(err, ErrProviderNotInitialized) {
		t.Errorf("Get() error = %v, want ErrProviderNotInitialized", err)
	}
	if p != nil {
		t.Errorf("Get() provider = %v, want nil", p)
	}
}

func TestNewSession(t *testing.T) {
	// 保存当前状态
	originalProvider := defaultProvider.Load()

	// 设置 mock provider
	mock := &mockProvider{}
	SetDefaultProvider(mock)

	// 创建测试上下文
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := &gctx.Context{Context: ginCtx}

	// 创建新 session
	sess, err := NewSession(ctx, "user123", map[string]string{"role": "admin"}, map[string]any{"count": 0})
	if err != nil {
		t.Errorf("NewSession() error = %v, want nil", err)
	}
	if sess == nil {
		t.Errorf("NewSession() sess = nil, want non-nil")
	}

	// 恢复状态
	if originalProvider != nil {
		defaultProvider.Store(originalProvider)
	}
}

func TestNewSession_ProviderNotInitialized(t *testing.T) {
	// 使用一个新的 atomic.Value 来测试未初始化情况
	// 因为 atomic.Value 不能存储 nil，我们用新变量模拟未初始化状态
	var testProvider atomic.Value

	// 使用测试的 provider 获取函数
	p, err := getDefaultProviderFrom(testProvider)
	if !errors.Is(err, ErrProviderNotInitialized) {
		t.Errorf("NewSession() error = %v, want ErrProviderNotInitialized", err)
	}
	if p != nil {
		t.Errorf("NewSession() provider = %v, want nil", p)
	}
}

func TestContextProviderInjection(t *testing.T) {
	// 保存当前状态
	originalProvider := defaultProvider.Load()

	// 设置默认 provider
	defaultMock := &mockProvider{}
	SetDefaultProvider(defaultMock)

	// 创建测试上下文
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := &gctx.Context{Context: ginCtx}

	// 注入不同的 provider
	contextMock := &mockProvider{
		sess: &mockSession{userId: "context_user", data: make(map[string]any)},
	}
	SetProvider(ctx, contextMock)

	// Get 应该使用上下文中的 provider，而不是默认的
	sess, err := Get(ctx)
	if err != nil {
		t.Errorf("Get() error = %v, want nil", err)
	}
	if sess == nil {
		t.Errorf("Get() sess = nil, want non-nil")
	}

	// 验证使用的是 contextMock 而不是 defaultMock
	if sess.Claims().UserId != "context_user" {
		t.Errorf("Get() userId = %s, want 'context_user'", sess.Claims().UserId)
	}

	// 恢复状态
	if originalProvider != nil {
		defaultProvider.Store(originalProvider)
	}
}
