package main

import (
	"context"

	expenseService "github.com/Fazann/notedbook-backend/internal/exspense/service"
	planService "github.com/Fazann/notedbook-backend/internal/plans/service"
	"github.com/Fazann/notedbook-backend/pkg/db"
	"github.com/Fazann/notedbook-backend/pkg/logger"
)

func main() {
	db.Connect()

	ctx := context.Background()

	categoriesCreated, err := expenseService.SeedDefaultCategories(ctx)
	if err != nil {
		logger.Fatal("seed default categories failed", "err", err)
	}

	areasCreated, err := planService.SeedDefaultPlanAreas(ctx)
	if err != nil {
		logger.Fatal("seed default plan areas failed", "err", err)
	}

	logger.Info("seed success", "categories_created", categoriesCreated, "plan_areas_created", areasCreated)
}
