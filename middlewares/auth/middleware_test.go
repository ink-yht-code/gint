package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ink-yht-code/gint/jwt"
)

type mockJWTManager struct {
	claims *jwt.Claims
	err    error
}

func (m *mockJWTManager) GenerateToken(claims jwt.Claims) (string, error) { return "", nil }
func (m *mockJWTManager) GenerateTokenPair(claims jwt.Claims) (*jwt.TokenPair, error) {
	return nil, nil
}
func (m *mockJWTManager) VerifyToken(token string) (*jwt.Claims, error) { return m.claims, m.err }
func (m *mockJWTManager) VerifyRefreshToken(token string) (*jwt.Claims, error) {
	return m.claims, m.err
}

func TestMiddlewareReturns500WhenJWTManagerMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Middleware(Config{}))
	r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestGatewayMiddlewareInjectsUserHeadersToRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GatewayMiddleware(Config{
		JWTManager: &mockJWTManager{
			claims: &jwt.Claims{UserId: "u-1"},
		},
	}))

	r.GET("/check", func(c *gin.Context) {
		if got := c.Request.Header.Get("X-User-ID"); got != "u-1" {
			t.Fatalf("X-User-ID in request = %s, want u-1", got)
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/check", nil)
	req.Header.Set("Authorization", "Bearer t")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestOptionalMiddlewareSkipsWhenJWTManagerMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	called := false
	r.Use(OptionalMiddleware(Config{}))
	r.GET("/ok", func(c *gin.Context) {
		called = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Header.Set("Authorization", "Bearer t")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !called {
		t.Fatal("handler not called, want called")
	}
}
