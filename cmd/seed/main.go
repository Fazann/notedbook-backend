package main

import (
	"context"

	"github.com/Fazann/notedbook-backend/internal/exspense/service"
	"github.com/Fazann/notedbook-backend/pkg/db"
	"github.com/Fazann/notedbook-backend/pkg/logger"
)

func main() {
	db.Connect()

	created, err := service.SeedDefaultCategories(context.Background())
	if err != nil {
		logger.Fatal("seed default categories failed", "err", err)
	}

	logger.Info("seed success", "categories_created", created)
}
