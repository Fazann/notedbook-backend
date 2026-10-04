package routes

import (
	"net/http"

	"github.com/Fazann/notedbook-backend/shared/middleware"
	"github.com/gin-gonic/gin"
)

func InitRouter(r *gin.Engine) {
	r.Use(middleware.CORS())

	initSwaggerRoute(r)

	r.GET("ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, "Ok")
	})

	initAPIRoute(r)
}
