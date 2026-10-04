package redis

import (
	"context"
	"crypto/tls"
	"net"
	"strings"
	"time"

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

	opts := &redis.Options{
		Addr:     addr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
		// Connection pooling optimization
		PoolSize:     100,             // Max number of connections in pool
		MaxRetries:   3,               // Max retries for network errors
		PoolTimeout:  5 * time.Second, // Wait for a free connection from the pool
		MinIdleConns: 20,              // Keep min idle connections
	}
	// Managed Redis (Upstash, Redis Cloud, ...) closes plain connections, which shows up as EOF.
	if config.RedisTLS {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			host = addr
		}
		opts.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12, ServerName: host}
	}

	redisClient := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Fatal("failed to connect redis", "err", err, "addr", addr, "tls", config.RedisTLS)
	}

	RedisClient = redisClient

	logger.Info("redis connected", "addr", addr, "tls", config.RedisTLS)
}
