package redis

import (
	"strings"

	"github.com/Fazann/notedbook-backend/config"
	"github.com/Fazann/notedbook-backend/pkg/logger"
	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitRedis() {
	connectToRedis()
}

func connectToRedis() {
	addr := config.RedisAddr
	// Only append port if RedisAddr doesn't already contain one
	if !strings.Contains(addr, ":") && config.RedisPort != "" {
		addr = addr + ":" + config.RedisPort
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
		// Connection pooling optimization
		PoolSize:     100, // Max number of connections in pool
		MaxRetries:   3,   // Max retries for network errors
		PoolTimeout:  5,   // Connection pool timeout in seconds
		MinIdleConns: 20,  // Keep min idle connections
	})

	RedisClient = redisClient

	logger.Info("Redis connected")
}
