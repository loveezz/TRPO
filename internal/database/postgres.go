package database

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(dsn string) (*gorm.DB, error) {
	var lastErr error

	for attempt := 1; attempt <= 20; attempt++ {
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		if err != nil {
			lastErr = err
			time.Sleep(2 * time.Second)
			continue
		}

		sqlDB, err := db.DB()
		if err != nil {
			lastErr = err
			time.Sleep(2 * time.Second)
			continue
		}
		if err := sqlDB.Ping(); err != nil {
			lastErr = err
			time.Sleep(2 * time.Second)
			continue
		}

		sqlDB.SetMaxOpenConns(10)
		sqlDB.SetMaxIdleConns(5)
		sqlDB.SetConnMaxLifetime(30 * time.Minute)
		return db, nil
	}

	return nil, fmt.Errorf("postgres connection failed after retries: %w", lastErr)
}
