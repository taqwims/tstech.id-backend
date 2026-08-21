package database

import (
	"log"

	"github.com/kotban/backend/internal/config"
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
		log.Println("📦 Connecting to SQLite database (kotban.db)...")
		db, err = gorm.Open(sqlite.Open("kotban.db"), &gorm.Config{
			Logger: logger.Default.LogMode(logLevel),
		})
	} else {
		log.Printf("🐘 Connecting to PostgreSQL database (%s:%s)...", cfg.DBHost, cfg.DBPort)
		db, err = gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
			Logger: logger.Default.LogMode(logLevel),
		})

		// Automatic fallback to SQLite in development mode if PostgreSQL connection fails
		if err != nil && cfg.APIEnv == "development" {
			log.Printf("⚠️ Could not connect to PostgreSQL (%v). Falling back to local SQLite database (kotban.db)...", err)
			db, err = gorm.Open(sqlite.Open("kotban.db"), &gorm.Config{
				Logger: logger.Default.LogMode(logLevel),
			})
		}
	}

	if err != nil {
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
