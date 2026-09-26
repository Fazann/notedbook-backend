package config

import (
	"log"
	"os"
	"strconv"

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
)

func init() {
	err := godotenv.Load()
	if err != nil {
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
	DbSSLMode = getEnvDefault("DB_SSLMODE", "disable")
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

	S3Bucket = os.Getenv("S3_BUCKET")
	S3ID = os.Getenv("S3_ID")
	S3Secret = os.Getenv("S3_SECRET")
	S3Region = os.Getenv("S3_REGION")
	S3PrefixUrl = os.Getenv("S3_PREFIX_URL")
	S3ProjectName = os.Getenv("S3_PROJECT_NAME")
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
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
