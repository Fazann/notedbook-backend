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

// HitFunc counts one request on key and reports whether it is within max for window (see redis.Hit).
type HitFunc func(ctx context.Context, key string, max int64, window time.Duration) (bool, time.Duration, error)

type RateLimiter struct {
	hit HitFunc
}

func NewRateLimiter(hit HitFunc) *RateLimiter {
	return &RateLimiter{hit: hit}
}

// Limit allows max requests per client IP to the route within window. Once over the limit, the IP is
// blocked for window and gets 429 TOO_MANY_REQUESTS with a Retry-After header.
//
//	auth.POST("/register", rl.Limit(10, time.Minute), h.authHandler.Register)
func (l *RateLimiter) LimitIP(max int64, window time.Duration) gin.HandlerFunc {
	if max <= 0 || window <= 0 {
		panic("middleware: rate limit max and window must be positive")
	}

	return func(c *gin.Context) {
		key := fmt.Sprintf("ratelimit:ip:%s", c.ClientIP())
		allowed, retryAfter, err := l.hit(c.Request.Context(), key, max, window)
		if err != nil {
			// Fail open so a Redis outage does not take the API down with it.
			logger.Error("rate limit check failed", "err", err, "path", c.FullPath())
			c.Next()
			return
		}

		if !allowed {
			seconds := int(math.Ceil(retryAfter.Seconds()))
			if seconds < 1 {
				seconds = 1
			}
			c.Header("Retry-After", strconv.Itoa(seconds))
			response.JsonErrRes(c, i18n.Err(i18n.TooManyRequests, i18n.Params{"seconds": seconds}), http.StatusTooManyRequests)
			c.Abort()
			return
		}

		c.Next()
	}
}
