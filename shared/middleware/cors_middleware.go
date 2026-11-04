package middleware

import (
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

var corsAllowMethods = strings.Join([]string{
	http.MethodGet,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodOptions,
}, ", ")

// CORS lets the browser call the API from the allowed origins, e.g. "https://notebook.vercel.app" or
// a pattern with * for Vercel preview deploys ("https://notebook-*-myteam.vercel.app"; * does not
// match "/"). With no origins, every origin is allowed (local development).
// The API uses Bearer tokens, not cookies, so credentials are never allowed.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowAll := len(allowedOrigins) == 0
	return func(c *gin.Context) {
		h := c.Writer.Header()
		origin := c.GetHeader("Origin")
		switch {
		case allowAll:
			h.Set("Access-Control-Allow-Origin", "*")
		case origin != "" && originAllowed(origin, allowedOrigins):
			h.Set("Access-Control-Allow-Origin", origin)
			h.Add("Vary", "Origin")
		default:
			// No CORS headers: the browser blocks the response. Other clients are not affected.
			h.Add("Vary", "Origin")
		}
		h.Set("Access-Control-Allow-Methods", corsAllowMethods)
		h.Set("Access-Control-Allow-Headers", "*")
		// Lets the frontend read how long to wait after a 429.
		h.Set("Access-Control-Expose-Headers", "Retry-After")
		h.Set("Access-Control-Max-Age", "86400")
		// Answer the browser's preflight here; no route handles OPTIONS, so it would be a 404.
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func originAllowed(origin string, allowed []string) bool {
	for _, pattern := range allowed {
		if ok, _ := path.Match(pattern, origin); ok {
			return true
		}
	}
	return false
}
