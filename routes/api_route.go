package routes

import (
	"time"

	"github.com/Fazann/notedbook-backend/config"
	"github.com/Fazann/notedbook-backend/internal/attachments/handler"
	attachmentService "github.com/Fazann/notedbook-backend/internal/attachments/service"
	expenseHandler "github.com/Fazann/notedbook-backend/internal/exspense/handler"
	planHandler "github.com/Fazann/notedbook-backend/internal/plans/handler"
	userHandler "github.com/Fazann/notedbook-backend/internal/users/handler"
	"github.com/Fazann/notedbook-backend/pkg/redis"
	"github.com/Fazann/notedbook-backend/shared/middleware"
	"github.com/gin-gonic/gin"
)

type handlers struct {
	attachmentHandler handler.AttachmentHandler
	authHandler       userHandler.AuthHandler
	userHandler       userHandler.UserHandler
	categoryHandler   expenseHandler.CategoryHandler
	expenseHandler    expenseHandler.ExpenseHandler
	planAreaHandler   planHandler.PlanAreaHandler
}

func newHandlers() *handlers {
	return &handlers{
		attachmentHandler: *handler.NewAttachmentHandler(),
		authHandler:       *userHandler.NewAuthHandler(),
		userHandler:       *userHandler.NewUserHandler(attachmentService.NewAttachmentSvc()),
		categoryHandler:   *expenseHandler.NewCategoryHandler(),
		expenseHandler:    *expenseHandler.NewExpenseHandler(),
		planAreaHandler:   *planHandler.NewPlanAreaHandler(),
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
	auth.POST("/mail-otp/request", rl.LimitIP(60, time.Minute), h.authHandler.RequestMailOTP)
	auth.POST("/mail-otp/verify", rl.LimitIP(60, time.Minute), h.authHandler.VerifyMailOTP)
	auth.POST("/reset-password", rl.LimitIP(60, time.Minute), h.authHandler.ResetPassword)
	auth.GET("/profile", authRequired, h.userHandler.GetProfile)
	auth.PATCH("/profile", authRequired, h.userHandler.UpdateProfile)
	auth.PUT("/change-password", authRequired, rl.LimitIP(60, time.Minute), h.userHandler.ChangePassword)

	categories := api.Group("/categories", authRequired)
	categories.GET("", h.categoryHandler.List)
	categories.GET("/:id", h.categoryHandler.GetDetail)
	categories.POST("", h.categoryHandler.Create)
	categories.PUT("/:id", h.categoryHandler.Update)
	categories.DELETE("/:id", h.categoryHandler.Delete)

	expenses := api.Group("/expenses", authRequired)
	expenses.GET("", h.expenseHandler.List)
	expenses.GET("/:id", h.expenseHandler.GetDetail)
	expenses.POST("", h.expenseHandler.Create)
	expenses.PUT("/:id", h.expenseHandler.Update)
	expenses.DELETE("/:id", h.expenseHandler.Delete)

	planAreas := api.Group("/plan-areas", authRequired)
	planAreas.GET("", h.planAreaHandler.List)
}
