package routes

import (
	"time"

	"github.com/Fazann/notedbook-backend/config"
	"github.com/Fazann/notedbook-backend/internal/attachments/handler"
	attachmentService "github.com/Fazann/notedbook-backend/internal/attachments/service"
	userHandler "github.com/Fazann/notedbook-backend/internal/users/handler"
	"github.com/Fazann/notedbook-backend/pkg/redis"
	"github.com/Fazann/notedbook-backend/shared/middleware"
	"github.com/gin-gonic/gin"
)

type handlers struct {
	attachmentHandler handler.AttachmentHandler
	authHandler       userHandler.AuthHandler
	userHandler       userHandler.UserHandler
}

func newHandlers() *handlers {
	return &handlers{
		attachmentHandler: *handler.NewAttachmentHandler(),
		authHandler:       *userHandler.NewAuthHandler(),
		userHandler:       *userHandler.NewUserHandler(attachmentService.NewAttachmentSvc()),
	}
}

func initAPIRoute(r *gin.Engine) {
	api := r.Group("/api/v1")

	h := newHandlers()
	rl := middleware.NewRateLimiter(redis.Hit)
	authRequired := middleware.Auth([]byte(config.JwtSecret))

	attachments := api.Group("/attachments")
	attachments.POST("", rl.LimitIP(100, time.Minute), h.attachmentHandler.UploadFile)
	attachments.GET("/:id", h.attachmentHandler.GetDetail)

	auth := api.Group("/auth")
	auth.POST("/register", rl.LimitIP(60, time.Minute), h.authHandler.Register)
	auth.POST("/login", rl.LimitIP(60, time.Minute), h.authHandler.Login)
	auth.POST("/refresh-token", rl.LimitIP(60, time.Minute), h.authHandler.RefreshToken)
	auth.GET("/profile", authRequired, h.userHandler.GetProfile)
	auth.PATCH("/profile", authRequired, h.userHandler.UpdateProfile)
}
