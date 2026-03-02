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

package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if len(config.AllowOrigins) == 0 || config.AllowOrigins[0] != "*" {
		t.Errorf("DefaultConfig().AllowOrigins = %v, want ['*']", config.AllowOrigins)
	}
	if config.AllowCredentials {
		t.Errorf("DefaultConfig().AllowCredentials = %v, want false", config.AllowCredentials)
	}
	if config.MaxAge != 86400 {
		t.Errorf("DefaultConfig().MaxAge = %d, want 86400", config.MaxAge)
	}
}

func TestNew_EmptyConfig(t *testing.T) {
	// 空配置应该使用默认配置
	middleware := New(Config{})
	if middleware == nil {
		t.Error("New(Config{}) returned nil")
	}
}

func TestDefault(t *testing.T) {
	middleware := Default()
	if middleware == nil {
		t.Error("Default() returned nil")
	}
}

func TestCORS_AllowAllOrigins_NoCredentials(t *testing.T) {
	r := gin.New()
	r.Use(New(Config{
		AllowOrigins:     []string{"*"},
		AllowCredentials: false,
	}))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Access-Control-Allow-Origin = %s, want '*'", w.Header().Get("Access-Control-Allow-Origin"))
	}
	// 不应该设置 Vary 头
	if w.Header().Get("Vary") == "Origin" {
		t.Errorf("Vary header should not be set when AllowCredentials=false and AllowOrigins='*'")
	}
}

func TestCORS_AllowAllOrigins_WithCredentials(t *testing.T) {
	r := gin.New()
	r.Use(New(Config{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
	}))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	origin := "http://localhost:3000"
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", origin)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	// 当 AllowCredentials=true 时，应该反射 Origin 而不是返回 "*"
	if w.Header().Get("Access-Control-Allow-Origin") != origin {
		t.Errorf("Access-Control-Allow-Origin = %s, want %s", w.Header().Get("Access-Control-Allow-Origin"), origin)
	}
	// 应该设置 Vary: Origin
	if w.Header().Get("Vary") != "Origin" {
		t.Errorf("Vary = %s, want 'Origin'", w.Header().Get("Vary"))
	}
	// 应该设置 Allow-Credentials
	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("Access-Control-Allow-Credentials = %s, want 'true'", w.Header().Get("Access-Control-Allow-Credentials"))
	}
}

func TestCORS_AllowAllOrigins_WithCredentials_NoOriginHeader(t *testing.T) {
	r := gin.New()
	r.Use(New(Config{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
	}))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	// 不设置 Origin header
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	// 没有 Origin header 时，应该返回 "*"
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Access-Control-Allow-Origin = %s, want '*'", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_SpecificOrigins_Allowed(t *testing.T) {
	r := gin.New()
	allowedOrigins := []string{"http://localhost:3000", "https://example.com"}
	r.Use(New(Config{
		AllowOrigins: allowedOrigins,
	}))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	origin := "http://localhost:3000"
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", origin)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != origin {
		t.Errorf("Access-Control-Allow-Origin = %s, want %s", w.Header().Get("Access-Control-Allow-Origin"), origin)
	}
	if w.Header().Get("Vary") != "Origin" {
		t.Errorf("Vary = %s, want 'Origin'", w.Header().Get("Vary"))
	}
}

func TestCORS_SpecificOrigins_NotAllowed(t *testing.T) {
	r := gin.New()
	allowedOrigins := []string{"http://localhost:3000", "https://example.com"}
	r.Use(New(Config{
		AllowOrigins: allowedOrigins,
	}))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	origin := "http://malicious.com"
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", origin)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	// 不在允许列表中的 origin 不应该被设置
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("Access-Control-Allow-Origin = %s, want empty (origin not allowed)", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_PreflightRequest(t *testing.T) {
	r := gin.New()
	r.Use(New(Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: false,
		MaxAge:           3600,
	}))
	r.POST("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	r.ServeHTTP(w, req)

	// 预检请求应该返回 204
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Access-Control-Allow-Origin = %s, want '*'", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Access-Control-Allow-Methods") != "GET, POST, PUT" {
		t.Errorf("Access-Control-Allow-Methods = %s, want 'GET, POST, PUT'", w.Header().Get("Access-Control-Allow-Methods"))
	}
	if w.Header().Get("Access-Control-Allow-Headers") != "Content-Type, Authorization" {
		t.Errorf("Access-Control-Allow-Headers = %s, want 'Content-Type, Authorization'", w.Header().Get("Access-Control-Allow-Headers"))
	}
	if w.Header().Get("Access-Control-Max-Age") != "3600" {
		t.Errorf("Access-Control-Max-Age = %s, want '3600'", w.Header().Get("Access-Control-Max-Age"))
	}
}

func TestCORS_PreflightRequest_WithCredentials(t *testing.T) {
	r := gin.New()
	r.Use(New(Config{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
		MaxAge:           7200,
	}))
	r.POST("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	origin := "http://localhost:3000"
	w := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", origin)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
	// 应该反射 Origin
	if w.Header().Get("Access-Control-Allow-Origin") != origin {
		t.Errorf("Access-Control-Allow-Origin = %s, want %s", w.Header().Get("Access-Control-Allow-Origin"), origin)
	}
	if w.Header().Get("Vary") != "Origin" {
		t.Errorf("Vary = %s, want 'Origin'", w.Header().Get("Vary"))
	}
	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("Access-Control-Allow-Credentials = %s, want 'true'", w.Header().Get("Access-Control-Allow-Credentials"))
	}
}

func TestCORS_ExposeHeaders(t *testing.T) {
	r := gin.New()
	r.Use(New(Config{
		AllowOrigins:  []string{"*"},
		ExposeHeaders: []string{"X-Custom-Header", "X-Another-Header"},
	}))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Expose-Headers") != "X-Custom-Header, X-Another-Header" {
		t.Errorf("Access-Control-Expose-Headers = %s, want 'X-Custom-Header, X-Another-Header'", w.Header().Get("Access-Control-Expose-Headers"))
	}
}

