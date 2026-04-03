package gint

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ink-yht-code/gint/gctx"
	"github.com/ink-yht-code/gint/jwt"
	authmw "github.com/ink-yht-code/gint/middlewares/auth"
	tracemw "github.com/ink-yht-code/gint/middlewares/trace"
	"github.com/ink-yht-code/gint/session"
)

type integrationJWTManager struct {
	claims *jwt.Claims
}

func (m *integrationJWTManager) GenerateToken(claims jwt.Claims) (string, error) {
	return "", nil
}

func (m *integrationJWTManager) GenerateTokenPair(claims jwt.Claims) (*jwt.TokenPair, error) {
	return nil, nil
}

func (m *integrationJWTManager) VerifyToken(token string) (*jwt.Claims, error) {
	return m.claims, nil
}

func (m *integrationJWTManager) VerifyRefreshToken(token string) (*jwt.Claims, error) {
	return m.claims, nil
}

type integrationSession struct {
	claims *jwt.Claims
	uc     *gctx.UserContext
}

func (s *integrationSession) Set(ctx context.Context, key string, val any) error { return nil }
func (s *integrationSession) Get(ctx context.Context, key string) (any, error)   { return nil, nil }
func (s *integrationSession) Del(ctx context.Context, key string) error          { return nil }
func (s *integrationSession) Destroy(ctx context.Context) error                  { return nil }
func (s *integrationSession) Claims() *jwt.Claims                                { return s.claims }
func (s *integrationSession) Refresh(ctx context.Context) error                  { return nil }
func (s *integrationSession) UserContext(ctx context.Context) (*gctx.UserContext, error) {
	return s.uc, nil
}

type integrationProvider struct {
	sess session.Session
}

func (p *integrationProvider) NewSession(ctx *gctx.Context, userId string, jwtData map[string]string, sessData map[string]any) (session.Session, error) {
	return p.sess, nil
}

func (p *integrationProvider) Get(ctx *gctx.Context) (session.Session, error) {
	return p.sess, nil
}

func (p *integrationProvider) Destroy(ctx *gctx.Context) error    { return nil }
func (p *integrationProvider) RenewToken(ctx *gctx.Context) error { return nil }

func TestTraceAuthWrapperIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	manager := &integrationJWTManager{
		claims: &jwt.Claims{UserId: "u-100"},
	}
	sess := &integrationSession{
		claims: &jwt.Claims{UserId: "u-100"},
		uc: &gctx.UserContext{
			UserId: "u-100",
			Role:   "admin",
		},
	}
	provider := &integrationProvider{sess: sess}

	r := gin.New()
	r.Use(tracemw.Middleware())
	r.Use(authmw.Middleware(authmw.Config{
		JWTManager:      manager,
		TokenExtractor:  authmw.HeaderExtractor("X-Token"),
		SessionProvider: provider,
	}))
	r.GET("/me", W(func(ctx *gctx.Context) (Result, error) {
		return Success("", gin.H{
			"user_id":  ctx.UserId(),
			"role":     ctx.Role(),
			"trace_id": ctx.TraceId(),
		}), nil
	}))

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("X-Token", "token")
	req.Header.Set("X-Trace-ID", "trace-123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Code int `json:"code"`
		Data struct {
			UserID  string `json:"user_id"`
			Role    string `json:"role"`
			TraceID string `json:"trace_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}

	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	if resp.Data.UserID != "u-100" {
		t.Fatalf("user_id = %s, want u-100", resp.Data.UserID)
	}
	if resp.Data.Role != "admin" {
		t.Fatalf("role = %s, want admin", resp.Data.Role)
	}
	if resp.Data.TraceID != "trace-123" {
		t.Fatalf("trace_id = %s, want trace-123", resp.Data.TraceID)
	}
}
