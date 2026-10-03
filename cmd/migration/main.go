package main

import (
	"context"

	planService "github.com/Fazann/notedbook-backend/internal/plans/service"
	"github.com/Fazann/notedbook-backend/pkg/db"
	"github.com/Fazann/notedbook-backend/pkg/logger"
	model "github.com/Fazann/notedbook-backend/shared/models"
)

func main() {
	db.Connect()

	db1 := db.DB

	err := db1.AutoMigrate(
		model.Attachment{},
		model.User{},
		model.RefreshToken{},
		model.Device{},
		model.ExpenseCategory{},
		model.ExpenseCategoryTr{},
		model.Expense{},
		model.PlanArea{},
		model.PlanAreaTr{},
		model.Plan{},
		model.PlanStep{},
		model.ScheduleActivity{},
		model.ScheduleActivityDay{},
		model.ScheduleActivityException{},
		model.Board{},
		model.BoardColumn{},
		model.BoardCard{},
	)

	if err != nil {
		logger.Fatal("migration failed", "err", err)
	}

	// Steps created before plan_steps.position existed get 1000, 2000… in creation order.
	backfilled, err := planService.BackfillStepPositions(context.Background())
	if err != nil {
		logger.Fatal("backfill plan step positions failed", "err", err)
	}

	logger.Info("migration success", "plan_steps_backfilled", backfilled)
}