func TestCORS_NoOriginHeader(t *testing.T) {
	r := gin.New()
	r.Use(New(Config{
		AllowOrigins: []string{"http://localhost:3000"},
	}))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	// 不设置 Origin header
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	// 没有 Origin header 时，不应该设置 Access-Control-Allow-Origin
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("Access-Control-Allow-Origin = %s, want empty (no Origin header)", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_AllowMethods(t *testing.T) {
	r := gin.New()
	r.Use(New(Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST"},
	}))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Methods") != "GET, POST" {
		t.Errorf("Access-Control-Allow-Methods = %s, want 'GET, POST'", w.Header().Get("Access-Control-Allow-Methods"))
	}
}

func TestCORS_AllowHeaders(t *testing.T) {
	r := gin.New()
	r.Use(New(Config{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"Content-Type", "Authorization"},
	}))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Headers") != "Content-Type, Authorization" {
		t.Errorf("Access-Control-Allow-Headers = %s, want 'Content-Type, Authorization'", w.Header().Get("Access-Control-Allow-Headers"))
	}
}

func TestCORS_MaxAge(t *testing.T) {
	r := gin.New()
	r.Use(New(Config{
		AllowOrigins: []string{"*"},
		MaxAge:       86400,
	}))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Max-Age") != "86400" {
		t.Errorf("Access-Control-Max-Age = %s, want '86400'", w.Header().Get("Access-Control-Max-Age"))
	}
}

func TestCORS_MaxAge_Zero(t *testing.T) {
	r := gin.New()
	r.Use(New(Config{
		AllowOrigins: []string{"*"},
		MaxAge:       0,
	}))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	r.ServeHTTP(w, req)

	// MaxAge=0 时不应该设置 Access-Control-Max-Age
	if w.Header().Get("Access-Control-Max-Age") != "" {
		t.Errorf("Access-Control-Max-Age = %s, want empty (MaxAge=0)", w.Header().Get("Access-Control-Max-Age"))
	}
}

// TestCORS_VaryHeader_MultipleOrigins 测试多个允许源时 Vary 头的设置
func TestCORS_VaryHeader_MultipleOrigins(t *testing.T) {
	r := gin.New()
	allowedOrigins := []string{"http://localhost:3000", "https://example.com", "https://api.example.com"}
	r.Use(New(Config{
		AllowOrigins: allowedOrigins,
	}))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	tests := []struct {
		name         string
		origin       string
		expectOrigin string
		expectVary   bool
	}{
		{
			name:         "allowed origin 1",
			origin:       "http://localhost:3000",
			expectOrigin: "http://localhost:3000",
			expectVary:   true,
		},
		{
			name:         "allowed origin 2",
			origin:       "https://example.com",
			expectOrigin: "https://example.com",
			expectVary:   true,
		},
		{
			name:         "not allowed origin",
			origin:       "http://malicious.com",
			expectOrigin: "",
			expectVary:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			r.ServeHTTP(w, req)

			gotOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if gotOrigin != tt.expectOrigin {
				t.Errorf("Access-Control-Allow-Origin = %s, want %s", gotOrigin, tt.expectOrigin)
			}

			gotVary := w.Header().Get("Vary") == "Origin"
			if gotVary != tt.expectVary {
				t.Errorf("Vary header set = %v, want %v", gotVary, tt.expectVary)
			}
		})
	}
}

// TestCORS_CredentialsWithSpecificOrigins 测试 AllowCredentials 与特定源的组合
func TestCORS_CredentialsWithSpecificOrigins(t *testing.T) {
	r := gin.New()
	allowedOrigins := []string{"http://localhost:3000", "https://example.com"}
	r.Use(New(Config{
		AllowOrigins:     allowedOrigins,
		AllowCredentials: true,
	}))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	origin := "http://localhost:3000"
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", origin)
	r.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != origin {
		t.Errorf("Access-Control-Allow-Origin = %s, want %s", w.Header().Get("Access-Control-Allow-Origin"), origin)
	}
	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("Access-Control-Allow-Credentials = %s, want 'true'", w.Header().Get("Access-Control-Allow-Credentials"))
	}
	if w.Header().Get("Vary") != "Origin" {
		t.Errorf("Vary = %s, want 'Origin'", w.Header().Get("Vary"))
	}
}
