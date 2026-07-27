package database

import (
	"github.com/SrivathsanRam/IsoMap/internal/models"
	"github.com/pkg/errors"
)

func (database *Database) Migrate() error {
	err := database.DB.AutoMigrate(
		&models.User{},
		&models.Address{},
		&models.AddressSearch{},
		&models.SavedAddress{},
		&models.Outing{},
		&models.OutingMember{},
		&models.CommunityPreset{},
		&models.CommunityPresetLocation{},
	)

	if err != nil {
		return errors.Wrap(err, "failed to run database migrations")
	}

	return nil
}
