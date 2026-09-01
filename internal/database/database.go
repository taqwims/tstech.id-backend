package database

import (
	"log"

	"github.com/tstech/backend/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.Config) *gorm.DB {
	logLevel := logger.Warn
	if cfg.APIEnv == "development" {
		logLevel = logger.Info
	}

	var db *gorm.DB
	var err error

	if cfg.DBDriver == "sqlite" {
		log.Println("📦 Connecting to SQLite database (tstech.db)...")
		db, err = gorm.Open(sqlite.Open("tstech.db"), &gorm.Config{
			Logger: logger.Default.LogMode(logLevel),
		})
	} else {
		log.Printf("🐘 Connecting to PostgreSQL database (%s:%s)...", cfg.DBHost, cfg.DBPort)
		db, err = gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
			Logger: logger.Default.LogMode(logLevel),
		})

		var pingErr error
		if err == nil {
			if sdb, sErr := db.DB(); sErr == nil {
				pingErr = sdb.Ping()
			}
		}

		// Fallback to SQLite if PostgreSQL fails
		if err != nil || pingErr != nil {
			failReason := err
			if pingErr != nil {
				failReason = pingErr
			}
			log.Printf("⚠️ Could not connect to PostgreSQL (%v). Falling back to local SQLite database (tstech.db)...", failReason)
			db, err = gorm.Open(sqlite.Open("tstech.db"), &gorm.Config{
				Logger: logger.Default.LogMode(logLevel),
			})
		}
	}

	if err != nil || db == nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get underlying sql.DB: %v", err)
	}

	// Connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	log.Println("✅ Database connected successfully")
	return db
}
