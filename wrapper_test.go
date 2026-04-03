// Copyright 2025 ink-yht-code
//
// Proprietary License
//
// IMPORTANT: This software is NOT open source.
// You may NOT use, copy, modify, merge, publish, distribute, sublicense,
// or sell copies of this file, in whole or in part, without prior written
// permission from the copyright holder.
//
// This software is provided "AS IS", without warranty of any kind.

package gint

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ink-yht-code/gint/gctx"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestW_Success(t *testing.T) {
	r := gin.New()
	r.GET("/test", W(func(ctx *gctx.Context) (Result, error) {
		return Result{Code: 0, Msg: "success", Data: "test"}, nil
	}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var result Result
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result.Code != 0 {
		t.Errorf("Expected code 0, got %d", result.Code)
	}
	if result.Msg != "success" {
		t.Errorf("Expected msg 'success', got '%s'", result.Msg)
	}
	if result.Data != "test" {
		t.Errorf("Expected data 'test', got '%v'", result.Data)
	}
}

func TestW_Error(t *testing.T) {
	r := gin.New()
	r.GET("/test", W(func(ctx *gctx.Context) (Result, error) {
		return Result{}, errors.New("test error")
	}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var result Result
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result.Code != CodeInternalError {
		t.Errorf("Expected code %d, got %d", CodeInternalError, result.Code)
	}
}

func TestW_NoResponse(t *testing.T) {
	r := gin.New()
	r.GET("/test", W(func(ctx *gctx.Context) (Result, error) {
		return Result{}, ErrNoResponse
	}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.Len() != 0 {
		t.Errorf("Expected empty body, got %s", w.Body.String())
	}
}

func TestW_Unauthorized(t *testing.T) {
	r := gin.New()
	r.GET("/test", W(func(ctx *gctx.Context) (Result, error) {
		return Result{}, ErrUnauthorized
	}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

type TestRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

func TestB_Success(t *testing.T) {
	r := gin.New()
	r.POST("/test", B(func(ctx *gctx.Context, req TestRequest) (Result, error) {
		return Result{Code: 0, Data: req}, nil
	}))

	reqBody := `{"name":"test","email":"test@example.com"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var result Result
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result.Code != 0 {
		t.Errorf("Expected code 0, got %d", result.Code)
	}
}

func TestB_InvalidParam(t *testing.T) {
	r := gin.New()
	r.POST("/test", B(func(ctx *gctx.Context, req TestRequest) (Result, error) {
		return Result{Code: 0, Data: req}, nil
	}))

	reqBody := `{"name":""}` // missing email and empty name
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var result Result
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result.Code != CodeInvalidParam {
		t.Errorf("Expected code %d, got %d", CodeInvalidParam, result.Code)
	}
}

func TestHandleResponse(t *testing.T) {
	tests := []struct {
		name           string
		result         Result
		err            error
		userId         string
		expectedStatus int
		expectedCode   int
	}{
		{
			name:           "Success",
			result:         Result{Code: 0, Msg: "success"},
			err:            nil,
			userId:         "",
			expectedStatus: http.StatusOK,
			expectedCode:   0,
		},
		{
			name:           "No Response",
			result:         Result{},
			err:            ErrNoResponse,
			userId:         "",
			expectedStatus: http.StatusOK,
			expectedCode:   0, // No response body
		},
		{
			name:           "Unauthorized",
			result:         Result{},
			err:            ErrUnauthorized,
			userId:         "",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   0, // No response body
		},
		{
			name:           "Internal Error",
			result:         Result{},
			err:            errors.New("internal error"),
			userId:         "user123",
			expectedStatus: http.StatusOK,
			expectedCode:   CodeInternalError,
		},
		{
			name:           "Business Error",
			result:         Result{Code: CodeInvalidParam, Msg: "invalid param"},
			err:            errors.New("business error"),
			userId:         "",
			expectedStatus: http.StatusOK,
			expectedCode:   CodeInvalidParam,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)

			handleResponse(c, tt.result, tt.err, tt.userId)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.err == ErrNoResponse || tt.err == ErrUnauthorized {
				// These cases don't return JSON response
				return
			}

			var result Result
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if result.Code != tt.expectedCode {
				t.Errorf("Expected code %d, got %d", tt.expectedCode, result.Code)
			}
		})
	}
}
