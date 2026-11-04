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

func serve(r *gin.Engine, method, path, ip string) int {
	req := httptest.NewRequest(method, path, nil)
	req.RemoteAddr = ip + ":1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func TestRateLimitKeyPerRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hit, _ := fakeHit(nil)
	limit := NewRateLimiter(hit).LimitIP(1, time.Minute)
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }
	r := gin.New()
	r.GET("/a", limit, ok)
	r.GET("/b/:id", limit, ok)
	r.POST("/b/:id", limit, ok)

	for _, req := range [][2]string{{http.MethodGet, "/a"}, {http.MethodGet, "/b/1"}, {http.MethodPost, "/b/1"}} {
		if got := serve(r, req[0], req[1], "1.1.1.1"); got != http.StatusOK {
			t.Fatalf("%s %s status = %d, want 200 (own counter per route)", req[0], req[1], got)
		}
	}
	// Same pattern, another ID: shares the GET /b/:id counter.
	if got := serve(r, http.MethodGet, "/b/2", "1.1.1.1"); got != http.StatusTooManyRequests {
		t.Fatalf("GET /b/2 status = %d, want 429", got)
	}
}

func TestLimitUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		method   string
		userID   uint
		requests int
		want     int
	}{
		{name: "reads under read limit", method: http.MethodGet, userID: 1, requests: 3, want: http.StatusOK},
		{name: "reads over read limit", method: http.MethodGet, userID: 1, requests: 4, want: http.StatusTooManyRequests},
		{name: "writes under write limit", method: http.MethodPost, userID: 1, requests: 2, want: http.StatusOK},
		{name: "writes over write limit", method: http.MethodPost, userID: 1, requests: 3, want: http.StatusTooManyRequests},
		{name: "no user falls back to IP", method: http.MethodGet, requests: 4, want: http.StatusTooManyRequests},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hit, counts := fakeHit(nil)
			setUser := func(c *gin.Context) {
				if tt.userID != 0 {
					c.Set(ctxUserID, tt.userID)
				}
			}
			r := gin.New()
			r.Handle(tt.method, "/x", setUser, NewRateLimiter(hit).LimitUser(3, 2, time.Minute), func(c *gin.Context) { c.Status(http.StatusOK) })

			var got int
			for range tt.requests {
				got = serve(r, tt.method, "/x", "1.1.1.1")
			}
			if got != tt.want {
				t.Fatalf("status = %d, want %d", got, tt.want)
			}

			wantKey := "ratelimit:ip:1.1.1.1:" + tt.method + " /x"
			if tt.userID != 0 {
				wantKey = "ratelimit:user:1:" + tt.method + " /x"
			}
			if _, ok := counts[wantKey]; !ok || len(counts) != 1 {
				t.Fatalf("keys = %v, want only %q", counts, wantKey)
			}
		})
	}
}

func TestLimitUserKeyPerUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hit, _ := fakeHit(nil)
	limit := NewRateLimiter(hit).LimitUser(1, 1, time.Minute)
	r := gin.New()
	r.GET("/x", func(c *gin.Context) {
		if c.Query("u") == "2" {
			c.Set(ctxUserID, uint(2))
		} else {
			c.Set(ctxUserID, uint(1))
		}
	}, limit, func(c *gin.Context) { c.Status(http.StatusOK) })

	// Same IP, different users: each has its own counter.
	for _, path := range []string{"/x?u=1", "/x?u=2"} {
		if got := serve(r, http.MethodGet, path, "1.1.1.1"); got != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", path, got)
		}
	}
}

func TestRateLimitDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRateLimiter(nil)
	r := gin.New()
	r.GET("/x", rl.LimitIP(1, time.Minute), rl.LimitUser(1, 1, time.Minute), func(c *gin.Context) { c.Status(http.StatusOK) })

	for range 3 {
		if got := serve(r, http.MethodGet, "/x", "1.1.1.1"); got != http.StatusOK {
			t.Fatalf("status = %d, want 200 with rate limiting off", got)
		}
	}
}
