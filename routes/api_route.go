package routes

import (
	"time"

	"github.com/Fazann/notedbook-backend/config"
	"github.com/Fazann/notedbook-backend/internal/attachments/handler"
	attachmentService "github.com/Fazann/notedbook-backend/internal/attachments/service"
	boardHandler "github.com/Fazann/notedbook-backend/internal/boards/handler"
	calendarHandler "github.com/Fazann/notedbook-backend/internal/calendar/handler"
	exchangeHandler "github.com/Fazann/notedbook-backend/internal/exchange/handler"
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
	budgetHandler     expenseHandler.BudgetHandler
	currencyHandler   expenseHandler.CurrencySettingHandler
	planAreaHandler   planHandler.PlanAreaHandler
	planHandler       planHandler.PlanHandler
	planStepHandler   planHandler.PlanStepHandler
	scheduleHandler   scheduleHandler.ScheduleHandler
	boardHandler      boardHandler.BoardHandler
	holidayHandler    calendarHandler.HolidayHandler
	exchangeHandler   exchangeHandler.ExchangeRateHandler
}

func newHandlers() *handlers {
	return &handlers{
		attachmentHandler: *handler.NewAttachmentHandler(),
		authHandler:       *userHandler.NewAuthHandler(),
		userHandler:       *userHandler.NewUserHandler(attachmentService.NewAttachmentSvc()),
		categoryHandler:   *expenseHandler.NewCategoryHandler(),
		expenseHandler:    *expenseHandler.NewExpenseHandler(),
		budgetHandler:     *expenseHandler.NewBudgetHandler(),
		currencyHandler:   *expenseHandler.NewCurrencySettingHandler(),
		planAreaHandler:   *planHandler.NewPlanAreaHandler(),
		planHandler:       *planHandler.NewPlanHandler(),
		planStepHandler:   *planHandler.NewPlanStepHandler(),
		scheduleHandler:   *scheduleHandler.NewScheduleHandler(),
		boardHandler:      *boardHandler.NewBoardHandler(),
		holidayHandler:    *calendarHandler.NewHolidayHandler(),
		exchangeHandler:   *exchangeHandler.NewExchangeRateHandler(),
	}
}

