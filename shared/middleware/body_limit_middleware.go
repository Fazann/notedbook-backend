package middleware

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

const ctxRawBody = "raw_body"

// BodyLimit caps every request body at limit bytes. Reading past it fails with *http.MaxBytesError,
// which response.JsonErrRes answers with 413 REQUEST_TOO_LARGE. Routes that accept more (uploads)
// raise it with SetBodyLimit.
func BodyLimit(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(ctxRawBody, c.Request.Body)
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}

// SetBodyLimit replaces the BodyLimit cap for this request with limit. Call it before reading the body.
func SetBodyLimit(c *gin.Context, limit int64) {
	v, _ := c.Get(ctxRawBody)
	raw, ok := v.(io.ReadCloser)
	if !ok {
		raw = c.Request.Body
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, raw, limit)
}
