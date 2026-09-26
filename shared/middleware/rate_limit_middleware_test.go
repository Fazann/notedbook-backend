package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/gin-gonic/gin"
)

// fakeHit counts per key in memory, like redis.Hit without expiry.
func fakeHit(err error) (HitFunc, map[string]int64) {
	counts := map[string]int64{}
	return func(_ context.Context, key string, max int64, window time.Duration) (bool, time.Duration, error) {
		if err != nil {
			return false, 0, err
		}
		counts[key]++
		return counts[key] <= max, window, nil
	}, counts
}

func TestRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		hitErr     error
		requests   int
		ip         string
		wantStatus int
	}{
		{name: "under limit", requests: 3, ip: "1.1.1.1", wantStatus: http.StatusOK},
		{name: "over limit", requests: 4, ip: "1.1.1.1", wantStatus: http.StatusTooManyRequests},
		{name: "redis error fails open", hitErr: errors.New("redis down"), requests: 10, ip: "1.1.1.1", wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hit, _ := fakeHit(tt.hitErr)
			r := gin.New()
			r.GET("/ping", NewRateLimiter(hit).LimitIP(3, 90*time.Second), func(c *gin.Context) { c.Status(http.StatusOK) })

			var w *httptest.ResponseRecorder
			for range tt.requests {
				req := httptest.NewRequest(http.MethodGet, "/ping", nil)
				req.RemoteAddr = tt.ip + ":1234"
				w = httptest.NewRecorder()
				r.ServeHTTP(w, req)
			}

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantStatus != http.StatusTooManyRequests {
				return
			}
			if got := w.Header().Get("Retry-After"); got != "90" {
				t.Fatalf("Retry-After = %q, want 90", got)
			}
			var body struct{ Code string }
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body.Code != string(i18n.TooManyRequests) {
				t.Fatalf("code = %q, want %q", body.Code, i18n.TooManyRequests)
			}
		})
	}
}

func TestRateLimitKeyPerIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hit, counts := fakeHit(nil)
	r := gin.New()
	r.GET("/ping", NewRateLimiter(hit).LimitIP(1, time.Minute), func(c *gin.Context) { c.Status(http.StatusOK) })

	for _, ip := range []string{"1.1.1.1", "2.2.2.2"} {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.RemoteAddr = ip + ":1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("ip %s status = %d, want 200", ip, w.Code)
		}
	}
	if len(counts) != 2 {
		t.Fatalf("keys = %v, want one per IP", counts)
	}
}
