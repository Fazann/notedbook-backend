package routes

import (
	"time"

	"github.com/Fazann/notedbook-backend/config"
	"github.com/Fazann/notedbook-backend/internal/attachments/handler"
	attachmentService "github.com/Fazann/notedbook-backend/internal/attachments/service"
	boardHandler "github.com/Fazann/notedbook-backend/internal/boards/handler"
	expenseHandler "github.com/Fazann/notedbook-backend/internal/exspense/handler"
	planHandler "github.com/Fazann/notedbook-backend/internal/plans/handler"
	scheduleHandler "github.com/Fazann/notedbook-backend/internal/schedule/handler"
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
	planHandler       planHandler.PlanHandler
	planStepHandler   planHandler.PlanStepHandler
	scheduleHandler   scheduleHandler.ScheduleHandler
	boardHandler      boardHandler.BoardHandler
}

func newHandlers() *handlers {
	return &handlers{
		attachmentHandler: *handler.NewAttachmentHandler(),
		authHandler:       *userHandler.NewAuthHandler(),
		userHandler:       *userHandler.NewUserHandler(attachmentService.NewAttachmentSvc()),
		categoryHandler:   *expenseHandler.NewCategoryHandler(),
		expenseHandler:    *expenseHandler.NewExpenseHandler(),
		planAreaHandler:   *planHandler.NewPlanAreaHandler(),
		planHandler:       *planHandler.NewPlanHandler(),
		planStepHandler:   *planHandler.NewPlanStepHandler(),
		scheduleHandler:   *scheduleHandler.NewScheduleHandler(),
		boardHandler:      *boardHandler.NewBoardHandler(),
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
	categories.GET("/all", h.categoryHandler.ListAll)
	categories.GET("/:id", h.categoryHandler.GetDetail)
	categories.POST("", h.categoryHandler.Create)
	categories.PUT("/:id", h.categoryHandler.Update)
	categories.DELETE("/:id", h.categoryHandler.Delete)

	expenses := api.Group("/expenses", authRequired)
	expenses.GET("", h.expenseHandler.List)
	expenses.GET("/summary", h.expenseHandler.Summary)
	expenses.GET("/:id", h.expenseHandler.GetDetail)
	expenses.POST("", h.expenseHandler.Create)
	expenses.PUT("/:id", h.expenseHandler.Update)
	expenses.DELETE("/:id", h.expenseHandler.Delete)

	planAreas := api.Group("/plan-areas", authRequired)
	planAreas.GET("", h.planAreaHandler.List)
	planAreas.DELETE("/cache", rl.LimitIP(10, time.Minute), h.planAreaHandler.ResetCache)

	plans := api.Group("/plans", authRequired)
	plans.GET("", h.planHandler.List)
	plans.GET("/stats", h.planHandler.Stats)
	plans.GET("/:id", h.planHandler.GetDetail)
	plans.POST("", h.planHandler.Create)
	plans.PUT("/:id", h.planHandler.Update)
	plans.PATCH("/:id/status", h.planHandler.UpdateStatus)
	plans.DELETE("/:id", h.planHandler.Delete)
	plans.POST("/:id/steps", h.planStepHandler.Create)

	steps := api.Group("/steps", authRequired)
	steps.PUT("/:id", h.planStepHandler.Update)
	steps.PATCH("/:id/toggle", h.planStepHandler.Toggle)
	steps.PATCH("/:id/move", h.planStepHandler.Move)
	steps.DELETE("/:id", h.planStepHandler.Delete)

	schedule := api.Group("/schedule", authRequired)
	schedule.GET("/occurrences", h.scheduleHandler.ListOccurrences)
	schedule.GET("/summary", h.scheduleHandler.GetWeekSummary)
	schedule.GET("/activities/:id", h.scheduleHandler.GetActivity)
	schedule.POST("/activities", h.scheduleHandler.CreateActivity)
	schedule.PUT("/activities/:id", h.scheduleHandler.UpdateActivity)
	schedule.DELETE("/activities/:id", h.scheduleHandler.DeleteActivity)

	boards := api.Group("/boards", authRequired)
	boards.GET("", h.boardHandler.List)

	cards := api.Group("/cards", authRequired)
	cards.GET("/due", h.boardHandler.ListDueCards)
}
