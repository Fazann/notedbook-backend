package db

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Fazann/notedbook-backend/config"
	"github.com/Fazann/notedbook-backend/pkg/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var (
	DB   *gorm.DB
	once sync.Once
)

// Connect opens the PostgreSQL connection pool once and returns the shared *gorm.DB.
func Connect() *gorm.DB {
	once.Do(func() {
		db, err := open()
		if err != nil {
			logger.Fatal("failed to connect database", "err", err, "host", config.DbHost, "db", config.DbName)
		}
		DB = db
		logger.Info("database connected", "host", config.DbHost, "db", config.DbName, "max_open_conns", config.DbMaxOpenConns)
	})
	return DB
}

func open() (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		config.DbHost, config.DbPort, config.DbUsername, config.DbPassword,
		config.DbName, config.DbSSLMode, config.DbTimeZone,
	)

	logLevel := gormlogger.Warn
	if config.Environment == "development" {
		logLevel = gormlogger.Info
	}

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: dsn,
		// Cache prepared statements server-side via pgx (faster repeated queries).
		PreferSimpleProtocol: false,
	}), &gorm.Config{
		// Skip the implicit transaction GORM wraps around single writes; ~30% faster writes.
		SkipDefaultTransaction: true,
		// Reuse prepared statements across calls.
		PrepareStmt: true,
		Logger: gormlogger.NewSlogLogger(logger.Log.With("component", "gorm"), gormlogger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      config.Environment == "production",
		}),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(config.DbMaxOpenConns)
	sqlDB.SetMaxIdleConns(config.DbMaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(config.DbConnMaxLifetime) * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Duration(config.DbConnMaxIdleTime) * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}

// Close releases all pooled connections; call on graceful shutdown.
func Close() error {
	if DB == nil {
		return nil
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

/*
Begin begins a transaction with any transaction options opts
*/
func StartTrans() *gorm.DB {
	return DB.Begin()
}

/*
Commit commits the changes in a transaction
*/
func CommitTrans(tx *gorm.DB) error {
	return tx.Commit().Error
}

/*
Rollback rollbacks the changes in a transaction
*/
func RollbackTrans(tx *gorm.DB) {
	tx.Rollback()
}
