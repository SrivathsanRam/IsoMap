package addresses

import (
	"github.com/SrivathsanRam/IsoMap/internal/database"
	"github.com/SrivathsanRam/IsoMap/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func FindOrCreate(db *database.Database, address models.Address) (*models.Address, error) {
	var existing models.Address

	query := db.DB.Where(
		"formatted_address = ? AND latitude = ? AND longitude = ?",
		address.FormattedAddress,
		address.Latitude,
		address.Longitude,
	)
	if address.PlaceID != "" {
		query = db.DB.Where("place_id = ?", address.PlaceID)
	}

	err := query.First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if err := db.DB.Create(&address).Error; err != nil {
		return nil, err
	}

	return &address, nil
}

func AddSearch(db *database.Database, userID uuid.UUID, addressID uuid.UUID, queryText string) error {
	search := models.AddressSearch{
		UserID:    userID,
		AddressID: addressID,
		QueryText: queryText,
	}

	return db.DB.Create(&search).Error
}

func ListRecent(db *database.Database, userID uuid.UUID, limit int) ([]models.AddressSearch, error) {
	var searches []models.AddressSearch

	err := db.DB.
		Preload("Address").
		Where("user_id = ?", userID).
		Order("searched_at DESC").
		Limit(limit).
		Find(&searches).Error

	return searches, err
}

func SaveAddress(db *database.Database, userID uuid.UUID, addressID uuid.UUID, nickname string) error {
	saved := models.SavedAddress{
		UserID:    userID,
		AddressID: addressID,
		Nickname:  nickname,
	}

	return db.DB.
		Where("user_id = ? AND address_id = ?", userID, addressID).
		FirstOrCreate(&saved).Error
}

func ListSaved(db *database.Database, userID uuid.UUID) ([]models.SavedAddress, error) {
	var saved []models.SavedAddress

	err := db.DB.
		Preload("Address").
		Where("user_id = ?", userID).
		Order("saved_at DESC").
		Find(&saved).Error

	return saved, err
}
