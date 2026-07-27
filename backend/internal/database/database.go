package database

import (
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Database struct {
	DB *gorm.DB
}

var (
	cachedDB  *Database
	cachedURL string
	dbMu      sync.Mutex
)

func GetDB() (*Database, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, errors.New("DATABASE_URL is not set")
	}

	dbMu.Lock()
	defer dbMu.Unlock()

	if cachedDB != nil && cachedURL == databaseURL {
		return cachedDB, nil
	}

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to database")
	}

	// Set connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get SQL db instance")
	}
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		return nil, errors.Wrap(err, "failed to ping database")
	}

	cachedDB = &Database{DB: db}
	cachedURL = databaseURL

	return cachedDB, nil
}
