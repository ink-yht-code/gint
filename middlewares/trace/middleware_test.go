package trace

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ink-yht-code/gint/gctx"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestMiddlewareUsesIncomingTraceID(t *testing.T) {
	r := gin.New()
	r.Use(Middleware())
	r.GET("/ping", func(c *gin.Context) {
		ctx := &gctx.Context{Context: c}
		if got := ctx.TraceId(); got != "trace-abc" {
			t.Fatalf("trace id in context = %s, want trace-abc", got)
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set(HeaderTraceID, "trace-abc")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if got := w.Header().Get(HeaderTraceID); got != "trace-abc" {
		t.Fatalf("response trace header = %s, want trace-abc", got)
	}
}

func TestMiddlewareGeneratesTraceIDWhenMissing(t *testing.T) {
	r := gin.New()
	r.Use(Middleware())
	r.GET("/ping", func(c *gin.Context) {
		ctx := &gctx.Context{Context: c}
		if ctx.TraceId() == "" {
			t.Fatal("trace id in context is empty")
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if got := w.Header().Get(HeaderTraceID); got == "" {
		t.Fatal("response trace header should not be empty")
	}
}

func TestMiddlewareExtractsUserContextFromHeaders(t *testing.T) {
	r := gin.New()
	r.Use(Middleware())
	r.GET("/me", func(c *gin.Context) {
		ctx := &gctx.Context{Context: c}
		uc := ctx.UserContext()
		if uc == nil {
			t.Fatal("user context is nil")
		}
		if uc.UserId != "u-1" || uc.Role != "admin" || uc.TenantId != "t-1" || uc.Username != "tom" || uc.Email != "tom@example.com" {
			t.Fatalf("unexpected user context: %+v", uc)
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set(HeaderUserID, "u-1")
	req.Header.Set(HeaderUserRole, "admin")
	req.Header.Set(HeaderTenantID, "t-1")
	req.Header.Set(HeaderUsername, "tom")
	req.Header.Set(HeaderEmail, "tom@example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestInjectGctxSetsHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	ctx := &gctx.Context{Context: c}
	ctx.SetTraceId("trace-1")
	ctx.SetUserContext(&gctx.UserContext{
		UserId:   "u-1",
		Role:     "admin",
		TenantId: "t-1",
		Username: "tom",
		Email:    "tom@example.com",
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	InjectGctx(ctx, req)

	if got := req.Header.Get(HeaderTraceID); got != "trace-1" {
		t.Fatalf("trace header = %s, want trace-1", got)
	}
	if got := req.Header.Get(HeaderUserID); got != "u-1" {
		t.Fatalf("user header = %s, want u-1", got)
	}
	if got := req.Header.Get(HeaderUserRole); got != "admin" {
		t.Fatalf("role header = %s, want admin", got)
	}
	if got := req.Header.Get(HeaderTenantID); got != "t-1" {
		t.Fatalf("tenant header = %s, want t-1", got)
	}
	if got := req.Header.Get(HeaderUsername); got != "tom" {
		t.Fatalf("username header = %s, want tom", got)
	}
	if got := req.Header.Get(HeaderEmail); got != "tom@example.com" {
		t.Fatalf("email header = %s, want tom@example.com", got)
	}
}
