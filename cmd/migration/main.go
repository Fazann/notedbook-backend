package main

import (
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
	)

	if err != nil {
		logger.Fatal("migration failed", "err", err)
	}

	logger.Info("migration success")
}
