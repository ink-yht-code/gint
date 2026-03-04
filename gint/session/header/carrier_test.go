package header

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ink-yht-code/gint/gctx"
)

func TestCarrier_Extract_Bearer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name   string
		auth   string
		want   string
		header string
	}{
		{name: "no prefix", auth: "abc", want: "abc", header: "Authorization"},
		{name: "bearer", auth: "Bearer abc", want: "abc", header: "Authorization"},
		{name: "bearer lowercase", auth: "bearer abc", want: "abc", header: "Authorization"},
		{name: "bearer extra spaces", auth: "Bearer    abc", want: "abc", header: "Authorization"},
		{name: "leading/trailing spaces", auth: "   Bearer abc   ", want: "abc", header: "Authorization"},
		{name: "custom header", auth: "Bearer abc", want: "abc", header: "X-Auth"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			c.Request.Header.Set(tt.header, tt.auth)

			ctx := &gctx.Context{Context: c}
			carrier := NewCarrierWithHeader(tt.header)

			got := carrier.Extract(ctx)
			if got != tt.want {
				t.Fatalf("Extract() = %q, want %q", got, tt.want)
			}
		})
	}
}
