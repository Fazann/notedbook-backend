package config

import (
	"errors"
	"io/fs"
	"log"
	"os"
	"strconv"
	"strings"

	"fmt"
	"net"

	"github.com/joho/godotenv"
)

var (
	Port string

	DbUsername string
	DbPassword string
	DbName     string
	DbHost     string
	DbPort     string
	DbSSLMode  string
	DbTimeZone string

	DbMaxOpenConns    int = 100
	DbMaxIdleConns    int = 25
	DbConnMaxLifetime int = 30 // minutes
	DbConnMaxIdleTime int = 5  // minutes

	RedisAddr     string
	RedisPassword string
	RedisDB       int = 0
	RedisPort     string
	RedisTLS      bool
	Environment   string

	JwtSecret       string
	AccessTokenExp  int = 1
	RefreshTokenExp int = 1

	S3ID          string
	S3Secret      string
	S3Bucket      string
	S3Region      string
	S3PrefixUrl   string
	S3ProjectName string

	SmtpHost     string
	SmtpPort     string
	SmtpUsername string
	SmtpPassword string
	SmtpFrom     string
	SmtpFromName string

	// Secret for the AES-GCM password reset key.
	OtpSecret string

	// Browser origins allowed to call the API (CORS_ALLOWED_ORIGINS, comma-separated, * patterns allowed).
	// Empty allows every origin.
	CorsAllowedOrigins []string

	// Proxies in front of the API that append to X-Forwarded-For; the client IP is that many entries
	// from the right. 0 uses the TCP peer address. Defaults to 1 on Cloud Run (K_SERVICE is set).
	TrustedProxyHops int

	// Rate limits per route, in requests a minute; blocks double each time up to RateLimitMaxBlock minutes.
	RateLimitEnabled    = true
	RateLimitIP         = 100
	RateLimitUserReads  = 60
	RateLimitUserWrites = 20
	RateLimitMaxBlock   = 60   // minutes
	RateLimitStrikesTTL = 1440 // minutes without a block before the doubling resets

	// Free, keyless source of daily exchange rates (base USD); see https://www.exchangerate-api.com/docs/free.
	ExchangeRateAPIURL string
)

