package middleware

import (
	"net/http"
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

// CORS allows any origin and header for now; tighten the origins before production.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", corsAllowMethods)
		h.Set("Access-Control-Allow-Headers", "*")
		h.Set("Access-Control-Max-Age", "86400")
		c.Next()
	}
}
