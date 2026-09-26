package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Fazann/notedbook-backend/pkg/i18n"
	token "github.com/Fazann/notedbook-backend/pkg/jwt"
	"github.com/Fazann/notedbook-backend/pkg/response"
	"github.com/gin-gonic/gin"
)

const (
	ctxUserID    = "auth_user_id"
	ctxSessionID = "auth_session_id"
)

// Auth requires a valid access token in "Authorization: Bearer <token>" and stores its user and
// session on the context (read them with CurrentUserID / CurrentSessionID).
func Auth(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || strings.TrimSpace(tokenStr) == "" {
			response.JsonErrRes(c, i18n.Unauthorized, http.StatusUnauthorized)
			c.Abort()
			return
		}

		claims, err := token.Parse(secret, token.Access, strings.TrimSpace(tokenStr))
		if err != nil {
			key := i18n.Unauthorized
			if errors.Is(err, token.ErrExpired) {
				key = i18n.TokenExpired
			}
			response.JsonErrRes(c, key, http.StatusUnauthorized)
			c.Abort()
			return
		}

		c.Set(ctxUserID, claims.UserID)
		c.Set(ctxSessionID, claims.SessionID)
		c.Next()
	}
}

// CurrentUserID returns the signed-in user's ID, or 0 on routes without Auth.
func CurrentUserID(c *gin.Context) uint {
	return c.GetUint(ctxUserID)
}

// CurrentSessionID returns the signed-in session's ID, or "" on routes without Auth.
func CurrentSessionID(c *gin.Context) string {
	return c.GetString(ctxSessionID)
}