func init() {
	// A missing .env is fine (env vars set by the host, or tests); a broken one is not.
	err := godotenv.Load()
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Fatal("Error loading .env file")
	}

	Port = os.Getenv("PORT")
	Environment = os.Getenv("ENVIRONMENT")

	JwtSecret = os.Getenv("JWT_SECRET")
	//convert access token expiry string to int
	accessTokenExpStr := os.Getenv("ACC_TK_EXPIRED")
	if accessTokenExpStr != "" {
		AccessTokenExp, err = strconv.Atoi(accessTokenExpStr)
		if err != nil {
			log.Fatal("Error converting ACCESS_TOKEN_EXP to number")
		}
	}

	//convert refresh token expiry string to int
	refreshTokenExp := os.Getenv("REF_TK_EXPIRED")
	if refreshTokenExp != "" {
		RefreshTokenExp, err = strconv.Atoi(refreshTokenExp)
		if err != nil {
			log.Fatal("Error converting REFRESH_TOKEN_EXP to number")
		}
	}

	// database connection
	DbUsername = os.Getenv("DB_USERNAME")
	DbPassword = os.Getenv("DB_PASSWORD")
	DbName = os.Getenv("DB_NAME")
	DbHost = os.Getenv("DB_HOST")
	DbPort = os.Getenv("DB_PORT")
	DbSSLMode = getEnvDefault("DB_SSLMODE", "require")
	DbTimeZone = getEnvDefault("DB_TIMEZONE", "UTC")

	DbMaxOpenConns = getEnvInt("DB_MAX_OPEN_CONNS", DbMaxOpenConns)
	DbMaxIdleConns = getEnvInt("DB_MAX_IDLE_CONNS", DbMaxIdleConns)
	DbConnMaxLifetime = getEnvInt("DB_CONN_MAX_LIFETIME", DbConnMaxLifetime)
	DbConnMaxIdleTime = getEnvInt("DB_CONN_MAX_IDLE_TIME", DbConnMaxIdleTime)

	RedisAddr = os.Getenv("REDIS_ADDR")
	RedisPassword = os.Getenv("REDIS_PASSWORD")
	RedisPort = os.Getenv("REDIS_PORT")
	redisDbStr := os.Getenv("REDIS_DB")
	if redisDbStr != "" {
		RedisDB, err = strconv.Atoi(redisDbStr)
		if err != nil {
			log.Fatal("Error converting REDIS_DB to number")
		}
	}
	if v := os.Getenv("REDIS_TLS"); v != "" {
		RedisTLS, err = strconv.ParseBool(v)
		if err != nil {
			log.Fatal("Error converting REDIS_TLS to bool")
		}
	}

	S3Bucket = os.Getenv("S3_BUCKET")
	S3ID = os.Getenv("S3_ID")
	S3Secret = os.Getenv("S3_SECRET")
	S3Region = os.Getenv("S3_REGION")
	S3PrefixUrl = os.Getenv("S3_PREFIX_URL")
	S3ProjectName = os.Getenv("S3_PROJECT_NAME")

	SmtpHost = os.Getenv("SMTP_HOST")
	SmtpPort = getEnvDefault("SMTP_PORT", "587")
	SmtpUsername = os.Getenv("SMTP_USERNAME")
	SmtpPassword = os.Getenv("SMTP_PASSWORD")
	SmtpFrom = os.Getenv("SMTP_FROM")
	SmtpFromName = getEnvDefault("SMTP_FROM_NAME", "Notebook")

	OtpSecret = os.Getenv("OTP_SECRET")

	for _, o := range strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",") {
		if o = strings.TrimSuffix(strings.TrimSpace(o), "/"); o != "" {
			CorsAllowedOrigins = append(CorsAllowedOrigins, o)
		}
	}
	if len(CorsAllowedOrigins) == 0 && Environment == "production" {
		log.Println("warning: CORS_ALLOWED_ORIGINS is empty, so every origin may call the API")
	}

	defaultHops := 0
	if os.Getenv("K_SERVICE") != "" {
		defaultHops = 1
	}
	TrustedProxyHops = getEnvInt("TRUSTED_PROXY_HOPS", defaultHops)

	RateLimitEnabled = getEnvBool("RATE_LIMIT_ENABLED", RateLimitEnabled)
	RateLimitIP = getEnvInt("RATE_LIMIT_IP", RateLimitIP)
	RateLimitUserReads = getEnvInt("RATE_LIMIT_USER_READS", RateLimitUserReads)
	RateLimitUserWrites = getEnvInt("RATE_LIMIT_USER_WRITES", RateLimitUserWrites)
	RateLimitMaxBlock = getEnvInt("RATE_LIMIT_MAX_BLOCK", RateLimitMaxBlock)
	RateLimitStrikesTTL = getEnvInt("RATE_LIMIT_STRIKES_TTL", RateLimitStrikesTTL)

	ExchangeRateAPIURL = getEnvDefault("EXCHANGE_RATE_API_URL", "https://open.er-api.com/v6/latest/USD")

	if TrustedProxyHops < 0 || RateLimitIP <= 0 || RateLimitUserReads <= 0 || RateLimitUserWrites <= 0 ||
		RateLimitMaxBlock <= 0 || RateLimitStrikesTTL <= 0 {
		log.Fatal("TRUSTED_PROXY_HOPS must be 0 or more and RATE_LIMIT_* values must be positive")
	}
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		log.Fatalf("Error converting %s to bool", key)
	}
	return b
}

func getEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Fatalf("Error converting %s to number", key)
	}
	return n
}

func GetOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}
