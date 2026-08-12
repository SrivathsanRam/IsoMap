package database

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const databaseConnectionTimeout = 10 * time.Second

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

	// GORM otherwise performs an automatic ping without a deadline. A blocked
	// PostgreSQL/TLS handshake would then make server startup appear frozen.
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{DisableAutomaticPing: true})
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

	ctx, cancel := context.WithTimeout(context.Background(), databaseConnectionTimeout)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, errors.Wrap(err, "failed to ping database")
	}

	cachedDB = &Database{DB: db}
	cachedURL = databaseURL

	return cachedDB, nil
}
