package routes

import (
	"net/http"

	"github.com/Fazann/notedbook-backend/config"
	"github.com/Fazann/notedbook-backend/pkg/logger"
	"github.com/Fazann/notedbook-backend/shared/middleware"
	"github.com/gin-gonic/gin"
)

func InitRouter(r *gin.Engine) {
	// Gin's own proxy handling trusts every proxy by default, so X-Forwarded-For could be faked.
	// Turn it off and let RealIP pick the client IP from the entries our proxies added.
	if err := r.SetTrustedProxies(nil); err != nil {
		logger.Fatal("set trusted proxies failed", "err", err)
	}
	r.Use(middleware.RealIP(config.TrustedProxyHops), middleware.CORS(config.CorsAllowedOrigins))

	initSwaggerRoute(r)

	r.GET("ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, "Ok")
	})

	initAPIRoute(r)
}