func initAPIRoute(r *gin.Engine) {
	api := r.Group("/api/v1")

	h := newHandlers()
	// Each block doubles the last one (1, 2, 4 ... min) up to RATE_LIMIT_MAX_BLOCK; see config for the env vars.
	rl := middleware.NewRateLimiter(redis.HitBackoff(
		time.Duration(config.RateLimitMaxBlock)*time.Minute,
		time.Duration(config.RateLimitStrikesTTL)*time.Minute,
	))
	if !config.RateLimitEnabled {
		rl = middleware.NewRateLimiter(nil)
	}
	authRequired := middleware.Auth([]byte(config.JwtSecret))
	// Default limits per route, for routes without a stricter one of their own.
	ipLimit := rl.LimitIP(int64(config.RateLimitIP), time.Minute)
	userLimit := rl.LimitUser(int64(config.RateLimitUserReads), int64(config.RateLimitUserWrites), time.Minute)
	signedIn := []gin.HandlerFunc{authRequired, ipLimit, userLimit}

	attachments := api.Group("/attachments", signedIn...)
	attachments.POST("", h.attachmentHandler.UploadFile)
	attachments.GET("/:id", h.attachmentHandler.GetDetail)

	auth := api.Group("/auth")
	auth.POST("/register", rl.LimitIP(60, time.Minute), h.authHandler.Register)
	auth.POST("/login", rl.LimitIP(60, time.Minute), h.authHandler.Login)
	auth.POST("/refresh-token", rl.LimitIP(60, time.Minute), h.authHandler.RefreshToken)
	auth.POST("/mail-otp/request", rl.LimitIP(60, time.Minute), h.authHandler.RequestMailOTP)
	auth.POST("/mail-otp/verify", rl.LimitIP(60, time.Minute), h.authHandler.VerifyMailOTP)
	auth.POST("/reset-password", rl.LimitIP(60, time.Minute), h.authHandler.ResetPassword)
	profile := auth.Group("/profile", signedIn...)
	profile.GET("", h.userHandler.GetProfile)
	profile.PATCH("", h.userHandler.UpdateProfile)
	auth.PUT("/change-password", authRequired, rl.LimitIP(60, time.Minute), h.userHandler.ChangePassword)

	categories := api.Group("/categories", signedIn...)
	categories.GET("", h.categoryHandler.List)
	categories.GET("/all", h.categoryHandler.ListAll)
	categories.GET("/:id", h.categoryHandler.GetDetail)
	categories.POST("", h.categoryHandler.Create)
	categories.PUT("/:id", h.categoryHandler.Update)
	categories.DELETE("/:id", h.categoryHandler.Delete)

	expenses := api.Group("/expenses", signedIn...)
	expenses.GET("", h.expenseHandler.List)
	expenses.GET("/summary", h.expenseHandler.Summary)
	expenses.GET("/:id", h.expenseHandler.GetDetail)
	expenses.POST("", h.expenseHandler.Create)
	expenses.PUT("/:id", h.expenseHandler.Update)
	expenses.DELETE("/:id", h.expenseHandler.Delete)

	budgets := api.Group("/budgets", signedIn...)
	budgets.GET("", h.budgetHandler.List)
	budgets.POST("", h.budgetHandler.Create)
	budgets.PUT("/:id", h.budgetHandler.Update)
	budgets.DELETE("/:id", h.budgetHandler.Delete)

	currencySettings := api.Group("/currency-settings", signedIn...)
	currencySettings.GET("", h.currencyHandler.Get)
	currencySettings.PUT("", h.currencyHandler.Update)

	planAreas := api.Group("/plan-areas", authRequired)
	planAreas.GET("", ipLimit, userLimit, h.planAreaHandler.List)
	planAreas.DELETE("/cache", rl.LimitIP(10, time.Minute), h.planAreaHandler.ResetCache)

	plans := api.Group("/plans", signedIn...)
	plans.GET("", h.planHandler.List)
	plans.GET("/stats", h.planHandler.Stats)
	plans.GET("/:id", h.planHandler.GetDetail)
	plans.POST("", h.planHandler.Create)
	plans.PUT("/:id", h.planHandler.Update)
	plans.PATCH("/:id/status", h.planHandler.UpdateStatus)
	plans.DELETE("/:id", h.planHandler.Delete)
	plans.POST("/:id/steps", h.planStepHandler.Create)

	steps := api.Group("/steps", signedIn...)
	steps.PUT("/:id", h.planStepHandler.Update)
	steps.PATCH("/:id/toggle", h.planStepHandler.Toggle)
	steps.PATCH("/:id/move", h.planStepHandler.Move)
	steps.DELETE("/:id", h.planStepHandler.Delete)

	schedule := api.Group("/schedule", signedIn...)
	schedule.GET("/occurrences", h.scheduleHandler.ListOccurrences)
	schedule.GET("/summary", h.scheduleHandler.GetWeekSummary)
	schedule.GET("/activities/:id", h.scheduleHandler.GetActivity)
	schedule.POST("/activities", h.scheduleHandler.CreateActivity)
	schedule.PUT("/activities/:id", h.scheduleHandler.UpdateActivity)
	schedule.DELETE("/activities/:id", h.scheduleHandler.DeleteActivity)

	api.Group("/holidays", signedIn...).GET("", h.holidayHandler.List)
	api.Group("/exchange-rates", signedIn...).GET("", h.exchangeHandler.List)

	boards := api.Group("/boards", signedIn...)
	boards.GET("", h.boardHandler.List)
	boards.GET("/:id", h.boardHandler.Get)
	boards.POST("", h.boardHandler.Create)
	boards.PUT("/:id", h.boardHandler.Update)
	boards.DELETE("/:id", h.boardHandler.Delete)
	boards.POST("/:id/columns", h.boardHandler.CreateColumn)

	columns := api.Group("/columns", signedIn...)
	columns.PUT("/:id", h.boardHandler.UpdateColumn)
	columns.DELETE("/:id", h.boardHandler.DeleteColumn)
	columns.POST("/:id/cards", h.boardHandler.CreateCard)

	cards := api.Group("/cards", signedIn...)
	cards.GET("/due", h.boardHandler.ListDueCards)
	cards.PUT("/:id", h.boardHandler.UpdateCard)
	cards.PATCH("/:id/move", h.boardHandler.MoveCard)
	cards.DELETE("/:id", h.boardHandler.DeleteCard)
}
