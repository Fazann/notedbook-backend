package main

import (
	"github.com/Fazann/notedbook-backend/config"
	"github.com/Fazann/notedbook-backend/pkg/db"
	"github.com/Fazann/notedbook-backend/pkg/logger"
	"github.com/Fazann/notedbook-backend/pkg/redis"
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

	logger.Info("application starting", "port", config.Port, "swagger", "/docs/index.html")
	if err := r.Run(":" + config.Port); err != nil {
		logger.Fatal("application start failed", "err", err)
	}
}
