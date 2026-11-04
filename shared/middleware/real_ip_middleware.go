package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// RealIP sets the request's remote address to the client IP that the trusted proxies saw, so that
// c.ClientIP() (used by rate limits and logs) returns it. Use it with engine.SetTrustedProxies(nil).
//
// hops is the number of proxies in front of the API that append to X-Forwarded-For: the client IP is
// the entry hops places from the right. Entries further left come from the client and can be faked,
// so they are never used. Cloud Run (its front end appends the client IP) is 1; a Google load
// balancer in front of Cloud Run adds one more. With hops 0, or too few entries, nothing changes.
func RealIP(hops int) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ip := forwardedIP(c.Request, hops); ip != "" {
			c.Request.RemoteAddr = net.JoinHostPort(ip, "0")
		}
		c.Next()
	}
}

func forwardedIP(r *http.Request, hops int) string {
	if hops <= 0 {
		return ""
	}
	var entries []string
	for _, h := range r.Header.Values("X-Forwarded-For") {
		entries = append(entries, strings.Split(h, ",")...)
	}
	if len(entries) < hops {
		return ""
	}
	ip := net.ParseIP(strings.TrimSpace(entries[len(entries)-hops]))
	if ip == nil {
		return ""
	}
	return ip.String()
}
