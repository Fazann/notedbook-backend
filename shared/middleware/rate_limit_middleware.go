package middleware

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/logger"
	"github.com/Fazann/notedbook-backend/pkg/response"
	"github.com/gin-gonic/gin"
)

// HitFunc counts one request on key and reports whether it is within max for window, and how long
// the caller must wait when it is not (see redis.Hit and redis.HitBackoff).
type HitFunc func(ctx context.Context, key string, max int64, window time.Duration) (bool, time.Duration, error)

type RateLimiter struct {
	hit HitFunc
}

// NewRateLimiter counts requests with hit. A nil hit turns rate limiting off (every request passes).
func NewRateLimiter(hit HitFunc) *RateLimiter {
	return &RateLimiter{hit: hit}
}

// LimitIP allows max requests per client IP to this route (method + path pattern) within window.
// Over the limit it answers 429 TOO_MANY_REQUESTS with a Retry-After header.
//
//	auth.POST("/register", rl.LimitIP(10, time.Minute), h.authHandler.Register)
func (l *RateLimiter) LimitIP(max int64, window time.Duration) gin.HandlerFunc {
	mustBePositive(max, window)
	return func(c *gin.Context) {
		l.check(c, "ip:"+c.ClientIP(), max, window)
	}
}

// LimitUser allows readMax GET/HEAD requests, or writeMax other requests, per signed-in user to this
// route within window. Put it after Auth; without a user it falls back to the client IP.
//
//	api.Group("/expenses", authRequired, rl.LimitUser(60, 20, time.Minute))
func (l *RateLimiter) LimitUser(readMax, writeMax int64, window time.Duration) gin.HandlerFunc {
	mustBePositive(readMax, window)
	mustBePositive(writeMax, window)
	return func(c *gin.Context) {
		limit := writeMax
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
			limit = readMax
		}
		subject := "ip:" + c.ClientIP()
		if id := CurrentUserID(c); id != 0 {
			subject = fmt.Sprintf("user:%d", id)
		}
		l.check(c, subject, limit, window)
	}
}

func (l *RateLimiter) check(c *gin.Context, subject string, limit int64, window time.Duration) {
	if l.hit == nil {
		c.Next()
		return
	}
	key := fmt.Sprintf("ratelimit:%s:%s %s", subject, c.Request.Method, c.FullPath())
	allowed, retryAfter, err := l.hit(c.Request.Context(), key, limit, window)
	if err != nil {
		// Fail open so a Redis outage does not take the API down with it.
		logger.Error("rate limit check failed", "err", err, "path", c.FullPath())
		c.Next()
		return
	}

	if !allowed {
		seconds := max(int(math.Ceil(retryAfter.Seconds())), 1)
		c.Header("Retry-After", strconv.Itoa(seconds))
		response.JsonErrRes(c, i18n.Err(i18n.TooManyRequests, i18n.Params{"seconds": seconds}), http.StatusTooManyRequests)
		c.Abort()
		return
	}

	c.Next()
}

func mustBePositive(max int64, window time.Duration) {
	if max <= 0 || window <= 0 {
		panic("middleware: rate limit max and window must be positive")
	}
}
