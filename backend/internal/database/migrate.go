package database

import (
	"github.com/SrivathsanRam/IsoMap/internal/models"
	"github.com/pkg/errors"
)

func (database *Database) Migrate() error {
	err := database.DB.Exec(`CREATE EXTENSION IF NOT EXISTS pgcrypto`).Error
	if err != nil {
		return errors.Wrap(err, "failed to create pgcrypto extension")
	}

	err = database.DB.AutoMigrate(
		&models.User{},
		&models.Address{},
		&models.AddressSearch{},
		&models.SavedAddress{},
	)
	
	if err != nil {
		return errors.Wrap(err, "failed to run database migrations")
	}

	return nil
}
