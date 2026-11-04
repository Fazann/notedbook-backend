package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Fazann/notedbook-backend/config"
	"github.com/Fazann/notedbook-backend/pkg/db"
	"github.com/Fazann/notedbook-backend/pkg/logger"
	"github.com/Fazann/notedbook-backend/pkg/redis"
	"github.com/Fazann/notedbook-backend/pkg/server"
	"github.com/Fazann/notedbook-backend/pkg/validation"
	"github.com/Fazann/notedbook-backend/routes"
	"github.com/gin-gonic/gin"
)

//	@title			Notebook API
//	@version		1.0
//	@description	REST API for the notebook app (expenses, planning, boards).
//	@description	Error responses have a `code` (translation key) and a `message` translated by the
//	@description	`Accept-Language` header or `?lang=` query: en (default), km, ms.
//	@BasePath		/api

//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Access token as "Bearer <token>".

func main() {
	// Init db connection
	db.Connect()

	// Init redis connection
	redis.InitRedis()

	// Report JSON field names in validation errors
	validation.Register()

	r := gin.Default()

	// Init api router
	routes.InitRouter(r)

	// Stop on Ctrl+C locally and on SIGTERM from Cloud Run / Docker.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info("application starting", "port", config.Port)
	if err := server.Run(ctx, server.New(":"+config.Port, r)); err != nil {
		logger.Fatal("server stopped", "err", err)
	}

	// Requests are done; release the pools.
	if err := db.Close(); err != nil {
		logger.Error("close database", "err", err)
	}
	if err := redis.RedisClient.Close(); err != nil {
		logger.Error("close redis", "err", err)
	}
	logger.Info("application stopped")
}
