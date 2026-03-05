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

package gint

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ink-yht-code/gint/gint/gctx"
	"github.com/ink-yht-code/gint/gint/internal/jwt"
	"github.com/ink-yht-code/gint/gint/session"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockSession 实现 session.Session 接口用于测试
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

// mockProvider 实现 session.Provider 接口用于测试
type mockProvider struct {
	sess *mockSession
	err  error
}

func (m *mockProvider) NewSession(ctx *gctx.Context, userId string, jwtData map[string]string, sessData map[string]any) (session.Session, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.sess = &mockSession{userId: userId, data: sessData}
	return m.sess, nil
}

func (m *mockProvider) Get(ctx *gctx.Context) (session.Session, error) {
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

func TestW_Success(t *testing.T) {
	r := gin.New()
	r.GET("/ping", W(func(ctx *gctx.Context) (Result, error) {
		return Success("pong", nil), nil
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ping", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), `"code":0`) {
		t.Errorf("body = %s, want code 0", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "pong") {
		t.Errorf("body = %s, want pong", w.Body.String())
	}
}

func TestW_BusinessError(t *testing.T) {
	r := gin.New()
	r.GET("/error", W(func(ctx *gctx.Context) (Result, error) {
		return ErrorWithCode(10001, "业务错误"), nil
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/error", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), `"code":10001`) {
		t.Errorf("body = %s, want code 10001", w.Body.String())
	}
}

func TestW_InternalError(t *testing.T) {
	r := gin.New()
	r.GET("/error", W(func(ctx *gctx.Context) (Result, error) {
		return Result{}, errors.New("internal error")
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/error", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), `"code":20000`) {
		t.Errorf("body = %s, want code 20000 (InternalError)", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "系统繁忙") {
		t.Errorf("body = %s, want 系统繁忙", w.Body.String())
	}
}

func TestW_ErrNoResponse(t *testing.T) {
	r := gin.New()
	r.GET("/no-response", W(func(ctx *gctx.Context) (Result, error) {
		return Result{}, ErrNoResponse
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/no-response", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != "" {
		t.Errorf("body = %s, want empty", w.Body.String())
	}
}

func TestW_ErrUnauthorized(t *testing.T) {
	r := gin.New()
	r.GET("/unauthorized", W(func(ctx *gctx.Context) (Result, error) {
		return Result{}, ErrUnauthorized
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/unauthorized", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestB_Success(t *testing.T) {
	type Req struct {
		Name string `json:"name" binding:"required"`
	}

	r := gin.New()
	r.POST("/login", B(func(ctx *gctx.Context, req Req) (Result, error) {
		return Success("登录成功", req.Name), nil
	}))

	w := httptest.NewRecorder()
	body := `{"name":"test"}`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), `"code":0`) {
		t.Errorf("body = %s, want code 0", w.Body.String())
	}
}

func TestB_BindError(t *testing.T) {
	type Req struct {
		Name string `json:"name" binding:"required"`
	}

	r := gin.New()
	r.POST("/login", B(func(ctx *gctx.Context, req Req) (Result, error) {
		return Success("", req.Name), nil
	}))

	w := httptest.NewRecorder()
	body := `{}`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), `"code":10000`) {
		t.Errorf("body = %s, want code 10000 (InvalidParam)", w.Body.String())
	}
}

func TestB_InternalError(t *testing.T) {
	type Req struct {
		Name string `json:"name"`
	}

	r := gin.New()
	r.POST("/error", B(func(ctx *gctx.Context, req Req) (Result, error) {
		return Result{}, errors.New("database error")
	}))

	w := httptest.NewRecorder()
	body := `{"name":"test"}`
	req := httptest.NewRequest("POST", "/error", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), `"code":20000`) {
		t.Errorf("body = %s, want code 20000 (InternalError)", w.Body.String())
	}
}

func TestS_Success(t *testing.T) {
	// 设置 mock provider
	provider := &mockProvider{sess: &mockSession{userId: "user123", data: make(map[string]any)}}
	session.SetDefaultProvider(provider)

	r := gin.New()
	r.GET("/profile", S(func(ctx *gctx.Context, sess session.Session) (Result, error) {
		return Success("", sess.Claims().UserId), nil
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/profile", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestS_SessionError(t *testing.T) {
	// 设置返回错误的 provider
	provider := &mockProvider{err: errors.New("session not found")}
	session.SetDefaultProvider(provider)

	r := gin.New()
	r.GET("/profile", S(func(ctx *gctx.Context, sess session.Session) (Result, error) {
		return Success("", nil), nil
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/profile", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestS_ErrUnauthorized(t *testing.T) {
	provider := &mockProvider{sess: &mockSession{userId: "user123", data: make(map[string]any)}}
	session.SetDefaultProvider(provider)

	r := gin.New()
	r.GET("/profile", S(func(ctx *gctx.Context, sess session.Session) (Result, error) {
		return Result{}, ErrUnauthorized
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/profile", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestBS_Success(t *testing.T) {
	type Req struct {
		Nickname string `json:"nickname"`
	}

	provider := &mockProvider{sess: &mockSession{userId: "user123", data: make(map[string]any)}}
	session.SetDefaultProvider(provider)

	r := gin.New()
	r.POST("/profile", BS(func(ctx *gctx.Context, req Req, sess session.Session) (Result, error) {
		return Success("更新成功", nil), nil
	}))

	w := httptest.NewRecorder()
	body := `{"nickname":"test"}`
	req := httptest.NewRequest("POST", "/profile", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestBS_SessionError(t *testing.T) {
	type Req struct {
		Nickname string `json:"nickname"`
	}

	provider := &mockProvider{err: errors.New("session not found")}
	session.SetDefaultProvider(provider)

	r := gin.New()
	r.POST("/profile", BS(func(ctx *gctx.Context, req Req, sess session.Session) (Result, error) {
		return Success("", nil), nil
	}))

	w := httptest.NewRecorder()
	body := `{"nickname":"test"}`
	req := httptest.NewRequest("POST", "/profile", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestBS_BindError(t *testing.T) {
	type Req struct {
		Nickname string `json:"nickname" binding:"required"`
	}

	provider := &mockProvider{sess: &mockSession{userId: "user123", data: make(map[string]any)}}
	session.SetDefaultProvider(provider)

	r := gin.New()
	r.POST("/profile", BS(func(ctx *gctx.Context, req Req, sess session.Session) (Result, error) {
		return Success("", nil), nil
	}))

	w := httptest.NewRecorder()
	body := `{}`
	req := httptest.NewRequest("POST", "/profile", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), `"code":10000`) {
		t.Errorf("body = %s, want code 10000 (InvalidParam)", w.Body.String())
	}
}

func TestW_InternalError_WithCustomCode(t *testing.T) {
	r := gin.New()
	r.GET("/error", W(func(ctx *gctx.Context) (Result, error) {
		// 返回自定义错误码，但 err != nil
		return Result{Code: 10001, Msg: ""}, errors.New("internal error")
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/error", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	// 应该使用调用方提供的 code，但 msg 应该用默认值
	if !strings.Contains(w.Body.String(), `"code":10001`) {
		t.Errorf("body = %s, want code 10001", w.Body.String())
	}
}
